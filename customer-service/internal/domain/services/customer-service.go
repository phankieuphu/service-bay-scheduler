package services

import (
	"context"
	"customer-service/config"
	"customer-service/internal/domain/entity"
	"customer-service/internal/domain/ports"
)

type CustomerService struct {
	config     config.Config
	repository ports.CustomerRepository
}

// Save implements ports.CustomerService.
func (e *CustomerService) Save(ctx context.Context, entity entity.Customer) error {
	// Handle business here:

	e.repository.Create(ctx, entity)

	return nil
}

func NewCustomerService(cfg config.Config, repository ports.CustomerRepository) ports.CustomerService {
	return &CustomerService{
		repository: repository,
		config:     cfg,
	}
}
