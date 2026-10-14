package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"scheduler-service/internal/constants"
	"scheduler-service/internal/domain/entity"
	"scheduler-service/internal/domain/events"
	"scheduler-service/internal/domain/ports"
	"scheduler-service/pkg/logger"
	"slices"
	"strconv"
	"time"

	"github.com/shopspring/decimal"
)

// slotStep is the spacing between offered start times.
const slotStep = 30 * time.Minute

// myAppointmentsLimit caps how many of a customer's appointments one call
// returns.
const myAppointmentsLimit = 100

type BookingService struct {
	catalog      ports.CatalogRepository
	appointments ports.AppointmentRepository
	outbox       ports.OutboxRepository
	txManager    ports.TxManager
	customers    ports.CustomerDirectory
	vehicles     ports.VehicleDirectory
	now          func() time.Time
}

func NewBookingService(
	catalog ports.CatalogRepository,
	appointments ports.AppointmentRepository,
	outbox ports.OutboxRepository,
	txManager ports.TxManager,
	customers ports.CustomerDirectory,
	vehicles ports.VehicleDirectory,
) *BookingService {
	return &BookingService{
		catalog:      catalog,
		appointments: appointments,
		outbox:       outbox,
		txManager:    txManager,
		customers:    customers,
		vehicles:     vehicles,
		now:          time.Now,
	}
}

// ListHubs implements [ports.BookingService].
func (s *BookingService) ListHubs(ctx context.Context) ([]entity.Hub, error) {
	return s.catalog.ListHubs(ctx)
}

// ListServices implements [ports.BookingService].
func (s *BookingService) ListServices(ctx context.Context) ([]entity.CatalogService, error) {
	return s.catalog.ListServices(ctx)
}

// Availability implements [ports.BookingService]. A slot is available when
// it starts in the future and at least one bookable bay has nothing
// reserved for any part of it. It's advisory: Book re-checks inside its
// transaction, and the database constraint has the final word.
func (s *BookingService) Availability(ctx context.Context, hubID int64, date string, serviceIDs []int64) (ports.Availability, error) {
	hub, err := s.catalog.GetHub(ctx, hubID)
	if err != nil {
		return ports.Availability{}, err
	}
	services, err := s.resolveServices(ctx, serviceIDs)
	if err != nil {
		return ports.Availability{}, err
	}
	loc := hubLocation(hub)
	day, err := time.ParseInLocation(time.DateOnly, date, loc)
	if err != nil {
		return ports.Availability{}, fmt.Errorf("%w: date must be YYYY-MM-DD", ports.ErrInvalidInput)
	}

	duration := totalDuration(services)
	open, closing := openingHours(hub, day, loc)
	bays, err := s.catalog.ListBookableBays(ctx, hubID)
	if err != nil {
		return ports.Availability{}, err
	}
	reservations, err := s.appointments.Reservations(ctx, hubID, open, closing)
	if err != nil {
		return ports.Availability{}, err
	}

	now := s.now()
	slots := []ports.Slot{}
	for start := open; !start.Add(duration).After(closing); start = start.Add(slotStep) {
		end := start.Add(duration)
		slots = append(slots, ports.Slot{
			Start:     start,
			End:       end,
			Available: start.After(now) && anyBayFree(bays, reservations, start, end),
		})
	}

	return ports.Availability{
		HubID:           hubID,
		Date:            date,
		TimeZone:        loc.String(),
		DurationMinutes: int(duration / time.Minute),
		Slots:           slots,
	}, nil
}

