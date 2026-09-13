package repository

import (
	"context"
	"customer-service/internal/adapters/database/models"
	"customer-service/internal/domain/entity"
	"customer-service/internal/domain/ports"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

// pgUniqueViolationCode is PostgreSQL's SQLSTATE code for a unique-key
// violation (e.g. the email unique constraint).
const pgUniqueViolationCode = "23505"

type CustomerRepository struct {
	db *gorm.DB
}

// Create implements [ports.CustomerRepository].
func (c CustomerRepository) Create(ctx context.Context, customer entity.Customer) (entity.Customer, error) {
	model := c.toModels(customer)

	if err := c.db.WithContext(ctx).Create(&model).Error; err != nil {
		if isDuplicateKeyError(err) {
			return entity.Customer{}, ports.ErrConflict
		}
		return entity.Customer{}, err
	}

	return c.toDomain(model), nil
}

// Delete implements [ports.CustomerRepository].
func (c CustomerRepository) Delete(ctx context.Context, id int64) error {
	result := c.db.WithContext(ctx).Delete(&models.Customer{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ports.ErrNotFound
	}

	return nil
}

// GetByID implements [ports.CustomerRepository].
func (c CustomerRepository) GetByID(ctx context.Context, id int64) (entity.Customer, error) {
	var model models.Customer

	if err := c.db.WithContext(ctx).First(&model, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return entity.Customer{}, ports.ErrNotFound
		}
		return entity.Customer{}, err
	}

	return c.toDomain(model), nil
}

// List implements [ports.CustomerRepository]. It keyset-paginates by id
// ascending: only rows with id greater than params.Cursor are returned, so a
// page stays stable even if earlier rows are inserted/deleted concurrently.
// It fetches one row past the requested limit to determine HasMore without a
// separate count query.
func (c CustomerRepository) List(ctx context.Context, params ports.ListCustomersParams) (ports.CustomerPage, error) {
	var rows []models.Customer

	query := c.db.WithContext(ctx).
		Order("id ASC").
		Limit(params.Limit + 1)
	if params.Cursor > 0 {
		query = query.Where("id > ?", params.Cursor)
	}

	if err := query.Find(&rows).Error; err != nil {
		return ports.CustomerPage{}, err
	}

	hasMore := len(rows) > params.Limit
	if hasMore {
		rows = rows[:params.Limit]
	}

	customers := make([]entity.Customer, len(rows))
	for i, row := range rows {
		customers[i] = c.toDomain(row)
	}

	var nextCursor int64
	if hasMore {
		nextCursor = customers[len(customers)-1].ID
	}

	return ports.CustomerPage{
		Customers:  customers,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}, nil
}

// Update implements [ports.CustomerRepository]. It optimistically locks on
// the row's updated_at, which the caller must have read (e.g. via GetByID)
// before mutating and passing back in customer.UpdatedAt. If another request
// updated or deleted the row first, this returns ErrConflict/ErrNotFound
// instead of silently overwriting the other write.
func (c CustomerRepository) Update(ctx context.Context, customer entity.Customer) error {
	model := c.toModels(customer)

	result := c.db.WithContext(ctx).
		Model(&models.Customer{}).
		Where("id = ? AND updated_at = ?", model.ID, model.UpdatedAt).
		Updates(&model)
	if result.Error != nil {
		if isDuplicateKeyError(result.Error) {
			return ports.ErrConflict
		}
		return result.Error
	}
	if result.RowsAffected == 0 {
		return c.updateFailureReason(ctx, model.ID)
	}

	return nil
}

// updateFailureReason distinguishes why an optimistic-locked update matched
// no rows: the customer no longer exists, vs. it exists but was changed
// (updated_at moved) by another write since the caller last read it.
func (c CustomerRepository) updateFailureReason(ctx context.Context, id int64) error {
	var count int64
	if err := c.db.WithContext(ctx).
		Model(&models.Customer{}).
		Where("id = ?", id).
		Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return ports.ErrNotFound
	}

	return ports.ErrConflict
}

func isDuplicateKeyError(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolationCode
}

func (c CustomerRepository) toModels(customer entity.Customer) models.Customer {
	return models.Customer{
		ID:        customer.ID,
		Name:      customer.Name,
		Email:     customer.Email,
		Phone:     customer.Phone,
		BirthDay:  customer.BirthDay,
		Status:    customer.Status,
		CreatedAt: customer.CreatedAt,
		UpdatedAt: customer.UpdatedAt,
	}
}

func (c CustomerRepository) toDomain(model models.Customer) entity.Customer {
	return entity.Customer{
		ID:        model.ID,
		Name:      model.Name,
		Email:     model.Email,
		Phone:     model.Phone,
		BirthDay:  model.BirthDay,
		Status:    model.Status,
		CreatedAt: model.CreatedAt,
		UpdatedAt: model.UpdatedAt,
	}
}

func NewCustomerRepository(db *gorm.DB) ports.CustomerRepository {
	return CustomerRepository{
		db: db,
	}
}
