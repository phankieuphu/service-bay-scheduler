package entity

// Customer is the caller's profile as customer-service reports it.
type Customer struct {
	ID    int64
	Name  string
	Email string
}

// Vehicle is a vehicle vehicle-service says a customer currently owns.
type Vehicle struct {
	ID           int64
	VIN          string
	LicensePlate string
}
