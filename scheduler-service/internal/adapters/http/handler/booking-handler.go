package handler

import (
	"errors"
	"fmt"
	"net/http"
	"scheduler-service/internal/adapters/http/dto"
	"scheduler-service/internal/adapters/http/middleware"
	"scheduler-service/internal/domain/entity"
	"scheduler-service/internal/domain/ports"
	"scheduler-service/pkg/times"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type BookingHandler struct {
	service ports.BookingService
}

func NewBookingHandler(service ports.BookingService) *BookingHandler {
	return &BookingHandler{service: service}
}

// RegisterRoutes expects r to already require authentication. Hubs, the
// service catalog and availability are open to any signed-in user; booking
// and a customer's own appointments are for CUSTOMER accounts, since an
// appointment belongs to the caller's linked customer profile. A hub's day
// and completing an appointment are staff work.
func (h *BookingHandler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/hubs", h.ListHubs)
	r.GET("/hubs/:id/availability", h.Availability)
	r.GET("/services", h.ListServices)

	customer := r.Group("", middleware.RequireRole(entity.RoleCustomer))
	customer.POST("/appointments", h.Book)
	customer.GET("/me/appointments", h.MyAppointments)
	customer.POST("/appointments/:id/cancel", h.Cancel)

	staff := r.Group("", middleware.RequireRole(entity.StaffRoles...))
	staff.GET("/hubs/:id/appointments", h.HubDay)
	staff.POST("/appointments/:id/complete", h.Complete)
}

// HubDay answers GET /hubs/:id/appointments?date=YYYY-MM-DD.
func (h *BookingHandler) HubDay(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	date := c.Query("date")
	if date == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "date is required (YYYY-MM-DD)"})
		return
	}

	day, err := h.service.HubDay(c.Request.Context(), id, date)
	if err != nil {
		writeError(c, err)
		return
	}
	out := make([]dto.AppointmentDTO, len(day.Appointments))
	for i, a := range day.Appointments {
		out[i] = toAppointmentDTO(a)
	}
	c.JSON(http.StatusOK, dto.HubDayResponseDTO{HubID: day.HubID, Date: day.Date, TimeZone: day.TimeZone, Appointments: out})
}

// Complete answers POST /appointments/:id/complete: 200 with the completed
// appointment, 409 if it was cancelled or already completed.
func (h *BookingHandler) Complete(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}

	appointment, err := h.service.Complete(c.Request.Context(), id)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toAppointmentDTO(appointment))
}

func (h *BookingHandler) ListHubs(c *gin.Context) {
	hubs, err := h.service.ListHubs(c.Request.Context())
	if err != nil {
		writeError(c, err)
		return
	}

	out := make([]dto.HubDTO, len(hubs))
	for i, hub := range hubs {
		out[i] = dto.HubDTO{
			ID:        hub.ID,
			Name:      hub.Name,
			Address:   hub.Address,
			OpenHour:  clock(hub.OpenHour),
			CloseHour: clock(hub.CloseHour),
			TimeZone:  hub.TimeZone,
			BayCount:  hub.BayCount,
		}
	}
	c.JSON(http.StatusOK, dto.HubsResponseDTO{Hubs: out})
}

func (h *BookingHandler) ListServices(c *gin.Context) {
	services, err := h.service.ListServices(c.Request.Context())
	if err != nil {
		writeError(c, err)
		return
	}

	out := make([]dto.ServiceDTO, len(services))
	for i, svc := range services {
		out[i] = dto.ServiceDTO{
			ID:              svc.ID,
			Name:            svc.Name,
			ServiceType:     svc.ServiceType,
			Fee:             svc.Fee.StringFixed(2),
			DurationMinutes: svc.DurationMinutes,
		}
	}
	c.JSON(http.StatusOK, dto.ServicesResponseDTO{Services: out})
}

