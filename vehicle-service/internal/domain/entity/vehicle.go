package entity

import (
	"time"
	"vehicle-service/internal/constants"
)

type Vehicle struct {
	ID              int64
	Vin             string
	LicensePlate    string
	WarrantyEndDate time.Time
	Status          constants.VehicleStatus
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
