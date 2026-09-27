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

// RedisCache is the Redis implementation of [ports.RefreshTokenStore].
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

func (r *RedisCache) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

func (r *RedisCache) Close() error {
	return r.client.Close()
}
