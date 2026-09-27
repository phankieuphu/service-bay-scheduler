package services

import (
	"context"
	"errors"
	"testing"
	"time"
	"vehicle-service/config"
	"vehicle-service/internal/domain/entity"
	"vehicle-service/internal/domain/events"
	"vehicle-service/internal/domain/ports"
)

// vehicleRepoWith returns a repository that knows exactly one vehicle.
func vehicleRepoWith(v entity.Vehicle) *MockVehicleRepository {
	return &MockVehicleRepository{GetByIDFunc: func(_ context.Context, id int64) (entity.Vehicle, error) {
		if id != v.ID {
			return entity.Vehicle{}, ports.ErrNotFound
		}
		return v, nil
	}}
}

func TestVehicleService_GetWarranty(t *testing.T) {
	end := time.Date(2027, 1, 31, 0, 0, 0, 0, time.UTC)
	withWarranty := entity.Vehicle{ID: 1, WarrantyEndDate: &end}
	svc := NewVehicleService(config.Config{}, vehicleRepoWith(withWarranty), &MockOutboxRepository{}, &MockVehicleCustomerRepository{}, &MockTxManager{}, &MockCache{}, &MockVehicleMaterialRepository{}, &MockServiceHistoryRepository{})

	cases := []struct {
		name       string
		asOf       time.Time
		wantActive bool
		wantDays   int
	}{
		{"well before the end", time.Date(2027, 1, 1, 9, 0, 0, 0, time.UTC), true, 30},
		{"the last day is still covered", time.Date(2027, 1, 31, 23, 59, 0, 0, time.UTC), true, 0},
		{"the day after", time.Date(2027, 2, 1, 0, 0, 0, 0, time.UTC), false, 0},
		// 2027-01-31 evening in UTC-8 is still Jan 31 for that caller.
		{"caller's own calendar day", time.Date(2027, 1, 31, 20, 0, 0, 0, time.FixedZone("UTC-8", -8*3600)), true, 0},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			w, err := svc.GetWarranty(context.Background(), 1, tt.asOf)
			if err != nil {
				t.Fatal(err)
			}
			if w.Active != tt.wantActive || w.DaysRemaining != tt.wantDays {
				t.Errorf("active=%v days=%d, want active=%v days=%d", w.Active, w.DaysRemaining, tt.wantActive, tt.wantDays)
			}
		})
	}

	t.Run("no warranty on record", func(t *testing.T) {
		svc := NewVehicleService(config.Config{}, vehicleRepoWith(entity.Vehicle{ID: 2}), &MockOutboxRepository{}, &MockVehicleCustomerRepository{}, &MockTxManager{}, &MockCache{}, &MockVehicleMaterialRepository{}, &MockServiceHistoryRepository{})
		w, err := svc.GetWarranty(context.Background(), 2, time.Now())
		if err != nil || w.Active || w.EndDate != nil {
			t.Errorf("got %+v, %v; want inactive with no end date", w, err)
		}
	})

	t.Run("unknown vehicle", func(t *testing.T) {
		if _, err := svc.GetWarranty(context.Background(), 99, time.Now()); !errors.Is(err, ports.ErrNotFound) {
			t.Errorf("err = %v, want ErrNotFound", err)
		}
	})
}

func TestVehicleService_GetVehicleMaterials(t *testing.T) {
	materials := &MockVehicleMaterialRepository{ListByVehicleFunc: func(_ context.Context, id int64) ([]entity.VehicleMaterial, error) {
		return []entity.VehicleMaterial{{ID: 10, VehicleID: id, MaterialID: 3, Count: 2}}, nil
	}}
	svc := NewVehicleService(config.Config{}, vehicleRepoWith(entity.Vehicle{ID: 1}), &MockOutboxRepository{}, &MockVehicleCustomerRepository{}, &MockTxManager{}, &MockCache{}, materials, &MockServiceHistoryRepository{})

	got, err := svc.GetVehicleMaterials(context.Background(), 1)
	if err != nil || len(got) != 1 || got[0].MaterialID != 3 {
		t.Errorf("got %+v, %v", got, err)
	}
	if _, err := svc.GetVehicleMaterials(context.Background(), 99); !errors.Is(err, ports.ErrNotFound) {
		t.Errorf("unknown vehicle: err = %v, want ErrNotFound (not an empty list)", err)
	}
}

