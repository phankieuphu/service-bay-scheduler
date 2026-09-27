package handler

import (
	"context"
	"encoding/json"
	"errors"
	"identity-service/internal/constants"
	"identity-service/internal/domain/entity"
	"identity-service/internal/domain/ports"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

type mockAuthService struct {
	registerFunc func(ctx context.Context, register entity.Register) (entity.User, error)
	loginFunc    func(ctx context.Context, email, password string) (entity.TokenPair, error)
	refreshFunc  func(ctx context.Context, refreshToken string) (entity.TokenPair, error)
	logoutFunc   func(ctx context.Context, refreshToken string) error
	meFunc       func(ctx context.Context, userID string) (entity.User, error)
}

func (m *mockAuthService) Register(ctx context.Context, r entity.Register) (entity.User, error) {
	return m.registerFunc(ctx, r)
}
func (m *mockAuthService) Login(ctx context.Context, email, password string) (entity.TokenPair, error) {
	return m.loginFunc(ctx, email, password)
}
func (m *mockAuthService) Refresh(ctx context.Context, token string) (entity.TokenPair, error) {
	return m.refreshFunc(ctx, token)
}
func (m *mockAuthService) Logout(ctx context.Context, token string) error {
	return m.logoutFunc(ctx, token)
}
func (m *mockAuthService) Me(ctx context.Context, userID string) (entity.User, error) {
	return m.meFunc(ctx, userID)
}

// mockTokens accepts exactly one token, "good".
type mockTokens struct{}

func (mockTokens) IssueAccessToken(entity.User) (string, time.Time, error) {
	return "", time.Time{}, nil
}

func (mockTokens) ParseAccessToken(token string) (entity.AccessClaims, error) {
	if token != "good" {
		return entity.AccessClaims{}, ports.ErrInvalidToken
	}
	return entity.AccessClaims{UserID: "user-1", Role: constants.RoleCustomer}, nil
}

func newTestRouter(svc ports.AuthService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	NewAuthHandler(svc, mockTokens{}).RegisterRoutes(r.Group("/api/v1"))
	return r
}

func do(r http.Handler, method, path, body string, header map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	for k, v := range header {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestAuthHandler_Register(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		err      error
		wantCode int
	}{
		{"created", `{"email":"jane@example.com","password":"s3cret-pass"}`, nil, http.StatusCreated},
		{"invalid email", `{"email":"nope","password":"s3cret-pass"}`, nil, http.StatusBadRequest},
		{"short password", `{"email":"jane@example.com","password":"short"}`, nil, http.StatusBadRequest},
		{"unknown role", `{"email":"jane@example.com","password":"s3cret-pass","role":"GOD"}`, nil, http.StatusBadRequest},
		{"duplicate email", `{"email":"jane@example.com","password":"s3cret-pass"}`, ports.ErrConflict, http.StatusConflict},
		{"privileged role", `{"email":"jane@example.com","password":"s3cret-pass","role":"ADMIN"}`, ports.ErrRoleNotAllowed, http.StatusForbidden},
		{"internal error hides details", `{"email":"jane@example.com","password":"s3cret-pass"}`, errors.New("pq: connection refused"), http.StatusInternalServerError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockAuthService{registerFunc: func(_ context.Context, r entity.Register) (entity.User, error) {
				if tt.err != nil {
					return entity.User{}, tt.err
				}
				return entity.User{ID: "user-1", Email: r.Email, Role: constants.RoleCustomer, Status: constants.UserActive}, nil
			}}
			w := do(newTestRouter(svc), http.MethodPost, "/api/v1/auth/register", tt.body, nil)
			if w.Code != tt.wantCode {
				t.Fatalf("code = %d, want %d; body %s", w.Code, tt.wantCode, w.Body)
			}
			if strings.Contains(w.Body.String(), "pq:") {
				t.Errorf("internal error leaked: %s", w.Body)
			}
			if w.Code == http.StatusCreated && strings.Contains(w.Body.String(), "password") {
				t.Errorf("response mentions password: %s", w.Body)
			}
		})
	}
}

func TestAuthHandler_Login(t *testing.T) {
	expires := time.Now().Add(15 * time.Minute)
	tests := []struct {
		name     string
		err      error
		wantCode int
	}{
		{"ok", nil, http.StatusOK},
		{"bad credentials", ports.ErrInvalidCredentials, http.StatusUnauthorized},
		{"inactive", ports.ErrUserInactive, http.StatusForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockAuthService{loginFunc: func(context.Context, string, string) (entity.TokenPair, error) {
				return entity.TokenPair{AccessToken: "a", AccessExpiresAt: expires, RefreshToken: "r"}, tt.err
			}}
			w := do(newTestRouter(svc), http.MethodPost, "/api/v1/auth/login", `{"email":"jane@example.com","password":"x"}`, nil)
			if w.Code != tt.wantCode {
				t.Fatalf("code = %d, want %d; body %s", w.Code, tt.wantCode, w.Body)
			}
			if tt.err != nil {
				return
			}
			var resp struct {
				AccessToken  string `json:"access_token"`
				TokenType    string `json:"token_type"`
				ExpiresIn    int64  `json:"expires_in"`
				RefreshToken string `json:"refresh_token"`
			}
			_ = json.Unmarshal(w.Body.Bytes(), &resp)
			if resp.AccessToken != "a" || resp.RefreshToken != "r" || resp.TokenType != "Bearer" || resp.ExpiresIn < 890 {
				t.Errorf("unexpected response %+v", resp)
			}
		})
	}
}

