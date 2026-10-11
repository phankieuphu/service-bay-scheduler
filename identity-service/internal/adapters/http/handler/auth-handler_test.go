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

// mockTokens accepts two tokens: "good" (a customer, user-1) and "admin"
// (an admin, admin-1).
type mockTokens struct{}

func (mockTokens) IssueAccessToken(entity.User) (string, time.Time, error) {
	return "", time.Time{}, nil
}

func (mockTokens) ParseAccessToken(token string) (entity.AccessClaims, error) {
	switch token {
	case "good":
		return entity.AccessClaims{UserID: "user-1", Role: constants.RoleCustomer}, nil
	case "admin":
		return entity.AccessClaims{UserID: "admin-1", Role: constants.RoleAdmin}, nil
	}
	return entity.AccessClaims{}, ports.ErrInvalidToken
}

func newTestRouter(svc ports.AuthService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	NewAuthHandler(svc, mockTokens{}, nil, RefreshCookies{Secure: true}).RegisterRoutes(r.Group("/api/v1"))
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
	refreshExpires := time.Now().Add(7 * 24 * time.Hour)
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
				return entity.TokenPair{AccessToken: "a", AccessExpiresAt: expires, RefreshToken: "r", RefreshExpiresAt: refreshExpires}, tt.err
			}}
			w := do(newTestRouter(svc), http.MethodPost, "/api/v1/auth/login", `{"email":"jane@example.com","password":"x"}`, nil)
			if w.Code != tt.wantCode {
				t.Fatalf("code = %d, want %d; body %s", w.Code, tt.wantCode, w.Body)
			}
			if tt.err != nil {
				if w.Header().Get("Set-Cookie") != "" {
					t.Error("a failed login must not set a cookie")
				}
				return
			}
			var resp map[string]any
			_ = json.Unmarshal(w.Body.Bytes(), &resp)
			if resp["access_token"] != "a" || resp["token_type"] != "Bearer" || resp["expires_in"].(float64) < 890 {
				t.Errorf("unexpected response %+v", resp)
			}
			if _, ok := resp["refresh_token"]; ok {
				t.Error("the refresh token must not be in the body, where scripts can read it")
			}
			assertRefreshCookie(t, w, "r")
		})
	}
}

// assertRefreshCookie checks w set the refresh cookie to value with every
// protective attribute.
func assertRefreshCookie(t *testing.T, w *httptest.ResponseRecorder, value string) {
	t.Helper()
	cookies := w.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("got %d cookies, want 1", len(cookies))
	}
	c := cookies[0]
	if c.Name != RefreshCookieName || c.Value != value || !c.HttpOnly || !c.Secure ||
		c.SameSite != http.SameSiteStrictMode || c.Path != "/api/v1/auth" || c.MaxAge <= 0 {
		t.Errorf("unexpected cookie %+v", c)
	}
}

func refreshCookie(token string) map[string]string {
	return map[string]string{"Cookie": RefreshCookieName + "=" + token, CSRFHeader: "fetch"}
}

func TestAuthHandler_Refresh(t *testing.T) {
	refreshExpires := time.Now().Add(7 * 24 * time.Hour)
	var gotToken string
	svc := &mockAuthService{refreshFunc: func(_ context.Context, token string) (entity.TokenPair, error) {
		gotToken = token
		if token != "old" {
			return entity.TokenPair{}, ports.ErrInvalidToken
		}
		return entity.TokenPair{AccessToken: "a2", AccessExpiresAt: time.Now().Add(time.Minute), RefreshToken: "new", RefreshExpiresAt: refreshExpires}, nil
	}}
	r := newTestRouter(svc)

	t.Run("rotates the cookie", func(t *testing.T) {
		w := do(r, http.MethodPost, "/api/v1/auth/refresh", "", refreshCookie("old"))
		if w.Code != http.StatusOK {
			t.Fatalf("code = %d; body %s", w.Code, w.Body)
		}
		if gotToken != "old" {
			t.Errorf("service got %q, want the cookie's token", gotToken)
		}
		assertRefreshCookie(t, w, "new")
	})

	t.Run("rejected token clears the cookie", func(t *testing.T) {
		w := do(r, http.MethodPost, "/api/v1/auth/refresh", "", refreshCookie("spent"))
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("code = %d, want 401", w.Code)
		}
		if c := w.Result().Cookies(); len(c) != 1 || c[0].MaxAge >= 0 {
			t.Errorf("cookie should be cleared, got %+v", c)
		}
	})

	t.Run("no cookie", func(t *testing.T) {
		if w := do(r, http.MethodPost, "/api/v1/auth/refresh", "", map[string]string{CSRFHeader: "fetch"}); w.Code != http.StatusUnauthorized {
			t.Errorf("code = %d, want 401", w.Code)
		}
	})

	t.Run("a token in the body is ignored", func(t *testing.T) {
		if w := do(r, http.MethodPost, "/api/v1/auth/refresh", `{"refresh_token":"old"}`, map[string]string{CSRFHeader: "fetch"}); w.Code != http.StatusUnauthorized {
			t.Errorf("code = %d, want 401", w.Code)
		}
	})

	t.Run("missing CSRF header", func(t *testing.T) {
		gotToken = ""
		w := do(r, http.MethodPost, "/api/v1/auth/refresh", "", map[string]string{"Cookie": RefreshCookieName + "=old"})
		if w.Code != http.StatusForbidden || gotToken != "" {
			t.Errorf("code = %d (service got %q), want 403 without reaching the service", w.Code, gotToken)
		}
	})
}

func TestAuthHandler_Logout(t *testing.T) {
	var revoked []string
	svc := &mockAuthService{logoutFunc: func(_ context.Context, token string) error {
		revoked = append(revoked, token)
		return nil
	}}
	r := newTestRouter(svc)

	w := do(r, http.MethodPost, "/api/v1/auth/logout", "", refreshCookie("r"))
	if w.Code != http.StatusNoContent {
		t.Fatalf("code = %d, want 204", w.Code)
	}
	if len(revoked) != 1 || revoked[0] != "r" {
		t.Errorf("revoked %v, want [r]", revoked)
	}
	if c := w.Result().Cookies(); len(c) != 1 || c[0].MaxAge >= 0 {
		t.Errorf("cookie should be cleared, got %+v", c)
	}

	if w := do(r, http.MethodPost, "/api/v1/auth/logout", "", map[string]string{CSRFHeader: "fetch"}); w.Code != http.StatusNoContent {
		t.Errorf("logout without a cookie: code = %d, want 204 (idempotent)", w.Code)
	}
	if len(revoked) != 1 {
		t.Error("nothing to revoke without a cookie")
	}
	if w := do(r, http.MethodPost, "/api/v1/auth/logout", "", map[string]string{"Cookie": RefreshCookieName + "=r"}); w.Code != http.StatusForbidden {
		t.Errorf("logout without CSRF header: code = %d, want 403", w.Code)
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
