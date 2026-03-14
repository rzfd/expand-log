package service

import (
	"net/http"
	"strings"
	"time"

	"github.com/rzfd/expand/internal/pkg/apperror"
)

const (
	maxEmailLength                 = 254
	maxPasswordLength              = 72
	minCategoryNameLength          = 2
	maxCategoryNameLength          = 50
	maxTransactionNoteLength       = 500
	maxRecurringNoteLength         = 500
	maxAmountCents           int64 = 1_000_000_000_000
	maxBudgetYearOffset            = 2
	minBudgetYearOffset            = -2
	maxTransactionPastYears        = 5
	maxTransactionFutureDays       = 1
	maxRecurringPastYears          = 2
	maxRecurringFutureYears        = 10
)

var commonPasswordDenylist = map[string]struct{}{
	"password":    {},
	"password123": {},
	"12345678":    {},
	"qwerty123":   {},
	"admin123":    {},
	"letmein123":  {},
}

func newValidationError(message string) error {
	return apperror.New(http.StatusBadRequest, "validation_error", message)
}

func newRateLimitError(message string) error {
	return apperror.New(http.StatusTooManyRequests, "rate_limited", message)
}

func isCommonPassword(password string) bool {
	_, exists := commonPasswordDenylist[strings.ToLower(strings.TrimSpace(password))]
	return exists
}

func validateAmountBounds(amount int64) error {
	if amount <= 0 {
		return newValidationError("amount must be greater than zero")
	}
	if amount > maxAmountCents {
		return newValidationError("amount exceeds maximum supported value")
	}
	return nil
}

func validateNoteLength(note string, limit int, field string) error {
	if len(strings.TrimSpace(note)) > limit {
		return newValidationError(field + " must be at most " + strconvItoa(limit) + " characters")
	}
	return nil
}

func isBudgetYearAllowed(year int, now time.Time) bool {
	minYear := now.Year() + minBudgetYearOffset
	maxYear := now.Year() + maxBudgetYearOffset
	return year >= minYear && year <= maxYear
}

func strconvItoa(value int) string {
	// Tiny local helper avoids extra imports in validation-heavy files.
	const digits = "0123456789"
	if value == 0 {
		return "0"
	}
	negative := value < 0
	if negative {
		value = -value
	}
	buf := make([]byte, 0, 12)
	for value > 0 {
		buf = append(buf, digits[value%10])
		value /= 10
	}
	if negative {
		buf = append(buf, '-')
	}
	for i, j := 0, len(buf)-1; i < j; i, j = i+1, j-1 {
		buf[i], buf[j] = buf[j], buf[i]
	}
	return string(buf)
}
