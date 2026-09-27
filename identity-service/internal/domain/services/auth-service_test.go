package services

import (
	"context"
	"encoding/json"
	"errors"
	"identity-service/config"
	"identity-service/internal/constants"
	"identity-service/internal/domain/entity"
	"identity-service/internal/domain/ports"
	"testing"
	"time"
)

type MockUserRepository struct {
	CreateFunc     func(ctx context.Context, user entity.User) (entity.User, error)
	GetByIDFunc    func(ctx context.Context, id string) (entity.User, error)
	GetByEmailFunc func(ctx context.Context, email string) (entity.User, error)
}

func (m *MockUserRepository) Create(ctx context.Context, user entity.User) (entity.User, error) {
	return m.CreateFunc(ctx, user)
}

func (m *MockUserRepository) GetByID(ctx context.Context, id string) (entity.User, error) {
	return m.GetByIDFunc(ctx, id)
}

func (m *MockUserRepository) GetByEmail(ctx context.Context, email string) (entity.User, error) {
	return m.GetByEmailFunc(ctx, email)
}

type MockOutboxRepository struct {
	CreateFunc           func(ctx context.Context, message entity.OutboxMessage) error
	FetchUnpublishedFunc func(ctx context.Context, limit int) ([]entity.OutboxMessage, error)
	MarkPublishedFunc    func(ctx context.Context, ids []int64) error
}

func (m *MockOutboxRepository) Create(ctx context.Context, message entity.OutboxMessage) error {
	return m.CreateFunc(ctx, message)
}

func (m *MockOutboxRepository) FetchUnpublished(ctx context.Context, limit int) ([]entity.OutboxMessage, error) {
	return m.FetchUnpublishedFunc(ctx, limit)
}

func (m *MockOutboxRepository) MarkPublished(ctx context.Context, ids []int64) error {
	return m.MarkPublishedFunc(ctx, ids)
}

type MockTxManager struct {
	RunInTxFunc func(ctx context.Context, fn func(ctx context.Context) error) error
}

func (m *MockTxManager) RunInTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if m.RunInTxFunc != nil {
		return m.RunInTxFunc(ctx, fn)
	}
	return fn(ctx)
}

// MockPasswordHasher "hashes" by prefixing, so tests stay fast and a hash
// is still distinguishable from its password.
type MockPasswordHasher struct {
	CompareCalls int
}

func (m *MockPasswordHasher) Hash(password string) (string, error) {
	return "hashed:" + password, nil
}

func (m *MockPasswordHasher) Compare(hash, password string) error {
	m.CompareCalls++
	if hash != "hashed:"+password {
		return errors.New("mismatch")
	}
	return nil
}

type MockTokenIssuer struct{}

func (MockTokenIssuer) IssueAccessToken(user entity.User) (string, time.Time, error) {
	return "access-for-" + user.ID, time.Now().Add(15 * time.Minute), nil
}

func (MockTokenIssuer) ParseAccessToken(string) (entity.AccessClaims, error) {
	return entity.AccessClaims{}, ports.ErrInvalidToken
}

// MockRefreshTokenStore is an in-memory store with the real semantics
// (Consume deletes), since rotation is what the refresh tests exercise.
type MockRefreshTokenStore struct {
	tokens map[string]string
}

func NewMockRefreshTokenStore() *MockRefreshTokenStore {
	return &MockRefreshTokenStore{tokens: map[string]string{}}
}

func (m *MockRefreshTokenStore) Save(_ context.Context, tokenHash, userID string, _ time.Duration) error {
	m.tokens[tokenHash] = userID
	return nil
}

func (m *MockRefreshTokenStore) Consume(_ context.Context, tokenHash string) (string, error) {
	userID, ok := m.tokens[tokenHash]
	if !ok {
		return "", ports.ErrInvalidToken
	}
	delete(m.tokens, tokenHash)
	return userID, nil
}

func (m *MockRefreshTokenStore) Delete(_ context.Context, tokenHash string) error {
	delete(m.tokens, tokenHash)
	return nil
}

var testConfig = config.Config{
	Kafka: config.Kafka{ProducerTopic: constants.UserEventsTopic},
	JWT:   config.JWT{RefreshTTL: time.Hour},
}

