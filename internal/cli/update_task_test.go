// Package cli_test tests the update-task command for modifying task status in tasks.yaml files.
// Related: internal/cli/update_task.go
// Tags: cli, task, update, status, yaml, validation
package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestIsValidStatus(t *testing.T) {
	tests := map[string]struct {
		status string
		want   bool
	}{
		"valid Pending":     {status: "Pending", want: true},
		"valid InProgress":  {status: "InProgress", want: true},
		"valid Completed":   {status: "Completed", want: true},
		"valid Blocked":     {status: "Blocked", want: true},
		"invalid lowercase": {status: "pending", want: false},
		"invalid mixed":     {status: "COMPLETED", want: false},
		"invalid status":    {status: "Done", want: false},
		"empty string":      {status: "", want: false},
		"with spaces":       {status: " Pending ", want: false},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := isValidStatus(tc.status)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestTaskIDPattern(t *testing.T) {
	tests := map[string]struct {
		taskID string
		want   bool
	}{
		"valid T001":           {taskID: "T001", want: true},
		"valid T1":             {taskID: "T1", want: true},
		"valid T123":           {taskID: "T123", want: true},
		"valid T99999":         {taskID: "T99999", want: true},
		"invalid lowercase t":  {taskID: "t001", want: false},
		"invalid no number":    {taskID: "T", want: false},
		"invalid with letters": {taskID: "T001a", want: false},
		"invalid prefix":       {taskID: "Task001", want: false},
		"invalid empty":        {taskID: "", want: false},
		"invalid spaces":       {taskID: "T 001", want: false},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := taskIDPattern.MatchString(tc.taskID)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestFindAndUpdateTask_FlatTaskList(t *testing.T) {
	yamlContent := `
tasks:
  - id: T001
    title: First task
    status: Pending
  - id: T002
    title: Second task
    status: InProgress
  - id: T003
    title: Third task
    status: Completed
`

	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(yamlContent), &root))

	// Update T002 to Completed
	prevStatus, found := findAndUpdateTask(&root, "T002", "Completed")

	assert.True(t, found)
	assert.Equal(t, "InProgress", prevStatus)

	// Verify the update by marshaling back
	output, err := yaml.Marshal(&root)
	require.NoError(t, err)
	assert.Contains(t, string(output), "status: Completed")
}

func TestFindAndUpdateTask_NestedPhasesStructure(t *testing.T) {
	yamlContent := `
phases:
  - number: 1
    title: Phase One
    tasks:
      - id: T001
        title: Task in phase 1
        status: Pending
  - number: 2
    title: Phase Two
    tasks:
      - id: T002
        title: Task in phase 2
        status: InProgress
      - id: T003
        title: Another task in phase 2
        status: Pending
`

	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(yamlContent), &root))

	// Update T003 in nested structure
	prevStatus, found := findAndUpdateTask(&root, "T003", "Completed")

	assert.True(t, found)
	assert.Equal(t, "Pending", prevStatus)
}

func TestFindAndUpdateTask_DeeplyNestedStructure(t *testing.T) {
	yamlContent := `
workflow:
  phases:
    phase1:
      tasks:
        - id: T001
          status: Pending
          subtasks:
            - id: T001a
              status: InProgress
`

	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(yamlContent), &root))

	// Update deeply nested task
	prevStatus, found := findAndUpdateTask(&root, "T001", "Completed")

	assert.True(t, found)
	assert.Equal(t, "Pending", prevStatus)
}

func TestFindAndUpdateTask_TaskNotFound(t *testing.T) {
	yamlContent := `
tasks:
  - id: T001
    status: Pending
  - id: T002
    status: InProgress
`

	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(yamlContent), &root))

	prevStatus, found := findAndUpdateTask(&root, "T999", "Completed")

	assert.False(t, found)
	assert.Equal(t, "", prevStatus)
}

func TestFindAndUpdateTask_NilNode(t *testing.T) {
	prevStatus, found := findAndUpdateTask(nil, "T001", "Completed")

	assert.False(t, found)
	assert.Equal(t, "", prevStatus)
}

func TestFindAndUpdateTask_EmptyYAML(t *testing.T) {
	yamlContent := ``

	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(yamlContent), &root))

	prevStatus, found := findAndUpdateTask(&root, "T001", "Completed")

	assert.False(t, found)
	assert.Equal(t, "", prevStatus)
}

