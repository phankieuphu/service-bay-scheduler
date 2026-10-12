package handler

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"identity-service/config"
	"identity-service/internal/domain/ports"
	"identity-service/pkg/logger"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// Throttle rate-limits login and registration (see [config.RateLimit]).
// If the counter store fails, requests are let through and the error is
// logged: Redis being down already breaks login (refresh tokens live
// there), and a limiter bug shouldn't lock everyone out on its own.
type Throttle struct {
	counter ports.AttemptCounter
	cfg     config.RateLimit
}

func NewThrottle(counter ports.AttemptCounter, cfg config.RateLimit) *Throttle {
	return &Throttle{counter: counter, cfg: cfg}
}

type limit struct {
	key string
	max int
}

func (t *Throttle) loginLimits(ip, email string) []limit {
	return []limit{
		{key: "login:ip:" + ip, max: t.cfg.LoginPerIP},
		{key: loginEmailKey(email), max: t.cfg.LoginPerEmail},
	}
}

func (t *Throttle) registerLimits(ip string) []limit {
	return []limit{{key: "register:ip:" + ip, max: t.cfg.RegisterPerIP}}
}

// loginEmailKey hashes the normalized email, so the counters don't put
// addresses in Redis.
func loginEmailKey(email string) string {
	sum := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(email))))
	return "login:email:" + hex.EncodeToString(sum[:])
}

// blocked reports whether any of limits is used up, and if so how long
// until the longest of those windows ends. A max of 0 or less disables
// that limit.
func (t *Throttle) blocked(ctx context.Context, limits []limit) (time.Duration, bool) {
	var wait time.Duration
	var isBlocked bool
	for _, l := range limits {
		if l.max <= 0 {
			continue
		}
		count, resetIn, err := t.counter.Attempts(ctx, l.key)
		if err != nil {
			logger.ErrorContext(ctx, "rate limit check failed", "error", err)
			continue
		}
		if count >= l.max {
			isBlocked = true
			wait = max(wait, resetIn)
		}
	}
	return wait, isBlocked
}

func (t *Throttle) add(ctx context.Context, limits []limit, window time.Duration) {
	for _, l := range limits {
		if l.max <= 0 {
			continue
		}
		if err := t.counter.AddAttempt(ctx, l.key, window); err != nil {
			logger.ErrorContext(ctx, "rate limit count failed", "error", err)
		}
	}
}

func (t *Throttle) reset(ctx context.Context, key string) {
	if err := t.counter.ResetAttempts(ctx, key); err != nil {
		logger.ErrorContext(ctx, "rate limit reset failed", "error", err)
	}
}

// tooManyRequests writes 429 with Retry-After in whole seconds (at least 1).
func tooManyRequests(c *gin.Context, wait time.Duration) {
	seconds := max(int(math.Ceil(wait.Seconds())), 1)
	c.Header("Retry-After", strconv.Itoa(seconds))
	c.JSON(http.StatusTooManyRequests, gin.H{"error": "too many attempts, try again later"})
}
