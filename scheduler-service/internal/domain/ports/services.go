package ports

import (
	"context"
	"scheduler-service/internal/domain/entity"
	"time"
)

// BookingRequest is a customer asking for one vehicle to be serviced at a
// hub, starting at Start.
type BookingRequest struct {
	HubID      int64
	VehicleID  int64
	ServiceIDs []int64
	Start      time.Time
}

// Slot is one possible start time on a day. Available is false when it's in
// the past or every bay is taken for some part of it.
type Slot struct {
	Start     time.Time
	End       time.Time
	Available bool
}

// Availability is a hub's bookable slots on one local day for a set of
// services, whose durations add up to DurationMinutes.
type Availability struct {
	HubID           int64
	Date            string
	TimeZone        string
	DurationMinutes int
	Slots           []Slot
}

// HubDay is a hub's appointments on one local day.
type HubDay struct {
	HubID        int64
	Date         string
	TimeZone     string
	Appointments []entity.Appointment
}

type BookingService interface {
	ListHubs(ctx context.Context) ([]entity.Hub, error)
	ListServices(ctx context.Context) ([]entity.CatalogService, error)
	// Availability lists the slots on date (YYYY-MM-DD, in the hub's time
	// zone) for the given services.
	Availability(ctx context.Context, hubID int64, date string, serviceIDs []int64) (Availability, error)
	Book(ctx context.Context, caller entity.Principal, req BookingRequest) (entity.Appointment, error)
	// MyAppointments lists the caller's own appointments.
	MyAppointments(ctx context.Context, caller entity.Principal) ([]entity.Appointment, error)
	// Cancel cancels one of the caller's own upcoming appointments.
	Cancel(ctx context.Context, caller entity.Principal, appointmentID int64) (entity.Appointment, error)
	// HubDay lists a hub's appointments on date (YYYY-MM-DD, in the hub's
	// time zone), earliest first. Staff only.
	HubDay(ctx context.Context, hubID int64, date string) (HubDay, error)
	// Complete marks an appointment's work done and publishes
	// ServiceCompleted, which billing-service prices the bill from. Staff
	// only.
	Complete(ctx context.Context, appointmentID int64) (entity.Appointment, error)
}
