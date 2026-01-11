package dag

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ariel-frischer/autospec/internal/dag"
)

func TestRunCmd_ValidateFileArg(t *testing.T) {
	tmpDir := t.TempDir()

	validFile := filepath.Join(tmpDir, "valid.yaml")
	if err := os.WriteFile(validFile, []byte("test"), 0o644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	tests := map[string]struct {
		path        string
		expectError bool
		errContains string
	}{
		"valid file": {
			path:        validFile,
			expectError: false,
		},
		"nonexistent file": {
			path:        filepath.Join(tmpDir, "nonexistent.yaml"),
			expectError: true,
			errContains: "file not found",
		},
		"directory instead of file": {
			path:        tmpDir,
			expectError: true,
			errContains: "directory",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			err := validateFileArg(tt.path)

			if tt.expectError && err == nil {
				t.Error("expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if tt.expectError && tt.errContains != "" && err != nil {
				if !bytes.Contains([]byte(err.Error()), []byte(tt.errContains)) {
					t.Errorf("error should contain %q, got %q", tt.errContains, err.Error())
				}
			}
		})
	}
}

func TestRunCmd_DryRunFlagParsing(t *testing.T) {
	tests := map[string]struct {
		args         []string
		expectDryRun bool
		expectForce  bool
	}{
		"no flags": {
			args:         []string{"file.yaml"},
			expectDryRun: false,
			expectForce:  false,
		},
		"dry-run enabled": {
			args:         []string{"file.yaml", "--dry-run"},
			expectDryRun: true,
			expectForce:  false,
		},
		"force enabled": {
			args:         []string{"file.yaml", "--force"},
			expectDryRun: false,
			expectForce:  true,
		},
		"both flags": {
			args:         []string{"file.yaml", "--dry-run", "--force"},
			expectDryRun: true,
			expectForce:  true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Reset flags for each test
			runCmd.Flags().Set("dry-run", "false")
			runCmd.Flags().Set("force", "false")

			if err := runCmd.ParseFlags(tt.args); err != nil {
				t.Fatalf("failed to parse flags: %v", err)
			}

			dryRun, _ := runCmd.Flags().GetBool("dry-run")
			force, _ := runCmd.Flags().GetBool("force")

			if dryRun != tt.expectDryRun {
				t.Errorf("expected dry-run=%v, got %v", tt.expectDryRun, dryRun)
			}
			if force != tt.expectForce {
				t.Errorf("expected force=%v, got %v", tt.expectForce, force)
			}
		})
	}
}

func TestFormatDagValidationErrors(t *testing.T) {
	tests := map[string]struct {
		errs      []error
		wantCount int
	}{
		"single error": {
			errs:      []error{os.ErrNotExist},
			wantCount: 1,
		},
		"multiple errors": {
			errs:      []error{os.ErrNotExist, os.ErrPermission},
			wantCount: 2,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			err := formatDagValidationErrors("test.yaml", tt.errs)
			if err == nil {
				t.Error("expected error but got nil")
			}
		})
	}
}

func TestPrintRunSuccess(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printRunSuccess("test-run-id")

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)

	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("test-run-id")) {
		t.Error("expected output to contain run ID")
	}
}

func TestPrintRunFailure(t *testing.T) {
	tests := map[string]struct {
		runID string
	}{
		"with run ID": {
			runID: "test-run-id",
		},
		"without run ID": {
			runID: "",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			err := printRunFailure(tt.runID, os.ErrNotExist)
			if err == nil {
				t.Error("expected error to be returned")
			}
		})
	}
}

