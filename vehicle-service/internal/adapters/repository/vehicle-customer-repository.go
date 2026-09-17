package repository

import (
	"context"
	"errors"
	"time"
	"vehicle-service/internal/adapters/database/models"
	database_provider "vehicle-service/internal/adapters/database/provider"
	"vehicle-service/internal/constants"
	"vehicle-service/internal/domain/entity"
	"vehicle-service/internal/domain/ports"
	"vehicle-service/pkg/logger"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type VehicleCustomerRepository struct {
	db *gorm.DB
}

// AssignVehicleToCustomer implements [ports.VehicleCustomerRepository].
func (c VehicleCustomerRepository) AssignVehicleToCustomer(ctx context.Context, vehicleID int64, customerID int64, date time.Time) error {
	model := models.CustomerVehicle{
		CustomerID: int64(customerID),
		VehicleID:  vehicleID,
		// Vehicle:    models.Vehicle{},
		OwnedFrom: date,
		// OwnedTo:   time.Time{},
		Status:    constants.OwnershipCurrent,
		CreatedAt: date,
	}

	err := database_provider.DBFromContext(ctx, c.db).Create(&model).Error
	if err != nil {
		logger.ErrorContext(ctx, "failed to assign vehicle to customer", "customerID", customerID, "vehicleID", vehicleID)
		return err
	}
	return nil
}

// GetCustomerVehicle implements [ports.VehicleCustomerRepository].
func (c VehicleCustomerRepository) GetCustomerVehicle(ctx context.Context, customerID int64) (entity.CustomerVehicle, error) {
	var model models.CustomerVehicle
	err := database_provider.DBFromContext(ctx, c.db).First(&model).Where("customer =? ", customerID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return entity.CustomerVehicle{}, ports.ErrNotFound
		}
		return entity.CustomerVehicle{}, err
	}
	return c.toDomain(model), nil
}

// TransferVehicleToCustomer implements [ports.VehicleCustomerRepository].
// func (c VehicleCustomerRepository) TransferVehicleToCustomer(ctx context.Context, customerID int64) error {
// 	panic("unimplemented")
// }

// UnassignVehicleFromCustomer implements [ports.VehicleCustomerRepository].
func (c VehicleCustomerRepository) UnassignVehicleFromCustomer(ctx context.Context, vehicleID int64, customerID int64, date time.Time) error {
	db := database_provider.DBFromContext(ctx, c.db)
	var model models.CustomerVehicle
	err := db.Clauses(clause.Locking{Strength: "UPDATE"}).Where("vehicle_id = ? AND status = ?", vehicleID, constants.OwnershipCurrent).First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ports.ErrNotFound
		}
		return err
	}
	if model.CustomerID != customerID {
		return ports.ErrConflict
	}
	return db.Model(&models.CustomerVehicle{}).
		Where("id = ?", model.ID).
		Updates(map[string]any{
			"owned_to": date,
			"status":   constants.OwnershipTransferred,
		}).Error
}

func (c VehicleCustomerRepository) toDomain(model models.CustomerVehicle) entity.CustomerVehicle {
	return entity.CustomerVehicle{
		ID: model.ID,
		Vehicle: entity.Vehicle{
			ID:              model.Vehicle.ID,
			Vin:             model.Vehicle.Vin,
			LicensePlate:    model.Vehicle.LicensePlate,
			WarrantyEndDate: model.Vehicle.WarrantyEndDate,
			Status:          model.Vehicle.Status,
			CreatedAt:       model.Vehicle.CreatedAt,
			UpdatedAt:       model.Vehicle.UpdatedAt,
		},
		CustomerID: model.CustomerID,
		OwnedFrom:  model.OwnedFrom,
		OwnedTo:    model.OwnedTo,
		Status:     model.Status,
		CreatedAt:  model.CreatedAt,
	}
}
func (c VehicleCustomerRepository) toModel(e entity.CustomerVehicle) models.CustomerVehicle {
	return models.CustomerVehicle{
		ID:         e.ID,
		CustomerID: e.CustomerID,
		VehicleID:  e.Vehicle.ID,
		Vehicle: models.Vehicle{
			ID:              e.Vehicle.ID,
			Vin:             e.Vehicle.Vin,
			LicensePlate:    e.Vehicle.LicensePlate,
			WarrantyEndDate: e.Vehicle.WarrantyEndDate,
			Status:          e.Vehicle.Status,
			CreatedAt:       e.Vehicle.CreatedAt,
			UpdatedAt:       e.Vehicle.UpdatedAt,
		},
		OwnedFrom: e.OwnedFrom,
		OwnedTo:   e.OwnedTo,
		Status:    e.Status,
		CreatedAt: e.CreatedAt,
	}
}

func NewVehicleCustomerRepository(db *gorm.DB) ports.VehicleCustomerRepository {
	return VehicleCustomerRepository{
		db: db,
	}
}
