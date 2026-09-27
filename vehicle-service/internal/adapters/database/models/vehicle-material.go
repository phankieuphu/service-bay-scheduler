package models

import (
	"time"
)

// VehicleMaterial mirrors vehicle_material, which has no created_at /
// updated_at columns: installed_at is the only timestamp.
type VehicleMaterial struct {
	ID          int64     `json:"id" gorm:"column:id;type:bigint"`
	VehicleID   int64     `json:"vehicle_id" gorm:"column:vehicle_id;type:bigint"`   // index
	MaterialID  int64     `json:"material_id" gorm:"column:material_id;type:bigint"` // manage from dealership service
	Description *string   `json:"description,omitempty" gorm:"column:description;type:varchar;size:255"`
	Count       int       `json:"count" gorm:"column:count;type:int;not null"`
	InstalledAt time.Time `json:"installed_at" gorm:"column:installed_at;type:timestamptz;not null"`
}

func (v VehicleMaterial) TableName() string {
	return "vehicle_material"
}
