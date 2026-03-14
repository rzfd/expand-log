package schedule

import (
	"fmt"
	"time"

	"github.com/rzfd/expand/internal/model"
)

func NormalizeDate(t time.Time) time.Time {
	return time.Date(t.UTC().Year(), t.UTC().Month(), t.UTC().Day(), 0, 0, 0, 0, time.UTC)
}

func MonthBounds(year, month int) (time.Time, time.Time, error) {
	if month < 1 || month > 12 {
		return time.Time{}, time.Time{}, fmt.Errorf("month must be between 1 and 12")
	}

	start := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0)
	return start, end, nil
}

func NextRunDate(frequency model.RecurringFrequency, from time.Time) (time.Time, error) {
	from = NormalizeDate(from)

	switch frequency {
	case model.RecurringFrequencyDaily:
		return from.AddDate(0, 0, 1), nil
	case model.RecurringFrequencyWeekly:
		return from.AddDate(0, 0, 7), nil
	case model.RecurringFrequencyMonthly:
		return from.AddDate(0, 1, 0), nil
	default:
		return time.Time{}, fmt.Errorf("unsupported frequency: %s", frequency)
	}
}
