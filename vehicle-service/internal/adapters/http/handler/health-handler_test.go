package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func serveReadyz(h *HealthHandler) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	h.RegisterRoutes(engine)

	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	return rec
}

func up(context.Context) error   { return nil }
func down(context.Context) error { return errors.New("connection refused") }

func TestReadiness(t *testing.T) {
	tests := []struct {
		name   string
		checks []HealthCheck
		want   int
	}{
		{
			name: "all up",
			checks: []HealthCheck{
				{Name: "postgres", Critical: true, Check: up},
				{Name: "kafka", Check: up},
			},
			want: http.StatusOK,
		},
		{
			name: "critical down",
			checks: []HealthCheck{
				{Name: "postgres", Critical: true, Check: down},
				{Name: "kafka", Check: up},
			},
			want: http.StatusServiceUnavailable,
		},
		{
			name: "non-critical down stays ready",
			checks: []HealthCheck{
				{Name: "postgres", Critical: true, Check: up},
				{Name: "kafka", Check: down},
			},
			want: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := serveReadyz(NewHealthHandler(tt.checks...))
			if rec.Code != tt.want {
				t.Fatalf("status = %d, want %d; body: %s", rec.Code, tt.want, rec.Body)
			}
		})
	}
}

func TestReadiness_HungCheckTimesOut(t *testing.T) {
	block := make(chan struct{})
	defer close(block)

	h := NewHealthHandler(HealthCheck{
		Name:     "postgres",
		Critical: true,
		// Ignores ctx, like sarama's client calls do.
		Check: func(context.Context) error { <-block; return nil },
	})
	h.timeout = 50 * time.Millisecond

	start := time.Now()
	rec := serveReadyz(h)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("readyz took %v, want it bounded by the check timeout", elapsed)
	}
}
