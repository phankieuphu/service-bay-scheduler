package entity

import (
	"time"
	"vehicle-service/internal/constants"
)

type CustomerVehicle struct {
	ID        int64
	Name      string
	Email     string
	Phone     string
	BirthDay  time.Time
	Status    constants.VehicleStatus
	CreatedAt time.Time
	UpdatedAt time.Time
}
