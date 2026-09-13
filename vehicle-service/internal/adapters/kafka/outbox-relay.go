package kafka

import (
	"context"
	"vehicle-service/internal/domain/ports"
	"vehicle-service/pkg/logger"
	"time"
)

// outboxRelayBatchSize caps how many rows a single poll claims, so one
// slow publish doesn't hold up the rest of the queue indefinitely.
const outboxRelayBatchSize = 100

// OutboxRelay polls the outbox table and publishes each unpublished row
// to Kafka, marking it published once the broker has acked it. It is the
// only thing that ever calls Producer.Publish for outbox-backed events —
// nothing publishes to Kafka synchronously from inside a write transaction.
type OutboxRelay struct {
	producer ports.Producer
	outbox   ports.OutboxRepository
	interval time.Duration
}

func NewOutboxRelay(producer ports.Producer, outbox ports.OutboxRepository, interval time.Duration) *OutboxRelay {
	return &OutboxRelay{
		producer: producer,
		outbox:   outbox,
		interval: interval,
	}
}

// Start polls until ctx is cancelled. Run it in its own goroutine.
func (r *OutboxRelay) Start(ctx context.Context) {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.relayOnce(ctx)
		}
	}
}

// every 2 minutes fetch the tables outbox messages
// use kafka publish the message don't use kafka directly
func (r *OutboxRelay) relayOnce(ctx context.Context) {
	messages, err := r.outbox.FetchUnpublished(ctx, outboxRelayBatchSize)
	if err != nil {
		logger.ErrorContext(ctx, "outbox relay: failed to fetch unpublished messages", "error", err)
		return
	}

	published := make([]int64, 0, len(messages))
	for _, msg := range messages {
		if err := r.producer.Publish(ctx, msg.Topic, msg.Key, msg.Payload); err != nil {
			logger.ErrorContext(ctx, "outbox relay: failed to publish message", "id", msg.ID, "topic", msg.Topic, "error", err)
			// Leave it unpublished; the next poll retries it. A gap here
			// (rather than continuing past a broker outage) preserves
			// per-key ordering within a poll.
			break
		}
		published = append(published, msg.ID)
	}

	if len(published) == 0 {
		return
	}
	if err := r.outbox.MarkPublished(ctx, published); err != nil {
		logger.ErrorContext(ctx, "outbox relay: failed to mark messages published", "ids", published, "error", err)
	}
}
