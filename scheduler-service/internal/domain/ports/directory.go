package ports

import (
	"context"
	"scheduler-service/internal/domain/entity"
)

// CustomerDirectory asks customer-service who the caller is. token is the
// caller's own access token.
type CustomerDirectory interface {
	// CurrentCustomer returns the caller's linked profile, or
	// ErrNoCustomerProfile.
	CurrentCustomer(ctx context.Context, token string) (entity.Customer, error)
}

// VehicleDirectory asks vehicle-service which vehicles a customer owns.
type VehicleDirectory interface {
	CustomerVehicles(ctx context.Context, token string, customerID int64) ([]entity.Vehicle, error)
}
