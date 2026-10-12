package ports

import (
	"context"
	"identity-service/internal/constants"
	"identity-service/internal/domain/entity"
	"time"
)

type UserRepository interface {
	// Create returns ErrConflict if the email is already taken.
	Create(ctx context.Context, user entity.User) (entity.User, error)
	GetByID(ctx context.Context, id string) (entity.User, error)
	// GetByEmail matches case-insensitively; emails are stored lowercased.
	GetByEmail(ctx context.Context, email string) (entity.User, error)
	// GetByIDForUpdate is GetByID with a row lock (SELECT ... FOR UPDATE)
	// held until the caller's transaction ends.
	GetByIDForUpdate(ctx context.Context, id string) (entity.User, error)
	// UpdateStatus sets status and updated_at, returning the updated user.
	UpdateStatus(ctx context.Context, id string, status constants.UserStatus, updatedAt time.Time) (entity.User, error)
	// ExistsWithRole reports whether any user has role.
	ExistsWithRole(ctx context.Context, role constants.Role) (bool, error)
}
