package services

import (
	"context"
	"vehicle-service/config"
	"vehicle-service/internal/domain/entity"
	"vehicle-service/internal/domain/ports"
)

type VehicleService struct {
	config     config.Config
	repository ports.VehicleRepository
	outbox     ports.OutboxRepository
	txManager  ports.TxManager
	cache      ports.Cache
}

// GetCustomerVehicle implements [ports.VehicleService].
func (v *VehicleService) GetCustomerVehicle(ctx context.Context, customerID int) ([]entity.Vehicle, error) {
	panic("unimplemented")
}

// GetVehicle implements [ports.VehicleService].
func (v *VehicleService) GetVehicle(ctx context.Context, vehicleID int) (entity.Vehicle, error) {
	panic("unimplemented")
}

// TransferVehicle implements [ports.VehicleService].
func (v *VehicleService) TransferVehicle(ctx context.Context, from int, to int, vehicle int) error {
	panic("unimplemented")
}

func NewVehicleService(cfg config.Config, repository ports.VehicleRepository, outbox ports.OutboxRepository, txManager ports.TxManager, cache ports.Cache) ports.VehicleService {
	return &VehicleService{
		repository: repository,
		config:     cfg,
		outbox:     outbox,
		txManager:  txManager,
		cache:      cache,
	}
}
