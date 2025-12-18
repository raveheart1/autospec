// Package lifecycle_test provides performance benchmarks for lifecycle operations.
// Related: /home/ari/repos/autospec/internal/lifecycle/lifecycle.go
// Tags: lifecycle, benchmark, performance

package lifecycle

import (
	"context"
	"testing"
	"time"
)

// benchHandler is a minimal handler for benchmarking.
type benchHandler struct{}

func (b *benchHandler) OnCommandComplete(name string, success bool, duration time.Duration) {}
func (b *benchHandler) OnStageComplete(name string, success bool)                           {}

func BenchmarkRun(b *testing.B) {
	handler := &benchHandler{}
	fn := func() error { return nil }

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_ = Run(handler, "bench", fn)
	}
}

func BenchmarkRunWithHandler(b *testing.B) {
	handler := &benchHandler{}
	fn := func() error { return nil }

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_ = Run(handler, "bench", fn)
	}
}

func BenchmarkRunWithoutHandler(b *testing.B) {
	fn := func() error { return nil }

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_ = Run(nil, "bench", fn)
	}
}

func BenchmarkRunWithContext(b *testing.B) {
	handler := &benchHandler{}
	ctx := context.Background()
	fn := func(ctx context.Context) error { return nil }

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_ = RunWithContext(ctx, handler, "bench", fn)
	}
}

func BenchmarkRunWithContextCancelled(b *testing.B) {
	handler := &benchHandler{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	fn := func(ctx context.Context) error { return nil }

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_ = RunWithContext(ctx, handler, "bench", fn)
	}
}

func BenchmarkRunStage(b *testing.B) {
	handler := &benchHandler{}
	fn := func() error { return nil }

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_ = RunStage(handler, "bench", fn)
	}
}
