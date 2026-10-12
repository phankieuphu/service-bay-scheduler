package cache

import (
	"context"
	"identity-service/config"
	"os"
	"strings"
	"testing"
	"time"
)

// newTestRedis connects to REDIS_TEST_ADDR ("host:port"), or skips: these
// tests need a real Redis for the Lua script and key expiry.
func newTestRedis(t *testing.T) *RedisCache {
	t.Helper()
	addr := os.Getenv("REDIS_TEST_ADDR")
	if addr == "" {
		t.Skip("REDIS_TEST_ADDR not set")
	}
	host, port, _ := strings.Cut(addr, ":")
	r, err := NewRedisCache(config.Redis{Host: host, Port: port})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = r.Close() })
	return r
}

func TestRedisCache_AttemptCounter(t *testing.T) {
	r := newTestRedis(t)
	ctx := context.Background()
	key := "test:" + t.Name() + ":" + time.Now().Format(time.RFC3339Nano)
	t.Cleanup(func() { _ = r.ResetAttempts(ctx, key) })

	if n, resetIn, err := r.Attempts(ctx, key); err != nil || n != 0 || resetIn != 0 {
		t.Fatalf("unknown key: got %d, %v, %v; want 0, 0, nil", n, resetIn, err)
	}

	for range 3 {
		if err := r.AddAttempt(ctx, key, time.Minute); err != nil {
			t.Fatal(err)
		}
	}
	n, resetIn, err := r.Attempts(ctx, key)
	if err != nil || n != 3 {
		t.Fatalf("got %d, %v; want 3, nil", n, err)
	}
	if resetIn <= 50*time.Second || resetIn > time.Minute {
		t.Errorf("resetIn = %v, want just under 1m (the window starts at the first attempt)", resetIn)
	}

	if err := r.ResetAttempts(ctx, key); err != nil {
		t.Fatal(err)
	}
	if n, _, _ := r.Attempts(ctx, key); n != 0 {
		t.Errorf("after reset: %d, want 0", n)
	}
}

func TestRedisCache_AttemptWindowExpires(t *testing.T) {
	r := newTestRedis(t)
	ctx := context.Background()
	key := "test:" + t.Name() + ":" + time.Now().Format(time.RFC3339Nano)

	if err := r.AddAttempt(ctx, key, 200*time.Millisecond); err != nil {
		t.Fatal(err)
	}
	time.Sleep(300 * time.Millisecond)
	if n, _, _ := r.Attempts(ctx, key); n != 0 {
		t.Errorf("after the window: %d, want 0", n)
	}
}
