package repository

import (
	"context"
	"errors"
	"vehicle-service/internal/adapters/database/models"
	database_provider "vehicle-service/internal/adapters/database/provider"
	"vehicle-service/internal/domain/entity"
	"vehicle-service/internal/domain/ports"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

// pgUniqueViolationCode is PostgreSQL's SQLSTATE code for a unique-key
// violation (e.g. the email unique constraint).
const pgUniqueViolationCode = "23505"

type VehicleRepository struct {
	db *gorm.DB
}

// Create implements [ports.VehicleRepository].
func (c VehicleRepository) Create(ctx context.Context, vehicle entity.Vehicle) (entity.Vehicle, error) {
	model := c.toModels(vehicle)

	if err := database_provider.DBFromContext(ctx, c.db).Create(&model).Error; err != nil {
		if isDuplicateKeyError(err) {
			return entity.Vehicle{}, ports.ErrConflict
		}
		return entity.Vehicle{}, err
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

	result := database_provider.DBFromContext(ctx, c.db).
		Model(&models.Vehicle{}).
		Where("id = ? AND updated_at = ?", model.ID, model.UpdatedAt).
		Updates(&model)
	if result.Error != nil {
		if isDuplicateKeyError(result.Error) {
			return ports.ErrConflict
		}
		return result.Error
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

func isDuplicateKeyError(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolationCode
}

func (c VehicleRepository) toModels(vehicle entity.Vehicle) models.Vehicle {
	return models.Vehicle{
		ID:        vehicle.ID,
		Status:    vehicle.Status,
		CreatedAt: vehicle.CreatedAt,
		UpdatedAt: vehicle.UpdatedAt,
	}
}

func (c VehicleRepository) toDomain(model models.Vehicle) entity.Vehicle {
	return entity.Vehicle{
		ID:        model.ID,
		Status:    model.Status,
		CreatedAt: model.CreatedAt,
		UpdatedAt: model.UpdatedAt,
	}
}

func NewVehicleRepository(db *gorm.DB) ports.VehicleRepository {
	return VehicleRepository{
		db: db,
	}
}
