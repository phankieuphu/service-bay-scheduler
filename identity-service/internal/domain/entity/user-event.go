package entity

import (
	"identity-service/internal/constants"
	"time"
)

// UserEvent is the JSON payload published to identity.user-events, keyed
// by user ID so every event for one user lands on the same partition in
// order. Consumers (customer-service, vehicle-service) parse this shape,
// so fields may be added but never renamed or removed.
type UserEvent struct {
	EventID    string                  `json:"event_id"`
	EventType  constants.UserEventType `json:"event_type"`
	OccurredAt time.Time               `json:"occurred_at"`
	User       UserEventPayload        `json:"user"`
}

// UserEventPayload deliberately omits the password hash.
type UserEventPayload struct {
	ID     string               `json:"id"`
	Email  string               `json:"email"`
	Role   constants.Role       `json:"role"`
	Status constants.UserStatus `json:"status"`
}
