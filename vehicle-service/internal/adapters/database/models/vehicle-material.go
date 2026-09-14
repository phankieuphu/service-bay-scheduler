package models

import (
	"time"
)

type VehicleMaterial struct {
	ID          int64     `json:"id" gorm:"column:id;type:bigint"`
	VehicleID   int64     `json:"vehicle_id" gorm:"column:vehicle_id;type:bigint"`
	Vehicle     Vehicle   `json:"vehicle" gorm:"foreignKey:VehicleID;references:ID"`
	MaterialID  int64     `json:"material_id" gorm:"column:material_id;type:bigint"` // manage from dealership service
	Description string    `json:"description,omitempty" gorm:"column:description;type:varchar;size:255"`
	Count       int       `json:"count" gorm:"column:count;type:int;not null"`
	InstalledAt time.Time `json:"installed_at" gorm:"column:installed_at;type:datetime;not null;"`
	CreatedAt   time.Time `json:"created_at" gorm:"column:created_at;type:timestamp;autoCreateTime"`
	UpdatedAt   time.Time `json:"updated_at" gorm:"column:updated_at;type:timestamp;autoUpdateTime"`
}

func (v VehicleMaterial) TableName() string {
	return "vehicle_material"
}
