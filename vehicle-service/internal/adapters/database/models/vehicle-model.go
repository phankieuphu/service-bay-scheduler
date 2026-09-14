package models

import (
	"time"
	"vehicle-service/internal/constants"
)

type Vehicle struct {
	ID              int64                   `json:"id" gorm:"column:id;type:bigint"`
	Vin             string                  `json:"vin" gorm:"column:vin;type:varchar;size:17"`
	LicensePlate    string                  `json:"license_plate" gorm:"column:license_plate;type:varchar;size:20"`
	WarrantyEndDate time.Time               `json:"warranty_end_date" gorm:"column:warranty_end_date;type:datetime"`
	Status          constants.VehicleStatus `json:"status" gorm:"column:status;type:varchar;size:20"`
	CreatedAt       time.Time               `json:"created_at" gorm:"column:created_at;type:timestamp;autoCreateTime"`
	UpdatedAt       time.Time               `json:"updated_at" gorm:"column:updated_at;type:timestamp;autoUpdateTime"`
}

func (v Vehicle) TableName() string {
	return "vehicle"
}
