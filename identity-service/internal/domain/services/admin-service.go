package services

import (
	"context"
	"errors"
	"fmt"
	"identity-service/config"
	"identity-service/internal/constants"
	"identity-service/internal/domain/entity"
	"identity-service/internal/domain/ports"
	"identity-service/pkg/logger"
	"time"
)

type AdminService struct {
	deps userDeps
}

func NewAdminService(
	cfg config.Config,
	users ports.UserRepository,
	outbox ports.OutboxRepository,
	txManager ports.TxManager,
	hasher ports.PasswordHasher,
) ports.AdminService {
	return &AdminService{deps: userDeps{
		config:    cfg,
		users:     users,
		outbox:    outbox,
		txManager: txManager,
		hasher:    hasher,
		now:       time.Now,
	}}
}

// CreateUser implements [ports.AdminService].
func (a *AdminService) CreateUser(ctx context.Context, actorID string, register entity.Register) (entity.User, error) {
	if !register.Role.Valid() {
		return entity.User{}, fmt.Errorf("%w: unknown role %q", ports.ErrInvalidInput, register.Role)
	}
	if err := a.requireAdmin(ctx, actorID); err != nil {
		return entity.User{}, err
	}

	user, err := createUser(ctx, a.deps, register)
	if err != nil {
		if !errors.Is(err, ports.ErrConflict) {
			logger.ErrorContext(ctx, "failed to create user", "actor", actorID, "error", err)
		}
		return entity.User{}, err
	}
	logger.InfoContext(ctx, "admin created user", "actor", actorID, "user", user.ID, "role", user.Role)
	return user, nil
}

// SetUserStatus implements [ports.AdminService]. The target row is locked
// while its status is read and changed, so two concurrent changes can't
// both emit an event for the same "previous" state.
func (a *AdminService) SetUserStatus(ctx context.Context, actorID, userID string, status constants.UserStatus) (entity.User, error) {
	if !status.Valid() {
		return entity.User{}, fmt.Errorf("%w: unknown status %q", ports.ErrInvalidInput, status)
	}
	if actorID == userID {
		return entity.User{}, fmt.Errorf("%w: you can't change your own status", ports.ErrInvalidInput)
	}
	if err := a.requireAdmin(ctx, actorID); err != nil {
		return entity.User{}, err
	}

	var updated entity.User
	err := a.deps.txManager.RunInTx(ctx, func(ctx context.Context) error {
		current, err := a.deps.users.GetByIDForUpdate(ctx, userID)
		if err != nil {
			return err
		}
		if current.Status == status {
			updated = current
			return nil
		}

		now := a.deps.now().UTC()
		if updated, err = a.deps.users.UpdateStatus(ctx, userID, status, now); err != nil {
			return err
		}
		return writeUserEvent(ctx, a.deps, constants.UserUpdated, updated, now)
	})
	if err != nil {
		if !errors.Is(err, ports.ErrNotFound) {
			logger.ErrorContext(ctx, "failed to set user status", "actor", actorID, "user", userID, "error", err)
		}
		return entity.User{}, err
	}
	logger.InfoContext(ctx, "admin set user status", "actor", actorID, "user", userID, "status", status)
	return updated, nil
}

// BootstrapAdmin implements [ports.AdminService]. Replicas starting
// together may both see no admin; the loser's insert fails on the unique
// email and it reports false. An existing non-admin account with email is
// never promoted: whoever registered it would become admin.
func (a *AdminService) BootstrapAdmin(ctx context.Context, email, password string) (bool, error) {
	exists, err := a.deps.users.ExistsWithRole(ctx, constants.RoleAdmin)
	if err != nil || exists {
		return false, err
	}

	_, err = createUser(ctx, a.deps, entity.Register{Email: email, Password: password, Role: constants.RoleAdmin})
	if errors.Is(err, ports.ErrConflict) {
		if existing, getErr := a.deps.users.GetByEmail(ctx, normalizeEmail(email)); getErr == nil && existing.Role == constants.RoleAdmin {
			return false, nil // another replica just created it
		}
		return false, fmt.Errorf("an account with ADMIN_EMAIL already exists and isn't an admin; it won't be promoted: %w", err)
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// requireAdmin re-reads the actor rather than trusting the token's role
// claim, which stays valid for up to JWT.AccessTTL after a demotion or lock.
func (a *AdminService) requireAdmin(ctx context.Context, actorID string) error {
	actor, err := a.deps.users.GetByID(ctx, actorID)
	if errors.Is(err, ports.ErrNotFound) {
		return ports.ErrForbidden
	}
	if err != nil {
		return err
	}
	if actor.Role != constants.RoleAdmin || actor.Status != constants.UserActive {
		return ports.ErrForbidden
	}
	return nil
}
