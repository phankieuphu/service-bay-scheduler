package services

import (
	"context"
	"encoding/json"
	"errors"
	"identity-service/internal/constants"
	"identity-service/internal/domain/entity"
	"identity-service/internal/domain/ports"
	"testing"
	"time"
)

const (
	adminID    = "0b9d4c1e-1111-4a5b-9c3d-000000000001"
	customerID = "0b9d4c1e-2222-4a5b-9c3d-000000000002"
)

// adminFixture is an in-memory user table with the real semantics the
// admin service relies on.
type adminFixture struct {
	users  map[string]entity.User
	events []entity.UserEvent
	repo   *MockUserRepository
	outbox *MockOutboxRepository
}

func newAdminFixture(t *testing.T) *adminFixture {
	t.Helper()
	f := &adminFixture{users: map[string]entity.User{
		adminID:    {ID: adminID, Email: "root@example.com", Role: constants.RoleAdmin, Status: constants.UserActive},
		customerID: {ID: customerID, Email: "jane@example.com", Role: constants.RoleCustomer, Status: constants.UserActive},
	}}
	get := func(_ context.Context, id string) (entity.User, error) {
		u, ok := f.users[id]
		if !ok {
			return entity.User{}, ports.ErrNotFound
		}
		return u, nil
	}
	f.repo = &MockUserRepository{
		GetByIDFunc:          get,
		GetByIDForUpdateFunc: get,
		GetByEmailFunc: func(_ context.Context, email string) (entity.User, error) {
			for _, u := range f.users {
				if u.Email == email {
					return u, nil
				}
			}
			return entity.User{}, ports.ErrNotFound
		},
		CreateFunc: func(_ context.Context, u entity.User) (entity.User, error) {
			for _, existing := range f.users {
				if existing.Email == u.Email {
					return entity.User{}, ports.ErrConflict
				}
			}
			f.users[u.ID] = u
			return u, nil
		},
		UpdateStatusFunc: func(_ context.Context, id string, status constants.UserStatus, at time.Time) (entity.User, error) {
			u, ok := f.users[id]
			if !ok {
				return entity.User{}, ports.ErrNotFound
			}
			u.Status, u.UpdatedAt = status, at
			f.users[id] = u
			return u, nil
		},
		ExistsWithRoleFunc: func(_ context.Context, role constants.Role) (bool, error) {
			for _, u := range f.users {
				if u.Role == role {
					return true, nil
				}
			}
			return false, nil
		},
	}
	f.outbox = &MockOutboxRepository{CreateFunc: func(_ context.Context, m entity.OutboxMessage) error {
		var e entity.UserEvent
		if err := json.Unmarshal(m.Payload, &e); err != nil {
			t.Fatalf("outbox payload: %v", err)
		}
		if m.Key != e.User.ID || m.Topic != constants.UserEventsTopic {
			t.Errorf("outbox key/topic = %q/%q", m.Key, m.Topic)
		}
		f.events = append(f.events, e)
		return nil
	}}
	return f
}

func (f *adminFixture) service() ports.AdminService {
	return NewAdminService(testConfig, f.repo, f.outbox, &MockTxManager{}, &MockPasswordHasher{})
}

func TestAdminService_CreateUser(t *testing.T) {
	t.Run("admin creates a manager and UserCreated is emitted", func(t *testing.T) {
		f := newAdminFixture(t)
		user, err := f.service().CreateUser(context.Background(), adminID, entity.Register{Email: " Boss@Example.com", Password: "s3cret-pass", Role: constants.RoleManager})
		if err != nil {
			t.Fatal(err)
		}
		if user.Role != constants.RoleManager || user.Email != "boss@example.com" || user.Status != constants.UserActive {
			t.Errorf("unexpected user %+v", user)
		}
		if user.PasswordHash != "hashed:s3cret-pass" {
			t.Error("password should be hashed")
		}
		if len(f.events) != 1 || f.events[0].EventType != constants.UserCreated || f.events[0].User.Role != constants.RoleManager {
			t.Errorf("events = %+v", f.events)
		}
	})

	errCases := []struct {
		name  string
		actor string
		edit  func(f *adminFixture)
		role  constants.Role
		want  error
	}{
		{"non-admin actor", customerID, nil, constants.RoleTechnician, ports.ErrForbidden},
		{"unknown actor", "0b9d4c1e-9999-4a5b-9c3d-000000000009", nil, constants.RoleTechnician, ports.ErrForbidden},
		{"locked admin, token still valid", adminID, func(f *adminFixture) {
			u := f.users[adminID]
			u.Status = constants.UserLocked
			f.users[adminID] = u
		}, constants.RoleTechnician, ports.ErrForbidden},
		{"unknown role", adminID, nil, "GOD", ports.ErrInvalidInput},
	}
	for _, tt := range errCases {
		t.Run(tt.name, func(t *testing.T) {
			f := newAdminFixture(t)
			if tt.edit != nil {
				tt.edit(f)
			}
			_, err := f.service().CreateUser(context.Background(), tt.actor, entity.Register{Email: "new@example.com", Password: "s3cret-pass", Role: tt.role})
			if !errors.Is(err, tt.want) {
				t.Errorf("err = %v, want %v", err, tt.want)
			}
			if len(f.events) != 0 {
				t.Error("no event should be emitted")
			}
		})
	}

	t.Run("duplicate email is a conflict", func(t *testing.T) {
		f := newAdminFixture(t)
		_, err := f.service().CreateUser(context.Background(), adminID, entity.Register{Email: "jane@example.com", Password: "s3cret-pass", Role: constants.RoleTechnician})
		if !errors.Is(err, ports.ErrConflict) {
			t.Errorf("err = %v, want ErrConflict", err)
		}
	})
}

