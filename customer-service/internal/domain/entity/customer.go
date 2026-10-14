package entity

import (
	"customer-service/internal/constants"
	"time"
)

type Customer struct {
	ID int64
	// UserID is identity-service's user id when this customer has signed
	// up; empty for a profile staff created that has no login.
	UserID    string
	Name      string
	Email     string
	Phone     string
	BirthDay  time.Time
	Status    constants.CustomerStatus
	CreatedAt time.Time
	UpdatedAt time.Time
}