// Availability takes `date` (YYYY-MM-DD, in the hub's time zone) and one
// or more `service_id` query params: ?date=2026-10-20&service_id=1&service_id=3.
func (h *BookingHandler) Availability(c *gin.Context) {
	hubID, ok := pathID(c)
	if !ok {
		return
	}
	var serviceIDs []int64
	for _, raw := range c.QueryArray("service_id") {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || id <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid service_id"})
			return
		}
		serviceIDs = append(serviceIDs, id)
	}

	availability, err := h.service.Availability(c.Request.Context(), hubID, c.Query("date"), serviceIDs)
	if err != nil {
		writeError(c, err)
		return
	}

	slots := make([]dto.SlotDTO, len(availability.Slots))
	for i, slot := range availability.Slots {
		slots[i] = dto.SlotDTO{Start: times.NewTime(slot.Start), End: times.NewTime(slot.End), Available: slot.Available}
	}
	c.JSON(http.StatusOK, dto.AvailabilityResponseDTO{
		HubID:           availability.HubID,
		Date:            availability.Date,
		TimeZone:        availability.TimeZone,
		DurationMinutes: availability.DurationMinutes,
		Slots:           slots,
	})
}

func (h *BookingHandler) Book(c *gin.Context) {
	var req dto.CreateAppointmentDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	appointment, err := h.service.Book(c.Request.Context(), principal(c), ports.BookingRequest{
		HubID:      req.HubID,
		VehicleID:  req.VehicleID,
		ServiceIDs: req.ServiceIDs,
		Start:      req.Start.Time,
	})
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, toAppointmentDTO(appointment))
}

func (h *BookingHandler) MyAppointments(c *gin.Context) {
	appointments, err := h.service.MyAppointments(c.Request.Context(), principal(c))
	if err != nil {
		writeError(c, err)
		return
	}

	out := make([]dto.AppointmentDTO, len(appointments))
	for i, a := range appointments {
		out[i] = toAppointmentDTO(a)
	}
	c.JSON(http.StatusOK, dto.AppointmentsResponseDTO{Appointments: out})
}

func (h *BookingHandler) Cancel(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}

	appointment, err := h.service.Cancel(c.Request.Context(), principal(c), id)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toAppointmentDTO(appointment))
}

// writeError maps the service's sentinel errors to status codes. The
// error text is the sentinel's message plus the detail the service added.
func writeError(c *gin.Context, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, ports.ErrInvalidInput):
		status = http.StatusBadRequest
	case errors.Is(err, ports.ErrNotFound), errors.Is(err, ports.ErrNoCustomerProfile):
		status = http.StatusNotFound
	case errors.Is(err, ports.ErrConflict):
		status = http.StatusConflict
	case errors.Is(err, ports.ErrUpstream):
		status = http.StatusBadGateway
	}
	c.JSON(status, gin.H{"error": err.Error()})
}

func principal(c *gin.Context) entity.Principal {
	p, _ := c.Value(middleware.PrincipalKey).(entity.Principal)
	return p
}

func pathID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return 0, false
	}
	return id, true
}

// clock formats an offset from midnight as "HH:MM".
func clock(offset time.Duration) string {
	return fmt.Sprintf("%02d:%02d", int(offset/time.Hour), int(offset%time.Hour/time.Minute))
}

func toAppointmentDTO(a entity.Appointment) dto.AppointmentDTO {
	services := make([]dto.AppointmentServiceDTO, len(a.Services))
	for i, s := range a.Services {
		services[i] = dto.AppointmentServiceDTO{ServiceID: s.ServiceID, Name: s.Name, Fee: s.Fee.StringFixed(2)}
	}
	return dto.AppointmentDTO{
		ID:          a.ID,
		HubID:       a.HubID,
		CustomerID:  a.CustomerID,
		VehicleID:   a.VehicleID,
		BayID:       a.BayID,
		Start:       times.NewTime(a.Start),
		End:         times.NewTime(a.End),
		Status:      a.Status,
		TotalAmount: a.TotalAmount.StringFixed(2),
		Services:    services,
		CreatedAt:   times.NewTime(a.CreatedAt),
		UpdatedAt:   times.NewTime(a.UpdatedAt),
	}
}
