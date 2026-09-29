package services

import (
	"context"
	"encoding/json"
	"testing"
	"time"
	"vehicle-service/config"
	"vehicle-service/internal/constants"
	"vehicle-service/internal/domain/entity"
	"vehicle-service/internal/domain/events"
)

func TestVehicleService_TransferVehicle_Event(t *testing.T) {
	var rows []entity.OutboxMessage
	vcr := &MockVehicleCustomerRepository{
		UnassignVehicleFromCustomerFunc: func(context.Context, int64, int64, time.Time) error { return nil },
		AssignVehicleToCustomerFunc:     func(context.Context, int64, int64, time.Time) error { return nil },
	}
	outbox := &MockOutboxRepository{CreateFunc: func(_ context.Context, m entity.OutboxMessage) error {
		rows = append(rows, m)
		return nil
	}}
	svc := NewVehicleService(config.Config{}, &MockVehicleRepository{}, outbox, vcr, &MockTxManager{}, &MockCache{}, &MockVehicleMaterialRepository{}, &MockServiceHistoryRepository{})

	err := svc.TransferVehicle(context.Background(), entity.TransferVehicle{
		VehicleID: 9,
		From:      10,
		To:        11,
		Date:      time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Topic != constants.TransferVehicle || rows[0].Key != "9" {
		t.Fatalf("outbox = %+v", rows)
	}

	var event events.VehicleEvent[events.Ownership]
	if err := json.Unmarshal(rows[0].Payload, &event); err != nil {
		t.Fatalf("payload: %v", err)
	}
	v := event.Vehicle
	if event.EventID == "" || event.EventType != events.VehicleTransferred || event.OccurredAt.IsZero() ||
		v.ID != 9 || v.CustomerID != 11 || v.PreviousCustomerID == nil || *v.PreviousCustomerID != 10 || v.OwnedFrom != "2026-09-01" {
		t.Errorf("unexpected event %s", rows[0].Payload)
	}
}
