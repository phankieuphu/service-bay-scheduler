package security

import (
	"customer-service/config"
	"customer-service/internal/domain/entity"
	"errors"

	"github.com/golang-jwt/jwt/v5"
)

// MinSecretLength matches identity-service: HS256 keys shorter than the
// 256-bit hash output weaken the signature.
const MinSecretLength = 32

// ErrInvalidToken is returned for any token that identity-service didn't
// issue, or that has expired.
var ErrInvalidToken = errors.New("auth: invalid or expired token")

// accessClaims mirrors identity-service's access token claims.
type accessClaims struct {
	Role string `json:"role"`
	jwt.RegisteredClaims
}

// JWTVerifier checks identity-service's HS256 access tokens. It only
// verifies; issuing stays in identity-service, which shares the secret.
type JWTVerifier struct {
	secret []byte
	issuer string
}

func NewJWTVerifier(cfg config.JWT) (*JWTVerifier, error) {
	if len(cfg.Secret) < MinSecretLength {
		return nil, errors.New("JWT_SECRET must be at least 32 bytes and match identity-service's")
	}
	return &JWTVerifier{secret: []byte(cfg.Secret), issuer: cfg.Issuer}, nil
}

func (j *JWTVerifier) ParseAccessToken(token string) (entity.Principal, error) {
	var claims accessClaims
	_, err := jwt.ParseWithClaims(token, &claims,
		func(*jwt.Token) (any, error) { return j.secret, nil },
		// Pinning the algorithm rejects "none" and alg-confusion tokens.
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(j.issuer),
		jwt.WithExpirationRequired(),
	)
	if err != nil || claims.Subject == "" {
		return entity.Principal{}, ErrInvalidToken
	}
	return entity.Principal{UserID: claims.Subject, Role: entity.Role(claims.Role)}, nil
}
