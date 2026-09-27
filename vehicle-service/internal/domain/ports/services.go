package ports

import (
	"context"
	"time"
	"vehicle-service/internal/domain/entity"
	"vehicle-service/internal/domain/events"
)

type VehicleService interface {
	GetCustomerVehicles(ctx context.Context, customerID int64) ([]entity.CustomerVehicle, error)
	GetVehicle(ctx context.Context, vehicleID int64) (entity.Vehicle, error)
	ListVehicles(ctx context.Context, params ListVehiclesParams) (VehiclePage, error)
	TransferVehicle(ctx context.Context, transferVehicle entity.TransferVehicle) error
	RegisterVehicle(ctx context.Context, vehicle entity.Vehicle) (entity.Vehicle, error) // register new vehicle
	UpdateVehicle(ctx context.Context, vehicleID int64, update entity.VehicleUpdate) (entity.Vehicle, error)
	// AssignInitialOwner gives a vehicle with no current owner its first
	// one; after that, ownership changes go through TransferVehicle.
	AssignInitialOwner(ctx context.Context, vehicleID, customerID int64, date time.Time) error
	GetVehicleMaterials(ctx context.Context, vehicleID int64) ([]entity.VehicleMaterial, error)
	// GetWarranty evaluates the warranty on asOf's calendar day, so billing
	// can ask about the day the service was done, not just today.
	GetWarranty(ctx context.Context, vehicleID int64, asOf time.Time) (entity.Warranty, error)
	GetServiceHistory(ctx context.Context, vehicleID int64, limit int) ([]entity.ServiceHistoryEntry, error)
	// RecordServiceCompleted adds a completed appointment to the vehicle's
	// history. Redelivering the same appointment is a no-op.
	RecordServiceCompleted(ctx context.Context, event events.ServiceCompleted) error
}
