package entity

import (
	"time"
)

type VehicleMaterial struct {
	ID          int64
	VehicleID   int64
	MaterialID  int64
	Description string
	Count       int
	InstalledAt time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
