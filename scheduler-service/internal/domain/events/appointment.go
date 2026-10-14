package events

import (
	"scheduler-service/internal/constants"
	"scheduler-service/internal/domain/entity"
	"scheduler-service/pkg/times"
	"time"

	"github.com/google/uuid"
)

// EventType is the event_type field of every event this service publishes.
type EventType string

const (
	AppointmentBooked    EventType = "AppointmentBooked"
	AppointmentCancelled EventType = "AppointmentCancelled"
)

// AppointmentEvent is the envelope for every event published through the
// outbox (one topic per event type, see constants/topics.go), in the same
// shape as identity.user-events. EventID lets consumers deduplicate
// at-least-once delivery. Fields may be added but never renamed or removed
// — see architecture-design.md §3d.
type AppointmentEvent struct {
	EventID     string           `json:"event_id"`
	EventType   EventType        `json:"event_type"`
	OccurredAt  times.Time       `json:"occurred_at"`
	Appointment AppointmentState `json:"appointment"`
}

// AppointmentState is the appointment after the change. dealership_id is
// the hub, named as in the ServiceCompleted contract (§3b).
type AppointmentState struct {
	ID             int64                       `json:"id"`
	DealershipID   int64                       `json:"dealership_id"`
	CustomerID     int64                       `json:"customer_id"`
	VehicleID      int64                       `json:"vehicle_id"`
	BayID          int64                       `json:"bay_id"`
	ScheduledStart times.Time                  `json:"scheduled_start"`
	ScheduledEnd   times.Time                  `json:"scheduled_end"`
	Status         constants.AppointmentStatus `json:"status"`
	// TotalAmount is a decimal string, e.g. "45.00", so no precision is lost.
	TotalAmount string         `json:"total_amount"`
	Services    []ServiceState `json:"services"`
}

type ServiceState struct {
	ServiceID int64  `json:"service_id"`
	Name      string `json:"name"`
	Fee       string `json:"fee"`
}

// NewAppointmentEvent stamps the appointment's state with a fresh event id
// and occurredAt.
func NewAppointmentEvent(eventType EventType, occurredAt time.Time, a entity.Appointment) AppointmentEvent {
	services := make([]ServiceState, len(a.Services))
	for i, s := range a.Services {
		services[i] = ServiceState{ServiceID: s.ServiceID, Name: s.Name, Fee: s.Fee.StringFixed(2)}
	}
	return AppointmentEvent{
		EventID:    uuid.NewString(),
		EventType:  eventType,
		OccurredAt: times.NewTime(occurredAt),
		Appointment: AppointmentState{
			ID:             a.ID,
			DealershipID:   a.HubID,
			CustomerID:     a.CustomerID,
			VehicleID:      a.VehicleID,
			BayID:          a.BayID,
			ScheduledStart: times.NewTime(a.Start),
			ScheduledEnd:   times.NewTime(a.End),
			Status:         a.Status,
			TotalAmount:    a.TotalAmount.StringFixed(2),
			Services:       services,
		},
	}
}
