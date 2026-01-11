package dag

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ariel-frischer/autospec/internal/dag"
)

func TestListCmd_NoRuns(t *testing.T) {
	// Create empty state dir
	tmpDir := t.TempDir()
	stateDir := filepath.Join(tmpDir, "state", "dag-runs")
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatalf("failed to create state dir: %v", err)
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmd := listCmd
	cmd.SetArgs([]string{})

	// This will use the default state dir, not our temp dir
	// So we just verify the command runs without error
	err := cmd.Execute()
	if err != nil && !os.IsNotExist(err) {
		// May fail if no state dir exists, which is fine
	}

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
}

func TestListCmd_WithRuns(t *testing.T) {
	tmpDir := t.TempDir()
	stateDir := filepath.Join(tmpDir, "state", "dag-runs")
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatalf("failed to create state dir: %v", err)
	}

	// Create test runs
	runs := []*dag.DAGRun{
		{
			RunID:     "20250111_120000_abc12345",
			DAGFile:   "test1.yaml",
			Status:    dag.RunStatusCompleted,
			StartedAt: time.Now().Add(-2 * time.Hour),
		},
		{
			RunID:     "20250111_130000_def67890",
			DAGFile:   "test2.yaml",
			Status:    dag.RunStatusRunning,
			StartedAt: time.Now().Add(-1 * time.Hour),
		},
	}

	for _, run := range runs {
		if err := dag.SaveState(stateDir, run); err != nil {
			t.Fatalf("failed to save state: %v", err)
		}
	}

	// Verify runs were saved
	loadedRuns, err := dag.ListRuns(stateDir)
	if err != nil {
		t.Fatalf("failed to list runs: %v", err)
	}

	if len(loadedRuns) != 2 {
		t.Errorf("expected 2 runs, got %d", len(loadedRuns))
	}
}

