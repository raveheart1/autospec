package dag

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewWatcher(t *testing.T) {
	tests := map[string]struct {
		stateDir string
		runID    string
		opts     []WatcherOption
	}{
		"creates watcher with defaults": {
			stateDir: "/tmp/state",
			runID:    "20260111_120000_abc12345",
			opts:     nil,
		},
		"creates watcher with custom output": {
			stateDir: "/tmp/state",
			runID:    "20260111_120000_abc12345",
			opts:     []WatcherOption{WithOutput(&bytes.Buffer{})},
		},
		"creates watcher with custom interval": {
			stateDir: "/tmp/state",
			runID:    "20260111_120000_abc12345",
			opts:     []WatcherOption{WithInterval(5 * time.Second)},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			w := NewWatcher(tc.stateDir, tc.runID, tc.opts...)

			if w == nil {
				t.Fatal("NewWatcher() returned nil")
			}

			if w.stateDir != tc.stateDir {
				t.Errorf("stateDir = %q, want %q", w.stateDir, tc.stateDir)
			}

			if w.runID != tc.runID {
				t.Errorf("runID = %q, want %q", w.runID, tc.runID)
			}
		})
	}
}

func TestFormatSpecStatus(t *testing.T) {
	tests := map[string]struct {
		status       SpecStatus
		wantContains string
	}{
		"pending status": {
			status:       SpecStatusPending,
			wantContains: "pending",
		},
		"running status": {
			status:       SpecStatusRunning,
			wantContains: "running",
		},
		"completed status": {
			status:       SpecStatusCompleted,
			wantContains: "completed",
		},
		"failed status": {
			status:       SpecStatusFailed,
			wantContains: "failed",
		},
		"blocked status": {
			status:       SpecStatusBlocked,
			wantContains: "blocked",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result := formatSpecStatus(tc.status)

			if !strings.Contains(result, tc.wantContains) {
				t.Errorf("formatSpecStatus(%q) = %q, want containing %q", tc.status, result, tc.wantContains)
			}
		})
	}
}

func TestFormatRunStatus(t *testing.T) {
	tests := map[string]struct {
		status       RunStatus
		wantContains string
	}{
		"running status": {
			status:       RunStatusRunning,
			wantContains: "running",
		},
		"completed status": {
			status:       RunStatusCompleted,
			wantContains: "completed",
		},
		"failed status": {
			status:       RunStatusFailed,
			wantContains: "failed",
		},
		"interrupted status": {
			status:       RunStatusInterrupted,
			wantContains: "interrupted",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result := formatRunStatus(tc.status)

			if !strings.Contains(result, tc.wantContains) {
				t.Errorf("formatRunStatus(%q) = %q, want containing %q", tc.status, result, tc.wantContains)
			}
		})
	}
}

func TestFormatDurationHuman(t *testing.T) {
	tests := map[string]struct {
		duration time.Duration
		want     string
	}{
		"seconds only": {
			duration: 45 * time.Second,
			want:     "45s",
		},
		"one minute": {
			duration: 60 * time.Second,
			want:     "1m 0s",
		},
		"minutes and seconds": {
			duration: 2*time.Minute + 30*time.Second,
			want:     "2m 30s",
		},
		"one hour": {
			duration: 60 * time.Minute,
			want:     "1h 0m",
		},
		"hours and minutes": {
			duration: 2*time.Hour + 15*time.Minute,
			want:     "2h 15m",
		},
		"zero seconds": {
			duration: 0,
			want:     "0s",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result := formatDurationHuman(tc.duration)

			if result != tc.want {
				t.Errorf("formatDurationHuman(%v) = %q, want %q", tc.duration, result, tc.want)
			}
		})
	}
}