func TestAuthHandler_RefreshAndLogout(t *testing.T) {
	svc := &mockAuthService{
		refreshFunc: func(context.Context, string) (entity.TokenPair, error) {
			return entity.TokenPair{}, ports.ErrInvalidToken
		},
		logoutFunc: func(context.Context, string) error { return nil },
	}
	r := newTestRouter(svc)

	if w := do(r, http.MethodPost, "/api/v1/auth/refresh", `{"refresh_token":"old"}`, nil); w.Code != http.StatusUnauthorized {
		t.Errorf("refresh with bad token: code = %d, want 401", w.Code)
	}
	if w := do(r, http.MethodPost, "/api/v1/auth/refresh", `{}`, nil); w.Code != http.StatusBadRequest {
		t.Errorf("refresh without token: code = %d, want 400", w.Code)
	}
	if w := do(r, http.MethodPost, "/api/v1/auth/logout", `{"refresh_token":"r"}`, nil); w.Code != http.StatusNoContent {
		t.Errorf("logout: code = %d, want 204", w.Code)
	}
}

func TestAuthHandler_Me(t *testing.T) {
	svc := &mockAuthService{meFunc: func(_ context.Context, userID string) (entity.User, error) {
		if userID != "user-1" {
			return entity.User{}, ports.ErrNotFound
		}
		return entity.User{ID: "user-1", Email: "jane@example.com", Role: constants.RoleCustomer, Status: constants.UserActive}, nil
	}}
	r := newTestRouter(svc)

	tests := []struct {
		name     string
		header   map[string]string
		wantCode int
	}{
		{"valid token", map[string]string{"Authorization": "Bearer good"}, http.StatusOK},
		{"lowercase scheme", map[string]string{"Authorization": "bearer good"}, http.StatusOK},
		{"no header", nil, http.StatusUnauthorized},
		{"wrong scheme", map[string]string{"Authorization": "Basic good"}, http.StatusUnauthorized},
		{"invalid token", map[string]string{"Authorization": "Bearer bad"}, http.StatusUnauthorized},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := do(r, http.MethodGet, "/api/v1/me", "", tt.header)
			if w.Code != tt.wantCode {
				t.Fatalf("code = %d, want %d; body %s", w.Code, tt.wantCode, w.Body)
			}
			if w.Code == http.StatusOK && !strings.Contains(w.Body.String(), `"email":"jane@example.com"`) {
				t.Errorf("unexpected body %s", w.Body)
			}
		})
	}

	t.Run("token for deleted user is 401", func(t *testing.T) {
		gone := &mockAuthService{meFunc: func(context.Context, string) (entity.User, error) { return entity.User{}, ports.ErrNotFound }}
		if w := do(newTestRouter(gone), http.MethodGet, "/api/v1/me", "", map[string]string{"Authorization": "Bearer good"}); w.Code != http.StatusUnauthorized {
			t.Errorf("code = %d, want 401", w.Code)
		}
	})
}
