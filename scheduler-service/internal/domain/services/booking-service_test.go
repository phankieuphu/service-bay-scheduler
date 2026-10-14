package services

import (
	"context"
	"encoding/json"
	"errors"
	"scheduler-service/internal/constants"
	"scheduler-service/internal/domain/entity"
	"scheduler-service/internal/domain/ports"
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

// --- mocks -----------------------------------------------------------------

type MockCatalogRepository struct {
	Hubs     []entity.Hub
	Bays     []entity.Bay
	Services []entity.CatalogService
}

func (m *MockCatalogRepository) ListHubs(context.Context) ([]entity.Hub, error) { return m.Hubs, nil }
func (m *MockCatalogRepository) GetHub(_ context.Context, id int64) (entity.Hub, error) {
	for _, h := range m.Hubs {
		if h.ID == id {
			return h, nil
		}
	}
	return entity.Hub{}, ports.ErrNotFound
}
func (m *MockCatalogRepository) ListBookableBays(_ context.Context, hubID int64) ([]entity.Bay, error) {
	var bays []entity.Bay
	for _, b := range m.Bays {
		if b.HubID == hubID {
			bays = append(bays, b)
		}
	}
	return bays, nil
}
func (m *MockCatalogRepository) ListServices(context.Context) ([]entity.CatalogService, error) {
	return m.Services, nil
}

type MockAppointmentRepository struct {
	Busy           bool
	FreeBayID      int64
	FreeBayErr     error
	CreateErr      error
	Reserved       []entity.BayReservation
	Stored         map[int64]entity.Appointment
	Created        []entity.Appointment
	Cancelled      []int64
	LockedVehicles []int64
}

func (m *MockAppointmentRepository) LockVehicle(_ context.Context, vehicleID int64) error {
	m.LockedVehicles = append(m.LockedVehicles, vehicleID)
	return nil
}
func (m *MockAppointmentRepository) VehicleBusy(context.Context, int64, time.Time, time.Time) (bool, error) {
	return m.Busy, nil
}
func (m *MockAppointmentRepository) FreeBay(context.Context, int64, time.Time, time.Time) (int64, error) {
	return m.FreeBayID, m.FreeBayErr
}
func (m *MockAppointmentRepository) Reservations(context.Context, int64, time.Time, time.Time) ([]entity.BayReservation, error) {
	return m.Reserved, nil
}
func (m *MockAppointmentRepository) Create(_ context.Context, a entity.Appointment) (entity.Appointment, error) {
	if m.CreateErr != nil {
		return entity.Appointment{}, m.CreateErr
	}
	a.ID = int64(100 + len(m.Created))
	m.Created = append(m.Created, a)
	return a, nil
}
func (m *MockAppointmentRepository) GetByID(_ context.Context, id int64) (entity.Appointment, error) {
	a, ok := m.Stored[id]
	if !ok {
		return entity.Appointment{}, ports.ErrNotFound
	}
	return a, nil
}
func (m *MockAppointmentRepository) ListByCustomer(_ context.Context, customerID int64, _ int) ([]entity.Appointment, error) {
	var out []entity.Appointment
	for _, a := range m.Stored {
		if a.CustomerID == customerID {
			out = append(out, a)
		}
	}
	return out, nil
}
func (m *MockAppointmentRepository) Cancel(_ context.Context, id int64) error {
	m.Cancelled = append(m.Cancelled, id)
	return nil
}

type MockOutboxRepository struct{ Written []entity.OutboxMessage }

func (m *MockOutboxRepository) Create(_ context.Context, msg entity.OutboxMessage) error {
	m.Written = append(m.Written, msg)
	return nil
}
func (m *MockOutboxRepository) FetchUnpublished(context.Context, int) ([]entity.OutboxMessage, error) {
	return nil, nil
}
func (m *MockOutboxRepository) MarkPublished(context.Context, []int64) error { return nil }

// MockTxManager runs fn directly; tests see what it wrote whether or not
// it "committed", so they check the returned error too.
type MockTxManager struct{}

func (MockTxManager) RunInTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

type MockCustomers struct {
	Customer entity.Customer
	Err      error
}

func (m MockCustomers) CurrentCustomer(context.Context, string) (entity.Customer, error) {
	return m.Customer, m.Err
}

type MockVehicles struct{ Owned []entity.Vehicle }

func (m MockVehicles) CustomerVehicles(context.Context, string, int64) ([]entity.Vehicle, error) {
	return m.Owned, nil
}

// --- fixtures --------------------------------------------------------------

var saigon, _ = time.LoadLocation("Asia/Ho_Chi_Minh")

// now is 09:10 local on 2026-10-20 at hub 1.
var now = time.Date(2026, 10, 20, 9, 10, 0, 0, saigon)

type fixture struct {
	svc          *BookingService
	appointments *MockAppointmentRepository
	outbox       *MockOutboxRepository
}

func newFixture(customers MockCustomers) fixture {
	catalog := &MockCatalogRepository{
		Hubs: []entity.Hub{{ID: 1, Name: "District 1 Hub", OpenHour: 8 * time.Hour, CloseHour: 12 * time.Hour, TimeZone: "Asia/Ho_Chi_Minh", Active: true}},
		Bays: []entity.Bay{{ID: 11, HubID: 1, Code: "BAY-01"}, {ID: 12, HubID: 1, Code: "BAY-02"}},
		Services: []entity.CatalogService{
			{ID: 1, Name: "Oil change", Fee: decimal.RequireFromString("45.00"), DurationMinutes: 30},
			{ID: 2, Name: "Brake inspection", Fee: decimal.RequireFromString("60.50"), DurationMinutes: 60},
		},
	}
	appointments := &MockAppointmentRepository{FreeBayID: 11, Stored: map[int64]entity.Appointment{}}
	outbox := &MockOutboxRepository{}
	svc := NewBookingService(catalog, appointments, outbox, MockTxManager{}, customers,
		MockVehicles{Owned: []entity.Vehicle{{ID: 7, VIN: "VIN7"}}})
	svc.now = func() time.Time { return now }
	return fixture{svc: svc, appointments: appointments, outbox: outbox}
}

var jane = MockCustomers{Customer: entity.Customer{ID: 42, Name: "Jane"}}

func at(hour, minute int) time.Time {
	return time.Date(2026, 10, 20, hour, minute, 0, 0, saigon)
}

// --- Availability ----------------------------------------------------------

func TestAvailability_SlotsWithinOpeningHours(t *testing.T) {
	f := newFixture(jane)
	// Brake inspection (60 min) at a hub open 08:00–12:00: starts 08:00..11:00.
	got, err := f.svc.Availability(t.Context(), 1, "2026-10-20", []int64{2})
	if err != nil {
		t.Fatal(err)
	}
	if got.DurationMinutes != 60 || got.TimeZone != "Asia/Ho_Chi_Minh" || len(got.Slots) != 7 {
		t.Fatalf("availability = %+v", got)
	}
	if !got.Slots[0].Start.Equal(at(8, 0)) || !got.Slots[6].Start.Equal(at(11, 0)) || !got.Slots[6].End.Equal(at(12, 0)) {
		t.Errorf("slots run %v..%v", got.Slots[0].Start, got.Slots[6].End)
	}
	// 08:00, 08:30 and 09:00 are before now (09:10).
	for i, slot := range got.Slots {
		if want := i >= 3; slot.Available != want {
			t.Errorf("slot %v available = %v, want %v", slot.Start.In(saigon).Format("15:04"), slot.Available, want)
		}
	}
}

func TestAvailability_TakenWhenEveryBayIsReserved(t *testing.T) {
	f := newFixture(jane)
	f.appointments.Reserved = []entity.BayReservation{
		{BayID: 11, Start: at(10, 0), End: at(11, 0)},
		{BayID: 12, Start: at(10, 30), End: at(11, 30)},
	}

	got, err := f.svc.Availability(t.Context(), 1, "2026-10-20", []int64{1})
	if err != nil {
		t.Fatal(err)
	}
	available := map[string]bool{}
	for _, slot := range got.Slots {
		available[slot.Start.In(saigon).Format("15:04")] = slot.Available
	}
	// 10:00: bay 12 is still free; 10:30: both taken; 11:00: bay 11 free again.
	if !available["10:00"] || available["10:30"] || !available["11:00"] {
		t.Errorf("available = %v", available)
	}
}

func TestAvailability_RejectsBadInput(t *testing.T) {
	f := newFixture(jane)
	cases := []struct {
		name       string
		hubID      int64
		date       string
		serviceIDs []int64
		want       error
	}{
		{"unknown hub", 9, "2026-10-20", []int64{1}, ports.ErrNotFound},
		{"no services", 1, "2026-10-20", nil, ports.ErrInvalidInput},
		{"unknown service", 1, "2026-10-20", []int64{99}, ports.ErrInvalidInput},
		{"bad date", 1, "20/10/2026", []int64{1}, ports.ErrInvalidInput},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := f.svc.Availability(t.Context(), tc.hubID, tc.date, tc.serviceIDs); !errors.Is(err, tc.want) {
				t.Errorf("err = %v, want %v", err, tc.want)
			}
		})
	}
}

