package ports

import (
	"context"
	"vehicle-service/internal/constants"
	"vehicle-service/internal/domain/entity"
)

type VehicleService interface {
	GetCustomerVehicle(ctx context.Context, customerID int) ([]entity.Vehicle, error)
	GetVehicle(ctx context.Context, vehicleID int64) (entity.Vehicle, error)
	TransferVehicle(ctx context.Context, transferVehicle entity.TransferVehicle) error
	RegisterVehicle(ctx context.Context, vehicle entity.Vehicle) (entity.Vehicle, error) // register new vehicle
	UpdateVehicleStatus(ctx context.Context, vehicleID int64, status constants.VehicleStatus) error
	InitialVehicleOwner(ctx context.Context, vehicleID int64, owner int64) error // assign just work with vehicle was has owner
	// GetWarranty(ctx context.Context, vehicleID int64)
	// VehicleHistory(ctx context.Context, vehicleID int64) // vehicle history
	GetVehicleMaterials(ctx context.Context, vehicleID int64) (entity.VehicleMaterial, error)
}
