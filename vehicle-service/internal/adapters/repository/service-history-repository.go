package repository

import (
	"context"
	"encoding/json"
	"vehicle-service/internal/adapters/database/models"
	database_provider "vehicle-service/internal/adapters/database/provider"
	"vehicle-service/internal/domain/entity"
	"vehicle-service/internal/domain/ports"
	"vehicle-service/pkg/utils"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ServiceHistoryRepository struct {
	db *gorm.DB
}

func NewServiceHistoryRepository(db *gorm.DB) ports.ServiceHistoryRepository {
	return ServiceHistoryRepository{db: db}
}

// Record implements [ports.ServiceHistoryRepository]. ON CONFLICT DO
// NOTHING on appointment_id makes a redelivered event a no-op instead of a
// duplicate row or an error.
func (r ServiceHistoryRepository) Record(ctx context.Context, entry entity.ServiceHistoryEntry) (bool, error) {
	services, err := json.Marshal(entry.Services)
	if err != nil {
		return false, err
	}
	model := models.ServiceHistory{
		VehicleID:     entry.VehicleID,
		AppointmentID: entry.AppointmentID,
		DealershipID:  entry.DealershipID,
		CompletedAt:   entry.CompletedAt,
		Services:      services,
	}

	result := database_provider.DBFromContext(ctx, r.db).
		Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "appointment_id"}}, DoNothing: true}).
		Create(&model)
	if result.Error != nil {
		if utils.IsForeignKeyError(result.Error) {
			return false, ports.ErrNotFound
		}
		return false, result.Error
	}
	return result.RowsAffected == 1, nil
}

// ListByVehicle implements [ports.ServiceHistoryRepository].
func (r ServiceHistoryRepository) ListByVehicle(ctx context.Context, vehicleID int64, limit int) ([]entity.ServiceHistoryEntry, error) {
	var rows []models.ServiceHistory
	if err := database_provider.DBFromContext(ctx, r.db).
		Where("vehicle_id = ?", vehicleID).
		Order("completed_at DESC, id DESC").
		Limit(limit).
		Find(&rows).Error; err != nil {
		return nil, err
	}

	entries := make([]entity.ServiceHistoryEntry, len(rows))
	for i, row := range rows {
		var services []entity.ServicePerformed
		if err := json.Unmarshal(row.Services, &services); err != nil {
			return nil, err
		}
		entries[i] = entity.ServiceHistoryEntry{
			ID:            row.ID,
			VehicleID:     row.VehicleID,
			AppointmentID: row.AppointmentID,
			DealershipID:  row.DealershipID,
			CompletedAt:   row.CompletedAt,
			Services:      services,
			CreatedAt:     row.CreatedAt,
		}
	}
	return entries, nil
}
