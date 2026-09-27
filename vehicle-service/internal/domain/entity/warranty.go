package entity

import "time"

// Warranty is a vehicle's warranty status on a given day.
type Warranty struct {
	VehicleID int64
	EndDate   *time.Time // nil: no warranty on record
	AsOf      time.Time  // the day the status was evaluated for
	// Active is true when EndDate is on or after AsOf: the warranty covers
	// its whole last day.
	Active        bool
	DaysRemaining int // days after AsOf still covered; 0 when inactive
}
