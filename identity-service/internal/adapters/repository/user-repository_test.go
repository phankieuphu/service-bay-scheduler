package repository

import (
	"context"
	"errors"
	"identity-service/internal/constants"
	"identity-service/internal/domain/entity"
	"identity-service/internal/domain/ports"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// newTestDB connects to IDENTITY_TEST_DSN, a database with the app_user
// table from postgres/init/01-identity-service.sql, or skips.
func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("IDENTITY_TEST_DSN")
	if dsn == "" {
		t.Skip("IDENTITY_TEST_DSN not set")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	return db
}

func TestUserRepository_StatusAndRoles(t *testing.T) {
	db := newTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Microsecond)
	user, err := repo.Create(ctx, entity.User{
		ID: uuid.NewString(), Email: uuid.NewString() + "@example.com", PasswordHash: "x",
		Role: constants.RoleManager, Status: constants.UserActive, CreatedAt: now, UpdatedAt: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Exec("DELETE FROM app_user WHERE id = ?", user.ID) })

	if ok, err := repo.ExistsWithRole(ctx, constants.RoleManager); err != nil || !ok {
		t.Errorf("ExistsWithRole(MANAGER) = %v, %v; want true", ok, err)
	}

	later := now.Add(time.Minute)
	updated, err := repo.UpdateStatus(ctx, user.ID, constants.UserLocked, later)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Status != constants.UserLocked || !updated.UpdatedAt.Equal(later) || updated.Email != user.Email {
		t.Errorf("unexpected updated user %+v", updated)
	}

	err = db.Transaction(func(tx *gorm.DB) error {
		locked, err := NewUserRepository(tx).GetByIDForUpdate(ctx, user.ID)
		if err == nil && locked.Status != constants.UserLocked {
			t.Errorf("GetByIDForUpdate status = %q", locked.Status)
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}

	missing := uuid.NewString()
	if _, err := repo.UpdateStatus(ctx, missing, constants.UserLocked, later); !errors.Is(err, ports.ErrNotFound) {
		t.Errorf("UpdateStatus(missing) err = %v, want ErrNotFound", err)
	}
	if _, err := repo.GetByIDForUpdate(ctx, "not-a-uuid"); !errors.Is(err, ports.ErrNotFound) {
		t.Errorf("GetByIDForUpdate(not-a-uuid) err = %v, want ErrNotFound", err)
	}
}
