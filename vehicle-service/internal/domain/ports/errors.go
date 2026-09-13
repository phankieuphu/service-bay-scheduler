package ports

import "errors"

var (
	// ErrNotFound is returned when a vehicle does not exist (or no longer
	// does, e.g. deleted concurrently by another request).
	ErrNotFound = errors.New("vehicle: not found")

	// ErrConflict is returned when a write loses a race with another
	// concurrent write: either a unique constraint (email) was violated, or
	// an update was based on stale data (someone else updated the row first).
	ErrConflict = errors.New("vehicle: conflict")
)
