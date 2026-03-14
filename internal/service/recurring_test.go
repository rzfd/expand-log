package service

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/rzfd/expand/internal/model"
	"github.com/rzfd/expand/internal/pkg/apperror"
)

type fakeRecurringRepo struct {
	item model.RecurringTransaction
}

func (f *fakeRecurringRepo) Create(_ context.Context, item *model.RecurringTransaction) error {
	item.ID = 1
	f.item = *item
	return nil
}

func (f *fakeRecurringRepo) GetByIDForUser(_ context.Context, id, userID int64) (*model.RecurringTransaction, error) {
	if f.item.ID == 0 || f.item.ID != id || f.item.UserID != userID {
		return nil, nil
	}
	copy := f.item
	return &copy, nil
}

func (f *fakeRecurringRepo) ListByUser(_ context.Context, userID int64) ([]model.RecurringTransaction, error) {
	if f.item.ID == 0 || f.item.UserID != userID {
		return nil, nil
	}
	return []model.RecurringTransaction{f.item}, nil
}

func (f *fakeRecurringRepo) Update(_ context.Context, item *model.RecurringTransaction) error {
	f.item = *item
	return nil
}

func (f *fakeRecurringRepo) Delete(_ context.Context, _ int64, _ int64) (bool, error) {
	return true, nil
}

func TestRecurringServiceCreateValidation(t *testing.T) {
	repo := &fakeRecurringRepo{}
	categoryRepo := &fakeCategoryLookupRepo{
		category: &model.Category{
			ID:   10,
			Name: "Rent",
			Type: model.TransactionTypeExpense,
		},
	}
	service := NewRecurringService(repo, categoryRepo)

	tests := []struct {
		name    string
		input   RecurringInput
		message string
	}{
		{
			name: "invalid frequency",
			input: RecurringInput{
				CategoryID:  10,
				Type:        model.TransactionTypeExpense,
				AmountCents: 10000,
				Frequency:   model.RecurringFrequency("yearly"),
				StartDate:   time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
				Active:      true,
			},
			message: "frequency must be daily, weekly, or monthly",
		},
		{
			name: "invalid date range",
			input: RecurringInput{
				CategoryID:  10,
				Type:        model.TransactionTypeExpense,
				AmountCents: 10000,
				Frequency:   model.RecurringFrequencyMonthly,
				StartDate:   time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC),
				EndDate:     ptrTime(time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)),
				Active:      true,
			},
			message: "end_date must be on or after start_date",
		},
		{
			name: "type mismatch with category",
			input: RecurringInput{
				CategoryID:  10,
				Type:        model.TransactionTypeIncome,
				AmountCents: 10000,
				Frequency:   model.RecurringFrequencyMonthly,
				StartDate:   time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
				Active:      true,
			},
			message: "recurring transaction type must match category type",
		},
		{
			name: "note too long",
			input: RecurringInput{
				CategoryID:  10,
				Type:        model.TransactionTypeExpense,
				AmountCents: 10000,
				Note:        "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				Frequency:   model.RecurringFrequencyMonthly,
				StartDate:   time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
				Active:      true,
			},
			message: "note must be at most 500 characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

func ptrTime(v time.Time) *time.Time {
	return &v
}
