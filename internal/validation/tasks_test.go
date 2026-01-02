// Package validation_test tests tasks.md parsing and task completion tracking.
// Related: internal/validation/tasks.go
// Tags: validation, tasks, phase, markdown, parsing, completion
package validation

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCountUncheckedTasks(t *testing.T) {
	tests := map[string]struct {
		content string
		want    int
		wantErr bool
	}{
		"all tasks checked": {
			content: `# Tasks
- [x] Task 1
- [X] Task 2
* [x] Task 3
`,
			want:    0,
			wantErr: false,
		},
		"all tasks unchecked": {
			content: `# Tasks
- [ ] Task 1
- [ ] Task 2
* [ ] Task 3
`,
			want:    3,
			wantErr: false,
		},
		"mixed tasks": {
			content: `# Tasks
- [x] Task 1
- [ ] Task 2
- [X] Task 3
- [ ] Task 4
`,
			want:    2,
			wantErr: false,
		},
		"no tasks": {
			content: `# Tasks
Just text, no tasks here.
`,
			want:    0,
			wantErr: false,
		},
		"tasks with indentation": {
			content: `# Tasks
- [ ] Task 1
  - [ ] Subtask 1.1
  - [x] Subtask 1.2
- [x] Task 2
`,
			want:    2,
			wantErr: false,
		},
		"file doesn't exist": {
			content: "",
			want:    0,
			wantErr: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if tc.wantErr {
				// Test with non-existent file
				got, err := CountUncheckedTasks("/nonexistent/tasks.md")
				if err == nil {
					t.Error("CountUncheckedTasks() expected error, got nil")
				}
				if got != 0 {
					t.Errorf("CountUncheckedTasks() = %v, want 0 on error", got)
				}
				return
			}

			// Create temp file with content
			tmpDir := t.TempDir()
			tasksPath := filepath.Join(tmpDir, "tasks.md")
			if err := os.WriteFile(tasksPath, []byte(tc.content), 0o644); err != nil {
				t.Fatal(err)
			}

			got, err := CountUncheckedTasks(tasksPath)
			if err != nil {
				t.Errorf("CountUncheckedTasks() unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("CountUncheckedTasks() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestValidateTasksComplete(t *testing.T) {
	tests := map[string]struct {
		content string
		want    bool
		wantErr bool
	}{
		"all complete": {
			content: `# Tasks
- [x] Task 1
- [X] Task 2
`,
			want:    true,
			wantErr: false,
		},
		"some incomplete": {
			content: `# Tasks
- [x] Task 1
- [ ] Task 2
`,
			want:    false,
			wantErr: false,
		},
		"all incomplete": {
			content: `# Tasks
- [ ] Task 1
- [ ] Task 2
`,
			want:    false,
			wantErr: false,
		},
		"no tasks": {
			content: `# Tasks`,
			want:    true,
			wantErr: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			tmpDir := t.TempDir()
			tasksPath := filepath.Join(tmpDir, "tasks.md")
			if err := os.WriteFile(tasksPath, []byte(tc.content), 0o644); err != nil {
				t.Fatal(err)
			}

			got, err := ValidateTasksComplete(tasksPath)
			if (err != nil) != tc.wantErr {
				t.Errorf("ValidateTasksComplete() error = %v, wantErr %v", err, tc.wantErr)
			}
			if got != tc.want {
				t.Errorf("ValidateTasksComplete() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestParseTasksByPhase(t *testing.T) {
	tests := map[string]struct {
		content     string
		wantPhases  int
		wantErr     bool
		checkPhase0 func(t *testing.T, phase *Phase)
	}{
		"single phase with tasks": {
			content: `# Tasks
## Phase 1: Setup
- [ ] Task 1
- [x] Task 2
- [ ] Task 3
`,
			wantPhases: 1,
			wantErr:    false,
			checkPhase0: func(t *testing.T, phase *Phase) {
				if phase.Name != "Phase 1: Setup" {
					t.Errorf("Phase name = %v, want 'Phase 1: Setup'", phase.Name)
				}
				if phase.TotalTasks != 3 {
					t.Errorf("TotalTasks = %v, want 3", phase.TotalTasks)
				}
				if phase.CheckedTasks != 1 {
					t.Errorf("CheckedTasks = %v, want 1", phase.CheckedTasks)
				}
				if unchecked := phase.UncheckedTasks(); unchecked != 2 {
					t.Errorf("UncheckedTasks() = %v, want 2", unchecked)
				}
			},
		},
		"multiple phases": {
			content: `# Tasks
## Phase 0: Research
- [x] Research task

## Phase 1: Foundation
- [ ] Foundation task 1
- [ ] Foundation task 2
`,
			wantPhases: 2,
			wantErr:    false,
			checkPhase0: func(t *testing.T, phase *Phase) {
				if phase.Name != "Phase 0: Research" {
					t.Errorf("Phase name = %v, want 'Phase 0: Research'", phase.Name)
				}
				if !phase.IsComplete() {
					t.Error("Phase 0 should be complete")
				}
			},
		},
		"phase with nested tasks": {
			content: `# Tasks
## Phase 1: Setup
- [ ] Main task
  - [ ] Subtask 1
  - [x] Subtask 2
- [x] Another task
`,
			wantPhases: 1,
			wantErr:    false,
			checkPhase0: func(t *testing.T, phase *Phase) {
				if phase.TotalTasks != 4 {
					t.Errorf("TotalTasks = %v, want 4 (includes subtasks)", phase.TotalTasks)
				}
			},
		},
		"empty file": {
			content:    ``,
			wantPhases: 0,
			wantErr:    false,
		},
		"no phases just tasks": {
			content: `# Tasks
- [ ] Orphan task
`,
			wantPhases: 1, // Creates "Uncategorized" phase for orphan tasks
			wantErr:    false,
			checkPhase0: func(t *testing.T, phase *Phase) {
				if phase.Name != "Uncategorized" {
					t.Errorf("Phase name = %v, want 'Uncategorized'", phase.Name)
				}
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			tmpDir := t.TempDir()
			tasksPath := filepath.Join(tmpDir, "tasks.md")
			if err := os.WriteFile(tasksPath, []byte(tc.content), 0o644); err != nil {
				t.Fatal(err)
			}

			phases, err := ParseTasksByPhase(tasksPath)
			if (err != nil) != tc.wantErr {
				t.Errorf("ParseTasksByPhase() error = %v, wantErr %v", err, tc.wantErr)
				return
			}

			if len(phases) != tc.wantPhases {
				t.Errorf("Got %v phases, want %v", len(phases), tc.wantPhases)
				return
			}

			if tc.checkPhase0 != nil && len(phases) > 0 {
				tc.checkPhase0(t, &phases[0])
			}
		})
	}
}

func TestPhase_Progress(t *testing.T) {
	tests := map[string]struct {
		phase Phase
		want  float64
	}{
		"all complete": {
			phase: Phase{TotalTasks: 10, CheckedTasks: 10},
			want:  1.0,
		},
		"half complete": {
			phase: Phase{TotalTasks: 10, CheckedTasks: 5},
			want:  0.5,
		},
		"none complete": {
			phase: Phase{TotalTasks: 10, CheckedTasks: 0},
			want:  0.0,
		},
		"no tasks": {
			phase: Phase{TotalTasks: 0, CheckedTasks: 0},
			want:  1.0,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := tc.phase.Progress()
			if got != tc.want {
				t.Errorf("Progress() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestPhase_IsComplete(t *testing.T) {
	tests := map[string]struct {
		phase Phase
		want  bool
	}{
		"complete": {
			phase: Phase{TotalTasks: 5, CheckedTasks: 5},
			want:  true,
		},
		"incomplete": {
			phase: Phase{TotalTasks: 5, CheckedTasks: 3},
			want:  false,
		},
		"no tasks": {
			phase: Phase{TotalTasks: 0, CheckedTasks: 0},
			want:  true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := tc.phase.IsComplete()
			if got != tc.want {
				t.Errorf("IsComplete() = %v, want %v", got, tc.want)
			}
		})
	}
}
