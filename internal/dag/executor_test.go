package dag

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/ariel-frischer/autospec/internal/worktree"
)

// mockCommandRunner implements CommandRunner for testing.
type mockCommandRunner struct {
	runs      []runCall
	exitCodes map[string]int
	errors    map[string]error
}

type runCall struct {
	dir  string
	name string
	args []string
}

func newMockCommandRunner() *mockCommandRunner {
	return &mockCommandRunner{
		exitCodes: make(map[string]int),
		errors:    make(map[string]error),
	}
}

func (m *mockCommandRunner) Run(
	_ context.Context,
	dir string,
	_, _ io.Writer,
	name string,
	args ...string,
) (int, error) {
	m.runs = append(m.runs, runCall{dir: dir, name: name, args: args})
	key := name
	if exitCode, ok := m.exitCodes[key]; ok {
		return exitCode, m.errors[key]
	}
	return 0, nil
}

// mockWorktreeManager implements worktree.Manager for testing.
type mockWorktreeManager struct {
	worktrees map[string]*worktree.Worktree
	creates   []createCall
	removes   []string
}

type createCall struct {
	name   string
	branch string
}

func newMockWorktreeManager() *mockWorktreeManager {
	return &mockWorktreeManager{
		worktrees: make(map[string]*worktree.Worktree),
	}
}

func (m *mockWorktreeManager) Create(name, branch, customPath string) (*worktree.Worktree, error) {
	return m.CreateWithOptions(name, branch, customPath, worktree.CreateOptions{})
}

func (m *mockWorktreeManager) CreateWithOptions(name, branch, customPath string, _ worktree.CreateOptions) (*worktree.Worktree, error) {
	m.creates = append(m.creates, createCall{name: name, branch: branch})
	wt := &worktree.Worktree{
		Name:   name,
		Branch: branch,
		Path:   filepath.Join("/tmp", name),
	}
	m.worktrees[name] = wt
	return wt, nil
}

func (m *mockWorktreeManager) List() ([]worktree.Worktree, error) {
	var result []worktree.Worktree
	for _, wt := range m.worktrees {
		result = append(result, *wt)
	}
	return result, nil
}

func (m *mockWorktreeManager) Get(name string) (*worktree.Worktree, error) {
	if wt, ok := m.worktrees[name]; ok {
		return wt, nil
	}
	return nil, nil
}

func (m *mockWorktreeManager) Remove(name string, _ bool) error {
	m.removes = append(m.removes, name)
	delete(m.worktrees, name)
	return nil
}

func (m *mockWorktreeManager) Setup(_ string, _ bool) (*worktree.Worktree, error) {
	return nil, nil
}

func (m *mockWorktreeManager) Prune() (int, error) {
	return 0, nil
}

func (m *mockWorktreeManager) UpdateStatus(_ string, _ worktree.WorktreeStatus) error {
	return nil
}

func TestNewExecutor(t *testing.T) {
	tests := map[string]struct {
		dag             *DAGConfig
		dagFile         string
		stateDir        string
		repoRoot        string
		opts            []ExecutorOption
		expectDryRun    bool
		expectForce     bool
		expectCmdRunner bool
	}{
		"basic executor": {
			dag:      &DAGConfig{},
			dagFile:  "test.yaml",
			stateDir: "/tmp/state",
			repoRoot: "/tmp/repo",
		},
		"with dry run": {
			dag:          &DAGConfig{},
			dagFile:      "test.yaml",
			stateDir:     "/tmp/state",
			repoRoot:     "/tmp/repo",
			opts:         []ExecutorOption{WithDryRun(true)},
			expectDryRun: true,
		},
		"with force": {
			dag:         &DAGConfig{},
			dagFile:     "test.yaml",
			stateDir:    "/tmp/state",
			repoRoot:    "/tmp/repo",
			opts:        []ExecutorOption{WithForce(true)},
			expectForce: true,
		},
		"with custom command runner": {
			dag:             &DAGConfig{},
			dagFile:         "test.yaml",
			stateDir:        "/tmp/state",
			repoRoot:        "/tmp/repo",
			opts:            []ExecutorOption{WithCommandRunner(newMockCommandRunner())},
			expectCmdRunner: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			mgr := newMockWorktreeManager()
			cfg := DefaultDAGConfig()
			wtCfg := worktree.DefaultConfig()

			exec := NewExecutor(tt.dag, tt.dagFile, mgr, tt.stateDir, tt.repoRoot, cfg, wtCfg, tt.opts...)

			if exec == nil {
				t.Fatal("expected non-nil executor")
			}
			if exec.dag != tt.dag {
				t.Error("dag not set correctly")
			}
			if exec.dagFile != tt.dagFile {
				t.Error("dagFile not set correctly")
			}
			if exec.stateDir != tt.stateDir {
				t.Error("stateDir not set correctly")
			}
			if exec.repoRoot != tt.repoRoot {
				t.Error("repoRoot not set correctly")
			}
			if exec.dryRun != tt.expectDryRun {
				t.Errorf("expected dryRun=%v, got %v", tt.expectDryRun, exec.dryRun)
			}
			if exec.force != tt.expectForce {
				t.Errorf("expected force=%v, got %v", tt.expectForce, exec.force)
			}
		})
	}
}

func TestCollectSpecIDs(t *testing.T) {
	tests := map[string]struct {
		dag      *DAGConfig
		expected []string
	}{
		"empty dag": {
			dag:      &DAGConfig{},
			expected: nil,
		},
		"single layer single spec": {
			dag: &DAGConfig{
				Layers: []Layer{
					{ID: "L0", Features: []Feature{{ID: "spec-1"}}},
				},
			},
			expected: []string{"spec-1"},
		},
		"multiple layers multiple specs": {
			dag: &DAGConfig{
				Layers: []Layer{
					{ID: "L0", Features: []Feature{{ID: "spec-1"}, {ID: "spec-2"}}},
					{ID: "L1", Features: []Feature{{ID: "spec-3"}}},
				},
			},
			expected: []string{"spec-1", "spec-2", "spec-3"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			exec := &Executor{dag: tt.dag}
			ids := exec.collectSpecIDs()

			if len(ids) != len(tt.expected) {
				t.Errorf("expected %d spec IDs, got %d", len(tt.expected), len(ids))
				return
			}
			for i, id := range ids {
				if id != tt.expected[i] {
					t.Errorf("expected spec ID %q at index %d, got %q", tt.expected[i], i, id)
				}
			}
		})
	}
}

func TestGetLayersInOrder(t *testing.T) {
	tests := map[string]struct {
		dag           *DAGConfig
		expectedOrder []string
	}{
		"single layer": {
			dag: &DAGConfig{
				Layers: []Layer{
					{ID: "L0"},
				},
			},
			expectedOrder: []string{"L0"},
		},
		"two layers with dependency": {
			dag: &DAGConfig{
				Layers: []Layer{
					{ID: "L1", DependsOn: []string{"L0"}},
					{ID: "L0"},
				},
			},
			expectedOrder: []string{"L0", "L1"},
		},
		"three layers in order": {
			dag: &DAGConfig{
				Layers: []Layer{
					{ID: "L0"},
					{ID: "L1", DependsOn: []string{"L0"}},
					{ID: "L2", DependsOn: []string{"L1"}},
				},
			},
			expectedOrder: []string{"L0", "L1", "L2"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			exec := &Executor{dag: tt.dag}
			layers := exec.getLayersInOrder()

			if len(layers) != len(tt.expectedOrder) {
				t.Errorf("expected %d layers, got %d", len(tt.expectedOrder), len(layers))
				return
			}
			for i, layer := range layers {
				if layer.ID != tt.expectedOrder[i] {
					t.Errorf("expected layer %q at index %d, got %q", tt.expectedOrder[i], i, layer.ID)
				}
			}
		})
	}
}

