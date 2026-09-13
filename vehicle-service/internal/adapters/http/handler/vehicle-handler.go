package handler

import (
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
func (h *VehicleHandler) TransferVehicle(c *gin.Context)    {}
func (h *VehicleHandler) GetVehicle(c *gin.Context)         {}
