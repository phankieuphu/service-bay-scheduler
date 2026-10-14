package entity

import (
	"scheduler-service/internal/constants"
	"time"

	"github.com/shopspring/decimal"
)

// Appointment is one booking: one vehicle in one bay at one hub for the
// services listed. TotalAmount is the sum of the services' fees at booking
// time, an estimate until billing-service bills it.
type Appointment struct {
	ID          int64
	HubID       int64
	CustomerID  int64
	VehicleID   int64
	BayID       int64
	Start       time.Time
	End         time.Time
	Status      constants.AppointmentStatus
	TotalAmount decimal.Decimal
	Services    []AppointmentService
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// AppointmentService is a booked service with the fee charged for it.
type AppointmentService struct {
	ServiceID int64
	Name      string
	Fee       decimal.Decimal
}
