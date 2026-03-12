package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rzfd/expand/internal/model"
	"github.com/rzfd/expand/internal/pkg/logging"
)

type UserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, user *model.User) error {
	logger := logging.FromContext(ctx)
	logger.Info().Str("email", user.Email).Msg("repository user create started")
	query := `
		INSERT INTO users (email, password_hash)
		VALUES ($1, $2)
		RETURNING id, created_at, updated_at
	`

	err := r.db.QueryRow(ctx, query, user.Email, user.PasswordHash).Scan(
		&user.ID,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		logger.Error().Err(err).Str("email", user.Email).Msg("repository user create failed")
		return err
	}
	logger.Info().Int64("user_id", user.ID).Msg("repository user create completed")
	return nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	logger := logging.FromContext(ctx)
	logger.Info().Str("email", email).Msg("repository user get by email started")
	query := `
		SELECT id, email, password_hash, created_at, updated_at
		FROM users
		WHERE email = $1
	`

	var user model.User
	err := r.db.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			logger.Info().Str("email", email).Msg("repository user get by email not found")
			return nil, nil
		}
		logger.Error().Err(err).Str("email", email).Msg("repository user get by email failed")
		return nil, err
	}

	logger.Info().Int64("user_id", user.ID).Msg("repository user get by email completed")
	return &user, nil
}