func TestAdminService_SetUserStatus(t *testing.T) {
	t.Run("locking emits UserUpdated with the new status", func(t *testing.T) {
		f := newAdminFixture(t)
		user, err := f.service().SetUserStatus(context.Background(), adminID, customerID, constants.UserLocked)
		if err != nil {
			t.Fatal(err)
		}
		if user.Status != constants.UserLocked || f.users[customerID].Status != constants.UserLocked {
			t.Errorf("status = %q", user.Status)
		}
		if len(f.events) != 1 || f.events[0].EventType != constants.UserUpdated ||
			f.events[0].User.ID != customerID || f.events[0].User.Status != constants.UserLocked {
			t.Errorf("events = %+v", f.events)
		}
	})

	t.Run("setting the current status is a no-op", func(t *testing.T) {
		f := newAdminFixture(t)
		if _, err := f.service().SetUserStatus(context.Background(), adminID, customerID, constants.UserActive); err != nil {
			t.Fatal(err)
		}
		if len(f.events) != 0 {
			t.Error("no event for an unchanged status")
		}
	})

	errCases := []struct {
		name   string
		actor  string
		target string
		status constants.UserStatus
		want   error
	}{
		{"own status", adminID, adminID, constants.UserLocked, ports.ErrInvalidInput},
		{"unknown status", adminID, customerID, "BANNED", ports.ErrInvalidInput},
		{"non-admin", customerID, adminID, constants.UserLocked, ports.ErrForbidden},
		{"unknown user", adminID, "0b9d4c1e-9999-4a5b-9c3d-000000000009", constants.UserLocked, ports.ErrNotFound},
	}
	for _, tt := range errCases {
		t.Run(tt.name, func(t *testing.T) {
			f := newAdminFixture(t)
			_, err := f.service().SetUserStatus(context.Background(), tt.actor, tt.target, tt.status)
			if !errors.Is(err, tt.want) {
				t.Errorf("err = %v, want %v", err, tt.want)
			}
			if len(f.events) != 0 {
				t.Error("no event should be emitted")
			}
		})
	}
}

func TestAdminService_BootstrapAdmin(t *testing.T) {
	t.Run("creates the first admin", func(t *testing.T) {
		f := newAdminFixture(t)
		delete(f.users, adminID)
		created, err := f.service().BootstrapAdmin(context.Background(), "Root@Example.com", "s3cret-pass")
		if err != nil || !created {
			t.Fatalf("got %v, %v; want true, nil", created, err)
		}
		if len(f.events) != 1 || f.events[0].User.Role != constants.RoleAdmin || f.events[0].User.Email != "root@example.com" {
			t.Errorf("events = %+v", f.events)
		}
	})

	t.Run("does nothing once an admin exists", func(t *testing.T) {
		f := newAdminFixture(t)
		created, err := f.service().BootstrapAdmin(context.Background(), "other@example.com", "s3cret-pass")
		if err != nil || created {
			t.Fatalf("got %v, %v; want false, nil", created, err)
		}
		if len(f.users) != 2 {
			t.Error("no user should be created")
		}
	})

	t.Run("never promotes an existing non-admin account", func(t *testing.T) {
		f := newAdminFixture(t)
		delete(f.users, adminID)
		created, err := f.service().BootstrapAdmin(context.Background(), "jane@example.com", "s3cret-pass")
		if err == nil || created {
			t.Fatalf("got %v, %v; want an error", created, err)
		}
		if f.users[customerID].Role != constants.RoleCustomer {
			t.Error("existing account was promoted")
		}
	})
}
