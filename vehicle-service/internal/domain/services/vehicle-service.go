package services

import (
	"context"
	"encoding/json"
	"strconv"
	"vehicle-service/config"
	"vehicle-service/internal/constants"
	"vehicle-service/internal/domain/entity"
	"vehicle-service/internal/domain/ports"
	"vehicle-service/pkg/logger"
)

type VehicleService struct {
	config                    config.Config
	repository                ports.VehicleRepository
	vehicleCustomerRepository ports.VehicleCustomerRepository
	outbox                    ports.OutboxRepository
	txManager                 ports.TxManager
	cache                     ports.Cache
}

// GetVehicleMaterials implements [ports.VehicleService].
func (v *VehicleService) GetVehicleMaterials(ctx context.Context, vehicleID int64) (entity.VehicleMaterial, error) {
	panic("unimplemented")
}

// InitialVehicleOwner implements [ports.VehicleService].
func (v *VehicleService) InitialVehicleOwner(ctx context.Context, vehicleID int64, owner int64) error {
	panic("unimplemented")
}

// RegisterVehicle implements [ports.VehicleService].
func (v *VehicleService) RegisterVehicle(ctx context.Context, vehicle entity.Vehicle) (entity.Vehicle, error) {
	panic("unimplemented")
}

// UpdateVehicleStatus implements [ports.VehicleService].
func (v *VehicleService) UpdateVehicleStatus(ctx context.Context, vehicleID int64, status constants.VehicleStatus) error {
	panic("unimplemented")
}

// GetCustomerVehicle implements [ports.VehicleService].
func (v *VehicleService) GetCustomerVehicle(ctx context.Context, customerID int) ([]entity.Vehicle, error) {
	panic("unimplemented")
}

// GetVehicle implements [ports.VehicleService].
func (v *VehicleService) GetVehicle(ctx context.Context, vehicleID int64) (entity.Vehicle, error) {
	vehicle, err := v.repository.GetByID(ctx, vehicleID)
	if err != nil {
		logger.ErrorContext(ctx, "failed to get vehicle", "vehicle", vehicleID, "error", err)
		return entity.Vehicle{}, err
	}
	return vehicle, nil
}

// TransferVehicle implements [ports.VehicleService].
func (v *VehicleService) TransferVehicle(ctx context.Context, transferVehicle entity.TransferVehicle) error {
	// call customer service to validate customer id
	// update the transfer
	err := v.txManager.RunInTx(ctx, func(ctx context.Context) error {
		err := v.vehicleCustomerRepository.UnassignVehicleFromCustomer(ctx, transferVehicle.VehicleID, transferVehicle.From, transferVehicle.Date)
		if err != nil {
			return err
		}
		err = v.vehicleCustomerRepository.AssignVehicleToCustomer(ctx, transferVehicle.VehicleID, transferVehicle.To, transferVehicle.Date)
		if err != nil {
			return err
		}
		payload, err := json.Marshal(transferVehicle)
		if err != nil {
			return err
		}
		return v.outbox.Create(ctx, entity.OutboxMessage{
			Key:     strconv.FormatInt(int64(transferVehicle.VehicleID), 10),
			Topic:   constants.TransferVehicle,
			Payload: payload,
		})
	})
	if err != nil {
		logger.ErrorContext(ctx, "failed to transfer vehicle", "error", err)
		return err
	}
	return err
}

func NewVehicleService(cfg config.Config, repository ports.VehicleRepository, outbox ports.OutboxRepository, vehicleCustomerRepository ports.VehicleCustomerRepository, txManager ports.TxManager, cache ports.Cache) ports.VehicleService {
	return &VehicleService{
		repository:                repository,
		config:                    cfg,
		outbox:                    outbox,
		txManager:                 txManager,
		cache:                     cache,
		vehicleCustomerRepository: vehicleCustomerRepository,
	}
}
