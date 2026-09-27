package ports

import (
	"context"
	"time"
	"vehicle-service/internal/domain/entity"
)

type VehicleService interface {
	GetCustomerVehicles(ctx context.Context, customerID int64) ([]entity.CustomerVehicle, error)
	GetVehicle(ctx context.Context, vehicleID int64) (entity.Vehicle, error)
	ListVehicles(ctx context.Context, params ListVehiclesParams) (VehiclePage, error)
	TransferVehicle(ctx context.Context, transferVehicle entity.TransferVehicle) error
	RegisterVehicle(ctx context.Context, vehicle entity.Vehicle) (entity.Vehicle, error) // register new vehicle
	UpdateVehicle(ctx context.Context, vehicleID int64, update entity.VehicleUpdate) (entity.Vehicle, error)
	// AssignInitialOwner gives a vehicle with no current owner its first
	// one; after that, ownership changes go through TransferVehicle.
	AssignInitialOwner(ctx context.Context, vehicleID, customerID int64, date time.Time) error
	// GetWarranty(ctx context.Context, vehicleID int64)
	// VehicleHistory(ctx context.Context, vehicleID int64) // vehicle history
	GetVehicleMaterials(ctx context.Context, vehicleID int64) (entity.VehicleMaterial, error)
}
