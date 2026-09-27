package ports

import "errors"

var (
	// ErrNotFound is returned when a user does not exist.
	ErrNotFound = errors.New("user: not found")

	// ErrConflict is returned when a write violates a unique constraint
	// (an account with that email already exists).
	ErrConflict = errors.New("user: conflict")

	// ErrInvalidCredentials is returned by Login for both an unknown email
	// and a wrong password, so the response doesn't reveal which emails
	// have accounts.
	ErrInvalidCredentials = errors.New("auth: invalid email or password")

	// ErrInvalidToken is returned for an access or refresh token that is
	// malformed, expired, revoked, or already used.
	ErrInvalidToken = errors.New("auth: invalid or expired token")

	// ErrUserInactive is returned when the credentials are right but the
	// account is INACTIVE or LOCKED.
	ErrUserInactive = errors.New("auth: user is not active")

	// ErrRoleNotAllowed is returned when registration asks for a role that
	// can't be self-assigned (see constants.Role.SelfRegisterable).
	ErrRoleNotAllowed = errors.New("auth: role cannot be self-registered")
)
