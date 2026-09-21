package services

import (
	"context"
	"errors"
	"testing"
	"time"
	"vehicle-service/config"
	"vehicle-service/internal/constants"
	"vehicle-service/internal/domain/entity"
	"vehicle-service/internal/domain/ports"
)

type MockVehicleRepository struct {
	CreateFunc  func(ctx context.Context, vehicle entity.Vehicle) (entity.Vehicle, error)
	GetByIDFunc func(ctx context.Context, id int64) (entity.Vehicle, error)
	ListFunc    func(ctx context.Context, params ports.ListVehiclesParams) (ports.VehiclePage, error)
	UpdateFunc  func(ctx context.Context, vehicle entity.Vehicle) error
	DeleteFunc  func(ctx context.Context, id int64) error
}

func (m *MockVehicleRepository) Create(ctx context.Context, vehicle entity.Vehicle) (entity.Vehicle, error) {
	return m.CreateFunc(ctx, vehicle)
}

func (m *MockVehicleRepository) GetByID(ctx context.Context, id int64) (entity.Vehicle, error) {
	return m.GetByIDFunc(ctx, id)
}

func (m *MockVehicleRepository) List(ctx context.Context, params ports.ListVehiclesParams) (ports.VehiclePage, error) {
	return m.ListFunc(ctx, params)
}

func (m *MockVehicleRepository) Update(ctx context.Context, vehicle entity.Vehicle) error {
	return m.UpdateFunc(ctx, vehicle)
}

func (m *MockVehicleRepository) Delete(ctx context.Context, id int64) error {
	return m.DeleteFunc(ctx, id)
}

type MockVehicleCustomerRepository struct {
	GetCustomerVehicleFunc          func(ctx context.Context, customerID int64) (entity.CustomerVehicle, error)
	AssignVehicleToCustomerFunc     func(ctx context.Context, vehicleID, customerID int64, date time.Time) error
	UnassignVehicleFromCustomerFunc func(ctx context.Context, vehicleID, customerID int64, date time.Time) error
}

func (m *MockVehicleCustomerRepository) GetCustomerVehicle(ctx context.Context, customerID int64) (entity.CustomerVehicle, error) {
	return m.GetCustomerVehicleFunc(ctx, customerID)
}

func (m *MockVehicleCustomerRepository) AssignVehicleToCustomer(ctx context.Context, vehicleID, customerID int64, date time.Time) error {
	return m.AssignVehicleToCustomerFunc(ctx, vehicleID, customerID, date)
}

func (m *MockVehicleCustomerRepository) UnassignVehicleFromCustomer(ctx context.Context, vehicleID, customerID int64, date time.Time) error {
	return m.UnassignVehicleFromCustomerFunc(ctx, vehicleID, customerID, date)
}

type MockOutboxRepository struct {
	CreateFunc           func(ctx context.Context, message entity.OutboxMessage) error
	FetchUnpublishedFunc func(ctx context.Context, limit int) ([]entity.OutboxMessage, error)
	MarkPublishedFunc    func(ctx context.Context, ids []int64) error
}

func (m *MockOutboxRepository) Create(ctx context.Context, message entity.OutboxMessage) error {
	return m.CreateFunc(ctx, message)
}

func (m *MockOutboxRepository) FetchUnpublished(ctx context.Context, limit int) ([]entity.OutboxMessage, error) {
	return m.FetchUnpublishedFunc(ctx, limit)
}

func (m *MockOutboxRepository) MarkPublished(ctx context.Context, ids []int64) error {
	return m.MarkPublishedFunc(ctx, ids)
}

type MockTxManager struct {
	RunInTxFunc func(ctx context.Context, fn func(ctx context.Context) error) error
}

func (m *MockTxManager) RunInTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if m.RunInTxFunc != nil {
		return m.RunInTxFunc(ctx, fn)
	}
	return fn(ctx)
}

type MockCache struct {
	SetFunc    func(ctx context.Context, key string, value any, ttl time.Duration) error
	GetFunc    func(ctx context.Context, key string) (string, error)
	DeleteFunc func(ctx context.Context, key string) error
}

func (m *MockCache) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	return m.SetFunc(ctx, key, value, ttl)
}

func (m *MockCache) Get(ctx context.Context, key string) (string, error) {
	return m.GetFunc(ctx, key)
}

func (m *MockCache) Delete(ctx context.Context, key string) error {
	return m.DeleteFunc(ctx, key)
}

