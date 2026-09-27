package entity

import (
	"time"
)

type VehicleMaterial struct {
	ID          int64
	VehicleID   int64
	MaterialID  int64 // owned by dealership-service
	Description string
	Count       int
	InstalledAt time.Time
}
