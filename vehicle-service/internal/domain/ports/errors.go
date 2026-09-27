package ports

import "errors"

var (
	// ErrNotFound is returned when a vehicle does not exist (or no longer
	// does, e.g. deleted concurrently by another request).
	ErrNotFound = errors.New("vehicle: not found")

	// ErrConflict is returned when a write loses a race with another
	// concurrent write: either a unique constraint (VIN, plate, current
	// owner) was violated, or an update was based on stale data (someone
	// else updated the row first).
	ErrConflict = errors.New("vehicle: conflict")

	// ErrInvalidInput is returned when a request is well-formed JSON but its
	// values are wrong (malformed VIN, unknown status, nothing to update).
	ErrInvalidInput = errors.New("vehicle: invalid input")

	// ErrInvalidState is returned when the vehicle's current state forbids
	// the operation, e.g. changing a SCRAPPED vehicle.
	ErrInvalidState = errors.New("vehicle: invalid state")
)
