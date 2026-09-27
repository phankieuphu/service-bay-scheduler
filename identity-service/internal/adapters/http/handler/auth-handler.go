package handler

import (
	"errors"
	"identity-service/internal/adapters/http/dto"
	"identity-service/internal/adapters/http/middleware"
	"identity-service/internal/constants"
	"identity-service/internal/domain/entity"
	"identity-service/internal/domain/ports"
	"identity-service/pkg/logger"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	service ports.AuthService
	tokens  ports.TokenIssuer
}

func NewAuthHandler(service ports.AuthService, tokens ports.TokenIssuer) *AuthHandler {
	return &AuthHandler{service: service, tokens: tokens}
}

func (h *AuthHandler) RegisterRoutes(r *gin.RouterGroup) {
	auth := r.Group("/auth")
	auth.POST("/register", h.Register)
	auth.POST("/login", h.Login)
	auth.POST("/refresh", h.Refresh)
	auth.POST("/logout", h.Logout)

	r.GET("/me", middleware.RequireAuth(h.tokens), h.Me)
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.service.Register(c.Request.Context(), entity.Register{
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

func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	pair, err := h.service.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.NewTokenResponse(pair, time.Now()))
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	var req dto.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	pair, err := h.service.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.NewTokenResponse(pair, time.Now()))
}

func (h *AuthHandler) Logout(c *gin.Context) {
	var req dto.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.Logout(c.Request.Context(), req.RefreshToken); err != nil {
		writeError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *AuthHandler) Me(c *gin.Context) {
	user, err := h.service.Me(c.Request.Context(), c.GetString(middleware.UserIDKey))
	if err != nil {
		// A valid token for a user that no longer exists: the token, not
		// the route, is what's wrong.
		if errors.Is(err, ports.ErrNotFound) {
			err = ports.ErrInvalidToken
		}
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.NewUserResponse(user))
}

// writeError maps the sentinel errors in ports/errors.go to status codes.
// Anything else is an internal failure: it's logged, and the client gets a
// generic message rather than driver/infra details.
func writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ports.ErrConflict):
		c.JSON(http.StatusConflict, gin.H{"error": "an account with this email already exists"})
	case errors.Is(err, ports.ErrRoleNotAllowed):
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
	case errors.Is(err, ports.ErrInvalidCredentials), errors.Is(err, ports.ErrInvalidToken):
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
	case errors.Is(err, ports.ErrUserInactive):
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
	case errors.Is(err, ports.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	default:
		logger.ErrorContext(c.Request.Context(), "request failed", "path", c.FullPath(), "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	}
}
