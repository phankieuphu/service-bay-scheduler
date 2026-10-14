package repository

import (
	"context"
	"errors"
	"fmt"
	"scheduler-service/internal/adapters/database/models"
	database_provider "scheduler-service/internal/adapters/database/provider"
	"scheduler-service/internal/constants"
	"scheduler-service/internal/domain/entity"
	"scheduler-service/internal/domain/ports"
	"slices"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

// pgExclusionViolationCode is PostgreSQL's SQLSTATE for an exclusion
// constraint violation: here, no_bay_double_booking.
const pgExclusionViolationCode = "23P01"

// vehicleLockNamespace keeps LockVehicle's advisory locks apart from any
// other advisory lock on the database (goose takes one while migrating).
const vehicleLockNamespace = 41_001

// liveStatuses are the appointment statuses that still hold their slot.
var liveStatuses = []constants.AppointmentStatus{
	constants.AppointmentBooked,
	constants.AppointmentCheckedIn,
	constants.AppointmentInProgress,
}

type AppointmentRepository struct {
	db *gorm.DB
}

func NewAppointmentRepository(db *gorm.DB) ports.AppointmentRepository {
	return AppointmentRepository{db: db}
}

// LockVehicle implements [ports.AppointmentRepository]. It takes a
// transaction-scoped advisory lock, so it must run inside RunInTx.
//
// The id goes in as a string: with `?::text` pgx plans a text parameter and
// refuses to encode an int64 into it ("cannot find encode plan"), which
// failed every booking.
func (r AppointmentRepository) LockVehicle(ctx context.Context, vehicleID int64) error {
	return database_provider.DBFromContext(ctx, r.db).
		Exec("SELECT pg_advisory_xact_lock(?, hashtext(?::text))", vehicleLockNamespace, strconv.FormatInt(vehicleID, 10)).Error
}

// VehicleBusy implements [ports.AppointmentRepository].
func (r AppointmentRepository) VehicleBusy(ctx context.Context, vehicleID int64, start, end time.Time) (bool, error) {
	var busy bool
	err := database_provider.DBFromContext(ctx, r.db).Raw(`
		SELECT EXISTS (
			SELECT 1
			FROM appointment a
			JOIN appointment_vehicle av ON av.appointment_id = a.id
			WHERE av.customer_vehicle_id = ?
			  AND a.status IN ?
			  AND tstzrange(a.scheduled_start, a.scheduled_end, '[)') && tstzrange(?, ?, '[)')
		)`, vehicleID, liveStatuses, start, end).Scan(&busy).Error
	return busy, err
}

// FreeBay implements [ports.AppointmentRepository].
func (r AppointmentRepository) FreeBay(ctx context.Context, hubID int64, start, end time.Time) (int64, error) {
	var ids []int64
	err := database_provider.DBFromContext(ctx, r.db).Raw(`
		SELECT b.id
		FROM hub_bay b
		WHERE b.hub_id = ?
		  AND b.status = ?
		  AND NOT EXISTS (
			SELECT 1 FROM appointment_bay ab
			WHERE ab.service_bay_id = b.id
			  AND ab.status <> 'CANCELLED'
			  AND ab.slot && tstzrange(?, ?, '[)')
		  )
		ORDER BY b.code
		LIMIT 1`, hubID, constants.BayAvailable, start, end).Scan(&ids).Error
	if err != nil {
		return 0, err
	}
	if len(ids) == 0 {
		return 0, fmt.Errorf("%w: no bay is free at that time", ports.ErrConflict)
	}
	return ids[0], nil
}

// Reservations implements [ports.AppointmentRepository].
func (r AppointmentRepository) Reservations(ctx context.Context, hubID int64, from, to time.Time) ([]entity.BayReservation, error) {
	var rows []models.AppointmentBay
	err := database_provider.DBFromContext(ctx, r.db).Raw(`
		SELECT ab.service_bay_id, ab.start_time, ab.end_time
		FROM appointment_bay ab
		JOIN hub_bay b ON b.id = ab.service_bay_id
		WHERE b.hub_id = ?
		  AND ab.status <> 'CANCELLED'
		  AND ab.slot && tstzrange(?, ?, '[)')`, hubID, from, to).Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	reservations := make([]entity.BayReservation, len(rows))
	for i, row := range rows {
		reservations[i] = entity.BayReservation{BayID: row.ServiceBayID, Start: row.StartTime, End: row.EndTime}
	}
	return reservations, nil
}

// Create implements [ports.AppointmentRepository].
func (r AppointmentRepository) Create(ctx context.Context, a entity.Appointment) (entity.Appointment, error) {
	db := database_provider.DBFromContext(ctx, r.db)

	appointment := models.Appointment{
		DealershipID:   a.HubID,
		CustomerID:     a.CustomerID,
		ScheduledStart: a.Start,
		ScheduledEnd:   a.End,
		Status:         string(a.Status),
		TotalAmount:    a.TotalAmount,
	}
	if err := db.Create(&appointment).Error; err != nil {
		return entity.Appointment{}, err
	}

	vehicle := models.AppointmentVehicle{
		AppointmentID:     appointment.ID,
		CustomerVehicleID: a.VehicleID,
		Status:            "PENDING",
	}
	if err := db.Create(&vehicle).Error; err != nil {
		return entity.Appointment{}, err
	}

	bay := models.AppointmentBay{
		AppointmentID: appointment.ID,
		ServiceBayID:  a.BayID,
		StartTime:     a.Start,
		EndTime:       a.End,
		Status:        "RESERVED",
	}
	if err := db.Create(&bay).Error; err != nil {
		if isExclusionViolation(err) {
			return entity.Appointment{}, fmt.Errorf("%w: that slot was just taken, pick another", ports.ErrConflict)
		}
		return entity.Appointment{}, err
	}

	if len(a.Services) > 0 {
		lines := make([]models.AppointmentService, len(a.Services))
		for i, svc := range a.Services {
			lines[i] = models.AppointmentService{
				AppointmentID:        appointment.ID,
				ServiceID:            svc.ServiceID,
				AppointmentVehicleID: &vehicle.ID,
				AppointmentBayID:     &bay.ID,
				FeeCharged:           svc.Fee,
				Status:               "PENDING",
			}
		}
		if err := db.Create(&lines).Error; err != nil {
			return entity.Appointment{}, err
		}
	}

	a.ID = appointment.ID
	a.CreatedAt = appointment.CreatedAt
	a.UpdatedAt = appointment.UpdatedAt
	return a, nil
}

// GetByID implements [ports.AppointmentRepository].
func (r AppointmentRepository) GetByID(ctx context.Context, id int64) (entity.Appointment, error) {
	appointments, err := r.load(ctx, "a.id = ?", []any{id}, 1)
	if err != nil {
		return entity.Appointment{}, err
	}
	if len(appointments) == 0 {
		return entity.Appointment{}, fmt.Errorf("%w: appointment %d", ports.ErrNotFound, id)
	}
	return appointments[0], nil
}

// ListByCustomer implements [ports.AppointmentRepository].
func (r AppointmentRepository) ListByCustomer(ctx context.Context, customerID int64, limit int) ([]entity.Appointment, error) {
	return r.load(ctx, "a.customer_id = ?", []any{customerID}, limit)
}

// Cancel implements [ports.AppointmentRepository].
func (r AppointmentRepository) Cancel(ctx context.Context, id int64) error {
	db := database_provider.DBFromContext(ctx, r.db)

	result := db.Model(&models.Appointment{}).
		Where("id = ? AND status = ?", id, constants.AppointmentBooked).
		Update("status", constants.AppointmentCancelled)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("%w: appointment %d isn't booked any more", ports.ErrConflict, id)
	}

	if err := db.Model(&models.AppointmentBay{}).
		Where("appointment_id = ?", id).
		Update("status", "CANCELLED").Error; err != nil {
		return err
	}
	return db.Model(&models.AppointmentService{}).
		Where("appointment_id = ?", id).
		Update("status", "CANCELLED").Error
}