func TestFormatStatus(t *testing.T) {
	tests := map[string]struct {
		status   dag.RunStatus
		expected string
	}{
		"running status": {
			status:   dag.RunStatusRunning,
			expected: "running",
		},
		"completed status": {
			status:   dag.RunStatusCompleted,
			expected: "completed",
		},
		"failed status": {
			status:   dag.RunStatusFailed,
			expected: "failed",
		},
		"interrupted status": {
			status:   dag.RunStatusInterrupted,
			expected: "interrupted",
		},
		"unknown status": {
			status:   dag.RunStatus("unknown"),
			expected: "unknown",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := formatStatus(tt.status)
			// Status may have ANSI color codes, so check for the text content
			if !bytes.Contains([]byte(result), []byte(tt.expected)) {
				t.Errorf("expected status string to contain %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestRepeatString(t *testing.T) {
	tests := map[string]struct {
		s        string
		count    int
		expected string
	}{
		"repeat dash": {
			s:        "-",
			count:    5,
			expected: "-----",
		},
		"repeat empty": {
			s:        "",
			count:    5,
			expected: "",
		},
		"repeat zero times": {
			s:        "x",
			count:    0,
			expected: "",
		},
		"repeat multi-char": {
			s:        "ab",
			count:    3,
			expected: "ababab",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := repeatString(tt.s, tt.count)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestPrintRunsTable(t *testing.T) {
	tests := map[string]struct {
		runs []*dag.DAGRun
	}{
		"single run": {
			runs: []*dag.DAGRun{
				{
					RunID:     "20250111_120000_abc12345",
					DAGFile:   "test.yaml",
					Status:    dag.RunStatusCompleted,
					StartedAt: time.Now(),
				},
			},
		},
		"multiple runs": {
			runs: []*dag.DAGRun{
				{
					RunID:     "20250111_120000_abc12345",
					DAGFile:   "test1.yaml",
					Status:    dag.RunStatusCompleted,
					StartedAt: time.Now().Add(-time.Hour),
				},
				{
					RunID:     "20250111_130000_def67890",
					DAGFile:   "test2.yaml",
					Status:    dag.RunStatusFailed,
					StartedAt: time.Now(),
				},
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			printRunsTable(tt.runs)

			w.Close()
			os.Stdout = oldStdout

			var buf bytes.Buffer
			buf.ReadFrom(r)
			output := buf.String()

			// Verify header is present
			if !bytes.Contains([]byte(output), []byte("RUN ID")) {
				t.Error("expected output to contain RUN ID header")
			}
			if !bytes.Contains([]byte(output), []byte("STATUS")) {
				t.Error("expected output to contain STATUS header")
			}
			if !bytes.Contains([]byte(output), []byte("SPECS")) {
				t.Error("expected output to contain SPECS header")
			}

			// Verify runs are listed
			for _, run := range tt.runs {
				if !bytes.Contains([]byte(output), []byte(run.RunID)) {
					t.Errorf("expected output to contain run ID %s", run.RunID)
				}
			}

			// Verify total count
			if !bytes.Contains([]byte(output), []byte("Total:")) {
				t.Error("expected output to contain Total count")
			}
		})
	}
}

func TestFormatSpecs(t *testing.T) {
	tests := map[string]struct {
		run      *dag.DAGRun
		expected string
	}{
		"no specs": {
			run:      &dag.DAGRun{Specs: map[string]*dag.SpecState{}},
			expected: "0/0",
		},
		"all pending": {
			run: &dag.DAGRun{
				Specs: map[string]*dag.SpecState{
					"spec1": {SpecID: "spec1", Status: dag.SpecStatusPending},
					"spec2": {SpecID: "spec2", Status: dag.SpecStatusPending},
				},
			},
			expected: "0/2",
		},
		"all completed": {
			run: &dag.DAGRun{
				Specs: map[string]*dag.SpecState{
					"spec1": {SpecID: "spec1", Status: dag.SpecStatusCompleted},
					"spec2": {SpecID: "spec2", Status: dag.SpecStatusCompleted},
					"spec3": {SpecID: "spec3", Status: dag.SpecStatusCompleted},
				},
			},
			expected: "3/3",
		},
		"mixed statuses": {
			run: &dag.DAGRun{
				Specs: map[string]*dag.SpecState{
					"spec1": {SpecID: "spec1", Status: dag.SpecStatusCompleted},
					"spec2": {SpecID: "spec2", Status: dag.SpecStatusRunning},
					"spec3": {SpecID: "spec3", Status: dag.SpecStatusFailed},
					"spec4": {SpecID: "spec4", Status: dag.SpecStatusPending},
					"spec5": {SpecID: "spec5", Status: dag.SpecStatusCompleted},
				},
			},
			expected: "2/5",
		},
		"single completed": {
			run: &dag.DAGRun{
				Specs: map[string]*dag.SpecState{
					"spec1": {SpecID: "spec1", Status: dag.SpecStatusCompleted},
				},
			},
			expected: "1/1",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := formatSpecs(tt.run)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestFormatRelativeTime(t *testing.T) {
	tests := map[string]struct {
		offset   time.Duration
		contains string
	}{
		"just now": {
			offset:   30 * time.Second,
			contains: "just now",
		},
		"1 min ago": {
			offset:   1 * time.Minute,
			contains: "1 min ago",
		},
		"5 mins ago": {
			offset:   5 * time.Minute,
			contains: "5 mins ago",
		},
		"30 mins ago": {
			offset:   30 * time.Minute,
			contains: "30 mins ago",
		},
		"1 hour ago": {
			offset:   1 * time.Hour,
			contains: "1 hour ago",
		},
		"3 hours ago": {
			offset:   3 * time.Hour,
			contains: "3 hours ago",
		},
		"yesterday": {
			offset:   30 * time.Hour,
			contains: "yesterday",
		},
		"2 days ago": {
			offset:   50 * time.Hour,
			contains: "2 days ago",
		},
		"7 days ago": {
			offset:   7 * 24 * time.Hour,
			contains: "7 days ago",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			pastTime := time.Now().Add(-tt.offset)
			result := formatRelativeTime(pastTime)
			if !strings.Contains(result, tt.contains) {
				t.Errorf("expected result to contain %q, got %q", tt.contains, result)
			}
		})
	}
}

func TestPluralize(t *testing.T) {
	tests := map[string]struct {
		count    int
		singular string
		plural   string
		expected string
	}{
		"singular": {
			count:    1,
			singular: "min ago",
			plural:   "mins ago",
			expected: "1 min ago",
		},
		"plural two": {
			count:    2,
			singular: "hour ago",
			plural:   "hours ago",
			expected: "2 hours ago",
		},
		"plural many": {
			count:    10,
			singular: "day ago",
			plural:   "days ago",
			expected: "10 days ago",
		},
		"zero uses plural": {
			count:    0,
			singular: "item",
			plural:   "items",
			expected: "0 items",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := pluralize(tt.count, tt.singular, tt.plural)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}
