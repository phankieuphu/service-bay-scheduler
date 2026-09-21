package services

import (
	"context"
	"customer-service/internal/domain/entity"
	"testing"
	"time"
)

type MockCustomerRepository struct {
	GetByIDFunc func(ctx context.Context, id int64) (entity.Customer, error)
}

func (m *MockCustomerRepository) GetByID(ctx context.Context, id int64) (entity.Customer, error) {
	return m.GetByIDFunc(ctx, id)
}

func TestCustomer_GetCustomer(t *testing.T) {
	expectedUser := &entity.Customer{
		ID:        0,
		Name:      "",
		Email:     "",
		Phone:     "",
		BirthDay:  time.Time{},
		Status:    "",
		CreatedAt: time.Time{},
		UpdatedAt: time.Time{},
	}
	repo := &MockCustomerRepository{
		GetByIDFunc: func(ctx context.Context, id int64) (entity.Customer, error) {
			return *expectedUser, nil
		},
	}
	customerService := NewCustomerService()
	result, err := customerService.GetCustomer(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != expectedUser.ID {
		t.Errorf("expected ID 1, got %d", result.ID)
	}
}
