package dag

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

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
			tmpDir := t.TempDir()
			stateDir := t.TempDir()
			filePath := filepath.Join(tmpDir, "test-workflow.yaml")
			mockMgr := &mockWorktreeManager{}

			// Create a minimal dag.yaml file (required by loadExistingState)
			dagConfig := &dag.DAGConfig{
				SchemaVersion: "1.0",
				DAG:           dag.DAGMetadata{ID: "test-dag"},
				Layers: []dag.Layer{
					{
						ID:       "L0",
						Features: []dag.Feature{{ID: "spec-a"}},
					},
				},
			}

			if tt.stateExists {
				// Create inline state in dag.yaml
				specs := make(map[string]*dag.InlineSpecState)
				if tt.specWorktreePaths != nil {
					for specID, wtPath := range tt.specWorktreePaths {
						specs[specID] = &dag.InlineSpecState{
							Status:   dag.InlineSpecStatusCompleted,
							Worktree: wtPath,
						}
					}
				} else {
					specs["spec-a"] = &dag.InlineSpecState{Status: dag.InlineSpecStatusCompleted}
				}
				dagConfig.Run = &dag.InlineRunState{
					Status: dag.InlineRunStatusFailed,
				}
				dagConfig.Specs = specs
			}

			if err := dag.SaveDAGWithState(filePath, dagConfig); err != nil {
				t.Fatalf("failed to create dag file: %v", err)
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

			// Check inline state was cleared
			if tt.expectStateDeleted {
				reloaded, err := dag.LoadDAGConfigFull(filePath)
				if err != nil {
					t.Fatalf("failed to reload dag file: %v", err)
				}
				if dag.HasInlineState(reloaded) {
					t.Error("inline state should have been cleared by fresh start")
				}
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

func TestRunCmd_OnlyFlag(t *testing.T) {
	tests := map[string]struct {
		args       []string
		expectOnly string
	}{
		"no only flag": {
			args:       []string{"file.yaml"},
			expectOnly: "",
		},
		"single spec": {
			args:       []string{"file.yaml", "--only", "spec1"},
			expectOnly: "spec1",
		},
		"multiple specs": {
			args:       []string{"file.yaml", "--only", "spec1,spec2,spec3"},
			expectOnly: "spec1,spec2,spec3",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			runCmd.Flags().Set("only", "")
			if err := runCmd.ParseFlags(tt.args); err != nil {
				t.Fatalf("failed to parse flags: %v", err)
			}

			only, _ := runCmd.Flags().GetString("only")
			if only != tt.expectOnly {
				t.Errorf("expected only=%q, got %q", tt.expectOnly, only)
			}
		})
	}
}

func TestRunCmd_CleanFlag(t *testing.T) {
	tests := map[string]struct {
		args        []string
		expectClean bool
	}{
		"no clean flag": {
			args:        []string{"file.yaml"},
			expectClean: false,
		},
		"clean enabled": {
			args:        []string{"file.yaml", "--only", "spec1", "--clean"},
			expectClean: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			runCmd.Flags().Set("clean", "false")
			runCmd.Flags().Set("only", "")
			if err := runCmd.ParseFlags(tt.args); err != nil {
				t.Fatalf("failed to parse flags: %v", err)
			}

			clean, _ := runCmd.Flags().GetBool("clean")
			if clean != tt.expectClean {
				t.Errorf("expected clean=%v, got %v", tt.expectClean, clean)
			}
		})
	}
}

func TestParseOnlySpecs(t *testing.T) {
	tests := map[string]struct {
		input    string
		expected []string
	}{
		"single spec": {
			input:    "spec1",
			expected: []string{"spec1"},
		},
		"multiple specs": {
			input:    "spec1,spec2,spec3",
			expected: []string{"spec1", "spec2", "spec3"},
		},
		"with whitespace": {
			input:    "spec1, spec2 , spec3",
			expected: []string{"spec1", "spec2", "spec3"},
		},
		"empty string": {
			input:    "",
			expected: []string{},
		},
		"only whitespace parts": {
			input:    ",,,",
			expected: []string{},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := parseOnlySpecs(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("expected %d specs, got %d", len(tt.expected), len(result))
				return
			}
			for i, spec := range result {
				if spec != tt.expected[i] {
					t.Errorf("expected spec[%d]=%q, got %q", i, tt.expected[i], spec)
				}
			}
		})
	}
}

