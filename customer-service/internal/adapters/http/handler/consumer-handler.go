package handler

import (
	"context"
	"customer-service/internal/adapters/kafka"
	"customer-service/internal/domain/events"
	"customer-service/pkg/logger"
	"encoding/json"
)

// ConsumerHandle processes one message. payload is the raw message value
// (the full envelope), so each handler decodes its own typed event.
type ConsumerHandle func(ctx context.Context, payload []byte) error

type TopicHandle struct {
	Topic    string
	Handlers map[events.EventType]ConsumerHandle
}

type ConsumerHandler struct {
	Event  events.EventType
	Handle ConsumerHandle
}

func NewConsumerHandler(event events.EventType, consumer ConsumerHandle) ConsumerHandler {
	return ConsumerHandler{
		Event:  event,
		Handle: consumer,
	}
}

func NewTopicHandle(topic string, consumerHandlers ...ConsumerHandler) TopicHandle {
	handlers := make(map[events.EventType]ConsumerHandle, len(consumerHandlers))
	for _, h := range consumerHandlers {
		handlers[h.Event] = h.Handle
	}
	return TopicHandle{
		Topic:    topic,
		Handlers: handlers,
	}
}

func (t TopicHandle) NewMessageHandler() kafka.MessageHandler {
	return func(ctx context.Context, _, value []byte) error {
		var event events.MessageEvent
		if err := json.Unmarshal(value, &event); err != nil {
			// A malformed message will never decode; skip it rather than retry.
			logger.ErrorContext(ctx, "decode event envelope", "topic", t.Topic, "error", err)
			return nil
		}

		handle, ok := t.Handlers[event.EventType]
		if !ok {
			logger.DebugContext(ctx, "no handler for event", "topic", t.Topic, "event_type", event.EventType)
			return nil
		}
		return handle(ctx, value)
	}
}

// RegisterConsumer wires the identity.user-events handlers for topic.
func RegisterConsumer(topic string) TopicHandle {
	return NewTopicHandle(topic,
		NewConsumerHandler(events.UserCreated, identifyHandle),
		NewConsumerHandler(events.UserUpdated, identifyHandle),
	)
}

func identifyHandle(ctx context.Context, value []byte) error {
	var event events.UserEvent
	if err := json.Unmarshal(value, &event); err != nil {
		logger.ErrorContext(ctx, "decode user event", "error", err)
		return nil
	}
	if event.User.Role != events.UserRoleCustomer {
		return nil
	}
	logger.InfoContext(ctx, "identify handle", "event_id", event.EventID, "event_type", event.EventType, "user_id", event.User.ID)
	return nil
}