func TestFindAndUpdateTask_PreservesYAMLStructure(t *testing.T) {
	yamlContent := `# Tasks file
_meta:
  version: "1.0"
  generator: autospec
phases:
  - number: 1
    title: "Phase One"
    purpose: "Setup infrastructure"
    tasks:
      - id: T001
        title: "Create database schema"
        status: Pending
        type: implementation
        dependencies: []
        acceptance_criteria:
          - Tables are created
          - Indexes are added
`

	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(yamlContent), &root))

	// Update T001
	prevStatus, found := findAndUpdateTask(&root, "T001", "InProgress")
	require.True(t, found)
	assert.Equal(t, "Pending", prevStatus)

	// Marshal and verify structure is preserved
	output, err := yaml.Marshal(&root)
	require.NoError(t, err)
	outputStr := string(output)

	// Check key fields are still present
	assert.Contains(t, outputStr, "_meta:")
	assert.Contains(t, outputStr, "version:")
	assert.Contains(t, outputStr, "generator:")
	assert.Contains(t, outputStr, "phases:")
	assert.Contains(t, outputStr, "acceptance_criteria:")
	assert.Contains(t, outputStr, "status: InProgress")
}

func TestFindAndUpdateTask_IDInNonTaskContext(t *testing.T) {
	// Test that task ID appearing elsewhere (like in a comment or different field)
	// doesn't incorrectly match
	yamlContent := `
metadata:
  id: T001  # This is not a task, just a metadata ID without status
tasks:
  - id: T002
    status: Pending
    description: "References T001 in text"
`

	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(yamlContent), &root))

	// T001 has no status field, so shouldn't be found as a task
	prevStatus, found := findAndUpdateTask(&root, "T001", "Completed")
	assert.False(t, found)
	assert.Equal(t, "", prevStatus)

	// T002 should be found and updated
	prevStatus, found = findAndUpdateTask(&root, "T002", "InProgress")
	assert.True(t, found)
	assert.Equal(t, "Pending", prevStatus)
}

func TestFindAndUpdateTask_SequenceOfMappings(t *testing.T) {
	yamlContent := `
- id: T001
  status: Pending
- id: T002
  status: InProgress
- id: T003
  status: Blocked
`

	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(yamlContent), &root))

	prevStatus, found := findAndUpdateTask(&root, "T003", "Completed")

	assert.True(t, found)
	assert.Equal(t, "Blocked", prevStatus)
}

func TestUpdateTaskIntegration(t *testing.T) {
	// Integration test with temp files
	tmpDir := t.TempDir()
	specsDir := filepath.Join(tmpDir, "specs", "001-test")
	require.NoError(t, os.MkdirAll(specsDir, 0o755))

	// Create a tasks.yaml file
	tasksContent := `_meta:
  version: "1.0"
phases:
  - number: 1
    title: Test Phase
    tasks:
      - id: T001
        title: Test Task
        status: Pending
        type: implementation
`

	tasksPath := filepath.Join(specsDir, "tasks.yaml")
	require.NoError(t, os.WriteFile(tasksPath, []byte(tasksContent), 0o644))

	// Read, update, and verify
	data, err := os.ReadFile(tasksPath)
	require.NoError(t, err)

	var root yaml.Node
	require.NoError(t, yaml.Unmarshal(data, &root))

	prevStatus, found := findAndUpdateTask(&root, "T001", "Completed")
	require.True(t, found)
	assert.Equal(t, "Pending", prevStatus)

	// Write back
	output, err := yaml.Marshal(&root)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(tasksPath, output, 0o644))

	// Read again and verify
	data, err = os.ReadFile(tasksPath)
	require.NoError(t, err)
	assert.Contains(t, string(data), "status: Completed")
}

func TestFindAndUpdateTask_AllValidStatuses(t *testing.T) {
	for _, status := range validStatuses {
		t.Run("update to "+status, func(t *testing.T) {
			yamlContent := `
- id: T001
  status: Pending
`
			var root yaml.Node
			require.NoError(t, yaml.Unmarshal([]byte(yamlContent), &root))

			prevStatus, found := findAndUpdateTask(&root, "T001", status)

			assert.True(t, found)
			assert.Equal(t, "Pending", prevStatus)

			// Verify the new status
			output, err := yaml.Marshal(&root)
			require.NoError(t, err)
			assert.Contains(t, string(output), "status: "+status)
		})
	}
}