func TestValidateSpecIDs(t *testing.T) {
	dagCfg := &dag.DAGConfig{
		Layers: []dag.Layer{
			{
				ID: "L0",
				Features: []dag.Feature{
					{ID: "spec-a"},
					{ID: "spec-b"},
				},
			},
			{
				ID: "L1",
				Features: []dag.Feature{
					{ID: "spec-c"},
				},
			},
		},
	}

	tests := map[string]struct {
		specIDs     []string
		expectError bool
	}{
		"all valid": {
			specIDs:     []string{"spec-a", "spec-b"},
			expectError: false,
		},
		"single valid": {
			specIDs:     []string{"spec-c"},
			expectError: false,
		},
		"one invalid": {
			specIDs:     []string{"spec-a", "invalid-spec"},
			expectError: true,
		},
		"all invalid": {
			specIDs:     []string{"foo", "bar"},
			expectError: true,
		},
		"empty list": {
			specIDs:     []string{},
			expectError: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			err := validateSpecIDs(dagCfg, tt.specIDs)
			if tt.expectError && err == nil {
				t.Error("expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateOnlyDependencies(t *testing.T) {
	dagCfg := &dag.DAGConfig{
		Layers: []dag.Layer{
			{
				ID: "L0",
				Features: []dag.Feature{
					{ID: "spec-a"},
				},
			},
			{
				ID: "L1",
				Features: []dag.Feature{
					{ID: "spec-b", DependsOn: []string{"spec-a"}},
				},
			},
			{
				ID: "L2",
				Features: []dag.Feature{
					{ID: "spec-c", DependsOn: []string{"spec-b"}},
				},
			},
		},
	}

	tests := map[string]struct {
		onlySpecs     []string
		existingState *dag.DAGRun
		expectError   bool
	}{
		"no dependencies": {
			onlySpecs: []string{"spec-a"},
			existingState: &dag.DAGRun{
				Specs: map[string]*dag.SpecState{
					"spec-a": {Status: dag.SpecStatusPending},
				},
			},
			expectError: false,
		},
		"dependency completed": {
			onlySpecs: []string{"spec-b"},
			existingState: &dag.DAGRun{
				Specs: map[string]*dag.SpecState{
					"spec-a": {Status: dag.SpecStatusCompleted},
					"spec-b": {Status: dag.SpecStatusPending},
				},
			},
			expectError: false,
		},
		"dependency not completed": {
			onlySpecs: []string{"spec-b"},
			existingState: &dag.DAGRun{
				Specs: map[string]*dag.SpecState{
					"spec-a": {Status: dag.SpecStatusPending},
					"spec-b": {Status: dag.SpecStatusPending},
				},
			},
			expectError: true,
		},
		"dependency failed": {
			onlySpecs: []string{"spec-b"},
			existingState: &dag.DAGRun{
				Specs: map[string]*dag.SpecState{
					"spec-a": {Status: dag.SpecStatusFailed},
					"spec-b": {Status: dag.SpecStatusPending},
				},
			},
			expectError: true,
		},
		"dependency in only list": {
			onlySpecs: []string{"spec-a", "spec-b"},
			existingState: &dag.DAGRun{
				Specs: map[string]*dag.SpecState{
					"spec-a": {Status: dag.SpecStatusPending},
					"spec-b": {Status: dag.SpecStatusPending},
				},
			},
			expectError: false, // spec-a is in the --only list, so it's fine
		},
		"chain of dependencies with only including all": {
			onlySpecs: []string{"spec-a", "spec-b", "spec-c"},
			existingState: &dag.DAGRun{
				Specs: map[string]*dag.SpecState{
					"spec-a": {Status: dag.SpecStatusPending},
					"spec-b": {Status: dag.SpecStatusPending},
					"spec-c": {Status: dag.SpecStatusPending},
				},
			},
			expectError: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			err := validateOnlyDependencies(dagCfg, tt.existingState, tt.onlySpecs)
			if tt.expectError && err == nil {
				t.Error("expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestCleanSpecs(t *testing.T) {
	tests := map[string]struct {
		specIDs           []string
		specStates        map[string]*dag.SpecState
		expectRemoveCalls int
		expectResetSpecs  []string
	}{
		"single spec with worktree": {
			specIDs: []string{"spec-a"},
			specStates: map[string]*dag.SpecState{
				"spec-a": {
					SpecID:        "spec-a",
					Status:        dag.SpecStatusCompleted,
					WorktreePath:  "/worktrees/spec-a-wt",
					FailureReason: "some reason",
				},
			},
			expectRemoveCalls: 1,
			expectResetSpecs:  []string{"spec-a"},
		},
		"spec without worktree": {
			specIDs: []string{"spec-a"},
			specStates: map[string]*dag.SpecState{
				"spec-a": {
					SpecID:       "spec-a",
					Status:       dag.SpecStatusFailed,
					WorktreePath: "",
				},
			},
			expectRemoveCalls: 0,
			expectResetSpecs:  []string{"spec-a"},
		},
		"multiple specs": {
			specIDs: []string{"spec-a", "spec-b"},
			specStates: map[string]*dag.SpecState{
				"spec-a": {
					SpecID:       "spec-a",
					Status:       dag.SpecStatusCompleted,
					WorktreePath: "/worktrees/spec-a-wt",
				},
				"spec-b": {
					SpecID:       "spec-b",
					Status:       dag.SpecStatusFailed,
					WorktreePath: "/worktrees/spec-b-wt",
				},
			},
			expectRemoveCalls: 2,
			expectResetSpecs:  []string{"spec-a", "spec-b"},
		},
		"spec not in state": {
			specIDs:           []string{"nonexistent"},
			specStates:        map[string]*dag.SpecState{},
			expectRemoveCalls: 0,
			expectResetSpecs:  []string{},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			tmpDir := t.TempDir()
			filePath := filepath.Join(tmpDir, "test.yaml")
			mockMgr := &mockWorktreeManager{}

			// Create dag.yaml file (required by saveCleanedState)
			dagConfig := &dag.DAGConfig{
				SchemaVersion: "1.0",
				DAG:           dag.DAGMetadata{ID: "test-dag"},
				Layers: []dag.Layer{
					{
						ID:       "L0",
						Features: []dag.Feature{{ID: "spec-a"}, {ID: "spec-b"}},
					},
				},
			}
			if err := dag.SaveDAGWithState(filePath, dagConfig); err != nil {
				t.Fatalf("failed to create dag file: %v", err)
			}

			// Create existing state
			existingState := &dag.DAGRun{
				WorkflowPath: filePath,
				Specs:        tt.specStates,
			}

			err := cleanSpecs(existingState, tt.specIDs, mockMgr, filePath)

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			// Check worktree removal calls
			if len(mockMgr.removeCalls) != tt.expectRemoveCalls {
				t.Errorf("expected %d remove calls, got %d", tt.expectRemoveCalls, len(mockMgr.removeCalls))
			}

			// Check spec state was reset
			for _, specID := range tt.expectResetSpecs {
				specState := existingState.Specs[specID]
				if specState == nil {
					continue
				}
				if specState.Status != dag.SpecStatusPending {
					t.Errorf("expected spec %s status to be pending, got %s", specID, specState.Status)
				}
				if specState.WorktreePath != "" {
					t.Errorf("expected spec %s worktree path to be empty", specID)
				}
				if specState.FailureReason != "" {
					t.Errorf("expected spec %s failure reason to be empty", specID)
				}
			}

			// Check that state was saved to dag.yaml
			reloaded, err := dag.LoadDAGConfigFull(filePath)
			if err != nil {
				t.Fatalf("failed to reload dag file: %v", err)
			}
			for _, specID := range tt.expectResetSpecs {
				if reloaded.Specs[specID] == nil {
					continue
				}
				if reloaded.Specs[specID].Status != dag.InlineSpecStatusPending {
					t.Errorf("expected saved spec %s status to be pending, got %s",
						specID, reloaded.Specs[specID].Status)
				}
			}
		})
	}
}

func TestHandleOnlySpecs(t *testing.T) {
	dagCfg := &dag.DAGConfig{
		Layers: []dag.Layer{
			{
				ID: "L0",
				Features: []dag.Feature{
					{ID: "spec-a"},
				},
			},
			{
				ID: "L1",
				Features: []dag.Feature{
					{ID: "spec-b", DependsOn: []string{"spec-a"}},
				},
			},
		},
	}

	tests := map[string]struct {
		existingState *dag.DAGRun
		onlySpecs     []string
		clean         bool
		expectError   bool
	}{
		"no existing state": {
			existingState: nil,
			onlySpecs:     []string{"spec-a"},
			clean:         false,
			expectError:   true,
		},
		"valid spec with existing state": {
			existingState: &dag.DAGRun{
				WorkflowPath: "test.yaml",
				Specs: map[string]*dag.SpecState{
					"spec-a": {Status: dag.SpecStatusCompleted},
					"spec-b": {Status: dag.SpecStatusPending},
				},
			},
			onlySpecs:   []string{"spec-b"},
			clean:       false,
			expectError: false,
		},
		"invalid spec ID": {
			existingState: &dag.DAGRun{
				WorkflowPath: "test.yaml",
				Specs: map[string]*dag.SpecState{
					"spec-a": {Status: dag.SpecStatusCompleted},
				},
			},
			onlySpecs:   []string{"invalid-spec"},
			clean:       false,
			expectError: true,
		},
		"dependency not completed": {
			existingState: &dag.DAGRun{
				WorkflowPath: "test.yaml",
				Specs: map[string]*dag.SpecState{
					"spec-a": {Status: dag.SpecStatusPending},
					"spec-b": {Status: dag.SpecStatusPending},
				},
			},
			onlySpecs:   []string{"spec-b"},
			clean:       false,
			expectError: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			stateDir := t.TempDir()
			mockMgr := &mockWorktreeManager{}

			_ = stateDir // unused since inline state is now used
			err := handleOnlySpecs(dagCfg, tt.existingState, tt.onlySpecs, tt.clean, "test.yaml", mockMgr)

			if tt.expectError && err == nil {
				t.Error("expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestCollectValidSpecIDs(t *testing.T) {
	tests := map[string]struct {
		dagCfg   *dag.DAGConfig
		expected map[string]bool
	}{
		"single layer single feature": {
			dagCfg: &dag.DAGConfig{
				Layers: []dag.Layer{
					{
						ID:       "L0",
						Features: []dag.Feature{{ID: "spec-a"}},
					},
				},
			},
			expected: map[string]bool{"spec-a": true},
		},
		"multiple layers multiple features": {
			dagCfg: &dag.DAGConfig{
				Layers: []dag.Layer{
					{
						ID:       "L0",
						Features: []dag.Feature{{ID: "spec-a"}, {ID: "spec-b"}},
					},
					{
						ID:       "L1",
						Features: []dag.Feature{{ID: "spec-c"}},
					},
				},
			},
			expected: map[string]bool{"spec-a": true, "spec-b": true, "spec-c": true},
		},
		"empty dag": {
			dagCfg:   &dag.DAGConfig{Layers: []dag.Layer{}},
			expected: map[string]bool{},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := collectValidSpecIDs(tt.dagCfg)
			if len(result) != len(tt.expected) {
				t.Errorf("expected %d IDs, got %d", len(tt.expected), len(result))
				return
			}
			for id := range tt.expected {
				if !result[id] {
					t.Errorf("expected ID %q to be present", id)
				}
			}
		})
	}
}

func TestGetSpecDependencies(t *testing.T) {
	dagCfg := &dag.DAGConfig{
		Layers: []dag.Layer{
			{
				ID:       "L0",
				Features: []dag.Feature{{ID: "spec-a"}},
			},
			{
				ID: "L1",
				Features: []dag.Feature{
					{ID: "spec-b", DependsOn: []string{"spec-a"}},
				},
			},
			{
				ID: "L2",
				Features: []dag.Feature{
					{ID: "spec-c", DependsOn: []string{"spec-a", "spec-b"}},
				},
			},
		},
	}

	tests := map[string]struct {
		specID   string
		expected []string
	}{
		"no dependencies": {
			specID:   "spec-a",
			expected: nil,
		},
		"single dependency": {
			specID:   "spec-b",
			expected: []string{"spec-a"},
		},
		"multiple dependencies": {
			specID:   "spec-c",
			expected: []string{"spec-a", "spec-b"},
		},
		"nonexistent spec": {
			specID:   "nonexistent",
			expected: nil,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := getSpecDependencies(dagCfg, tt.specID)
			if len(result) != len(tt.expected) {
				t.Errorf("expected %d dependencies, got %d", len(tt.expected), len(result))
				return
			}
			for i, dep := range result {
				if dep != tt.expected[i] {
					t.Errorf("expected dep[%d]=%q, got %q", i, tt.expected[i], dep)
				}
			}
		})
	}
}

func TestValidateDAGIDMatch(t *testing.T) {
	tests := map[string]struct {
		dagCfg        *dag.DAGConfig
		existingState *dag.DAGRun
		filePath      string
		expectError   bool
		errContains   string
	}{
		"matching ID from name": {
			dagCfg: &dag.DAGConfig{
				DAG: dag.DAGMetadata{Name: "GitStats CLI v1"},
			},
			existingState: &dag.DAGRun{
				DAGId:   "gitstats-cli-v1",
				DAGName: "GitStats CLI v1",
			},
			filePath:    "workflow.yaml",
			expectError: false,
		},
		"matching ID from explicit ID": {
			dagCfg: &dag.DAGConfig{
				DAG: dag.DAGMetadata{Name: "Some Long Name", ID: "short"},
			},
			existingState: &dag.DAGRun{
				DAGId:   "short",
				DAGName: "Some Long Name",
			},
			filePath:    "workflow.yaml",
			expectError: false,
		},
		"mismatch when dag.name changes": {
			dagCfg: &dag.DAGConfig{
				DAG: dag.DAGMetadata{Name: "New Feature Name"},
			},
			existingState: &dag.DAGRun{
				DAGId:   "old-feature-name",
				DAGName: "Old Feature Name",
			},
			filePath:    "workflow.yaml",
			expectError: true,
			errContains: "DAG ID mismatch",
		},
		"mismatch when dag.id changes": {
			dagCfg: &dag.DAGConfig{
				DAG: dag.DAGMetadata{Name: "Feature", ID: "new-id"},
			},
			existingState: &dag.DAGRun{
				DAGId:   "old-id",
				DAGName: "Feature",
			},
			filePath:    "workflow.yaml",
			expectError: true,
			errContains: "DAG ID mismatch",
		},
		"no error when dag.name changes but dag.id matches": {
			dagCfg: &dag.DAGConfig{
				DAG: dag.DAGMetadata{Name: "Completely New Display Name", ID: "stable-id"},
			},
			existingState: &dag.DAGRun{
				DAGId:   "stable-id",
				DAGName: "Original Display Name",
			},
			filePath:    "workflow.yaml",
			expectError: false,
		},
		"legacy state without DAGId is exempt": {
			dagCfg: &dag.DAGConfig{
				DAG: dag.DAGMetadata{Name: "Any Name"},
			},
			existingState: &dag.DAGRun{
				DAGId:   "", // Legacy state without DAGId
				DAGName: "",
			},
			filePath:    "workflow.yaml",
			expectError: false, // Exempt from validation
		},
		"matching ID from workflow filename fallback": {
			dagCfg: &dag.DAGConfig{
				DAG: dag.DAGMetadata{Name: "", ID: ""}, // Empty name and ID
			},
			existingState: &dag.DAGRun{
				DAGId:   "my-workflow",
				DAGName: "",
			},
			filePath:    "my-workflow.yaml",
			expectError: false,
		},
		"mismatch with workflow filename fallback": {
			dagCfg: &dag.DAGConfig{
				DAG: dag.DAGMetadata{Name: "", ID: ""},
			},
			existingState: &dag.DAGRun{
				DAGId:   "old-workflow",
				DAGName: "",
			},
			filePath:    "new-workflow.yaml",
			expectError: true,
			errContains: "DAG ID mismatch",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			err := validateDAGIDMatch(tt.dagCfg, tt.existingState, tt.filePath)

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

func TestFormatDAGIDMismatchError(t *testing.T) {
	tests := map[string]struct {
		resolvedID    string
		existingState *dag.DAGRun
		filePath      string
	}{
		"basic mismatch with DAGName": {
			resolvedID: "new-id",
			existingState: &dag.DAGRun{
				DAGId:   "old-id",
				DAGName: "Original Name",
			},
			filePath: "workflow.yaml",
		},
		"mismatch without DAGName": {
			resolvedID: "new-id",
			existingState: &dag.DAGRun{
				DAGId:   "old-id",
				DAGName: "",
			},
			filePath: "workflow.yaml",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			err := formatDAGIDMismatchError(tt.resolvedID, tt.existingState, tt.filePath)

			if err == nil {
				t.Error("expected error but got nil")
			}

			// Verify error contains both IDs in the message
			errMsg := err.Error()
			if !bytes.Contains([]byte(errMsg), []byte(tt.resolvedID)) {
				t.Errorf("error should contain resolved ID %q", tt.resolvedID)
			}
			if !bytes.Contains([]byte(errMsg), []byte(tt.existingState.DAGId)) {
				t.Errorf("error should contain stored ID %q", tt.existingState.DAGId)
			}
		})
	}
}

func TestRunCmd_AutocommitFlags(t *testing.T) {
	tests := map[string]struct {
		args               []string
		expectAutocommit   bool
		expectNoAutocommit bool
	}{
		"no autocommit flags": {
			args:               []string{"file.yaml"},
			expectAutocommit:   false,
			expectNoAutocommit: false,
		},
		"autocommit enabled": {
			args:               []string{"file.yaml", "--autocommit"},
			expectAutocommit:   true,
			expectNoAutocommit: false,
		},
		"no-autocommit enabled": {
			args:               []string{"file.yaml", "--no-autocommit"},
			expectAutocommit:   false,
			expectNoAutocommit: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			runCmd.Flags().Set("autocommit", "false")
			runCmd.Flags().Set("no-autocommit", "false")
			if err := runCmd.ParseFlags(tt.args); err != nil {
				t.Fatalf("failed to parse flags: %v", err)
			}

			autocommit, _ := runCmd.Flags().GetBool("autocommit")
			noAutocommit, _ := runCmd.Flags().GetBool("no-autocommit")

			if autocommit != tt.expectAutocommit {
				t.Errorf("expected autocommit=%v, got %v", tt.expectAutocommit, autocommit)
			}
			if noAutocommit != tt.expectNoAutocommit {
				t.Errorf("expected no-autocommit=%v, got %v", tt.expectNoAutocommit, noAutocommit)
			}
		})
	}
}

func TestBuildAutocommitOverride(t *testing.T) {
	tests := map[string]struct {
		autocommit   bool
		noAutocommit bool
		expected     *bool
	}{
		"neither flag set": {
			autocommit:   false,
			noAutocommit: false,
			expected:     nil,
		},
		"autocommit set": {
			autocommit:   true,
			noAutocommit: false,
			expected:     boolPtr(true),
		},
		"no-autocommit set": {
			autocommit:   false,
			noAutocommit: true,
			expected:     boolPtr(false),
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := buildAutocommitOverride(tt.autocommit, tt.noAutocommit)

			if tt.expected == nil {
				if result != nil {
					t.Errorf("expected nil, got %v", *result)
				}
				return
			}

			if result == nil {
				t.Errorf("expected %v, got nil", *tt.expected)
				return
			}

			if *result != *tt.expected {
				t.Errorf("expected %v, got %v", *tt.expected, *result)
			}
		})
	}
}

// boolPtr returns a pointer to the given bool.
func boolPtr(b bool) *bool {
	return &b
}

func TestRunCmd_MergeFlags(t *testing.T) {
	tests := map[string]struct {
		args                []string
		expectMerge         bool
		expectNoMergePrompt bool
	}{
		"no merge flags": {
			args:                []string{"file.yaml"},
			expectMerge:         false,
			expectNoMergePrompt: false,
		},
		"merge enabled": {
			args:                []string{"file.yaml", "--merge"},
			expectMerge:         true,
			expectNoMergePrompt: false,
		},
		"no-merge-prompt enabled": {
			args:                []string{"file.yaml", "--no-merge-prompt"},
			expectMerge:         false,
			expectNoMergePrompt: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			runCmd.Flags().Set("merge", "false")
			runCmd.Flags().Set("no-merge-prompt", "false")
			if err := runCmd.ParseFlags(tt.args); err != nil {
				t.Fatalf("failed to parse flags: %v", err)
			}

			merge, _ := runCmd.Flags().GetBool("merge")
			noMergePrompt, _ := runCmd.Flags().GetBool("no-merge-prompt")

			if merge != tt.expectMerge {
				t.Errorf("expected merge=%v, got %v", tt.expectMerge, merge)
			}
			if noMergePrompt != tt.expectNoMergePrompt {
				t.Errorf("expected no-merge-prompt=%v, got %v", tt.expectNoMergePrompt, noMergePrompt)
			}
		})
	}
}

func TestDecideMerge(t *testing.T) {
	tests := map[string]struct {
		autoMerge   bool
		hasFailures bool
		expected    bool
	}{
		"auto-merge enabled always returns true": {
			autoMerge:   true,
			hasFailures: false,
			expected:    true,
		},
		"auto-merge with failures still returns true": {
			autoMerge:   true,
			hasFailures: true,
			expected:    true,
		},
		// Note: non-interactive and interactive cases would require mocking
		// isInteractiveTerminal, which is more complex. These tests cover
		// the auto-merge path which is the primary CI use case.
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := decideMerge(tt.autoMerge, tt.hasFailures)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestAnalyzeMergeState(t *testing.T) {
	tests := map[string]struct {
		run               *dag.DAGRun
		expectHasWork     bool
		expectHasFailures bool
	}{
		"no specs": {
			run: &dag.DAGRun{
				Specs: map[string]*dag.SpecState{},
			},
			expectHasWork:     false,
			expectHasFailures: false,
		},
		"all completed": {
			run: &dag.DAGRun{
				Specs: map[string]*dag.SpecState{
					"spec-a": {Status: dag.SpecStatusCompleted},
					"spec-b": {Status: dag.SpecStatusCompleted},
				},
			},
			expectHasWork:     true,
			expectHasFailures: false,
		},
		"some failed": {
			run: &dag.DAGRun{
				Specs: map[string]*dag.SpecState{
					"spec-a": {Status: dag.SpecStatusCompleted},
					"spec-b": {Status: dag.SpecStatusFailed},
				},
			},
			expectHasWork:     true,
			expectHasFailures: true,
		},
		"all failed": {
			run: &dag.DAGRun{
				Specs: map[string]*dag.SpecState{
					"spec-a": {Status: dag.SpecStatusFailed},
					"spec-b": {Status: dag.SpecStatusFailed},
				},
			},
			expectHasWork:     false,
			expectHasFailures: true,
		},
		"only pending": {
			run: &dag.DAGRun{
				Specs: map[string]*dag.SpecState{
					"spec-a": {Status: dag.SpecStatusPending},
				},
			},
			expectHasWork:     false,
			expectHasFailures: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			hasWork, hasFailures := analyzeMergeState(tt.run)
			if hasWork != tt.expectHasWork {
				t.Errorf("hasWork: expected %v, got %v", tt.expectHasWork, hasWork)
			}
			if hasFailures != tt.expectHasFailures {
				t.Errorf("hasFailures: expected %v, got %v", tt.expectHasFailures, hasFailures)
			}
		})
	}
}
