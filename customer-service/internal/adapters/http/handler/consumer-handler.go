package handler

import (
	"context"
	"customer-service/internal/adapters/kafka"
	"customer-service/internal/domain/events"
	"customer-service/internal/domain/ports"
	"customer-service/pkg/logger"
	"encoding/json"
	"errors"
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
func RegisterConsumer(topic string, service ports.CustomerService) TopicHandle {
	handle := identifyHandle(service)
	return NewTopicHandle(topic,
		NewConsumerHandler(events.UserCreated, handle),
		NewConsumerHandler(events.UserUpdated, handle),
	)
}

// identifyHandle gives every CUSTOMER user a linked customer profile.
// LinkUser is a no-op for a user that already has one, so redelivered
// events are harmless.
func identifyHandle(service ports.CustomerService) ConsumerHandle {
	return func(ctx context.Context, value []byte) error {
		var event events.UserEvent
		if err := json.Unmarshal(value, &event); err != nil {
			logger.ErrorContext(ctx, "decode user event", "error", err)
			return nil
		}
		if event.User.Role != events.UserRoleCustomer || event.User.ID == "" || event.User.Email == "" {
			return nil
		}

		customer, err := service.LinkUser(ctx, event.User.ID, event.User.Email)
		if errors.Is(err, ports.ErrConflict) {
			// The email belongs to a customer already linked to another user
			// (or one that was deleted). Retrying won't change that.
			logger.WarnContext(ctx, "user not linked: email taken by another customer", "event_id", event.EventID, "user_id", event.User.ID)
			return nil
		}
		if err != nil {
			return err
		}
		logger.InfoContext(ctx, "user linked to customer", "event_id", event.EventID, "event_type", event.EventType, "user_id", event.User.ID, "customer_id", customer.ID)
		return nil
	}
}
