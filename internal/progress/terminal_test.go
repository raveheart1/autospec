package progress_test

import (
	"os"
	"testing"

	"github.com/ariel-frischer/autospec/internal/progress"
)

// TestDetectTerminalCapabilities tests terminal capability detection
func TestDetectTerminalCapabilities(t *testing.T) {
	tests := []struct {
		name                string
		setupEnv            func()
		cleanupEnv          func()
		wantSupportsColor   bool
		wantSupportsUnicode bool
	}{
		{
			name: "NO_COLOR disables color",
			setupEnv: func() {
				os.Setenv("NO_COLOR", "1")
			},
			cleanupEnv: func() {
				os.Unsetenv("NO_COLOR")
			},
			wantSupportsColor: false,
			// Unicode support depends on TTY, we'll just verify color is disabled
		},
		{
			name: "AUTOSPEC_ASCII forces ASCII",
			setupEnv: func() {
				os.Setenv("AUTOSPEC_ASCII", "1")
			},
			cleanupEnv: func() {
				os.Unsetenv("AUTOSPEC_ASCII")
			},
			wantSupportsUnicode: false,
		},
		{
			name: "both NO_COLOR and AUTOSPEC_ASCII",
			setupEnv: func() {
				os.Setenv("NO_COLOR", "1")
				os.Setenv("AUTOSPEC_ASCII", "1")
			},
			cleanupEnv: func() {
				os.Unsetenv("NO_COLOR")
				os.Unsetenv("AUTOSPEC_ASCII")
			},
			wantSupportsColor:   false,
			wantSupportsUnicode: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupEnv != nil {
				tt.setupEnv()
				defer tt.cleanupEnv()
			}

			caps := progress.DetectTerminalCapabilities()

			// Verify width is non-negative
			if caps.Width < 0 {
				t.Errorf("DetectTerminalCapabilities() Width = %d, want >= 0", caps.Width)
			}

			// If NO_COLOR is set, color should be disabled
			if os.Getenv("NO_COLOR") != "" && caps.SupportsColor {
				t.Errorf("DetectTerminalCapabilities() SupportsColor = true with NO_COLOR set, want false")
			}

			// If AUTOSPEC_ASCII is set, Unicode should be disabled
			if os.Getenv("AUTOSPEC_ASCII") == "1" && caps.SupportsUnicode {
				t.Errorf("DetectTerminalCapabilities() SupportsUnicode = true with AUTOSPEC_ASCII=1, want false")
			}

			// If not TTY, color and Unicode should be disabled
			if !caps.IsTTY {
				if caps.SupportsColor {
					t.Errorf("DetectTerminalCapabilities() SupportsColor = true when !IsTTY, want false")
				}
				if caps.SupportsUnicode {
					t.Errorf("DetectTerminalCapabilities() SupportsUnicode = true when !IsTTY, want false")
				}
			}
		})
	}
}

// TestSelectSymbols tests symbol selection based on capabilities
func TestSelectSymbols(t *testing.T) {
	tests := []struct {
		name                string
		capabilities        progress.TerminalCapabilities
		wantCheckmark       string
		wantFailure         string
		wantSpinnerNonEmpty bool
	}{
		{
			name: "Unicode support enabled",
			capabilities: progress.TerminalCapabilities{
				IsTTY:           true,
				SupportsUnicode: true,
				SupportsColor:   true,
			},
			wantCheckmark:       "✓",
			wantFailure:         "✗",
			wantSpinnerNonEmpty: true,
		},
		{
			name: "ASCII fallback mode",
			capabilities: progress.TerminalCapabilities{
				IsTTY:           true,
				SupportsUnicode: false,
				SupportsColor:   false,
			},
			wantCheckmark:       "[OK]",
			wantFailure:         "[FAIL]",
			wantSpinnerNonEmpty: true,
		},
		{
			name: "non-TTY mode",
			capabilities: progress.TerminalCapabilities{
				IsTTY:           false,
				SupportsUnicode: false,
				SupportsColor:   false,
			},
			wantCheckmark:       "[OK]",
			wantFailure:         "[FAIL]",
			wantSpinnerNonEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			symbols := progress.SelectSymbols(tt.capabilities)

			if symbols.Checkmark != tt.wantCheckmark {
				t.Errorf("SelectSymbols() Checkmark = %q, want %q", symbols.Checkmark, tt.wantCheckmark)
			}

			if symbols.Failure != tt.wantFailure {
				t.Errorf("SelectSymbols() Failure = %q, want %q", symbols.Failure, tt.wantFailure)
			}

			if tt.wantSpinnerNonEmpty && symbols.SpinnerSet < 0 {
				t.Errorf("SelectSymbols() SpinnerSet = %d, want >= 0", symbols.SpinnerSet)
			}
		})
	}
}
