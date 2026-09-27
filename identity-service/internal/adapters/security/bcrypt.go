package security

import (
	"identity-service/internal/domain/ports"

	"golang.org/x/crypto/bcrypt"
)

// BcryptHasher is the bcrypt implementation of [ports.PasswordHasher].
type BcryptHasher struct {
	cost int
}

func NewBcryptHasher(cost int) ports.PasswordHasher {
	return BcryptHasher{cost: cost}
}

func (h BcryptHasher) Hash(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), h.cost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func (h BcryptHasher) Compare(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}
