package security

import (
	"errors"
	"scheduler-service/config"
	"scheduler-service/internal/domain/entity"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const testSecret = "test-secret-0123456789-0123456789-abcdef"

// sign builds a token the way identity-service does.
func sign(t *testing.T, method jwt.SigningMethod, key any, claims jwt.MapClaims) string {
	t.Helper()
	token, err := jwt.NewWithClaims(method, claims).SignedString(key)
	if err != nil {
		t.Fatal(err)
	}
	return token
}

func validClaims() jwt.MapClaims {
	return jwt.MapClaims{
		"sub":  "7f1c9a52-3e1b-4b8a-9a44-1f0c2b6d8e10",
		"role": "TECHNICIAN",
		"iss":  "identity-service",
		"iat":  time.Now().Unix(),
		"exp":  time.Now().Add(15 * time.Minute).Unix(),
	}
}

func TestJWTVerifier(t *testing.T) {
	v, err := NewJWTVerifier(config.JWT{Secret: testSecret, Issuer: "identity-service"})
	if err != nil {
		t.Fatal(err)
	}

	p, err := v.ParseAccessToken(sign(t, jwt.SigningMethodHS256, []byte(testSecret), validClaims()))
	if err != nil {
		t.Fatalf("valid token: %v", err)
	}
	if p.UserID != "7f1c9a52-3e1b-4b8a-9a44-1f0c2b6d8e10" || p.Role != entity.RoleTechnician {
		t.Errorf("principal = %+v", p)
	}

	with := func(k string, v any) jwt.MapClaims {
		c := validClaims()
		if v == nil {
			delete(c, k)
		} else {
			c[k] = v
		}
		return c
	}
	bad := map[string]string{
		"expired":      sign(t, jwt.SigningMethodHS256, []byte(testSecret), with("exp", time.Now().Add(-time.Minute).Unix())),
		"no exp":       sign(t, jwt.SigningMethodHS256, []byte(testSecret), with("exp", nil)),
		"wrong issuer": sign(t, jwt.SigningMethodHS256, []byte(testSecret), with("iss", "someone-else")),
		"no subject":   sign(t, jwt.SigningMethodHS256, []byte(testSecret), with("sub", nil)),
		"wrong secret": sign(t, jwt.SigningMethodHS256, []byte("another-secret-0123456789-0123456789"), validClaims()),
		"alg none":     sign(t, jwt.SigningMethodNone, jwt.UnsafeAllowNoneSignatureType, validClaims()),
		"HS512":        sign(t, jwt.SigningMethodHS512, []byte(testSecret), validClaims()),
		"garbage":      "not.a.jwt",
	}
	for name, token := range bad {
		t.Run(name, func(t *testing.T) {
			if _, err := v.ParseAccessToken(token); !errors.Is(err, ErrInvalidToken) {
				t.Errorf("err = %v, want ErrInvalidToken", err)
			}
		})
	}
}

func TestNewJWTVerifier_RejectsShortSecret(t *testing.T) {
	if _, err := NewJWTVerifier(config.JWT{Secret: "short"}); err == nil {
		t.Error("expected an error for a secret under 32 bytes")
	}
}
