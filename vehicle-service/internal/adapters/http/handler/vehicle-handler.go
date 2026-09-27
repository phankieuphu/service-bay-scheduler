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
	r.GET("/customer-vehicle", h.GetCustomerVehicle)
	r.POST("/transfer", h.TransferVehicle)
	r.POST("/vehicle", h.RegisterVehicle)
	r.GET("/vehicle", h.ListVehicles)
	r.GET("/vehicle/:id", h.GetVehicle)
	r.PATCH("/vehicle/:id", h.UpdateVehicle)
	r.POST("/vehicle/:id/owner", h.AssignInitialOwner)
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

func (h *VehicleHandler) GetCustomerVehicle(c *gin.Context) {}

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
