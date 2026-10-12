package handler

import (
	"context"
	"fmt"
	"identity-service/internal/constants"
	"identity-service/internal/domain/entity"
	"identity-service/internal/domain/ports"
	"net/http"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

type mockAdminService struct {
	createFunc    func(ctx context.Context, actorID string, r entity.Register) (entity.User, error)
	setStatusFunc func(ctx context.Context, actorID, userID string, status constants.UserStatus) (entity.User, error)
}

func (m *mockAdminService) CreateUser(ctx context.Context, actorID string, r entity.Register) (entity.User, error) {
	return m.createFunc(ctx, actorID, r)
}

func (m *mockAdminService) SetUserStatus(ctx context.Context, actorID, userID string, status constants.UserStatus) (entity.User, error) {
	return m.setStatusFunc(ctx, actorID, userID, status)
}

func (m *mockAdminService) BootstrapAdmin(context.Context, string, string) (bool, error) {
	return false, nil
}

func newAdminRouter(svc ports.AdminService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	NewAdminHandler(svc, mockTokens{}).RegisterRoutes(r.Group("/api/v1"))
	return r
}

func TestAdminHandler_CreateUser(t *testing.T) {
	var gotActor string
	var gotRole constants.Role
	svc := &mockAdminService{createFunc: func(_ context.Context, actorID string, r entity.Register) (entity.User, error) {
		gotActor, gotRole = actorID, r.Role
		if r.Email == "taken@example.com" {
			return entity.User{}, ports.ErrConflict
		}
		return entity.User{ID: "new-1", Email: r.Email, Role: r.Role, Status: constants.UserActive}, nil
	}}
	r := newAdminRouter(svc)
	admin := map[string]string{"Authorization": "Bearer admin"}

	tests := []struct {
		name     string
		body     string
		header   map[string]string
		wantCode int
	}{
		{"admin creates a manager", `{"email":"boss@example.com","password":"s3cret-pass","role":"MANAGER"}`, admin, http.StatusCreated},
		{"role is required", `{"email":"boss@example.com","password":"s3cret-pass"}`, admin, http.StatusBadRequest},
		{"unknown role", `{"email":"boss@example.com","password":"s3cret-pass","role":"GOD"}`, admin, http.StatusBadRequest},
		{"email taken", `{"email":"taken@example.com","password":"s3cret-pass","role":"TECHNICIAN"}`, admin, http.StatusConflict},
		{"customer token", `{"email":"boss@example.com","password":"s3cret-pass","role":"MANAGER"}`, map[string]string{"Authorization": "Bearer good"}, http.StatusForbidden},
		{"no token", `{"email":"boss@example.com","password":"s3cret-pass","role":"MANAGER"}`, nil, http.StatusUnauthorized},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotActor = ""
			w := do(r, http.MethodPost, "/api/v1/admin/users", tt.body, tt.header)
			if w.Code != tt.wantCode {
				t.Fatalf("code = %d, want %d; body %s", w.Code, tt.wantCode, w.Body)
			}
			if w.Code == http.StatusCreated {
				if gotActor != "admin-1" || gotRole != constants.RoleManager {
					t.Errorf("service got actor %q role %q", gotActor, gotRole)
				}
				if !strings.Contains(w.Body.String(), `"role":"MANAGER"`) || strings.Contains(w.Body.String(), "password") {
					t.Errorf("unexpected body %s", w.Body)
				}
			}
			if (w.Code == http.StatusForbidden || w.Code == http.StatusUnauthorized) && gotActor != "" {
				t.Error("service must not be called without an admin token")
			}
		})
	}
}

func TestAdminHandler_UpdateUserStatus(t *testing.T) {
	svc := &mockAdminService{setStatusFunc: func(_ context.Context, actorID, userID string, status constants.UserStatus) (entity.User, error) {
		switch {
		case userID == "missing":
			return entity.User{}, ports.ErrNotFound
		case userID == actorID:
			return entity.User{}, fmt.Errorf("%w: you can't change your own status", ports.ErrInvalidInput)
		case actorID != "admin-1":
			return entity.User{}, ports.ErrForbidden
		}
		return entity.User{ID: userID, Email: "jane@example.com", Role: constants.RoleCustomer, Status: status}, nil
	}}
	r := newAdminRouter(svc)
	admin := map[string]string{"Authorization": "Bearer admin"}

	tests := []struct {
		name     string
		path     string
		body     string
		wantCode int
	}{
		{"lock", "/api/v1/admin/users/user-1", `{"status":"LOCKED"}`, http.StatusOK},
		{"unknown status", "/api/v1/admin/users/user-1", `{"status":"BANNED"}`, http.StatusBadRequest},
		{"own account", "/api/v1/admin/users/admin-1", `{"status":"LOCKED"}`, http.StatusBadRequest},
		{"unknown user", "/api/v1/admin/users/missing", `{"status":"LOCKED"}`, http.StatusNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := do(r, http.MethodPatch, tt.path, tt.body, admin)
			if w.Code != tt.wantCode {
				t.Fatalf("code = %d, want %d; body %s", w.Code, tt.wantCode, w.Body)
			}
			if w.Code == http.StatusOK && !strings.Contains(w.Body.String(), `"status":"LOCKED"`) {
				t.Errorf("unexpected body %s", w.Body)
			}
		})
	}
}
