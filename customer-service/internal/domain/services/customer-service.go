package services

import (
	"context"
	"customer-service/config"
	"customer-service/internal/constants"
	"customer-service/internal/domain/entity"
	"customer-service/internal/domain/ports"
	"customer-service/pkg/logger"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"
)

// defaultListLimit is used when the caller doesn't specify a page size;
// maxListLimit caps it to keep a single page cheap to query and transfer.
const (
	defaultListLimit = 20
	maxListLimit     = 100
)

// customersPageCacheTTL bounds how stale a cached listing page can be; kept
// short since customer creation/updates don't invalidate this cache.
const customersPageCacheTTL = 30 * time.Second

// customersPageCacheKey formats to e.g. "customers:list:cursor=0:limit=20".
const customersPageCacheKeyFormat = "customers:list:cursor=%d:limit=%d"

type CustomerService struct {
	config     config.Config
	repository ports.CustomerRepository
	outbox     ports.OutboxRepository
	txManager  ports.TxManager
	cache      ports.Cache
}

// CreateCustomer implements [ports.CustomerService]. The customer row and
// its CustomerCreated outbox row are written in one transaction, so the
// event can never be lost or published for a write that got rolled back.
// A separate relay (see kafka.OutboxRelay) delivers outbox rows to Kafka
// afterwards — nothing here talks to Kafka directly.
func (e *CustomerService) CreateCustomer(ctx context.Context, customer entity.Customer) (entity.Customer, error) {
	customer.Status = constants.StatusActive

	err := e.txManager.RunInTx(ctx, func(ctx context.Context) error {
		created, err := e.repository.Create(ctx, customer)
		if err != nil {
			return err
		}
		customer = created

		payload, err := json.Marshal(customer)
		if err != nil {
			return err
		}

		return e.outbox.Create(ctx, entity.OutboxMessage{
			Topic:   constants.CustomerCreated,
			Key:     strconv.FormatInt(customer.ID, 10),
			Payload: payload,
		})
	})
	if err != nil {
		logger.ErrorContext(ctx, "failed to create customer", "error", err)
		return entity.Customer{}, err
	}

	return customer, nil
}

// GetCustomer implements [ports.CustomerService].
func (e *CustomerService) GetCustomer(ctx context.Context, customerID int64) (entity.Customer, error) {
	customer, err := e.repository.GetByID(ctx, customerID)
	if err != nil {
		logger.ErrorContext(ctx, "failed to get customer", "customer_id", customerID, "error", err)
		return entity.Customer{}, err
	}
	return customer, nil
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

	cacheKey := fmt.Sprintf(customersPageCacheKeyFormat, params.Cursor, params.Limit)

	if cached, err := e.cache.Get(ctx, cacheKey); err == nil {
		var page ports.CustomerPage
		if err := json.Unmarshal([]byte(cached), &page); err == nil {
			return page, nil
		}
	} else if !errors.Is(err, ports.ErrCacheMiss) {
		logger.ErrorContext(ctx, "failed to read customers page from cache", "cursor", params.Cursor, "limit", params.Limit, "error", err)
	}

	page, err := e.repository.List(ctx, params)
	if err != nil {
		logger.ErrorContext(ctx, "failed to list customers", "cursor", params.Cursor, "limit", params.Limit, "error", err)
		return ports.CustomerPage{}, err
	}

	if payload, err := json.Marshal(page); err == nil {
		if err := e.cache.Set(ctx, cacheKey, payload, customersPageCacheTTL); err != nil {
			logger.ErrorContext(ctx, "failed to cache customers page", "cursor", params.Cursor, "limit", params.Limit, "error", err)
		}
	}

	return page, nil
}

func NewCustomerService(cfg config.Config, repository ports.CustomerRepository, outbox ports.OutboxRepository, txManager ports.TxManager, cache ports.Cache) ports.CustomerService {
	return &CustomerService{
		repository: repository,
		config:     cfg,
		outbox:     outbox,
		txManager:  txManager,
		cache:      cache,
	}
}
