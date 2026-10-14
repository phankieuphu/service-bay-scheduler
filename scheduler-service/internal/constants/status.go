package constants

// AppointmentStatus is appointment.status; the CHECK constraint in
// migrations/00001_baseline.sql lists the same values.
type AppointmentStatus string

const (
	AppointmentBooked     AppointmentStatus = "BOOKED"
	AppointmentCheckedIn  AppointmentStatus = "CHECKED_IN"
	AppointmentInProgress AppointmentStatus = "IN_PROGRESS"
	AppointmentCompleted  AppointmentStatus = "COMPLETED"
	AppointmentCancelled  AppointmentStatus = "CANCELLED"
	AppointmentNoShow     AppointmentStatus = "NO_SHOW"
)

// BayAvailable is the only hub_bay.status that takes new bookings.
const BayAvailable = "AVAILABLE"
