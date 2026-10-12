package handler

import (
	"context"
	"errors"
	"identity-service/config"
	"identity-service/internal/constants"
	"identity-service/internal/domain/entity"
	"identity-service/internal/domain/ports"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

// memoryCounter is an in-memory [ports.AttemptCounter] whose windows never
// end on their own; resetIn is fixed.
type memoryCounter struct {
	counts  map[string]int
	resetIn time.Duration
	err     error
}

func newMemoryCounter() *memoryCounter {
	return &memoryCounter{counts: map[string]int{}, resetIn: 90 * time.Second}
}

func (m *memoryCounter) Attempts(_ context.Context, key string) (int, time.Duration, error) {
	if m.err != nil {
		return 0, 0, m.err
	}
	return m.counts[key], m.resetIn, nil
}

func (m *memoryCounter) AddAttempt(_ context.Context, key string, _ time.Duration) error {
	if m.err != nil {
		return m.err
	}
	m.counts[key]++
	return nil
}

func (m *memoryCounter) ResetAttempts(_ context.Context, key string) error {
	delete(m.counts, key)
	return nil
}

var testRateLimit = config.RateLimit{
	LoginPerEmail:  3,
	LoginPerIP:     5,
	LoginWindow:    15 * time.Minute,
	RegisterPerIP:  2,
	RegisterWindow: time.Hour,
}

func newThrottledRouter(svc ports.AuthService, counter ports.AttemptCounter) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	NewAuthHandler(svc, mockTokens{}, NewThrottle(counter, testRateLimit)).RegisterRoutes(r.Group("/api/v1"))
	return r
}

// loginService accepts one password, "right", for every email.
func loginService(calls *int) *mockAuthService {
	return &mockAuthService{loginFunc: func(_ context.Context, _, password string) (entity.TokenPair, error) {
		*calls++
		if password != "right" {
			return entity.TokenPair{}, ports.ErrInvalidCredentials
		}
		return entity.TokenPair{AccessToken: "a", AccessExpiresAt: time.Now().Add(time.Minute), RefreshToken: "r"}, nil
	}}
}

