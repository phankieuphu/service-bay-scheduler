package ports

import "errors"

var (
	// ErrNotFound is returned when a customer does not exist (or no longer
	// does, e.g. deleted concurrently by another request).
	ErrNotFound = errors.New("customer: not found")

	// ErrConflict is returned when a write loses a race with another
	// concurrent write: either a unique constraint (email) was violated, or
	// an update was based on stale data (someone else updated the row first).
	ErrConflict = errors.New("customer: conflict")

	// ErrInvalidInput is returned when a request is well-formed but carries
	// nothing the service can act on (e.g. an update with no fields set).
	ErrInvalidInput = errors.New("customer: invalid input")
)
