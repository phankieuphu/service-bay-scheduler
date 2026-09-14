package ports

import (
	"context"
	"vehicle-service/internal/domain/entity"
)

type VehicleService interface {
	GetCustomerVehicle(ctx context.Context, customerID int) ([]entity.Vehicle, error)
	GetVehicle(ctx context.Context, vehicleID int64) (entity.Vehicle, error)
	TransferVehicle(ctx context.Context, transferVehicle entity.TransferVehicle) error
}
