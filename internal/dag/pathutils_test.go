package dag

import (
	"path/filepath"
	"testing"
)

func TestNormalizeWorkflowPath(t *testing.T) {
	tests := map[string]struct {
		input    string
		expected string
	}{
		"simple filename": {
			input:    "workflow.yaml",
			expected: "workflow.yaml.state",
		},
		"relative path with single directory": {
			input:    "features/v1.yaml",
			expected: "features-v1.yaml.state",
		},
		"relative path with multiple directories": {
			input:    "dags/features/v2.yaml",
			expected: "dags-features-v2.yaml.state",
		},
		"deeply nested relative path": {
			input:    "a/b/c/d/workflow.yaml",
			expected: "a-b-c-d-workflow.yaml.state",
		},
		"absolute path uses basename only": {
			input:    "/abs/path/workflow.yaml",
			expected: "workflow.yaml.state",
		},
		"absolute path with deep nesting uses basename only": {
			input:    "/home/user/projects/autospec/.autospec/dags/workflow.yaml",
			expected: "workflow.yaml.state",
		},
		"current directory prefix": {
			input:    "./workflow.yaml",
			expected: "workflow.yaml.state",
		},
		"current directory with nested path": {
			input:    "./features/v1.yaml",
			expected: "features-v1.yaml.state",
		},
		"path with redundant separators": {
			input:    "features//v1.yaml",
			expected: "features-v1.yaml.state",
		},
		"path with parent directory references": {
			input:    "features/../other/v1.yaml",
			expected: "other-v1.yaml.state",
		},
		"dag.yaml common filename": {
			input:    "dag.yaml",
			expected: "dag.yaml.state",
		},
		"yml extension": {
			input:    "features/workflow.yml",
			expected: "features-workflow.yml.state",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := NormalizeWorkflowPath(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizeWorkflowPath(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGetStatePathForWorkflow(t *testing.T) {
	tests := map[string]struct {
		stateDir     string
		workflowPath string
		expected     string
	}{
		"simple state dir and workflow": {
			stateDir:     "/tmp/state",
			workflowPath: "workflow.yaml",
			expected:     filepath.Join("/tmp/state", "workflow.yaml.state"),
		},
		"default state dir with relative workflow": {
			stateDir:     ".autospec/state/dag-runs",
			workflowPath: "features/v1.yaml",
			expected:     filepath.Join(".autospec/state/dag-runs", "features-v1.yaml.state"),
		},
		"absolute workflow path": {
			stateDir:     "/home/user/.autospec/state",
			workflowPath: "/abs/path/workflow.yaml",
			expected:     filepath.Join("/home/user/.autospec/state", "workflow.yaml.state"),
		},
		"nested state dir": {
			stateDir:     "/home/user/.autospec/state/dag-runs",
			workflowPath: "dags/features/main.yaml",
			expected:     filepath.Join("/home/user/.autospec/state/dag-runs", "dags-features-main.yaml.state"),
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := GetStatePathForWorkflow(tt.stateDir, tt.workflowPath)
			if result != tt.expected {
				t.Errorf("GetStatePathForWorkflow(%q, %q) = %q, want %q",
					tt.stateDir, tt.workflowPath, result, tt.expected)
			}
		})
	}
}
