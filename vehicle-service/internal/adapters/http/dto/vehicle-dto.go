package dto

import (
	"time"
	"vehicle-service/internal/constants"
	"vehicle-service/internal/domain/entity"
)

type VehicleResponseDTO struct {
	ID              int64                   `json:"id"`
	Vin             string                  `json:"vin"`
	LicensePlate    string                  `json:"license_plate"`
	WarrantyEndDate time.Time               `json:"warranty_end_date"`
	Status          constants.VehicleStatus `json:"status"`
	CreatedAt       time.Time               `json:"created_at"`
	UpdatedAt       time.Time               `json:"updated_at"`
}

func NewVehicleResponseDTO(v entity.Vehicle) VehicleResponseDTO {
	return VehicleResponseDTO{
		ID:              v.ID,
		Vin:             v.Vin,
		LicensePlate:    v.LicensePlate,
		WarrantyEndDate: v.WarrantyEndDate,
		Status:          v.Status,
		CreatedAt:       v.CreatedAt,
		UpdatedAt:       v.UpdatedAt,
	}
}