func TestVehicleService_GetVehicle(t *testing.T) {
	tests := []struct {
		name        string
		vehicleID   int64
		getByIDFunc func(ctx context.Context, id int64) (entity.Vehicle, error)
		wantVehicle entity.Vehicle
		wantErr     error
	}{
		{
			name:      "returns vehicle on success",
			vehicleID: 1,
			getByIDFunc: func(ctx context.Context, id int64) (entity.Vehicle, error) {
				return entity.Vehicle{ID: 1, Vin: "VIN123", LicensePlate: "ABC-123"}, nil
			},
			wantVehicle: entity.Vehicle{ID: 1, Vin: "VIN123", LicensePlate: "ABC-123"},
			wantErr:     nil,
		},
		{
			name:      "propagates not found error",
			vehicleID: 2,
			getByIDFunc: func(ctx context.Context, id int64) (entity.Vehicle, error) {
				return entity.Vehicle{}, ports.ErrNotFound
			},
			wantVehicle: entity.Vehicle{},
			wantErr:     ports.ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &MockVehicleRepository{GetByIDFunc: tt.getByIDFunc}
			vehicleService := NewVehicleService(config.Config{}, repo, &MockOutboxRepository{}, &MockVehicleCustomerRepository{}, &MockTxManager{}, &MockCache{})

			result, err := vehicleService.GetVehicle(context.Background(), tt.vehicleID)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected error %v, got %v", tt.wantErr, err)
			}
			if result.ID != tt.wantVehicle.ID {
				t.Errorf("expected ID %d, got %d", tt.wantVehicle.ID, result.ID)
			}
		})
	}
}

