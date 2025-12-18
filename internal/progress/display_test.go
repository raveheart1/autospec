// Package progress_test tests progress display rendering, stage counters, checkmarks, and spinner lifecycle.
// Related: internal/progress/display.go
// Tags: progress, display, rendering, stages, spinner, tty
package progress_test

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/ariel-frischer/autospec/internal/progress"
)

// captureOutput captures stdout during function execution
func captureOutput(fn func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	fn()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

// TestProgressDisplay_StartStage tests stage counter rendering
func TestProgressDisplay_StartStage(t *testing.T) {
	tests := map[string]struct {
		capabilities progress.TerminalCapabilities
		stage        progress.StageInfo
		wantContains []string
		wantErr      bool
	}{
		"TTY mode with Unicode - first stage": {
			capabilities: progress.TerminalCapabilities{
				IsTTY:           true,
				SupportsUnicode: true,
				SupportsColor:   true,
				Width:           80,
			},
			stage: progress.StageInfo{
				Name:        "specify",
				Number:      1,
				TotalStages: 3,
				Status:      progress.StageInProgress,
				RetryCount:  0,
				MaxRetries:  3,
			},
			wantContains: []string{"[1/3]", "specify"},
			wantErr:      false,
		},
		"non-TTY mode - second stage": {
			capabilities: progress.TerminalCapabilities{
				IsTTY:           false,
				SupportsUnicode: false,
				SupportsColor:   false,
				Width:           0,
			},
			stage: progress.StageInfo{
				Name:        "plan",
				Number:      2,
				TotalStages: 3,
				Status:      progress.StageInProgress,
				RetryCount:  0,
				MaxRetries:  3,
			},
			wantContains: []string{"[2/3]", "Plan"},
			wantErr:      false,
		},
		"retry attempt - show retry count": {
			capabilities: progress.TerminalCapabilities{
				IsTTY:           true,
				SupportsUnicode: true,
				SupportsColor:   true,
				Width:           80,
			},
			stage: progress.StageInfo{
				Name:        "tasks",
				Number:      3,
				TotalStages: 3,
				Status:      progress.StageInProgress,
				RetryCount:  1,
				MaxRetries:  3,
			},
			wantContains: []string{"[3/3]", "tasks", "(retry 2/3)"},
			wantErr:      false,
		},
		"invalid stage - empty name": {
			capabilities: progress.TerminalCapabilities{
				IsTTY:           true,
				SupportsUnicode: true,
				SupportsColor:   true,
				Width:           80,
			},
			stage: progress.StageInfo{
				Name:        "",
				Number:      1,
				TotalStages: 3,
			},
			wantErr: true,
		},
		"four-stage workflow - implement stage": {
			capabilities: progress.TerminalCapabilities{
				IsTTY:           true,
				SupportsUnicode: true,
				SupportsColor:   true,
				Width:           80,
			},
			stage: progress.StageInfo{
				Name:        "implement",
				Number:      4,
				TotalStages: 4,
				Status:      progress.StageInProgress,
				RetryCount:  0,
				MaxRetries:  3,
			},
			wantContains: []string{"[4/4]", "implement"},
			wantErr:      false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			display := progress.NewProgressDisplay(tt.capabilities)

			var output string
			var err error

			if tt.capabilities.IsTTY {
				// For TTY mode, spinner starts, so we just check error
				err = display.StartStage(tt.stage)
			} else {
				// For non-TTY mode, capture stdout
				output = captureOutput(func() {
					err = display.StartStage(tt.stage)
				})
			}

			if tt.wantErr {
				if err == nil {
					t.Errorf("StartStage() error = nil, want error")
				}
				return
			}

			if err != nil {
				t.Errorf("StartStage() unexpected error = %v", err)
				return
			}

			// For non-TTY mode, verify output contains expected strings
			if !tt.capabilities.IsTTY {
				for _, want := range tt.wantContains {
					if !strings.Contains(output, want) {
						t.Errorf("StartStage() output = %q, want to contain %q", output, want)
					}
				}
			}
		})
	}
}

// TestProgressDisplay_UpdateRetry tests retry count display
func TestProgressDisplay_UpdateRetry(t *testing.T) {
	caps := progress.TerminalCapabilities{
		IsTTY:           false,
		SupportsUnicode: false,
		SupportsColor:   false,
	}

	stage := progress.StageInfo{
		Name:        "specify",
		Number:      1,
		TotalStages: 3,
		Status:      progress.StageInProgress,
		RetryCount:  2,
		MaxRetries:  3,
	}

	display := progress.NewProgressDisplay(caps)

	output := captureOutput(func() {
		_ = display.UpdateRetry(stage)
	})

	// Should show retry 3/3 (RetryCount is 0-indexed, display is 1-indexed)
	if !strings.Contains(output, "(retry 3/3)") {
		t.Errorf("UpdateRetry() output = %q, want to contain '(retry 3/3)'", output)
	}
}

