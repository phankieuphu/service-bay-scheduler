package dto

import (
	"identity-service/internal/domain/entity"
	"time"
)

type RegisterRequest struct {
	Email string `json:"email" binding:"required,email,max=255"`
	// max=72 is bcrypt's input limit; anything past it would be silently
	// ignored when hashing.
	Password string `json:"password" binding:"required,min=8,max=72"`
	// Role defaults to CUSTOMER when omitted; any other role is refused
	// with 403 (staff are provisioned through POST /admin/users).
	Role string `json:"role" binding:"omitempty,oneof=CUSTOMER TECHNICIAN MANAGER ADMIN"`
}

// CreateUserRequest is the body of POST /admin/users. Unlike registration,
// role is required and may be any role.
type CreateUserRequest struct {
	Email    string `json:"email" binding:"required,email,max=255"`
	Password string `json:"password" binding:"required,min=8,max=72"`
	Role     string `json:"role" binding:"required,oneof=CUSTOMER TECHNICIAN MANAGER ADMIN"`
}

// UpdateUserStatusRequest is the body of PATCH /admin/users/:id.
type UpdateUserStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=ACTIVE INACTIVE LOCKED"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// TokenResponse carries only the access token. The refresh token goes in
// an HttpOnly cookie (see handler.RefreshCookies), out of scripts' reach.
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"` // seconds until AccessToken expires
}

func NewTokenResponse(pair entity.TokenPair, now time.Time) TokenResponse {
	return TokenResponse{
		AccessToken: pair.AccessToken,
		TokenType:   "Bearer",
		ExpiresIn:   int64(pair.AccessExpiresAt.Sub(now).Seconds()),
	}
}

type UserResponse struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func NewUserResponse(user entity.User) UserResponse {
	return UserResponse{
		ID:        user.ID,
		Email:     user.Email,
		Role:      string(user.Role),
		Status:    string(user.Status),
		CreatedAt: user.CreatedAt.UTC(),
		UpdatedAt: user.UpdatedAt.UTC(),
	}
}
