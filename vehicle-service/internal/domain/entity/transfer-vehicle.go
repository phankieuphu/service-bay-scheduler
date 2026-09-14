package entity

import (
	"time"
)

type TransferVehicle struct {
	Date      time.Time
	From      int // from customer
	To        int // to customer
	VehicleID int // which vehicle
}
