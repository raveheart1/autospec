package dag

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
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
	exec := &Executor{
		state: &DAGRun{RunID: "20250111_120000_abc12345"},
	}

	name := exec.worktreeName("my-spec")
	expected := "dag-20250111_120000_abc12345-my-spec"

	if name != expected {
		t.Errorf("expected %q, got %q", expected, name)
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
