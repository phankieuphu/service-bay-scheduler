package services

import (
	"context"
	"customer-service/config"
	"customer-service/internal/constants"
	"customer-service/internal/domain/entity"
	"customer-service/internal/domain/ports"
	"customer-service/pkg/times"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

type MockCustomerRepository struct {
	GetByIDFunc    func(ctx context.Context, id int64) (entity.Customer, error)
	CreateFunc     func(ctx context.Context, customer entity.Customer) (entity.Customer, error)
	ListFunc       func(ctx context.Context, params ports.ListCustomersParams) (ports.CustomerPage, error)
	UpdateFunc     func(ctx context.Context, customer entity.Customer) error
	DeleteFunc     func(ctx context.Context, id int64) error
	SoftDeleteFunc func(ctx context.Context, id int64) error
}

func (m *MockCustomerRepository) GetByID(ctx context.Context, id int64) (entity.Customer, error) {
	return m.GetByIDFunc(ctx, id)
}
func (m *MockCustomerRepository) Create(ctx context.Context, customer entity.Customer) (entity.Customer, error) {
	return m.CreateFunc(ctx, customer)
}
func (m *MockCustomerRepository) List(ctx context.Context, params ports.ListCustomersParams) (ports.CustomerPage, error) {
	return m.ListFunc(ctx, params)
}
func (m *MockCustomerRepository) Update(ctx context.Context, customer entity.Customer) error {
	return m.UpdateFunc(ctx, customer)
}
func (m *MockCustomerRepository) Delete(ctx context.Context, id int64) error {
	return m.DeleteFunc(ctx, id)
}

func (m *MockCustomerRepository) SoftDelete(ctx context.Context, id int64) error {
	return m.SoftDeleteFunc(ctx, id)
}

type MockTxMangerRepository struct {
	RunInTxFunc func(ctx context.Context, fn func(ctx context.Context) error) error
}

func (m *MockTxMangerRepository) RunInTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return m.RunInTxFunc(ctx, fn)
}

type MockOutBoxRepository struct {
	CreateFunc           func(ctx context.Context, message entity.OutboxMessage) error
	FetchUnpublishedFunc func(ctx context.Context, limit int) ([]entity.OutboxMessage, error)
	MarkPublishedFunc    func(ctx context.Context, ids []int64) error
}

func (m *MockOutBoxRepository) Create(ctx context.Context, message entity.OutboxMessage) error {
	return m.CreateFunc(ctx, message)
}
func (m *MockOutBoxRepository) FetchUnpublished(ctx context.Context, limit int) ([]entity.OutboxMessage, error) {
	return m.FetchUnpublishedFunc(ctx, limit)
}
func (m *MockOutBoxRepository) MarkPublished(ctx context.Context, ids []int64) error {
	return m.MarkPublishedFunc(ctx, ids)
}

type MockCacheRepository struct {
	SetFunc    func(ctx context.Context, key string, value any, ttl time.Duration) error
	GetFunc    func(ctx context.Context, key string) (string, error)
	DeleteFunc func(ctx context.Context, key string) error
}

func (m *MockCacheRepository) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	return m.SetFunc(ctx, key, value, ttl)
}
func (m *MockCacheRepository) Get(ctx context.Context, key string) (string, error) {
	return m.GetFunc(ctx, key)
}
func (m *MockCacheRepository) Delete(ctx context.Context, key string) error {
	return m.DeleteFunc(ctx, key)
}

func TestCustomer_GetCustomer(t *testing.T) {
	tests := []struct {
		name        string
		customer    entity.Customer
		getByIDFunc func(ctx context.Context, id int64) (entity.Customer, error)
		wantID      int64
		wantErr     bool
	}{
		{
			name: "success",
			customer: entity.Customer{
				ID: 1,
			},
			getByIDFunc: func(ctx context.Context, id int64) (entity.Customer, error) {
				return entity.Customer{ID: id}, nil
			},
			wantID: 1,
		},
		{
			name: "customer not found",
			customer: entity.Customer{
				ID: 99,
			},
			getByIDFunc: func(ctx context.Context, id int64) (entity.Customer, error) {
				return entity.Customer{}, errors.New("customer not found")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &MockCustomerRepository{
				GetByIDFunc: tt.getByIDFunc,
			}

			customerService := NewCustomerService(config.Config{}, repo, &MockOutBoxRepository{}, &MockTxMangerRepository{}, &MockCacheRepository{})
			result, err := customerService.GetCustomer(context.Background(), tt.customer.ID)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.ID != tt.wantID {
				t.Fatalf("expected ID %d, got %d", tt.wantID, result.ID)
			}
		})
	}
}

func TestCustomer_DeleteCustomer(t *testing.T) {
	tests := []struct {
		name       string
		id         int64
		deleteFunc func(ctx context.Context, id int64) error
		wantErr    bool
	}{
		{
			name: "customer not found",
			id:   0,
			deleteFunc: func(ctx context.Context, id int64) error {
				return errors.New("customer not found")
			},
			wantErr: true,
		},
		{
			name: "deleted and outbox event written",
			id:   42,
			deleteFunc: func(ctx context.Context, id int64) error {
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &MockCustomerRepository{
				SoftDeleteFunc: tt.deleteFunc,
			}
			var written []entity.OutboxMessage
			outbox := &MockOutBoxRepository{
				CreateFunc: func(ctx context.Context, message entity.OutboxMessage) error {
					written = append(written, message)
					return nil
				},
			}
			txManager := &MockTxMangerRepository{
				RunInTxFunc: func(ctx context.Context, fn func(ctx context.Context) error) error {
					return fn(ctx)
				},
			}

			service := NewCustomerService(config.Config{}, repo, outbox, txManager, &MockCacheRepository{})
			err := service.DeleteProfile(t.Context(), tt.id)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error but got nil")
				}
				if len(written) != 0 {
					t.Fatalf("expected no outbox message on failure, got %d", len(written))
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(written) != 1 {
				t.Fatalf("expected 1 outbox message, got %d", len(written))
			}
			msg := written[0]
			if msg.Topic != constants.CustomerDelete || msg.Key != "42" {
				t.Fatalf("unexpected topic/key: %q/%q", msg.Topic, msg.Key)
			}

			var event struct {
				ID        int64  `json:"id"`
				DeletedAt string `json:"deleted_at"`
			}
			if err := json.Unmarshal(msg.Payload, &event); err != nil {
				t.Fatalf("payload is not valid JSON: %v", err)
			}
			if event.ID != 42 {
				t.Fatalf("expected id 42, got %d", event.ID)
			}
			if _, err := time.Parse(times.WireLayout, event.DeletedAt); err != nil || !strings.HasSuffix(event.DeletedAt, "Z") || len(event.DeletedAt) != len("2006-01-02T15:04:05.000Z") {
				t.Fatalf("deleted_at %q is not UTC millisecond format", event.DeletedAt)
			}
		})
	}
}