func TestVehicleService_Transfer(t *testing.T) {
	errUnassign := errors.New("unassign boom")
	errOutbox := errors.New("outbox boom")
	errBeginTx := errors.New("begin tx boom")

	tests := []struct {
		name                            string
		transferVehicle                 entity.TransferVehicle
		unassignVehicleFromCustomerFunc func(ctx context.Context, vehicleID int64, from int64, date time.Time) error
		assignVehicleToCustomerFunc     func(ctx context.Context, vehicleID int64, to int64, date time.Time) error
		createFunc                      func(
			ctx context.Context,
			outBoxMessage entity.OutboxMessage,
		) error
		runInTxFunc func(ctx context.Context, fn func(ctx context.Context) error) error
		wantErr     error
		// assertCalls runs after TransferVehicle returns, with which mocks
		// were invoked, so each case can check the short-circuit behavior
		// of the RunInTx closure (e.g. assign/create must not run once
		// unassign fails).
		assertCalls func(t *testing.T, unassigned, assigned, created bool)
	}{
		{
			name: "success: unassign, assign, and outbox create all run in order",
			transferVehicle: entity.TransferVehicle{
				VehicleID: 1,
				From:      10,
				To:        20,
				Date:      time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			unassignVehicleFromCustomerFunc: func(ctx context.Context, vehicleID int64, from int64, date time.Time) error {
				return nil
			},
			assignVehicleToCustomerFunc: func(ctx context.Context, vehicleID int64, to int64, date time.Time) error {
				return nil
			},
			createFunc: func(ctx context.Context, outBoxMessage entity.OutboxMessage) error {
				if outBoxMessage.Topic != constants.TransferVehicle {
					t.Errorf("expected topic %q, got %q", constants.TransferVehicle, outBoxMessage.Topic)
				}
				if outBoxMessage.Key != "1" {
					t.Errorf("expected key %q, got %q", "1", outBoxMessage.Key)
				}
				return nil
			},
			wantErr: nil,
			assertCalls: func(t *testing.T, unassigned, assigned, created bool) {
				if !unassigned || !assigned || !created {
					t.Errorf("expected all three steps to run, got unassigned=%v assigned=%v created=%v", unassigned, assigned, created)
				}
			},
		},
		{
			name: "unassign failure short-circuits assign and outbox create",
			transferVehicle: entity.TransferVehicle{
				VehicleID: 1,
				From:      10,
				To:        20,
			},
			unassignVehicleFromCustomerFunc: func(ctx context.Context, vehicleID int64, from int64, date time.Time) error {
				return errUnassign
			},
			assignVehicleToCustomerFunc: func(ctx context.Context, vehicleID int64, to int64, date time.Time) error {
				t.Error("assign should not be called when unassign fails")
				return nil
			},
			createFunc: func(ctx context.Context, outBoxMessage entity.OutboxMessage) error {
				t.Error("outbox create should not be called when unassign fails")
				return nil
			},
			wantErr: errUnassign,
			assertCalls: func(t *testing.T, unassigned, assigned, created bool) {
				if !unassigned || assigned || created {
					t.Errorf("expected only unassign to run, got unassigned=%v assigned=%v created=%v", unassigned, assigned, created)
				}
			},
		},
		{
			name: "assign failure (e.g. new owner not found) short-circuits outbox create",
			transferVehicle: entity.TransferVehicle{
				VehicleID: 1,
				From:      10,
				To:        20,
			},
			unassignVehicleFromCustomerFunc: func(ctx context.Context, vehicleID int64, from int64, date time.Time) error {
				return nil
			},
			assignVehicleToCustomerFunc: func(ctx context.Context, vehicleID int64, to int64, date time.Time) error {
				return ports.ErrNotFound
			},
			createFunc: func(ctx context.Context, outBoxMessage entity.OutboxMessage) error {
				t.Error("outbox create should not be called when assign fails")
				return nil
			},
			wantErr: ports.ErrNotFound,
			assertCalls: func(t *testing.T, unassigned, assigned, created bool) {
				if !unassigned || !assigned || created {
					t.Errorf("expected unassign and assign to run but not create, got unassigned=%v assigned=%v created=%v", unassigned, assigned, created)
				}
			},
		},
		{
			name: "outbox create failure propagates after unassign and assign succeed",
			transferVehicle: entity.TransferVehicle{
				VehicleID: 1,
				From:      10,
				To:        20,
			},
			unassignVehicleFromCustomerFunc: func(ctx context.Context, vehicleID int64, from int64, date time.Time) error {
				return nil
			},
			assignVehicleToCustomerFunc: func(ctx context.Context, vehicleID int64, to int64, date time.Time) error {
				return nil
			},
			createFunc: func(ctx context.Context, outBoxMessage entity.OutboxMessage) error {
				return errOutbox
			},
			wantErr: errOutbox,
			assertCalls: func(t *testing.T, unassigned, assigned, created bool) {
				if !unassigned || !assigned || !created {
					t.Errorf("expected all three steps to be attempted, got unassigned=%v assigned=%v created=%v", unassigned, assigned, created)
				}
			},
		},
		{
			name: "RunInTx failure (e.g. cannot begin transaction) never invokes the closure",
			transferVehicle: entity.TransferVehicle{
				VehicleID: 1,
				From:      10,
				To:        20,
			},
			unassignVehicleFromCustomerFunc: func(ctx context.Context, vehicleID int64, from int64, date time.Time) error {
				t.Error("unassign should not be called when RunInTx itself fails")
				return nil
			},
			assignVehicleToCustomerFunc: func(ctx context.Context, vehicleID int64, to int64, date time.Time) error {
				t.Error("assign should not be called when RunInTx itself fails")
				return nil
			},
			createFunc: func(ctx context.Context, outBoxMessage entity.OutboxMessage) error {
				t.Error("outbox create should not be called when RunInTx itself fails")
				return nil
			},
			runInTxFunc: func(ctx context.Context, fn func(ctx context.Context) error) error {
				return errBeginTx
			},
			wantErr: errBeginTx,
			assertCalls: func(t *testing.T, unassigned, assigned, created bool) {
				if unassigned || assigned || created {
					t.Errorf("expected no steps to run, got unassigned=%v assigned=%v created=%v", unassigned, assigned, created)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var unassigned, assigned, created bool

			vehicleCustomerRepo := &MockVehicleCustomerRepository{
				UnassignVehicleFromCustomerFunc: func(ctx context.Context, vehicleID, customerID int64, date time.Time) error {
					unassigned = true
					return tt.unassignVehicleFromCustomerFunc(ctx, vehicleID, customerID, date)
				},
				AssignVehicleToCustomerFunc: func(ctx context.Context, vehicleID, customerID int64, date time.Time) error {
					assigned = true
					return tt.assignVehicleToCustomerFunc(ctx, vehicleID, customerID, date)
				},
			}
			outboxRepo := &MockOutboxRepository{
				CreateFunc: func(ctx context.Context, message entity.OutboxMessage) error {
					created = true
					return tt.createFunc(ctx, message)
				},
			}
			txManager := &MockTxManager{
				RunInTxFunc: tt.runInTxFunc,
			}

			vehicleService := NewVehicleService(config.Config{}, &MockVehicleRepository{}, outboxRepo, vehicleCustomerRepo, txManager, &MockCache{})

			err := vehicleService.TransferVehicle(context.Background(), tt.transferVehicle)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected error %v, got %v", tt.wantErr, err)
			}
			if tt.assertCalls != nil {
				tt.assertCalls(t, unassigned, assigned, created)
			}
		})
	}
}
