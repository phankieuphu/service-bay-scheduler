package models

import (
	"time"
	"vehicle-service/internal/constants"
)

type Vehicle struct {
	ID  int64  `json:"id" gorm:"column:id;type:bigint"`
	Vin string `json:"vin" gorm:"column:vin;type:varchar;size:17"`
	// Pointer so a vehicle without a plate is stored as NULL, not '': the
	// unique plate index would otherwise let only one plate-less vehicle in.
	LicensePlate   *string `json:"license_plate" gorm:"column:license_plate;type:varchar;size:20"`
	VehicleModelID int64   `json:"vehicle_model_id" gorm:"column:vehicle_model_id;type:bigint"`
	// Pointer so a missing warranty round-trips as NULL instead of
	// 0001-01-01.
	WarrantyEndDate *time.Time              `json:"warranty_end_date" gorm:"column:warranty_end_date;type:date"`
	Status          constants.VehicleStatus `json:"status" gorm:"column:status;type:varchar;size:20"`
	CreatedAt       time.Time               `json:"created_at" gorm:"column:created_at;type:timestamptz;autoCreateTime"`
	UpdatedAt       time.Time               `json:"updated_at" gorm:"column:updated_at;type:timestamptz;autoUpdateTime"`
}

func (v Vehicle) TableName() string {
	return "vehicle"
}
