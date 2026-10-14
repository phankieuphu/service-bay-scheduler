package entity

import (
	"time"

	"github.com/shopspring/decimal"
)

// Hub is scheduler's snapshot of a dealership-service hub: where and when
// customers can book. OpenHour and CloseHour are offsets from local
// midnight in TimeZone.
type Hub struct {
	ID        int64
	Name      string
	Address   string
	OpenHour  time.Duration
	CloseHour time.Duration
	TimeZone  string
	Active    bool
	BayCount  int
}

// Bay is one service bay at a hub.
type Bay struct {
	ID     int64
	HubID  int64
	Code   string
	Status string
}

// CatalogService is one bookable service: what it costs and how long it
// keeps a bay busy.
type CatalogService struct {
	ID              int64
	Name            string
	ServiceType     string
	Fee             decimal.Decimal
	DurationMinutes int
	Active          bool
}

// BayReservation is a window a bay is already taken for.
type BayReservation struct {
	BayID int64
	Start time.Time
	End   time.Time
}
