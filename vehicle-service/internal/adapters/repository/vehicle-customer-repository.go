package repository

import (
	"context"
	"time"
	"vehicle-service/internal/adapters/database/models"
	"vehicle-service/internal/domain/entity"
	"vehicle-service/internal/domain/ports"

	"gorm.io/gorm"
)

type VehicleCustomerRepository struct {
	db *gorm.DB
}

// AssignVehicleToCustomer implements [ports.VehicleCustomerRepository].
func (c VehicleCustomerRepository) AssignVehicleToCustomer(ctx context.Context, vehicleID int, customerID int, date time.Time) error {
	panic("unimplemented")
}

// GetCustomerVehicle implements [ports.VehicleCustomerRepository].
func (c VehicleCustomerRepository) GetCustomerVehicle(ctx context.Context, customerID int) (entity.CustomerVehicle, error) {
	panic("unimplemented")
}

// TransferVehicleToCustomer implements [ports.VehicleCustomerRepository].
func (c VehicleCustomerRepository) TransferVehicleToCustomer(ctx context.Context, customerID int) error {
	panic("unimplemented")
}

// UnassignVehicleFromCustomer implements [ports.VehicleCustomerRepository].
func (c VehicleCustomerRepository) UnassignVehicleFromCustomer(ctx context.Context, vehicleID int, customerID int, date time.Time) error {
	panic("unimplemented")
}

func (c VehicleCustomerRepository) toDomain(models.CustomerVehicle) entity.CustomerVehicle {
	return entity.CustomerVehicle{
		ID:        0,
		Name:      "",
		Email:     "",
		Phone:     "",
		BirthDay:  time.Time{},
		Status:    "",
		CreatedAt: time.Time{},
		UpdatedAt: time.Time{},
	}
}
func (c VehicleCustomerRepository) toModel(entity.CustomerVehicle) models.CustomerVehicle {
	return models.CustomerVehicle{
		ID:         0,
		CustomerID: 0,
		VehicleID:  0,
		Vehicle:    models.Vehicle{},
		OwnedFrom:  time.Time{},
		OwnedTo:    time.Time{},
		Status:     "",
		CreatedAt:  time.Time{},
		UpdatedAt:  time.Time{},
	}
}

func NewVehicleCustomerRepository(db *gorm.DB) ports.VehicleCustomerRepository {
	return VehicleCustomerRepository{
		db: db,
	}
}