// --- Book ------------------------------------------------------------------

func TestBook_ReservesBayAndWritesEvent(t *testing.T) {
	f := newFixture(jane)

	got, err := f.svc.Book(t.Context(), entity.Principal{Token: "t"}, ports.BookingRequest{
		HubID: 1, VehicleID: 7, ServiceIDs: []int64{1, 2, 1}, Start: at(10, 0),
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != 100 || got.CustomerID != 42 || got.BayID != 11 || got.Status != constants.AppointmentBooked {
		t.Errorf("appointment = %+v", got)
	}
	// The duplicate service id is booked once: 30 + 60 minutes, 45.00 + 60.50.
	if !got.End.Equal(at(11, 30)) || got.TotalAmount.StringFixed(2) != "105.50" || len(got.Services) != 2 {
		t.Errorf("end = %v, total = %s, services = %+v", got.End, got.TotalAmount, got.Services)
	}
	if len(f.appointments.LockedVehicles) != 1 || f.appointments.LockedVehicles[0] != 7 {
		t.Errorf("locked vehicles = %v, want [7]", f.appointments.LockedVehicles)
	}

	if len(f.outbox.Written) != 1 || f.outbox.Written[0].Topic != constants.TopicAppointmentBooked || f.outbox.Written[0].Key != "100" {
		t.Fatalf("outbox = %+v", f.outbox.Written)
	}
	var event struct {
		EventType   string         `json:"event_type"`
		Appointment map[string]any `json:"appointment"`
	}
	if err := json.Unmarshal(f.outbox.Written[0].Payload, &event); err != nil {
		t.Fatal(err)
	}
	a := event.Appointment
	if event.EventType != "AppointmentBooked" || a["dealership_id"] != float64(1) || a["vehicle_id"] != float64(7) ||
		a["total_amount"] != "105.50" || a["scheduled_start"] != "2026-10-20T03:00:00.000Z" {
		t.Errorf("payload = %s", f.outbox.Written[0].Payload)
	}
}

func TestBook_Rejections(t *testing.T) {
	cases := []struct {
		name      string
		customers MockCustomers
		setup     func(*MockAppointmentRepository)
		req       ports.BookingRequest
		want      error
	}{
		{
			name: "vehicle not owned",
			req:  ports.BookingRequest{HubID: 1, VehicleID: 8, ServiceIDs: []int64{1}, Start: at(10, 0)},
			want: ports.ErrInvalidInput,
		},
		{
			name: "in the past",
			req:  ports.BookingRequest{HubID: 1, VehicleID: 7, ServiceIDs: []int64{1}, Start: at(9, 0)},
			want: ports.ErrInvalidInput,
		},
		{
			name: "runs past closing",
			req:  ports.BookingRequest{HubID: 1, VehicleID: 7, ServiceIDs: []int64{2}, Start: at(11, 30)},
			want: ports.ErrInvalidInput,
		},
		{
			name:      "no customer profile",
			customers: MockCustomers{Err: ports.ErrNoCustomerProfile},
			req:       ports.BookingRequest{HubID: 1, VehicleID: 7, ServiceIDs: []int64{1}, Start: at(10, 0)},
			want:      ports.ErrNoCustomerProfile,
		},
		{
			name:  "vehicle already booked",
			setup: func(m *MockAppointmentRepository) { m.Busy = true },
			req:   ports.BookingRequest{HubID: 1, VehicleID: 7, ServiceIDs: []int64{1}, Start: at(10, 0)},
			want:  ports.ErrConflict,
		},
		{
			name:  "no bay free",
			setup: func(m *MockAppointmentRepository) { m.FreeBayErr = ports.ErrConflict },
			req:   ports.BookingRequest{HubID: 1, VehicleID: 7, ServiceIDs: []int64{1}, Start: at(10, 0)},
			want:  ports.ErrConflict,
		},
		{
			name:  "lost the race for the bay",
			setup: func(m *MockAppointmentRepository) { m.CreateErr = ports.ErrConflict },
			req:   ports.BookingRequest{HubID: 1, VehicleID: 7, ServiceIDs: []int64{1}, Start: at(10, 0)},
			want:  ports.ErrConflict,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			customers := jane
			if tc.customers != (MockCustomers{}) {
				customers = tc.customers
			}
			f := newFixture(customers)
			if tc.setup != nil {
				tc.setup(f.appointments)
			}

			_, err := f.svc.Book(t.Context(), entity.Principal{Token: "t"}, tc.req)
			if !errors.Is(err, tc.want) {
				t.Fatalf("err = %v, want %v", err, tc.want)
			}
			if len(f.outbox.Written) != 0 {
				t.Errorf("a rejected booking wrote %d outbox rows", len(f.outbox.Written))
			}
		})
	}
}

// --- Cancel ----------------------------------------------------------------

func TestCancel(t *testing.T) {
	booked := entity.Appointment{ID: 5, CustomerID: 42, HubID: 1, Start: at(11, 0), End: at(11, 30), Status: constants.AppointmentBooked}

	t.Run("own upcoming booking", func(t *testing.T) {
		f := newFixture(jane)
		f.appointments.Stored[5] = booked

		got, err := f.svc.Cancel(t.Context(), entity.Principal{}, 5)
		if err != nil {
			t.Fatal(err)
		}
		if got.Status != constants.AppointmentCancelled || len(f.appointments.Cancelled) != 1 {
			t.Errorf("got %+v, cancelled %v", got, f.appointments.Cancelled)
		}
		if len(f.outbox.Written) != 1 || f.outbox.Written[0].Topic != constants.TopicAppointmentCancelled {
			t.Errorf("outbox = %+v", f.outbox.Written)
		}
	})

	cases := []struct {
		name   string
		mutate func(*entity.Appointment)
		want   error
	}{
		{"someone else's", func(a *entity.Appointment) { a.CustomerID = 43 }, ports.ErrNotFound},
		{"already cancelled", func(a *entity.Appointment) { a.Status = constants.AppointmentCancelled }, ports.ErrConflict},
		{"already started", func(a *entity.Appointment) { a.Start = at(9, 0) }, ports.ErrConflict},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(jane)
			a := booked
			tc.mutate(&a)
			f.appointments.Stored[5] = a

			if _, err := f.svc.Cancel(t.Context(), entity.Principal{}, 5); !errors.Is(err, tc.want) {
				t.Fatalf("err = %v, want %v", err, tc.want)
			}
			if len(f.appointments.Cancelled) != 0 || len(f.outbox.Written) != 0 {
				t.Errorf("cancelled %v, outbox %d", f.appointments.Cancelled, len(f.outbox.Written))
			}
		})
	}
}
