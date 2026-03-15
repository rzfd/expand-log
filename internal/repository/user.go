package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/attribute"

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
	ctx, span := startRepositorySpan(ctx, "repository.user.create")
	defer span.End()

	logger := logging.FromContext(ctx)
	logger.Info().Str("email", user.Email).Msg("repository user create started")
	query := `
		INSERT INTO users (email, password_hash)
		VALUES ($1, $2)
		RETURNING id, created_at, updated_at
	`

	dbCtx, dbSpan := startDBSpan(ctx, "insert", attribute.String("db.table", "users"))
	setDBStatement(dbSpan, query)
	err := r.db.QueryRow(dbCtx, query, user.Email, user.PasswordHash).Scan(
		&user.ID,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		markSpanError(dbSpan, err, "insert user failed")
	} else {
		dbSpan.SetAttributes(attribute.Int64("app.user.id", user.ID))
	}
	dbSpan.End()
	if err != nil {
		markSpanError(span, err, "create user failed")
		logger.Error().Err(err).Str("email", user.Email).Msg("repository user create failed")
		return err
	}
	logger.Info().Int64("user_id", user.ID).Msg("repository user create completed")
	return nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	ctx, span := startRepositorySpan(ctx, "repository.user.get_by_email")
	defer span.End()

	logger := logging.FromContext(ctx)
	logger.Info().Str("email", email).Msg("repository user get by email started")
	query := `
		SELECT id, email, password_hash, created_at, updated_at
		FROM users
		WHERE email = $1
	`

	var user model.User
	dbCtx, dbSpan := startDBSpan(ctx, "select", attribute.String("db.table", "users"))
	setDBStatement(dbSpan, query)
	err := r.db.QueryRow(dbCtx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	defer dbSpan.End()
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			span.AddEvent("user_not_found")
			logger.Info().Str("email", email).Msg("repository user get by email not found")
			return nil, nil
		}
		markSpanError(dbSpan, err, "select user failed")
		markSpanError(span, err, "get user by email failed")
		logger.Error().Err(err).Str("email", email).Msg("repository user get by email failed")
		return nil, err
	}

	dbSpan.SetAttributes(attribute.Int64("app.user.id", user.ID))
	logger.Info().Int64("user_id", user.ID).Msg("repository user get by email completed")
	return &user, nil
}
