package handler

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// defaultHealthCheckTimeout bounds each dependency check, so a hung
// dependency makes /readyz fail fast instead of stalling the probe.
const defaultHealthCheckTimeout = 2 * time.Second

// HealthCheck probes one dependency.
type HealthCheck struct {
	Name string
	// Critical checks gate readiness: if one fails, /readyz returns 503 and
	// the pod is taken out of the load balancer. Non-critical checks are
	// only reported, for dependencies the service keeps working without
	// (Kafka — writes queue in the outbox; Redis — cache falls through).
	Critical bool
	Check    func(ctx context.Context) error
}

type HealthHandler struct {
	checks  []HealthCheck
	timeout time.Duration
}

func NewHealthHandler(checks ...HealthCheck) *HealthHandler {
	return &HealthHandler{checks: checks, timeout: defaultHealthCheckTimeout}
}

func (h *HealthHandler) RegisterRoutes(r gin.IRoutes) {
	r.GET("/healthz", h.Liveness)
	r.GET("/readyz", h.Readiness)
}

// Liveness reports only that the process is up and serving HTTP. It must
// not check dependencies: a Postgres outage would otherwise make the
// orchestrator restart every replica, which fixes nothing.
func (h *HealthHandler) Liveness(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

type checkResult struct {
	Status   string `json:"status"`
	Critical bool   `json:"critical"`
	Error    string `json:"error,omitempty"`
}

// Readiness runs every check concurrently and returns 503 if any critical
// one fails.
func (h *HealthHandler) Readiness(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), h.timeout)
	defer cancel()

	results := make(map[string]checkResult, len(h.checks))
	var mu sync.Mutex
	var wg sync.WaitGroup
	for _, check := range h.checks {
		wg.Go(func() {
			res := checkResult{Status: "up", Critical: check.Critical}
			if err := runCheck(ctx, check.Check); err != nil {
				res.Status, res.Error = "down", err.Error()
			}
			mu.Lock()
			results[check.Name] = res
			mu.Unlock()
		})
	}
	wg.Wait()

	ready := true
	for _, res := range results {
		if res.Critical && res.Status != "up" {
			ready = false
		}
	}

	status, code := "ready", http.StatusOK
	if !ready {
		status, code = "not_ready", http.StatusServiceUnavailable
	}
	c.JSON(code, gin.H{"status": status, "checks": results})
}

// runCheck enforces ctx's deadline even on checks that ignore ctx (sarama's
// client calls take no context); such a check is abandoned, not awaited.
func runCheck(ctx context.Context, check func(ctx context.Context) error) error {
	done := make(chan error, 1)
	go func() { done <- check(ctx) }()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}
