package constants

const (
	VehicleCreated = "vehicle.vehicle.created.v1"
	VehicleUpdate  = "vehicle.vehicle.updated.v1"
	// WarrantyChanged is consumed by billing-service to price the next bill
	// correctly (architecture-design.md §3).
	WarrantyChanged = "vehicle.vehicle.warranty-changed.v1"
	TransferVehicle = "vehicle.vehicle-customer.transfer.v1"
)
