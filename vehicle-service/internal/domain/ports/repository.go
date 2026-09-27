package ports

import (
	"context"
	"time"
	"vehicle-service/internal/domain/entity"
)

type VehicleRepository interface {
	Create(ctx context.Context, vehicle entity.Vehicle) (entity.Vehicle, error)
	GetByID(ctx context.Context, id int64) (entity.Vehicle, error)
	// GetByIDForUpdate is GetByID plus a row lock held until the calling
	// transaction ends; only meaningful inside TxManager.RunInTx.
	GetByIDForUpdate(ctx context.Context, id int64) (entity.Vehicle, error)
	List(ctx context.Context, params ListVehiclesParams) (VehiclePage, error)
	Update(ctx context.Context, vehicle entity.Vehicle) error
	Delete(ctx context.Context, id int64) error
}

type VehicleCustomerRepository interface {
	// ListCurrentByCustomer returns the customer's current (not transferred)
	// ownerships with their vehicles, most recently acquired first.
	ListCurrentByCustomer(ctx context.Context, customerID int64) ([]entity.CustomerVehicle, error)
	AssignVehicleToCustomer(ctx context.Context, vehicleID, customerID int64, date time.Time) error
	UnassignVehicleFromCustomer(ctx context.Context, vehicleID, customerID int64, date time.Time) error
}

type VehicleMaterialRepository interface {
	// ListByVehicle returns the materials installed on a vehicle, most
	// recently installed first.
	ListByVehicle(ctx context.Context, vehicleID int64) ([]entity.VehicleMaterial, error)
}

type ServiceHistoryRepository interface {
	// Record stores entry unless one already exists for its AppointmentID,
	// reporting whether it was new. Returns ErrNotFound if the vehicle
	// doesn't exist.
	Record(ctx context.Context, entry entity.ServiceHistoryEntry) (created bool, err error)
	// ListByVehicle returns up to limit entries, most recent first.
	ListByVehicle(ctx context.Context, vehicleID int64, limit int) ([]entity.ServiceHistoryEntry, error)
}
