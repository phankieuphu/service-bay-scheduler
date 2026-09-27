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
