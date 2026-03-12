package service

import (
	"context"
	"testing"
	"time"

	"github.com/rzfd/expand/internal/model"
)

type fakeTransactionRepo struct {
	item *model.Transaction
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
