package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"
	"vehicle-service/config"
	"vehicle-service/internal/constants"
	"vehicle-service/internal/domain/entity"
	"vehicle-service/internal/domain/events"
	"vehicle-service/internal/domain/ports"
)

const validVin = "1HGCM82633A004352"

func TestVehicleService_RegisterVehicle(t *testing.T) {
	t.Run("normalizes, creates and writes VehicleCreated in the same transaction", func(t *testing.T) {
		var inTx bool
		var stored entity.Vehicle
		var outboxMsg entity.OutboxMessage
		repo := &MockVehicleRepository{CreateFunc: func(ctx context.Context, v entity.Vehicle) (entity.Vehicle, error) {
			if !inTx {
				t.Error("vehicle created outside the transaction")
			}
			stored = v
			v.ID, v.CreatedAt = 42, time.Now()
			return v, nil
		}}
		outbox := &MockOutboxRepository{CreateFunc: func(ctx context.Context, m entity.OutboxMessage) error {
			if !inTx {
				t.Error("outbox row written outside the transaction")
			}
			outboxMsg = m
			return nil
		}}
		tx := &MockTxManager{RunInTxFunc: func(ctx context.Context, fn func(ctx context.Context) error) error {
			inTx = true
			defer func() { inTx = false }()
			return fn(ctx)
		}}
		svc := NewVehicleService(config.Config{}, repo, outbox, &MockVehicleCustomerRepository{}, tx, &MockCache{}, &MockVehicleMaterialRepository{}, &MockServiceHistoryRepository{})

		// Local midnight in UTC+7 is still July 11 — not July 10 in UTC.
		warranty := time.Date(2030, 7, 11, 0, 0, 0, 0, time.FixedZone("UTC+7", 7*3600))
		got, err := svc.RegisterVehicle(context.Background(), entity.Vehicle{
			Vin:             " 1hgcm82633a004352 ",
			LicensePlate:    "51a-123.45",
			VehicleModelID:  7,
			WarrantyEndDate: &warranty,
		})
		if err != nil {
			t.Fatalf("RegisterVehicle: %v", err)
		}

		if stored.Vin != validVin || stored.LicensePlate != "51A12345" {
			t.Errorf("stored vin/plate = %q/%q, want normalized", stored.Vin, stored.LicensePlate)
		}
		if stored.Status != constants.StatusActive {
			t.Errorf("status = %q, want default ACTIVE", stored.Status)
		}
		if want := time.Date(2030, 7, 11, 0, 0, 0, 0, time.UTC); !stored.WarrantyEndDate.Equal(want) {
			t.Errorf("warranty = %v, want %v", stored.WarrantyEndDate, want)
		}
		if got.ID != 42 {
			t.Errorf("returned ID = %d", got.ID)
		}

		if outboxMsg.Topic != constants.VehicleCreated || outboxMsg.Key != "42" {
			t.Errorf("outbox topic/key = %q/%q", outboxMsg.Topic, outboxMsg.Key)
		}
		var event events.VehicleCreated
		if err := json.Unmarshal(outboxMsg.Payload, &event); err != nil {
			t.Fatalf("payload: %v", err)
		}
		if event.EventID == "" || event.VehicleID != 42 || event.Vin != validVin || event.WarrantyEndDate == nil || *event.WarrantyEndDate != "2030-07-11" {
			t.Errorf("unexpected event %+v", event)
		}
	})

	invalid := []struct {
		name    string
		vehicle entity.Vehicle
	}{
		{"short VIN", entity.Vehicle{Vin: "1HGCM8263", VehicleModelID: 1}},
		{"VIN with O", entity.Vehicle{Vin: "1HGCM82633AOO4352", VehicleModelID: 1}},
		{"no model", entity.Vehicle{Vin: validVin}},
		{"unknown status", entity.Vehicle{Vin: validVin, VehicleModelID: 1, Status: "STOLEN"}},
		{"plate too long", entity.Vehicle{Vin: validVin, VehicleModelID: 1, LicensePlate: "ABCDEFGHIJKLMNOPQRSTU"}},
		{"plate with no letters or digits", entity.Vehicle{Vin: validVin, VehicleModelID: 1, LicensePlate: "--"}},
	}
	for _, tt := range invalid {
		t.Run("rejects "+tt.name, func(t *testing.T) {
			repo := &MockVehicleRepository{CreateFunc: func(context.Context, entity.Vehicle) (entity.Vehicle, error) {
				t.Error("repository called for invalid input")
				return entity.Vehicle{}, nil
			}}
			svc := NewVehicleService(config.Config{}, repo, &MockOutboxRepository{}, &MockVehicleCustomerRepository{}, &MockTxManager{}, &MockCache{}, &MockVehicleMaterialRepository{}, &MockServiceHistoryRepository{})
			if _, err := svc.RegisterVehicle(context.Background(), tt.vehicle); !errors.Is(err, ports.ErrInvalidInput) {
				t.Errorf("err = %v, want ErrInvalidInput", err)
			}
		})
	}

	t.Run("no plate stays empty", func(t *testing.T) {
		var stored entity.Vehicle
		repo := &MockVehicleRepository{CreateFunc: func(_ context.Context, v entity.Vehicle) (entity.Vehicle, error) { stored = v; return v, nil }}
		outbox := &MockOutboxRepository{CreateFunc: func(context.Context, entity.OutboxMessage) error { return nil }}
		svc := NewVehicleService(config.Config{}, repo, outbox, &MockVehicleCustomerRepository{}, &MockTxManager{}, &MockCache{}, &MockVehicleMaterialRepository{}, &MockServiceHistoryRepository{})
		if _, err := svc.RegisterVehicle(context.Background(), entity.Vehicle{Vin: validVin, VehicleModelID: 1}); err != nil {
			t.Fatal(err)
		}
		if stored.LicensePlate != "" || stored.WarrantyEndDate != nil {
			t.Errorf("stored %+v, want no plate and no warranty", stored)
		}
	})

	t.Run("duplicate VIN propagates conflict and writes no event", func(t *testing.T) {
		repo := &MockVehicleRepository{CreateFunc: func(context.Context, entity.Vehicle) (entity.Vehicle, error) {
			return entity.Vehicle{}, fmt.Errorf("%w: a vehicle with this VIN is already registered", ports.ErrConflict)
		}}
		outbox := &MockOutboxRepository{CreateFunc: func(context.Context, entity.OutboxMessage) error {
			t.Error("event written for a failed insert")
			return nil
		}}
		svc := NewVehicleService(config.Config{}, repo, outbox, &MockVehicleCustomerRepository{}, &MockTxManager{}, &MockCache{}, &MockVehicleMaterialRepository{}, &MockServiceHistoryRepository{})
		if _, err := svc.RegisterVehicle(context.Background(), entity.Vehicle{Vin: validVin, VehicleModelID: 1}); !errors.Is(err, ports.ErrConflict) {
			t.Errorf("err = %v, want ErrConflict", err)
		}
	})
}
