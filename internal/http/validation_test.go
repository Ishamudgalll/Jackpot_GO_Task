package httpapi

import (
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
)

func TestValidateDateRange(t *testing.T) {
	v := validator.New()
	from, to, err := ValidateDateRange(v, "2026-01-01", "2026-01-31", 40*24*time.Hour)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !from.Before(to) {
		t.Fatalf("expected from before to")
	}
}

func TestValidateDateRangeErrors(t *testing.T) {
	v := validator.New()
	_, _, err := ValidateDateRange(v, "", "2026-01-31", 40*24*time.Hour)
	if err == nil {
		t.Fatalf("expected validation error")
	}

	_, _, err = ValidateDateRange(v, "2026-03-01", "2026-01-31", 40*24*time.Hour)
	if err == nil {
		t.Fatalf("expected date ordering error")
	}
}

