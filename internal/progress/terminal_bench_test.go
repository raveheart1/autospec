// Package progress_test benchmarks terminal capability detection and symbol selection performance.
// Related: internal/progress/terminal.go
// Tags: progress, benchmark, performance, terminal, capabilities
package progress_test

import (
	"testing"

	"github.com/ariel-frischer/autospec/internal/progress"
)

// BenchmarkDetectTerminalCapabilities verifies terminal detection meets <10ms performance contract
func BenchmarkDetectTerminalCapabilities(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = progress.DetectTerminalCapabilities()
	}
}

// BenchmarkSelectSymbols verifies symbol selection is fast
func BenchmarkSelectSymbols(b *testing.B) {
	caps := progress.TerminalCapabilities{
		IsTTY:           true,
		SupportsColor:   true,
		SupportsUnicode: true,
		Width:           80,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = progress.SelectSymbols(caps)
	}
}
