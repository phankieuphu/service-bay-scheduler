package entity

import "time"

// ServiceHistoryEntry is one completed appointment in a vehicle's service
// history, as reported by scheduler-service's ServiceCompleted event.
type ServiceHistoryEntry struct {
	ID            int64
	VehicleID     int64
	AppointmentID int64
	DealershipID  int64
	CompletedAt   time.Time
	Services      []ServicePerformed
	CreatedAt     time.Time
}

// ServicePerformed is a copy of a catalog service as it was when the work
// was done, so renaming it in dealership-service doesn't rewrite history.
type ServicePerformed struct {
	ServiceID int64  `json:"service_id"`
	Name      string `json:"name"`
}
