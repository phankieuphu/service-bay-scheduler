package repository

import (
	"context"
	"errors"
	"identity-service/internal/adapters/database/models"
	database_provider "identity-service/internal/adapters/database/provider"
	"identity-service/internal/constants"
	"identity-service/internal/domain/entity"
	"identity-service/internal/domain/ports"
	"identity-service/pkg/utils"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) ports.UserRepository {
	return UserRepository{db: db}
}

// Create implements [ports.UserRepository].
func (u UserRepository) Create(ctx context.Context, user entity.User) (entity.User, error) {
	model := u.toModel(user)

	if err := database_provider.DBFromContext(ctx, u.db).Create(&model).Error; err != nil {
		if utils.IsDuplicateKeyError(err) {
			return entity.User{}, ports.ErrConflict
		}
		return entity.User{}, err
	}

	return u.toDomain(model), nil
}

// GetByID implements [ports.UserRepository].
func (u UserRepository) GetByID(ctx context.Context, id string) (entity.User, error) {
	// A non-UUID id can't match any row, and passing it through would make
	// Postgres fail the cast with an error instead of returning no rows.
	if uuid.Validate(id) != nil {
		return entity.User{}, ports.ErrNotFound
	}
	return u.first(ctx, "id = ?", id)
}

// GetByEmail implements [ports.UserRepository].
func (u UserRepository) GetByEmail(ctx context.Context, email string) (entity.User, error) {
	return u.first(ctx, "lower(email) = lower(?)", email)
}

// GetByIDForUpdate implements [ports.UserRepository].
func (u UserRepository) GetByIDForUpdate(ctx context.Context, id string) (entity.User, error) {
	if uuid.Validate(id) != nil {
		return entity.User{}, ports.ErrNotFound
	}
	return u.firstWith(database_provider.DBFromContext(ctx, u.db).Clauses(clause.Locking{Strength: "UPDATE"}), "id = ?", id)
}

// UpdateStatus implements [ports.UserRepository].
func (u UserRepository) UpdateStatus(ctx context.Context, id string, status constants.UserStatus, updatedAt time.Time) (entity.User, error) {
	if uuid.Validate(id) != nil {
		return entity.User{}, ports.ErrNotFound
	}
	db := database_provider.DBFromContext(ctx, u.db)
	result := db.Model(&models.User{}).Where("id = ?", id).
		Updates(map[string]any{"status": string(status), "updated_at": updatedAt})
	if result.Error != nil {
		return entity.User{}, result.Error
	}
	if result.RowsAffected == 0 {
		return entity.User{}, ports.ErrNotFound
	}
	return u.firstWith(db, "id = ?", id)
}

// ExistsWithRole implements [ports.UserRepository].
func (u UserRepository) ExistsWithRole(ctx context.Context, role constants.Role) (bool, error) {
	var count int64
	err := database_provider.DBFromContext(ctx, u.db).Model(&models.User{}).
		Where("role = ?", string(role)).Limit(1).Count(&count).Error
	return count > 0, err
}

func (u UserRepository) first(ctx context.Context, query string, args ...any) (entity.User, error) {
	return u.firstWith(database_provider.DBFromContext(ctx, u.db), query, args...)
}

func (u UserRepository) firstWith(db *gorm.DB, query string, args ...any) (entity.User, error) {
	var model models.User

	if err := db.Where(query, args...).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return entity.User{}, ports.ErrNotFound
		}
		return entity.User{}, err
	}

	return u.toDomain(model), nil
}

func (u UserRepository) toModel(user entity.User) models.User {
	return models.User{
		ID:           user.ID,
		Email:        user.Email,
		PasswordHash: user.PasswordHash,
		Role:         string(user.Role),
		Status:       string(user.Status),
		CreatedAt:    user.CreatedAt,
		UpdatedAt:    user.UpdatedAt,
	}
}

func (u UserRepository) toDomain(model models.User) entity.User {
	return entity.User{
		ID:           model.ID,
		Email:        model.Email,
		PasswordHash: model.PasswordHash,
		Role:         constants.Role(model.Role),
		Status:       constants.UserStatus(model.Status),
		CreatedAt:    model.CreatedAt,
		UpdatedAt:    model.UpdatedAt,
	}
}
