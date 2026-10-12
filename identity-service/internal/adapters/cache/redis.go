package cache

import (
	"context"
	"errors"
	"identity-service/config"
	"identity-service/internal/domain/ports"
	"identity-service/pkg/logger"
	"time"

	"github.com/redis/go-redis/v9"
)

// refreshTokenKeyPrefix namespaces refresh tokens in Redis.
const refreshTokenKeyPrefix = "refresh:"

// attemptKeyPrefix namespaces rate-limit counters in Redis.
const attemptKeyPrefix = "attempts:"

// addAttemptScript increments a counter and, on its first increment, sets
// the window expiry: in one round trip, and atomically, so a crash between
// the two can't leave a counter that never expires.
var addAttemptScript = redis.NewScript(`
local n = redis.call('INCR', KEYS[1])
if n == 1 then
  redis.call('PEXPIRE', KEYS[1], ARGV[1])
end
return n
`)

// RedisCache is the Redis implementation of [ports.RefreshTokenStore] and
// [ports.AttemptCounter].
type RedisCache struct {
	client *redis.Client
}

func NewRedisCache(cfg config.Redis) (*RedisCache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr(),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, err
	}

	logger.Info("Redis connected", "addr", cfg.Addr())
	return &RedisCache{client: client}, nil
}

// Save implements [ports.RefreshTokenStore].
func (r *RedisCache) Save(ctx context.Context, tokenHash, userID string, ttl time.Duration) error {
	return r.client.Set(ctx, refreshTokenKeyPrefix+tokenHash, userID, ttl).Err()
}

// Consume implements [ports.RefreshTokenStore]. GETDEL is a single atomic
// command, so two concurrent refreshes with the same token can't both win.
func (r *RedisCache) Consume(ctx context.Context, tokenHash string) (string, error) {
	userID, err := r.client.GetDel(ctx, refreshTokenKeyPrefix+tokenHash).Result()
	if errors.Is(err, redis.Nil) {
		return "", ports.ErrInvalidToken
	}
	return userID, err
}

// Delete implements [ports.RefreshTokenStore].
func (r *RedisCache) Delete(ctx context.Context, tokenHash string) error {
	return r.client.Del(ctx, refreshTokenKeyPrefix+tokenHash).Err()
}

// Attempts implements [ports.AttemptCounter].
func (r *RedisCache) Attempts(ctx context.Context, key string) (int, time.Duration, error) {
	pipe := r.client.Pipeline()
	get := pipe.Get(ctx, attemptKeyPrefix+key)
	ttl := pipe.PTTL(ctx, attemptKeyPrefix+key)
	if _, err := pipe.Exec(ctx); err != nil && !errors.Is(err, redis.Nil) {
		return 0, 0, err
	}
	count, err := get.Int()
	if errors.Is(err, redis.Nil) {
		return 0, 0, nil
	}
	if err != nil {
		return 0, 0, err
	}
	// PTTL is negative for a missing key or one without an expiry.
	return count, max(ttl.Val(), 0), nil
}

// AddAttempt implements [ports.AttemptCounter].
func (r *RedisCache) AddAttempt(ctx context.Context, key string, window time.Duration) error {
	return addAttemptScript.Run(ctx, r.client, []string{attemptKeyPrefix + key}, window.Milliseconds()).Err()
}

// ResetAttempts implements [ports.AttemptCounter].
func (r *RedisCache) ResetAttempts(ctx context.Context, key string) error {
	return r.client.Del(ctx, attemptKeyPrefix+key).Err()
}

func (r *RedisCache) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

func (r *RedisCache) Close() error {
	return r.client.Close()
}
