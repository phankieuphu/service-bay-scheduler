package repository

import (
	"context"
	"vehicle-service/internal/adapters/database/models"
	"vehicle-service/internal/domain/entity"
	"vehicle-service/internal/domain/ports"
	"vehicle-service/pkg/logger"

	"gorm.io/gorm"
)

type VehicleMaterialRepository struct {
	db *gorm.DB
}

// GetVehicleMaterials implements [ports.VehicleMaterialRepository].
func (c VehicleMaterialRepository) GetVehicleMaterials(ctx context.Context, vehicle_id int) ([]entity.VehicleMaterial, error) {
	var rows []models.VehicleMaterial
	if err := c.db.Where("vehicle_id = ?", vehicle_id).Find(&rows).Error; err != nil {
		logger.ErrorContext(ctx, "failed to list vehicle material", "vehicle_id", vehicle_id, "error", err)
		return []entity.VehicleMaterial{}, err
	}
	vehicle_materials := make([]entity.VehicleMaterial, len(rows))
	for i, row := range rows {
		vehicle_materials[i] = c.toDomain(row)
	}
	return vehicle_materials, nil
}

func (c VehicleMaterialRepository) toDomain(model models.VehicleMaterial) entity.VehicleMaterial {
	return entity.VehicleMaterial{
		ID:          model.ID,
		VehicleID:   model.VehicleID,
		MaterialID:  model.MaterialID,
		Description: model.Description,
		Count:       model.Count,
		InstalledAt: model.InstalledAt,
		CreatedAt:   model.CreatedAt,
		UpdatedAt:   model.UpdatedAt,
	}
}

func NewVehicleMaterialRepository(db *gorm.DB) ports.VehicleMaterialRepository {
	return VehicleMaterialRepository{
		db: db,
	}
}