func TestFormatProgress(t *testing.T) {
	tests := map[string]struct {
		spec *SpecState
		want string
	}{
		"with current task": {
			spec: &SpecState{CurrentTask: "5/10"},
			want: "5/10",
		},
		"with current stage only": {
			spec: &SpecState{CurrentStage: "implement"},
			want: "implement",
		},
		"with both task and stage": {
			spec: &SpecState{CurrentTask: "3/8", CurrentStage: "implement"},
			want: "3/8",
		},
		"with neither": {
			spec: &SpecState{},
			want: "-",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result := formatProgress(tc.spec)

			if result != tc.want {
				t.Errorf("formatProgress() = %q, want %q", result, tc.want)
			}
		})
	}
}

func TestFormatDuration(t *testing.T) {
	now := time.Now()
	oneMinAgo := now.Add(-1 * time.Minute)
	twoMinsAgo := now.Add(-2 * time.Minute)

	tests := map[string]struct {
		spec         *SpecState
		wantContains string
	}{
		"not started": {
			spec:         &SpecState{},
			wantContains: "-",
		},
		"running spec": {
			spec: &SpecState{
				StartedAt: &twoMinsAgo,
			},
			wantContains: "m",
		},
		"completed spec": {
			spec: &SpecState{
				StartedAt:   &twoMinsAgo,
				CompletedAt: &oneMinAgo,
			},
			wantContains: "1m",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result := formatDuration(tc.spec)

			if !strings.Contains(result, tc.wantContains) {
				t.Errorf("formatDuration() = %q, want containing %q", result, tc.wantContains)
			}
		})
	}
}

func TestFormatRelativeTime(t *testing.T) {
	now := time.Now()

	tests := map[string]struct {
		time         time.Time
		wantContains string
	}{
		"just now": {
			time:         now.Add(-30 * time.Second),
			wantContains: "just now",
		},
		"one minute ago": {
			time:         now.Add(-1 * time.Minute),
			wantContains: "1 min ago",
		},
		"several minutes ago": {
			time:         now.Add(-15 * time.Minute),
			wantContains: "15 mins ago",
		},
		"one hour ago": {
			time:         now.Add(-1 * time.Hour),
			wantContains: "1 hour ago",
		},
		"several hours ago": {
			time:         now.Add(-3 * time.Hour),
			wantContains: "3 hours ago",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result := formatRelativeTime(tc.time)

			if !strings.Contains(result, tc.wantContains) {
				t.Errorf("formatRelativeTime(%v) = %q, want containing %q", tc.time, result, tc.wantContains)
			}
		})
	}
}

func TestFormatLastUpdate(t *testing.T) {
	now := time.Now()
	oneMinAgo := now.Add(-1 * time.Minute)

	tests := map[string]struct {
		spec         *SpecState
		wantContains string
	}{
		"no timestamps": {
			spec:         &SpecState{},
			wantContains: "-",
		},
		"with started only": {
			spec: &SpecState{
				StartedAt: &oneMinAgo,
			},
			wantContains: "1 min ago",
		},
		"with completed": {
			spec: &SpecState{
				StartedAt:   &now,
				CompletedAt: &oneMinAgo,
			},
			wantContains: "1 min ago",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result := formatLastUpdate(tc.spec)

			if !strings.Contains(result, tc.wantContains) {
				t.Errorf("formatLastUpdate() = %q, want containing %q", result, tc.wantContains)
			}
		})
	}
}

