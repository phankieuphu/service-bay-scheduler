package ports

import (
	"context"
	"identity-service/internal/constants"
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

// AdminService provisions and manages users on behalf of an ADMIN. actorID
// is the caller's user id: every method re-reads it and requires an ACTIVE
// ADMIN, so a demoted or locked admin is refused even while their access
// token is still valid.
type AdminService interface {
	// CreateUser creates a user with any role and emits UserCreated, like
	// Register does.
	CreateUser(ctx context.Context, actorID string, register entity.Register) (entity.User, error)
	// SetUserStatus changes a user's status and emits UserUpdated. An admin
	// can't change their own status (so the last admin can't lock
	// themselves out).
	SetUserStatus(ctx context.Context, actorID, userID string, status constants.UserStatus) (entity.User, error)
	// BootstrapAdmin creates an ADMIN with email and password if no ADMIN
	// exists yet. It reports whether it created one.
	BootstrapAdmin(ctx context.Context, email, password string) (bool, error)
}
