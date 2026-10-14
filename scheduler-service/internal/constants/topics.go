package constants

// Topics scheduler-service publishes through the outbox, one per event type
// (architecture-design.md §3d).
const (
	TopicAppointmentBooked    = "scheduler.appointment.booked.v1"
	TopicAppointmentCancelled = "scheduler.appointment.cancelled.v1"
	// TopicServiceCompleted carries ServiceCompleted (architecture-design.md
	// §3b), which vehicle-service and billing-service consume.
	TopicServiceCompleted = "scheduler.appointment.service-completed.v1"
)
