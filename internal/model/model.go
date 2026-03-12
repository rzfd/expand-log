package model

import (
	"time"

	"github.com/rzfd/expand/internal/pkg/logging"
)

type TransactionType string

const (
	TransactionTypeIncome  TransactionType = "income"
	TransactionTypeExpense TransactionType = "expense"
)

func (t TransactionType) IsValid() bool {
	result := t == TransactionTypeIncome || t == TransactionTypeExpense
	logging.FromContext(nil).Info().Str("transaction_type", string(t)).Bool("valid", result).Msg("model transaction type validate")
	return result
}

type RecurringFrequency string

const (
	RecurringFrequencyDaily   RecurringFrequency = "daily"
	RecurringFrequencyWeekly  RecurringFrequency = "weekly"
	RecurringFrequencyMonthly RecurringFrequency = "monthly"
)

func (f RecurringFrequency) IsValid() bool {
	result := f == RecurringFrequencyDaily || f == RecurringFrequencyWeekly || f == RecurringFrequencyMonthly
	logging.FromContext(nil).Info().Str("frequency", string(f)).Bool("valid", result).Msg("model recurring frequency validate")
	return result
}

type User struct {
	ID           int64
	Email        string
	PasswordHash string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Category struct {
	ID        int64
	UserID    int64
	Name      string
	Type      TransactionType
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Transaction struct {
	ID                     int64
	UserID                 int64
	CategoryID             int64
	CategoryName           string
	Type                   TransactionType
	AmountCents            int64
	Note                   string
	TransactionDate        time.Time
	Source                 string
	RecurringTransactionID *int64
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

type TransactionFilter struct {
	StartDate      *time.Time
	EndDate        *time.Time
	Type           *TransactionType
	CategoryID     *int64
	MinAmountCents *int64
	MaxAmountCents *int64
	Page           int
	PageSize       int
	Offset         int
}

type Budget struct {
	ID           int64
	UserID       int64
	CategoryID   int64
	CategoryName string
	Year         int
	Month        int
	AmountCents  int64
	SpentCents   int64
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type RecurringTransaction struct {
	ID           int64
	UserID       int64
	CategoryID   int64
	CategoryName string
	Type         TransactionType
	AmountCents  int64
	Note         string
	Frequency    RecurringFrequency
	StartDate    time.Time
	EndDate      *time.Time
	NextRunDate  *time.Time
	Active       bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type CategorySpending struct {
	CategoryID   int64
	CategoryName string
	AmountCents  int64
}

type MonthlySummary struct {
	Year               int
	Month              int
	IncomeCents        int64
	ExpenseCents       int64
	NetBalanceCents    int64
	SpendingByCategory []CategorySpending
}

type DashboardSummary struct {
	Year               int
	Month              int
	IncomeCents        int64
	ExpenseCents       int64
	NetBalanceCents    int64
	SpendingByCategory []CategorySpending
	RecentTransactions []Transaction
}
