package events

import "customer-service/pkg/times"

// DeleteCustomerEvent is the payload published on constants.CustomerDelete.
type DeleteCustomerEvent struct {
	ID        int64      `json:"id"`
	DeletedAt times.Time `json:"deleted_at"`
}

// UpdateCustomerEvents is the payload published on constants.CustomerUpdate.
// Only changed fields are set; BirthDay is a date, not an instant, so it is
// sent as "2006-01-02" rather than a timestamp.
type UpdateCustomerEvents struct {
	ID        int64      `json:"id"`
	Name      string     `json:"name,omitempty"`
	BirthDay  string     `json:"birth_day,omitempty"`
	UpdatedAt times.Time `json:"updated_at"`
}
