// Package validation_test tests continuation prompt generation for incomplete phases.
// Related: internal/validation/prompt.go
// Tags: validation, prompt, phase, continuation, tasks, implementation
package validation

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestListIncompletePhasesWithTasks(t *testing.T) {
	tests := map[string]struct {
		phases       []Phase
		wantContains []string
		wantEmpty    bool
	}{
		"all phases complete": {
			phases: []Phase{
				{
					Name:         "Phase 1",
					TotalTasks:   2,
					CheckedTasks: 2,
					Tasks: []Task{
						{Description: "Task 1", Checked: true},
						{Description: "Task 2", Checked: true},
					},
				},
			},
			wantEmpty: true,
		},
		"one incomplete phase": {
			phases: []Phase{
				{
					Name:         "Setup",
					TotalTasks:   2,
					CheckedTasks: 1,
					Tasks: []Task{
						{Description: "Task 1", Checked: true},
						{Description: "Task 2", Checked: false},
					},
				},
			},
			wantContains: []string{
				"Setup",
				"1/2",
				"- [ ] Task 2",
			},
		},
		"truncates tasks at 5": {
			phases: []Phase{
				{
					Name:         "Large Phase",
					TotalTasks:   8,
					CheckedTasks: 0,
					Tasks: []Task{
						{Description: "Task 1", Checked: false},
						{Description: "Task 2", Checked: false},
						{Description: "Task 3", Checked: false},
						{Description: "Task 4", Checked: false},
						{Description: "Task 5", Checked: false},
						{Description: "Task 6", Checked: false},
						{Description: "Task 7", Checked: false},
						{Description: "Task 8", Checked: false},
					},
				},
			},
			wantContains: []string{
				"Task 1",
				"Task 2",
				"Task 3",
				"Task 4",
				"Task 5",
				"and 3 more",
			},
		},
		"mixed complete and incomplete phases": {
			phases: []Phase{
				{
					Name:         "Complete Phase",
					TotalTasks:   2,
					CheckedTasks: 2,
					Tasks: []Task{
						{Description: "Done 1", Checked: true},
						{Description: "Done 2", Checked: true},
					},
				},
				{
					Name:         "Incomplete Phase",
					TotalTasks:   3,
					CheckedTasks: 1,
					Tasks: []Task{
						{Description: "Task A", Checked: true},
						{Description: "Task B", Checked: false},
						{Description: "Task C", Checked: false},
					},
				},
			},
			wantContains: []string{
				"Incomplete Phase",
				"1/3",
				"- [ ] Task B",
				"- [ ] Task C",
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result := ListIncompletePhasesWithTasks(tc.phases)

			if tc.wantEmpty {
				assert.Empty(t, result)
				return
			}

			for _, want := range tc.wantContains {
				assert.Contains(t, result, want)
			}
		})
	}
}

func TestListIncompletePhasesWithTasks_EmptyPhases(t *testing.T) {
	result := ListIncompletePhasesWithTasks([]Phase{})
	assert.Empty(t, result)
}

func TestListIncompletePhasesWithTasks_ExcludesCompletedTasks(t *testing.T) {
	phases := []Phase{
		{
			Name:         "Mixed Phase",
			TotalTasks:   4,
			CheckedTasks: 2,
			Tasks: []Task{
				{Description: "Complete 1", Checked: true},
				{Description: "Incomplete 1", Checked: false},
				{Description: "Complete 2", Checked: true},
				{Description: "Incomplete 2", Checked: false},
			},
		},
	}

	result := ListIncompletePhasesWithTasks(phases)

	// Should only list unchecked tasks
	assert.Contains(t, result, "Incomplete 1")
	assert.Contains(t, result, "Incomplete 2")
	assert.NotContains(t, result, "Complete 1")
	assert.NotContains(t, result, "Complete 2")
}