// Complete implements [ports.AppointmentRepository].
func (r AppointmentRepository) Complete(ctx context.Context, id int64) error {
	db := database_provider.DBFromContext(ctx, r.db)

	result := db.Model(&models.Appointment{}).
		Where("id = ? AND status IN ?", id, []constants.AppointmentStatus{
			constants.AppointmentBooked, constants.AppointmentCheckedIn, constants.AppointmentInProgress,
		}).
		Update("status", constants.AppointmentCompleted)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("%w: appointment %d can't be completed from its current status", ports.ErrConflict, id)
	}

	// RELEASED: the work is done. The slot stays recorded (the exclusion
	// constraint only ignores CANCELLED), which is right for time that was used.
	if err := db.Model(&models.AppointmentBay{}).
		Where("appointment_id = ? AND status IN ?", id, []string{"RESERVED", "OCCUPIED"}).
		Update("status", "RELEASED").Error; err != nil {
		return err
	}
	return db.Model(&models.AppointmentService{}).
		Where("appointment_id = ? AND status <> ?", id, "CANCELLED").
		Update("status", "DONE").Error
}

// ListByHub implements [ports.AppointmentRepository].
func (r AppointmentRepository) ListByHub(ctx context.Context, hubID int64, from, to time.Time) ([]entity.Appointment, error) {
	appointments, err := r.load(ctx, "a.dealership_id = ? AND a.scheduled_start >= ? AND a.scheduled_start < ?",
		[]any{hubID, from, to}, hubDayLimit)
	if err != nil {
		return nil, err
	}
	slices.Reverse(appointments) // load returns latest first
	return appointments, nil
}

