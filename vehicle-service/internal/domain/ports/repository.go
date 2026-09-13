package ports

import (
	"context"
	"vehicle-service/internal/domain/entity"
)

type VehicleRepository interface {
	Create(ctx context.Context, vehicle entity.Vehicle) (entity.Vehicle, error)
	GetByID(ctx context.Context, id int64) (entity.Vehicle, error)
	List(ctx context.Context, params ListVehiclesParams) (VehiclePage, error)
	Update(ctx context.Context, vehicle entity.Vehicle) error
	Delete(ctx context.Context, id int64) error
}

type CustomerVehicle interface {
	GetCustomerVehicle(ctx context.Context, customerID int) (entity.CustomerVehicle, error)
	TransferVehicle(ctx context.Context, customerID int) error
	AssignVehicleToCustomer(ctx context.Context, vehicleID, customerID int) error
}

type VehicleMaterial interface {
	GetVehicleMaterials(ctx context.Context, vehicle_model_id int) ([]entity.VehicleMaterial, error)
}
