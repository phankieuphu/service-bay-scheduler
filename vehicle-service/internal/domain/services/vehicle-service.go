package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"vehicle-service/config"
	"vehicle-service/internal/constants"
	"vehicle-service/internal/domain/entity"
	"vehicle-service/internal/domain/events"
	"vehicle-service/internal/domain/ports"
	"vehicle-service/pkg/logger"

	"github.com/google/uuid"
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

// RegisterVehicle implements [ports.VehicleService]. The vehicle row and its
// VehicleCreated outbox row commit together.
func (v *VehicleService) RegisterVehicle(ctx context.Context, vehicle entity.Vehicle) (entity.Vehicle, error) {
	vin, err := normalizeVin(vehicle.Vin)
	if err != nil {
		return entity.Vehicle{}, err
	}
	plate, err := normalizePlate(vehicle.LicensePlate)
	if err != nil {
		return entity.Vehicle{}, err
	}
	if vehicle.VehicleModelID <= 0 {
		return entity.Vehicle{}, fmt.Errorf("%w: vehicle_model_id is required", ports.ErrInvalidInput)
	}
	if vehicle.Status == "" {
		vehicle.Status = constants.StatusActive
	}
	if !vehicle.Status.Valid() {
		return entity.Vehicle{}, fmt.Errorf("%w: unknown status %q", ports.ErrInvalidInput, vehicle.Status)
	}
	if vehicle.WarrantyEndDate != nil {
		date := calendarDate(*vehicle.WarrantyEndDate)
		vehicle.WarrantyEndDate = &date
	}
	vehicle.Vin, vehicle.LicensePlate = vin, plate

	var created entity.Vehicle
	err = v.txManager.RunInTx(ctx, func(ctx context.Context) error {
		created, err = v.repository.Create(ctx, vehicle)
		if err != nil {
			return err
		}
		payload, err := json.Marshal(events.VehicleCreated{
			EventID:         uuid.NewString(),
			OccurredAt:      created.CreatedAt,
			VehicleID:       created.ID,
			Vin:             created.Vin,
			LicensePlate:    created.LicensePlate,
			VehicleModelID:  created.VehicleModelID,
			WarrantyEndDate: formatDate(created.WarrantyEndDate),
			Status:          string(created.Status),
		})
		if err != nil {
			return err
		}
		return v.outbox.Create(ctx, entity.OutboxMessage{
			Key:     strconv.FormatInt(created.ID, 10),
			Topic:   constants.VehicleCreated,
			Payload: payload,
		})
	})
	if err != nil {
		if !errors.Is(err, ports.ErrConflict) && !errors.Is(err, ports.ErrInvalidInput) {
			logger.ErrorContext(ctx, "failed to register vehicle", "vin", vin, "error", err)
		}
		return entity.Vehicle{}, err
	}
	return created, nil
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

const (
	defaultListLimit = 20
	maxListLimit     = 100
)

// ListVehicles implements [ports.VehicleService].
func (v *VehicleService) ListVehicles(ctx context.Context, params ports.ListVehiclesParams) (ports.VehiclePage, error) {
	if params.Cursor < 0 {
		params.Cursor = 0
	}
	switch {
	case params.Limit <= 0:
		params.Limit = defaultListLimit
	case params.Limit > maxListLimit:
		params.Limit = maxListLimit
	}

	var err error
	if params.Vin != "" {
		if params.Vin, err = normalizeVin(params.Vin); err != nil {
			return ports.VehiclePage{}, err
		}
	}
	if params.LicensePlate != "" {
		if params.LicensePlate, err = normalizePlate(params.LicensePlate); err != nil {
			return ports.VehiclePage{}, err
		}
	}
	if params.Status != "" && !params.Status.Valid() {
		return ports.VehiclePage{}, fmt.Errorf("%w: unknown status %q", ports.ErrInvalidInput, params.Status)
	}

	page, err := v.repository.List(ctx, params)
	if err != nil {
		logger.ErrorContext(ctx, "failed to list vehicles", "cursor", params.Cursor, "limit", params.Limit, "error", err)
		return ports.VehiclePage{}, err
	}
	return page, nil
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
