package repository

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

func IsUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func IsForeignKeyViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503"
}

func PgErrorCode(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code
	}
	return ""
}

func PgErrorKind(err error) string {
	code := PgErrorCode(err)
	switch code {
	case "23505":
		return "unique_violation"
	case "23503":
		return "foreign_key_violation"
	case "":
		return ""
	default:
		return "database_error"
	}
}
