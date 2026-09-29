package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"
	"vehicle-service/config"
	"vehicle-service/internal/constants"
	"vehicle-service/internal/domain/entity"
	"vehicle-service/internal/domain/events"
	"vehicle-service/internal/domain/ports"
	"vehicle-service/pkg/logger"
)

type VehicleService struct {
	config                    config.Config
	repository                ports.VehicleRepository
	vehicleCustomerRepository ports.VehicleCustomerRepository
	materials                 ports.VehicleMaterialRepository
	history                   ports.ServiceHistoryRepository
	outbox                    ports.OutboxRepository
	txManager                 ports.TxManager
	cache                     ports.Cache
}

const (
	defaultHistoryLimit = 50
	maxHistoryLimit     = 200
)

// GetVehicleMaterials implements [ports.VehicleService]. An unknown vehicle
// is a 404 rather than an empty list, so a typo in the id isn't mistaken
// for "nothing installed".
func (v *VehicleService) GetVehicleMaterials(ctx context.Context, vehicleID int64) ([]entity.VehicleMaterial, error) {
	if _, err := v.repository.GetByID(ctx, vehicleID); err != nil {
		return nil, err
	}
	materials, err := v.materials.ListByVehicle(ctx, vehicleID)
	if err != nil {
		logger.ErrorContext(ctx, "failed to list vehicle materials", "vehicle", vehicleID, "error", err)
		return nil, err
	}
	return materials, nil
}

// GetWarranty implements [ports.VehicleService].
func (v *VehicleService) GetWarranty(ctx context.Context, vehicleID int64, asOf time.Time) (entity.Warranty, error) {
	vehicle, err := v.repository.GetByID(ctx, vehicleID)
	if err != nil {
		return entity.Warranty{}, err
	}

	day := calendarDate(asOf)
	warranty := entity.Warranty{VehicleID: vehicleID, EndDate: vehicle.WarrantyEndDate, AsOf: day}
	if end := vehicle.WarrantyEndDate; end != nil && !end.Before(day) {
		warranty.Active = true
		warranty.DaysRemaining = int(end.Sub(day).Hours() / 24)
	}
	return warranty, nil
}

// GetServiceHistory implements [ports.VehicleService].
func (v *VehicleService) GetServiceHistory(ctx context.Context, vehicleID int64, limit int) ([]entity.ServiceHistoryEntry, error) {
	switch {
	case limit <= 0:
		limit = defaultHistoryLimit
	case limit > maxHistoryLimit:
		limit = maxHistoryLimit
	}
	if _, err := v.repository.GetByID(ctx, vehicleID); err != nil {
		return nil, err
	}
	entries, err := v.history.ListByVehicle(ctx, vehicleID, limit)
	if err != nil {
		logger.ErrorContext(ctx, "failed to list service history", "vehicle", vehicleID, "error", err)
		return nil, err
	}
	return entries, nil
}

// RecordServiceCompleted implements [ports.VehicleService].
func (v *VehicleService) RecordServiceCompleted(ctx context.Context, event events.ServiceCompleted) error {
	if event.AppointmentID <= 0 || event.VehicleID <= 0 || event.CompletedAt.IsZero() {
		return fmt.Errorf("%w: ServiceCompleted needs appointment_id, vehicle_id and completed_at", ports.ErrInvalidInput)
	}
	services := event.Services
	if services == nil {
		services = []entity.ServicePerformed{}
	}

	created, err := v.history.Record(ctx, entity.ServiceHistoryEntry{
		VehicleID:     event.VehicleID,
		AppointmentID: event.AppointmentID,
		DealershipID:  event.DealershipID,
		CompletedAt:   event.CompletedAt,
		Services:      services,
	})
	if err != nil {
		return err
	}
	if !created {
		logger.InfoContext(ctx, "service history: duplicate ServiceCompleted ignored", "appointment", event.AppointmentID, "event", event.EventID)
	}
	return nil
}

