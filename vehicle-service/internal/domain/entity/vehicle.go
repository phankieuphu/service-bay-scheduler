package entity

import (
	"time"
	"vehicle-service/internal/constants"
)

type Vehicle struct {
	ID             int64
	Vin            string
	LicensePlate   string // "" when the vehicle has no plate yet
	VehicleModelID int64
	// WarrantyEndDate is a calendar date (midnight UTC), or nil when the
	// vehicle has no warranty on record.
	WarrantyEndDate *time.Time
	Status          constants.VehicleStatus
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
