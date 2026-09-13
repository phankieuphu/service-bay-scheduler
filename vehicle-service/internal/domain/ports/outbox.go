package ports

import (
	"context"
	"vehicle-service/internal/domain/entity"
)

// OutboxRepository persists outbox rows written inside the same
// transaction as the domain write they describe, and lets a relay find
// and mark them once they've been published to Kafka.
type OutboxRepository interface {
	Create(ctx context.Context, message entity.OutboxMessage) error
	FetchUnpublished(ctx context.Context, limit int) ([]entity.OutboxMessage, error)
	MarkPublished(ctx context.Context, ids []int64) error
}
