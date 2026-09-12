package repository

import (
	"context"
	"customer-service/internal/adapters/database/models"
	"customer-service/internal/domain/entity"
	"customer-service/internal/domain/ports"

	"gorm.io/gorm"
)

type CustomerRepository struct {
	db *gorm.DB
}

// Create implements ports.CustomerRepository.
func (a CustomerRepository) Create(ctx context.Context, customer entity.Customer) {
	models := a.toModels(customer)
	a.db.Save(models)
	panic("unimplemented")
}

func (a CustomerRepository) toModels(customer entity.Customer) models.Customer {
	panic("unimplemented")
}

func (a CustomerRepository) toDomain(model models.Customer) entity.Customer {
	panic("unimplemented")
}

func NewCustomerRepository(db *gorm.DB) ports.CustomerRepository {
	return CustomerRepository{
		db: db,
	}
}