func TestGenerateContinuationPrompt(t *testing.T) {
	tests := map[string]struct {
		specDir      string
		phase        string
		phases       []Phase
		wantContains []string
	}{
		"implement phase with remaining tasks": {
			specDir: "specs/001-feature",
			phase:   "implement",
			phases: []Phase{
				{
					Name:         "Implementation",
					TotalTasks:   5,
					CheckedTasks: 2,
					Tasks: []Task{
						{Description: "Task 1", Checked: true},
						{Description: "Task 2", Checked: true},
						{Description: "Task 3", Checked: false},
						{Description: "Task 4", Checked: false},
						{Description: "Task 5", Checked: false},
					},
				},
			},
			wantContains: []string{
				"implement phase is incomplete",
				"3 task(s) remain unchecked",
				"Implementation",
				"specs/001-feature",
				"tasks.md",
			},
		},
		"multiple phases with tasks": {
			specDir: "specs/002-test",
			phase:   "tasks",
			phases: []Phase{
				{
					Name:         "Setup",
					TotalTasks:   2,
					CheckedTasks: 0,
					Tasks: []Task{
						{Description: "Setup 1", Checked: false},
						{Description: "Setup 2", Checked: false},
					},
				},
				{
					Name:         "Build",
					TotalTasks:   1,
					CheckedTasks: 0,
					Tasks: []Task{
						{Description: "Build 1", Checked: false},
					},
				},
			},
			wantContains: []string{
				"tasks phase is incomplete",
				"3 task(s) remain unchecked",
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result := GenerateContinuationPrompt(tc.specDir, tc.phase, tc.phases)

			for _, want := range tc.wantContains {
				assert.Contains(t, result, want)
			}
		})
	}
}

func TestGenerateContinuationPrompt_IncludesTaskList(t *testing.T) {
	phases := []Phase{
		{
			Name:         "Test Phase",
			TotalTasks:   2,
			CheckedTasks: 0,
			Tasks: []Task{
				{Description: "First task", Checked: false},
				{Description: "Second task", Checked: false},
			},
		},
	}

	result := GenerateContinuationPrompt("specs/test", "implement", phases)

	// Should include the incomplete phases list
	assert.Contains(t, result, "Test Phase")
	assert.Contains(t, result, "0/2")
	assert.Contains(t, result, "First task")
	assert.Contains(t, result, "Second task")
}

func TestGenerateContinuationPrompt_CountsTotalUnchecked(t *testing.T) {
	phases := []Phase{
		{
			Name:         "Phase A",
			TotalTasks:   3,
			CheckedTasks: 1,
			Tasks: []Task{
				{Description: "A1", Checked: true},
				{Description: "A2", Checked: false},
				{Description: "A3", Checked: false},
			},
		},
		{
			Name:         "Phase B",
			TotalTasks:   2,
			CheckedTasks: 0,
			Tasks: []Task{
				{Description: "B1", Checked: false},
				{Description: "B2", Checked: false},
			},
		},
	}

	result := GenerateContinuationPrompt("specs/test", "implement", phases)

	// Total unchecked = 2 + 2 = 4
	assert.Contains(t, result, "4 task(s) remain unchecked")
}

func TestPhase_UncheckedTasks(t *testing.T) {
	tests := map[string]struct {
		phase Phase
		want  int
	}{
		"all checked": {
			phase: Phase{TotalTasks: 5, CheckedTasks: 5},
			want:  0,
		},
		"none checked": {
			phase: Phase{TotalTasks: 5, CheckedTasks: 0},
			want:  5,
		},
		"partial": {
			phase: Phase{TotalTasks: 10, CheckedTasks: 7},
			want:  3,
		},
		"empty phase": {
			phase: Phase{TotalTasks: 0, CheckedTasks: 0},
			want:  0,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := tc.phase.UncheckedTasks()
			assert.Equal(t, tc.want, got)
		})
	}
}

// Phase_IsComplete and Phase_Progress are tested in tasks_test.go
