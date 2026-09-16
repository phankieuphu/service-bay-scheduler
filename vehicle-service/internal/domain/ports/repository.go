package ports

import (
	"context"
	"time"
	"vehicle-service/internal/domain/entity"
)

type VehicleRepository interface {
	Create(ctx context.Context, vehicle entity.Vehicle) (entity.Vehicle, error)
	GetByID(ctx context.Context, id int64) (entity.Vehicle, error)
	List(ctx context.Context, params ListVehiclesParams) (VehiclePage, error)
	Update(ctx context.Context, vehicle entity.Vehicle) error
	Delete(ctx context.Context, id int64) error
}

type VehicleCustomerRepository interface {
	GetCustomerVehicle(ctx context.Context, customerID int) (entity.CustomerVehicle, error)
	TransferVehicleToCustomer(ctx context.Context, customerID int) error
	AssignVehicleToCustomer(ctx context.Context, vehicleID, customerID int, date time.Time) error
	UnassignVehicleFromCustomer(ctx context.Context, vehicleID, customerID int, date time.Time) error
}

type VehicleMaterialRepository interface {
	GetVehicleMaterials(ctx context.Context, vehicle_id int) ([]entity.VehicleMaterial, error)
}
