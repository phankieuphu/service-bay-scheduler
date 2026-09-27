package security

import (
	"errors"
	"identity-service/config"
	"identity-service/internal/constants"
	"identity-service/internal/domain/entity"
	"identity-service/internal/domain/ports"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// MinSecretLength is the shortest JWT secret accepted: HS256 keys shorter
// than the 256-bit hash output weaken the signature.
const MinSecretLength = 32

type accessClaims struct {
	Role constants.Role `json:"role"`
	jwt.RegisteredClaims
}

// JWTIssuer is the HS256 implementation of [ports.TokenIssuer].
type JWTIssuer struct {
	secret []byte
	issuer string
	ttl    time.Duration
	now    func() time.Time
}

func NewJWTIssuer(cfg config.JWT) (*JWTIssuer, error) {
	if len(cfg.Secret) < MinSecretLength {
		return nil, errors.New("JWT_SECRET must be at least 32 bytes")
	}
	return &JWTIssuer{
		secret: []byte(cfg.Secret),
		issuer: cfg.Issuer,
		ttl:    cfg.AccessTTL,
		now:    time.Now,
	}, nil
}

func (j *JWTIssuer) IssueAccessToken(user entity.User) (string, time.Time, error) {
	now := j.now()
	expiresAt := now.Add(j.ttl)
	claims := accessClaims{
		Role: user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID,
			Issuer:    j.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}

	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(j.secret)
	if err != nil {
		return "", time.Time{}, err
	}
	return token, expiresAt, nil
}

func (j *JWTIssuer) ParseAccessToken(token string) (entity.AccessClaims, error) {
	var claims accessClaims
	_, err := jwt.ParseWithClaims(token, &claims,
		func(*jwt.Token) (any, error) { return j.secret, nil },
		// Pinning the algorithm rejects "none" and alg-confusion tokens.
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(j.issuer),
		jwt.WithExpirationRequired(),
		jwt.WithTimeFunc(j.now),
	)
	if err != nil || claims.Subject == "" {
		return entity.AccessClaims{}, ports.ErrInvalidToken
	}

	return entity.AccessClaims{UserID: claims.Subject, Role: claims.Role}, nil
}
