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

type fakeTransactionRepo struct {
	item      *model.Transaction
	hasRecent bool
}

func (f *fakeTransactionRepo) Create(_ context.Context, transaction *model.Transaction) error {
	transaction.ID = 1
	f.item = transaction
	return nil
}

func (f *fakeTransactionRepo) GetByIDForUser(_ context.Context, id, _ int64) (*model.Transaction, error) {
	if f.item == nil || f.item.ID != id {
		return nil, nil
	}
	return f.item, nil
}

func (f *fakeTransactionRepo) ListByUser(_ context.Context, _ int64, _ model.TransactionFilter) ([]model.Transaction, int, error) {
	if f.item == nil {
		return nil, 0, nil
	}
	return []model.Transaction{*f.item}, 1, nil
}

func (f *fakeTransactionRepo) Update(_ context.Context, transaction *model.Transaction) error {
	f.item = transaction
	return nil
}

func (f *fakeTransactionRepo) Delete(_ context.Context, _ int64, _ int64) (bool, error) {
	return true, nil
}

func (f *fakeTransactionRepo) HasRecentManualTransaction(_ context.Context, _ int64, _ time.Time) (bool, error) {
	return f.hasRecent, nil
}

type fakeCategoryLookupRepo struct {
	category *model.Category
}

func (f *fakeCategoryLookupRepo) GetByIDForUser(_ context.Context, _ int64, _ int64) (*model.Category, error) {
	return f.category, nil
}

func TestTransactionServiceCreate(t *testing.T) {
	transactionRepo := &fakeTransactionRepo{}
	categoryRepo := &fakeCategoryLookupRepo{
		category: &model.Category{
			ID:   10,
			Name: "Food",
			Type: model.TransactionTypeExpense,
		},
	}
	service := NewTransactionService(transactionRepo, categoryRepo)

	item, err := service.Create(context.Background(), 1, TransactionInput{
		CategoryID:      10,
		Type:            model.TransactionTypeExpense,
		AmountCents:     1250,
		Note:            "Lunch",
		TransactionDate: time.Date(2026, 3, 12, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	if item == nil || item.ID == 0 {
		t.Fatalf("expected created transaction")
	}
}

func TestTransactionServiceCreateValidation(t *testing.T) {
	transactionRepo := &fakeTransactionRepo{}
	categoryRepo := &fakeCategoryLookupRepo{
		category: &model.Category{
			ID:   10,
			Name: "Food",
			Type: model.TransactionTypeExpense,
		},
	}
	service := NewTransactionService(transactionRepo, categoryRepo)

	tests := []struct {
		name    string
		input   TransactionInput
		message string
	}{
		{
			name: "invalid category id",
			input: TransactionInput{
				CategoryID:      0,
				Type:            model.TransactionTypeExpense,
				AmountCents:     1000,
				TransactionDate: time.Now().UTC(),
			},
			message: "category_id must be greater than zero",
		},
		{
			name: "invalid amount",
			input: TransactionInput{
				CategoryID:      10,
				Type:            model.TransactionTypeExpense,
				AmountCents:     0,
				TransactionDate: time.Now().UTC(),
			},
			message: "amount must be greater than zero",
		},
		{
			name: "transaction date too far in future",
			input: TransactionInput{
				CategoryID:      10,
				Type:            model.TransactionTypeExpense,
				AmountCents:     1000,
				TransactionDate: time.Now().UTC().AddDate(0, 0, 3),
			},
			message: "transaction_date is too far in the future",
		},
		{
			name: "transaction date too far in past",
			input: TransactionInput{
				CategoryID:      10,
				Type:            model.TransactionTypeExpense,
				AmountCents:     1000,
				TransactionDate: time.Now().UTC().AddDate(-6, 0, 0),
			},
			message: "transaction_date is too far in the past",
		},
		{
			name: "missing transaction date",
			input: TransactionInput{
				CategoryID:  10,
				Type:        model.TransactionTypeExpense,
				AmountCents: 1000,
			},
			message: "transaction_date is required",
		},
		{
			name: "category type mismatch",
			input: TransactionInput{
				CategoryID:      10,
				Type:            model.TransactionTypeIncome,
				AmountCents:     1000,
				TransactionDate: time.Now().UTC(),
			},
			message: "transaction type must match category type",
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

func TestTransactionServiceCreateRateLimited(t *testing.T) {
	transactionRepo := &fakeTransactionRepo{hasRecent: true}
	categoryRepo := &fakeCategoryLookupRepo{
		category: &model.Category{
			ID:   10,
			Name: "Food",
			Type: model.TransactionTypeExpense,
		},
	}
	service := NewTransactionService(transactionRepo, categoryRepo)

	_, err := service.Create(context.Background(), 1, TransactionInput{
		CategoryID:      10,
		Type:            model.TransactionTypeExpense,
		AmountCents:     1250,
		Note:            "Lunch",
		TransactionDate: time.Now().UTC(),
	})
	if err == nil {
		t.Fatalf("expected rate limit error")
	}

	var appErr *apperror.Error
	if !errors.As(err, &appErr) {
		t.Fatalf("expected app error, got %T", err)
	}
	if appErr.Status != http.StatusTooManyRequests {
		t.Fatalf("expected status 429, got %d", appErr.Status)
	}
}

func TestTransactionServiceUpdateRecurringSourceBlocked(t *testing.T) {
	transactionRepo := &fakeTransactionRepo{
		item: &model.Transaction{
			ID:              1,
			UserID:          1,
			CategoryID:      10,
			Type:            model.TransactionTypeExpense,
			AmountCents:     1000,
			TransactionDate: time.Now().UTC(),
			Source:          "recurring",
		},
	}
	categoryRepo := &fakeCategoryLookupRepo{
		category: &model.Category{
			ID:   10,
			Name: "Food",
			Type: model.TransactionTypeExpense,
		},
	}
	service := NewTransactionService(transactionRepo, categoryRepo)

	_, err := service.Update(context.Background(), 1, 1, TransactionInput{
		CategoryID:      10,
		Type:            model.TransactionTypeExpense,
		AmountCents:     1200,
		TransactionDate: time.Now().UTC(),
	})
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
}
