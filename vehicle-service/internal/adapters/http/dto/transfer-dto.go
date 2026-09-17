package dto

import "time"

type TransferVehicleDTO struct {
	VehicleID int64      `json:"vehicle_id" binding:"required"`
	From      int64      `json:"from" binding:"required"`
	To        int64      `json:"to" binding:"required"`
	Date      *time.Time `json:"date"`
}
