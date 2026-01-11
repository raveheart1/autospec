package dag

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ariel-frischer/autospec/internal/dag"
	"github.com/ariel-frischer/autospec/internal/worktree"
)

// mockWorktreeManager implements worktree.Manager for testing.
type mockWorktreeManager struct {
	removeFunc  func(name string, force bool) error
	removeCalls []removeCall
}

type removeCall struct {
	name  string
	force bool
}

func (m *mockWorktreeManager) Create(name, branch, customPath string) (*worktree.Worktree, error) {
	return nil, nil
}

func (m *mockWorktreeManager) CreateWithOptions(name, branch, customPath string, opts worktree.CreateOptions) (*worktree.Worktree, error) {
	return nil, nil
}

func (m *mockWorktreeManager) List() ([]worktree.Worktree, error) {
	return nil, nil
}

func (m *mockWorktreeManager) Get(name string) (*worktree.Worktree, error) {
	return nil, nil
}

func (m *mockWorktreeManager) Remove(name string, force bool) error {
	m.removeCalls = append(m.removeCalls, removeCall{name: name, force: force})
	if m.removeFunc != nil {
		return m.removeFunc(name, force)
	}
	return nil
}

func (m *mockWorktreeManager) Setup(path string, addToState bool) (*worktree.Worktree, error) {
	return nil, nil
}

func (m *mockWorktreeManager) Prune() (int, error) {
	return 0, nil
}

func (m *mockWorktreeManager) UpdateStatus(name string, status worktree.WorktreeStatus) error {
	return nil
}

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
		stateExists        bool
		specWorktreePaths  map[string]string
		expectError        bool
		expectRemoveCalls  int
		expectRemoveForce  bool
		expectStateDeleted bool
	}{
		"no existing state": {
			stateExists:        false,
			expectError:        false,
			expectRemoveCalls:  0,
			expectStateDeleted: false, // nothing to delete
		},
		"existing state gets deleted": {
			stateExists:        true,
			expectError:        false,
			expectRemoveCalls:  0,
			expectStateDeleted: true,
		},
		"worktrees get cleaned up": {
			stateExists: true,
			specWorktreePaths: map[string]string{
				"spec-a": "/worktrees/spec-a-worktree",
				"spec-b": "/worktrees/spec-b-worktree",
			},
			expectError:        false,
			expectRemoveCalls:  2,
			expectRemoveForce:  true,
			expectStateDeleted: true,
		},
		"specs without worktree paths are skipped": {
			stateExists: true,
			specWorktreePaths: map[string]string{
				"spec-a": "/worktrees/spec-a-worktree",
				"spec-b": "", // no worktree path
			},
			expectError:        false,
			expectRemoveCalls:  1,
			expectRemoveForce:  true,
			expectStateDeleted: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			stateDir := t.TempDir()
			filePath := "test-workflow.yaml"
			mockMgr := &mockWorktreeManager{}

			if tt.stateExists {
				// Create a state file with specs
				specs := make(map[string]*dag.SpecState)
				if tt.specWorktreePaths != nil {
					for specID, wtPath := range tt.specWorktreePaths {
						specs[specID] = &dag.SpecState{
							SpecID:       specID,
							Status:       dag.SpecStatusCompleted,
							WorktreePath: wtPath,
						}
					}
				} else {
					specs["spec-a"] = &dag.SpecState{Status: dag.SpecStatusCompleted}
				}

				run := &dag.DAGRun{
					WorkflowPath: filePath,
					Status:       dag.RunStatusFailed,
					StartedAt:    time.Now(),
					Specs:        specs,
				}
				if err := dag.SaveStateByWorkflow(stateDir, run); err != nil {
					t.Fatalf("failed to create test state: %v", err)
				}
				// Verify it exists
				if !dag.StateExistsForWorkflow(stateDir, filePath) {
					t.Fatal("state should exist before fresh start")
				}
			}

			err := handleFreshStart(stateDir, filePath, mockMgr)

			if tt.expectError && err == nil {
				t.Error("expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			// Check worktree removal calls
			if len(mockMgr.removeCalls) != tt.expectRemoveCalls {
				t.Errorf("expected %d remove calls, got %d", tt.expectRemoveCalls, len(mockMgr.removeCalls))
			}

			// Check that force=true was used for all removals
			if tt.expectRemoveForce {
				for _, call := range mockMgr.removeCalls {
					if !call.force {
						t.Errorf("expected force=true for remove call %s, got false", call.name)
					}
				}
			}

			// Check state deletion
			stateExists := dag.StateExistsForWorkflow(stateDir, filePath)
			if tt.expectStateDeleted && stateExists {
				t.Error("state file should have been deleted by fresh start")
			}
			if !tt.expectStateDeleted && !tt.stateExists && stateExists {
				t.Error("state file should not exist when there was no existing state")
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
			// Note: This test just verifies the function doesn't panic.
			// Output checking is skipped because color.Print() bypasses os.Stdout capture.
			printAllSpecsCompleted("test.yaml", tt.run)
		})
	}
}

func TestPrintResumeDetails(t *testing.T) {
	tests := map[string]struct {
		run *dag.DAGRun
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
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Note: This test just verifies the function doesn't panic.
			// Output is printed to stdout (plain fmt.Printf, not colored).
			printResumeDetails(tt.run)
		})
	}
}
