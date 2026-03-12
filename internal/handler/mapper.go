package handler

import (
	"time"

	"github.com/rzfd/expand/internal/model"
	"github.com/rzfd/expand/internal/pkg/logging"
	"github.com/rzfd/expand/internal/pkg/money"
)

type userResponse struct {
	ID        int64     `json:"id"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type authResponse struct {
	User        userResponse `json:"user"`
	AccessToken string       `json:"access_token"`
	ExpiresAt   time.Time    `json:"expires_at"`
}

type categoryResponse struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type transactionResponse struct {
	ID                     int64     `json:"id"`
	CategoryID             int64     `json:"category_id"`
	CategoryName           string    `json:"category_name"`
	Type                   string    `json:"type"`
	Amount                 string    `json:"amount"`
	Note                   string    `json:"note"`
	TransactionDate        string    `json:"transaction_date"`
	Source                 string    `json:"source"`
	RecurringTransactionID *int64    `json:"recurring_transaction_id,omitempty"`
	CreatedAt              time.Time `json:"created_at"`
	UpdatedAt              time.Time `json:"updated_at"`
}

type spendingByCategoryResponse struct {
	CategoryID   int64  `json:"category_id"`
	CategoryName string `json:"category_name"`
	Amount       string `json:"amount"`
}

type monthlySummaryResponse struct {
	Year               int                          `json:"year"`
	Month              int                          `json:"month"`
	IncomeTotal        string                       `json:"income_total"`
	ExpenseTotal       string                       `json:"expense_total"`
	NetBalance         string                       `json:"net_balance"`
	SpendingByCategory []spendingByCategoryResponse `json:"spending_by_category"`
}

type dashboardSummaryResponse struct {
	Year               int                          `json:"year"`
	Month              int                          `json:"month"`
	IncomeTotal        string                       `json:"income_total"`
	ExpenseTotal       string                       `json:"expense_total"`
	NetBalance         string                       `json:"net_balance"`
	SpendingByCategory []spendingByCategoryResponse `json:"spending_by_category"`
	RecentTransactions []transactionResponse        `json:"recent_transactions"`
}

type budgetResponse struct {
	ID              int64     `json:"id"`
	CategoryID      int64     `json:"category_id"`
	CategoryName    string    `json:"category_name"`
	Year            int       `json:"year"`
	Month           int       `json:"month"`
	Amount          string    `json:"amount"`
	Spent           string    `json:"spent"`
	Remaining       string    `json:"remaining"`
	UsagePercentage float64   `json:"usage_percentage"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type recurringTransactionResponse struct {
	ID           int64     `json:"id"`
	CategoryID   int64     `json:"category_id"`
	CategoryName string    `json:"category_name"`
	Type         string    `json:"type"`
	Amount       string    `json:"amount"`
	Note         string    `json:"note"`
	Frequency    string    `json:"frequency"`
	StartDate    string    `json:"start_date"`
	EndDate      *string   `json:"end_date,omitempty"`
	NextRunDate  *string   `json:"next_run_date,omitempty"`
	Active       bool      `json:"active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func newUserResponse(user model.User) userResponse {
	logging.FromContext(nil).Info().Int64("user_id", user.ID).Msg("mapper new user response")
	return userResponse{
		ID:        user.ID,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
}

func newAuthResponse(result userAuthResult) authResponse {
	logging.FromContext(nil).Info().Int64("user_id", result.User.ID).Msg("mapper new auth response")
	return authResponse{
		User:        newUserResponse(result.User),
		AccessToken: result.Token,
		ExpiresAt:   result.ExpiresAt,
	}
}

type userAuthResult struct {
	User      model.User
	Token     string
	ExpiresAt time.Time
}

func newCategoryResponse(item model.Category) categoryResponse {
	logging.FromContext(nil).Info().Int64("category_id", item.ID).Msg("mapper new category response")
	return categoryResponse{
		ID:        item.ID,
		Name:      item.Name,
		Type:      string(item.Type),
		CreatedAt: item.CreatedAt,
		UpdatedAt: item.UpdatedAt,
	}
}

func newCategoryResponses(items []model.Category) []categoryResponse {
	logging.FromContext(nil).Info().Int("count", len(items)).Msg("mapper new category responses")
	responses := make([]categoryResponse, 0, len(items))
	for _, item := range items {
		responses = append(responses, newCategoryResponse(item))
	}
	return responses
}

func newTransactionResponse(item model.Transaction) transactionResponse {
	logging.FromContext(nil).Info().Int64("transaction_id", item.ID).Msg("mapper new transaction response")
	return transactionResponse{
		ID:                     item.ID,
		CategoryID:             item.CategoryID,
		CategoryName:           item.CategoryName,
		Type:                   string(item.Type),
		Amount:                 money.FormatDecimal(item.AmountCents),
		Note:                   item.Note,
		TransactionDate:        item.TransactionDate.UTC().Format(dateLayout),
		Source:                 item.Source,
		RecurringTransactionID: item.RecurringTransactionID,
		CreatedAt:              item.CreatedAt,
		UpdatedAt:              item.UpdatedAt,
	}
}

func newTransactionResponses(items []model.Transaction) []transactionResponse {
	logging.FromContext(nil).Info().Int("count", len(items)).Msg("mapper new transaction responses")
	responses := make([]transactionResponse, 0, len(items))
	for _, item := range items {
		responses = append(responses, newTransactionResponse(item))
	}
	return responses
}

func newMonthlySummaryResponse(item model.MonthlySummary) monthlySummaryResponse {
	logging.FromContext(nil).Info().Int("year", item.Year).Int("month", item.Month).Msg("mapper new monthly summary response")
	return monthlySummaryResponse{
		Year:               item.Year,
		Month:              item.Month,
		IncomeTotal:        money.FormatDecimal(item.IncomeCents),
		ExpenseTotal:       money.FormatDecimal(item.ExpenseCents),
		NetBalance:         money.FormatDecimal(item.NetBalanceCents),
		SpendingByCategory: newSpendingResponses(item.SpendingByCategory),
	}
}

func newDashboardSummaryResponse(item model.DashboardSummary) dashboardSummaryResponse {
	logging.FromContext(nil).Info().Int("year", item.Year).Int("month", item.Month).Msg("mapper new dashboard summary response")
	return dashboardSummaryResponse{
		Year:               item.Year,
		Month:              item.Month,
		IncomeTotal:        money.FormatDecimal(item.IncomeCents),
		ExpenseTotal:       money.FormatDecimal(item.ExpenseCents),
		NetBalance:         money.FormatDecimal(item.NetBalanceCents),
		SpendingByCategory: newSpendingResponses(item.SpendingByCategory),
		RecentTransactions: newTransactionResponses(item.RecentTransactions),
	}
}

func newSpendingResponses(items []model.CategorySpending) []spendingByCategoryResponse {
	logging.FromContext(nil).Info().Int("count", len(items)).Msg("mapper new spending responses")
	responses := make([]spendingByCategoryResponse, 0, len(items))
	for _, item := range items {
		responses = append(responses, spendingByCategoryResponse{
			CategoryID:   item.CategoryID,
			CategoryName: item.CategoryName,
			Amount:       money.FormatDecimal(item.AmountCents),
		})
	}
	return responses
}

func newBudgetResponse(item model.Budget) budgetResponse {
	logging.FromContext(nil).Info().Int64("budget_id", item.ID).Msg("mapper new budget response")
	remaining := item.AmountCents - item.SpentCents
	usage := 0.0
	if item.AmountCents > 0 {
		usage = (float64(item.SpentCents) / float64(item.AmountCents)) * 100
	}

	return budgetResponse{
		ID:              item.ID,
		CategoryID:      item.CategoryID,
		CategoryName:    item.CategoryName,
		Year:            item.Year,
		Month:           item.Month,
		Amount:          money.FormatDecimal(item.AmountCents),
		Spent:           money.FormatDecimal(item.SpentCents),
		Remaining:       money.FormatDecimal(remaining),
		UsagePercentage: usage,
		CreatedAt:       item.CreatedAt,
		UpdatedAt:       item.UpdatedAt,
	}
}

func newBudgetResponses(items []model.Budget) []budgetResponse {
	logging.FromContext(nil).Info().Int("count", len(items)).Msg("mapper new budget responses")
	responses := make([]budgetResponse, 0, len(items))
	for _, item := range items {
		responses = append(responses, newBudgetResponse(item))
	}
	return responses
}

func newRecurringResponse(item model.RecurringTransaction) recurringTransactionResponse {
	logging.FromContext(nil).Info().Int64("recurring_id", item.ID).Msg("mapper new recurring response")
	return recurringTransactionResponse{
		ID:           item.ID,
		CategoryID:   item.CategoryID,
		CategoryName: item.CategoryName,
		Type:         string(item.Type),
		Amount:       money.FormatDecimal(item.AmountCents),
		Note:         item.Note,
		Frequency:    string(item.Frequency),
		StartDate:    item.StartDate.UTC().Format(dateLayout),
		EndDate:      formatDatePointer(item.EndDate),
		NextRunDate:  formatDatePointer(item.NextRunDate),
		Active:       item.Active,
		CreatedAt:    item.CreatedAt,
		UpdatedAt:    item.UpdatedAt,
	}
}

func newRecurringResponses(items []model.RecurringTransaction) []recurringTransactionResponse {
	logging.FromContext(nil).Info().Int("count", len(items)).Msg("mapper new recurring responses")
	responses := make([]recurringTransactionResponse, 0, len(items))
	for _, item := range items {
		responses = append(responses, newRecurringResponse(item))
	}
	return responses
}

func formatDatePointer(value *time.Time) *string {
	logging.FromContext(nil).Info().Bool("is_nil", value == nil).Msg("mapper format date pointer")
	if value == nil {
		return nil
	}
	formatted := value.UTC().Format(dateLayout)
	return &formatted
}
