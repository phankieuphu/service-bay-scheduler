package security

import (
	"errors"
	"identity-service/config"
	"identity-service/internal/constants"
	"identity-service/internal/domain/entity"
	"identity-service/internal/domain/ports"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const testSecret = "0123456789abcdef0123456789abcdef"

func newTestIssuer(t *testing.T, secret string) *JWTIssuer {
	t.Helper()
	issuer, err := NewJWTIssuer(config.JWT{Secret: secret, Issuer: "identity-service", AccessTTL: 15 * time.Minute})
	if err != nil {
		t.Fatalf("NewJWTIssuer: %v", err)
	}
	return issuer
}

var testUser = entity.User{ID: "7f1c9a52-3e1b-4b8a-9a44-1f0c2b6d8e10", Role: constants.RoleTechnician}

func TestNewJWTIssuer_RejectsShortSecret(t *testing.T) {
	if _, err := NewJWTIssuer(config.JWT{Secret: "too-short"}); err == nil {
		t.Error("expected an error for a short secret")
	}
}

func TestJWTIssuer_RoundTrip(t *testing.T) {
	issuer := newTestIssuer(t, testSecret)
	token, expiresAt, err := issuer.IssueAccessToken(testUser)
	if err != nil {
		t.Fatalf("IssueAccessToken: %v", err)
	}
	if d := time.Until(expiresAt); d < 14*time.Minute || d > 15*time.Minute {
		t.Errorf("expiresAt %v from now, want ~15m", d)
	}

	claims, err := issuer.ParseAccessToken(token)
	if err != nil {
		t.Fatalf("ParseAccessToken: %v", err)
	}
	if claims.UserID != testUser.ID || claims.Role != testUser.Role {
		t.Errorf("claims = %+v", claims)
	}
}

func TestJWTIssuer_RejectsBadTokens(t *testing.T) {
	issuer := newTestIssuer(t, testSecret)
	valid, _, _ := issuer.IssueAccessToken(testUser)

	expiredIssuer := newTestIssuer(t, testSecret)
	expiredIssuer.now = func() time.Time { return time.Now().Add(-time.Hour) }
	expired, _, _ := expiredIssuer.IssueAccessToken(testUser)

	otherSecret, _, _ := newTestIssuer(t, strings.Repeat("x", 32)).IssueAccessToken(testUser)

	otherIssuer := newTestIssuer(t, testSecret)
	otherIssuer.issuer = "someone-else"
	wrongIssuer, _, _ := otherIssuer.IssueAccessToken(testUser)

	unsigned, _ := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.RegisteredClaims{
		Subject: testUser.ID, Issuer: "identity-service", ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	}).SignedString(jwt.UnsafeAllowNoneSignatureType)

	// valid's signature over a different payload.
	v, e := strings.Split(valid, "."), strings.Split(wrongIssuer, ".")
	tampered := strings.Join([]string{v[0], e[1], v[2]}, ".")

	tests := map[string]string{
		"expired":      expired,
		"wrong secret": otherSecret,
		"wrong issuer": wrongIssuer,
		"alg none":     unsigned,
		"tampered":     tampered,
		"garbage":      "not.a.jwt",
		"empty":        "",
	}
	for name, token := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := issuer.ParseAccessToken(token); !errors.Is(err, ports.ErrInvalidToken) {
				t.Errorf("err = %v, want ErrInvalidToken", err)
			}
		})
	}
}

func TestBcryptHasher(t *testing.T) {
	h := NewBcryptHasher(4) // bcrypt.MinCost keeps the test fast
	hash, err := h.Hash("s3cret-pass")
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}
	if hash == "s3cret-pass" {
		t.Fatal("hash is the plain password")
	}
	if err := h.Compare(hash, "s3cret-pass"); err != nil {
		t.Errorf("Compare(correct): %v", err)
	}
	if err := h.Compare(hash, "wrong"); err == nil {
		t.Error("Compare(wrong) = nil")
	}
}
