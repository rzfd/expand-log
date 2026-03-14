package money

import (
	"fmt"
	"strconv"
	"strings"
)

func ParseDecimal(value string) (int64, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0, fmt.Errorf("amount is required")
	}

	sign := int64(1)
	if trimmed[0] == '-' {
		sign = -1
		trimmed = trimmed[1:]
	}

	parts := strings.Split(trimmed, ".")
	if len(parts) > 2 {
		return 0, fmt.Errorf("invalid amount format")
	}

	wholePart := parts[0]
	if wholePart == "" {
		wholePart = "0"
	}

	whole, err := strconv.ParseInt(wholePart, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid amount format")
	}

	fraction := int64(0)
	if len(parts) == 2 {
		fractionPart := parts[1]
		if len(fractionPart) > 2 {
			return 0, fmt.Errorf("amount supports max two decimal places")
		}
		if len(fractionPart) == 1 {
			fractionPart += "0"
		}
		if fractionPart != "" {
			fraction, err = strconv.ParseInt(fractionPart, 10, 64)
			if err != nil {
				return 0, fmt.Errorf("invalid amount format")
			}
		}
	}

	result := sign * ((whole * 100) + fraction)
	return result, nil
}

func FormatDecimal(cents int64) string {
	sign := ""
	if cents < 0 {
		sign = "-"
		cents = -cents
	}
	result := fmt.Sprintf("%s%d.%02d", sign, cents/100, cents%100)
	return result
}
