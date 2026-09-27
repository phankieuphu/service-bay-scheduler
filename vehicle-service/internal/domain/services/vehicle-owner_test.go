package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
	"vehicle-service/config"
	"vehicle-service/internal/constants"
	"vehicle-service/internal/domain/entity"
	"vehicle-service/internal/domain/events"
	"vehicle-service/internal/domain/ports"
)

func TestVehicleService_AssignInitialOwner(t *testing.T) {
	active := entity.Vehicle{ID: 3, Status: constants.StatusActive}
	type call struct {
		vehicleID, customerID int64
		date                  time.Time
	}

	setup := func(stored entity.Vehicle, assignErr error) (ports.VehicleService, *[]call, *[]entity.OutboxMessage) {
		var calls []call
		var outboxRows []entity.OutboxMessage
		repo := &MockVehicleRepository{GetByIDForUpdateFunc: func(_ context.Context, id int64) (entity.Vehicle, error) {
			if id != stored.ID {
				return entity.Vehicle{}, ports.ErrNotFound
			}
			return stored, nil
		}}
		vcr := &MockVehicleCustomerRepository{AssignVehicleToCustomerFunc: func(_ context.Context, vehicleID, customerID int64, date time.Time) error {
			calls = append(calls, call{vehicleID, customerID, date})
			return assignErr
		}}
		outbox := &MockOutboxRepository{CreateFunc: func(_ context.Context, m entity.OutboxMessage) error {
			outboxRows = append(outboxRows, m)
			return nil
		}}
		return NewVehicleService(config.Config{}, repo, outbox, vcr, &MockTxManager{}, &MockCache{}, &MockVehicleMaterialRepository{}, &MockServiceHistoryRepository{}), &calls, &outboxRows
	}

	t.Run("assigns and emits OwnerAssigned", func(t *testing.T) {
		svc, calls, rows := setup(active, nil)
		date := time.Date(2026, 3, 4, 15, 30, 0, 0, time.UTC)
		if err := svc.AssignInitialOwner(context.Background(), 3, 77, date); err != nil {
			t.Fatal(err)
		}
		if len(*calls) != 1 || (*calls)[0].customerID != 77 || !(*calls)[0].date.Equal(time.Date(2026, 3, 4, 0, 0, 0, 0, time.UTC)) {
			t.Fatalf("assign calls = %+v", *calls)
		}
		if len(*rows) != 1 || (*rows)[0].Topic != constants.OwnerAssigned || (*rows)[0].Key != "3" {
			t.Fatalf("outbox = %+v", *rows)
		}
		var event events.OwnerAssigned
		_ = json.Unmarshal((*rows)[0].Payload, &event)
		if event.VehicleID != 3 || event.CustomerID != 77 || event.OwnedFrom != "2026-03-04" || event.EventID == "" {
			t.Errorf("event = %+v", event)
		}
	})

	t.Run("already owned is a conflict pointing at /transfer", func(t *testing.T) {
		svc, _, rows := setup(active, fmt.Errorf("%w: the vehicle already has a current owner", ports.ErrConflict))
		err := svc.AssignInitialOwner(context.Background(), 3, 77, time.Now())
		if !errors.Is(err, ports.ErrConflict) || !strings.Contains(err.Error(), "/transfer") {
			t.Errorf("err = %v, want ErrConflict mentioning /transfer", err)
		}
		if len(*rows) != 0 {
			t.Error("event written for a failed assignment")
		}
	})

	scrapped := active
	scrapped.Status = constants.StatusScrapped
	for _, tt := range []struct {
		name       string
		stored     entity.Vehicle
		vehicleID  int64
		customerID int64
		date       time.Time
		want       error
	}{
		{"scrapped vehicle", scrapped, 3, 77, time.Now(), ports.ErrInvalidState},
		{"unknown vehicle", active, 9, 77, time.Now(), ports.ErrNotFound},
		{"future date", active, 3, 77, time.Now().AddDate(0, 0, 2), ports.ErrInvalidInput},
		{"missing customer", active, 3, 0, time.Now(), ports.ErrInvalidInput},
	} {
		t.Run("rejects "+tt.name, func(t *testing.T) {
			svc, calls, rows := setup(tt.stored, nil)
			if err := svc.AssignInitialOwner(context.Background(), tt.vehicleID, tt.customerID, tt.date); !errors.Is(err, tt.want) {
				t.Errorf("err = %v, want %v", err, tt.want)
			}
			if len(*calls) != 0 || len(*rows) != 0 {
				t.Error("nothing should be written")
			}
		})
	}
}
