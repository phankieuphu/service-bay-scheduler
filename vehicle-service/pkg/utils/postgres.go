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