var activeUser = entity.User{
	ID:           "7f1c9a52-3e1b-4b8a-9a44-1f0c2b6d8e10",
	Email:        "jane@example.com",
	PasswordHash: "hashed:correct-password",
	Role:         constants.RoleCustomer,
	Status:       constants.UserActive,
}

func newTestService(t *testing.T, users *MockUserRepository, outbox *MockOutboxRepository, tx *MockTxManager, hasher *MockPasswordHasher, store *MockRefreshTokenStore) ports.AuthService {
	t.Helper()
	if outbox == nil {
		outbox = &MockOutboxRepository{}
	}
	if tx == nil {
		tx = &MockTxManager{}
	}
	if hasher == nil {
		hasher = &MockPasswordHasher{}
	}
	if store == nil {
		store = NewMockRefreshTokenStore()
	}
	svc, err := NewAuthService(testConfig, users, outbox, tx, hasher, MockTokenIssuer{}, store)
	if err != nil {
		t.Fatalf("NewAuthService: %v", err)
	}
	return svc
}

func TestAuthService_Register(t *testing.T) {
	t.Run("creates user and writes UserCreated outbox row in the same transaction", func(t *testing.T) {
		var inTx bool
		var createdUser entity.User
		var outboxMsg entity.OutboxMessage
		users := &MockUserRepository{CreateFunc: func(ctx context.Context, user entity.User) (entity.User, error) {
			if !inTx {
				t.Error("user created outside the transaction")
			}
			createdUser = user
			return user, nil
		}}
		outbox := &MockOutboxRepository{CreateFunc: func(ctx context.Context, message entity.OutboxMessage) error {
			if !inTx {
				t.Error("outbox row written outside the transaction")
			}
			outboxMsg = message
			return nil
		}}
		tx := &MockTxManager{RunInTxFunc: func(ctx context.Context, fn func(ctx context.Context) error) error {
			inTx = true
			defer func() { inTx = false }()
			return fn(ctx)
		}}

		svc := newTestService(t, users, outbox, tx, nil, nil)
		user, err := svc.Register(context.Background(), entity.Register{Email: "  Jane@Example.COM ", Password: "s3cret-pass"})
		if err != nil {
			t.Fatalf("Register: %v", err)
		}

		if user.Email != "jane@example.com" {
			t.Errorf("email = %q, want normalized jane@example.com", user.Email)
		}
		if user.Role != constants.RoleCustomer {
			t.Errorf("role = %q, want default CUSTOMER", user.Role)
		}
		if user.Status != constants.UserActive {
			t.Errorf("status = %q, want ACTIVE", user.Status)
		}
		if user.ID == "" {
			t.Error("ID not minted")
		}
		if createdUser.PasswordHash != "hashed:s3cret-pass" {
			t.Errorf("password stored as %q, want the hash", createdUser.PasswordHash)
		}

		if outboxMsg.Topic != constants.UserEventsTopic {
			t.Errorf("topic = %q, want %q", outboxMsg.Topic, constants.UserEventsTopic)
		}
		if outboxMsg.Key != user.ID {
			t.Errorf("key = %q, want user ID %q", outboxMsg.Key, user.ID)
		}
		var event entity.UserEvent
		if err := json.Unmarshal(outboxMsg.Payload, &event); err != nil {
			t.Fatalf("payload is not a UserEvent: %v", err)
		}
		if event.EventType != constants.UserCreated || event.EventID == "" || event.User.ID != user.ID ||
			event.User.Email != user.Email || event.User.Role != user.Role || event.User.Status != user.Status {
			t.Errorf("unexpected event %+v", event)
		}
		var raw map[string]map[string]any
		_ = json.Unmarshal(outboxMsg.Payload, &raw)
		if _, leaked := raw["user"]["password_hash"]; leaked {
			t.Error("event payload contains the password hash")
		}
	})

	t.Run("accepts technician", func(t *testing.T) {
		users := &MockUserRepository{CreateFunc: func(ctx context.Context, user entity.User) (entity.User, error) { return user, nil }}
		outbox := &MockOutboxRepository{CreateFunc: func(context.Context, entity.OutboxMessage) error { return nil }}
		svc := newTestService(t, users, outbox, nil, nil, nil)

		user, err := svc.Register(context.Background(), entity.Register{Email: "t@example.com", Password: "s3cret-pass", Role: constants.RoleTechnician})
		if err != nil || user.Role != constants.RoleTechnician {
			t.Fatalf("got (%+v, %v), want technician user", user, err)
		}
	})

	for _, role := range []constants.Role{constants.RoleManager, constants.RoleAdmin} {
		t.Run("rejects self-registering as "+string(role), func(t *testing.T) {
			users := &MockUserRepository{CreateFunc: func(context.Context, entity.User) (entity.User, error) {
				t.Error("user should not be created")
				return entity.User{}, nil
			}}
			svc := newTestService(t, users, nil, nil, nil, nil)

			_, err := svc.Register(context.Background(), entity.Register{Email: "x@example.com", Password: "s3cret-pass", Role: role})
			if !errors.Is(err, ports.ErrRoleNotAllowed) {
				t.Errorf("err = %v, want ErrRoleNotAllowed", err)
			}
		})
	}

	t.Run("propagates conflict and writes no outbox row", func(t *testing.T) {
		users := &MockUserRepository{CreateFunc: func(context.Context, entity.User) (entity.User, error) {
			return entity.User{}, ports.ErrConflict
		}}
		outbox := &MockOutboxRepository{CreateFunc: func(context.Context, entity.OutboxMessage) error {
			t.Error("outbox row written for a failed insert")
			return nil
		}}
		svc := newTestService(t, users, outbox, nil, nil, nil)

		_, err := svc.Register(context.Background(), entity.Register{Email: "jane@example.com", Password: "s3cret-pass"})
		if !errors.Is(err, ports.ErrConflict) {
			t.Errorf("err = %v, want ErrConflict", err)
		}
	})

	t.Run("outbox failure fails the registration", func(t *testing.T) {
		boom := errors.New("outbox down")
		users := &MockUserRepository{CreateFunc: func(ctx context.Context, user entity.User) (entity.User, error) { return user, nil }}
		outbox := &MockOutboxRepository{CreateFunc: func(context.Context, entity.OutboxMessage) error { return boom }}
		svc := newTestService(t, users, outbox, nil, nil, nil)

		if _, err := svc.Register(context.Background(), entity.Register{Email: "jane@example.com", Password: "s3cret-pass"}); !errors.Is(err, boom) {
			t.Errorf("err = %v, want %v", err, boom)
		}
	})
}