func TestRunCmd_CommandDefinition(t *testing.T) {
	tests := map[string]struct {
		checkFunc func() bool
		desc      string
	}{
		"has use": {
			checkFunc: func() bool { return runCmd.Use != "" },
			desc:      "command should have Use field set",
		},
		"has short description": {
			checkFunc: func() bool { return runCmd.Short != "" },
			desc:      "command should have Short description",
		},
		"has long description": {
			checkFunc: func() bool { return runCmd.Long != "" },
			desc:      "command should have Long description",
		},
		"has example": {
			checkFunc: func() bool { return runCmd.Example != "" },
			desc:      "command should have Example",
		},
		"requires exactly one arg": {
			checkFunc: func() bool { return runCmd.Args != nil },
			desc:      "command should have Args validator",
		},
		"has RunE function": {
			checkFunc: func() bool { return runCmd.RunE != nil },
			desc:      "command should have RunE function",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if !tt.checkFunc() {
				t.Error(tt.desc)
			}
		})
	}
}

func TestRunCmd_Flags(t *testing.T) {
	tests := map[string]struct {
		flagName string
		flagType string
	}{
		"dry-run flag": {
			flagName: "dry-run",
			flagType: "bool",
		},
		"force flag": {
			flagName: "force",
			flagType: "bool",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			flag := runCmd.Flags().Lookup(tt.flagName)
			if flag == nil {
				t.Errorf("expected flag %q to exist", tt.flagName)
				return
			}
			if flag.Value.Type() != tt.flagType {
				t.Errorf("expected flag type %q, got %q", tt.flagType, flag.Value.Type())
			}
		})
	}
}

func TestRunCmd_FreshFlag(t *testing.T) {
	tests := map[string]struct {
		args        []string
		expectFresh bool
	}{
		"no fresh flag": {
			args:        []string{"file.yaml"},
			expectFresh: false,
		},
		"fresh enabled": {
			args:        []string{"file.yaml", "--fresh"},
			expectFresh: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			runCmd.Flags().Set("fresh", "false")
			if err := runCmd.ParseFlags(tt.args); err != nil {
				t.Fatalf("failed to parse flags: %v", err)
			}

			fresh, _ := runCmd.Flags().GetBool("fresh")
			if fresh != tt.expectFresh {
				t.Errorf("expected fresh=%v, got %v", tt.expectFresh, fresh)
			}
		})
	}
}

