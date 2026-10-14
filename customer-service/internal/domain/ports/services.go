package ports

import (
	"context"
	"customer-service/internal/domain/entity"
)

type CustomerService interface {
	CreateCustomer(ctx context.Context, customer entity.Customer) (entity.Customer, error)
	GetCustomer(ctx context.Context, customerID int64) (entity.Customer, error)
	// GetCustomerByUserID returns the profile linked to an identity user.
	GetCustomerByUserID(ctx context.Context, userID string) (entity.Customer, error)
	// LinkUser gives a CUSTOMER identity user a profile: it links the
	// existing customer with the same email, or creates one. Safe to call
	// again for a user that is already linked.
	LinkUser(ctx context.Context, userID, email string) (entity.Customer, error)
	GetCustomers(ctx context.Context, params ListCustomersParams) (CustomerPage, error)
	UpdateProfile(ctx context.Context, customerID int64, payload entity.Customer) error
	DeleteProfile(ctx context.Context, customerID int64) error // soft delete
}
