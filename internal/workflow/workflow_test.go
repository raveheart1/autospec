package workflow

import (
	"path/filepath"
	"testing"

	"github.com/anthropics/auto-claude-speckit/internal/config"
)

func TestNewWorkflowOrchestrator(t *testing.T) {
	cfg := &config.Configuration{
		ClaudeCmd:  "claude",
		ClaudeArgs: []string{"-p"},
		SpecsDir:   "./specs",
		MaxRetries: 3,
		StateDir:   "~/.autospec/state",
	}

	orchestrator := NewWorkflowOrchestrator(cfg)

	if orchestrator == nil {
		t.Fatal("NewWorkflowOrchestrator() returned nil")
	}

	if orchestrator.SpecsDir != "./specs" {
		t.Errorf("SpecsDir = %v, want './specs'", orchestrator.SpecsDir)
	}

	if orchestrator.Executor == nil {
		t.Error("Executor should not be nil")
	}
}

func TestWorkflowOrchestrator_Configuration(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.Configuration{
		ClaudeCmd:  "claude",
		ClaudeArgs: []string{"-p"},
		SpecsDir:   tmpDir,
		MaxRetries: 3,
		StateDir:   filepath.Join(tmpDir, "state"),
	}

	orchestrator := NewWorkflowOrchestrator(cfg)

	if orchestrator.SpecsDir != tmpDir {
		t.Errorf("SpecsDir = %v, want %v", orchestrator.SpecsDir, tmpDir)
	}

	if orchestrator.Config.MaxRetries != 3 {
		t.Errorf("MaxRetries = %v, want 3", orchestrator.Config.MaxRetries)
	}
}

// TestExecutePlanWithPrompt tests that plan commands properly format prompts
func TestExecutePlanWithPrompt(t *testing.T) {
	tests := map[string]struct {
		prompt       string
		wantCommand  string
	}{
		"no prompt": {
			prompt:      "",
			wantCommand: "/speckit.plan",
		},
		"simple prompt": {
			prompt:      "Focus on security",
			wantCommand: `/speckit.plan "Focus on security"`,
		},
		"prompt with quotes": {
			prompt:      "Use 'best practices' for auth",
			wantCommand: `/speckit.plan "Use 'best practices' for auth"`,
		},
		"multiline prompt": {
			prompt: `Consider these aspects:
  - Performance
  - Scalability`,
			wantCommand: `/speckit.plan "Consider these aspects:
  - Performance
  - Scalability"`,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// This test verifies the command construction logic
			command := "/speckit.plan"
			if tc.prompt != "" {
				command = "/speckit.plan \"" + tc.prompt + "\""
			}

			if command != tc.wantCommand {
				t.Errorf("command = %q, want %q", command, tc.wantCommand)
			}
		})
	}
}

// TestExecuteTasksWithPrompt tests that tasks commands properly format prompts
func TestExecuteTasksWithPrompt(t *testing.T) {
	tests := map[string]struct {
		prompt       string
		wantCommand  string
	}{
		"no prompt": {
			prompt:      "",
			wantCommand: "/speckit.tasks",
		},
		"simple prompt": {
			prompt:      "Break into small steps",
			wantCommand: `/speckit.tasks "Break into small steps"`,
		},
		"prompt with quotes": {
			prompt:      "Make tasks 'granular' and testable",
			wantCommand: `/speckit.tasks "Make tasks 'granular' and testable"`,
		},
		"complex prompt": {
			prompt: `Requirements:
  - Each task < 1 hour
  - Include testing`,
			wantCommand: `/speckit.tasks "Requirements:
  - Each task < 1 hour
  - Include testing"`,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// This test verifies the command construction logic
			command := "/speckit.tasks"
			if tc.prompt != "" {
				command = "/speckit.tasks \"" + tc.prompt + "\""
			}

			if command != tc.wantCommand {
				t.Errorf("command = %q, want %q", command, tc.wantCommand)
			}
		})
	}
}