func TestIsAllSpecsCompleted(t *testing.T) {
	tests := map[string]struct {
		run      *dag.DAGRun
		expected bool
	}{
		"nil run": {
			run:      nil,
			expected: false,
		},
		"empty specs": {
			run:      &dag.DAGRun{Specs: map[string]*dag.SpecState{}},
			expected: false,
		},
		"all completed": {
			run: &dag.DAGRun{
				Specs: map[string]*dag.SpecState{
					"spec-a": {Status: dag.SpecStatusCompleted},
					"spec-b": {Status: dag.SpecStatusCompleted},
				},
			},
			expected: true,
		},
		"some pending": {
			run: &dag.DAGRun{
				Specs: map[string]*dag.SpecState{
					"spec-a": {Status: dag.SpecStatusCompleted},
					"spec-b": {Status: dag.SpecStatusPending},
				},
			},
			expected: false,
		},
		"some failed": {
			run: &dag.DAGRun{
				Specs: map[string]*dag.SpecState{
					"spec-a": {Status: dag.SpecStatusCompleted},
					"spec-b": {Status: dag.SpecStatusFailed},
				},
			},
			expected: false,
		},
		"some running": {
			run: &dag.DAGRun{
				Specs: map[string]*dag.SpecState{
					"spec-a": {Status: dag.SpecStatusCompleted},
					"spec-b": {Status: dag.SpecStatusRunning},
				},
			},
			expected: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := isAllSpecsCompleted(tt.run)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestHandleFreshStart(t *testing.T) {
	tests := map[string]struct {
		stateExists bool
		expectError bool
	}{
		"no existing state": {
			stateExists: false,
			expectError: false,
		},
		"existing state gets deleted": {
			stateExists: true,
			expectError: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			stateDir := t.TempDir()
			filePath := "test-workflow.yaml"

			if tt.stateExists {
				// Create a state file
				run := &dag.DAGRun{
					WorkflowPath: filePath,
					Status:       dag.RunStatusFailed,
					StartedAt:    time.Now(),
					Specs: map[string]*dag.SpecState{
						"spec-a": {Status: dag.SpecStatusCompleted},
					},
				}
				if err := dag.SaveStateByWorkflow(stateDir, run); err != nil {
					t.Fatalf("failed to create test state: %v", err)
				}
				// Verify it exists
				if !dag.StateExistsForWorkflow(stateDir, filePath) {
					t.Fatal("state should exist before fresh start")
				}
			}

			err := handleFreshStart(stateDir, filePath)

			if tt.expectError && err == nil {
				t.Error("expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			// After fresh start, state should not exist
			if tt.stateExists && dag.StateExistsForWorkflow(stateDir, filePath) {
				t.Error("state file should have been deleted by fresh start")
			}
		})
	}
}

func TestPrintRunStatus(t *testing.T) {
	tests := map[string]struct {
		isResume      bool
		existingState *dag.DAGRun
	}{
		"new run": {
			isResume:      false,
			existingState: nil,
		},
		"resume run": {
			isResume: true,
			existingState: &dag.DAGRun{
				Specs: map[string]*dag.SpecState{
					"spec-a": {Status: dag.SpecStatusCompleted},
					"spec-b": {Status: dag.SpecStatusPending},
					"spec-c": {Status: dag.SpecStatusFailed},
				},
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Note: This test just verifies the function doesn't panic.
			// Output checking is skipped because color.Print() bypasses os.Stdout capture.
			printRunStatus("test.yaml", tt.isResume, tt.existingState)
		})
	}
}

func TestPrintAllSpecsCompleted(t *testing.T) {
	tests := map[string]struct {
		run *dag.DAGRun
	}{
		"all specs completed": {
			run: &dag.DAGRun{
				Specs: map[string]*dag.SpecState{
					"spec-a": {Status: dag.SpecStatusCompleted},
					"spec-b": {Status: dag.SpecStatusCompleted},
					"spec-c": {Status: dag.SpecStatusCompleted},
				},
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			printAllSpecsCompleted("test.yaml", tt.run)

			w.Close()
			os.Stdout = oldStdout

			var buf bytes.Buffer
			buf.ReadFrom(r)
			output := buf.String()

			if !bytes.Contains([]byte(output), []byte("All specs already completed")) {
				t.Error("expected output to contain 'All specs already completed'")
			}
			if !bytes.Contains([]byte(output), []byte("3/3")) {
				t.Error("expected output to contain spec count '3/3'")
			}
			if !bytes.Contains([]byte(output), []byte("--fresh")) {
				t.Error("expected output to mention --fresh flag")
			}
		})
	}
}

func TestPrintResumeDetails(t *testing.T) {
	tests := map[string]struct {
		run             *dag.DAGRun
		expectCompleted int
		expectPending   int
		expectFailed    int
	}{
		"nil run": {
			run: nil,
		},
		"mixed status specs": {
			run: &dag.DAGRun{
				Specs: map[string]*dag.SpecState{
					"spec-a": {Status: dag.SpecStatusCompleted},
					"spec-b": {Status: dag.SpecStatusCompleted},
					"spec-c": {Status: dag.SpecStatusPending},
					"spec-d": {Status: dag.SpecStatusFailed},
				},
			},
			expectCompleted: 2,
			expectPending:   1,
			expectFailed:    1,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			printResumeDetails(tt.run)

			w.Close()
			os.Stdout = oldStdout

			var buf bytes.Buffer
			buf.ReadFrom(r)
			output := buf.String()

			if tt.run == nil {
				if output != "" {
					t.Error("expected no output for nil run")
				}
				return
			}

			// Check that counts appear in output
			if tt.expectCompleted > 0 {
				if !bytes.Contains([]byte(output), []byte("Completed:")) {
					t.Error("expected output to contain 'Completed:'")
				}
			}
			if tt.expectFailed > 0 {
				if !bytes.Contains([]byte(output), []byte("Failed:")) {
					t.Error("expected output to contain 'Failed:'")
				}
			}
		})
	}
}