func TestAuthService_Login(t *testing.T) {
	byEmail := func(user entity.User) *MockUserRepository {
		return &MockUserRepository{GetByEmailFunc: func(_ context.Context, email string) (entity.User, error) {
			if email != user.Email {
				return entity.User{}, ports.ErrNotFound
			}
			return user, nil
		}}
	}

	t.Run("issues access and refresh tokens", func(t *testing.T) {
		store := NewMockRefreshTokenStore()
		svc := newTestService(t, byEmail(activeUser), nil, nil, nil, store)

		pair, err := svc.Login(context.Background(), " JANE@example.com", "correct-password")
		if err != nil {
			t.Fatalf("Login: %v", err)
		}
		if pair.AccessToken != "access-for-"+activeUser.ID {
			t.Errorf("access token = %q", pair.AccessToken)
		}
		if pair.RefreshToken == "" {
			t.Fatal("no refresh token")
		}
		if userID := store.tokens[hashRefreshToken(pair.RefreshToken)]; userID != activeUser.ID {
			t.Errorf("refresh token stored for %q, want %q", userID, activeUser.ID)
		}
		if _, raw := store.tokens[pair.RefreshToken]; raw {
			t.Error("refresh token stored in plain text")
		}
	})

	t.Run("wrong password", func(t *testing.T) {
		svc := newTestService(t, byEmail(activeUser), nil, nil, nil, nil)
		if _, err := svc.Login(context.Background(), activeUser.Email, "wrong"); !errors.Is(err, ports.ErrInvalidCredentials) {
			t.Errorf("err = %v, want ErrInvalidCredentials", err)
		}
	})

	t.Run("unknown email returns the same error and still does a password comparison", func(t *testing.T) {
		hasher := &MockPasswordHasher{}
		svc := newTestService(t, byEmail(activeUser), nil, nil, hasher, nil)

		_, err := svc.Login(context.Background(), "nobody@example.com", "whatever")
		if !errors.Is(err, ports.ErrInvalidCredentials) {
			t.Errorf("err = %v, want ErrInvalidCredentials", err)
		}
		if hasher.CompareCalls != 1 {
			t.Errorf("Compare called %d times, want 1 (timing equalization)", hasher.CompareCalls)
		}
	})

	t.Run("inactive user with correct password", func(t *testing.T) {
		locked := activeUser
		locked.Status = constants.UserLocked
		svc := newTestService(t, byEmail(locked), nil, nil, nil, nil)

		if _, err := svc.Login(context.Background(), locked.Email, "correct-password"); !errors.Is(err, ports.ErrUserInactive) {
			t.Errorf("err = %v, want ErrUserInactive", err)
		}
	})

	t.Run("inactive user with wrong password doesn't reveal status", func(t *testing.T) {
		locked := activeUser
		locked.Status = constants.UserLocked
		svc := newTestService(t, byEmail(locked), nil, nil, nil, nil)

		if _, err := svc.Login(context.Background(), locked.Email, "wrong"); !errors.Is(err, ports.ErrInvalidCredentials) {
			t.Errorf("err = %v, want ErrInvalidCredentials", err)
		}
	})
}

