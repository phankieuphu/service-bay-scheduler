package entity

import "time"

// OutboxMessage is a Kafka message recorded in the same database
// transaction as the state change that produced it. A relay process
// publishes it afterwards, so a write never has to reach Kafka
// synchronously to succeed.
type OutboxMessage struct {
	ID          int64
	Topic       string
	Key         string
	Payload     []byte
	CreatedAt   time.Time
	PublishedAt *time.Time
}
