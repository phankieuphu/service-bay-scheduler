package ports

import (
	"context"
	"customer-service/internal/domain/entity"
)

type CustomerRepository interface {
	Create(ctx context.Context, customer entity.Customer) (entity.Customer, error)
	GetByID(ctx context.Context, id int64) (entity.Customer, error)
	// GetByUserID finds the customer linked to an identity-service user.
	GetByUserID(ctx context.Context, userID string) (entity.Customer, error)
	// LinkUserByEmail sets user_id on the unlinked customer with this email
	// and returns it, or returns ErrNotFound if there is none.
	LinkUserByEmail(ctx context.Context, userID, email string) (entity.Customer, error)
	List(ctx context.Context, params ListCustomersParams) (CustomerPage, error)
	Update(ctx context.Context, customer entity.Customer) error
	Delete(ctx context.Context, id int64) error
	SoftDelete(ctx context.Context, id int64) error
}
