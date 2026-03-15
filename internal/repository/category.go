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

type CategoryRepository struct {
	db *pgxpool.Pool
}

func NewCategoryRepository(db *pgxpool.Pool) *CategoryRepository {
	return &CategoryRepository{db: db}
}

func (r *CategoryRepository) Create(ctx context.Context, category *model.Category) error {
	ctx, span := startRepositorySpan(ctx, "repository.category.create",
		attribute.Int64("app.user.id", category.UserID),
	)
	defer span.End()

	logger := logging.FromContext(ctx)
	logger.Info().Int64("user_id", category.UserID).Msg("repository category create started")
	query := `
		INSERT INTO categories (user_id, name, type)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at
	`

	dbCtx, dbSpan := startDBSpan(ctx, "insert", attribute.String("db.table", "categories"))
	setDBStatement(dbSpan, query)
	err := r.db.QueryRow(dbCtx, query, category.UserID, category.Name, category.Type).Scan(
		&category.ID,
		&category.CreatedAt,
		&category.UpdatedAt,
	)
	if err != nil {
		markSpanError(dbSpan, err, "insert category failed")
	} else {
		dbSpan.SetAttributes(attribute.Int64("app.category.id", category.ID))
	}
	dbSpan.End()
	if err != nil {
		markSpanError(span, err, "create category failed")
		logger.Error().Err(err).Int64("user_id", category.UserID).Msg("repository category create failed")
		return err
	}
	logger.Info().Int64("user_id", category.UserID).Int64("category_id", category.ID).Msg("repository category create completed")
	return nil
}

func (r *CategoryRepository) ListByUser(ctx context.Context, userID int64) ([]model.Category, error) {
	ctx, span := startRepositorySpan(ctx, "repository.category.list_by_user",
		attribute.Int64("app.user.id", userID),
	)
	defer span.End()

	logger := logging.FromContext(ctx)
	logger.Info().Int64("user_id", userID).Msg("repository category list by user started")
	query := `
		SELECT id, user_id, name, type, created_at, updated_at
		FROM categories
		WHERE user_id = $1
		ORDER BY created_at DESC, id DESC
	`

	dbCtx, dbSpan := startDBSpan(ctx, "select", attribute.String("db.table", "categories"))
	setDBStatement(dbSpan, query)
	rows, err := r.db.Query(dbCtx, query, userID)
	if err != nil {
		markSpanError(dbSpan, err, "query categories failed")
		dbSpan.End()
		markSpanError(span, err, "list categories failed")
		logger.Error().Err(err).Int64("user_id", userID).Msg("repository category list by user query failed")
		return nil, err
	}
	defer rows.Close()
	defer dbSpan.End()

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
			markSpanError(dbSpan, err, "scan categories failed")
			markSpanError(span, err, "scan categories failed")
			logger.Error().Err(err).Int64("user_id", userID).Msg("repository category list by user scan failed")
			return nil, err
		}
		categories = append(categories, category)
	}

	if err := rows.Err(); err != nil {
		markSpanError(dbSpan, err, "rows categories failed")
		markSpanError(span, err, "rows categories failed")
		logger.Error().Err(err).Int64("user_id", userID).Msg("repository category list by user rows failed")
		return nil, err
	}
	dbSpan.SetAttributes(attribute.Int("app.category.count", len(categories)))
	logger.Info().Int64("user_id", userID).Int("count", len(categories)).Msg("repository category list by user completed")
	return categories, nil
}

