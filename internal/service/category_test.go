package service

import (
	"context"
	"testing"

	"github.com/rzfd/expand/internal/model"
)

type fakeCategoryRepo struct {
	items map[int64]model.Category
}

func (f *fakeCategoryRepo) Create(_ context.Context, category *model.Category) error {
	if f.items == nil {
		f.items = make(map[int64]model.Category)
	}
	category.ID = int64(len(f.items) + 1)
	f.items[category.ID] = *category
	return nil
}

func (f *fakeCategoryRepo) ListByUser(_ context.Context, userID int64) ([]model.Category, error) {
	items := make([]model.Category, 0)
	for _, item := range f.items {
		if item.UserID == userID {
			items = append(items, item)
		}
	}
	return items, nil
}

func (f *fakeCategoryRepo) GetByIDForUser(_ context.Context, id, userID int64) (*model.Category, error) {
	item, ok := f.items[id]
	if !ok || item.UserID != userID {
		return nil, nil
	}
	copy := item
	return &copy, nil
}

func (f *fakeCategoryRepo) Update(_ context.Context, category *model.Category) error {
	f.items[category.ID] = *category
	return nil
}

func (f *fakeCategoryRepo) Delete(_ context.Context, id, _ int64) (bool, error) {
	if _, ok := f.items[id]; !ok {
		return false, nil
	}
	delete(f.items, id)
	return true, nil
}

func TestCategoryServiceCreate(t *testing.T) {
	repo := &fakeCategoryRepo{}
	service := NewCategoryService(repo)

	item, err := service.Create(context.Background(), 1, "Salary", model.TransactionTypeIncome)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	if item.ID == 0 {
		t.Fatalf("expected created category to have ID")
	}
}
