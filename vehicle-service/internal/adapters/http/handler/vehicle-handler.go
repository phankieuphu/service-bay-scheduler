package handler

import (
	"errors"
	"net/http"
	"time"
	"vehicle-service/internal/adapters/http/dto"
	"vehicle-service/internal/domain/entity"
	"vehicle-service/internal/domain/ports"

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
	r.GET("/vehicle/:id", h.GetVehicle)
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
		switch {
		case errors.Is(err, ports.ErrNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		case errors.Is(err, ports.ErrConflict):
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *VehicleHandler) GetVehicle(c *gin.Context) {}
