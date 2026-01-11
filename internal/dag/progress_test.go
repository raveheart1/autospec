package dag

import (
	"bytes"
	"sync"
	"testing"
)

func TestProgressTracker_Render(t *testing.T) {
	tests := map[string]struct {
		total        int
		completed    int
		running      int
		failed       int
		blocked      int
		wantRender   string
		wantDetailed string
	}{
		"initial state": {
			total:        5,
			wantRender:   "0/5 specs complete",
			wantDetailed: "0/5 specs complete (0 running, 0 failed, 0 blocked)",
		},
		"partial completion": {
			total:        5,
			completed:    2,
			running:      1,
			wantRender:   "2/5 specs complete",
			wantDetailed: "2/5 specs complete (1 running, 0 failed, 0 blocked)",
		},
		"with failures": {
			total:        5,
			completed:    2,
			failed:       1,
			blocked:      1,
			wantRender:   "2/5 specs complete",
			wantDetailed: "2/5 specs complete (0 running, 1 failed, 1 blocked)",
		},
		"all complete": {
			total:        3,
			completed:    3,
			wantRender:   "3/3 specs complete",
			wantDetailed: "3/3 specs complete (0 running, 0 failed, 0 blocked)",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			pt := NewProgressTracker(tt.total)

			// Apply state changes
			for i := 0; i < tt.completed; i++ {
				pt.MarkRunning()
				pt.MarkCompleted()
			}
			for i := 0; i < tt.running; i++ {
				pt.MarkRunning()
			}
			for i := 0; i < tt.failed; i++ {
				pt.MarkRunning()
				pt.MarkFailed()
			}
			for i := 0; i < tt.blocked; i++ {
				pt.MarkBlocked()
			}

			if got := pt.Render(); got != tt.wantRender {
				t.Errorf("Render() = %q, want %q", got, tt.wantRender)
			}

			if got := pt.RenderDetailed(); got != tt.wantDetailed {
				t.Errorf("RenderDetailed() = %q, want %q", got, tt.wantDetailed)
			}
		})
	}
}

func TestProgressTracker_OnChange(t *testing.T) {
	tests := map[string]struct {
		action    func(pt *ProgressTracker)
		wantStats ProgressStats
	}{
		"mark running": {
			action: func(pt *ProgressTracker) { pt.MarkRunning() },
			wantStats: ProgressStats{
				Total:   5,
				Running: 1,
				Pending: 4,
			},
		},
		"mark completed": {
			action: func(pt *ProgressTracker) {
				pt.MarkRunning()
				pt.MarkCompleted()
			},
			wantStats: ProgressStats{
				Total:     5,
				Completed: 1,
				Pending:   4,
			},
		},
		"mark failed": {
			action: func(pt *ProgressTracker) {
				pt.MarkRunning()
				pt.MarkFailed()
			},
			wantStats: ProgressStats{
				Total:   5,
				Failed:  1,
				Pending: 4,
			},
		},
		"mark blocked": {
			action: func(pt *ProgressTracker) { pt.MarkBlocked() },
			wantStats: ProgressStats{
				Total:   5,
				Blocked: 1,
				Pending: 4,
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			pt := NewProgressTracker(5)
			var receivedStats ProgressStats
			var callCount int

			pt.OnChange(func(stats ProgressStats) {
				receivedStats = stats
				callCount++
			})

			tt.action(pt)

			if callCount == 0 {
				t.Fatal("callback was not called")
			}

			if receivedStats.Total != tt.wantStats.Total {
				t.Errorf("stats.Total = %d, want %d", receivedStats.Total, tt.wantStats.Total)
			}
			if receivedStats.Completed != tt.wantStats.Completed {
				t.Errorf("stats.Completed = %d, want %d", receivedStats.Completed, tt.wantStats.Completed)
			}
			if receivedStats.Running != tt.wantStats.Running {
				t.Errorf("stats.Running = %d, want %d", receivedStats.Running, tt.wantStats.Running)
			}
			if receivedStats.Failed != tt.wantStats.Failed {
				t.Errorf("stats.Failed = %d, want %d", receivedStats.Failed, tt.wantStats.Failed)
			}
			if receivedStats.Blocked != tt.wantStats.Blocked {
				t.Errorf("stats.Blocked = %d, want %d", receivedStats.Blocked, tt.wantStats.Blocked)
			}
		})
	}
}

func TestProgressTracker_ConcurrentUpdates(t *testing.T) {
	const (
		numGoroutines = 100
		opsPerRoutine = 50
	)

	pt := NewProgressTracker(numGoroutines * opsPerRoutine)

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < opsPerRoutine; j++ {
				pt.MarkRunning()
				pt.MarkCompleted()
			}
		}()
	}

	wg.Wait()

	stats := pt.Stats()
	expectedCompleted := numGoroutines * opsPerRoutine

	if stats.Completed != expectedCompleted {
		t.Errorf("expected %d completed, got %d", expectedCompleted, stats.Completed)
	}

	if stats.Running != 0 {
		t.Errorf("expected 0 running, got %d", stats.Running)
	}
}

