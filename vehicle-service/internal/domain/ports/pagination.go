package ports

import (
	"vehicle-service/internal/constants"
	"vehicle-service/internal/domain/entity"
)

// ListVehiclesParams is a cursor-based pagination request. Cursor is the ID
// of the last vehicle the caller has already seen (0 to start from the
// beginning); results are ordered by ID ascending and only IDs greater than
// Cursor are returned.
//
// Vin, LicensePlate and Status are optional exact-match filters; Vin and
// LicensePlate are matched after the same normalization used on write.
type ListVehiclesParams struct {
	Cursor       int64
	Limit        int
	Vin          string
	LicensePlate string
	Status       constants.VehicleStatus
}

// VehiclePage is a single page of a cursor-paginated vehicle listing.
// NextCursor is only meaningful when HasMore is true.
type VehiclePage struct {
	Vehicles   []entity.Vehicle
	NextCursor int64
	HasMore    bool
}
