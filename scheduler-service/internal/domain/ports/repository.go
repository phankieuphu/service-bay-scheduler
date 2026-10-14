package ports

import (
	"context"
	"scheduler-service/internal/domain/entity"
	"time"
)

// CatalogRepository reads scheduler's snapshot of dealership data: hubs,
// bays and bookable services.
type CatalogRepository interface {
	// ListHubs returns active hubs, each with its count of bookable bays.
	ListHubs(ctx context.Context) ([]entity.Hub, error)
	// GetHub returns an active hub, or ErrNotFound.
	GetHub(ctx context.Context, id int64) (entity.Hub, error)
	// ListBookableBays returns the hub's bays that take new bookings.
	ListBookableBays(ctx context.Context, hubID int64) ([]entity.Bay, error)
	// ListServices returns active catalog services.
	ListServices(ctx context.Context) ([]entity.CatalogService, error)
}

// AppointmentRepository stores appointments and answers what's already
// booked. The write methods are meant to run inside TxManager.RunInTx.
type AppointmentRepository interface {
	// LockVehicle serializes bookings for one vehicle until the current
	// transaction ends, so two concurrent bookings can't both pass
	// VehicleBusy.
	LockVehicle(ctx context.Context, vehicleID int64) error
	// VehicleBusy says whether the vehicle has a live appointment
	// overlapping [start, end).
	VehicleBusy(ctx context.Context, vehicleID int64, start, end time.Time) (bool, error)
	// FreeBay returns a bookable bay at the hub with nothing reserved in
	// [start, end), or ErrConflict if there is none.
	FreeBay(ctx context.Context, hubID int64, start, end time.Time) (int64, error)
	// Reservations returns the live bay reservations at the hub that
	// overlap [from, to).
	Reservations(ctx context.Context, hubID int64, from, to time.Time) ([]entity.BayReservation, error)
	// Create inserts the appointment with its vehicle, bay and services. A
	// bay another transaction reserved first is ErrConflict (the
	// no_bay_double_booking constraint).
	Create(ctx context.Context, appointment entity.Appointment) (entity.Appointment, error)
	GetByID(ctx context.Context, id int64) (entity.Appointment, error)
	// ListByCustomer returns the customer's appointments, latest start first.
	ListByCustomer(ctx context.Context, customerID int64, limit int) ([]entity.Appointment, error)
	// Cancel marks a BOOKED appointment and its bay reservation cancelled,
	// freeing the bay. ErrConflict if it isn't BOOKED any more.
	Cancel(ctx context.Context, id int64) error
	// Complete marks a BOOKED, CHECKED_IN or IN_PROGRESS appointment
	// COMPLETED, releases its bay and marks its services DONE. ErrConflict if
	// it's in any other state.
	Complete(ctx context.Context, id int64) error
	// ListByHub returns the hub's appointments starting in [from, to),
	// earliest first.
	ListByHub(ctx context.Context, hubID int64, from, to time.Time) ([]entity.Appointment, error)
}