func (r *CategoryRepository) GetByIDForUser(ctx context.Context, id, userID int64) (*model.Category, error) {
	ctx, span := startRepositorySpan(ctx, "repository.category.get_by_id_for_user",
		attribute.Int64("app.user.id", userID),
		attribute.Int64("app.category.id", id),
	)
	defer span.End()

	logger := logging.FromContext(ctx)
	logger.Info().Int64("user_id", userID).Int64("category_id", id).Msg("repository category get by id started")
	query := `
		SELECT id, user_id, name, type, created_at, updated_at
		FROM categories
		WHERE id = $1 AND user_id = $2
	`

	var category model.Category
	dbCtx, dbSpan := startDBSpan(ctx, "select", attribute.String("db.table", "categories"))
	setDBStatement(dbSpan, query)
	err := r.db.QueryRow(dbCtx, query, id, userID).Scan(
		&category.ID,
		&category.UserID,
		&category.Name,
		&category.Type,
		&category.CreatedAt,
		&category.UpdatedAt,
	)
	defer dbSpan.End()
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			dbSpan.SetAttributes(attribute.Bool("app.category.found", false))
			span.SetAttributes(attribute.Bool("app.category.found", false))
			logger.Info().Int64("user_id", userID).Int64("category_id", id).Msg("repository category get by id not found")
			return nil, nil
		}
		markSpanError(dbSpan, err, "select category failed")
		markSpanError(span, err, "get category failed")
		logger.Error().Err(err).Int64("user_id", userID).Int64("category_id", id).Msg("repository category get by id failed")
		return nil, err
	}

	dbSpan.SetAttributes(attribute.Bool("app.category.found", true))
	span.SetAttributes(attribute.Bool("app.category.found", true))
	logger.Info().Int64("user_id", userID).Int64("category_id", category.ID).Msg("repository category get by id completed")
	return &category, nil
}

func (r *CategoryRepository) Update(ctx context.Context, category *model.Category) error {
	ctx, span := startRepositorySpan(ctx, "repository.category.update",
		attribute.Int64("app.user.id", category.UserID),
		attribute.Int64("app.category.id", category.ID),
	)
	defer span.End()

	logger := logging.FromContext(ctx)
	logger.Info().Int64("user_id", category.UserID).Int64("category_id", category.ID).Msg("repository category update started")
	query := `
		UPDATE categories
		SET name = $1, type = $2, updated_at = NOW()
		WHERE id = $3 AND user_id = $4
		RETURNING updated_at
	`

	dbCtx, dbSpan := startDBSpan(ctx, "update", attribute.String("db.table", "categories"))
	setDBStatement(dbSpan, query)
	err := r.db.QueryRow(dbCtx, query, category.Name, category.Type, category.ID, category.UserID).Scan(&category.UpdatedAt)
	if err != nil {
		markSpanError(dbSpan, err, "update category failed")
	}
	dbSpan.End()
	if err != nil {
		markSpanError(span, err, "update category failed")
		logger.Error().Err(err).Int64("user_id", category.UserID).Int64("category_id", category.ID).Msg("repository category update failed")
		return err
	}
	logger.Info().Int64("user_id", category.UserID).Int64("category_id", category.ID).Msg("repository category update completed")
	return nil
}

func (r *CategoryRepository) Delete(ctx context.Context, id, userID int64) (bool, error) {
	ctx, span := startRepositorySpan(ctx, "repository.category.delete",
		attribute.Int64("app.user.id", userID),
		attribute.Int64("app.category.id", id),
	)
	defer span.End()

	logger := logging.FromContext(ctx)
	logger.Info().Int64("user_id", userID).Int64("category_id", id).Msg("repository category delete started")
	dbCtx, dbSpan := startDBSpan(ctx, "delete", attribute.String("db.table", "categories"))
	statement := `DELETE FROM categories WHERE id = $1 AND user_id = $2`
	setDBStatement(dbSpan, statement)
	result, err := r.db.Exec(dbCtx, statement, id, userID)
	if err != nil {
		markSpanError(dbSpan, err, "delete category failed")
		dbSpan.End()
		markSpanError(span, err, "delete category failed")
		logger.Error().Err(err).Int64("user_id", userID).Int64("category_id", id).Msg("repository category delete failed")
		return false, err
	}
	deleted := result.RowsAffected() > 0
	dbSpan.SetAttributes(attribute.Int64("db.rows_affected", result.RowsAffected()))
	dbSpan.End()
	logger.Info().Int64("user_id", userID).Int64("category_id", id).Bool("deleted", deleted).Msg("repository category delete completed")
	return deleted, nil
}
