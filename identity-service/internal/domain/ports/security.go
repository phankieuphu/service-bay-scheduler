package ports

import (
	"context"
	"identity-service/internal/domain/entity"
	"time"
)

type PasswordHasher interface {
	Hash(password string) (string, error)
	// Compare returns nil only if password matches hash.
	Compare(hash, password string) error
}

type TokenIssuer interface {
	IssueAccessToken(user entity.User) (token string, expiresAt time.Time, err error)
	// ParseAccessToken returns ErrInvalidToken for any token it didn't
	// issue or that has expired.
	ParseAccessToken(token string) (entity.AccessClaims, error)
}

// RefreshTokenStore keeps refresh tokens server-side so they can be
// revoked. Implementations key on a hash of the token, never the token
// itself, so a leaked store doesn't leak usable tokens.
type RefreshTokenStore interface {
	Save(ctx context.Context, tokenHash, userID string, ttl time.Duration) error
	// Consume atomically looks up and deletes tokenHash, returning the user
	// it belonged to, or ErrInvalidToken if it's unknown or already used.
	Consume(ctx context.Context, tokenHash string) (userID string, err error)
	Delete(ctx context.Context, tokenHash string) error
}
