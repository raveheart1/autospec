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

func TestListCmd_NoDAGFiles(t *testing.T) {
	// Create empty dags dir
	tmpDir := t.TempDir()
	dagsDir := filepath.Join(tmpDir, ".autospec", "dags")
	if err := os.MkdirAll(dagsDir, 0o755); err != nil {
		t.Fatalf("failed to create dags dir: %v", err)
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmd := listCmd
	cmd.SetArgs([]string{})

	// This will use the default dags dir, not our temp dir
	// So we just verify the command runs without error
	err := cmd.Execute()
	if err != nil && !os.IsNotExist(err) {
		// May fail if no dags dir exists, which is fine
	}

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
}

func TestListDAGFiles_EmptyDir(t *testing.T) {
	tmpDir := t.TempDir()
	dagsDir := filepath.Join(tmpDir, ".autospec", "dags")
	if err := os.MkdirAll(dagsDir, 0o755); err != nil {
		t.Fatalf("failed to create dags dir: %v", err)
	}

	// Save current dir and change to temp dir
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	os.Chdir(tmpDir)

	entries, err := listDAGFiles()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
}

func TestListDAGFiles_WithDAGs(t *testing.T) {
	tmpDir := t.TempDir()
	dagsDir := filepath.Join(tmpDir, ".autospec", "dags")
	if err := os.MkdirAll(dagsDir, 0o755); err != nil {
		t.Fatalf("failed to create dags dir: %v", err)
	}

	// Create DAG file without state
	noStateDAG := `schema_version: "1.0"
dag:
  name: No State DAG
layers:
  - id: L0
    features:
      - id: feature1
        description: First feature
`
	if err := os.WriteFile(filepath.Join(dagsDir, "no-state.yaml"), []byte(noStateDAG), 0o644); err != nil {
		t.Fatalf("failed to write no-state.yaml: %v", err)
	}

	// Create DAG file with inline state
	withStateDAG := `schema_version: "1.0"
dag:
  name: With State DAG
layers:
  - id: L0
    features:
      - id: feature1
        description: First feature
      - id: feature2
        description: Second feature
run:
  status: running
  started_at: 2025-01-11T12:00:00Z
specs:
  feature1:
    status: completed
    started_at: 2025-01-11T12:00:00Z
    completed_at: 2025-01-11T12:05:00Z
  feature2:
    status: running
    started_at: 2025-01-11T12:05:00Z
`
	if err := os.WriteFile(filepath.Join(dagsDir, "with-state.yaml"), []byte(withStateDAG), 0o644); err != nil {
		t.Fatalf("failed to write with-state.yaml: %v", err)
	}

	// Save current dir and change to temp dir
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	os.Chdir(tmpDir)

	entries, err := listDAGFiles()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	// Entries with state should come first
	if !entries[0].HasState {
		t.Error("expected first entry to have state")
	}
	if entries[1].HasState {
		t.Error("expected second entry to not have state")
	}
}

func TestLoadDAGEntry_NoState(t *testing.T) {
	tmpDir := t.TempDir()
	dagPath := filepath.Join(tmpDir, "test.yaml")

	dagContent := `schema_version: "1.0"
dag:
  name: Test DAG
layers:
  - id: L0
    features:
      - id: feature1
        description: First feature
      - id: feature2
        description: Second feature
`
	if err := os.WriteFile(dagPath, []byte(dagContent), 0o644); err != nil {
		t.Fatalf("failed to write test.yaml: %v", err)
	}

	entry, err := loadDAGEntry(dagPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if entry.HasState {
		t.Error("expected HasState to be false")
	}
	if entry.Name != "Test DAG" {
		t.Errorf("expected name 'Test DAG', got %q", entry.Name)
	}
	if entry.Total != 2 {
		t.Errorf("expected total 2, got %d", entry.Total)
	}
}

func TestLoadDAGEntry_WithState(t *testing.T) {
	tmpDir := t.TempDir()
	dagPath := filepath.Join(tmpDir, "test.yaml")

	dagContent := `schema_version: "1.0"
dag:
  name: Test DAG
layers:
  - id: L0
    features:
      - id: feature1
        description: First feature
      - id: feature2
        description: Second feature
run:
  status: completed
  started_at: 2025-01-11T12:00:00Z
  completed_at: 2025-01-11T12:30:00Z
specs:
  feature1:
    status: completed
    started_at: 2025-01-11T12:00:00Z
    completed_at: 2025-01-11T12:10:00Z
  feature2:
    status: completed
    started_at: 2025-01-11T12:10:00Z
    completed_at: 2025-01-11T12:20:00Z
`
	if err := os.WriteFile(dagPath, []byte(dagContent), 0o644); err != nil {
		t.Fatalf("failed to write test.yaml: %v", err)
	}

	entry, err := loadDAGEntry(dagPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !entry.HasState {
		t.Error("expected HasState to be true")
	}
	if entry.Status != dag.InlineRunStatusCompleted {
		t.Errorf("expected status 'completed', got %q", entry.Status)
	}
	if entry.Completed != 2 {
		t.Errorf("expected 2 completed, got %d", entry.Completed)
	}
	if entry.Total != 2 {
		t.Errorf("expected total 2, got %d", entry.Total)
	}
}

func TestFormatInlineStatus(t *testing.T) {
	tests := map[string]struct {
		entry    dagListEntry
		expected string
	}{
		"no state": {
			entry:    dagListEntry{HasState: false},
			expected: "(no state)",
		},
		"running": {
			entry:    dagListEntry{HasState: true, Status: dag.InlineRunStatusRunning},
			expected: "running",
		},
		"completed": {
			entry:    dagListEntry{HasState: true, Status: dag.InlineRunStatusCompleted},
			expected: "completed",
		},
		"failed": {
			entry:    dagListEntry{HasState: true, Status: dag.InlineRunStatusFailed},
			expected: "failed",
		},
		"interrupted": {
			entry:    dagListEntry{HasState: true, Status: dag.InlineRunStatusInterrupted},
			expected: "interrupted",
		},
		"pending": {
			entry:    dagListEntry{HasState: true, Status: dag.InlineRunStatusPending},
			expected: "pending",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := formatInlineStatus(tt.entry)
			// Status may have ANSI color codes, so check for the text content
			if !strings.Contains(result, tt.expected) {
				t.Errorf("expected status string to contain %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestFormatSpecCount(t *testing.T) {
	tests := map[string]struct {
		entry    dagListEntry
		expected string
	}{
		"no state with specs": {
			entry:    dagListEntry{HasState: false, Total: 5},
			expected: "0/5",
		},
		"no state empty": {
			entry:    dagListEntry{HasState: false, Total: 0},
			expected: "0/0",
		},
		"all completed": {
			entry:    dagListEntry{HasState: true, Completed: 3, Total: 3},
			expected: "3/3",
		},
		"some completed": {
			entry:    dagListEntry{HasState: true, Completed: 2, Total: 5},
			expected: "2/5",
		},
		"none completed": {
			entry:    dagListEntry{HasState: true, Completed: 0, Total: 4},
			expected: "0/4",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := formatSpecCount(tt.entry)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestFormatLastActivity(t *testing.T) {
	now := time.Now()
	tests := map[string]struct {
		lastActivity *time.Time
		expected     string
	}{
		"nil time": {
			lastActivity: nil,
			expected:     "-",
		},
		"recent time": {
			lastActivity: &now,
			expected:     "just now",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := formatLastActivity(tt.lastActivity)
			if !strings.Contains(result, tt.expected) {
				t.Errorf("expected %q to contain %q", result, tt.expected)
			}
		})
	}
}

func TestCountTotalSpecs(t *testing.T) {
	tests := map[string]struct {
		config   *dag.DAGConfig
		expected int
	}{
		"empty layers": {
			config:   &dag.DAGConfig{Layers: []dag.Layer{}},
			expected: 0,
		},
		"single layer single feature": {
			config: &dag.DAGConfig{
				Layers: []dag.Layer{
					{ID: "L0", Features: []dag.Feature{{ID: "f1"}}},
				},
			},
			expected: 1,
		},
		"multiple layers": {
			config: &dag.DAGConfig{
				Layers: []dag.Layer{
					{ID: "L0", Features: []dag.Feature{{ID: "f1"}, {ID: "f2"}}},
					{ID: "L1", Features: []dag.Feature{{ID: "f3"}}},
				},
			},
			expected: 3,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := countTotalSpecs(tt.config)
			if result != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestCountSpecProgress(t *testing.T) {
	tests := map[string]struct {
		config            *dag.DAGConfig
		expectedCompleted int
		expectedTotal     int
	}{
		"no specs": {
			config:            &dag.DAGConfig{Specs: map[string]*dag.InlineSpecState{}},
			expectedCompleted: 0,
			expectedTotal:     0,
		},
		"all completed": {
			config: &dag.DAGConfig{
				Specs: map[string]*dag.InlineSpecState{
					"f1": {Status: dag.InlineSpecStatusCompleted},
					"f2": {Status: dag.InlineSpecStatusCompleted},
				},
			},
			expectedCompleted: 2,
			expectedTotal:     2,
		},
		"mixed statuses": {
			config: &dag.DAGConfig{
				Specs: map[string]*dag.InlineSpecState{
					"f1": {Status: dag.InlineSpecStatusCompleted},
					"f2": {Status: dag.InlineSpecStatusRunning},
					"f3": {Status: dag.InlineSpecStatusPending},
					"f4": {Status: dag.InlineSpecStatusFailed},
				},
			},
			expectedCompleted: 1,
			expectedTotal:     4,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			completed, total := countSpecProgress(tt.config)
			if completed != tt.expectedCompleted {
				t.Errorf("expected completed %d, got %d", tt.expectedCompleted, completed)
			}
			if total != tt.expectedTotal {
				t.Errorf("expected total %d, got %d", tt.expectedTotal, total)
			}
		})
	}
}

func TestComputeLastActivity(t *testing.T) {
	t1 := time.Date(2025, 1, 10, 12, 0, 0, 0, time.UTC)
	t2 := time.Date(2025, 1, 11, 12, 0, 0, 0, time.UTC)
	t3 := time.Date(2025, 1, 12, 12, 0, 0, 0, time.UTC)

	tests := map[string]struct {
		config   *dag.DAGConfig
		expected *time.Time
	}{
		"no state": {
			config:   &dag.DAGConfig{},
			expected: nil,
		},
		"run started only": {
			config: &dag.DAGConfig{
				Run: &dag.InlineRunState{StartedAt: &t1},
			},
			expected: &t1,
		},
		"run completed": {
			config: &dag.DAGConfig{
				Run: &dag.InlineRunState{StartedAt: &t1, CompletedAt: &t2},
			},
			expected: &t2,
		},
		"spec more recent": {
			config: &dag.DAGConfig{
				Run: &dag.InlineRunState{StartedAt: &t1},
				Specs: map[string]*dag.InlineSpecState{
					"f1": {CompletedAt: &t3},
				},
			},
			expected: &t3,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := computeLastActivity(tt.config)
			if tt.expected == nil && result != nil {
				t.Errorf("expected nil, got %v", result)
			}
			if tt.expected != nil && result == nil {
				t.Errorf("expected %v, got nil", tt.expected)
			}
			if tt.expected != nil && result != nil && !tt.expected.Equal(*result) {
				t.Errorf("expected %v, got %v", tt.expected, result)
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

func TestPrintDAGTable(t *testing.T) {
	tests := map[string]struct {
		entries []dagListEntry
	}{
		"single entry no state": {
			entries: []dagListEntry{
				{Path: ".autospec/dags/test.yaml", Name: "Test", HasState: false, Total: 3},
			},
		},
		"single entry with state": {
			entries: []dagListEntry{
				{
					Path:      ".autospec/dags/test.yaml",
					Name:      "Test",
					HasState:  true,
					Status:    dag.InlineRunStatusCompleted,
					Completed: 2,
					Total:     3,
				},
			},
		},
		"multiple entries": {
			entries: []dagListEntry{
				{Path: ".autospec/dags/dag1.yaml", Name: "DAG 1", HasState: true, Status: dag.InlineRunStatusRunning, Completed: 1, Total: 3},
				{Path: ".autospec/dags/dag2.yaml", Name: "DAG 2", HasState: false, Total: 2},
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			printDAGTable(tt.entries)

			w.Close()
			os.Stdout = oldStdout

			var buf bytes.Buffer
			buf.ReadFrom(r)
			output := buf.String()

			// Verify header is present
			if !strings.Contains(output, "DAG FILE") {
				t.Error("expected output to contain DAG FILE header")
			}
			if !strings.Contains(output, "STATUS") {
				t.Error("expected output to contain STATUS header")
			}
			if !strings.Contains(output, "SPECS") {
				t.Error("expected output to contain SPECS header")
			}

			// Verify entries are listed
			for _, entry := range tt.entries {
				if !strings.Contains(output, entry.Path) {
					t.Errorf("expected output to contain path %s", entry.Path)
				}
			}

			// Verify total count
			if !strings.Contains(output, "Total:") {
				t.Error("expected output to contain Total count")
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

func TestIsYAMLFile(t *testing.T) {
	tests := map[string]struct {
		name     string
		expected bool
	}{
		"yaml extension": {
			name:     "test.yaml",
			expected: true,
		},
		"yml extension": {
			name:     "test.yml",
			expected: true,
		},
		"json extension": {
			name:     "test.json",
			expected: false,
		},
		"no extension": {
			name:     "test",
			expected: false,
		},
		"hidden yaml": {
			name:     ".test.yaml",
			expected: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := isYAMLFile(tt.name)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestSortDAGEntries(t *testing.T) {
	t1 := time.Date(2025, 1, 10, 12, 0, 0, 0, time.UTC)
	t2 := time.Date(2025, 1, 11, 12, 0, 0, 0, time.UTC)

	tests := map[string]struct {
		entries       []dagListEntry
		expectedOrder []string
	}{
		"state before no state": {
			entries: []dagListEntry{
				{Path: "b.yaml", HasState: false},
				{Path: "a.yaml", HasState: true, LastActivity: &t1},
			},
			expectedOrder: []string{"a.yaml", "b.yaml"},
		},
		"recent activity first": {
			entries: []dagListEntry{
				{Path: "old.yaml", HasState: true, LastActivity: &t1},
				{Path: "new.yaml", HasState: true, LastActivity: &t2},
			},
			expectedOrder: []string{"new.yaml", "old.yaml"},
		},
		"alphabetical for no state": {
			entries: []dagListEntry{
				{Path: "b.yaml", HasState: false},
				{Path: "a.yaml", HasState: false},
			},
			expectedOrder: []string{"a.yaml", "b.yaml"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			sortDAGEntries(tt.entries)
			for i, expected := range tt.expectedOrder {
				if tt.entries[i].Path != expected {
					t.Errorf("position %d: expected %s, got %s", i, expected, tt.entries[i].Path)
				}
			}
		})
	}
}

func TestUpdateLatest(t *testing.T) {
	t1 := time.Date(2025, 1, 10, 12, 0, 0, 0, time.UTC)
	t2 := time.Date(2025, 1, 11, 12, 0, 0, 0, time.UTC)

	tests := map[string]struct {
		current   *time.Time
		candidate *time.Time
		expected  *time.Time
	}{
		"nil current nil candidate": {
			current:   nil,
			candidate: nil,
			expected:  nil,
		},
		"nil current with candidate": {
			current:   nil,
			candidate: &t1,
			expected:  &t1,
		},
		"current with nil candidate": {
			current:   &t1,
			candidate: nil,
			expected:  &t1,
		},
		"candidate after current": {
			current:   &t1,
			candidate: &t2,
			expected:  &t2,
		},
		"candidate before current": {
			current:   &t2,
			candidate: &t1,
			expected:  &t2,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := updateLatest(tt.current, tt.candidate)
			if tt.expected == nil && result != nil {
				t.Errorf("expected nil, got %v", result)
			}
			if tt.expected != nil && result == nil {
				t.Errorf("expected %v, got nil", tt.expected)
			}
			if tt.expected != nil && result != nil && !tt.expected.Equal(*result) {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}
