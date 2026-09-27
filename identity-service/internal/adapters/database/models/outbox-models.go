package models

import "time"

type OutboxMessage struct {
	ID          int64      `json:"id" gorm:"column:id;type:bigint"`
	Topic       string     `json:"topic" gorm:"column:topic;type:varchar;size:255"`
	MessageKey  string     `json:"message_key" gorm:"column:message_key;type:varchar;size:255"`
	Payload     []byte     `json:"payload" gorm:"column:payload;type:jsonb"`
	CreatedAt   time.Time  `json:"created_at" gorm:"column:created_at;type:timestamptz;autoCreateTime"`
	PublishedAt *time.Time `json:"published_at" gorm:"column:published_at;type:timestamptz"`
}

func (OutboxMessage) TableName() string {
	return "outbox_message"
}
