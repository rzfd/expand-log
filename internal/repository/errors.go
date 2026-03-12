package repository

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/rzfd/expand/internal/pkg/logging"
)

func IsUniqueViolation(err error) bool {
	logging.FromContext(nil).Info().Msg("repository check unique violation")
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func IsForeignKeyViolation(err error) bool {
	logging.FromContext(nil).Info().Msg("repository check foreign key violation")
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503"
}
