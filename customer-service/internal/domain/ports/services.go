package ports

import (
	"context"
	"customer-service/internal/domain/entity"
)

type CustomerService interface {
	CreateCustomer(ctx context.Context, customer entity.Customer) (entity.Customer, error)
	GetCustomer(ctx context.Context, customerID int64) (entity.Customer, error)
	GetCustomers(ctx context.Context, params ListCustomersParams) (CustomerPage, error)
	UpdateProfile(ctx context.Context, customerID int64, payload entity.Customer) error
	DeleteProfile(ctx context.Context, customerID int64) error // soft delete
}
