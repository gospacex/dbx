package dbx_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/gospacex/dbx"
)

func TestSentinelErrors(t *testing.T) {
	if dbx.ErrInvalidConfig == nil {
		t.Error("ErrInvalidConfig should not be nil")
	}
	if dbx.ErrDriverUnsupported == nil {
		t.Error("ErrDriverUnsupported should not be nil")
	}
	if !errors.Is(dbx.ErrInvalidConfig, dbx.ErrInvalidConfig) {
		t.Error("ErrInvalidConfig should match itself")
	}
}

func TestSentinelErrors_ErrorMessages(t *testing.T) {
	if msg := dbx.ErrInvalidConfig.Error(); msg == "" {
		t.Error("ErrInvalidConfig.Error() should return non-empty message")
	}
	if msg := dbx.ErrDriverUnsupported.Error(); msg == "" {
		t.Error("ErrDriverUnsupported.Error() should return non-empty message")
	}
}

func TestSentinelErrors_Distinct(t *testing.T) {
	if errors.Is(dbx.ErrInvalidConfig, dbx.ErrDriverUnsupported) {
		t.Error("ErrInvalidConfig and ErrDriverUnsupported should not match each other")
	}
}

func TestSentinelErrors_IsAgainstOther(t *testing.T) {
	if errors.Is(dbx.ErrInvalidConfig, fmt.Errorf("unrelated")) {
		t.Error("ErrInvalidConfig should not match an unrelated error")
	}
}

func TestSentinelErrors_WrappedInFmt(t *testing.T) {
	wrapped := fmt.Errorf("loading driver: %w", dbx.ErrDriverUnsupported)
	if !errors.Is(wrapped, dbx.ErrDriverUnsupported) {
		t.Error("errors.Is should unwrap fmt.Errorf %w chain to find sentinel")
	}
}

func TestRedactDSN(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"with password", "user:pass@tcp(localhost:3306)/db", "user:xxxxx@tcp(localhost:3306)/db"},
		{"without password", "noauth@tcp(localhost:3306)/db", "noauth@tcp(localhost:3306)/db"},
		{"empty", "", ""},
		{"no colon before at", "tcp(localhost:3306)/db", "tcp(localhost:3306)/db"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := dbx.RedactDSN(tt.input)
			if got != tt.want {
				t.Errorf("RedactDSN(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
