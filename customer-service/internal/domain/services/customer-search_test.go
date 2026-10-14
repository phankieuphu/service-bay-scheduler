package services

import (
	"context"
	"customer-service/config"
	"customer-service/internal/domain/entity"
	"customer-service/internal/domain/ports"
	"testing"
	"time"
)

func TestCustomer_GetCustomers_EmailSearchSkipsCache(t *testing.T) {
	var gotParams ports.ListCustomersParams
	repo := &MockCustomerRepository{
		ListFunc: func(_ context.Context, params ports.ListCustomersParams) (ports.CustomerPage, error) {
			gotParams = params
			return ports.CustomerPage{Customers: []entity.Customer{{ID: 7, Email: "jane@example.com"}}}, nil
		},
	}
	cache := &MockCacheRepository{
		GetFunc: func(context.Context, string) (string, error) {
			t.Fatal("cache read for an email search")
			return "", nil
		},
		SetFunc: func(context.Context, string, any, time.Duration) error {
			t.Fatal("cache write for an email search")
			return nil
		},
	}
	svc := NewCustomerService(config.Config{}, repo, &MockOutBoxRepository{}, &MockTxMangerRepository{}, cache)

	page, err := svc.GetCustomers(t.Context(), ports.ListCustomersParams{Email: "  Jane@ "})
	if err != nil {
		t.Fatalf("GetCustomers: %v", err)
	}
	if len(page.Customers) != 1 || page.Customers[0].ID != 7 {
		t.Errorf("page = %+v, want customer 7", page)
	}
	if gotParams.Email != "Jane@" {
		t.Errorf("repository got email %q, want trimmed %q", gotParams.Email, "Jane@")
	}
	if gotParams.Limit != defaultListLimit {
		t.Errorf("repository got limit %d, want default %d", gotParams.Limit, defaultListLimit)
	}
}
