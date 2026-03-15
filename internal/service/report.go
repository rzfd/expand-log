package service

import (
	"context"
	"net/http"
	"time"

	"github.com/rzfd/expand/internal/model"
	"github.com/rzfd/expand/internal/pkg/apperror"
	"github.com/rzfd/expand/internal/pkg/logging"
	"github.com/rzfd/expand/internal/pkg/schedule"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

var reportTracer = otel.Tracer("service.report")

type reportRepository interface {
	GetMonthlyTotals(ctx context.Context, userID int64, start, end string) (int64, int64, error)
	GetMonthlySpendingByCategory(ctx context.Context, userID int64, start, end string) ([]model.CategorySpending, error)
	GetRecentTransactions(ctx context.Context, userID int64, limit int) ([]model.Transaction, error)
}

type ReportService struct {
	reports reportRepository
}

func NewReportService(reports reportRepository) *ReportService {
	return &ReportService{reports: reports}
}

func (s *ReportService) MonthlySummary(ctx context.Context, userID int64, year, month int) (*model.MonthlySummary, error) {
	ctx, span := reportTracer.Start(ctx, "service.report.monthly_summary")
	defer span.End()
	span.SetAttributes(
		attribute.Int64("app.user.id", userID),
		attribute.Int("app.report.year", year),
		attribute.Int("app.report.month", month),
	)

	logger := logging.FromContext(ctx)
	logger.Info().Int64("user_id", userID).Int("year", year).Int("month", month).Msg("service report monthly summary started")
	start, end, err := resolveMonth(ctx, year, month)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "resolve month failed")
		logger.Warn().Err(err).Msg("service report monthly summary invalid month")
		return nil, err
	}

	income, expense, err := s.reports.GetMonthlyTotals(ctx, userID, start.Format("2006-01-02"), end.Format("2006-01-02"))
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "get monthly totals failed")
		logger.Error().Err(err).Msg("service report monthly summary totals failed")
		return nil, apperror.Wrap(http.StatusInternalServerError, "internal_error", "failed to build monthly summary", err)
	}

	spending, err := s.reports.GetMonthlySpendingByCategory(ctx, userID, start.Format("2006-01-02"), end.Format("2006-01-02"))
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "get monthly spending failed")
		logger.Error().Err(err).Msg("service report monthly summary category spending failed")
		return nil, apperror.Wrap(http.StatusInternalServerError, "internal_error", "failed to build monthly summary", err)
	}
	span.SetAttributes(
		attribute.Int64("app.report.income_cents", income),
		attribute.Int64("app.report.expense_cents", expense),
		attribute.Int("app.report.category_count", len(spending)),
	)

	logger.Info().Int64("user_id", userID).Int64("income_cents", income).Int64("expense_cents", expense).Msg("service report monthly summary completed")
	return &model.MonthlySummary{
		Year:               start.Year(),
		Month:              int(start.Month()),
		IncomeCents:        income,
		ExpenseCents:       expense,
		NetBalanceCents:    income - expense,
		SpendingByCategory: spending,
	}, nil
}

func (s *ReportService) DashboardSummary(ctx context.Context, userID int64, year, month int) (*model.DashboardSummary, error) {
	ctx, span := reportTracer.Start(ctx, "service.report.dashboard_summary")
	defer span.End()
	span.SetAttributes(
		attribute.Int64("app.user.id", userID),
		attribute.Int("app.report.year", year),
		attribute.Int("app.report.month", month),
	)

	logger := logging.FromContext(ctx)
	logger.Info().Int64("user_id", userID).Int("year", year).Int("month", month).Msg("service report dashboard summary started")
	monthly, err := s.MonthlySummary(ctx, userID, year, month)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "monthly summary failed")
		logger.Warn().Err(err).Msg("service report dashboard summary monthly failed")
		return nil, err
	}

	recent, err := s.reports.GetRecentTransactions(ctx, userID, 5)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "load recent transactions failed")
		logger.Error().Err(err).Msg("service report dashboard summary recent transactions failed")
		return nil, apperror.Wrap(http.StatusInternalServerError, "internal_error", "failed to load recent transactions", err)
	}
	span.SetAttributes(attribute.Int("app.report.recent_count", len(recent)))

	logger.Info().Int64("user_id", userID).Int("recent_count", len(recent)).Msg("service report dashboard summary completed")
	return &model.DashboardSummary{
		Year:               monthly.Year,
		Month:              monthly.Month,
		IncomeCents:        monthly.IncomeCents,
		ExpenseCents:       monthly.ExpenseCents,
		NetBalanceCents:    monthly.NetBalanceCents,
		SpendingByCategory: monthly.SpendingByCategory,
		RecentTransactions: recent,
	}, nil
}

func resolveMonth(ctx context.Context, year, month int) (time.Time, time.Time, error) {
	logging.FromContext(ctx).Info().Int("year", year).Int("month", month).Msg("service report resolve month started")
	if year != 0 && (year < 2000 || year > 9999) {
		logging.FromContext(ctx).Warn().Int("year", year).Msg("service report resolve month invalid year")
		return time.Time{}, time.Time{}, apperror.New(http.StatusBadRequest, "validation_error", "year must be between 2000 and 9999")
	}
	if month != 0 && (month < 1 || month > 12) {
		logging.FromContext(ctx).Warn().Int("month", month).Msg("service report resolve month invalid month")
		return time.Time{}, time.Time{}, apperror.New(http.StatusBadRequest, "validation_error", "month must be between 1 and 12")
	}

	now := time.Now().UTC()
	if year == 0 {
		year = now.Year()
	}
	if month == 0 {
		month = int(now.Month())
	}

	start, end, err := schedule.MonthBounds(year, month)
	if err != nil {
		logging.FromContext(ctx).Warn().Err(err).Int("year", year).Int("month", month).Msg("service report resolve month failed")
		return time.Time{}, time.Time{}, apperror.New(http.StatusBadRequest, "validation_error", err.Error())
	}

	logging.FromContext(ctx).Info().Int("year", year).Int("month", month).Msg("service report resolve month completed")
	return start, end, nil
}