// Book implements [ports.BookingService]. Who the caller is and that they
// own the vehicle are checked with customer-service and vehicle-service
// before the write (architecture-design.md §5a). The bay is then picked and
// reserved, and the AppointmentBooked event written, in one transaction.
func (s *BookingService) Book(ctx context.Context, caller entity.Principal, req ports.BookingRequest) (entity.Appointment, error) {
	hub, err := s.catalog.GetHub(ctx, req.HubID)
	if err != nil {
		return entity.Appointment{}, err
	}
	services, err := s.resolveServices(ctx, req.ServiceIDs)
	if err != nil {
		return entity.Appointment{}, err
	}

	loc := hubLocation(hub)
	start := req.Start.In(loc)
	end := start.Add(totalDuration(services))
	if !start.After(s.now()) {
		return entity.Appointment{}, fmt.Errorf("%w: pick a start time in the future", ports.ErrInvalidInput)
	}
	open, closing := openingHours(hub, start, loc)
	if start.Before(open) || end.After(closing) {
		return entity.Appointment{}, fmt.Errorf("%w: %s is open %s–%s and the services take %d minutes",
			ports.ErrInvalidInput, hub.Name, open.Format("15:04"), closing.Format("15:04"), int(end.Sub(start)/time.Minute))
	}

	customer, err := s.customers.CurrentCustomer(ctx, caller.Token)
	if err != nil {
		return entity.Appointment{}, err
	}
	owned, err := s.vehicles.CustomerVehicles(ctx, caller.Token, customer.ID)
	if err != nil {
		return entity.Appointment{}, err
	}
	if !slices.ContainsFunc(owned, func(v entity.Vehicle) bool { return v.ID == req.VehicleID }) {
		return entity.Appointment{}, fmt.Errorf("%w: vehicle %d isn't registered to you", ports.ErrInvalidInput, req.VehicleID)
	}

	appointment := entity.Appointment{
		HubID:       hub.ID,
		CustomerID:  customer.ID,
		VehicleID:   req.VehicleID,
		Start:       start.UTC(),
		End:         end.UTC(),
		Status:      constants.AppointmentBooked,
		TotalAmount: decimal.Zero,
	}
	for _, svc := range services {
		appointment.Services = append(appointment.Services, entity.AppointmentService{ServiceID: svc.ID, Name: svc.Name, Fee: svc.Fee})
		appointment.TotalAmount = appointment.TotalAmount.Add(svc.Fee)
	}

	err = s.txManager.RunInTx(ctx, func(ctx context.Context) error {
		if err := s.appointments.LockVehicle(ctx, req.VehicleID); err != nil {
			return err
		}
		busy, err := s.appointments.VehicleBusy(ctx, req.VehicleID, appointment.Start, appointment.End)
		if err != nil {
			return err
		}
		if busy {
			return fmt.Errorf("%w: this vehicle already has a booking at that time", ports.ErrConflict)
		}

		bayID, err := s.appointments.FreeBay(ctx, hub.ID, appointment.Start, appointment.End)
		if err != nil {
			return err
		}
		appointment.BayID = bayID

		created, err := s.appointments.Create(ctx, appointment)
		if err != nil {
			return err
		}
		created.Services = appointment.Services
		appointment = created

		return s.writeEvent(ctx, events.AppointmentBooked, constants.TopicAppointmentBooked, appointment)
	})
	if err != nil {
		if !isClientError(err) {
			logger.ErrorContext(ctx, "failed to book appointment", "hub_id", req.HubID, "vehicle_id", req.VehicleID, "error", err)
		}
		return entity.Appointment{}, err
	}

	return appointment, nil
}

// MyAppointments implements [ports.BookingService].
func (s *BookingService) MyAppointments(ctx context.Context, caller entity.Principal) ([]entity.Appointment, error) {
	customer, err := s.customers.CurrentCustomer(ctx, caller.Token)
	if err != nil {
		return nil, err
	}
	return s.appointments.ListByCustomer(ctx, customer.ID, myAppointmentsLimit)
}

