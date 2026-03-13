package service

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/rzfd/expand/internal/model"
	"github.com/rzfd/expand/internal/pkg/apperror"
)

type fakeBudgetRepo struct {
	item model.Budget
}

func (f *fakeBudgetRepo) Create(_ context.Context, budget *model.Budget) error {
	budget.ID = 1
	f.item = *budget
	return nil
}

func (f *fakeBudgetRepo) GetByIDForUser(_ context.Context, id, userID int64) (*model.Budget, error) {
	if f.item.ID == 0 || f.item.ID != id || f.item.UserID != userID {
		return nil, nil
	}
	copy := f.item
	return &copy, nil
}

func (f *fakeBudgetRepo) ListByUser(_ context.Context, userID int64, year, month int) ([]model.Budget, error) {
	if f.item.ID == 0 || f.item.UserID != userID || f.item.Year != year || f.item.Month != month {
		return nil, nil
	}
	return []model.Budget{f.item}, nil
}

func (f *fakeBudgetRepo) Update(_ context.Context, budget *model.Budget) error {
	f.item = *budget
	return nil
}

func (f *fakeBudgetRepo) Delete(_ context.Context, _ int64, _ int64) (bool, error) {
	return true, nil
}

func TestBudgetServiceCreateValidation(t *testing.T) {
	repo := &fakeBudgetRepo{}
	categoryRepo := &fakeCategoryLookupRepo{
		category: &model.Category{
			ID:   10,
			Name: "Food",
			Type: model.TransactionTypeExpense,
		},
	}
	service := NewBudgetService(repo, categoryRepo)

	tests := []struct {
		name    string
		input   BudgetInput
		message string
	}{
		{
			name: "invalid category id",
			input: BudgetInput{
				CategoryID:  0,
				Year:        2026,
				Month:       3,
				AmountCents: 50000,
			},
			message: "category_id must be greater than zero",
		},
		{
			name: "invalid year",
			input: BudgetInput{
				CategoryID:  10,
				Year:        1999,
				Month:       3,
				AmountCents: 50000,
			},
			message: "year and month must be valid",
		},
		{
			name: "invalid amount",
			input: BudgetInput{
				CategoryID:  10,
				Year:        2026,
				Month:       3,
				AmountCents: 0,
			},
			message: "amount must be greater than zero",
		},
		{
			name: "category must be expense",
			input: BudgetInput{
				CategoryID:  10,
				Year:        2026,
				Month:       3,
				AmountCents: 50000,
			},
			message: "budget category must be an expense category",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "category must be expense" {
				categoryRepo.category.Type = model.TransactionTypeIncome
			} else {
				categoryRepo.category.Type = model.TransactionTypeExpense
			}

			_, err := service.Create(context.Background(), 1, tt.input)
			if err == nil {
				t.Fatalf("expected validation error")
			}

			var appErr *apperror.Error
			if !errors.As(err, &appErr) {
				t.Fatalf("expected app error, got %T", err)
			}
			if appErr.Status != http.StatusBadRequest {
				t.Fatalf("expected status 400, got %d", appErr.Status)
			}
			if appErr.Code != "validation_error" {
				t.Fatalf("expected code validation_error, got %s", appErr.Code)
			}
			if appErr.Message != tt.message {
				t.Fatalf("expected message %q, got %q", tt.message, appErr.Message)
			}
		})
	}
}