func TestWatcher_BuildTableLines(t *testing.T) {
	now := time.Now()
	fiveMinAgo := now.Add(-5 * time.Minute)

	tests := map[string]struct {
		run              *DAGRun
		wantContainsAll  []string
		wantMinLineCount int
	}{
		"single pending spec": {
			run: &DAGRun{
				RunID:  "20260111_120000_abc12345",
				Status: RunStatusRunning,
				Specs: map[string]*SpecState{
					"001-feature": {
						SpecID: "001-feature",
						Status: SpecStatusPending,
					},
				},
			},
			wantContainsAll:  []string{"DAG Run:", "20260111_120000_abc12345", "SPEC", "STATUS", "001-feature", "pending"},
			wantMinLineCount: 6,
		},
		"multiple specs in different states": {
			run: &DAGRun{
				RunID:  "20260111_120000_abc12345",
				Status: RunStatusRunning,
				Specs: map[string]*SpecState{
					"001-first": {
						SpecID:    "001-first",
						Status:    SpecStatusCompleted,
						StartedAt: &fiveMinAgo,
					},
					"002-second": {
						SpecID:       "002-second",
						Status:       SpecStatusRunning,
						CurrentStage: "implement",
						CurrentTask:  "3/8",
						StartedAt:    &now,
					},
					"003-third": {
						SpecID: "003-third",
						Status: SpecStatusPending,
					},
				},
			},
			wantContainsAll:  []string{"001-first", "002-second", "003-third", "completed", "running", "pending", "3/8"},
			wantMinLineCount: 8,
		},
		"failed run": {
			run: &DAGRun{
				RunID:  "20260111_120000_abc12345",
				Status: RunStatusFailed,
				Specs: map[string]*SpecState{
					"001-feature": {
						SpecID: "001-feature",
						Status: SpecStatusFailed,
					},
				},
			},
			wantContainsAll:  []string{"failed"},
			wantMinLineCount: 6,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			w := NewWatcher("/tmp/state", tc.run.RunID)
			lines := w.buildTableLines(tc.run)

			if len(lines) < tc.wantMinLineCount {
				t.Errorf("got %d lines, want at least %d", len(lines), tc.wantMinLineCount)
			}

			fullOutput := strings.Join(lines, "\n")
			for _, want := range tc.wantContainsAll {
				if !strings.Contains(fullOutput, want) {
					t.Errorf("output missing %q:\n%s", want, fullOutput)
				}
			}
		})
	}
}

func TestWatcher_FormatSpecRow(t *testing.T) {
	now := time.Now()
	twoMinsAgo := now.Add(-2 * time.Minute)

	tests := map[string]struct {
		specID       string
		spec         *SpecState
		wantContains []string
	}{
		"pending spec": {
			specID: "001-feature",
			spec: &SpecState{
				Status: SpecStatusPending,
			},
			wantContains: []string{"001-feature", "pending", "-"},
		},
		"running spec with progress": {
			specID: "002-feature",
			spec: &SpecState{
				Status:      SpecStatusRunning,
				CurrentTask: "5/10",
				StartedAt:   &twoMinsAgo,
			},
			wantContains: []string{"002-feature", "running", "5/10", "2m"},
		},
		"completed spec": {
			specID: "003-feature",
			spec: &SpecState{
				Status:      SpecStatusCompleted,
				StartedAt:   &twoMinsAgo,
				CompletedAt: &now,
			},
			wantContains: []string{"003-feature", "completed", "2m"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			w := NewWatcher("/tmp/state", "test-run")
			result := w.formatSpecRow(tc.specID, tc.spec)

			for _, want := range tc.wantContains {
				if !strings.Contains(result, want) {
					t.Errorf("formatSpecRow() = %q, missing %q", result, want)
				}
			}
		})
	}
}

func TestWatcher_RenderTable(t *testing.T) {
	tests := map[string]struct {
		run          *DAGRun
		wantContains []string
	}{
		"renders running DAG": {
			run: &DAGRun{
				RunID:     "20260111_120000_abc12345",
				Status:    RunStatusRunning,
				StartedAt: time.Now(),
				Specs: map[string]*SpecState{
					"001-feature": {
						SpecID: "001-feature",
						Status: SpecStatusRunning,
					},
				},
			},
			wantContains: []string{"DAG Run:", "001-feature", "running"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			tmpDir := t.TempDir()
			stateDir := filepath.Join(tmpDir, ".autospec", "state", "dag-runs")

			// Save state file
			if err := SaveState(stateDir, tc.run); err != nil {
				t.Fatalf("SaveState() error: %v", err)
			}

			var buf bytes.Buffer
			w := NewWatcher(stateDir, tc.run.RunID, WithOutput(&buf))

			if err := w.renderTable(); err != nil {
				t.Fatalf("renderTable() error: %v", err)
			}

			output := buf.String()
			for _, want := range tc.wantContains {
				if !strings.Contains(output, want) {
					t.Errorf("output missing %q:\n%s", want, output)
				}
			}
		})
	}
}

