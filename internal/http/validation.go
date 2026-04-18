package httpapi

import (
	"fmt"
	"time"

	"github.com/go-playground/validator/v10"
)

type DateRange struct {
	From string `validate:"required"`
	To   string `validate:"required"`
}

func ValidateDateRange(v *validator.Validate, fromRaw, toRaw string, maxRange time.Duration) (time.Time, time.Time, error) {
	payload := DateRange{From: fromRaw, To: toRaw}
	if err := v.Struct(payload); err != nil {
		return time.Time{}, time.Time{}, err
	}

	from, err := parseTime(fromRaw)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid from date: %w", err)
	}

	to, err := parseTime(toRaw)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid to date: %w", err)
	}

	if !from.Before(to) {
		return time.Time{}, time.Time{}, fmt.Errorf("from must be earlier than to")
	}

	if to.Sub(from) > maxRange {
		return time.Time{}, time.Time{}, fmt.Errorf("requested date range exceeds max allowed range")
	}

	return from.UTC(), to.UTC(), nil
}

func parseTime(s string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}

	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t, nil
	}

	return time.Time{}, fmt.Errorf("expected RFC3339 or YYYY-MM-DD")
}

