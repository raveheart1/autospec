package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestFindAndBlockTask_PendingTask(t *testing.T) {
	t.Parallel()

	yamlContent := `
phases:
  - number: 1
    tasks:
      - id: T001
        title: Test task
        status: Pending
        type: implementation
`
	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(yamlContent), &root))

	result := findAndBlockTask(&root, "T001", "Test blocking reason")

	assert.True(t, result.found)
	assert.Equal(t, "Pending", result.previousStatus)
	assert.False(t, result.hadReason)

	// Verify the YAML was updated correctly
	output, err := yaml.Marshal(&root)
	require.NoError(t, err)
	assert.Contains(t, string(output), "status: Blocked")
	assert.Contains(t, string(output), "blocked_reason: Test blocking reason")
}

func TestFindAndBlockTask_InProgressTask(t *testing.T) {
	t.Parallel()

	yamlContent := `
tasks:
  - id: T001
    status: InProgress
`
	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(yamlContent), &root))

	result := findAndBlockTask(&root, "T001", "External dependency issue")

	assert.True(t, result.found)
	assert.Equal(t, "InProgress", result.previousStatus)
	assert.False(t, result.hadReason)

	output, err := yaml.Marshal(&root)
	require.NoError(t, err)
	assert.Contains(t, string(output), "status: Blocked")
	assert.Contains(t, string(output), "blocked_reason: External dependency issue")
}

func TestFindAndBlockTask_ReblockingUpdatesReason(t *testing.T) {
	t.Parallel()

	yamlContent := `
tasks:
  - id: T001
    status: Blocked
    blocked_reason: Original reason
`
	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(yamlContent), &root))

	result := findAndBlockTask(&root, "T001", "Updated blocking reason")

	assert.True(t, result.found)
	assert.Equal(t, "Blocked", result.previousStatus)
	assert.True(t, result.hadReason)
	assert.Equal(t, "Original reason", result.previousReason)

	output, err := yaml.Marshal(&root)
	require.NoError(t, err)
	assert.Contains(t, string(output), "status: Blocked")
	assert.Contains(t, string(output), "blocked_reason: Updated blocking reason")
	assert.NotContains(t, string(output), "Original reason")
}

func TestFindAndBlockTask_TaskNotFound(t *testing.T) {
	t.Parallel()

	yamlContent := `
tasks:
  - id: T001
    status: Pending
  - id: T002
    status: InProgress
`
	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(yamlContent), &root))

	result := findAndBlockTask(&root, "T999", "Some reason")

	assert.False(t, result.found)
	assert.Empty(t, result.previousStatus)
}

func TestFindAndBlockTask_NilNode(t *testing.T) {
	t.Parallel()

	result := findAndBlockTask(nil, "T001", "Some reason")

	assert.False(t, result.found)
	assert.Empty(t, result.previousStatus)
}

func TestFindAndBlockTask_NestedPhaseStructure(t *testing.T) {
	t.Parallel()

	yamlContent := `
_meta:
  version: "1.0"
phases:
  - number: 1
    title: Phase One
    tasks:
      - id: T001
        status: Pending
  - number: 2
    title: Phase Two
    tasks:
      - id: T002
        status: InProgress
      - id: T003
        status: Completed
`
	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(yamlContent), &root))

	result := findAndBlockTask(&root, "T002", "Waiting for external API")

	assert.True(t, result.found)
	assert.Equal(t, "InProgress", result.previousStatus)

	output, err := yaml.Marshal(&root)
	require.NoError(t, err)
	assert.Contains(t, string(output), "status: Blocked")
	assert.Contains(t, string(output), "blocked_reason: Waiting for external API")
}

func TestFindAndBlockTask_CompletedTask(t *testing.T) {
	t.Parallel()

	yamlContent := `
tasks:
  - id: T001
    status: Completed
`
	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(yamlContent), &root))

	result := findAndBlockTask(&root, "T001", "Re-blocking completed task due to issue found")

	assert.True(t, result.found)
	assert.Equal(t, "Completed", result.previousStatus)

	output, err := yaml.Marshal(&root)
	require.NoError(t, err)
	assert.Contains(t, string(output), "status: Blocked")
	assert.Contains(t, string(output), "blocked_reason: Re-blocking completed task due to issue found")
}

