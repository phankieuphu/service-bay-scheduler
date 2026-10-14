package ports

import "errors"

var (
	// ErrNotFound is returned when a hub, service or appointment doesn't
	// exist, or isn't the caller's to see.
	ErrNotFound = errors.New("not found")

	// ErrConflict is returned when a booking can't take the slot: no bay is
	// free, the vehicle is already booked then, or a concurrent booking won
	// the bay first. Also when an appointment is no longer cancellable.
	ErrConflict = errors.New("conflict")

	// ErrInvalidInput is returned for a well-formed request the service
	// can't act on: an unknown service, a time outside opening hours, a
	// vehicle the caller doesn't own.
	ErrInvalidInput = errors.New("invalid input")

	// ErrNoCustomerProfile is returned when the caller has no customer
	// profile linked to their account (customer-service GET /customer/me
	// returned 404).
	ErrNoCustomerProfile = errors.New("no customer profile is linked to this account")

	// ErrUpstream is returned when customer-service or vehicle-service
	// can't be reached or answers with an unexpected error.
	ErrUpstream = errors.New("upstream service unavailable")
)
