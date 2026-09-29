package events

import (
	"time"
	"vehicle-service/internal/domain/entity"

	"github.com/google/uuid"
)

// EventType is the event_type field of every event this service publishes.
type EventType string

const (
	VehicleCreated     EventType = "VehicleCreated"
	VehicleUpdated     EventType = "VehicleUpdated"
	WarrantyChanged    EventType = "WarrantyChanged"
	OwnerAssigned      EventType = "OwnerAssigned"
	VehicleTransferred EventType = "VehicleTransferred"
)

// VehicleEvent is the envelope for every event published through the
// outbox (one topic per event type, see constants/topics.go), in the same
// shape as identity-service's identity.user-events. EventID lets consumers
// deduplicate at-least-once delivery. Calendar dates (warranty end,
// ownership start) are sent as "2006-01-02" strings, not timestamps. Fields
// may be added but never renamed or removed — see architecture-design.md
// §3b.
type VehicleEvent[T any] struct {
	EventID    string    `json:"event_id"`
	EventType  EventType `json:"event_type"`
	OccurredAt time.Time `json:"occurred_at"`
	Vehicle    T         `json:"vehicle"`
}

// NewVehicleEvent stamps payload with a fresh event id and occurredAt.
func NewVehicleEvent[T any](eventType EventType, occurredAt time.Time, payload T) VehicleEvent[T] {
	return VehicleEvent[T]{
		EventID:    uuid.NewString(),
		EventType:  eventType,
		OccurredAt: occurredAt.UTC(),
		Vehicle:    payload,
	}
}

// VehicleState is the vehicle as stored after a create or update; it's the
// payload of VehicleCreated and VehicleUpdated.
type VehicleState struct {
	ID              int64     `json:"id"`
	Vin             string    `json:"vin"`
	LicensePlate    string    `json:"license_plate,omitempty"`
	VehicleModelID  int64     `json:"vehicle_model_id"`
	WarrantyEndDate *string   `json:"warranty_end_date"`
	Status          string    `json:"status"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// WarrantyChange is the payload of WarrantyChanged, published in addition
// to VehicleUpdated when warranty_end_date changes. billing-service reads
// it to price the next bill.
type WarrantyChange struct {
	ID                      int64   `json:"id"`
	PreviousWarrantyEndDate *string `json:"previous_warranty_end_date"`
	WarrantyEndDate         *string `json:"warranty_end_date"`
}

// Ownership is the payload of OwnerAssigned (a vehicle's first owner) and
// VehicleTransferred (every later change of owner, when
// PreviousCustomerID is set).
type Ownership struct {
	ID                 int64  `json:"id"`
	CustomerID         int64  `json:"customer_id"`
	PreviousCustomerID *int64 `json:"previous_customer_id,omitempty"`
	OwnedFrom          string `json:"owned_from"`
}

// ServiceCompleted is consumed from scheduler-service (config
// Kafka.ServiceCompletedTopic). scheduler-service doesn't exist yet; this
// is the contract it's expected to publish — see architecture-design.md
// §3b.
type ServiceCompleted struct {
	EventID       string                    `json:"event_id"`
	OccurredAt    time.Time                 `json:"occurred_at"`
	AppointmentID int64                     `json:"appointment_id"`
	VehicleID     int64                     `json:"vehicle_id"`
	CustomerID    int64                     `json:"customer_id"`
	DealershipID  int64                     `json:"dealership_id"`
	CompletedAt   time.Time                 `json:"completed_at"`
	Services      []entity.ServicePerformed `json:"services"`
}
