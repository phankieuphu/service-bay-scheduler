package handler

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
	"vehicle-service/internal/adapters/http/dto"
	"vehicle-service/internal/constants"
	"vehicle-service/internal/domain/entity"
	"vehicle-service/internal/domain/ports"
	"vehicle-service/pkg/logger"

	"github.com/gin-gonic/gin"
)

type VehicleHandler struct {
	service ports.VehicleService
}

func NewVehicleHandler(service ports.VehicleService) *VehicleHandler {
	return &VehicleHandler{service: service}
}

func (h *VehicleHandler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/customers/:id/vehicles", h.GetCustomerVehicles)
	r.POST("/transfer", h.TransferVehicle)
	r.POST("/vehicle", h.RegisterVehicle)
	r.GET("/vehicle", h.ListVehicles)
	r.GET("/vehicle/:id", h.GetVehicle)
	r.PATCH("/vehicle/:id", h.UpdateVehicle)
	r.POST("/vehicle/:id/owner", h.AssignInitialOwner)
	r.GET("/vehicle/:id/warranty", h.GetWarranty)
	r.GET("/vehicle/:id/history", h.GetServiceHistory)
	r.GET("/vehicle/:id/materials", h.GetVehicleMaterials)
}

func (h *VehicleHandler) RegisterVehicle(c *gin.Context) {
	var req dto.RegisterVehicleDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	vehicle, err := h.service.RegisterVehicle(c.Request.Context(), entity.Vehicle{
		Vin:             req.Vin,
		LicensePlate:    req.LicensePlate,
		VehicleModelID:  req.VehicleModelID,
		WarrantyEndDate: req.WarrantyEndDate,
		Status:          req.Status,
	})
	if err != nil {
		writeError(c, err)
		return
	}

	c.Header("Location", fmt.Sprintf("%s/%d", c.FullPath(), vehicle.ID))
	c.JSON(http.StatusCreated, dto.NewVehicleResponseDTO(vehicle))
}

// GetCustomerVehicles lists the vehicles a customer currently owns.
func (h *VehicleHandler) GetCustomerVehicles(c *gin.Context) {
	customerID, ok := pathID(c, "id", "customer")
	if !ok {
		return
	}

	ownerships, err := h.service.GetCustomerVehicles(c.Request.Context(), customerID)
	if err != nil {
		writeError(c, err)
		return
	}

	vehicles := make([]dto.CustomerVehicleDTO, len(ownerships))
	for i, o := range ownerships {
		vehicles[i] = dto.CustomerVehicleDTO{
			VehicleResponseDTO: dto.NewVehicleResponseDTO(o.Vehicle),
			OwnedFrom:          o.OwnedFrom,
		}
	}
	c.JSON(http.StatusOK, dto.CustomerVehiclesResponseDTO{Vehicles: vehicles})
}

func (h *VehicleHandler) TransferVehicle(c *gin.Context) {
	var req dto.TransferVehicleDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	date := time.Now()
	if req.Date != nil {
		date = *req.Date
	}

	err := h.service.TransferVehicle(c.Request.Context(), entity.TransferVehicle{
		Date:      date,
		From:      req.From,
		To:        req.To,
		VehicleID: req.VehicleID,
	})
	if err != nil {
		writeError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// ListVehicles handles cursor-based pagination via the `cursor` and `limit`
// query params (same contract as customer-service's GET /customer), with
// optional exact-match `vin`, `plate` and `status` filters.
func (h *VehicleHandler) ListVehicles(c *gin.Context) {
	cursor, err := strconv.ParseInt(c.DefaultQuery("cursor", "0"), 10, 64)
	if err != nil || cursor < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid cursor"})
		return
	}
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "0"))
	if err != nil || limit < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid limit"})
		return
	}

	page, err := h.service.ListVehicles(c.Request.Context(), ports.ListVehiclesParams{
		Cursor:       cursor,
		Limit:        limit,
		Vin:          c.Query("vin"),
		LicensePlate: c.Query("plate"),
		Status:       constants.VehicleStatus(strings.ToUpper(c.Query("status"))),
	})
	if err != nil {
		writeError(c, err)
		return
	}

	vehicles := make([]dto.VehicleResponseDTO, len(page.Vehicles))
	for i, vehicle := range page.Vehicles {
		vehicles[i] = dto.NewVehicleResponseDTO(vehicle)
	}
	c.JSON(http.StatusOK, dto.ListVehiclesResponseDTO{
		Vehicles:   vehicles,
		NextCursor: page.NextCursor,
		HasMore:    page.HasMore,
	})
}

