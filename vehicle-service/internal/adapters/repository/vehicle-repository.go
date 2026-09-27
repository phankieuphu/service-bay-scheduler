package repository

import (
	"context"
	"errors"
	"fmt"
	"time"
	"vehicle-service/internal/adapters/database/models"
	database_provider "vehicle-service/internal/adapters/database/provider"
	"vehicle-service/internal/constants"
	"vehicle-service/internal/domain/entity"
	"vehicle-service/internal/domain/ports"
	"vehicle-service/pkg/utils"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type VehicleRepository struct {
	db *gorm.DB
}

// Create implements [ports.VehicleRepository].
func (c VehicleRepository) Create(ctx context.Context, vehicle entity.Vehicle) (entity.Vehicle, error) {
	model := c.toModels(vehicle)

	if err := database_provider.DBFromContext(ctx, c.db).Create(&model).Error; err != nil {
		return entity.Vehicle{}, writeError(err)
	}

	return c.toDomain(model), nil
}

// Delete implements [ports.VehicleRepository].
func (c VehicleRepository) Delete(ctx context.Context, id int64) error {
	result := database_provider.DBFromContext(ctx, c.db).Delete(&models.Vehicle{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ports.ErrNotFound
	}

	return nil
}

// GetByID implements [ports.VehicleRepository].
func (c VehicleRepository) GetByID(ctx context.Context, id int64) (entity.Vehicle, error) {
	var model models.Vehicle

	if err := database_provider.DBFromContext(ctx, c.db).First(&model, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return entity.Vehicle{}, ports.ErrNotFound
		}
		return entity.Vehicle{}, err
	}

	return c.toDomain(model), nil
}

// GetByIDForUpdate implements [ports.VehicleRepository].
func (c VehicleRepository) GetByIDForUpdate(ctx context.Context, id int64) (entity.Vehicle, error) {
	var model models.Vehicle

	err := database_provider.DBFromContext(ctx, c.db).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		First(&model, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return entity.Vehicle{}, ports.ErrNotFound
		}
		return entity.Vehicle{}, err
	}

	return c.toDomain(model), nil
}

// List implements [ports.VehicleRepository]. It keyset-paginates by id
// ascending: only rows with id greater than params.Cursor are returned, so a
// page stays stable even if earlier rows are inserted/deleted concurrently.
// It fetches one row past the requested limit to determine HasMore without a
// separate count query.
func (c VehicleRepository) List(ctx context.Context, params ports.ListVehiclesParams) (ports.VehiclePage, error) {
	var rows []models.Vehicle

	query := database_provider.DBFromContext(ctx, c.db).
		Order("id ASC").
		Limit(params.Limit + 1)
	if params.Cursor > 0 {
		query = query.Where("id > ?", params.Cursor)
	}
	if params.Vin != "" {
		query = query.Where("vin = ?", params.Vin)
	}
	if params.LicensePlate != "" {
		query = query.Where("license_plate = ?", params.LicensePlate)
	}
	if params.Status != "" {
		query = query.Where("status = ?", params.Status)
	}

	if err := query.Find(&rows).Error; err != nil {
		return ports.VehiclePage{}, err
	}

	hasMore := len(rows) > params.Limit
	if hasMore {
		rows = rows[:params.Limit]
	}

	vehicles := make([]entity.Vehicle, len(rows))
	for i, row := range rows {
		vehicles[i] = c.toDomain(row)
	}

	var nextCursor int64
	if hasMore {
		nextCursor = vehicles[len(vehicles)-1].ID
	}

	return ports.VehiclePage{
		Vehicles:   vehicles,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}, nil
}

