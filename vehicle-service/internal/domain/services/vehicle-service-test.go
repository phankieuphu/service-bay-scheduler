package services

import (
	"context"
	"testing"
	"time"
	"vehicle-service/config"
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
	expectedVehicle := entity.Vehicle{
		ID:           1,
		Vin:          "VIN123",
		LicensePlate: "ABC-123",
	}
	repo := &MockVehicleRepository{
		GetByIDFunc: func(ctx context.Context, id int64) (entity.Vehicle, error) {
			return expectedVehicle, nil
		},
	}
	vehicleService := NewVehicleService(config.Config{}, repo, &MockOutboxRepository{}, &MockVehicleCustomerRepository{}, &MockTxManager{}, &MockCache{})

	result, err := vehicleService.GetVehicle(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != expectedVehicle.ID {
		t.Errorf("expected ID %d, got %d", expectedVehicle.ID, result.ID)
	}
}
