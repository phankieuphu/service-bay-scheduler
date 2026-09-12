package ports

import (
	"context"
	"customer-service/internal/domain/entity"
)

type CustomerService interface {
	Save(context.Context, entity.Customer) error
}
