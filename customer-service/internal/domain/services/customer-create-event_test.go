package services

import (
	"context"
	"customer-service/config"
	"customer-service/internal/constants"
	"customer-service/internal/domain/entity"
	"encoding/json"
	"testing"
	"time"
)

func TestCustomer_CreateCustomer_Event(t *testing.T) {
	createdAt := time.Date(2026, 9, 29, 10, 0, 0, 0, time.UTC)
	repo := &MockCustomerRepository{
		CreateFunc: func(_ context.Context, c entity.Customer) (entity.Customer, error) {
			c.ID, c.CreatedAt, c.UpdatedAt = 7, createdAt, createdAt
			return c, nil
		},
	}
	var written []entity.OutboxMessage
	outbox := &MockOutBoxRepository{CreateFunc: func(_ context.Context, m entity.OutboxMessage) error {
		written = append(written, m)
		return nil
	}}
	txManager := &MockTxMangerRepository{RunInTxFunc: func(ctx context.Context, fn func(ctx context.Context) error) error {
		return fn(ctx)
	}}
	service := NewCustomerService(config.Config{}, repo, outbox, txManager, &MockCacheRepository{})

	_, err := service.CreateCustomer(t.Context(), entity.Customer{
		Name:     "Jane",
		Email:    "jane@example.com",
		BirthDay: time.Date(1990, 5, 17, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(written) != 1 || written[0].Topic != constants.CustomerCreated || written[0].Key != "7" {
		t.Fatalf("outbox = %+v", written)
	}

	var event struct {
		EventID    string         `json:"event_id"`
		EventType  string         `json:"event_type"`
		OccurredAt string         `json:"occurred_at"`
		Customer   map[string]any `json:"customer"`
	}
	if err := json.Unmarshal(written[0].Payload, &event); err != nil {
		t.Fatalf("payload is not valid JSON: %v", err)
	}
	if event.EventID == "" || event.EventType != "CustomerCreated" || event.OccurredAt != "2026-09-29T10:00:00.000Z" {
		t.Fatalf("unexpected envelope: %s", written[0].Payload)
	}
	c := event.Customer
	if c["id"] != float64(7) || c["name"] != "Jane" || c["email"] != "jane@example.com" ||
		c["birth_day"] != "1990-05-17" || c["status"] != "ACTIVE" || c["created_at"] != "2026-09-29T10:00:00.000Z" {
		t.Fatalf("unexpected customer payload: %s", written[0].Payload)
	}
	if _, ok := c["phone"]; ok {
		t.Errorf("empty phone should be omitted: %s", written[0].Payload)
	}
}
