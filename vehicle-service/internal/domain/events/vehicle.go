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

// VehicleUpdated is published on constants.VehicleUpdate with the
// vehicle's state after the change.
type VehicleUpdated struct {
	EventID         string    `json:"event_id"`
	OccurredAt      time.Time `json:"occurred_at"`
	VehicleID       int64     `json:"vehicle_id"`
	Status          string    `json:"status"`
	WarrantyEndDate *string   `json:"warranty_end_date"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// WarrantyChanged is published on constants.WarrantyChanged, in addition to
// VehicleUpdated, when warranty_end_date changes.
type WarrantyChanged struct {
	EventID                 string    `json:"event_id"`
	OccurredAt              time.Time `json:"occurred_at"`
	VehicleID               int64     `json:"vehicle_id"`
	PreviousWarrantyEndDate *string   `json:"previous_warranty_end_date"`
	WarrantyEndDate         *string   `json:"warranty_end_date"`
}

// OwnerAssigned is published on constants.OwnerAssigned when a vehicle gets
// its first owner. Later ownership changes are published on
// constants.TransferVehicle.
type OwnerAssigned struct {
	EventID    string    `json:"event_id"`
	OccurredAt time.Time `json:"occurred_at"`
	VehicleID  int64     `json:"vehicle_id"`
	CustomerID int64     `json:"customer_id"`
	OwnedFrom  string    `json:"owned_from"`
}