func TestAuthService_RefreshAndLogout(t *testing.T) {
	users := &MockUserRepository{
		GetByEmailFunc: func(context.Context, string) (entity.User, error) { return activeUser, nil },
		GetByIDFunc: func(_ context.Context, id string) (entity.User, error) {
			if id != activeUser.ID {
				return entity.User{}, ports.ErrNotFound
			}
			return activeUser, nil
		},
	}
	login := func(t *testing.T, svc ports.AuthService) entity.TokenPair {
		t.Helper()
		pair, err := svc.Login(context.Background(), activeUser.Email, "correct-password")
		if err != nil {
			t.Fatalf("Login: %v", err)
		}
		return pair
	}

	t.Run("refresh rotates the token and the old one can't be replayed", func(t *testing.T) {
		svc := newTestService(t, users, nil, nil, nil, nil)
		first := login(t, svc)

		second, err := svc.Refresh(context.Background(), first.RefreshToken)
		if err != nil {
			t.Fatalf("Refresh: %v", err)
		}
		if second.RefreshToken == first.RefreshToken {
			t.Error("refresh token was not rotated")
		}
		if _, err := svc.Refresh(context.Background(), first.RefreshToken); !errors.Is(err, ports.ErrInvalidToken) {
			t.Errorf("replayed refresh: err = %v, want ErrInvalidToken", err)
		}
		if _, err := svc.Refresh(context.Background(), second.RefreshToken); err != nil {
			t.Errorf("rotated token should work: %v", err)
		}
	})

	t.Run("unknown refresh token", func(t *testing.T) {
		svc := newTestService(t, users, nil, nil, nil, nil)
		if _, err := svc.Refresh(context.Background(), "not-a-token"); !errors.Is(err, ports.ErrInvalidToken) {
			t.Errorf("err = %v, want ErrInvalidToken", err)
		}
	})

	t.Run("refresh for a user deactivated since login", func(t *testing.T) {
		store := NewMockRefreshTokenStore()
		svc := newTestService(t, users, nil, nil, nil, store)
		pair := login(t, svc)

		inactive := activeUser
		inactive.Status = constants.UserInactive
		deactivated := &MockUserRepository{GetByIDFunc: func(context.Context, string) (entity.User, error) { return inactive, nil }}
		svc = newTestService(t, deactivated, nil, nil, nil, store)

		if _, err := svc.Refresh(context.Background(), pair.RefreshToken); !errors.Is(err, ports.ErrUserInactive) {
			t.Errorf("err = %v, want ErrUserInactive", err)
		}
	})

	t.Run("logout revokes the refresh token and is idempotent", func(t *testing.T) {
		svc := newTestService(t, users, nil, nil, nil, nil)
		pair := login(t, svc)

		if err := svc.Logout(context.Background(), pair.RefreshToken); err != nil {
			t.Fatalf("Logout: %v", err)
		}
		if err := svc.Logout(context.Background(), pair.RefreshToken); err != nil {
			t.Errorf("second Logout: %v", err)
		}
		if _, err := svc.Refresh(context.Background(), pair.RefreshToken); !errors.Is(err, ports.ErrInvalidToken) {
			t.Errorf("refresh after logout: err = %v, want ErrInvalidToken", err)
		}
	})
}
