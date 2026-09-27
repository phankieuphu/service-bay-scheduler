package events

import "time"

// Payloads published through the outbox, one topic per event type (see
// constants/topics.go). Every event carries an EventID so at-least-once
// delivery can be deduplicated by consumers. Calendar dates (warranty end,
// ownership start) are sent as "2006-01-02" strings, not timestamps.

// VehicleCreated is published on constants.VehicleCreated.
type VehicleCreated struct {
	EventID         string    `json:"event_id"`
	OccurredAt      time.Time `json:"occurred_at"`
	VehicleID       int64     `json:"vehicle_id"`
	Vin             string    `json:"vin"`
	LicensePlate    string    `json:"license_plate,omitempty"`
	VehicleModelID  int64     `json:"vehicle_model_id"`
	WarrantyEndDate *string   `json:"warranty_end_date"`
	Status          string    `json:"status"`
}
