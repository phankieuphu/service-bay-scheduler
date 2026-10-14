package constants

// Topics scheduler-service publishes through the outbox, one per event type
// (architecture-design.md §3d).
const (
	TopicAppointmentBooked    = "scheduler.appointment.booked.v1"
	TopicAppointmentCancelled = "scheduler.appointment.cancelled.v1"
)