func TestVehicleService_ServiceHistory(t *testing.T) {
	t.Run("list applies default and max limit, 404 for unknown vehicle", func(t *testing.T) {
		var gotLimit int
		history := &MockServiceHistoryRepository{ListByVehicleFunc: func(_ context.Context, _ int64, limit int) ([]entity.ServiceHistoryEntry, error) {
			gotLimit = limit
			return nil, nil
		}}
		svc := NewVehicleService(config.Config{}, vehicleRepoWith(entity.Vehicle{ID: 1}), &MockOutboxRepository{}, &MockVehicleCustomerRepository{}, &MockTxManager{}, &MockCache{}, &MockVehicleMaterialRepository{}, history)

		for in, want := range map[int]int{0: defaultHistoryLimit, 10: 10, 5000: maxHistoryLimit} {
			if _, err := svc.GetServiceHistory(context.Background(), 1, in); err != nil {
				t.Fatal(err)
			}
			if gotLimit != want {
				t.Errorf("limit %d -> %d, want %d", in, gotLimit, want)
			}
		}
		if _, err := svc.GetServiceHistory(context.Background(), 99, 0); !errors.Is(err, ports.ErrNotFound) {
			t.Errorf("err = %v, want ErrNotFound", err)
		}
	})

	completed := events.ServiceCompleted{
		EventID: "e1", AppointmentID: 500, VehicleID: 1, DealershipID: 2,
		CompletedAt: time.Date(2026, 9, 1, 14, 0, 0, 0, time.UTC),
	}

	t.Run("record stores the entry; nil services become an empty list", func(t *testing.T) {
		var stored entity.ServiceHistoryEntry
		history := &MockServiceHistoryRepository{RecordFunc: func(_ context.Context, e entity.ServiceHistoryEntry) (bool, error) {
			stored = e
			return true, nil
		}}
		svc := NewVehicleService(config.Config{}, &MockVehicleRepository{}, &MockOutboxRepository{}, &MockVehicleCustomerRepository{}, &MockTxManager{}, &MockCache{}, &MockVehicleMaterialRepository{}, history)
		if err := svc.RecordServiceCompleted(context.Background(), completed); err != nil {
			t.Fatal(err)
		}
		if stored.AppointmentID != 500 || stored.VehicleID != 1 || stored.Services == nil {
			t.Errorf("stored %+v", stored)
		}
	})

	t.Run("a duplicate delivery is not an error", func(t *testing.T) {
		history := &MockServiceHistoryRepository{RecordFunc: func(context.Context, entity.ServiceHistoryEntry) (bool, error) { return false, nil }}
		svc := NewVehicleService(config.Config{}, &MockVehicleRepository{}, &MockOutboxRepository{}, &MockVehicleCustomerRepository{}, &MockTxManager{}, &MockCache{}, &MockVehicleMaterialRepository{}, history)
		if err := svc.RecordServiceCompleted(context.Background(), completed); err != nil {
			t.Errorf("err = %v", err)
		}
	})

	t.Run("rejects an event missing required fields", func(t *testing.T) {
		history := &MockServiceHistoryRepository{RecordFunc: func(context.Context, entity.ServiceHistoryEntry) (bool, error) {
			t.Error("should not be recorded")
			return true, nil
		}}
		svc := NewVehicleService(config.Config{}, &MockVehicleRepository{}, &MockOutboxRepository{}, &MockVehicleCustomerRepository{}, &MockTxManager{}, &MockCache{}, &MockVehicleMaterialRepository{}, history)
		bad := completed
		bad.CompletedAt = time.Time{}
		if err := svc.RecordServiceCompleted(context.Background(), bad); !errors.Is(err, ports.ErrInvalidInput) {
			t.Errorf("err = %v, want ErrInvalidInput", err)
		}
	})
}
