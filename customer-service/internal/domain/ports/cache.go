package ports

import (
	"context"
	"errors"
	"time"
)

// ErrCacheMiss is returned by Cache.Get when the key does not exist.
var ErrCacheMiss = errors.New("cache: miss")

type Cache interface {
	Set(ctx context.Context, key string, value any, ttl time.Duration) error
	Get(ctx context.Context, key string) (string, error)
	Delete(ctx context.Context, key string) error
}
