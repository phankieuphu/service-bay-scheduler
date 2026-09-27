package ports

import (
	"context"
	"identity-service/internal/domain/entity"
)

type UserRepository interface {
	// Create returns ErrConflict if the email is already taken.
	Create(ctx context.Context, user entity.User) (entity.User, error)
	GetByID(ctx context.Context, id string) (entity.User, error)
	// GetByEmail matches case-insensitively; emails are stored lowercased.
	GetByEmail(ctx context.Context, email string) (entity.User, error)
}
