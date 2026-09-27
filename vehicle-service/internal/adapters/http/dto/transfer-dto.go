package dto

import "time"

type TransferVehicleDTO struct {
	VehicleID int64      `json:"vehicle_id" binding:"required"`
	From      int64      `json:"from" binding:"required"`
	To        int64      `json:"to" binding:"required"`
	Date      *time.Time `json:"date"`
}

// AssignOwnerDTO is the body of POST /vehicle/:id/owner. Date is the day
// ownership starts; it defaults to today and can't be in the future.
type AssignOwnerDTO struct {
	CustomerID int64      `json:"customer_id" binding:"required,gt=0"`
	Date       *time.Time `json:"date"`
}
