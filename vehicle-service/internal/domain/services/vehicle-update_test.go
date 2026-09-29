package services

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
	"vehicle-service/config"
	"vehicle-service/internal/constants"
	"vehicle-service/internal/domain/entity"
	"vehicle-service/internal/domain/events"
	"vehicle-service/internal/domain/ports"
)

func TestVehicleService_UpdateVehicle(t *testing.T) {
	readAt := time.Date(2026, 9, 1, 10, 0, 0, 123456000, time.UTC)
	oldWarranty := time.Date(2027, 1, 31, 0, 0, 0, 0, time.UTC)
	current := entity.Vehicle{ID: 5, Vin: validVin, Status: constants.StatusActive, WarrantyEndDate: &oldWarranty, UpdatedAt: readAt}

	type harness struct {
		svc     ports.VehicleService
		updated *entity.Vehicle
		events  map[string][]byte
	}
	setup := func(t *testing.T, stored entity.Vehicle) *harness {
		t.Helper()
		h := &harness{events: map[string][]byte{}}
		repo := &MockVehicleRepository{
			GetByIDForUpdateFunc: func(_ context.Context, id int64) (entity.Vehicle, error) {
				if id != stored.ID {
					return entity.Vehicle{}, ports.ErrNotFound
				}
				return stored, nil
			},
			UpdateFunc: func(_ context.Context, v entity.Vehicle) error {
				if !v.UpdatedAt.Equal(stored.UpdatedAt) {
					t.Errorf("Update must lock on the updated_at that was read, got %v", v.UpdatedAt)
				}
				v.UpdatedAt = readAt.Add(time.Minute)
				h.updated = &v
				return nil
			},
			GetByIDFunc: func(context.Context, int64) (entity.Vehicle, error) { return *h.updated, nil },
		}
		outbox := &MockOutboxRepository{CreateFunc: func(_ context.Context, m entity.OutboxMessage) error {
			h.events[m.Topic] = m.Payload
			return nil
		}}
		h.svc = NewVehicleService(config.Config{}, repo, outbox, &MockVehicleCustomerRepository{}, &MockTxManager{}, &MockCache{}, &MockVehicleMaterialRepository{}, &MockServiceHistoryRepository{})
		return h
	}
	status := func(s constants.VehicleStatus) *constants.VehicleStatus { return &s }

	t.Run("warranty change emits VehicleUpdated and WarrantyChanged with the previous date", func(t *testing.T) {
		h := setup(t, current)
		newWarranty := time.Date(2029, 6, 30, 23, 0, 0, 0, time.FixedZone("UTC-5", -5*3600))
		got, err := h.svc.UpdateVehicle(context.Background(), 5, entity.VehicleUpdate{WarrantyEndDate: &newWarranty, UpdatedAt: readAt})
		if err != nil {
			t.Fatalf("UpdateVehicle: %v", err)
		}
		if !got.UpdatedAt.After(readAt) {
			t.Error("returned vehicle should carry the new updated_at")
		}
		if want := time.Date(2029, 6, 30, 0, 0, 0, 0, time.UTC); !h.updated.WarrantyEndDate.Equal(want) {
			t.Errorf("stored warranty %v, want %v (the date as the client wrote it)", h.updated.WarrantyEndDate, want)
		}
		if _, ok := h.events[constants.VehicleUpdate]; !ok {
			t.Error("no VehicleUpdated event")
		}
		var updatedEvent events.VehicleEvent[events.VehicleState]
		if err := json.Unmarshal(h.events[constants.VehicleUpdate], &updatedEvent); err != nil {
			t.Fatalf("VehicleUpdated: %v", err)
		}
		if updatedEvent.EventType != events.VehicleUpdated || updatedEvent.Vehicle.ID != 5 || *updatedEvent.Vehicle.WarrantyEndDate != "2029-06-30" {
			t.Errorf("unexpected VehicleUpdated %+v", updatedEvent)
		}
		var wc events.VehicleEvent[events.WarrantyChange]
		if err := json.Unmarshal(h.events[constants.WarrantyChanged], &wc); err != nil {
			t.Fatalf("WarrantyChanged: %v", err)
		}
		if wc.EventID == "" || wc.EventType != events.WarrantyChanged || wc.Vehicle.ID != 5 ||
			*wc.Vehicle.PreviousWarrantyEndDate != "2027-01-31" || *wc.Vehicle.WarrantyEndDate != "2029-06-30" {
			t.Errorf("unexpected WarrantyChanged %+v", wc)
		}
	})

	t.Run("status-only change emits VehicleUpdated but not WarrantyChanged", func(t *testing.T) {
		h := setup(t, current)
		if _, err := h.svc.UpdateVehicle(context.Background(), 5, entity.VehicleUpdate{Status: status(constants.StatusSold), UpdatedAt: readAt}); err != nil {
			t.Fatal(err)
		}
		if h.updated.Status != constants.StatusSold {
			t.Errorf("status = %q", h.updated.Status)
		}
		if _, ok := h.events[constants.WarrantyChanged]; ok {
			t.Error("WarrantyChanged emitted without a warranty change")
		}
		if _, ok := h.events[constants.VehicleUpdate]; !ok {
			t.Error("no VehicleUpdated event")
		}
	})

	t.Run("no-op update writes nothing", func(t *testing.T) {
		h := setup(t, current)
		same := oldWarranty
		got, err := h.svc.UpdateVehicle(context.Background(), 5, entity.VehicleUpdate{Status: status(constants.StatusActive), WarrantyEndDate: &same, UpdatedAt: readAt})
		if err != nil {
			t.Fatal(err)
		}
		if h.updated != nil || len(h.events) != 0 {
			t.Error("a no-op update should not write the row or emit events")
		}
		if !got.UpdatedAt.Equal(readAt) {
			t.Error("no-op should return the vehicle unchanged")
		}
	})

	errCases := []struct {
		name   string
		stored entity.Vehicle
		id     int64
		update entity.VehicleUpdate
		want   error
	}{
		{"stale updated_at", current, 5, entity.VehicleUpdate{Status: status(constants.StatusSold), UpdatedAt: readAt.Add(-time.Second)}, ports.ErrConflict},
		{"scrapped is final", func() entity.Vehicle { v := current; v.Status = constants.StatusScrapped; return v }(), 5,
			entity.VehicleUpdate{Status: status(constants.StatusActive), UpdatedAt: readAt}, ports.ErrInvalidState},
		{"unknown vehicle", current, 99, entity.VehicleUpdate{Status: status(constants.StatusSold), UpdatedAt: readAt}, ports.ErrNotFound},
		{"nothing to update", current, 5, entity.VehicleUpdate{UpdatedAt: readAt}, ports.ErrInvalidInput},
		{"unknown status", current, 5, entity.VehicleUpdate{Status: status("STOLEN"), UpdatedAt: readAt}, ports.ErrInvalidInput},
		{"missing updated_at", current, 5, entity.VehicleUpdate{Status: status(constants.StatusSold)}, ports.ErrInvalidInput},
	}
	for _, tt := range errCases {
		t.Run(tt.name, func(t *testing.T) {
			h := setup(t, tt.stored)
			_, err := h.svc.UpdateVehicle(context.Background(), tt.id, tt.update)
			if !errors.Is(err, tt.want) {
				t.Errorf("err = %v, want %v", err, tt.want)
			}
			if h.updated != nil || len(h.events) != 0 {
				t.Error("a rejected update should not write the row or emit events")
			}
		})
	}
}
