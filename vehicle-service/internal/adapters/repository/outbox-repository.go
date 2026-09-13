package repository

import (
	"context"
	"vehicle-service/internal/adapters/database/models"
	database_provider "vehicle-service/internal/adapters/database/provider"
	"vehicle-service/internal/domain/entity"
	"vehicle-service/internal/domain/ports"
	"time"

	"gorm.io/gorm"
)

type OutboxRepository struct {
	db *gorm.DB
}

// Create implements [ports.OutboxRepository]. Called from inside
// TxManager.RunInTx alongside the domain write it describes, so it
// commits (or rolls back) atomically with that write.
func (o OutboxRepository) Create(ctx context.Context, message entity.OutboxMessage) error {
	model := models.OutboxMessage{
		Topic:      message.Topic,
		MessageKey: message.Key,
		Payload:    message.Payload,
	}

	return database_provider.DBFromContext(ctx, o.db).Create(&model).Error
}

// FetchUnpublished implements [ports.OutboxRepository].
func (o OutboxRepository) FetchUnpublished(ctx context.Context, limit int) ([]entity.OutboxMessage, error) {
	var rows []models.OutboxMessage

	if err := database_provider.DBFromContext(ctx, o.db).
		Where("published_at IS NULL").
		Order("id ASC").
		Limit(limit).
		Find(&rows).Error; err != nil {
		return nil, err
	}

	messages := make([]entity.OutboxMessage, len(rows))
	for i, row := range rows {
		messages[i] = entity.OutboxMessage{
			ID:          row.ID,
			Topic:       row.Topic,
			Key:         row.MessageKey,
			Payload:     row.Payload,
			CreatedAt:   row.CreatedAt,
			PublishedAt: row.PublishedAt,
		}
	}

	return messages, nil
}

// MarkPublished implements [ports.OutboxRepository].
func (o OutboxRepository) MarkPublished(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}

	now := time.Now()
	return database_provider.DBFromContext(ctx, o.db).
		Model(&models.OutboxMessage{}).
		Where("id IN ?", ids).
		Update("published_at", now).Error
}

func NewOutboxRepository(db *gorm.DB) ports.OutboxRepository {
	return OutboxRepository{
		db: db,
	}
}