// doFrom is do with the request coming from remoteAddr ("ip:port").
func doFrom(r http.Handler, method, path, body string, header map[string]string, remoteAddr string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.RemoteAddr = remoteAddr
	req.Header.Set("Content-Type", "application/json")
	for k, v := range header {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func login(r http.Handler, email, password, remoteAddr string) int {
	body := `{"email":"` + email + `","password":"` + password + `"}`
	return doFrom(r, http.MethodPost, "/api/v1/auth/login", body, nil, remoteAddr).Code
}

func TestThrottle_LoginPerEmail(t *testing.T) {
	var calls int
	counter := newMemoryCounter()
	r := newThrottledRouter(loginService(&calls), counter)

	for i := range testRateLimit.LoginPerEmail {
		if code := login(r, "jane@example.com", "wrong", "10.0.0.1:1234"); code != http.StatusUnauthorized {
			t.Fatalf("attempt %d: code = %d, want 401", i+1, code)
		}
	}

	w := doFrom(r, http.MethodPost, "/api/v1/auth/login", `{"email":" Jane@Example.com ","password":"right"}`, nil, "10.0.0.2:1234")
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("locked email, even differently cased and from another IP: code = %d, want 429", w.Code)
	}
	if got := w.Header().Get("Retry-After"); got != strconv.Itoa(90) {
		t.Errorf("Retry-After = %q, want 90", got)
	}
	if calls != testRateLimit.LoginPerEmail {
		t.Errorf("service called %d times; a blocked login must not reach bcrypt", calls)
	}

	if code := login(r, "other@example.com", "right", "10.0.0.1:1234"); code != http.StatusOK {
		t.Errorf("another email from the same IP: code = %d, want 200", code)
	}
}

func TestThrottle_LoginSuccessResetsEmailNotIP(t *testing.T) {
	var calls int
	counter := newMemoryCounter()
	r := newThrottledRouter(loginService(&calls), counter)
	const ip = "10.0.0.1:1234"

	for range testRateLimit.LoginPerEmail - 1 {
		login(r, "jane@example.com", "wrong", ip)
	}
	if code := login(r, "jane@example.com", "right", ip); code != http.StatusOK {
		t.Fatalf("code = %d, want 200", code)
	}
	if n := counter.counts[loginEmailKey("jane@example.com")]; n != 0 {
		t.Errorf("email count after success = %d, want 0", n)
	}
	if n := counter.counts["login:ip:10.0.0.1"]; n != testRateLimit.LoginPerEmail-1 {
		t.Errorf("IP count after success = %d, want %d (success must not clear it)", n, testRateLimit.LoginPerEmail-1)
	}
}

func TestThrottle_LoginPerIP(t *testing.T) {
	var calls int
	r := newThrottledRouter(loginService(&calls), newMemoryCounter())
	const ip = "10.0.0.9:5555"

	// Spread over many emails so only the IP limit can trip.
	for i := range testRateLimit.LoginPerIP {
		login(r, "user"+strconv.Itoa(i)+"@example.com", "wrong", ip)
	}
	if code := login(r, "fresh@example.com", "right", ip); code != http.StatusTooManyRequests {
		t.Errorf("code = %d, want 429 once the IP is used up", code)
	}
	if code := login(r, "fresh@example.com", "right", "10.0.0.10:5555"); code != http.StatusOK {
		t.Errorf("another IP: code = %d, want 200", code)
	}
}

func TestThrottle_ForwardedForIsIgnoredWithoutTrustedProxies(t *testing.T) {
	var calls int
	counter := newMemoryCounter()
	r := newThrottledRouter(loginService(&calls), counter)
	if err := r.SetTrustedProxies(nil); err != nil {
		t.Fatal(err)
	}

	for i := range testRateLimit.LoginPerIP {
		doFrom(r, http.MethodPost, "/api/v1/auth/login",
			`{"email":"u`+strconv.Itoa(i)+`@example.com","password":"wrong"}`,
			map[string]string{"X-Forwarded-For": "203.0.113." + strconv.Itoa(i)}, "10.0.0.1:1234")
	}
	if code := login(r, "fresh@example.com", "right", "10.0.0.1:1234"); code != http.StatusTooManyRequests {
		t.Errorf("code = %d, want 429: a spoofed X-Forwarded-For must not reset the per-IP count", code)
	}
}

func TestThrottle_OnlyFailuresCount(t *testing.T) {
	inactive := &mockAuthService{loginFunc: func(context.Context, string, string) (entity.TokenPair, error) {
		return entity.TokenPair{}, ports.ErrUserInactive
	}}
	counter := newMemoryCounter()
	r := newThrottledRouter(inactive, counter)
	for range testRateLimit.LoginPerEmail + 1 {
		if code := login(r, "jane@example.com", "right", "10.0.0.1:1234"); code != http.StatusForbidden {
			t.Fatalf("code = %d, want 403: a right password on an inactive account isn't a guessing failure", code)
		}
	}
}

func TestThrottle_Register(t *testing.T) {
	svc := &mockAuthService{registerFunc: func(_ context.Context, reg entity.Register) (entity.User, error) {
		return entity.User{ID: "u", Email: reg.Email, Role: constants.RoleCustomer, Status: constants.UserActive}, nil
	}}
	r := newThrottledRouter(svc, newMemoryCounter())

	register := func(n int, ip string) int {
		body := `{"email":"user` + strconv.Itoa(n) + `@example.com","password":"s3cret-pass"}`
		return doFrom(r, http.MethodPost, "/api/v1/auth/register", body, nil, ip).Code
	}
	for i := range testRateLimit.RegisterPerIP {
		if code := register(i, "10.0.0.1:1"); code != http.StatusCreated {
			t.Fatalf("register %d: code = %d, want 201", i, code)
		}
	}
	if code := register(99, "10.0.0.1:1"); code != http.StatusTooManyRequests {
		t.Errorf("code = %d, want 429 after %d registrations from one IP", code, testRateLimit.RegisterPerIP)
	}
	if code := register(100, "10.0.0.2:1"); code != http.StatusCreated {
		t.Errorf("another IP: code = %d, want 201", code)
	}
}

func TestThrottle_FailsOpenWhenCounterErrors(t *testing.T) {
	var calls int
	counter := newMemoryCounter()
	counter.err = errors.New("redis: connection refused")
	r := newThrottledRouter(loginService(&calls), counter)
	if code := login(r, "jane@example.com", "right", "10.0.0.1:1234"); code != http.StatusOK {
		t.Errorf("code = %d, want 200 when the counter store is down", code)
	}
}
