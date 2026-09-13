package ports

import (
	"context"
	"customer-service/internal/domain/entity"
)

type CustomerRepository interface {
	Create(ctx context.Context, customer entity.Customer) (entity.Customer, error)
	GetByID(ctx context.Context, id int64) (entity.Customer, error)
	List(ctx context.Context, params ListCustomersParams) (CustomerPage, error)
	Update(ctx context.Context, customer entity.Customer) error
	Delete(ctx context.Context, id int64) error
}
