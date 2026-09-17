package entity

import (
	"time"
	"vehicle-service/internal/constants"
)

type CustomerVehicle struct {
	ID         int64
	Vehicle    Vehicle
	CustomerID int64
	OwnedFrom  time.Time
	OwnedTo    *time.Time
	Status     constants.OwnershipStatus
	CreatedAt  time.Time
}
