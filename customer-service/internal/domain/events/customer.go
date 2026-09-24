package events

import "customer-service/pkg/times"

// DeleteCustomerEvent is the payload published on constants.CustomerDelete.
type DeleteCustomerEvent struct {
	ID        int64      `json:"id"`
	DeletedAt times.Time `json:"deleted_at"`
}

type UpdateCustomerEvents struct {
	ID int64 `json:"id"`
}