// Cancel implements [ports.BookingService]. Another customer's appointment
// is reported as not found rather than forbidden, so ids can't be probed.
func (s *BookingService) Cancel(ctx context.Context, caller entity.Principal, appointmentID int64) (entity.Appointment, error) {
	customer, err := s.customers.CurrentCustomer(ctx, caller.Token)
	if err != nil {
		return entity.Appointment{}, err
	}
	appointment, err := s.appointments.GetByID(ctx, appointmentID)
	if err != nil {
		return entity.Appointment{}, err
	}
	if appointment.CustomerID != customer.ID {
		return entity.Appointment{}, ports.ErrNotFound
	}
	if appointment.Status != constants.AppointmentBooked {
		return entity.Appointment{}, fmt.Errorf("%w: only a booked appointment can be cancelled (this one is %s)", ports.ErrConflict, appointment.Status)
	}
	if !appointment.Start.After(s.now()) {
		return entity.Appointment{}, fmt.Errorf("%w: this appointment has already started", ports.ErrConflict)
	}

	err = s.txManager.RunInTx(ctx, func(ctx context.Context) error {
		if err := s.appointments.Cancel(ctx, appointmentID); err != nil {
			return err
		}
		appointment.Status = constants.AppointmentCancelled
		appointment.UpdatedAt = s.now()
		return s.writeEvent(ctx, events.AppointmentCancelled, constants.TopicAppointmentCancelled, appointment)
	})
	if err != nil {
		if !isClientError(err) {
			logger.ErrorContext(ctx, "failed to cancel appointment", "appointment_id", appointmentID, "error", err)
		}
		return entity.Appointment{}, err
	}

	return appointment, nil
}

func (s *BookingService) writeEvent(ctx context.Context, eventType events.EventType, topic string, appointment entity.Appointment) error {
	payload, err := json.Marshal(events.NewAppointmentEvent(eventType, s.now(), appointment))
	if err != nil {
		return err
	}
	return s.outbox.Create(ctx, entity.OutboxMessage{
		Topic:   topic,
		Key:     strconv.FormatInt(appointment.ID, 10),
		Payload: payload,
	})
}

// resolveServices looks up the requested services in the active catalog,
// keeping the caller's order and dropping duplicates.
func (s *BookingService) resolveServices(ctx context.Context, ids []int64) ([]entity.CatalogService, error) {
	if len(ids) == 0 {
		return nil, fmt.Errorf("%w: pick at least one service", ports.ErrInvalidInput)
	}
	catalog, err := s.catalog.ListServices(ctx)
	if err != nil {
		return nil, err
	}

	byID := make(map[int64]entity.CatalogService, len(catalog))
	for _, svc := range catalog {
		byID[svc.ID] = svc
	}
	var resolved []entity.CatalogService
	seen := map[int64]bool{}
	for _, id := range ids {
		if seen[id] {
			continue
		}
		svc, ok := byID[id]
		if !ok {
			return nil, fmt.Errorf("%w: service %d doesn't exist or can't be booked", ports.ErrInvalidInput, id)
		}
		seen[id] = true
		resolved = append(resolved, svc)
	}
	return resolved, nil
}

func totalDuration(services []entity.CatalogService) time.Duration {
	var total time.Duration
	for _, svc := range services {
		total += time.Duration(svc.DurationMinutes) * time.Minute
	}
	return total
}

// hubLocation is the hub's time zone, or UTC if it isn't a known IANA name.
func hubLocation(hub entity.Hub) *time.Location {
	loc, err := time.LoadLocation(hub.TimeZone)
	if err != nil {
		return time.UTC
	}
	return loc
}

// openingHours returns when the hub opens and closes on day's local date.
func openingHours(hub entity.Hub, day time.Time, loc *time.Location) (time.Time, time.Time) {
	y, m, d := day.In(loc).Date()
	// time.Date rather than midnight.Add, so a DST change that day doesn't
	// shift the wall-clock hours.
	at := func(offset time.Duration) time.Time {
		return time.Date(y, m, d, int(offset/time.Hour), int(offset%time.Hour/time.Minute), 0, 0, loc)
	}
	return at(hub.OpenHour), at(hub.CloseHour)
}

// anyBayFree says whether some bay has no reservation overlapping
// [start, end).
func anyBayFree(bays []entity.Bay, reservations []entity.BayReservation, start, end time.Time) bool {
	for _, bay := range bays {
		free := true
		for _, r := range reservations {
			if r.BayID == bay.ID && r.Start.Before(end) && start.Before(r.End) {
				free = false
				break
			}
		}
		if free {
			return true
		}
	}
	return false
}

// isClientError is an error the caller caused or can retry, not a fault
// worth logging here.
func isClientError(err error) bool {
	return errors.Is(err, ports.ErrInvalidInput) || errors.Is(err, ports.ErrConflict) ||
		errors.Is(err, ports.ErrNotFound) || errors.Is(err, ports.ErrNoCustomerProfile)
}
