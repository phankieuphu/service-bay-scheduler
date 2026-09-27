package dto

import (
	"time"
	"vehicle-service/internal/constants"
	"vehicle-service/internal/domain/entity"
)

// RegisterVehicleDTO is the body of POST /vehicle. VIN and plate are
// normalized server-side (uppercased; the plate keeps only A-Z and 0-9).
// WarrantyEndDate is a date: only its calendar day, in the offset given,
// is kept.
type RegisterVehicleDTO struct {
	Vin             string                  `json:"vin" binding:"required"`
	LicensePlate    string                  `json:"license_plate"`
	VehicleModelID  int64                   `json:"vehicle_model_id" binding:"required,gt=0"`
	WarrantyEndDate *time.Time              `json:"warranty_end_date"`
	Status          constants.VehicleStatus `json:"status"` // defaults to ACTIVE
}

// UpdateVehicleDTO is the body of PATCH /vehicle/:id. Omitted fields are
// left unchanged. UpdatedAt must be the updated_at the client last read
// (from GET /vehicle/:id); the update is rejected with 409 if the vehicle
// has changed since.
type UpdateVehicleDTO struct {
	Status          *constants.VehicleStatus `json:"status"`
	WarrantyEndDate *time.Time               `json:"warranty_end_date"`
	UpdatedAt       time.Time                `json:"updated_at" binding:"required"`
}

type VehicleResponseDTO struct {
	ID             int64  `json:"id"`
	Vin            string `json:"vin"`
	LicensePlate   string `json:"license_plate"`
	VehicleModelID int64  `json:"vehicle_model_id"`
	// WarrantyEndDate is null when the vehicle has no warranty on record.
	WarrantyEndDate *time.Time              `json:"warranty_end_date"`
	Status          constants.VehicleStatus `json:"status"`
	CreatedAt       time.Time               `json:"created_at"`
	UpdatedAt       time.Time               `json:"updated_at"`
}

func NewVehicleResponseDTO(v entity.Vehicle) VehicleResponseDTO {
	return VehicleResponseDTO{
		ID:              v.ID,
		Vin:             v.Vin,
		LicensePlate:    v.LicensePlate,
		VehicleModelID:  v.VehicleModelID,
		WarrantyEndDate: v.WarrantyEndDate,
		Status:          v.Status,
		CreatedAt:       v.CreatedAt,
		UpdatedAt:       v.UpdatedAt,
	}
}

// ListVehiclesResponseDTO is a single cursor-paginated page of vehicles.
// NextCursor is only set when HasMore is true; pass it back as the `cursor`
// query param to fetch the next page.
type ListVehiclesResponseDTO struct {
	Vehicles   []VehicleResponseDTO `json:"vehicles"`
	NextCursor int64                `json:"next_cursor,omitempty"`
	HasMore    bool                 `json:"has_more"`
}

// CustomerVehicleDTO is a vehicle the customer currently owns, plus the
// day that ownership started.
type CustomerVehicleDTO struct {
	VehicleResponseDTO
	OwnedFrom time.Time `json:"owned_from"`
}

type CustomerVehiclesResponseDTO struct {
	Vehicles []CustomerVehicleDTO `json:"vehicles"`
}

type WarrantyResponseDTO struct {
	VehicleID       int64      `json:"vehicle_id"`
	WarrantyEndDate *time.Time `json:"warranty_end_date"` // null: no warranty on record
	AsOf            string     `json:"as_of"`             // YYYY-MM-DD the status is for
	InWarranty      bool       `json:"in_warranty"`
	DaysRemaining   int        `json:"days_remaining"`
}

func NewWarrantyResponseDTO(w entity.Warranty) WarrantyResponseDTO {
	return WarrantyResponseDTO{
		VehicleID:       w.VehicleID,
		WarrantyEndDate: w.EndDate,
		AsOf:            w.AsOf.Format(time.DateOnly),
		InWarranty:      w.Active,
		DaysRemaining:   w.DaysRemaining,
	}
}

type VehicleMaterialDTO struct {
	ID          int64     `json:"id"`
	MaterialID  int64     `json:"material_id"`
	Description string    `json:"description,omitempty"`
	Count       int       `json:"count"`
	InstalledAt time.Time `json:"installed_at"`
}

type VehicleMaterialsResponseDTO struct {
	Materials []VehicleMaterialDTO `json:"materials"`
}

type ServiceHistoryEntryDTO struct {
	ID            int64                     `json:"id"`
	AppointmentID int64                     `json:"appointment_id"`
	DealershipID  int64                     `json:"dealership_id"`
	CompletedAt   time.Time                 `json:"completed_at"`
	Services      []entity.ServicePerformed `json:"services"`
}

type ServiceHistoryResponseDTO struct {
	History []ServiceHistoryEntryDTO `json:"history"`
}
