package events

import (
	"customer-service/internal/constants"
	"customer-service/internal/domain/entity"
	"customer-service/pkg/times"
	"time"

	"github.com/google/uuid"
)

// EventType is the event_type field of every event this service publishes.
type EventType string

const (
	CustomerCreated EventType = "CustomerCreated"
	CustomerUpdated EventType = "CustomerUpdated"
	CustomerDeleted EventType = "CustomerDeleted"
)

// CustomerEvent is the envelope for every event published through the
// outbox (one topic per event type, see constants/topics.go), in the same
// shape as identity-service's identity.user-events. EventID lets consumers
// deduplicate at-least-once delivery. Fields may be added but never renamed
// or removed — see architecture-design.md §3c.
type CustomerEvent[T any] struct {
	EventID    string     `json:"event_id"`
	EventType  EventType  `json:"event_type"`
	OccurredAt times.Time `json:"occurred_at"`
	Customer   T          `json:"customer"`
}

// NewCustomerEvent stamps payload with a fresh event id and occurredAt.
func NewCustomerEvent[T any](eventType EventType, occurredAt time.Time, payload T) CustomerEvent[T] {
	return CustomerEvent[T]{
		EventID:    uuid.NewString(),
		EventType:  eventType,
		OccurredAt: times.NewTime(occurredAt),
		Customer:   payload,
	}
}

// CustomerState is the customer as stored after a create or update.
// BirthDay is a calendar date, so it's sent as "2006-01-02", not a timestamp.
type CustomerState struct {
	ID        int64                    `json:"id"`
	Name      string                   `json:"name"`
	Email     string                   `json:"email"`
	Phone     string                   `json:"phone,omitempty"`
	BirthDay  string                   `json:"birth_day,omitempty"`
	Status    constants.CustomerStatus `json:"status"`
	CreatedAt times.Time               `json:"created_at"`
	UpdatedAt times.Time               `json:"updated_at"`
}

func NewCustomerState(c entity.Customer) CustomerState {
	state := CustomerState{
		ID:        c.ID,
		Name:      c.Name,
		Email:     c.Email,
		Phone:     c.Phone,
		Status:    c.Status,
		CreatedAt: times.NewTime(c.CreatedAt),
		UpdatedAt: times.NewTime(c.UpdatedAt),
	}
	if !c.BirthDay.IsZero() {
		state.BirthDay = c.BirthDay.Format(time.DateOnly)
	}
	return state
}

// CustomerDeletion is the payload of a CustomerDeleted event.
type CustomerDeletion struct {
	ID        int64      `json:"id"`
	DeletedAt times.Time `json:"deleted_at"`
}
