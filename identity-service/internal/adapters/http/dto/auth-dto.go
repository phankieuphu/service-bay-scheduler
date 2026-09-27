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
	// Role defaults to CUSTOMER when omitted.
	Role string `json:"role" binding:"omitempty,oneof=CUSTOMER TECHNICIAN MANAGER ADMIN"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"` // seconds until AccessToken expires
	RefreshToken string `json:"refresh_token"`
}

func NewTokenResponse(pair entity.TokenPair, now time.Time) TokenResponse {
	return TokenResponse{
		AccessToken:  pair.AccessToken,
		TokenType:    "Bearer",
		ExpiresIn:    int64(pair.AccessExpiresAt.Sub(now).Seconds()),
		RefreshToken: pair.RefreshToken,
	}
}

type UserResponse struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

func NewUserResponse(user entity.User) UserResponse {
	return UserResponse{
		ID:        user.ID,
		Email:     user.Email,
		Role:      string(user.Role),
		Status:    string(user.Status),
		CreatedAt: user.CreatedAt.UTC(),
	}
}