// TestProgressDisplay_CompleteStage tests completion checkmarks (User Story 3)
func TestProgressDisplay_CompleteStage(t *testing.T) {
	tests := map[string]struct {
		capabilities progress.TerminalCapabilities
		stage        progress.StageInfo
		wantContains []string
	}{
		"Unicode checkmark with color": {
			capabilities: progress.TerminalCapabilities{
				IsTTY:           true,
				SupportsUnicode: true,
				SupportsColor:   true,
				Width:           80,
			},
			stage: progress.StageInfo{
				Name:        "specify",
				Number:      1,
				TotalStages: 3,
				Status:      progress.StageCompleted,
			},
			wantContains: []string{"✓", "[1/3]", "Specify", "complete"},
		},
		"ASCII checkmark without color": {
			capabilities: progress.TerminalCapabilities{
				IsTTY:           true,
				SupportsUnicode: false,
				SupportsColor:   false,
				Width:           80,
			},
			stage: progress.StageInfo{
				Name:        "plan",
				Number:      2,
				TotalStages: 3,
				Status:      progress.StageCompleted,
			},
			wantContains: []string{"[OK]", "[2/3]", "Plan", "complete"},
		},
		"non-TTY mode completion": {
			capabilities: progress.TerminalCapabilities{
				IsTTY:           false,
				SupportsUnicode: false,
				SupportsColor:   false,
			},
			stage: progress.StageInfo{
				Name:        "tasks",
				Number:      3,
				TotalStages: 3,
				Status:      progress.StageCompleted,
			},
			wantContains: []string{"[OK]", "[3/3]", "Tasks", "complete"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			display := progress.NewProgressDisplay(tt.capabilities)

			output := captureOutput(func() {
				_ = display.CompleteStage(tt.stage)
			})

			for _, want := range tt.wantContains {
				if !strings.Contains(output, want) {
					t.Errorf("CompleteStage() output = %q, want to contain %q", output, want)
				}
			}
		})
	}
}

// TestProgressDisplay_FailStage tests failure indicators (User Story 3)
func TestProgressDisplay_FailStage(t *testing.T) {
	tests := map[string]struct {
		capabilities progress.TerminalCapabilities
		stage        progress.StageInfo
		err          error
		wantContains []string
	}{
		"Unicode failure mark with color": {
			capabilities: progress.TerminalCapabilities{
				IsTTY:           true,
				SupportsUnicode: true,
				SupportsColor:   true,
				Width:           80,
			},
			stage: progress.StageInfo{
				Name:        "specify",
				Number:      1,
				TotalStages: 3,
				Status:      progress.StageFailed,
			},
			err:          fmt.Errorf("validation failed"),
			wantContains: []string{"✗", "[1/3]", "Specify", "failed", "validation failed"},
		},
		"ASCII failure mark without color": {
			capabilities: progress.TerminalCapabilities{
				IsTTY:           true,
				SupportsUnicode: false,
				SupportsColor:   false,
				Width:           80,
			},
			stage: progress.StageInfo{
				Name:        "plan",
				Number:      2,
				TotalStages: 3,
				Status:      progress.StageFailed,
			},
			err:          fmt.Errorf("file not found"),
			wantContains: []string{"[FAIL]", "[2/3]", "Plan", "failed", "file not found"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			display := progress.NewProgressDisplay(tt.capabilities)

			output := captureOutput(func() {
				_ = display.FailStage(tt.stage, tt.err)
			})

			for _, want := range tt.wantContains {
				if !strings.Contains(output, want) {
					t.Errorf("FailStage() output = %q, want to contain %q", output, want)
				}
			}
		})
	}
}

// TestSpinnerLifecycle tests spinner start/stop behavior (User Story 2)
func TestSpinnerLifecycle(t *testing.T) {
	// TTY mode - spinner should start
	capsTTY := progress.TerminalCapabilities{
		IsTTY:           true,
		SupportsUnicode: true,
		SupportsColor:   true,
		Width:           80,
	}

	stage := progress.StageInfo{
		Name:        "specify",
		Number:      1,
		TotalStages: 3,
		Status:      progress.StageInProgress,
	}

	display := progress.NewProgressDisplay(capsTTY)

	// Start stage - spinner starts
	err := display.StartStage(stage)
	if err != nil {
		t.Fatalf("StartStage() unexpected error = %v", err)
	}

	// Complete stage - spinner should stop
	output := captureOutput(func() {
		_ = display.CompleteStage(stage)
	})

	if !strings.Contains(output, "✓") {
		t.Errorf("CompleteStage() output = %q, want to contain checkmark", output)
	}
}

// TestSpinnerDisabledNonTTY tests spinner is disabled in non-TTY mode (User Story 2)
func TestSpinnerDisabledNonTTY(t *testing.T) {
	capsNonTTY := progress.TerminalCapabilities{
		IsTTY:           false,
		SupportsUnicode: false,
		SupportsColor:   false,
	}

	stage := progress.StageInfo{
		Name:        "plan",
		Number:      1,
		TotalStages: 3,
		Status:      progress.StageInProgress,
	}

	display := progress.NewProgressDisplay(capsNonTTY)

	output := captureOutput(func() {
		_ = display.StartStage(stage)
	})

	// Non-TTY mode should just print the message, no spinner
	if !strings.Contains(output, "[1/3]") || !strings.Contains(output, "Plan") {
		t.Errorf("StartStage() non-TTY output = %q, want stage message", output)
	}
}
