package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rzfd/expand/internal/model"
	"github.com/rzfd/expand/internal/pkg/logging"
)

type CategoryRepository struct {
	db *pgxpool.Pool
}

func NewCategoryRepository(db *pgxpool.Pool) *CategoryRepository {
	return &CategoryRepository{db: db}
}

func (r *CategoryRepository) Create(ctx context.Context, category *model.Category) error {
	logger := logging.FromContext(ctx)
	logger.Info().Int64("user_id", category.UserID).Msg("repository category create started")
	query := `
		INSERT INTO categories (user_id, name, type)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at
	`

	err := r.db.QueryRow(ctx, query, category.UserID, category.Name, category.Type).Scan(
		&category.ID,
		&category.CreatedAt,
		&category.UpdatedAt,
	)
	if err != nil {
		logger.Error().Err(err).Int64("user_id", category.UserID).Msg("repository category create failed")
		return err
	}
	logger.Info().Int64("user_id", category.UserID).Int64("category_id", category.ID).Msg("repository category create completed")
	return nil
}

func (r *CategoryRepository) ListByUser(ctx context.Context, userID int64) ([]model.Category, error) {
	logger := logging.FromContext(ctx)
	logger.Info().Int64("user_id", userID).Msg("repository category list by user started")
	query := `
		SELECT id, user_id, name, type, created_at, updated_at
		FROM categories
		WHERE user_id = $1
		ORDER BY created_at DESC, id DESC
	`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		logger.Error().Err(err).Int64("user_id", userID).Msg("repository category list by user query failed")
		return nil, err
	}
	defer rows.Close()

	categories := make([]model.Category, 0)
	for rows.Next() {
		var category model.Category
		if err := rows.Scan(
			&category.ID,
			&category.UserID,
			&category.Name,
			&category.Type,
			&category.CreatedAt,
			&category.UpdatedAt,
		); err != nil {
			logger.Error().Err(err).Int64("user_id", userID).Msg("repository category list by user scan failed")
			return nil, err
		}
		categories = append(categories, category)
	}

	if err := rows.Err(); err != nil {
		logger.Error().Err(err).Int64("user_id", userID).Msg("repository category list by user rows failed")
		return nil, err
	}
	logger.Info().Int64("user_id", userID).Int("count", len(categories)).Msg("repository category list by user completed")
	return categories, nil
}

func (r *CategoryRepository) GetByIDForUser(ctx context.Context, id, userID int64) (*model.Category, error) {
	logger := logging.FromContext(ctx)
	logger.Info().Int64("user_id", userID).Int64("category_id", id).Msg("repository category get by id started")
	query := `
		SELECT id, user_id, name, type, created_at, updated_at
		FROM categories
		WHERE id = $1 AND user_id = $2
	`

	var category model.Category
	err := r.db.QueryRow(ctx, query, id, userID).Scan(
		&category.ID,
		&category.UserID,
		&category.Name,
		&category.Type,
		&category.CreatedAt,
		&category.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			logger.Info().Int64("user_id", userID).Int64("category_id", id).Msg("repository category get by id not found")
			return nil, nil
		}
		logger.Error().Err(err).Int64("user_id", userID).Int64("category_id", id).Msg("repository category get by id failed")
		return nil, err
	}

	logger.Info().Int64("user_id", userID).Int64("category_id", category.ID).Msg("repository category get by id completed")
	return &category, nil
}

func (r *CategoryRepository) Update(ctx context.Context, category *model.Category) error {
	logger := logging.FromContext(ctx)
	logger.Info().Int64("user_id", category.UserID).Int64("category_id", category.ID).Msg("repository category update started")
	query := `
		UPDATE categories
		SET name = $1, type = $2, updated_at = NOW()
		WHERE id = $3 AND user_id = $4
		RETURNING updated_at
	`

	err := r.db.QueryRow(ctx, query, category.Name, category.Type, category.ID, category.UserID).Scan(&category.UpdatedAt)
	if err != nil {
		logger.Error().Err(err).Int64("user_id", category.UserID).Int64("category_id", category.ID).Msg("repository category update failed")
		return err
	}
	logger.Info().Int64("user_id", category.UserID).Int64("category_id", category.ID).Msg("repository category update completed")
	return nil
}

func (r *CategoryRepository) Delete(ctx context.Context, id, userID int64) (bool, error) {
	logger := logging.FromContext(ctx)
	logger.Info().Int64("user_id", userID).Int64("category_id", id).Msg("repository category delete started")
	result, err := r.db.Exec(ctx, `DELETE FROM categories WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		logger.Error().Err(err).Int64("user_id", userID).Int64("category_id", id).Msg("repository category delete failed")
		return false, err
	}
	deleted := result.RowsAffected() > 0
	logger.Info().Int64("user_id", userID).Int64("category_id", id).Bool("deleted", deleted).Msg("repository category delete completed")
	return deleted, nil
}
