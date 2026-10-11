package services

import (
	"context"
	"encoding/json"
	"identity-service/config"
	"identity-service/internal/constants"
	"identity-service/internal/domain/entity"
	"identity-service/internal/domain/ports"
	"time"

	"github.com/google/uuid"
)

// userDeps is what writing a user and its outbox event needs; AuthService
// and AdminService share it.
type userDeps struct {
	config    config.Config
	users     ports.UserRepository
	outbox    ports.OutboxRepository
	txManager ports.TxManager
	hasher    ports.PasswordHasher
	now       func() time.Time
}

// createUser hashes the password and writes the user plus its UserCreated
// outbox row in one transaction. register.Role must already be set and
// allowed for the caller.
func createUser(ctx context.Context, d userDeps, register entity.Register) (entity.User, error) {
	hash, err := d.hasher.Hash(register.Password)
	if err != nil {
		return entity.User{}, err
	}

	now := d.now().UTC()
	user := entity.User{
		ID:           uuid.NewString(),
		Email:        normalizeEmail(register.Email),
		PasswordHash: hash,
		Role:         register.Role,
		Status:       constants.UserActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	var created entity.User
	err = d.txManager.RunInTx(ctx, func(ctx context.Context) error {
		created, err = d.users.Create(ctx, user)
		if err != nil {
			return err
		}
		return writeUserEvent(ctx, d, constants.UserCreated, created, now)
	})
	if err != nil {
		return entity.User{}, err
	}
	return created, nil
}

// writeUserEvent writes user's event to the outbox. Call it inside the
// transaction that made the change.
func writeUserEvent(ctx context.Context, d userDeps, eventType constants.UserEventType, user entity.User, occurredAt time.Time) error {
	payload, err := json.Marshal(entity.UserEvent{
		EventID:    uuid.NewString(),
		EventType:  eventType,
		OccurredAt: occurredAt.UTC(),
		User: entity.UserEventPayload{
			ID:     user.ID,
			Email:  user.Email,
			Role:   user.Role,
			Status: user.Status,
		},
	})
	if err != nil {
		return err
	}

	return d.outbox.Create(ctx, entity.OutboxMessage{
		Topic:   d.config.Kafka.ProducerTopic,
		Key:     user.ID,
		Payload: payload,
	})
}