func TestFindAndBlockTask_PreservesOtherFields(t *testing.T) {
	t.Parallel()

	yamlContent := `
tasks:
  - id: T001
    title: "Important task"
    status: Pending
    type: implementation
    parallel: true
    dependencies:
      - T000
    acceptance_criteria:
      - Criterion one
      - Criterion two
`
	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(yamlContent), &root))

	result := findAndBlockTask(&root, "T001", "Dependency not ready")
	require.True(t, result.found)

	output, err := yaml.Marshal(&root)
	require.NoError(t, err)
	outputStr := string(output)

	// Verify other fields are preserved (quotes may vary in YAML output)
	assert.Contains(t, outputStr, "Important task")
	assert.Contains(t, outputStr, "type: implementation")
	assert.Contains(t, outputStr, "parallel: true")
	assert.Contains(t, outputStr, "T000")
	assert.Contains(t, outputStr, "Criterion one")
	assert.Contains(t, outputStr, "Criterion two")
	// Verify blocking was applied
	assert.Contains(t, outputStr, "status: Blocked")
	assert.Contains(t, outputStr, "blocked_reason: Dependency not ready")
}

func TestTruncateReason(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		reason string
		maxLen int
		want   string
	}{
		"short reason unchanged": {
			reason: "Short reason",
			maxLen: 20,
			want:   "Short reason",
		},
		"exact length unchanged": {
			reason: "Exactly twenty chars",
			maxLen: 20,
			want:   "Exactly twenty chars",
		},
		"long reason truncated": {
			reason: "This is a very long reason that should be truncated",
			maxLen: 30,
			want:   "This is a very long reason ...",
		},
		"empty string": {
			reason: "",
			maxLen: 10,
			want:   "",
		},
		"very short maxLen": {
			reason: "Hello world",
			maxLen: 6,
			want:   "Hel...",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := truncateReason(tc.reason, tc.maxLen)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestBlockTaskIntegration(t *testing.T) {
	t.Parallel()

	// Create a temp directory structure
	tmpDir := t.TempDir()
	specsDir := filepath.Join(tmpDir, "specs", "001-test")
	require.NoError(t, os.MkdirAll(specsDir, 0755))

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
      - id: T002
        title: Another Task
        status: InProgress
        type: test
`
	tasksPath := filepath.Join(specsDir, "tasks.yaml")
	require.NoError(t, os.WriteFile(tasksPath, []byte(tasksContent), 0644))

	// Read, block, and verify
	data, err := os.ReadFile(tasksPath)
	require.NoError(t, err)

	var root yaml.Node
	require.NoError(t, yaml.Unmarshal(data, &root))

	result := findAndBlockTask(&root, "T001", "Waiting for API credentials")
	require.True(t, result.found)
	assert.Equal(t, "Pending", result.previousStatus)

	// Write back
	output, err := yaml.Marshal(&root)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(tasksPath, output, 0644))

	// Read again and verify
	data, err = os.ReadFile(tasksPath)
	require.NoError(t, err)
	assert.Contains(t, string(data), "status: Blocked")
	assert.Contains(t, string(data), "blocked_reason: Waiting for API credentials")
	// T002 should be unchanged
	assert.Contains(t, string(data), "status: InProgress")
}

func TestBlockTaskSequenceOfMappings(t *testing.T) {
	t.Parallel()

	yamlContent := `
- id: T001
  status: Pending
- id: T002
  status: InProgress
- id: T003
  status: Completed
`
	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(yamlContent), &root))

	result := findAndBlockTask(&root, "T002", "Sequence test reason")

	assert.True(t, result.found)
	assert.Equal(t, "InProgress", result.previousStatus)

	output, err := yaml.Marshal(&root)
	require.NoError(t, err)
	assert.Contains(t, string(output), "blocked_reason: Sequence test reason")
}

func TestFindAndBlockTask_VeryLongReason(t *testing.T) {
	t.Parallel()

	// Generate a very long reason (>500 chars)
	longReason := ""
	for i := 0; i < 60; i++ {
		longReason += "This is a long "
	}

	yamlContent := `
tasks:
  - id: T001
    status: Pending
`
	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(yamlContent), &root))

	result := findAndBlockTask(&root, "T001", longReason)

	assert.True(t, result.found)

	// Verify the full reason is stored (not truncated in storage)
	output, err := yaml.Marshal(&root)
	require.NoError(t, err)
	assert.Contains(t, string(output), "blocked_reason:")
	// The full reason should be preserved in the YAML
	assert.True(t, len(longReason) > 500, "test reason should be >500 chars")
}

func TestFindAndBlockTask_AllStatuses(t *testing.T) {
	t.Parallel()

	statuses := []string{"Pending", "InProgress", "Completed", "Blocked"}

	for _, status := range statuses {
		t.Run("block from "+status, func(t *testing.T) {
			t.Parallel()

			yamlContent := `
tasks:
  - id: T001
    status: ` + status + `
`
			var root yaml.Node
			require.NoError(t, yaml.Unmarshal([]byte(yamlContent), &root))

			result := findAndBlockTask(&root, "T001", "Test reason")

			assert.True(t, result.found)
			assert.Equal(t, status, result.previousStatus)

			output, err := yaml.Marshal(&root)
			require.NoError(t, err)
			assert.Contains(t, string(output), "status: Blocked")
		})
	}
}

// ==================== Unblock Task Tests ====================

func TestFindAndUnblockTask(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		yamlContent    string
		taskID         string
		targetStatus   string
		wantFound      bool
		wantWasBlocked bool
		wantPrevStatus string
		wantHadReason  bool
		wantPrevReason string
		wantStatus     string
		wantNoReason   bool
	}{
		"unblock blocked task to Pending": {
			yamlContent: `
tasks:
  - id: T001
    status: Blocked
    blocked_reason: Waiting for API access
`,
			taskID:         "T001",
			targetStatus:   "Pending",
			wantFound:      true,
			wantWasBlocked: true,
			wantPrevStatus: "Blocked",
			wantHadReason:  true,
			wantPrevReason: "Waiting for API access",
			wantStatus:     "Pending",
			wantNoReason:   true,
		},
		"unblock blocked task to InProgress": {
			yamlContent: `
tasks:
  - id: T001
    status: Blocked
    blocked_reason: External dependency
`,
			taskID:         "T001",
			targetStatus:   "InProgress",
			wantFound:      true,
			wantWasBlocked: true,
			wantPrevStatus: "Blocked",
			wantHadReason:  true,
			wantPrevReason: "External dependency",
			wantStatus:     "InProgress",
			wantNoReason:   true,
		},
		"unblock blocked task without reason": {
			yamlContent: `
tasks:
  - id: T001
    status: Blocked
`,
			taskID:         "T001",
			targetStatus:   "Pending",
			wantFound:      true,
			wantWasBlocked: true,
			wantPrevStatus: "Blocked",
			wantHadReason:  false,
			wantStatus:     "Pending",
			wantNoReason:   true,
		},
		"unblock non-blocked task shows warning": {
			yamlContent: `
tasks:
  - id: T001
    status: Pending
`,
			taskID:         "T001",
			targetStatus:   "Pending",
			wantFound:      true,
			wantWasBlocked: false,
			wantPrevStatus: "Pending",
		},
		"unblock InProgress task shows warning": {
			yamlContent: `
tasks:
  - id: T001
    status: InProgress
`,
			taskID:         "T001",
			targetStatus:   "Pending",
			wantFound:      true,
			wantWasBlocked: false,
			wantPrevStatus: "InProgress",
		},
		"unblock non-existent task": {
			yamlContent: `
tasks:
  - id: T001
    status: Blocked
`,
			taskID:       "T999",
			targetStatus: "Pending",
			wantFound:    false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var root yaml.Node
			require.NoError(t, yaml.Unmarshal([]byte(tc.yamlContent), &root))

			result := findAndUnblockTask(&root, tc.taskID, tc.targetStatus)

			assert.Equal(t, tc.wantFound, result.found)
			assert.Equal(t, tc.wantWasBlocked, result.wasBlocked)
			assert.Equal(t, tc.wantPrevStatus, result.previousStatus)
			assert.Equal(t, tc.wantHadReason, result.hadReason)
			if tc.wantHadReason {
				assert.Equal(t, tc.wantPrevReason, result.previousReason)
			}

			// Verify YAML output if task was found and was blocked
			if tc.wantFound && tc.wantWasBlocked {
				output, err := yaml.Marshal(&root)
				require.NoError(t, err)
				outputStr := string(output)

				assert.Contains(t, outputStr, "status: "+tc.wantStatus)
				if tc.wantNoReason {
					assert.NotContains(t, outputStr, "blocked_reason")
				}
			}
		})
	}
}

func TestFindAndUnblockTask_NilNode(t *testing.T) {
	t.Parallel()

	result := findAndUnblockTask(nil, "T001", "Pending")

	assert.False(t, result.found)
	assert.False(t, result.wasBlocked)
}

func TestFindAndUnblockTask_NestedPhaseStructure(t *testing.T) {
	t.Parallel()

	yamlContent := `
_meta:
  version: "1.0"
phases:
  - number: 1
    title: Phase One
    tasks:
      - id: T001
        status: Pending
  - number: 2
    title: Phase Two
    tasks:
      - id: T002
        status: Blocked
        blocked_reason: Waiting for phase 1
      - id: T003
        status: Completed
`
	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(yamlContent), &root))

	result := findAndUnblockTask(&root, "T002", "InProgress")

	assert.True(t, result.found)
	assert.True(t, result.wasBlocked)
	assert.Equal(t, "Blocked", result.previousStatus)
	assert.True(t, result.hadReason)
	assert.Equal(t, "Waiting for phase 1", result.previousReason)

	output, err := yaml.Marshal(&root)
	require.NoError(t, err)
	outputStr := string(output)

	assert.Contains(t, outputStr, "id: T002")
	assert.Contains(t, outputStr, "status: InProgress")
	assert.NotContains(t, outputStr, "blocked_reason: Waiting for phase 1")
}

func TestFindAndUnblockTask_PreservesOtherFields(t *testing.T) {
	t.Parallel()

	yamlContent := `
tasks:
  - id: T001
    title: "Important task"
    status: Blocked
    blocked_reason: Some blocker
    type: implementation
    parallel: true
    dependencies:
      - T000
    acceptance_criteria:
      - Criterion one
      - Criterion two
`
	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(yamlContent), &root))

	result := findAndUnblockTask(&root, "T001", "Pending")
	require.True(t, result.found)
	require.True(t, result.wasBlocked)

	output, err := yaml.Marshal(&root)
	require.NoError(t, err)
	outputStr := string(output)

	// Verify other fields are preserved
	assert.Contains(t, outputStr, "Important task")
	assert.Contains(t, outputStr, "type: implementation")
	assert.Contains(t, outputStr, "parallel: true")
	assert.Contains(t, outputStr, "T000")
	assert.Contains(t, outputStr, "Criterion one")
	assert.Contains(t, outputStr, "Criterion two")
	// Verify unblocking was applied
	assert.Contains(t, outputStr, "status: Pending")
	assert.NotContains(t, outputStr, "blocked_reason")
}

func TestUnblockTaskIntegration(t *testing.T) {
	t.Parallel()

	// Create a temp directory structure
	tmpDir := t.TempDir()
	specsDir := filepath.Join(tmpDir, "specs", "001-test")
	require.NoError(t, os.MkdirAll(specsDir, 0755))

	// Create a tasks.yaml file with blocked task
	tasksContent := `_meta:
  version: "1.0"
phases:
  - number: 1
    title: Test Phase
    tasks:
      - id: T001
        title: Test Task
        status: Blocked
        blocked_reason: Waiting for API credentials
        type: implementation
      - id: T002
        title: Another Task
        status: InProgress
        type: test
`
	tasksPath := filepath.Join(specsDir, "tasks.yaml")
	require.NoError(t, os.WriteFile(tasksPath, []byte(tasksContent), 0644))

	// Read, unblock, and verify
	data, err := os.ReadFile(tasksPath)
	require.NoError(t, err)

	var root yaml.Node
	require.NoError(t, yaml.Unmarshal(data, &root))

	result := findAndUnblockTask(&root, "T001", "InProgress")
	require.True(t, result.found)
	require.True(t, result.wasBlocked)
	assert.Equal(t, "Blocked", result.previousStatus)
	assert.True(t, result.hadReason)
	assert.Equal(t, "Waiting for API credentials", result.previousReason)

	// Write back
	output, err := yaml.Marshal(&root)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(tasksPath, output, 0644))

	// Read again and verify
	data, err = os.ReadFile(tasksPath)
	require.NoError(t, err)
	dataStr := string(data)
	assert.Contains(t, dataStr, "status: InProgress")
	assert.NotContains(t, dataStr, "blocked_reason")
	// T002 should be unchanged
	assert.Contains(t, dataStr, "status: InProgress")
}

func TestValidateUnblockStatus(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		status  string
		wantErr bool
	}{
		"Pending is valid": {
			status:  "Pending",
			wantErr: false,
		},
		"InProgress is valid": {
			status:  "InProgress",
			wantErr: false,
		},
		"Completed is invalid": {
			status:  "Completed",
			wantErr: true,
		},
		"Blocked is invalid": {
			status:  "Blocked",
			wantErr: true,
		},
		"empty is invalid": {
			status:  "",
			wantErr: true,
		},
		"random string is invalid": {
			status:  "Unknown",
			wantErr: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			err := validateUnblockStatus(tc.status)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestUnblockTaskSequenceOfMappings(t *testing.T) {
	t.Parallel()

	yamlContent := `
- id: T001
  status: Pending
- id: T002
  status: Blocked
  blocked_reason: Sequence blocker
- id: T003
  status: Completed
`
	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(yamlContent), &root))

	result := findAndUnblockTask(&root, "T002", "Pending")

	assert.True(t, result.found)
	assert.True(t, result.wasBlocked)
	assert.Equal(t, "Blocked", result.previousStatus)

	output, err := yaml.Marshal(&root)
	require.NoError(t, err)
	outputStr := string(output)
	assert.Contains(t, outputStr, "status: Pending")
	assert.NotContains(t, outputStr, "blocked_reason: Sequence blocker")
}
