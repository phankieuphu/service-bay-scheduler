package handler

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"
	"vehicle-service/internal/adapters/http/dto"
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
	r.GET("/vehicle/:id", h.GetVehicle)
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
