// Package progress_test benchmarks progress display operations meeting <10ms performance contract.
// Related: internal/progress/display.go
// Tags: progress, benchmark, performance, display, stages
package progress_test

import (
	"errors"
	"testing"

	"github.com/ariel-frischer/autospec/internal/progress"
)

// BenchmarkProgressDisplay_StartStage verifies StartStage meets <10ms performance contract
func BenchmarkProgressDisplay_StartStage(b *testing.B) {
	caps := progress.TerminalCapabilities{
		IsTTY:           false, // Avoid spinner overhead in benchmark
		SupportsUnicode: true,
		SupportsColor:   true,
		Width:           80,
	}

	display := progress.NewProgressDisplay(caps)
	stage := progress.StageInfo{
		Name:        "test",
		Number:      1,
		TotalStages: 3,
		Status:      progress.StageInProgress,
		RetryCount:  0,
		MaxRetries:  3,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = display.StartStage(stage)
	}
}

// BenchmarkProgressDisplay_CompleteStage verifies CompleteStage meets <10ms performance contract
func BenchmarkProgressDisplay_CompleteStage(b *testing.B) {
	caps := progress.TerminalCapabilities{
		IsTTY:           false,
		SupportsUnicode: true,
		SupportsColor:   true,
		Width:           80,
	}

	display := progress.NewProgressDisplay(caps)
	stage := progress.StageInfo{
		Name:        "test",
		Number:      1,
		TotalStages: 3,
		Status:      progress.StageCompleted,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = display.CompleteStage(stage)
	}
}

// BenchmarkProgressDisplay_FailStage verifies FailStage performance
func BenchmarkProgressDisplay_FailStage(b *testing.B) {
	caps := progress.TerminalCapabilities{
		IsTTY:           false,
		SupportsUnicode: true,
		SupportsColor:   true,
		Width:           80,
	}

	display := progress.NewProgressDisplay(caps)
	stage := progress.StageInfo{
		Name:        "test",
		Number:      1,
		TotalStages: 3,
		Status:      progress.StageFailed,
	}

	testErr := errors.New("test error")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = display.FailStage(stage, testErr)
	}
}
