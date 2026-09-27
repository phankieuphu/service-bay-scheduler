package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"identity-service/config"
	"identity-service/internal/constants"
	"identity-service/internal/domain/entity"
	"identity-service/internal/domain/ports"
	"identity-service/pkg/logger"
	"strings"
	"time"

	"github.com/google/uuid"
)

// refreshTokenBytes is the entropy of an opaque refresh token (256 bits).
const refreshTokenBytes = 32

type AuthService struct {
	config        config.Config
	users         ports.UserRepository
	outbox        ports.OutboxRepository
	txManager     ports.TxManager
	hasher        ports.PasswordHasher
	tokens        ports.TokenIssuer
	refreshTokens ports.RefreshTokenStore
	now           func() time.Time

	// dummyHash is compared against when a login names an unknown email,
	// so that path costs the same bcrypt work as a wrong password and
	// response timing doesn't reveal which emails have accounts.
	dummyHash string
}

func NewAuthService(
	cfg config.Config,
	users ports.UserRepository,
	outbox ports.OutboxRepository,
	txManager ports.TxManager,
	hasher ports.PasswordHasher,
	tokens ports.TokenIssuer,
	refreshTokens ports.RefreshTokenStore,
) (ports.AuthService, error) {
	dummyHash, err := hasher.Hash("identity-service-timing-equalizer")
	if err != nil {
		return nil, err
	}
	return &AuthService{
		config:        cfg,
		users:         users,
		outbox:        outbox,
		txManager:     txManager,
		hasher:        hasher,
		tokens:        tokens,
		refreshTokens: refreshTokens,
		now:           time.Now,
		dummyHash:     dummyHash,
	}, nil
}

// Register implements [ports.AuthService]. The user row and its UserCreated
// outbox row commit in one transaction, so a user never exists without
// downstream services eventually hearing about it (and vice versa).
func (s *AuthService) Register(ctx context.Context, register entity.Register) (entity.User, error) {
	if register.Role == "" {
		register.Role = constants.RoleCustomer
	}
	if !register.Role.SelfRegisterable() {
		return entity.User{}, ports.ErrRoleNotAllowed
	}

	hash, err := s.hasher.Hash(register.Password)
	if err != nil {
		return entity.User{}, err
	}

	now := s.now().UTC()
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
	err = s.txManager.RunInTx(ctx, func(ctx context.Context) error {
		created, err = s.users.Create(ctx, user)
		if err != nil {
			return err
		}

		payload, err := json.Marshal(entity.UserEvent{
			EventID:    uuid.NewString(),
			EventType:  constants.UserCreated,
			OccurredAt: now,
			User: entity.UserEventPayload{
				ID:     created.ID,
				Email:  created.Email,
				Role:   created.Role,
				Status: created.Status,
			},
		})
		if err != nil {
			return err
		}

		return s.outbox.Create(ctx, entity.OutboxMessage{
			Topic:   s.config.Kafka.ProducerTopic,
			Key:     created.ID,
			Payload: payload,
		})
	})
	if err != nil {
		if !errors.Is(err, ports.ErrConflict) {
			logger.ErrorContext(ctx, "failed to register user", "error", err)
		}
		return entity.User{}, err
	}

	return created, nil
}

// Login implements [ports.AuthService].
func (s *AuthService) Login(ctx context.Context, email, password string) (entity.TokenPair, error) {
	user, err := s.users.GetByEmail(ctx, normalizeEmail(email))
	if errors.Is(err, ports.ErrNotFound) {
		_ = s.hasher.Compare(s.dummyHash, password)
		return entity.TokenPair{}, ports.ErrInvalidCredentials
	}
	if err != nil {
		return entity.TokenPair{}, err
	}

	if err := s.hasher.Compare(user.PasswordHash, password); err != nil {
		return entity.TokenPair{}, ports.ErrInvalidCredentials
	}
	// Checked only after the password, so an inactive account's status
	// isn't disclosed to someone who doesn't know its password.
	if user.Status != constants.UserActive {
		return entity.TokenPair{}, ports.ErrUserInactive
	}

	return s.issueTokenPair(ctx, user)
}

// Refresh implements [ports.AuthService]. Consume deletes the old token
// before a new one is issued, so replaying a refresh token (e.g. one that
// leaked) fails instead of minting a second session.
func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (entity.TokenPair, error) {
	userID, err := s.refreshTokens.Consume(ctx, hashRefreshToken(refreshToken))
	if err != nil {
		return entity.TokenPair{}, err
	}

	user, err := s.users.GetByID(ctx, userID)
	if errors.Is(err, ports.ErrNotFound) {
		return entity.TokenPair{}, ports.ErrInvalidToken
	}
	if err != nil {
		return entity.TokenPair{}, err
	}
	if user.Status != constants.UserActive {
		return entity.TokenPair{}, ports.ErrUserInactive
	}

	return s.issueTokenPair(ctx, user)
}

// Logout implements [ports.AuthService]. The access token stays valid until
// it expires (at most JWT.AccessTTL): it's stateless, and checking a
// denylist on every request isn't worth it at a 15-minute TTL.
func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	return s.refreshTokens.Delete(ctx, hashRefreshToken(refreshToken))
}

// Me implements [ports.AuthService].
func (s *AuthService) Me(ctx context.Context, userID string) (entity.User, error) {
	return s.users.GetByID(ctx, userID)
}

func (s *AuthService) issueTokenPair(ctx context.Context, user entity.User) (entity.TokenPair, error) {
	accessToken, accessExpiresAt, err := s.tokens.IssueAccessToken(user)
	if err != nil {
		return entity.TokenPair{}, err
	}

	refreshToken, err := newRefreshToken()
	if err != nil {
		return entity.TokenPair{}, err
	}
	ttl := s.config.JWT.RefreshTTL
	if err := s.refreshTokens.Save(ctx, hashRefreshToken(refreshToken), user.ID, ttl); err != nil {
		return entity.TokenPair{}, err
	}

	return entity.TokenPair{
		AccessToken:      accessToken,
		AccessExpiresAt:  accessExpiresAt,
		RefreshToken:     refreshToken,
		RefreshExpiresAt: s.now().Add(ttl),
	}, nil
}

func newRefreshToken() (string, error) {
	b := make([]byte, refreshTokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// hashRefreshToken is what the store is keyed on. A fast hash is enough
// here (unlike passwords): the token already has 256 bits of entropy.
func hashRefreshToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
