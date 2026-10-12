package constants

// UserEventsTopic carries every user lifecycle event. customer-service and
// vehicle-service consume it (their KAFKA_CONSUMER_TOPIC default), so the
// name is a cross-service contract — see docs/architecture-design.md.
const UserEventsTopic = "identity.user-events"

// UserEventType is the event_type field of a message on UserEventsTopic.
type UserEventType string

const (
	UserCreated UserEventType = "UserCreated"
	// UserUpdated is emitted when an admin changes a user's status
	// (PATCH /admin/users/:id).
	UserUpdated UserEventType = "UserUpdated"
)
