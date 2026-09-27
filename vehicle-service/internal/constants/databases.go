package constants

const (
	PgUniqueViolationCode     = "23505"
	PgForeignKeyViolationCode = "23503"
)

// Constraint names from postgres/init/03-vehicle-service.sql, used to tell
// which unique rule a write broke.
const (
	ConstraintVehicleVin          = "vehicle_vin_key"
	ConstraintVehicleLicensePlate = "uq_vehicle_license_plate"
	ConstraintVehicleModelFK      = "vehicle_vehicle_model_id_fkey"
	ConstraintCurrentOwner        = "uq_vehicle_current_owner"
)
