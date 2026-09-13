package services

import (
	"context"
	"customer-service/config"
	"customer-service/internal/constants"
	"customer-service/internal/domain/entity"
	"customer-service/internal/domain/ports"
)

// defaultListLimit is used when the caller doesn't specify a page size;
// maxListLimit caps it to keep a single page cheap to query and transfer.
const (
	defaultListLimit = 20
	maxListLimit     = 100
)

type CustomerService struct {
	config     config.Config
	repository ports.CustomerRepository
}

// CreateCustomer implements [ports.CustomerService].
func (e *CustomerService) CreateCustomer(ctx context.Context, customer entity.Customer) (entity.Customer, error) {
	customer.Status = constants.StatusActive

	customer, err := e.repository.Create(ctx, customer)
	if err != nil {

		return entity.Customer{}, err
	}

	return customer, err
}

// GetCustomer implements [ports.CustomerService].
func (e *CustomerService) GetCustomer(ctx context.Context, customerID int64) (entity.Customer, error) {
	return e.repository.GetByID(ctx, customerID)
}

// GetCustomers implements [ports.CustomerService].
func (e *CustomerService) GetCustomers(ctx context.Context, params ports.ListCustomersParams) (ports.CustomerPage, error) {
	if params.Cursor < 0 {
		params.Cursor = 0
	}
	switch {
	case params.Limit <= 0:
		params.Limit = defaultListLimit
	case params.Limit > maxListLimit:
		params.Limit = maxListLimit
	}

	return e.repository.List(ctx, params)
}

func NewCustomerService(cfg config.Config, repository ports.CustomerRepository) ports.CustomerService {
	return &CustomerService{
		repository: repository,
		config:     cfg,
	}
}
