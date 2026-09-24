package dto

import (
	"customer-service/internal/constants"
	"time"
)

type CreateCustomerDTO struct {
	Name     string    `json:"name" binding:"required"`
	Email    string    `json:"email" binding:"required,email"`
	Phone    string    `json:"phone"`
	BirthDay time.Time `json:"birth_day" binding:"required"`
}

// return
type CustomerDTO struct {
	ID        int64                    `json:"id"`
	Name      string                   `json:"name"`
	Email     string                   `json:"email"`
	Phone     string                   `json:"phone"`
	BirthDay  time.Time                `json:"birth_day"`
	Status    constants.CustomerStatus `json:"status"`
	CreatedAt time.Time                `json:"created_at"`
	UpdatedAt time.Time                `json:"updated_at"`
}

// ListCustomersResponseDTO is a single cursor-paginated page of customers.
// NextCursor is only set when HasMore is true; pass it back as the `cursor`
// query param to fetch the next page.
type ListCustomersResponseDTO struct {
	Customers  []CustomerDTO `json:"customers"`
	NextCursor int64         `json:"next_cursor,omitempty"`
	HasMore    bool          `json:"has_more"`
}

type UpdateProfileDTO struct {
	Name     string    `json:"name"`
	BirthDay time.Time `json:"birth_day" binding:"-"`
}
