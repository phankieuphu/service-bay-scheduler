package services

import (
	"context"
	"customer-service/config"
	"customer-service/internal/constants"
	"customer-service/internal/domain/entity"
	"customer-service/internal/domain/ports"
	"encoding/json"
	"errors"
	"testing"
)

func newLinkUserService(repo *MockCustomerRepository, written *[]entity.OutboxMessage) ports.CustomerService {
	outbox := &MockOutBoxRepository{CreateFunc: func(_ context.Context, m entity.OutboxMessage) error {
		*written = append(*written, m)
		return nil
	}}
	txManager := &MockTxMangerRepository{RunInTxFunc: func(ctx context.Context, fn func(ctx context.Context) error) error {
		return fn(ctx)
	}}
	return NewCustomerService(config.Config{}, repo, outbox, txManager, &MockCacheRepository{})
}

func notLinked(context.Context, string) (entity.Customer, error) {
	return entity.Customer{}, ports.ErrNotFound
}

func TestCustomer_LinkUser_AlreadyLinked(t *testing.T) {
	repo := &MockCustomerRepository{
		GetByUserIDFunc: func(context.Context, string) (entity.Customer, error) {
			return entity.Customer{ID: 3, UserID: "u-1"}, nil
		},
	}
	var written []entity.OutboxMessage

	got, err := newLinkUserService(repo, &written).LinkUser(t.Context(), "u-1", "jane@example.com")
	if err != nil || got.ID != 3 {
		t.Fatalf("LinkUser = %+v, %v; want customer 3", got, err)
	}
	if len(written) != 0 {
		t.Errorf("an already-linked user wrote %d outbox rows, want 0", len(written))
	}
}

func TestCustomer_LinkUser_ClaimsExistingCustomer(t *testing.T) {
	repo := &MockCustomerRepository{
		GetByUserIDFunc: notLinked,
		LinkUserByEmailFunc: func(_ context.Context, userID, email string) (entity.Customer, error) {
			return entity.Customer{ID: 5, UserID: userID, Email: email, Name: "Jane Doe"}, nil
		},
		CreateFunc: func(context.Context, entity.Customer) (entity.Customer, error) {
			t.Fatal("Create called although a customer with this email exists")
			return entity.Customer{}, nil
		},
	}
	var written []entity.OutboxMessage

	got, err := newLinkUserService(repo, &written).LinkUser(t.Context(), "u-1", "jane@example.com")
	if err != nil || got.ID != 5 || got.Name != "Jane Doe" {
		t.Fatalf("LinkUser = %+v, %v", got, err)
	}
	if len(written) != 1 || written[0].Topic != constants.CustomerUpdate || written[0].Key != "5" {
		t.Fatalf("outbox = %+v, want one CustomerUpdated row", written)
	}
	var event struct {
		EventType string         `json:"event_type"`
		Customer  map[string]any `json:"customer"`
	}
	if err := json.Unmarshal(written[0].Payload, &event); err != nil {
		t.Fatal(err)
	}
	if event.EventType != "CustomerUpdated" || event.Customer["user_id"] != "u-1" {
		t.Errorf("payload = %s", written[0].Payload)
	}
}

func TestCustomer_LinkUser_CreatesProfile(t *testing.T) {
	var created entity.Customer
	repo := &MockCustomerRepository{
		GetByUserIDFunc: notLinked,
		LinkUserByEmailFunc: func(context.Context, string, string) (entity.Customer, error) {
			return entity.Customer{}, ports.ErrNotFound
		},
		CreateFunc: func(_ context.Context, c entity.Customer) (entity.Customer, error) {
			created = c
			c.ID = 9
			return c, nil
		},
	}
	var written []entity.OutboxMessage

	got, err := newLinkUserService(repo, &written).LinkUser(t.Context(), "u-2", "sam.lee@example.com")
	if err != nil || got.ID != 9 {
		t.Fatalf("LinkUser = %+v, %v", got, err)
	}
	if created.UserID != "u-2" || created.Email != "sam.lee@example.com" || created.Name != "sam.lee" ||
		created.Status != constants.StatusActive || !created.BirthDay.IsZero() {
		t.Errorf("created = %+v", created)
	}
	if len(written) != 1 || written[0].Topic != constants.CustomerCreated || written[0].Key != "9" {
		t.Fatalf("outbox = %+v, want one CustomerCreated row", written)
	}
}

func TestCustomer_LinkUser_EmailTaken(t *testing.T) {
	repo := &MockCustomerRepository{
		GetByUserIDFunc: notLinked,
		LinkUserByEmailFunc: func(context.Context, string, string) (entity.Customer, error) {
			return entity.Customer{}, ports.ErrNotFound
		},
		CreateFunc: func(context.Context, entity.Customer) (entity.Customer, error) {
			return entity.Customer{}, ports.ErrConflict
		},
	}
	var written []entity.OutboxMessage

	_, err := newLinkUserService(repo, &written).LinkUser(t.Context(), "u-3", "taken@example.com")
	if !errors.Is(err, ports.ErrConflict) {
		t.Fatalf("err = %v, want ErrConflict", err)
	}
	if len(written) != 0 {
		t.Errorf("wrote %d outbox rows for a failed link", len(written))
	}
}
