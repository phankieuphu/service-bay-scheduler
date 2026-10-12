package handler

import (
	"identity-service/internal/adapters/http/dto"
	"identity-service/internal/adapters/http/middleware"
	"identity-service/internal/constants"
	"identity-service/internal/domain/entity"
	"identity-service/internal/domain/ports"
	"net/http"

	"github.com/gin-gonic/gin"
)

// AdminHandler serves /admin/*, which only ADMIN tokens may call.
type AdminHandler struct {
	service ports.AdminService
	tokens  ports.TokenIssuer
}

func NewAdminHandler(service ports.AdminService, tokens ports.TokenIssuer) *AdminHandler {
	return &AdminHandler{service: service, tokens: tokens}
}

func (h *AdminHandler) RegisterRoutes(r *gin.RouterGroup) {
	admin := r.Group("/admin", middleware.RequireAuth(h.tokens), middleware.RequireRole(constants.RoleAdmin))
	admin.POST("/users", h.CreateUser)
	admin.PATCH("/users/:id", h.UpdateUserStatus)
}

func (h *AdminHandler) CreateUser(c *gin.Context) {
	var req dto.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.service.CreateUser(c.Request.Context(), c.GetString(middleware.UserIDKey), entity.Register{
		Email:    req.Email,
		Password: req.Password,
		Role:     constants.Role(req.Role),
	})
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusCreated, dto.NewUserResponse(user))
}

func (h *AdminHandler) UpdateUserStatus(c *gin.Context) {
	var req dto.UpdateUserStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.service.SetUserStatus(c.Request.Context(), c.GetString(middleware.UserIDKey), c.Param("id"), constants.UserStatus(req.Status))
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.NewUserResponse(user))
}
