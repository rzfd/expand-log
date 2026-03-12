package handler

import (
	"github.com/labstack/echo/v4"

	authmiddleware "github.com/rzfd/expand/internal/middleware"
	"github.com/rzfd/expand/internal/pkg/logging"
	"github.com/rzfd/expand/internal/pkg/response"
	"github.com/rzfd/expand/internal/service"
)

type ReportHandler struct {
	reports *service.ReportService
}

func NewReportHandler(reports *service.ReportService) *ReportHandler {
	return &ReportHandler{reports: reports}
}

func (h *ReportHandler) Monthly(c echo.Context) error {
	logger := logging.FromContext(c.Request().Context())
	logger.Info().Msg("report monthly started")
	userID, ok := authmiddleware.UserIDFromContext(c)
	if !ok {
		logger.Warn().Msg("report monthly unauthorized")
		return unauthorized(c)
	}

	year, month, err := parseYearMonth(c)
	if err != nil {
		logger.Warn().Err(err).Int64("user_id", userID).Msg("report monthly invalid year month")
		return response.Error(c, err)
	}

	item, err := h.reports.MonthlySummary(c.Request().Context(), userID, year, month)
	if err != nil {
		logger.Warn().Err(err).Int64("user_id", userID).Int("year", year).Int("month", month).Msg("report monthly failed")
		return response.Error(c, err)
	}

	logger.Info().Int64("user_id", userID).Int("year", item.Year).Int("month", item.Month).Msg("report monthly completed")
	return response.OK(c, newMonthlySummaryResponse(*item))
}

func (h *ReportHandler) Dashboard(c echo.Context) error {
	logger := logging.FromContext(c.Request().Context())
	logger.Info().Msg("report dashboard started")
	userID, ok := authmiddleware.UserIDFromContext(c)
	if !ok {
		logger.Warn().Msg("report dashboard unauthorized")
		return unauthorized(c)
	}

	year, month, err := parseYearMonth(c)
	if err != nil {
		logger.Warn().Err(err).Int64("user_id", userID).Msg("report dashboard invalid year month")
		return response.Error(c, err)
	}

	item, err := h.reports.DashboardSummary(c.Request().Context(), userID, year, month)
	if err != nil {
		logger.Warn().Err(err).Int64("user_id", userID).Int("year", year).Int("month", month).Msg("report dashboard failed")
		return response.Error(c, err)
	}

	logger.Info().Int64("user_id", userID).Int("year", item.Year).Int("month", item.Month).Msg("report dashboard completed")
	return response.OK(c, newDashboardSummaryResponse(*item))
}
