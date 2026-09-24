package handler

import (
	"customer-service/internal/adapters/http/dto"
	"customer-service/internal/domain/entity"
	"customer-service/internal/domain/ports"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type CustomerHandler struct {
	service ports.CustomerService
}

func NewCustomerHandler(service ports.CustomerService) *CustomerHandler {
	return &CustomerHandler{service: service}
}

func (h *CustomerHandler) RegisterRoutes(r *gin.RouterGroup) {
	r.POST("/customer", h.CreateCustomer)
	r.GET("/customer", h.ListCustomers)
	r.GET("/customer/:id", h.GetCustomer)
	r.PUT("/customer/:id", h.UpdateCustomerProfile)
	r.DELETE("/customer/:id", h.DeleteProfile)
}

func (h *CustomerHandler) CreateCustomer(c *gin.Context) {
	var req dto.CreateCustomerDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	customer := entity.Customer{
		Name:     req.Name,
		Email:    req.Email,
		Phone:    req.Phone,
		BirthDay: req.BirthDay,
	}

	created, err := h.service.CreateCustomer(c.Request.Context(), customer)
	if err != nil {
		if errors.Is(err, ports.ErrConflict) {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, toCustomerDTO(created))
}

func (h *CustomerHandler) GetCustomer(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid customer id"})
		return
	}

	customer, err := h.service.GetCustomer(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ports.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, toCustomerDTO(customer))
}

func (h *CustomerHandler) UpdateCustomerProfile(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalided customer id"})
		return
	}
	var req dto.UpdateProfileDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
	}
	updateCustomer := entity.Customer{
		Name:     req.Name,
		BirthDay: req.BirthDay,
	}
	err = h.service.UpdateProfile(c, id, updateCustomer)
	if err != nil {
		if errors.Is(err, ports.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusNoContent, gin.H{"messages": "updated success"})

}

// ListCustomers handles cursor-based pagination via the `cursor` and `limit`
// query params. `cursor` is the id of the last customer the caller has
// already seen (omit/0 for the first page); the response's `next_cursor`
// is fed back as `cursor` to fetch the next page.
func (h *CustomerHandler) ListCustomers(c *gin.Context) {
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

	page, err := h.service.GetCustomers(c.Request.Context(), ports.ListCustomersParams{
		Cursor: cursor,
		Limit:  limit,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	customers := make([]dto.CustomerDTO, len(page.Customers))
	for i, customer := range page.Customers {
		customers[i] = toCustomerDTO(customer)
	}

	c.JSON(http.StatusOK, dto.ListCustomersResponseDTO{
		Customers:  customers,
		NextCursor: page.NextCursor,
		HasMore:    page.HasMore,
	})
}

// soft delete customer profile
func (h *CustomerHandler) DeleteProfile(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid customer id"})
		return
	}

	err = h.service.DeleteProfile(c, id)
	if err != nil {
		if errors.Is(err, ports.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
	c.JSON(http.StatusNoContent, gin.H{"message": "ok"})
}

func toCustomerDTO(customer entity.Customer) dto.CustomerDTO {
	return dto.CustomerDTO{
		ID:        customer.ID,
		Name:      customer.Name,
		Email:     customer.Email,
		Phone:     customer.Phone,
		BirthDay:  customer.BirthDay,
		Status:    customer.Status,
		CreatedAt: customer.CreatedAt,
		UpdatedAt: customer.UpdatedAt,
	}
}
