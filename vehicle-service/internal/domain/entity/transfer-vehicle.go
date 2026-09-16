package entity

import (
	"time"
)

type TransferVehicle struct {
	Date      time.Time
	From      int64 // from customer
	To        int64 // to customer
	VehicleID int64 // which vehicle
}
