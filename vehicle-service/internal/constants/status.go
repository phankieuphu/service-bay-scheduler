package constants

type VehicleStatus string

const (
	StatusActive   VehicleStatus = "ACTIVE"
	StatusSold     VehicleStatus = "SOLD"
	StatusScrapped VehicleStatus = "SCRAPPED"
)

// OwnershipStatus is the status of a customer_vehicle row (who currently
// owns/owned a vehicle), distinct from VehicleStatus (the vehicle itself).
type OwnershipStatus string

const (
	OwnershipCurrent     OwnershipStatus = "CURRENT"
	OwnershipTransferred OwnershipStatus = "TRANSFERRED"
)
