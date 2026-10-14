package dto

import (
	"scheduler-service/internal/constants"
	"scheduler-service/pkg/times"
)

type HubDTO struct {
	ID      int64  `json:"id"`
	Name    string `json:"name"`
	Address string `json:"address"`
	// OpenHour and CloseHour are "HH:MM" in TimeZone.
	OpenHour  string `json:"open_hour"`
	CloseHour string `json:"close_hour"`
	TimeZone  string `json:"timezone"`
	BayCount  int    `json:"bay_count"`
}

type HubsResponseDTO struct {
	Hubs []HubDTO `json:"hubs"`
}

// ServiceDTO is a bookable service. Fee is a decimal string, e.g. "45.00".
type ServiceDTO struct {
	ID              int64  `json:"id"`
	Name            string `json:"name"`
	ServiceType     string `json:"service_type"`
	Fee             string `json:"fee"`
	DurationMinutes int    `json:"duration_minutes"`
}

type ServicesResponseDTO struct {
	Services []ServiceDTO `json:"services"`
}

type SlotDTO struct {
	Start     times.Time `json:"start"`
	End       times.Time `json:"end"`
	Available bool       `json:"available"`
}

type AvailabilityResponseDTO struct {
	HubID           int64     `json:"hub_id"`
	Date            string    `json:"date"`
	TimeZone        string    `json:"timezone"`
	DurationMinutes int       `json:"duration_minutes"`
	Slots           []SlotDTO `json:"slots"`
}

// CreateAppointmentDTO books VehicleID at HubID for ServiceIDs, starting at
// Start (RFC3339, any offset).
type CreateAppointmentDTO struct {
	HubID      int64      `json:"hub_id" binding:"required,gt=0"`
	VehicleID  int64      `json:"vehicle_id" binding:"required,gt=0"`
	ServiceIDs []int64    `json:"service_ids" binding:"required,min=1,max=10,dive,gt=0"`
	Start      times.Time `json:"start" binding:"required"`
}

type AppointmentServiceDTO struct {
	ServiceID int64  `json:"service_id"`
	Name      string `json:"name"`
	Fee       string `json:"fee"`
}

type AppointmentDTO struct {
	ID          int64                       `json:"id"`
	HubID       int64                       `json:"hub_id"`
	CustomerID  int64                       `json:"customer_id"`
	VehicleID   int64                       `json:"vehicle_id"`
	BayID       int64                       `json:"bay_id"`
	Start       times.Time                  `json:"start"`
	End         times.Time                  `json:"end"`
	Status      constants.AppointmentStatus `json:"status"`
	TotalAmount string                      `json:"total_amount"`
	Services    []AppointmentServiceDTO     `json:"services"`
	CreatedAt   times.Time                  `json:"created_at"`
	UpdatedAt   times.Time                  `json:"updated_at"`
}

type AppointmentsResponseDTO struct {
	Appointments []AppointmentDTO `json:"appointments"`
}
