package ports

import "context"

type Producer interface {
	Publish(ctx context.Context, topic string, key string, payload []byte) error
	Close() error
}

type KafkaConsumer interface {
	Start(ctx context.Context)
	Close() error
}
