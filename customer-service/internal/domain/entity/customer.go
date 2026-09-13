package entity

import (
	"customer-service/internal/constants"
	"time"
)

type Customer struct {
	ID        int64
	Name      string
	Email     string
	Phone     string
	BirthDay  time.Time
	Status    constants.CustomerStatus
	CreatedAt time.Time
	UpdatedAt time.Time
}
