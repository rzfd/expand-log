package schedule

import (
	"fmt"
	"time"

	"github.com/rzfd/expand/internal/model"
	"github.com/rzfd/expand/internal/pkg/logging"
)

func NormalizeDate(t time.Time) time.Time {
	logging.FromContext(nil).Info().Msg("schedule normalize date")
	return time.Date(t.UTC().Year(), t.UTC().Month(), t.UTC().Day(), 0, 0, 0, 0, time.UTC)
}

func MonthBounds(year, month int) (time.Time, time.Time, error) {
	logging.FromContext(nil).Info().Int("year", year).Int("month", month).Msg("schedule month bounds started")
	if month < 1 || month > 12 {
		logging.FromContext(nil).Warn().Int("month", month).Msg("schedule month bounds invalid month")
		return time.Time{}, time.Time{}, fmt.Errorf("month must be between 1 and 12")
	}

	start := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0)
	logging.FromContext(nil).Info().Msg("schedule month bounds completed")
	return start, end, nil
}

func NextRunDate(frequency model.RecurringFrequency, from time.Time) (time.Time, error) {
	logging.FromContext(nil).Info().Str("frequency", string(frequency)).Msg("schedule next run date started")
	from = NormalizeDate(from)

	switch frequency {
	case model.RecurringFrequencyDaily:
		result := from.AddDate(0, 0, 1)
		logging.FromContext(nil).Info().Msg("schedule next run date completed")
		return result, nil
	case model.RecurringFrequencyWeekly:
		result := from.AddDate(0, 0, 7)
		logging.FromContext(nil).Info().Msg("schedule next run date completed")
		return result, nil
	case model.RecurringFrequencyMonthly:
		result := from.AddDate(0, 1, 0)
		logging.FromContext(nil).Info().Msg("schedule next run date completed")
		return result, nil
	default:
		logging.FromContext(nil).Warn().Str("frequency", string(frequency)).Msg("schedule next run date unsupported frequency")
		return time.Time{}, fmt.Errorf("unsupported frequency: %s", frequency)
	}
}
