package handler

import (
	"customer-service/internal/domain/entity"
	"customer-service/internal/domain/ports"
	"net/http"

	"github.com/gin-gonic/gin"
)

type CustomerHandler struct {
	service ports.CustomerService
}

func NewCustomerHandler(service ports.CustomerService) *CustomerHandler {
	return &CustomerHandler{service: service}
}

func (h *CustomerHandler) RegisterRoutes(r *gin.RouterGroup) {
	r.POST("/accounts", h.CreateCustomer)
}

func (h *CustomerHandler) CreateCustomer(c *gin.Context) {
	var acc entity.Customer
	if err := c.ShouldBindJSON(&acc); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.Save(c.Request.Context(), acc); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, acc)
}
