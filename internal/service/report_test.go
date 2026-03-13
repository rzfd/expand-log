package service

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/rzfd/expand/internal/model"
	"github.com/rzfd/expand/internal/pkg/apperror"
)

type fakeReportRepo struct{}

func (f *fakeReportRepo) GetMonthlyTotals(_ context.Context, _ int64, _ string, _ string) (int64, int64, error) {
	return 0, 0, nil
}

func (f *fakeReportRepo) GetMonthlySpendingByCategory(_ context.Context, _ int64, _ string, _ string) ([]model.CategorySpending, error) {
	return nil, nil
}

func (f *fakeReportRepo) GetRecentTransactions(_ context.Context, _ int64, _ int) ([]model.Transaction, error) {
	return nil, nil
}

func TestReportServiceMonthlySummaryValidation(t *testing.T) {
	service := NewReportService(&fakeReportRepo{})

	tests := []struct {
		name    string
		year    int
		month   int
		message string
	}{
		{
			name:    "year too small",
			year:    1999,
			month:   1,
			message: "year must be between 2000 and 9999",
		},
		{
			name:    "year too large",
			year:    10000,
			month:   1,
			message: "year must be between 2000 and 9999",
		},
		{
			name:    "month too small",
			year:    2026,
			month:   -1,
			message: "month must be between 1 and 12",
		},
		{
			name:    "month too large",
			year:    2026,
			month:   13,
			message: "month must be between 1 and 12",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.MonthlySummary(context.Background(), 1, tt.year, tt.month)
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
