package money

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/rzfd/expand/internal/pkg/logging"
)

func ParseDecimal(value string) (int64, error) {
	logging.FromContext(nil).Info().Str("value", value).Msg("money parse decimal started")
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		logging.FromContext(nil).Warn().Msg("money parse decimal empty value")
		return 0, fmt.Errorf("amount is required")
	}

	sign := int64(1)
	if trimmed[0] == '-' {
		sign = -1
		trimmed = trimmed[1:]
	}

	parts := strings.Split(trimmed, ".")
	if len(parts) > 2 {
		logging.FromContext(nil).Warn().Str("value", value).Msg("money parse decimal invalid format")
		return 0, fmt.Errorf("invalid amount format")
	}

	wholePart := parts[0]
	if wholePart == "" {
		wholePart = "0"
	}

	whole, err := strconv.ParseInt(wholePart, 10, 64)
	if err != nil {
		logging.FromContext(nil).Warn().Err(err).Str("value", value).Msg("money parse decimal invalid whole")
		return 0, fmt.Errorf("invalid amount format")
	}

	fraction := int64(0)
	if len(parts) == 2 {
		fractionPart := parts[1]
		if len(fractionPart) > 2 {
			logging.FromContext(nil).Warn().Str("value", value).Msg("money parse decimal too many decimals")
			return 0, fmt.Errorf("amount supports max two decimal places")
		}
		if len(fractionPart) == 1 {
			fractionPart += "0"
		}
		if fractionPart != "" {
			fraction, err = strconv.ParseInt(fractionPart, 10, 64)
			if err != nil {
				logging.FromContext(nil).Warn().Err(err).Str("value", value).Msg("money parse decimal invalid fraction")
				return 0, fmt.Errorf("invalid amount format")
			}
		}
	}

	result := sign * ((whole * 100) + fraction)
	logging.FromContext(nil).Info().Int64("amount_cents", result).Msg("money parse decimal completed")
	return result, nil
}

func FormatDecimal(cents int64) string {
	logging.FromContext(nil).Info().Int64("amount_cents", cents).Msg("money format decimal started")
	sign := ""
	if cents < 0 {
		sign = "-"
		cents = -cents
	}
	result := fmt.Sprintf("%s%d.%02d", sign, cents/100, cents%100)
	logging.FromContext(nil).Info().Str("amount", result).Msg("money format decimal completed")
	return result
}
