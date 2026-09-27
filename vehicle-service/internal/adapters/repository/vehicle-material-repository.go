package repository

import (
	"context"
	"vehicle-service/internal/adapters/database/models"
	database_provider "vehicle-service/internal/adapters/database/provider"
	"vehicle-service/internal/domain/entity"
	"vehicle-service/internal/domain/ports"

	"gorm.io/gorm"
)

type VehicleMaterialRepository struct {
	db *gorm.DB
}

// ListByVehicle implements [ports.VehicleMaterialRepository].
func (c VehicleMaterialRepository) ListByVehicle(ctx context.Context, vehicleID int64) ([]entity.VehicleMaterial, error) {
	var rows []models.VehicleMaterial
	if err := database_provider.DBFromContext(ctx, c.db).
		Where("vehicle_id = ?", vehicleID).
		Order("installed_at DESC, id DESC").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	materials := make([]entity.VehicleMaterial, len(rows))
	for i, row := range rows {
		materials[i] = c.toDomain(row)
	}
	return materials, nil
}

func (c VehicleMaterialRepository) toDomain(model models.VehicleMaterial) entity.VehicleMaterial {
	var description string
	if model.Description != nil {
		description = *model.Description
	}
	return entity.VehicleMaterial{
		ID:          model.ID,
		VehicleID:   model.VehicleID,
		MaterialID:  model.MaterialID,
		Description: description,
		Count:       model.Count,
		InstalledAt: model.InstalledAt,
	}
}

func NewVehicleMaterialRepository(db *gorm.DB) ports.VehicleMaterialRepository {
	return VehicleMaterialRepository{
		db: db,
	}
}