// AssignInitialOwner implements [ports.VehicleService]. The vehicle row is
// locked first so concurrent assignments queue up behind each other; the
// uq_vehicle_current_owner index is what finally guarantees at most one
// current owner.
func (v *VehicleService) AssignInitialOwner(ctx context.Context, vehicleID, customerID int64, date time.Time) error {
	if customerID <= 0 {
		return fmt.Errorf("%w: customer_id is required", ports.ErrInvalidInput)
	}
	ownedFrom := calendarDate(date)
	if ownedFrom.After(calendarDate(time.Now())) {
		return fmt.Errorf("%w: date can't be in the future", ports.ErrInvalidInput)
	}

	err := v.txManager.RunInTx(ctx, func(ctx context.Context) error {
		vehicle, err := v.repository.GetByIDForUpdate(ctx, vehicleID)
		if err != nil {
			return err
		}
		if vehicle.Status == constants.StatusScrapped {
			return fmt.Errorf("%w: a scrapped vehicle can't be given an owner", ports.ErrInvalidState)
		}
		if err := v.vehicleCustomerRepository.AssignVehicleToCustomer(ctx, vehicleID, customerID, ownedFrom); err != nil {
			if errors.Is(err, ports.ErrConflict) {
				return fmt.Errorf("%w; use POST /transfer to change owners", err)
			}
			return err
		}
		return v.writeEvent(ctx, constants.OwnerAssigned, strconv.FormatInt(vehicleID, 10), events.NewVehicleEvent(events.OwnerAssigned, time.Now(), events.Ownership{
			ID:         vehicleID,
			CustomerID: customerID,
			OwnedFrom:  ownedFrom.Format(time.DateOnly),
		}))
	})
	if err != nil && !isClientError(err) {
		logger.ErrorContext(ctx, "failed to assign initial owner", "vehicle", vehicleID, "customer", customerID, "error", err)
	}
	return err
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
		return v.writeEvent(ctx, constants.VehicleCreated, strconv.FormatInt(created.ID, 10),
			events.NewVehicleEvent(events.VehicleCreated, created.CreatedAt, vehicleState(created)))
	})
	if err != nil {
		if !isClientError(err) {
			logger.ErrorContext(ctx, "failed to register vehicle", "vin", vin, "error", err)
		}
		return entity.Vehicle{}, err
	}
	return created, nil
}

// UpdateVehicle implements [ports.VehicleService]. The row is locked for the
// whole transaction, so the "previous" warranty in the WarrantyChanged event
// is exactly what this update replaced.
func (v *VehicleService) UpdateVehicle(ctx context.Context, vehicleID int64, update entity.VehicleUpdate) (entity.Vehicle, error) {
	if update.Status == nil && update.WarrantyEndDate == nil {
		return entity.Vehicle{}, fmt.Errorf("%w: nothing to update", ports.ErrInvalidInput)
	}
	if update.Status != nil && !update.Status.Valid() {
		return entity.Vehicle{}, fmt.Errorf("%w: unknown status %q", ports.ErrInvalidInput, *update.Status)
	}
	if update.UpdatedAt.IsZero() {
		return entity.Vehicle{}, fmt.Errorf("%w: updated_at is required", ports.ErrInvalidInput)
	}

	var updated entity.Vehicle
	err := v.txManager.RunInTx(ctx, func(ctx context.Context) error {
		current, err := v.repository.GetByIDForUpdate(ctx, vehicleID)
		if err != nil {
			return err
		}
		if !current.UpdatedAt.Equal(update.UpdatedAt) {
			return fmt.Errorf("%w: the vehicle was changed since you read it; reload and try again", ports.ErrConflict)
		}
		if current.Status == constants.StatusScrapped {
			return fmt.Errorf("%w: a scrapped vehicle can no longer be changed", ports.ErrInvalidState)
		}

		next := current
		if update.Status != nil {
			next.Status = *update.Status
		}
		if update.WarrantyEndDate != nil {
			date := calendarDate(*update.WarrantyEndDate)
			next.WarrantyEndDate = &date
		}
		statusChanged := next.Status != current.Status
		warrantyChanged := !sameDate(next.WarrantyEndDate, current.WarrantyEndDate)
		if !statusChanged && !warrantyChanged {
			updated = current
			return nil
		}

		if err := v.repository.Update(ctx, next); err != nil {
			return err
		}
		if updated, err = v.repository.GetByID(ctx, vehicleID); err != nil {
			return err
		}

		key := strconv.FormatInt(vehicleID, 10)
		if err := v.writeEvent(ctx, constants.VehicleUpdate, key,
			events.NewVehicleEvent(events.VehicleUpdated, updated.UpdatedAt, vehicleState(updated))); err != nil {
			return err
		}
		if warrantyChanged {
			return v.writeEvent(ctx, constants.WarrantyChanged, key, events.NewVehicleEvent(events.WarrantyChanged, updated.UpdatedAt, events.WarrantyChange{
				ID:                      vehicleID,
				PreviousWarrantyEndDate: formatDate(current.WarrantyEndDate),
				WarrantyEndDate:         formatDate(updated.WarrantyEndDate),
			}))
		}
		return nil
	})
	if err != nil {
		if !isClientError(err) {
			logger.ErrorContext(ctx, "failed to update vehicle", "vehicle", vehicleID, "error", err)
		}
		return entity.Vehicle{}, err
	}
	return updated, nil
}

