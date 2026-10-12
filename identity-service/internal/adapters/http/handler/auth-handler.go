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
	service  ports.AuthService
	tokens   ports.TokenIssuer
	throttle *Throttle // nil disables rate limiting
	cookies  RefreshCookies
}

func NewAuthHandler(service ports.AuthService, tokens ports.TokenIssuer, throttle *Throttle, cookies RefreshCookies) *AuthHandler {
	return &AuthHandler{service: service, tokens: tokens, throttle: throttle, cookies: cookies}
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

	// Every attempt counts, not just failures: what's being limited is
	// mass account creation, which is a stream of successes.
	if t := h.throttle; t != nil {
		limits := t.registerLimits(c.ClientIP())
		if wait, blocked := t.blocked(c.Request.Context(), limits); blocked {
			tooManyRequests(c, wait)
			return
		}
		t.add(c.Request.Context(), limits, t.cfg.RegisterWindow)
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

	// Checked before the password, so a locked-out email costs no bcrypt
	// work. The 429 is the same whether or not the email has an account.
	var limits []limit
	if t := h.throttle; t != nil {
		limits = t.loginLimits(c.ClientIP(), req.Email)
		if wait, blocked := t.blocked(c.Request.Context(), limits); blocked {
			tooManyRequests(c, wait)
			return
		}
	}

	pair, err := h.service.Login(c.Request.Context(), req.Email, req.Password)
	if t := h.throttle; t != nil {
		switch {
		case errors.Is(err, ports.ErrInvalidCredentials):
			t.add(c.Request.Context(), limits, t.cfg.LoginWindow)
		case err == nil:
			// Only the email's count: resetting the IP's would let one valid
			// account clear the way for guessing at others.
			t.reset(c.Request.Context(), loginEmailKey(req.Email))
		}
	}
	if err != nil {
		writeError(c, err)
		return
	}

	h.cookies.set(c, pair.RefreshToken, pair.RefreshExpiresAt)
	c.JSON(http.StatusOK, dto.NewTokenResponse(pair, time.Now()))
}

// Refresh reads the refresh token from its cookie (never the body) and
// sets the rotated one the same way.
func (h *AuthHandler) Refresh(c *gin.Context) {
	token, ok := h.cookies.read(c, true)
	if !ok {
		return
	}

	pair, err := h.service.Refresh(c.Request.Context(), token)
	if err != nil {
		// The token is spent or invalid either way; don't keep sending it.
		if errors.Is(err, ports.ErrInvalidToken) || errors.Is(err, ports.ErrUserInactive) {
			h.cookies.clear(c)
		}
		writeError(c, err)
		return
	}

	h.cookies.set(c, pair.RefreshToken, pair.RefreshExpiresAt)
	c.JSON(http.StatusOK, dto.NewTokenResponse(pair, time.Now()))
}

// Logout revokes the cookie's refresh token, if any, and clears the cookie.
func (h *AuthHandler) Logout(c *gin.Context) {
	token, ok := h.cookies.read(c, false)
	if !ok {
		return
	}

	if token != "" {
		if err := h.service.Logout(c.Request.Context(), token); err != nil {
			writeError(c, err)
			return
		}
	}

	h.cookies.clear(c)
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
	case errors.Is(err, ports.ErrUserInactive), errors.Is(err, ports.ErrForbidden):
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
	case errors.Is(err, ports.ErrInvalidInput):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, ports.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	default:
		logger.ErrorContext(c.Request.Context(), "request failed", "path", c.FullPath(), "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	}
}