// hubDayLimit bounds one hub-day listing; a hub can't fit more bookings
// than this into its opening hours.
const hubDayLimit = 500

// appointmentRow is one appointment joined with its (single) vehicle and
// bay, as load selects it.
type appointmentRow struct {
	models.Appointment
	VehicleID int64 `gorm:"column:vehicle_id"`
	BayID     int64 `gorm:"column:bay_id"`
}

// serviceRow is a booked service with its catalog name.
type serviceRow struct {
	models.AppointmentService
	Name string `gorm:"column:name"`
}

// load reads appointments matching where (on alias a), latest start first,
// with their services. Bookings made here always have one vehicle and one
// bay; the lowest id is taken if a row somehow has more.
func (r AppointmentRepository) load(ctx context.Context, where string, args []any, limit int) ([]entity.Appointment, error) {
	db := database_provider.DBFromContext(ctx, r.db)

	var rows []appointmentRow
	err := db.Raw(`
		SELECT a.*,
		       COALESCE((SELECT av.customer_vehicle_id FROM appointment_vehicle av
		                 WHERE av.appointment_id = a.id ORDER BY av.id LIMIT 1), 0) AS vehicle_id,
		       COALESCE((SELECT ab.service_bay_id FROM appointment_bay ab
		                 WHERE ab.appointment_id = a.id ORDER BY ab.id LIMIT 1), 0) AS bay_id
		FROM appointment a
		WHERE `+where+`
		ORDER BY a.scheduled_start DESC, a.id DESC
		LIMIT ?`, append(args, limit)...).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return []entity.Appointment{}, nil
	}

	ids := make([]int64, len(rows))
	for i, row := range rows {
		ids[i] = row.ID
	}
	var lines []serviceRow
	err = db.Raw(`
		SELECT s.*, COALESCE(c.name, 'Service #' || s.service_id) AS name
		FROM appointment_service s
		LEFT JOIN catalog_service c ON c.id = s.service_id
		WHERE s.appointment_id IN ?
		ORDER BY s.id`, ids).Scan(&lines).Error
	if err != nil {
		return nil, err
	}
	services := map[int64][]entity.AppointmentService{}
	for _, line := range lines {
		services[line.AppointmentID] = append(services[line.AppointmentID], entity.AppointmentService{
			ServiceID: line.ServiceID,
			Name:      line.Name,
			Fee:       line.FeeCharged,
		})
	}

	appointments := make([]entity.Appointment, len(rows))
	for i, row := range rows {
		appointments[i] = entity.Appointment{
			ID:          row.ID,
			HubID:       row.DealershipID,
			CustomerID:  row.CustomerID,
			VehicleID:   row.VehicleID,
			BayID:       row.BayID,
			Start:       row.ScheduledStart,
			End:         row.ScheduledEnd,
			Status:      constants.AppointmentStatus(row.Status),
			TotalAmount: row.TotalAmount,
			Services:    services[row.ID],
			CreatedAt:   row.CreatedAt,
			UpdatedAt:   row.UpdatedAt,
		}
	}
	return appointments, nil
}

func isExclusionViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == pgExclusionViolationCode
}