func TestProgressStats_IsComplete(t *testing.T) {
	tests := map[string]struct {
		stats    ProgressStats
		expected bool
	}{
		"not complete - pending": {
			stats:    ProgressStats{Total: 5, Completed: 2, Pending: 3},
			expected: false,
		},
		"complete - all success": {
			stats:    ProgressStats{Total: 5, Completed: 5},
			expected: true,
		},
		"complete - mixed": {
			stats:    ProgressStats{Total: 5, Completed: 3, Failed: 1, Blocked: 1},
			expected: true,
		},
		"not complete - running": {
			stats:    ProgressStats{Total: 5, Completed: 2, Running: 1, Pending: 2},
			expected: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if got := tt.stats.IsComplete(); got != tt.expected {
				t.Errorf("IsComplete() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestProgressStats_SuccessRate(t *testing.T) {
	tests := map[string]struct {
		stats    ProgressStats
		expected float64
	}{
		"zero total": {
			stats:    ProgressStats{Total: 0},
			expected: 100.0,
		},
		"all complete": {
			stats:    ProgressStats{Total: 10, Completed: 10},
			expected: 100.0,
		},
		"half complete": {
			stats:    ProgressStats{Total: 10, Completed: 5},
			expected: 50.0,
		},
		"none complete": {
			stats:    ProgressStats{Total: 10, Completed: 0},
			expected: 0.0,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if got := tt.stats.SuccessRate(); got != tt.expected {
				t.Errorf("SuccessRate() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestWriteProgressCallback(t *testing.T) {
	tests := map[string]struct {
		stats    ProgressStats
		detailed bool
		expected string
	}{
		"simple format": {
			stats:    ProgressStats{Total: 5, Completed: 2},
			detailed: false,
			expected: "Progress: 2/5 specs complete\n",
		},
		"detailed format": {
			stats:    ProgressStats{Total: 5, Completed: 2, Running: 1, Failed: 1, Blocked: 1},
			detailed: true,
			expected: "Progress: 2/5 specs complete (1 running, 1 failed, 1 blocked)\n",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			var buf bytes.Buffer
			callback := WriteProgressCallback(&buf, tt.detailed)
			callback(tt.stats)

			if got := buf.String(); got != tt.expected {
				t.Errorf("WriteProgressCallback output = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestProgressStats_Render(t *testing.T) {
	tests := map[string]struct {
		stats        ProgressStats
		wantRender   string
		wantDetailed string
	}{
		"basic": {
			stats:        ProgressStats{Total: 5, Completed: 2},
			wantRender:   "2/5 specs complete",
			wantDetailed: "2/5 specs complete (0 running, 0 failed, 0 blocked)",
		},
		"with all states": {
			stats:        ProgressStats{Total: 10, Completed: 5, Running: 2, Failed: 1, Blocked: 2},
			wantRender:   "5/10 specs complete",
			wantDetailed: "5/10 specs complete (2 running, 1 failed, 2 blocked)",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if got := tt.stats.Render(); got != tt.wantRender {
				t.Errorf("Render() = %q, want %q", got, tt.wantRender)
			}
			if got := tt.stats.RenderDetailed(); got != tt.wantDetailed {
				t.Errorf("RenderDetailed() = %q, want %q", got, tt.wantDetailed)
			}
		})
	}
}

func TestNewProgressTrackerFromState(t *testing.T) {
	run := &DAGRun{
		Specs: map[string]*SpecState{
			"spec-1": {Status: SpecStatusCompleted},
			"spec-2": {Status: SpecStatusCompleted},
			"spec-3": {Status: SpecStatusRunning},
			"spec-4": {Status: SpecStatusFailed},
			"spec-5": {Status: SpecStatusBlocked},
			"spec-6": {Status: SpecStatusPending},
		},
	}

	pt := NewProgressTrackerFromState(run)
	stats := pt.Stats()

	if stats.Total != 6 {
		t.Errorf("Total = %d, want 6", stats.Total)
	}
	if stats.Completed != 2 {
		t.Errorf("Completed = %d, want 2", stats.Completed)
	}
	if stats.Running != 1 {
		t.Errorf("Running = %d, want 1", stats.Running)
	}
	if stats.Failed != 1 {
		t.Errorf("Failed = %d, want 1", stats.Failed)
	}
	if stats.Blocked != 1 {
		t.Errorf("Blocked = %d, want 1", stats.Blocked)
	}
	if stats.Pending != 1 {
		t.Errorf("Pending = %d, want 1", stats.Pending)
	}
}
