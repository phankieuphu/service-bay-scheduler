package models

import (
	"time"
	"vehicle-service/internal/constants"
)

type CustomerVehicle struct {
	ID         int64                   `json:"id" gorm:"column:id;type:bigint"`
	CustomerID int64                   `json:"customer_id" gorm:"column:customer_id;type:bigint"` // from customer service don't have references
	VehicleID  int64                   `json:"vehicle_id" gorm:"column:vehicle_id;type:bigint"`
	Vehicle    Vehicle                 `json:"vehicle" gorm:"foreignKey:VehicleID;references:ID"`
	OwnedFrom  time.Time               `json:"owned_from" gorm:"column:owned_from;type:datetime"`
	OwnedTo    time.Time               `json:"owned_to" gorm:"column:owned_from;type:datetime"`
	Status     constants.VehicleStatus `json:"status" gorm:"column:status;type:varchar;size:20"`
	CreatedAt  time.Time               `json:"created_at" gorm:"column:created_at;type:timestamp;autoCreateTime"`
	UpdatedAt  time.Time               `json:"updated_at" gorm:"column:updated_at;type:timestamp;autoUpdateTime"`
}

func (c CustomerVehicle) TableName() string {
	return "customer-vehicle"
}