// writeEvent JSON-encodes event into an outbox row. Call it inside the
// transaction that made the change the event describes.
func (v *VehicleService) writeEvent(ctx context.Context, topic, key string, event any) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return v.outbox.Create(ctx, entity.OutboxMessage{Topic: topic, Key: key, Payload: payload})
}

// vehicleState is the VehicleCreated / VehicleUpdated payload for v.
func vehicleState(v entity.Vehicle) events.VehicleState {
	return events.VehicleState{
		ID:              v.ID,
		Vin:             v.Vin,
		LicensePlate:    v.LicensePlate,
		VehicleModelID:  v.VehicleModelID,
		WarrantyEndDate: formatDate(v.WarrantyEndDate),
		Status:          string(v.Status),
		CreatedAt:       v.CreatedAt,
		UpdatedAt:       v.UpdatedAt,
	}
}

// isClientError reports whether err is one of the sentinel errors a caller
// caused (and gets told about), as opposed to an internal failure worth
// logging.
func isClientError(err error) bool {
	return errors.Is(err, ports.ErrNotFound) || errors.Is(err, ports.ErrConflict) ||
		errors.Is(err, ports.ErrInvalidInput) || errors.Is(err, ports.ErrInvalidState)
}

// GetCustomerVehicles implements [ports.VehicleService]. An unknown customer
// just has no vehicles: customers live in customer-service, which this
// service doesn't call on the read path.
func (v *VehicleService) GetCustomerVehicles(ctx context.Context, customerID int64) ([]entity.CustomerVehicle, error) {
	ownerships, err := v.vehicleCustomerRepository.ListCurrentByCustomer(ctx, customerID)
	if err != nil {
		logger.ErrorContext(ctx, "failed to list customer vehicles", "customer", customerID, "error", err)
		return nil, err
	}
	return ownerships, nil
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
		previousOwner := transferVehicle.From
		return v.writeEvent(ctx, constants.TransferVehicle, strconv.FormatInt(transferVehicle.VehicleID, 10), events.NewVehicleEvent(events.VehicleTransferred, time.Now(), events.Ownership{
			ID:                 transferVehicle.VehicleID,
			CustomerID:         transferVehicle.To,
			PreviousCustomerID: &previousOwner,
			OwnedFrom:          transferVehicle.Date.Format(time.DateOnly),
		}))
	})
	if err != nil {
		logger.ErrorContext(ctx, "failed to transfer vehicle", "error", err)
		return err
	}
	return err
}

func NewVehicleService(cfg config.Config, repository ports.VehicleRepository, outbox ports.OutboxRepository, vehicleCustomerRepository ports.VehicleCustomerRepository, txManager ports.TxManager, cache ports.Cache, materials ports.VehicleMaterialRepository, history ports.ServiceHistoryRepository) ports.VehicleService {
	return &VehicleService{
		repository:                repository,
		config:                    cfg,
		outbox:                    outbox,
		txManager:                 txManager,
		cache:                     cache,
		vehicleCustomerRepository: vehicleCustomerRepository,
		materials:                 materials,
		history:                   history,
	}
}
