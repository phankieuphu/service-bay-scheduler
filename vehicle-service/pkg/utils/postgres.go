package utils

import (
	"errors"
	"vehicle-service/internal/constants"

	"github.com/jackc/pgx/v5/pgconn"
)

func IsDuplicateKeyError(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == constants.PgUniqueViolationCode
}

func IsForeignKeyError(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == constants.PgForeignKeyViolationCode
}

// ConstraintName returns the name of the constraint a Postgres error
// violated, or "" if err isn't a constraint violation.
func ConstraintName(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.ConstraintName
	}
	return ""
}
