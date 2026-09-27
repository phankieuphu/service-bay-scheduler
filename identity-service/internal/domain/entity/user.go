package entity

import (
	"identity-service/internal/constants"
	"time"
)

// User is a login identity. ID is a UUID minted here, so it stays
// unambiguous when other services store it as a reference
// (architecture-design.md §5a).
type User struct {
	ID           string
	Email        string
	PasswordHash string
	Role         constants.Role
	Status       constants.UserStatus
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Register is the input to AuthService.Register.
type Register struct {
	Email    string
	Password string
	Role     constants.Role
}

// TokenPair is what a successful login or refresh returns.
type TokenPair struct {
	AccessToken      string
	AccessExpiresAt  time.Time
	RefreshToken     string
	RefreshExpiresAt time.Time
}

// AccessClaims is what a verified access token says about its bearer.
type AccessClaims struct {
	UserID string
	Role   constants.Role
}