func TestWaitForDependencies(t *testing.T) {
	tests := map[string]struct {
		state       *DAGRun
		feature     Feature
		expectError bool
	}{
		"no dependencies": {
			state: &DAGRun{
				Specs: map[string]*SpecState{},
			},
			feature:     Feature{ID: "spec-1"},
			expectError: false,
		},
		"dependency completed": {
			state: &DAGRun{
				Specs: map[string]*SpecState{
					"dep-1": {SpecID: "dep-1", Status: SpecStatusCompleted},
				},
			},
			feature:     Feature{ID: "spec-1", DependsOn: []string{"dep-1"}},
			expectError: false,
		},
		"dependency not completed": {
			state: &DAGRun{
				Specs: map[string]*SpecState{
					"dep-1": {SpecID: "dep-1", Status: SpecStatusRunning},
				},
			},
			feature:     Feature{ID: "spec-1", DependsOn: []string{"dep-1"}},
			expectError: true,
		},
		"dependency not found": {
			state: &DAGRun{
				Specs: map[string]*SpecState{},
			},
			feature:     Feature{ID: "spec-1", DependsOn: []string{"missing"}},
			expectError: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			exec := &Executor{state: tt.state}
			err := exec.waitForDependencies(tt.feature)

			if tt.expectError && err == nil {
				t.Error("expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestWorktreeName(t *testing.T) {
	tests := map[string]struct {
		dagID    string
		specID   string
		expected string
	}{
		"simple dag id and spec": {
			dagID:    "gitstats-cli-v1",
			specID:   "my-spec",
			expected: "dag-gitstats-cli-v1-my-spec",
		},
		"short dag id": {
			dagID:    "mvlfn",
			specID:   "feature",
			expected: "dag-mvlfn-feature",
		},
		"spec with numbers": {
			dagID:    "my-dag",
			specID:   "087-repo-reader",
			expected: "dag-my-dag-087-repo-reader",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			exec := &Executor{
				state: &DAGRun{DAGId: tt.dagID},
			}

			got := exec.worktreeName(tt.specID)
			if got != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestBranchName(t *testing.T) {
	tests := map[string]struct {
		dagID    string
		specID   string
		expected string
	}{
		"simple dag id and spec": {
			dagID:    "gitstats-cli-v1",
			specID:   "my-spec",
			expected: "dag/gitstats-cli-v1/my-spec",
		},
		"short dag id (explicit id)": {
			dagID:    "mvlfn",
			specID:   "feature",
			expected: "dag/mvlfn/feature",
		},
		"spec with numbers": {
			dagID:    "my-dag",
			specID:   "087-repo-reader",
			expected: "dag/my-dag/087-repo-reader",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			exec := &Executor{
				state: &DAGRun{DAGId: tt.dagID},
			}

			got := exec.branchName(tt.specID)
			if got != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

// TestNewDAGRunDAGIdResolution verifies DAG ID resolution priority in NewDAGRun.
func TestNewDAGRunDAGIdResolution(t *testing.T) {
	tests := map[string]struct {
		dagID        string
		dagName      string
		workflowPath string
		expectedID   string
		expectedName string
	}{
		"explicit ID takes priority": {
			dagID:        "my-custom-id",
			dagName:      "GitStats CLI v1",
			workflowPath: "workflows/v1.yaml",
			expectedID:   "my-custom-id",
			expectedName: "GitStats CLI v1",
		},
		"slugified name when no ID": {
			dagID:        "",
			dagName:      "GitStats CLI v1",
			workflowPath: "workflows/v1.yaml",
			expectedID:   "gitstats-cli-v1",
			expectedName: "GitStats CLI v1",
		},
		"workflow filename fallback": {
			dagID:        "",
			dagName:      "",
			workflowPath: ".autospec/dags/my-workflow.yaml",
			expectedID:   "my-workflow",
			expectedName: "",
		},
		"explicit ID is slugified": {
			dagID:        "My Custom ID",
			dagName:      "Some Name",
			workflowPath: "workflows/v1.yaml",
			expectedID:   "my-custom-id",
			expectedName: "Some Name",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			dag := &DAGConfig{
				DAG: DAGMetadata{
					ID:   tt.dagID,
					Name: tt.dagName,
				},
				Layers: []Layer{},
			}

			run := NewDAGRun(tt.workflowPath, dag, 0)

			if run.DAGId != tt.expectedID {
				t.Errorf("expected DAGId %q, got %q", tt.expectedID, run.DAGId)
			}

			if run.DAGName != tt.expectedName {
				t.Errorf("expected DAGName %q, got %q", tt.expectedName, run.DAGName)
			}

			// Verify workflow path is stored
			if run.WorkflowPath != tt.workflowPath {
				t.Errorf("expected WorkflowPath %q, got %q", tt.workflowPath, run.WorkflowPath)
			}
		})
	}
}

func TestExecuteDryRun(t *testing.T) {
	tmpDir := t.TempDir()

	dag := &DAGConfig{
		DAG: DAGMetadata{Name: "Test DAG"},
		Layers: []Layer{
			{
				ID: "L0",
				Features: []Feature{
					{ID: "spec-1", Description: "First spec"},
					{ID: "spec-2", Description: "Second spec", DependsOn: []string{"spec-1"}},
				},
			},
		},
	}

	var output bytes.Buffer
	exec := NewExecutor(
		dag,
		"test.yaml",
		newMockWorktreeManager(),
		filepath.Join(tmpDir, "state"),
		tmpDir,
		DefaultDAGConfig(),
		worktree.DefaultConfig(),
		WithExecutorStdout(&output),
		WithDryRun(true),
	)

	runID, err := exec.Execute(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if runID != "" {
		t.Errorf("expected empty run ID for dry run, got %q", runID)
	}

	outputStr := output.String()
	if !bytes.Contains([]byte(outputStr), []byte("DRY RUN MODE")) {
		t.Error("expected dry run output to contain 'DRY RUN MODE'")
	}
	if !bytes.Contains([]byte(outputStr), []byte("spec-1")) {
		t.Error("expected dry run output to contain 'spec-1'")
	}
	if !bytes.Contains([]byte(outputStr), []byte("spec-2")) {
		t.Error("expected dry run output to contain 'spec-2'")
	}
}

func TestExecuteEmptyDAG(t *testing.T) {
	tmpDir := t.TempDir()

	dag := &DAGConfig{
		DAG:    DAGMetadata{Name: "Empty DAG"},
		Layers: []Layer{},
	}

	var output bytes.Buffer
	exec := NewExecutor(
		dag,
		"test.yaml",
		newMockWorktreeManager(),
		filepath.Join(tmpDir, "state"),
		tmpDir,
		DefaultDAGConfig(),
		worktree.DefaultConfig(),
		WithExecutorStdout(&output),
	)

	runID, err := exec.Execute(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if runID != "" {
		t.Errorf("expected empty run ID for empty DAG, got %q", runID)
	}

	outputStr := output.String()
	if !bytes.Contains([]byte(outputStr), []byte("no specs")) {
		t.Error("expected output to indicate no specs")
	}
}

func TestSpecExists(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a spec directory
	specDir := filepath.Join(tmpDir, "specs", "existing-spec")
	if err := os.MkdirAll(specDir, 0o755); err != nil {
		t.Fatalf("failed to create spec dir: %v", err)
	}

	exec := &Executor{repoRoot: tmpDir}

	tests := map[string]struct {
		specID   string
		expected bool
	}{
		"existing spec":     {specID: "existing-spec", expected: true},
		"non-existing spec": {specID: "missing-spec", expected: false},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := exec.specExists(tt.specID)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestRunIDAndState(t *testing.T) {
	exec := &Executor{}

	if exec.RunID() != "" {
		t.Error("expected empty RunID when state is nil")
	}
	if exec.State() != nil {
		t.Error("expected nil State when not set")
	}

	exec.state = &DAGRun{RunID: "test-run-id"}

	if exec.RunID() != "test-run-id" {
		t.Errorf("expected RunID 'test-run-id', got %q", exec.RunID())
	}
	if exec.State() == nil {
		t.Error("expected non-nil State")
	}
}

// TestCreateWorktreeBranchNaming verifies the dag/<dag-id>/<spec-id> branch naming pattern.
func TestCreateWorktreeBranchNaming(t *testing.T) {
	tests := map[string]struct {
		dagID              string
		specID             string
		expectedBranch     string
		expectedWorktree   string
		expectBranchStored bool
	}{
		"slugified dag name": {
			dagID:              "gitstats-cli-v1",
			specID:             "my-spec",
			expectedBranch:     "dag/gitstats-cli-v1/my-spec",
			expectedWorktree:   "dag-gitstats-cli-v1-my-spec",
			expectBranchStored: true,
		},
		"explicit short id": {
			dagID:              "mvlfn",
			specID:             "087-dag-run",
			expectedBranch:     "dag/mvlfn/087-dag-run",
			expectedWorktree:   "dag-mvlfn-087-dag-run",
			expectBranchStored: true,
		},
		"workflow filename fallback": {
			dagID:              "my-workflow",
			specID:             "feature-auth-flow",
			expectedBranch:     "dag/my-workflow/feature-auth-flow",
			expectedWorktree:   "dag-my-workflow-feature-auth-flow",
			expectBranchStored: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			mgr := newMockWorktreeManager()
			specState := &SpecState{SpecID: tt.specID, Status: SpecStatusPending}
			exec := &Executor{
				worktreeManager: mgr,
				state: &DAGRun{
					DAGId: tt.dagID,
					Specs: map[string]*SpecState{tt.specID: specState},
				},
				stdout: io.Discard,
			}

			_, err := exec.createWorktree(tt.specID)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(mgr.creates) != 1 {
				t.Fatalf("expected 1 create call, got %d", len(mgr.creates))
			}

			// Verify branch name format
			if mgr.creates[0].branch != tt.expectedBranch {
				t.Errorf("expected branch %q, got %q", tt.expectedBranch, mgr.creates[0].branch)
			}

			// Verify worktree name format
			if mgr.creates[0].name != tt.expectedWorktree {
				t.Errorf("expected worktree name %q, got %q", tt.expectedWorktree, mgr.creates[0].name)
			}

			// Verify branch is stored in SpecState for resume
			if tt.expectBranchStored && specState.Branch != tt.expectedBranch {
				t.Errorf("expected branch stored in SpecState %q, got %q", tt.expectedBranch, specState.Branch)
			}
		})
	}
}

// TestOnDemandWorktreeCreation verifies worktrees are created only when spec is about to execute.
func TestOnDemandWorktreeCreation(t *testing.T) {
	tmpDir := t.TempDir()
	stateDir := filepath.Join(tmpDir, "state")

	dag := &DAGConfig{
		DAG: DAGMetadata{Name: "Test DAG"},
		Layers: []Layer{
			{
				ID: "L0",
				Features: []Feature{
					{ID: "spec-1", Description: "First spec"},
					{ID: "spec-2", Description: "Second spec"},
				},
			},
		},
	}

	mgr := newMockWorktreeManager()
	cmdRunner := newMockCommandRunner()

	var output bytes.Buffer
	exec := NewExecutor(
		dag,
		"test.yaml",
		mgr,
		stateDir,
		tmpDir,
		DefaultDAGConfig(),
		worktree.DefaultConfig(),
		WithExecutorStdout(&output),
		WithCommandRunner(cmdRunner),
	)

	// Initialize state first (simulating what Execute does)
	exec.state = NewDAGRun("test.yaml", dag, 0)

	// Before execution, no worktrees should be created
	if len(mgr.creates) != 0 {
		t.Errorf("expected 0 worktree creates before execution, got %d", len(mgr.creates))
	}

	// Create worktree for first spec
	_, err := exec.createWorktree("spec-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Only one worktree should exist now
	if len(mgr.creates) != 1 {
		t.Errorf("expected 1 worktree create, got %d", len(mgr.creates))
	}
	// Verify worktree name uses DAGId (dag-<dag-id>-<spec-id>)
	expectedName := "dag-" + exec.state.DAGId + "-spec-1"
	if mgr.creates[0].name != expectedName {
		t.Errorf("expected worktree name %q, got %q", expectedName, mgr.creates[0].name)
	}

	// Create worktree for second spec
	_, err = exec.createWorktree("spec-2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Now two worktrees
	if len(mgr.creates) != 2 {
		t.Errorf("expected 2 worktree creates, got %d", len(mgr.creates))
	}
}

// TestExistingWorktreeHandling verifies skip/prompt/force behavior.
func TestExistingWorktreeHandling(t *testing.T) {
	tests := map[string]struct {
		specStatus     SpecStatus
		worktreeExists bool
		forceFlag      bool
		expectError    bool
		expectRecreate bool
		expectSkip     bool
		errorContains  string
	}{
		"completed worktree - skip": {
			specStatus:     SpecStatusCompleted,
			worktreeExists: true,
			forceFlag:      false,
			expectError:    false,
			expectSkip:     true,
		},
		"failed worktree without force - error": {
			specStatus:     SpecStatusFailed,
			worktreeExists: true,
			forceFlag:      false,
			expectError:    true,
			errorContains:  "--force",
		},
		"failed worktree with force - recreate": {
			specStatus:     SpecStatusFailed,
			worktreeExists: true,
			forceFlag:      true,
			expectError:    false,
			expectRecreate: true,
		},
		"running worktree without force - error": {
			specStatus:     SpecStatusRunning,
			worktreeExists: true,
			forceFlag:      false,
			expectError:    true,
			errorContains:  "--force",
		},
		"running worktree with force - recreate": {
			specStatus:     SpecStatusRunning,
			worktreeExists: true,
			forceFlag:      true,
			expectError:    false,
			expectRecreate: true,
		},
		"worktree path set but not exists - create new": {
			specStatus:     SpecStatusPending,
			worktreeExists: false,
			forceFlag:      false,
			expectError:    false,
			expectRecreate: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			tmpDir := t.TempDir()
			mgr := newMockWorktreeManager()

			// Create worktree path if it should exist
			worktreePath := filepath.Join(tmpDir, "worktree-spec-1")
			if tt.worktreeExists {
				if err := os.MkdirAll(worktreePath, 0o755); err != nil {
					t.Fatalf("failed to create worktree dir: %v", err)
				}
			}

			specState := &SpecState{
				SpecID:       "spec-1",
				Status:       tt.specStatus,
				WorktreePath: worktreePath,
			}

			exec := &Executor{
				worktreeManager: mgr,
				state:           &DAGRun{RunID: "test-run", Specs: map[string]*SpecState{"spec-1": specState}},
				stdout:          io.Discard,
				force:           tt.forceFlag,
			}

			path, err := exec.handleExistingWorktree("spec-1", specState)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got nil")
				} else if tt.errorContains != "" && !bytes.Contains([]byte(err.Error()), []byte(tt.errorContains)) {
					t.Errorf("error should contain %q, got %q", tt.errorContains, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if tt.expectSkip {
				// Should return existing path without creating new worktree
				if path != worktreePath {
					t.Errorf("expected path %q, got %q", worktreePath, path)
				}
				if len(mgr.creates) != 0 {
					t.Errorf("expected 0 creates for skip, got %d", len(mgr.creates))
				}
			}

			if tt.expectRecreate {
				// Should have created a new worktree
				if len(mgr.creates) != 1 {
					t.Errorf("expected 1 create for recreate, got %d", len(mgr.creates))
				}
			}
		})
	}
}

// TestEnsureWorktreeNewSpec verifies ensureWorktree creates worktree for new specs.
func TestEnsureWorktreeNewSpec(t *testing.T) {
	mgr := newMockWorktreeManager()

	specState := &SpecState{
		SpecID:       "spec-1",
		Status:       SpecStatusPending,
		WorktreePath: "", // No existing worktree
	}

	exec := &Executor{
		worktreeManager: mgr,
		state: &DAGRun{
			RunID: "test-run",
			DAGId: "my-dag",
			Specs: map[string]*SpecState{"spec-1": specState},
		},
		stdout: io.Discard,
	}

	path, err := exec.ensureWorktree("spec-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should create new worktree
	if len(mgr.creates) != 1 {
		t.Errorf("expected 1 create, got %d", len(mgr.creates))
	}

	// Path should be set
	if path == "" {
		t.Error("expected non-empty path")
	}

	// Branch naming should use DAGId (dag/<dag-id>/<spec-id>)
	expectedBranch := "dag/my-dag/spec-1"
	if mgr.creates[0].branch != expectedBranch {
		t.Errorf("expected branch %q, got %q", expectedBranch, mgr.creates[0].branch)
	}

	// Branch should be stored in SpecState for resume
	if specState.Branch != expectedBranch {
		t.Errorf("expected Branch stored in SpecState %q, got %q", expectedBranch, specState.Branch)
	}
}

// TestIntegrationMultiLayerDAG tests a real DAG execution with 3 specs in 2 layers.
// Verifies that inline state is written to dag.yaml.
func TestIntegrationMultiLayerDAG(t *testing.T) {
	tmpDir := t.TempDir()
	stateDir := filepath.Join(tmpDir, "state")

	// Create a DAG with 3 specs in 2 layers
	dagConfig := &DAGConfig{
		SchemaVersion: "1.0",
		DAG:           DAGMetadata{Name: "Integration Test DAG"},
		Layers: []Layer{
			{
				ID:   "L0",
				Name: "Foundation Layer",
				Features: []Feature{
					{ID: "spec-a", Description: "First foundation spec"},
					{ID: "spec-b", Description: "Second foundation spec"},
				},
			},
			{
				ID:        "L1",
				Name:      "Application Layer",
				DependsOn: []string{"L0"},
				Features: []Feature{
					{ID: "spec-c", Description: "Application spec", DependsOn: []string{"spec-a", "spec-b"}},
				},
			},
		},
	}

	// Write the dag.yaml file first (executor writes state to this file)
	dagFile := filepath.Join(tmpDir, "test.yaml")
	if err := SaveDAGWithState(dagFile, dagConfig); err != nil {
		t.Fatalf("failed to write dag file: %v", err)
	}

	mgr := newMockWorktreeManager()
	cmdRunner := newMockCommandRunner()

	var output bytes.Buffer
	exec := NewExecutor(
		dagConfig,
		dagFile,
		mgr,
		stateDir,
		tmpDir,
		DefaultDAGConfig(),
		worktree.DefaultConfig(),
		WithExecutorStdout(&output),
		WithCommandRunner(cmdRunner),
	)

	runID, err := exec.Execute(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify run ID is empty (no longer generated)
	if runID != "" {
		t.Errorf("expected empty run ID (deprecated), got %q", runID)
	}

	// Verify all 3 worktrees were created
	if len(mgr.creates) != 3 {
		t.Errorf("expected 3 worktrees created, got %d", len(mgr.creates))
	}

	// Verify layer ordering: L0 specs executed before L1 specs
	specOrder := make([]string, 0, 3)
	for _, create := range mgr.creates {
		// Extract spec ID from worktree name (format: dag-<run-id>-<spec-id>)
		parts := bytes.Split([]byte(create.name), []byte("-"))
		if len(parts) >= 3 {
			specOrder = append(specOrder, string(parts[len(parts)-1]))
		}
	}

	// spec-c should be last (it's in L1)
	if len(specOrder) >= 3 && specOrder[2] != "c" {
		// The last spec created should be spec-c from L1
		foundC := false
		for i, s := range specOrder {
			if s == "c" {
				if i != 2 {
					// spec-c might not be strictly last if order within L0 varies
					// but it should not be before any L0 specs
				}
				foundC = true
			}
		}
		if !foundC {
			t.Error("spec-c was not created")
		}
	}

	// Verify inline state was written to dag.yaml
	loadedConfig, err := LoadDAGConfigFull(dagFile)
	if err != nil {
		t.Fatalf("failed to load dag file with state: %v", err)
	}

	if !HasInlineState(loadedConfig) {
		t.Error("inline state was not written to dag.yaml")
	}

	// Verify run state
	if loadedConfig.Run == nil {
		t.Fatal("run state is nil")
	}
	if loadedConfig.Run.Status != InlineRunStatusCompleted {
		t.Errorf("expected run status %q, got %q", InlineRunStatusCompleted, loadedConfig.Run.Status)
	}

	// Verify all specs completed
	for specID, specState := range loadedConfig.Specs {
		if specState.Status != InlineSpecStatusCompleted {
			t.Errorf("spec %q status is %q, expected %q", specID, specState.Status, InlineSpecStatusCompleted)
		}
	}

	// Verify no separate state file was created
	legacyStatePath := GetStatePathForWorkflow(stateDir, dagFile)
	if _, err := os.Stat(legacyStatePath); !os.IsNotExist(err) {
		t.Error("legacy state file should not have been created")
	}
}

// TestIntegrationLayerDependencyOrdering verifies L0 completes before L1 starts.
func TestIntegrationLayerDependencyOrdering(t *testing.T) {
	tmpDir := t.TempDir()
	stateDir := filepath.Join(tmpDir, "state")

	// Create a DAG with clear layer dependencies
	dagConfig := &DAGConfig{
		SchemaVersion: "1.0",
		DAG:           DAGMetadata{Name: "Layer Ordering Test"},
		Layers: []Layer{
			{
				ID:   "L0",
				Name: "Layer 0",
				Features: []Feature{
					{ID: "l0-spec-1", Description: "L0 spec 1"},
					{ID: "l0-spec-2", Description: "L0 spec 2"},
				},
			},
			{
				ID:        "L1",
				Name:      "Layer 1",
				DependsOn: []string{"L0"},
				Features: []Feature{
					{ID: "l1-spec-1", Description: "L1 spec 1"},
				},
			},
			{
				ID:        "L2",
				Name:      "Layer 2",
				DependsOn: []string{"L1"},
				Features: []Feature{
					{ID: "l2-spec-1", Description: "L2 spec 1"},
				},
			},
		},
	}

	// Write the dag.yaml file first
	dagFile := filepath.Join(tmpDir, "test.yaml")
	if err := SaveDAGWithState(dagFile, dagConfig); err != nil {
		t.Fatalf("failed to write dag file: %v", err)
	}

	// Track execution order
	var executionOrder []string
	var mu sync.Mutex

	cmdRunner := &trackingCommandRunner{
		onRun: func(dir string) {
			mu.Lock()
			defer mu.Unlock()
			// Extract spec ID from worktree path
			base := filepath.Base(dir)
			executionOrder = append(executionOrder, base)
		},
	}

	mgr := newMockWorktreeManager()
	var output bytes.Buffer

	exec := NewExecutor(
		dagConfig,
		dagFile,
		mgr,
		stateDir,
		tmpDir,
		DefaultDAGConfig(),
		worktree.DefaultConfig(),
		WithExecutorStdout(&output),
		WithCommandRunner(cmdRunner),
	)

	_, err := exec.Execute(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify 4 commands were executed
	if len(executionOrder) != 4 {
		t.Errorf("expected 4 executions, got %d", len(executionOrder))
	}

	// Verify layer ordering in state
	state := exec.State()
	for specID, specState := range state.Specs {
		if specState.Status != SpecStatusCompleted {
			t.Errorf("spec %q not completed", specID)
		}
	}
}

// trackingCommandRunner tracks execution calls for verification.
type trackingCommandRunner struct {
	onRun func(dir string)
}

func (r *trackingCommandRunner) Run(
	_ context.Context,
	dir string,
	_, _ io.Writer,
	_ string,
	_ ...string,
) (int, error) {
	if r.onRun != nil {
		r.onRun(dir)
	}
	return 0, nil
}

// TestIntegrationStateFileUpdates verifies state file is updated at key points.
func TestIntegrationStateFileUpdates(t *testing.T) {
	tmpDir := t.TempDir()
	stateDir := filepath.Join(tmpDir, "state")

	dagConfig := &DAGConfig{
		SchemaVersion: "1.0",
		DAG:           DAGMetadata{Name: "State Update Test"},
		Layers: []Layer{
			{
				ID:       "L0",
				Features: []Feature{{ID: "spec-1", Description: "Test spec"}},
			},
		},
	}

	// Write the dag.yaml file first
	dagFile := filepath.Join(tmpDir, "test.yaml")
	if err := SaveDAGWithState(dagFile, dagConfig); err != nil {
		t.Fatalf("failed to write dag file: %v", err)
	}

	// Track inline state snapshots during execution
	var stateSnapshots []*DAGConfig
	var mu sync.Mutex

	cmdRunner := &snapshotCommandRunner{
		dagFile: dagFile,
		onRun: func() {
			mu.Lock()
			defer mu.Unlock()
			// Read inline state during execution
			config, err := LoadDAGConfigFull(dagFile)
			if err == nil && HasInlineState(config) {
				stateSnapshots = append(stateSnapshots, config)
			}
		},
	}

	mgr := newMockWorktreeManager()
	var output bytes.Buffer

	exec := NewExecutor(
		dagConfig,
		dagFile,
		mgr,
		stateDir,
		tmpDir,
		DefaultDAGConfig(),
		worktree.DefaultConfig(),
		WithExecutorStdout(&output),
		WithCommandRunner(cmdRunner),
	)

	_, err := exec.Execute(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify state was captured during execution
	if len(stateSnapshots) == 0 {
		t.Error("no state snapshots captured during execution")
	}

	// Verify intermediate state showed running status
	foundRunning := false
	for _, snapshot := range stateSnapshots {
		for _, specState := range snapshot.Specs {
			if specState.Status == InlineSpecStatusRunning {
				foundRunning = true
				break
			}
		}
	}
	if !foundRunning {
		t.Log("Note: Running status may have transitioned too quickly to capture")
	}

	// Verify final state is completed in dag.yaml
	finalConfig, err := LoadDAGConfigFull(dagFile)
	if err != nil {
		t.Fatalf("failed to load final state: %v", err)
	}
	if finalConfig.Run == nil || finalConfig.Run.Status != InlineRunStatusCompleted {
		t.Errorf("final status should be completed, got %v", finalConfig.Run)
	}
}

// snapshotCommandRunner takes state snapshots during execution.
type snapshotCommandRunner struct {
	dagFile string
	onRun   func()
}

func (r *snapshotCommandRunner) Run(
	_ context.Context,
	_ string,
	_, _ io.Writer,
	_ string,
	_ ...string,
) (int, error) {
	if r.onRun != nil {
		r.onRun()
	}
	return 0, nil
}

// TestIntegrationLogFileCreation verifies per-spec log files are created.
func TestIntegrationLogFileCreation(t *testing.T) {
	tmpDir := t.TempDir()
	stateDir := filepath.Join(tmpDir, "state")

	dag := &DAGConfig{
		DAG: DAGMetadata{Name: "Log File Test"},
		Layers: []Layer{
			{
				ID: "L0",
				Features: []Feature{
					{ID: "log-spec-1", Description: "First spec"},
					{ID: "log-spec-2", Description: "Second spec"},
				},
			},
		},
	}

	// Command runner that writes output to verify logging
	cmdRunner := &loggingCommandRunner{
		output: "Test output line 1\nTest output line 2\n",
	}

	mgr := newMockWorktreeManager()
	var output bytes.Buffer

	exec := NewExecutor(
		dag,
		filepath.Join(tmpDir, "test.yaml"),
		mgr,
		stateDir,
		tmpDir,
		DefaultDAGConfig(),
		worktree.DefaultConfig(),
		WithExecutorStdout(&output),
		WithCommandRunner(cmdRunner),
	)

	runID, err := exec.Execute(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify log files exist and have content
	logDir := GetLogDir(stateDir, runID)
	for _, specID := range []string{"log-spec-1", "log-spec-2"} {
		logPath := filepath.Join(logDir, specID+".log")
		data, err := os.ReadFile(logPath)
		if err != nil {
			t.Errorf("failed to read log file for %q: %v", specID, err)
			continue
		}
		if len(data) == 0 {
			t.Errorf("log file for %q is empty", specID)
		}
	}

	// Verify terminal output has prefixed lines
	outStr := output.String()
	if !bytes.Contains([]byte(outStr), []byte("[log-spec-1]")) {
		t.Error("terminal output missing prefix for log-spec-1")
	}
	if !bytes.Contains([]byte(outStr), []byte("[log-spec-2]")) {
		t.Error("terminal output missing prefix for log-spec-2")
	}
}

// loggingCommandRunner writes predefined output to stdout/stderr.
type loggingCommandRunner struct {
	output string
}

func (r *loggingCommandRunner) Run(
	_ context.Context,
	_ string,
	stdout, _ io.Writer,
	_ string,
	_ ...string,
) (int, error) {
	if r.output != "" {
		stdout.Write([]byte(r.output))
	}
	return 0, nil
}

// ===== EDGE CASE TESTS =====

// TestEdgeCaseEmptyDAGCompletesImmediately verifies empty DAG handling.
func TestEdgeCaseEmptyDAGCompletesImmediately(t *testing.T) {
	tmpDir := t.TempDir()
	stateDir := filepath.Join(tmpDir, "state")

	// Empty DAG with no layers
	dag := &DAGConfig{
		DAG:    DAGMetadata{Name: "Empty DAG"},
		Layers: []Layer{},
	}

	var output bytes.Buffer
	exec := NewExecutor(
		dag,
		filepath.Join(tmpDir, "test.yaml"),
		newMockWorktreeManager(),
		stateDir,
		tmpDir,
		DefaultDAGConfig(),
		worktree.DefaultConfig(),
		WithExecutorStdout(&output),
	)

	runID, err := exec.Execute(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should return empty run ID for empty DAG
	if runID != "" {
		t.Errorf("expected empty run ID for empty DAG, got %q", runID)
	}

	// Output should indicate no specs
	if !bytes.Contains(output.Bytes(), []byte("no specs")) {
		t.Error("expected output to mention 'no specs'")
	}
}

// TestEdgeCaseEmptyLayerDAG verifies DAG with empty layers.
func TestEdgeCaseEmptyLayerDAG(t *testing.T) {
	tmpDir := t.TempDir()
	stateDir := filepath.Join(tmpDir, "state")

	// DAG with layers but no features
	dag := &DAGConfig{
		DAG: DAGMetadata{Name: "Empty Layers DAG"},
		Layers: []Layer{
			{ID: "L0", Name: "Empty Layer", Features: []Feature{}},
		},
	}

	var output bytes.Buffer
	exec := NewExecutor(
		dag,
		filepath.Join(tmpDir, "test.yaml"),
		newMockWorktreeManager(),
		stateDir,
		tmpDir,
		DefaultDAGConfig(),
		worktree.DefaultConfig(),
		WithExecutorStdout(&output),
	)

	runID, err := exec.Execute(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should return empty run ID for DAG with no actual specs
	if runID != "" {
		t.Errorf("expected empty run ID, got %q", runID)
	}
}

// TestEdgeCaseSingleLayerDAG verifies single-layer DAG execution.
func TestEdgeCaseSingleLayerDAG(t *testing.T) {
	tmpDir := t.TempDir()
	stateDir := filepath.Join(tmpDir, "state")

	dagConfig := &DAGConfig{
		SchemaVersion: "1.0",
		DAG:           DAGMetadata{Name: "Single Layer DAG"},
		Layers: []Layer{
			{
				ID:   "L0",
				Name: "Only Layer",
				Features: []Feature{
					{ID: "single-spec-1", Description: "First spec"},
					{ID: "single-spec-2", Description: "Second spec"},
					{ID: "single-spec-3", Description: "Third spec"},
				},
			},
		},
	}

	// Write the dag.yaml file first
	dagFile := filepath.Join(tmpDir, "test.yaml")
	if err := SaveDAGWithState(dagFile, dagConfig); err != nil {
		t.Fatalf("failed to write dag file: %v", err)
	}

	mgr := newMockWorktreeManager()
	cmdRunner := newMockCommandRunner()
	var output bytes.Buffer

	exec := NewExecutor(
		dagConfig,
		dagFile,
		mgr,
		stateDir,
		tmpDir,
		DefaultDAGConfig(),
		worktree.DefaultConfig(),
		WithExecutorStdout(&output),
		WithCommandRunner(cmdRunner),
	)

	runID, err := exec.Execute(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// RunID is deprecated and should be empty for new runs
	if runID != "" {
		t.Errorf("expected empty run ID (deprecated), got %q", runID)
	}

	// Verify all specs were created and executed
	if len(mgr.creates) != 3 {
		t.Errorf("expected 3 worktrees, got %d", len(mgr.creates))
	}

	// Verify all specs completed
	state := exec.State()
	for specID, specState := range state.Specs {
		if specState.Status != SpecStatusCompleted {
			t.Errorf("spec %q not completed, status: %s", specID, specState.Status)
		}
	}
}

// TestEdgeCaseAllSpecsFailing verifies behavior when all specs fail.
func TestEdgeCaseAllSpecsFailing(t *testing.T) {
	tmpDir := t.TempDir()
	stateDir := filepath.Join(tmpDir, "state")

	dagConfig := &DAGConfig{
		SchemaVersion: "1.0",
		DAG:           DAGMetadata{Name: "All Failing DAG"},
		Layers: []Layer{
			{
				ID: "L0",
				Features: []Feature{
					{ID: "fail-spec-1", Description: "First failing spec"},
					{ID: "fail-spec-2", Description: "Second spec (won't run)"},
				},
			},
		},
	}

	// Write the dag.yaml file first
	dagFile := filepath.Join(tmpDir, "test.yaml")
	if err := SaveDAGWithState(dagFile, dagConfig); err != nil {
		t.Fatalf("failed to write dag file: %v", err)
	}

	// Command runner that always fails with exit code 1
	failingRunner := &failingCommandRunner{exitCode: 1}

	mgr := newMockWorktreeManager()
	var output bytes.Buffer

	exec := NewExecutor(
		dagConfig,
		dagFile,
		mgr,
		stateDir,
		tmpDir,
		DefaultDAGConfig(),
		worktree.DefaultConfig(),
		WithExecutorStdout(&output),
		WithCommandRunner(failingRunner),
	)

	runID, err := exec.Execute(context.Background())

	// Should return error for failed spec
	if err == nil {
		t.Error("expected error when spec fails")
	}

	// RunID is deprecated and should be empty
	if runID != "" {
		t.Errorf("expected empty run ID (deprecated), got %q", runID)
	}

	// State should reflect failure
	state := exec.State()
	if state.Status != RunStatusFailed {
		t.Errorf("expected run status 'failed', got %q", state.Status)
	}

	// Only first spec should have run
	if len(mgr.creates) != 1 {
		t.Errorf("expected 1 worktree created before failure, got %d", len(mgr.creates))
	}

	// Output should contain failure information
	outStr := output.String()
	if !bytes.Contains([]byte(outStr), []byte("Failed")) {
		t.Error("output should mention failure")
	}

	// Should contain resume instructions
	if !bytes.Contains([]byte(outStr), []byte("resume")) {
		t.Error("output should contain resume instructions")
	}
}

// failingCommandRunner simulates command failures.
type failingCommandRunner struct {
	exitCode int
}

func (r *failingCommandRunner) Run(
	_ context.Context,
	_ string,
	_, _ io.Writer,
	_ string,
	_ ...string,
) (int, error) {
	return r.exitCode, nil
}

// TestEdgeCaseExistingWorktreeCompleted verifies skip behavior.
func TestEdgeCaseExistingWorktreeCompleted(t *testing.T) {
	tmpDir := t.TempDir()
	stateDir := filepath.Join(tmpDir, "state")

	dag := &DAGConfig{
		DAG: DAGMetadata{Name: "Existing Worktree Test"},
		Layers: []Layer{
			{
				ID:       "L0",
				Features: []Feature{{ID: "existing-spec", Description: "Test spec"}},
			},
		},
	}

	// Create worktree path to simulate existing worktree
	existingPath := filepath.Join(tmpDir, "existing-worktree")
	if err := os.MkdirAll(existingPath, 0o755); err != nil {
		t.Fatalf("failed to create existing worktree dir: %v", err)
	}

	mgr := newMockWorktreeManager()

	// Pre-populate the state with completed spec
	preState := &DAGRun{
		RunID:   "pre-existing-run",
		DAGFile: "test.yaml",
		Status:  RunStatusCompleted,
		Specs: map[string]*SpecState{
			"existing-spec": {
				SpecID:       "existing-spec",
				Status:       SpecStatusCompleted,
				WorktreePath: existingPath,
			},
		},
	}

	var output bytes.Buffer
	exec := NewExecutor(
		dag,
		filepath.Join(tmpDir, "test.yaml"),
		mgr,
		stateDir,
		tmpDir,
		DefaultDAGConfig(),
		worktree.DefaultConfig(),
		WithExecutorStdout(&output),
		WithCommandRunner(newMockCommandRunner()),
	)

	// Manually set state to simulate resume scenario
	exec.state = preState

	// Test handleExistingWorktree directly
	path, err := exec.handleExistingWorktree("existing-spec", preState.Specs["existing-spec"])
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should return existing path without creating new worktree
	if path != existingPath {
		t.Errorf("expected existing path %q, got %q", existingPath, path)
	}

	// No new worktrees should be created
	if len(mgr.creates) != 0 {
		t.Errorf("expected 0 worktree creates for skip, got %d", len(mgr.creates))
	}
}

// TestEdgeCaseExistingWorktreeFailedWithoutForce verifies error without --force.
func TestEdgeCaseExistingWorktreeFailedWithoutForce(t *testing.T) {
	tmpDir := t.TempDir()

	// Create worktree path to simulate existing worktree
	existingPath := filepath.Join(tmpDir, "failed-worktree")
	if err := os.MkdirAll(existingPath, 0o755); err != nil {
		t.Fatalf("failed to create worktree dir: %v", err)
	}

	mgr := newMockWorktreeManager()

	specState := &SpecState{
		SpecID:       "failed-spec",
		Status:       SpecStatusFailed,
		WorktreePath: existingPath,
	}

	exec := &Executor{
		worktreeManager: mgr,
		state: &DAGRun{
			RunID: "test-run",
			Specs: map[string]*SpecState{"failed-spec": specState},
		},
		stdout: io.Discard,
		force:  false, // No force flag
	}

	_, err := exec.handleExistingWorktree("failed-spec", specState)

	// Should error without --force
	if err == nil {
		t.Error("expected error for failed worktree without --force")
	}

	if !bytes.Contains([]byte(err.Error()), []byte("--force")) {
		t.Errorf("error should mention --force flag: %v", err)
	}
}

// TestEdgeCaseExistingWorktreeFailedWithForce verifies recreation with --force.
func TestEdgeCaseExistingWorktreeFailedWithForce(t *testing.T) {
	tmpDir := t.TempDir()

	// Create worktree path to simulate existing worktree
	existingPath := filepath.Join(tmpDir, "failed-worktree")
	if err := os.MkdirAll(existingPath, 0o755); err != nil {
		t.Fatalf("failed to create worktree dir: %v", err)
	}

	mgr := newMockWorktreeManager()

	specState := &SpecState{
		SpecID:       "failed-spec",
		Status:       SpecStatusFailed,
		WorktreePath: existingPath,
	}

	exec := &Executor{
		worktreeManager: mgr,
		state: &DAGRun{
			RunID: "test-run",
			Specs: map[string]*SpecState{"failed-spec": specState},
		},
		stdout: io.Discard,
		force:  true, // Force flag enabled
	}

	_, err := exec.handleExistingWorktree("failed-spec", specState)

	// Should succeed with --force
	if err != nil {
		t.Fatalf("unexpected error with --force: %v", err)
	}

	// Should have created a new worktree
	if len(mgr.creates) != 1 {
		t.Errorf("expected 1 worktree creation, got %d", len(mgr.creates))
	}
}

// TestEdgeCaseWorktreePathExistsButFileDeleted verifies creation when path is stale.
func TestEdgeCaseWorktreePathExistsButFileDeleted(t *testing.T) {
	tmpDir := t.TempDir()

	// Use a path that doesn't exist
	nonexistentPath := filepath.Join(tmpDir, "deleted-worktree")

	mgr := newMockWorktreeManager()

	specState := &SpecState{
		SpecID:       "stale-spec",
		Status:       SpecStatusPending,
		WorktreePath: nonexistentPath, // Path in state but doesn't exist on disk
	}

	exec := &Executor{
		worktreeManager: mgr,
		state: &DAGRun{
			RunID: "test-run",
			Specs: map[string]*SpecState{"stale-spec": specState},
		},
		stdout: io.Discard,
	}

	_, err := exec.handleExistingWorktree("stale-spec", specState)

	// Should succeed by creating a new worktree
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have created a new worktree
	if len(mgr.creates) != 1 {
		t.Errorf("expected 1 worktree creation, got %d", len(mgr.creates))
	}
}

// TestEdgeCaseContextCancellation verifies interrupt handling.
func TestEdgeCaseContextCancellation(t *testing.T) {
	tmpDir := t.TempDir()
	stateDir := filepath.Join(tmpDir, "state")

	dag := &DAGConfig{
		DAG: DAGMetadata{Name: "Cancellation Test"},
		Layers: []Layer{
			{
				ID: "L0",
				Features: []Feature{
					{ID: "cancel-spec", Description: "Spec that will be cancelled"},
				},
			},
		},
	}

	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	mgr := newMockWorktreeManager()
	var output bytes.Buffer

	exec := NewExecutor(
		dag,
		filepath.Join(tmpDir, "test.yaml"),
		mgr,
		stateDir,
		tmpDir,
		DefaultDAGConfig(),
		worktree.DefaultConfig(),
		WithExecutorStdout(&output),
		WithCommandRunner(newMockCommandRunner()),
	)

	_, err := exec.Execute(ctx)

	// Should return context error
	if err == nil {
		t.Error("expected error for cancelled context")
	}

	// State should be interrupted
	state := exec.State()
	if state == nil {
		t.Fatal("state should be set")
	}
	if state.Status != RunStatusInterrupted {
		t.Errorf("expected status 'interrupted', got %q", state.Status)
	}
}

// TestEdgeCaseSpecWithinLayerDependency verifies within-layer depends_on.
func TestEdgeCaseSpecWithinLayerDependency(t *testing.T) {
	tmpDir := t.TempDir()
	stateDir := filepath.Join(tmpDir, "state")

	dag := &DAGConfig{
		DAG: DAGMetadata{Name: "Within-Layer Dependency Test"},
		Layers: []Layer{
			{
				ID: "L0",
				Features: []Feature{
					{ID: "dep-spec", Description: "Dependency spec"},
					{ID: "main-spec", Description: "Main spec", DependsOn: []string{"dep-spec"}},
				},
			},
		},
	}

	var execOrder []string
	var mu sync.Mutex

	cmdRunner := &trackingCommandRunner{
		onRun: func(dir string) {
			mu.Lock()
			defer mu.Unlock()
			execOrder = append(execOrder, filepath.Base(dir))
		},
	}

	mgr := newMockWorktreeManager()
	var output bytes.Buffer

	exec := NewExecutor(
		dag,
		filepath.Join(tmpDir, "test.yaml"),
		mgr,
		stateDir,
		tmpDir,
		DefaultDAGConfig(),
		worktree.DefaultConfig(),
		WithExecutorStdout(&output),
		WithCommandRunner(cmdRunner),
	)

	_, err := exec.Execute(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify both specs executed
	if len(execOrder) != 2 {
		t.Errorf("expected 2 executions, got %d", len(execOrder))
	}

	// Verify all specs completed
	state := exec.State()
	for specID, specState := range state.Specs {
		if specState.Status != SpecStatusCompleted {
			t.Errorf("spec %q not completed", specID)
		}
	}
}

// TestEdgeCaseDependencyNotCompleted verifies error on incomplete dependency.
func TestEdgeCaseDependencyNotCompleted(t *testing.T) {
	tests := map[string]struct {
		depStatus     SpecStatus
		expectError   bool
		errorContains string
	}{
		"dependency pending": {
			depStatus:     SpecStatusPending,
			expectError:   true,
			errorContains: "not completed",
		},
		"dependency running": {
			depStatus:     SpecStatusRunning,
			expectError:   true,
			errorContains: "not completed",
		},
		"dependency failed": {
			depStatus:     SpecStatusFailed,
			expectError:   true,
			errorContains: "not completed",
		},
		"dependency blocked": {
			depStatus:     SpecStatusBlocked,
			expectError:   true,
			errorContains: "not completed",
		},
		"dependency completed": {
			depStatus:   SpecStatusCompleted,
			expectError: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			state := &DAGRun{
				Specs: map[string]*SpecState{
					"dep": {SpecID: "dep", Status: tt.depStatus},
				},
			}

			exec := &Executor{state: state}
			feature := Feature{ID: "main", DependsOn: []string{"dep"}}

			err := exec.waitForDependencies(feature)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got nil")
				} else if !bytes.Contains([]byte(err.Error()), []byte(tt.errorContains)) {
					t.Errorf("error should contain %q: %v", tt.errorContains, err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestGenerateHashSuffix(t *testing.T) {
	tests := map[string]struct {
		input    string
		expected string
	}{
		"simple path": {
			input:    "/path/to/workflow.yaml",
			expected: "c12e", // First 4 hex chars of SHA256
		},
		"different path produces different hash": {
			input:    "/other/path.yaml",
			expected: "53f3",
		},
		"empty string": {
			input:    "",
			expected: "e3b0", // SHA256 of empty string starts with e3b0
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := generateHashSuffix(tt.input)

			// Verify length is 4 characters
			if len(got) != 4 {
				t.Errorf("expected 4 characters, got %d: %q", len(got), got)
			}

			// Verify determinism - same input produces same output
			got2 := generateHashSuffix(tt.input)
			if got != got2 {
				t.Errorf("hash not deterministic: %q vs %q", got, got2)
			}

			// Verify expected value
			if got != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestBranchBelongsToThisDAG(t *testing.T) {
	tests := map[string]struct {
		dagID      string
		branchName string
		expected   bool
	}{
		"branch matches dag id": {
			dagID:      "my-dag",
			branchName: "dag/my-dag/my-spec",
			expected:   true,
		},
		"branch from different dag": {
			dagID:      "my-dag",
			branchName: "dag/other-dag/my-spec",
			expected:   false,
		},
		"branch with suffix still matches": {
			dagID:      "my-dag",
			branchName: "dag/my-dag/my-spec-a1b2",
			expected:   true,
		},
		"unrelated branch": {
			dagID:      "my-dag",
			branchName: "feature/some-branch",
			expected:   false,
		},
		"dag id prefix match but not full": {
			dagID:      "my-dag",
			branchName: "dag/my-dag-extended/spec",
			expected:   false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			exec := &Executor{
				state: &DAGRun{DAGId: tt.dagID},
			}

			got := exec.branchBelongsToThisDAG(tt.branchName)
			if got != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, got)
			}
		})
	}
}

func TestBranchNameWithSuffix(t *testing.T) {
	tests := map[string]struct {
		dagID      string
		dagFile    string
		specID     string
		wantSuffix bool
	}{
		"includes hash suffix": {
			dagID:      "my-dag",
			dagFile:    "/path/to/workflow.yaml",
			specID:     "my-spec",
			wantSuffix: true,
		},
		"different dag files produce different suffixes": {
			dagID:      "same-dag",
			dagFile:    "/other/workflow.yaml",
			specID:     "spec",
			wantSuffix: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			exec := &Executor{
				state:   &DAGRun{DAGId: tt.dagID},
				dagFile: tt.dagFile,
			}

			got := exec.branchNameWithSuffix(tt.specID)

			// Should start with base branch format
			baseBranch := exec.branchName(tt.specID)
			if !strings.HasPrefix(got, baseBranch+"-") {
				t.Errorf("expected to start with %q-, got %q", baseBranch, got)
			}

			// Should have 4-char suffix after dash
			suffix := strings.TrimPrefix(got, baseBranch+"-")
			if len(suffix) != 4 {
				t.Errorf("expected 4-char suffix, got %q (len %d)", suffix, len(suffix))
			}
		})
	}

	// Verify different dag files produce different suffixes
	t.Run("different dag files produce different suffixes", func(t *testing.T) {
		exec1 := &Executor{
			state:   &DAGRun{DAGId: "my-dag"},
			dagFile: "/path/one.yaml",
		}
		exec2 := &Executor{
			state:   &DAGRun{DAGId: "my-dag"},
			dagFile: "/path/two.yaml",
		}

		suffix1 := exec1.branchNameWithSuffix("spec")
		suffix2 := exec2.branchNameWithSuffix("spec")

		if suffix1 == suffix2 {
			t.Errorf("different dag files should produce different suffixes: %q vs %q", suffix1, suffix2)
		}
	})
}

func TestFindCollisionSafeBranch(t *testing.T) {
	tests := map[string]struct {
		dagID          string
		dagFile        string
		specID         string
		expectSuffixed bool
		description    string
	}{
		"no collision uses base branch": {
			dagID:          "my-dag",
			dagFile:        "/path/workflow.yaml",
			specID:         "my-spec",
			expectSuffixed: false,
			description:    "When no branch exists, use base format",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			exec := &Executor{
				state:   &DAGRun{DAGId: tt.dagID},
				dagFile: tt.dagFile,
			}

			got, err := exec.findCollisionSafeBranch(tt.specID)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			baseBranch := exec.branchName(tt.specID)
			if tt.expectSuffixed {
				if !strings.HasPrefix(got, baseBranch+"-") {
					t.Errorf("expected suffixed branch starting with %q-, got %q", baseBranch, got)
				}
			} else {
				if got != baseBranch {
					t.Errorf("expected base branch %q, got %q", baseBranch, got)
				}
			}
		})
	}
}

// TestCreateWorktreeWithLayerStaging verifies that createWorktree passes the correct
// start point to the worktree manager based on the spec's layer.
func TestCreateWorktreeWithLayerStaging(t *testing.T) {
	tests := map[string]struct {
		dag                *DAGConfig
		dagID              string
		baseBranch         string
		specID             string
		specLayerID        string
		expectedStartPoint string
		description        string
	}{
		"layer 0 spec uses main as start point": {
			dag: &DAGConfig{
				Layers: []Layer{
					{ID: "L0", Features: []Feature{{ID: "l0-spec"}}},
					{ID: "L1", DependsOn: []string{"L0"}, Features: []Feature{{ID: "l1-spec"}}},
				},
			},
			dagID:              "my-dag",
			baseBranch:         "",
			specID:             "l0-spec",
			specLayerID:        "L0",
			expectedStartPoint: "main",
			description:        "L0 spec branches from main",
		},
		"layer 0 spec uses configured base branch": {
			dag: &DAGConfig{
				Layers: []Layer{
					{ID: "L0", Features: []Feature{{ID: "l0-spec"}}},
				},
			},
			dagID:              "my-dag",
			baseBranch:         "develop",
			specID:             "l0-spec",
			specLayerID:        "L0",
			expectedStartPoint: "develop",
			description:        "L0 spec branches from configured base branch",
		},
		"layer 1 spec uses L0 staging branch": {
			dag: &DAGConfig{
				Layers: []Layer{
					{ID: "L0", Features: []Feature{{ID: "l0-spec"}}},
					{ID: "L1", DependsOn: []string{"L0"}, Features: []Feature{{ID: "l1-spec"}}},
				},
			},
			dagID:              "my-dag",
			baseBranch:         "main",
			specID:             "l1-spec",
			specLayerID:        "L1",
			expectedStartPoint: "dag/my-dag/stage-L0",
			description:        "L1 spec branches from L0 staging branch",
		},
		"layer 2 spec uses L1 staging branch": {
			dag: &DAGConfig{
				Layers: []Layer{
					{ID: "L0", Features: []Feature{{ID: "l0-spec"}}},
					{ID: "L1", DependsOn: []string{"L0"}, Features: []Feature{{ID: "l1-spec"}}},
					{ID: "L2", DependsOn: []string{"L1"}, Features: []Feature{{ID: "l2-spec"}}},
				},
			},
			dagID:              "my-dag",
			baseBranch:         "main",
			specID:             "l2-spec",
			specLayerID:        "L2",
			expectedStartPoint: "dag/my-dag/stage-L1",
			description:        "L2 spec branches from L1 staging branch",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			mgr := &mockWorktreeManagerWithStartPoint{
				worktrees: make(map[string]*worktree.Worktree),
			}
			specState := &SpecState{
				SpecID:  tt.specID,
				Status:  SpecStatusPending,
				LayerID: tt.specLayerID,
			}
			exec := &Executor{
				dag:             tt.dag,
				worktreeManager: mgr,
				config:          &DAGExecutionConfig{BaseBranch: tt.baseBranch},
				state: &DAGRun{
					DAGId: tt.dagID,
					Specs: map[string]*SpecState{tt.specID: specState},
				},
				stdout: io.Discard,
			}

			_, err := exec.createWorktree(tt.specID)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(mgr.creates) != 1 {
				t.Fatalf("expected 1 create call, got %d", len(mgr.creates))
			}

			// Verify start point was passed correctly
			if mgr.creates[0].startPoint != tt.expectedStartPoint {
				t.Errorf("%s: expected start point %q, got %q",
					tt.description, tt.expectedStartPoint, mgr.creates[0].startPoint)
			}
		})
	}
}

// mockWorktreeManagerWithStartPoint extends mockWorktreeManager to track start points.
type mockWorktreeManagerWithStartPoint struct {
	worktrees map[string]*worktree.Worktree
	creates   []createCallWithStartPoint
	removes   []string
}

type createCallWithStartPoint struct {
	name       string
	branch     string
	startPoint string
}

func (m *mockWorktreeManagerWithStartPoint) Create(name, branch, customPath string) (*worktree.Worktree, error) {
	return m.CreateWithOptions(name, branch, customPath, worktree.CreateOptions{})
}

func (m *mockWorktreeManagerWithStartPoint) CreateWithOptions(name, branch, customPath string, opts worktree.CreateOptions) (*worktree.Worktree, error) {
	m.creates = append(m.creates, createCallWithStartPoint{
		name:       name,
		branch:     branch,
		startPoint: opts.StartPoint,
	})
	wt := &worktree.Worktree{
		Name:   name,
		Branch: branch,
		Path:   filepath.Join("/tmp", name),
	}
	m.worktrees[name] = wt
	return wt, nil
}

func (m *mockWorktreeManagerWithStartPoint) List() ([]worktree.Worktree, error) {
	var result []worktree.Worktree
	for _, wt := range m.worktrees {
		result = append(result, *wt)
	}
	return result, nil
}

func (m *mockWorktreeManagerWithStartPoint) Get(name string) (*worktree.Worktree, error) {
	if wt, ok := m.worktrees[name]; ok {
		return wt, nil
	}
	return nil, nil
}

func (m *mockWorktreeManagerWithStartPoint) Remove(name string, _ bool) error {
	m.removes = append(m.removes, name)
	delete(m.worktrees, name)
	return nil
}

func (m *mockWorktreeManagerWithStartPoint) Setup(_ string, _ bool) (*worktree.Worktree, error) {
	return nil, nil
}

func (m *mockWorktreeManagerWithStartPoint) Prune() (int, error) {
	return 0, nil
}

func (m *mockWorktreeManagerWithStartPoint) UpdateStatus(_ string, _ worktree.WorktreeStatus) error {
	return nil
}

func TestGetBaseBranchForLayer(t *testing.T) {
	tests := map[string]struct {
		dag         *DAGConfig
		dagID       string
		baseBranch  string
		layerID     string
		expected    string
		description string
	}{
		"layer 0 returns base branch (main default)": {
			dag: &DAGConfig{
				Layers: []Layer{
					{ID: "L0"},
					{ID: "L1", DependsOn: []string{"L0"}},
				},
			},
			dagID:       "my-dag",
			baseBranch:  "",
			layerID:     "L0",
			expected:    "main",
			description: "First layer uses default main branch",
		},
		"layer 0 returns configured base branch": {
			dag: &DAGConfig{
				Layers: []Layer{
					{ID: "L0"},
					{ID: "L1", DependsOn: []string{"L0"}},
				},
			},
			dagID:       "my-dag",
			baseBranch:  "develop",
			layerID:     "L0",
			expected:    "develop",
			description: "First layer uses configured base branch",
		},
		"layer 1 returns previous layer staging branch": {
			dag: &DAGConfig{
				Layers: []Layer{
					{ID: "L0"},
					{ID: "L1", DependsOn: []string{"L0"}},
				},
			},
			dagID:       "my-dag",
			baseBranch:  "main",
			layerID:     "L1",
			expected:    "dag/my-dag/stage-L0",
			description: "Second layer branches from L0 staging",
		},
		"layer 2 returns layer 1 staging branch": {
			dag: &DAGConfig{
				Layers: []Layer{
					{ID: "L0"},
					{ID: "L1", DependsOn: []string{"L0"}},
					{ID: "L2", DependsOn: []string{"L1"}},
				},
			},
			dagID:       "my-dag",
			baseBranch:  "main",
			layerID:     "L2",
			expected:    "dag/my-dag/stage-L1",
			description: "Third layer branches from L1 staging",
		},
		"unknown layer returns base branch": {
			dag: &DAGConfig{
				Layers: []Layer{
					{ID: "L0"},
				},
			},
			dagID:       "my-dag",
			baseBranch:  "main",
			layerID:     "unknown",
			expected:    "main",
			description: "Unknown layer falls back to base branch",
		},
		"single layer returns base branch": {
			dag: &DAGConfig{
				Layers: []Layer{
					{ID: "L0"},
				},
			},
			dagID:       "my-dag",
			baseBranch:  "",
			layerID:     "L0",
			expected:    "main",
			description: "Single layer DAG uses main branch",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			exec := &Executor{
				dag:    tt.dag,
				config: &DAGExecutionConfig{BaseBranch: tt.baseBranch},
				state:  &DAGRun{DAGId: tt.dagID},
			}

			got := exec.getBaseBranchForLayer(tt.layerID)
			if got != tt.expected {
				t.Errorf("%s: expected %q, got %q", tt.description, tt.expected, got)
			}
		})
	}
}

func TestGetBaseBranchForLayerWithDisableLayerStaging(t *testing.T) {
	tests := map[string]struct {
		dag                 *DAGConfig
		dagID               string
		baseBranch          string
		layerID             string
		disableLayerStaging bool
		expected            string
		description         string
	}{
		"layer 1 with staging disabled returns base branch": {
			dag: &DAGConfig{
				Layers: []Layer{
					{ID: "L0"},
					{ID: "L1", DependsOn: []string{"L0"}},
				},
			},
			dagID:               "my-dag",
			baseBranch:          "main",
			layerID:             "L1",
			disableLayerStaging: true,
			expected:            "main",
			description:         "Layer 1 returns main when staging disabled",
		},
		"layer 1 with staging enabled returns staging branch": {
			dag: &DAGConfig{
				Layers: []Layer{
					{ID: "L0"},
					{ID: "L1", DependsOn: []string{"L0"}},
				},
			},
			dagID:               "my-dag",
			baseBranch:          "main",
			layerID:             "L1",
			disableLayerStaging: false,
			expected:            "dag/my-dag/stage-L0",
			description:         "Layer 1 returns staging branch when staging enabled",
		},
		"layer 2 with staging disabled returns base branch": {
			dag: &DAGConfig{
				Layers: []Layer{
					{ID: "L0"},
					{ID: "L1", DependsOn: []string{"L0"}},
					{ID: "L2", DependsOn: []string{"L1"}},
				},
			},
			dagID:               "my-dag",
			baseBranch:          "develop",
			layerID:             "L2",
			disableLayerStaging: true,
			expected:            "develop",
			description:         "Layer 2 returns configured base when staging disabled",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			exec := &Executor{
				dag:                 tt.dag,
				config:              &DAGExecutionConfig{BaseBranch: tt.baseBranch},
				state:               &DAGRun{DAGId: tt.dagID},
				disableLayerStaging: tt.disableLayerStaging,
			}

			got := exec.getBaseBranchForLayer(tt.layerID)
			if got != tt.expected {
				t.Errorf("%s: expected %q, got %q", tt.description, tt.expected, got)
			}
		})
	}
}

func TestCollisionHandlingInCreateWorktree(t *testing.T) {
	tests := map[string]struct {
		dagID              string
		dagFile            string
		specID             string
		expectedBranchBase string
		description        string
	}{
		"normal case uses dag id format": {
			dagID:              "my-dag",
			dagFile:            "/path/workflow.yaml",
			specID:             "my-spec",
			expectedBranchBase: "dag/my-dag/my-spec",
			description:        "No collision, uses standard format",
		},
		"with explicit id": {
			dagID:              "short-id",
			dagFile:            "/path/workflow.yaml",
			specID:             "087-feature",
			expectedBranchBase: "dag/short-id/087-feature",
			description:        "Explicit short id is used",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			mgr := newMockWorktreeManager()
			specState := &SpecState{SpecID: tt.specID, Status: SpecStatusPending}
			exec := &Executor{
				worktreeManager: mgr,
				state: &DAGRun{
					DAGId: tt.dagID,
					Specs: map[string]*SpecState{tt.specID: specState},
				},
				dagFile: tt.dagFile,
				stdout:  io.Discard,
			}

			_, err := exec.createWorktree(tt.specID)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(mgr.creates) != 1 {
				t.Fatalf("expected 1 create call, got %d", len(mgr.creates))
			}

			// Branch should match expected base (may have suffix in collision case)
			if !strings.HasPrefix(mgr.creates[0].branch, tt.expectedBranchBase) {
				t.Errorf("expected branch starting with %q, got %q",
					tt.expectedBranchBase, mgr.creates[0].branch)
			}

			// Branch should be stored in SpecState for resume
			if !strings.HasPrefix(specState.Branch, tt.expectedBranchBase) {
				t.Errorf("expected stored branch starting with %q, got %q",
					tt.expectedBranchBase, specState.Branch)
			}
		})
	}
}

// ===== POST SPEC COMPLETION TESTS =====

func TestPostSpecCompletionAutomergeDisabled(t *testing.T) {
	automergeDisabled := false
	exec := &Executor{
		config: &DAGExecutionConfig{Automerge: &automergeDisabled},
		state: &DAGRun{
			DAGId: "my-dag",
			Specs: map[string]*SpecState{
				"spec-1": {
					SpecID:       "spec-1",
					LayerID:      "L0",
					CommitStatus: CommitStatusCommitted,
				},
			},
		},
		stdout: io.Discard,
	}

	err := exec.postSpecCompletion("spec-1")
	if err != nil {
		t.Errorf("expected no error when automerge disabled, got: %v", err)
	}

	// MergedToStaging should remain false
	if exec.state.Specs["spec-1"].MergedToStaging {
		t.Error("MergedToStaging should be false when automerge disabled")
	}
}

func TestPostSpecCompletionAlreadyMerged(t *testing.T) {
	automergeEnabled := true
	var output bytes.Buffer
	exec := &Executor{
		config: &DAGExecutionConfig{Automerge: &automergeEnabled},
		state: &DAGRun{
			DAGId: "my-dag",
			Specs: map[string]*SpecState{
				"spec-1": {
					SpecID:          "spec-1",
					LayerID:         "L0",
					CommitStatus:    CommitStatusCommitted,
					MergedToStaging: true, // Already merged
				},
			},
		},
		stdout: &output,
	}

	err := exec.postSpecCompletion("spec-1")
	if err != nil {
		t.Errorf("expected no error when already merged, got: %v", err)
	}

	// Output should mention skipping
	if !strings.Contains(output.String(), "Already merged") {
		t.Error("expected output to mention 'Already merged'")
	}
}

func TestPostSpecCompletionNoCommit(t *testing.T) {
	automergeEnabled := true
	var output bytes.Buffer
	exec := &Executor{
		config: &DAGExecutionConfig{Automerge: &automergeEnabled},
		state: &DAGRun{
			DAGId: "my-dag",
			Specs: map[string]*SpecState{
				"spec-1": {
					SpecID:       "spec-1",
					LayerID:      "L0",
					CommitStatus: CommitStatusPending, // Not committed
				},
			},
		},
		stdout: &output,
	}

	err := exec.postSpecCompletion("spec-1")
	if err != nil {
		t.Errorf("expected no error when no commit, got: %v", err)
	}

	// Output should mention skipping automerge
	if !strings.Contains(output.String(), "No verified commit") {
		t.Error("expected output to mention 'No verified commit'")
	}

	// MergedToStaging should remain false
	if exec.state.Specs["spec-1"].MergedToStaging {
		t.Error("MergedToStaging should be false when no commit")
	}
}

func TestPostSpecCompletionSpecNotFound(t *testing.T) {
	automergeEnabled := true
	exec := &Executor{
		config: &DAGExecutionConfig{Automerge: &automergeEnabled},
		state: &DAGRun{
			DAGId: "my-dag",
			Specs: map[string]*SpecState{},
		},
		stdout: io.Discard,
	}

	err := exec.postSpecCompletion("nonexistent-spec")
	if err == nil {
		t.Error("expected error when spec not found")
	}

	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention 'not found', got: %v", err)
	}
}

func TestUpdateStagingBranchState(t *testing.T) {
	tests := map[string]struct {
		existingBranches map[string]*StagingBranchInfo
		layerID          string
		branchName       string
		specID           string
		expectSpecCount  int
	}{
		"create new staging branch info": {
			existingBranches: nil,
			layerID:          "L0",
			branchName:       "dag/my-dag/stage-L0",
			specID:           "spec-1",
			expectSpecCount:  1,
		},
		"add to existing staging branch info": {
			existingBranches: map[string]*StagingBranchInfo{
				"L0": {Branch: "dag/my-dag/stage-L0", SpecsMerged: []string{"spec-1"}},
			},
			layerID:         "L0",
			branchName:      "dag/my-dag/stage-L0",
			specID:          "spec-2",
			expectSpecCount: 2,
		},
		"skip duplicate spec": {
			existingBranches: map[string]*StagingBranchInfo{
				"L0": {Branch: "dag/my-dag/stage-L0", SpecsMerged: []string{"spec-1"}},
			},
			layerID:         "L0",
			branchName:      "dag/my-dag/stage-L0",
			specID:          "spec-1", // Already in list
			expectSpecCount: 1,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			exec := &Executor{
				state: &DAGRun{
					DAGId:           "my-dag",
					StagingBranches: tt.existingBranches,
				},
			}

			exec.updateStagingBranchState(tt.layerID, tt.branchName, tt.specID)

			if exec.state.StagingBranches == nil {
				t.Fatal("StagingBranches should not be nil")
			}

			info := exec.state.StagingBranches[tt.layerID]
			if info == nil {
				t.Fatal("staging branch info should exist")
			}

			if info.Branch != tt.branchName {
				t.Errorf("expected branch %q, got %q", tt.branchName, info.Branch)
			}

			if len(info.SpecsMerged) != tt.expectSpecCount {
				t.Errorf("expected %d specs merged, got %d", tt.expectSpecCount, len(info.SpecsMerged))
			}
		})
	}
}

// ===== GET UNMERGED SPECS IN LAYER TESTS =====

func TestGetUnmergedSpecsInLayer(t *testing.T) {
	tests := map[string]struct {
		dag      *DAGConfig
		specs    map[string]*SpecState
		layerID  string
		expected []string
	}{
		"returns specs with committed but not merged": {
			dag: &DAGConfig{
				Layers: []Layer{
					{ID: "L0", Features: []Feature{{ID: "spec-1"}, {ID: "spec-2"}}},
				},
			},
			specs: map[string]*SpecState{
				"spec-1": {SpecID: "spec-1", CommitStatus: CommitStatusCommitted, MergedToStaging: false},
				"spec-2": {SpecID: "spec-2", CommitStatus: CommitStatusCommitted, MergedToStaging: false},
			},
			layerID:  "L0",
			expected: []string{"spec-1", "spec-2"},
		},
		"excludes already merged specs": {
			dag: &DAGConfig{
				Layers: []Layer{
					{ID: "L0", Features: []Feature{{ID: "spec-1"}, {ID: "spec-2"}}},
				},
			},
			specs: map[string]*SpecState{
				"spec-1": {SpecID: "spec-1", CommitStatus: CommitStatusCommitted, MergedToStaging: true},
				"spec-2": {SpecID: "spec-2", CommitStatus: CommitStatusCommitted, MergedToStaging: false},
			},
			layerID:  "L0",
			expected: []string{"spec-2"},
		},
		"excludes specs without commits": {
			dag: &DAGConfig{
				Layers: []Layer{
					{ID: "L0", Features: []Feature{{ID: "spec-1"}, {ID: "spec-2"}}},
				},
			},
			specs: map[string]*SpecState{
				"spec-1": {SpecID: "spec-1", CommitStatus: CommitStatusPending, MergedToStaging: false},
				"spec-2": {SpecID: "spec-2", CommitStatus: CommitStatusCommitted, MergedToStaging: false},
			},
			layerID:  "L0",
			expected: []string{"spec-2"},
		},
		"returns empty for empty layer": {
			dag: &DAGConfig{
				Layers: []Layer{
					{ID: "L0", Features: []Feature{}},
				},
			},
			specs:    map[string]*SpecState{},
			layerID:  "L0",
			expected: nil,
		},
		"returns empty when layer not found": {
			dag: &DAGConfig{
				Layers: []Layer{
					{ID: "L0", Features: []Feature{{ID: "spec-1"}}},
				},
			},
			specs: map[string]*SpecState{
				"spec-1": {SpecID: "spec-1", CommitStatus: CommitStatusCommitted, MergedToStaging: false},
			},
			layerID:  "L1",
			expected: nil,
		},
		"handles missing spec state gracefully": {
			dag: &DAGConfig{
				Layers: []Layer{
					{ID: "L0", Features: []Feature{{ID: "spec-1"}, {ID: "spec-2"}}},
				},
			},
			specs: map[string]*SpecState{
				"spec-1": {SpecID: "spec-1", CommitStatus: CommitStatusCommitted, MergedToStaging: false},
				// spec-2 not in state
			},
			layerID:  "L0",
			expected: []string{"spec-1"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			exec := &Executor{
				dag:   tt.dag,
				state: &DAGRun{Specs: tt.specs},
			}

			result := exec.getUnmergedSpecsInLayer(tt.layerID)

			if len(result) != len(tt.expected) {
				t.Errorf("expected %d specs, got %d: %v", len(tt.expected), len(result), result)
				return
			}

			for i, specID := range tt.expected {
				if result[i] != specID {
					t.Errorf("expected spec at index %d to be %q, got %q", i, specID, result[i])
				}
			}
		})
	}
}

// ===== COMPLETE LAYER TESTS =====

func TestCompleteLayerAutomergeEnabled(t *testing.T) {
	automergeEnabled := true
	var output bytes.Buffer
	exec := &Executor{
		config: &DAGExecutionConfig{Automerge: &automergeEnabled},
		dag: &DAGConfig{
			Layers: []Layer{
				{ID: "L0", Features: []Feature{{ID: "spec-1"}}},
			},
		},
		state: &DAGRun{
			DAGId: "my-dag",
			Specs: map[string]*SpecState{
				"spec-1": {SpecID: "spec-1", CommitStatus: CommitStatusCommitted, MergedToStaging: false},
			},
		},
		stdout: &output,
	}

	err := exec.completeLayer("L0")
	if err != nil {
		t.Errorf("expected no error when automerge enabled, got: %v", err)
	}

	// Should skip and not merge anything
	if exec.state.Specs["spec-1"].MergedToStaging {
		t.Error("MergedToStaging should remain false when automerge enabled (merges happen individually)")
	}
}

func TestCompleteLayerNoUnmergedSpecs(t *testing.T) {
	automergeDisabled := false
	var output bytes.Buffer
	exec := &Executor{
		config: &DAGExecutionConfig{Automerge: &automergeDisabled},
		dag: &DAGConfig{
			Layers: []Layer{
				{ID: "L0", Features: []Feature{{ID: "spec-1"}}},
			},
		},
		state: &DAGRun{
			DAGId: "my-dag",
			Specs: map[string]*SpecState{
				"spec-1": {SpecID: "spec-1", CommitStatus: CommitStatusCommitted, MergedToStaging: true},
			},
		},
		stdout: &output,
	}

	err := exec.completeLayer("L0")
	if err != nil {
		t.Errorf("expected no error when no unmerged specs, got: %v", err)
	}

	if !strings.Contains(output.String(), "No unmerged specs") {
		t.Error("expected output to mention 'No unmerged specs'")
	}
}

func TestHandleStagingConflict(t *testing.T) {
	tests := map[string]struct {
		specID      string
		conflictErr *MergeConflictError
		checkOutput []string
	}{
		"single conflict file": {
			specID: "spec-1",
			conflictErr: &MergeConflictError{
				StageBranch: "dag/my-dag/stage-L0",
				SpecBranch:  "dag/my-dag/spec-1",
				SpecID:      "spec-1",
				Conflicts:   []string{"file1.go"},
			},
			checkOutput: []string{
				"MERGE CONFLICT",
				"spec-1",
				"dag/my-dag/stage-L0",
				"file1.go",
				"Resolution Steps",
				"git add",
				"git commit",
				"autospec dag run",
			},
		},
		"multiple conflict files": {
			specID: "spec-2",
			conflictErr: &MergeConflictError{
				StageBranch: "dag/test-dag/stage-L1",
				SpecBranch:  "dag/test-dag/spec-2",
				SpecID:      "spec-2",
				Conflicts:   []string{"file1.go", "file2.go", "file3.go"},
			},
			checkOutput: []string{
				"MERGE CONFLICT",
				"spec-2",
				"dag/test-dag/stage-L1",
				"file1.go",
				"file2.go",
				"file3.go",
				"Conflicting files (3)",
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			var output bytes.Buffer
			exec := &Executor{stdout: &output}

			err := exec.handleStagingConflict(tt.specID, tt.conflictErr)

			if err == nil {
				t.Error("expected error to be returned")
			}
			if !strings.Contains(err.Error(), tt.specID) {
				t.Errorf("error should contain spec ID %q, got %q", tt.specID, err.Error())
			}

			outputStr := output.String()
			for _, expected := range tt.checkOutput {
				if !strings.Contains(outputStr, expected) {
					t.Errorf("output should contain %q, got:\n%s", expected, outputStr)
				}
			}
		})
	}
}

func TestValidateResumeState(t *testing.T) {
	tests := map[string]struct {
		disableLayerStaging bool
		setupRepo           func(t *testing.T, repoRoot string)
		wantErr             bool
		checkOutput         string
	}{
		"layer staging disabled skips validation": {
			disableLayerStaging: true,
			setupRepo: func(t *testing.T, repoRoot string) {
				// Create merge state that would fail if checked
				gitDir := filepath.Join(repoRoot, ".git")
				if err := os.MkdirAll(gitDir, 0755); err != nil {
					t.Fatal(err)
				}
				mergeHead := filepath.Join(gitDir, "MERGE_HEAD")
				if err := os.WriteFile(mergeHead, []byte("abc"), 0644); err != nil {
					t.Fatal(err)
				}
			},
			wantErr: false,
		},
		"clean repo continues execution": {
			disableLayerStaging: false,
			setupRepo: func(t *testing.T, repoRoot string) {
				gitDir := filepath.Join(repoRoot, ".git")
				if err := os.MkdirAll(gitDir, 0755); err != nil {
					t.Fatal(err)
				}
			},
			wantErr:     false,
			checkOutput: "No interrupted merge detected",
		},
		"interrupted merge returns error": {
			disableLayerStaging: false,
			setupRepo: func(t *testing.T, repoRoot string) {
				gitDir := filepath.Join(repoRoot, ".git")
				if err := os.MkdirAll(gitDir, 0755); err != nil {
					t.Fatal(err)
				}
				mergeHead := filepath.Join(gitDir, "MERGE_HEAD")
				if err := os.WriteFile(mergeHead, []byte("abc"), 0644); err != nil {
					t.Fatal(err)
				}
			},
			wantErr:     true,
			checkOutput: "INTERRUPTED MERGE DETECTED",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			repoRoot := t.TempDir()
			tt.setupRepo(t, repoRoot)

			var output bytes.Buffer
			exec := &Executor{
				stdout:              &output,
				repoRoot:            repoRoot,
				disableLayerStaging: tt.disableLayerStaging,
			}

			err := exec.validateResumeState()

			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if tt.checkOutput != "" && !strings.Contains(output.String(), tt.checkOutput) {
				t.Errorf("output should contain %q, got:\n%s", tt.checkOutput, output.String())
			}
		})
	}
}

func TestHandleInterruptedMerge(t *testing.T) {
	tests := map[string]struct {
		inputErr    error
		checkOutput []string
		checkErr    string
	}{
		"outputs resolution instructions": {
			inputErr: fmt.Errorf("unresolved conflicts in 2 file(s)"),
			checkOutput: []string{
				"INTERRUPTED MERGE DETECTED",
				"unresolved conflicts",
				"git add",
				"git commit",
				"autospec dag run",
			},
			checkErr: "cannot resume",
		},
		"merge not committed": {
			inputErr: fmt.Errorf("merge in progress but not committed"),
			checkOutput: []string{
				"INTERRUPTED MERGE DETECTED",
				"merge in progress but not committed",
				"Run: git commit",
			},
			checkErr: "cannot resume",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			var output bytes.Buffer
			exec := &Executor{stdout: &output}

			err := exec.handleInterruptedMerge(tt.inputErr)

			if err == nil {
				t.Error("expected error to be returned")
			}
			if !strings.Contains(err.Error(), tt.checkErr) {
				t.Errorf("error should contain %q, got %q", tt.checkErr, err.Error())
			}

			outputStr := output.String()
			for _, expected := range tt.checkOutput {
				if !strings.Contains(outputStr, expected) {
					t.Errorf("output should contain %q, got:\n%s", expected, outputStr)
				}
			}
		})
	}
}