func (h *VehicleHandler) GetVehicle(c *gin.Context) {
	vehicleID, ok := pathID(c, "id", "vehicle")
	if !ok {
		return
	}

	vehicle, err := h.service.GetVehicle(c.Request.Context(), vehicleID)
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.NewVehicleResponseDTO(vehicle))
}

// UpdateVehicle returns the vehicle after the update, including its new
// updated_at for the client's next PATCH.
func (h *VehicleHandler) UpdateVehicle(c *gin.Context) {
	vehicleID, ok := pathID(c, "id", "vehicle")
	if !ok {
		return
	}
	var req dto.UpdateVehicleDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	vehicle, err := h.service.UpdateVehicle(c.Request.Context(), vehicleID, entity.VehicleUpdate{
		Status:          req.Status,
		WarrantyEndDate: req.WarrantyEndDate,
		UpdatedAt:       req.UpdatedAt,
	})
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.NewVehicleResponseDTO(vehicle))
}

// AssignInitialOwner is for a vehicle that has never had an owner (or whose
// last owner was removed); it returns 409 if the vehicle already has one.
func (h *VehicleHandler) AssignInitialOwner(c *gin.Context) {
	vehicleID, ok := pathID(c, "id", "vehicle")
	if !ok {
		return
	}
	var req dto.AssignOwnerDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	date := time.Now()
	if req.Date != nil {
		date = *req.Date
	}
	if err := h.service.AssignInitialOwner(c.Request.Context(), vehicleID, req.CustomerID, date); err != nil {
		writeError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// GetWarranty reports warranty status for today, or for `date`
// (YYYY-MM-DD) — billing asks about the day the service was done.
func (h *VehicleHandler) GetWarranty(c *gin.Context) {
	vehicleID, ok := pathID(c, "id", "vehicle")
	if !ok {
		return
	}
	asOf := time.Now()
	if date := c.Query("date"); date != "" {
		parsed, err := time.Parse(time.DateOnly, date)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "date must be YYYY-MM-DD"})
			return
		}
		asOf = parsed
	}

	warranty, err := h.service.GetWarranty(c.Request.Context(), vehicleID, asOf)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.NewWarrantyResponseDTO(warranty))
}

// GetServiceHistory lists completed appointments for the vehicle, most
// recent first; `limit` defaults to 50 (max 200).
func (h *VehicleHandler) GetServiceHistory(c *gin.Context) {
	vehicleID, ok := pathID(c, "id", "vehicle")
	if !ok {
		return
	}
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "0"))
	if err != nil || limit < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid limit"})
		return
	}

	entries, err := h.service.GetServiceHistory(c.Request.Context(), vehicleID, limit)
	if err != nil {
		writeError(c, err)
		return
	}
	history := make([]dto.ServiceHistoryEntryDTO, len(entries))
	for i, e := range entries {
		history[i] = dto.ServiceHistoryEntryDTO{
			ID:            e.ID,
			AppointmentID: e.AppointmentID,
			DealershipID:  e.DealershipID,
			CompletedAt:   e.CompletedAt,
			Services:      e.Services,
		}
	}
	c.JSON(http.StatusOK, dto.ServiceHistoryResponseDTO{History: history})
}

// GetVehicleMaterials lists parts installed on the vehicle, most recent first.
func (h *VehicleHandler) GetVehicleMaterials(c *gin.Context) {
	vehicleID, ok := pathID(c, "id", "vehicle")
	if !ok {
		return
	}

	materials, err := h.service.GetVehicleMaterials(c.Request.Context(), vehicleID)
	if err != nil {
		writeError(c, err)
		return
	}
	items := make([]dto.VehicleMaterialDTO, len(materials))
	for i, m := range materials {
		items[i] = dto.VehicleMaterialDTO{
			ID:          m.ID,
			MaterialID:  m.MaterialID,
			Description: m.Description,
			Count:       m.Count,
			InstalledAt: m.InstalledAt,
		}
	}
	c.JSON(http.StatusOK, dto.VehicleMaterialsResponseDTO{Materials: items})
}

// writeError maps the sentinel errors in ports/errors.go to status codes.
// Their messages are written for the client (e.g. "vehicle: conflict: a
// vehicle with this VIN is already registered"). Anything else is an
// internal failure: logged, and replaced by a generic message so driver
// details don't leak.
func writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ports.ErrInvalidInput):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, ports.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, ports.ErrConflict), errors.Is(err, ports.ErrInvalidState):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	default:
		logger.ErrorContext(c.Request.Context(), "request failed", "path", c.FullPath(), "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	}
}

// pathID parses the :id-style path parameter name as a positive int64,
// writing a 400 and returning false if it isn't one.
func pathID(c *gin.Context, name, label string) (int64, bool) {
	id, err := strconv.ParseInt(c.Param(name), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid " + label + " id"})
		return 0, false
	}
	return id, true
}