func TestWatcher_RenderTableError(t *testing.T) {
	tests := map[string]struct {
		runID   string
		wantErr bool
	}{
		"returns error for missing run": {
			runID:   "nonexistent-run-id",
			wantErr: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			tmpDir := t.TempDir()
			stateDir := filepath.Join(tmpDir, ".autospec", "state", "dag-runs")

			// Ensure directory exists but no state file
			if err := os.MkdirAll(stateDir, 0o755); err != nil {
				t.Fatalf("failed to create dir: %v", err)
			}

			var buf bytes.Buffer
			w := NewWatcher(stateDir, tc.runID, WithOutput(&buf))

			err := w.renderTable()

			if tc.wantErr && err == nil {
				t.Error("expected error but got nil")
			}

			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestWatcher_ClearPreviousOutput(t *testing.T) {
	tests := map[string]struct {
		lastRowCount int
		wantPrefix   string
	}{
		"clears no rows when lastRowCount is 0": {
			lastRowCount: 0,
			wantPrefix:   "",
		},
		"clears rows with ANSI escape codes": {
			lastRowCount: 3,
			wantPrefix:   "\033[1A\033[2K",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var buf bytes.Buffer
			w := NewWatcher("/tmp/state", "test-run", WithOutput(&buf))
			w.lastRowCount = tc.lastRowCount

			w.clearPreviousOutput()

			output := buf.String()
			if tc.wantPrefix != "" && !strings.HasPrefix(output, tc.wantPrefix) {
				t.Errorf("output prefix = %q, want %q", output, tc.wantPrefix)
			}

			if tc.lastRowCount == 0 && output != "" {
				t.Errorf("expected empty output for lastRowCount=0, got %q", output)
			}
		})
	}
}

func TestWatcher_WatchContextCancellation(t *testing.T) {
	tests := map[string]struct {
		cancelAfter time.Duration
	}{
		"cancellation stops watch": {
			cancelAfter: 50 * time.Millisecond,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			tmpDir := t.TempDir()
			stateDir := filepath.Join(tmpDir, ".autospec", "state", "dag-runs")

			run := &DAGRun{
				RunID:     "test-run",
				Status:    RunStatusRunning,
				StartedAt: time.Now(),
				Specs: map[string]*SpecState{
					"001-feature": {SpecID: "001-feature", Status: SpecStatusRunning},
				},
			}

			if err := SaveState(stateDir, run); err != nil {
				t.Fatalf("SaveState() error: %v", err)
			}

			var buf bytes.Buffer
			w := NewWatcher(stateDir, run.RunID, WithOutput(&buf), WithInterval(10*time.Millisecond))

			ctx, cancel := context.WithCancel(context.Background())

			done := make(chan error, 1)
			go func() {
				done <- w.Watch(ctx)
			}()

			time.Sleep(tc.cancelAfter)
			cancel()

			select {
			case err := <-done:
				if err != nil {
					t.Errorf("Watch() error: %v", err)
				}
			case <-time.After(1 * time.Second):
				t.Error("Watch() did not exit after context cancellation")
			}
		})
	}
}

func TestWithOutput(t *testing.T) {
	tests := map[string]struct {
		writer *bytes.Buffer
	}{
		"sets custom output": {
			writer: &bytes.Buffer{},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			w := NewWatcher("/tmp/state", "test-run", WithOutput(tc.writer))

			if w.out != tc.writer {
				t.Error("WithOutput did not set custom writer")
			}
		})
	}
}

func TestWithInterval(t *testing.T) {
	tests := map[string]struct {
		interval time.Duration
	}{
		"sets 1 second interval": {
			interval: 1 * time.Second,
		},
		"sets 5 second interval": {
			interval: 5 * time.Second,
		},
		"sets 500ms interval": {
			interval: 500 * time.Millisecond,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			w := NewWatcher("/tmp/state", "test-run", WithInterval(tc.interval))

			if w.interval != tc.interval {
				t.Errorf("interval = %v, want %v", w.interval, tc.interval)
			}
		})
	}
}
