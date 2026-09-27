package ports

import (
	"context"
	"identity-service/internal/domain/entity"
)

type AuthService interface {
	Register(ctx context.Context, register entity.Register) (entity.User, error)
	Login(ctx context.Context, email, password string) (entity.TokenPair, error)
	// Refresh consumes refreshToken (it can't be used again) and issues a
	// new pair.
	Refresh(ctx context.Context, refreshToken string) (entity.TokenPair, error)
	// Logout revokes refreshToken. Revoking an unknown token is not an error.
	Logout(ctx context.Context, refreshToken string) error
	Me(ctx context.Context, userID string) (entity.User, error)
}
