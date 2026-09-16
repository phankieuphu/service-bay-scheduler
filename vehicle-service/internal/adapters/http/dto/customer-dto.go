package dto

import (
	"time"
	"vehicle-service/internal/constants"
)

type CreateVehicleDTO struct {
	Name     string    `json:"name" binding:"required"`
	Email    string    `json:"email" binding:"required,email"`
	Phone    string    `json:"phone"`
	BirthDay time.Time `json:"birth_day" binding:"required"`
}

// return
type VehicleDTO struct {
	ID        int64                   `json:"id"`
	Name      string                  `json:"name"`
	Email     string                  `json:"email"`
	Phone     string                  `json:"phone"`
	BirthDay  time.Time               `json:"birth_day"`
	Status    constants.VehicleStatus `json:"status"`
	CreatedAt time.Time               `json:"created_at"`
	UpdatedAt time.Time               `json:"updated_at"`
}

// ListVehiclesResponseDTO is a single cursor-paginated page of vehicles.
// NextCursor is only set when HasMore is true; pass it back as the `cursor`
// query param to fetch the next page.
type ListVehiclesResponseDTO struct {
	Vehicles   []VehicleDTO `json:"vehicles"`
	NextCursor int64        `json:"next_cursor,omitempty"`
	HasMore    bool         `json:"has_more"`
}
