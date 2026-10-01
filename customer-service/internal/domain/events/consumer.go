package events

import "customer-service/pkg/times"

// Event types consumed from identity.user-events (architecture-design.md §3a).
const (
	UserCreated EventType = "UserCreated"
	UserUpdated EventType = "UserUpdated"
)

// UserRoleCustomer is the only identity role customer-service keeps a
// profile for; events for other roles are ignored.
const UserRoleCustomer = "CUSTOMER"

// MessageEvent is the part of the shared envelope needed to route a
// message to its handler, before its payload is decoded.
type MessageEvent struct {
	EventID   string    `json:"event_id"`
	EventType EventType `json:"event_type"`
}

// UserEvent is the full identity.user-events envelope.
type UserEvent struct {
	EventID    string     `json:"event_id"`
	EventType  EventType  `json:"event_type"`
	OccurredAt times.Time `json:"occurred_at"`
	User       UserState  `json:"user"`
}

type UserState struct {
	ID     string `json:"id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	Status string `json:"status"`
}
