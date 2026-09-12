package ports

import (
	"context"
	"customer-service/internal/domain/entity"
)

type CustomerRepository interface {
	Create(ctx context.Context, customer entity.Customer)
}