// Update implements [ports.VehicleRepository]. It optimistically locks on
// the row's updated_at, which the caller must have read (e.g. via GetByID)
// before mutating and passing back in vehicle.UpdatedAt. If another request
// updated or deleted the row first, this returns ErrConflict/ErrNotFound
// instead of silently overwriting the other write.
func (c VehicleRepository) Update(ctx context.Context, vehicle entity.Vehicle) error {
	model := c.toModels(vehicle)

	// Name the mutable columns explicitly. Updates(&model) would also SET
	// id (a GENERATED ALWAYS column Postgres refuses to update) and
	// created_at, and would skip fields being set back to NULL.
	result := database_provider.DBFromContext(ctx, c.db).
		Model(&models.Vehicle{}).
		Where("id = ? AND updated_at = ?", model.ID, model.UpdatedAt).
		Updates(map[string]any{
			"license_plate":     model.LicensePlate,
			"vehicle_model_id":  model.VehicleModelID,
			"warranty_end_date": model.WarrantyEndDate,
			"status":            model.Status,
			"updated_at":        time.Now(),
		})
	if result.Error != nil {
		return writeError(result.Error)
	}
	if result.RowsAffected == 0 {
		return c.updateFailureReason(ctx, model.ID)
	}

	return nil
}

// updateFailureReason distinguishes why an optimistic-locked update matched
// no rows: the vehicle no longer exists, vs. it exists but was changed
// (updated_at moved) by another write since the caller last read it.
func (c VehicleRepository) updateFailureReason(ctx context.Context, id int64) error {
	var count int64
	if err := database_provider.DBFromContext(ctx, c.db).
		Model(&models.Vehicle{}).
		Where("id = ?", id).
		Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return ports.ErrNotFound
	}

	return ports.ErrConflict
}

// writeError maps constraint violations on the vehicle table to the
// port's sentinel errors, naming the field so the client knows what to fix.
func writeError(err error) error {
	switch {
	case utils.IsDuplicateKeyError(err):
		switch utils.ConstraintName(err) {
		case constants.ConstraintVehicleVin:
			return fmt.Errorf("%w: a vehicle with this VIN is already registered", ports.ErrConflict)
		case constants.ConstraintVehicleLicensePlate:
			return fmt.Errorf("%w: a vehicle with this license plate is already registered", ports.ErrConflict)
		}
		return ports.ErrConflict
	case utils.IsForeignKeyError(err) && utils.ConstraintName(err) == constants.ConstraintVehicleModelFK:
		return fmt.Errorf("%w: unknown vehicle_model_id", ports.ErrInvalidInput)
	}
	return err
}

func (c VehicleRepository) toModels(vehicle entity.Vehicle) models.Vehicle {
	return toVehicleModel(vehicle)
}

func (c VehicleRepository) toDomain(model models.Vehicle) entity.Vehicle {
	return toVehicleEntity(model)
}

// toVehicleModel and toVehicleEntity are shared with the repositories that
// load a vehicle alongside their own rows (ownership, materials).
func toVehicleModel(vehicle entity.Vehicle) models.Vehicle {
	var plate *string
	if vehicle.LicensePlate != "" {
		plate = &vehicle.LicensePlate
	}
	return models.Vehicle{
		ID:              vehicle.ID,
		Vin:             vehicle.Vin,
		LicensePlate:    plate,
		VehicleModelID:  vehicle.VehicleModelID,
		WarrantyEndDate: vehicle.WarrantyEndDate,
		Status:          vehicle.Status,
		CreatedAt:       vehicle.CreatedAt,
		UpdatedAt:       vehicle.UpdatedAt,
	}
}

func toVehicleEntity(model models.Vehicle) entity.Vehicle {
	var plate string
	if model.LicensePlate != nil {
		plate = *model.LicensePlate
	}
	var warranty *time.Time
	if model.WarrantyEndDate != nil {
		// A date column scans in the connection's zone; pin it to UTC
		// midnight so it matches what the service layer writes.
		date := time.Date(model.WarrantyEndDate.Year(), model.WarrantyEndDate.Month(), model.WarrantyEndDate.Day(), 0, 0, 0, 0, time.UTC)
		warranty = &date
	}
	return entity.Vehicle{
		ID:              model.ID,
		Vin:             model.Vin,
		LicensePlate:    plate,
		VehicleModelID:  model.VehicleModelID,
		WarrantyEndDate: warranty,
		Status:          model.Status,
		CreatedAt:       model.CreatedAt,
		UpdatedAt:       model.UpdatedAt,
	}
}

func NewVehicleRepository(db *gorm.DB) ports.VehicleRepository {
	return VehicleRepository{
		db: db,
	}
}
