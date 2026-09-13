package ports

import (
	"context"
	"vehicle-service/internal/domain/entity"
)

type VehicleService interface {
	GetCustomerVehicle(ctx context.Context, customerID int) ([]entity.Vehicle, error)
	GetVehicle(ctx context.Context, vehicleID int) (entity.Vehicle, error)
	TransferVehicle(ctx context.Context, from, to, vehicle int) error
}
