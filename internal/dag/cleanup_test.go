package dag

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ariel-frischer/autospec/internal/worktree"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockCleanupManager implements worktree.Manager for testing.
type mockCleanupManager struct {
	removeFunc  func(name string, force bool) error
	removeCalls []removeCall
}

type removeCall struct {
	name  string
	force bool
}

func (m *mockCleanupManager) Create(name, branch, customPath string) (*worktree.Worktree, error) {
	return nil, nil
}

func (m *mockCleanupManager) CreateWithOptions(name, branch, customPath string, opts worktree.CreateOptions) (*worktree.Worktree, error) {
	return nil, nil
}

func (m *mockCleanupManager) List() ([]worktree.Worktree, error) {
	return nil, nil
}

func (m *mockCleanupManager) Get(name string) (*worktree.Worktree, error) {
	return nil, nil
}

func (m *mockCleanupManager) Remove(name string, force bool) error {
	m.removeCalls = append(m.removeCalls, removeCall{name: name, force: force})
	if m.removeFunc != nil {
		return m.removeFunc(name, force)
	}
	return nil
}

func (m *mockCleanupManager) Setup(path string, addToState bool) (*worktree.Worktree, error) {
	return nil, nil
}

func (m *mockCleanupManager) Prune() (int, error) {
	return 0, nil
}

func (m *mockCleanupManager) UpdateStatus(name string, status worktree.WorktreeStatus) error {
	return nil
}

func TestCleanupExecutor_CleanupRun_RemovesMergedWorktrees(t *testing.T) {
	tests := map[string]struct {
		specs         map[string]*SpecState
		expectedClean []string
		expectedKept  []string
	}{
		"removes merged spec worktree": {
			specs: map[string]*SpecState{
				"spec-1": {
					SpecID:       "spec-1",
					WorktreePath: "/tmp/wt-spec-1",
					Status:       SpecStatusCompleted,
					Merge: &MergeState{
						Status: MergeStatusMerged,
					},
				},
			},
			expectedClean: []string{"spec-1"},
			expectedKept:  []string{},
		},
		"preserves unmerged spec worktree": {
			specs: map[string]*SpecState{
				"spec-1": {
					SpecID:       "spec-1",
					WorktreePath: "/tmp/wt-spec-1",
					Status:       SpecStatusCompleted,
					Merge:        nil, // Not merged yet
				},
			},
			expectedClean: []string{},
			expectedKept:  []string{"spec-1"},
		},
		"preserves merge_failed spec worktree": {
			specs: map[string]*SpecState{
				"spec-1": {
					SpecID:       "spec-1",
					WorktreePath: "/tmp/wt-spec-1",
					Status:       SpecStatusCompleted,
					Merge: &MergeState{
						Status: MergeStatusMergeFailed,
					},
				},
			},
			expectedClean: []string{},
			expectedKept:  []string{"spec-1"},
		},
		"preserves skipped spec worktree": {
			specs: map[string]*SpecState{
				"spec-1": {
					SpecID:       "spec-1",
					WorktreePath: "/tmp/wt-spec-1",
					Status:       SpecStatusCompleted,
					Merge: &MergeState{
						Status: MergeStatusSkipped,
					},
				},
			},
			expectedClean: []string{},
			expectedKept:  []string{"spec-1"},
		},
		"mixed merged and unmerged specs": {
			specs: map[string]*SpecState{
				"spec-merged": {
					SpecID:       "spec-merged",
					WorktreePath: "/tmp/wt-spec-merged",
					Status:       SpecStatusCompleted,
					Merge: &MergeState{
						Status: MergeStatusMerged,
					},
				},
				"spec-pending": {
					SpecID:       "spec-pending",
					WorktreePath: "/tmp/wt-spec-pending",
					Status:       SpecStatusCompleted,
					Merge: &MergeState{
						Status: MergeStatusPending,
					},
				},
			},
			expectedClean: []string{"spec-merged"},
			expectedKept:  []string{"spec-pending"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Create temp directory structure
			tmpDir := t.TempDir()
			stateDir := filepath.Join(tmpDir, ".autospec", "state", "dag-runs")
			require.NoError(t, os.MkdirAll(stateDir, 0o755))

			// Create worktree directories that exist
			for _, spec := range tt.specs {
				if spec.WorktreePath != "" {
					// Create a temp dir to simulate existing worktree
					wtDir := filepath.Join(tmpDir, filepath.Base(spec.WorktreePath))
					require.NoError(t, os.MkdirAll(wtDir, 0o755))
					spec.WorktreePath = wtDir
				}
			}

			// Create run state
			run := &DAGRun{
				RunID:     "test-run-123",
				DAGFile:   "dag.yaml",
				Status:    RunStatusCompleted,
				StartedAt: time.Now(),
				Specs:     tt.specs,
			}
			require.NoError(t, SaveState(stateDir, run))

			// Create mock manager
			mock := &mockCleanupManager{}

			// Create executor
			var stdout bytes.Buffer
			exec := NewCleanupExecutor(
				stateDir,
				mock,
				WithCleanupStdout(&stdout),
			)

			result, err := exec.CleanupRun("test-run-123")
			require.NoError(t, err)
			require.NotNil(t, result)

			assert.ElementsMatch(t, tt.expectedClean, result.Cleaned)
			assert.ElementsMatch(t, tt.expectedKept, result.Kept)
		})
	}
}

func TestCleanupExecutor_CleanupRun_ForceBypassesChecks(t *testing.T) {
	tests := map[string]struct {
		force         bool
		mergeStatus   MergeStatus
		expectedClean bool
	}{
		"force cleans unmerged specs": {
			force:         true,
			mergeStatus:   MergeStatusPending,
			expectedClean: true,
		},
		"force cleans failed specs": {
			force:         true,
			mergeStatus:   MergeStatusMergeFailed,
			expectedClean: true,
		},
		"no force preserves unmerged": {
			force:         false,
			mergeStatus:   MergeStatusPending,
			expectedClean: false,
		},
		"no force preserves failed": {
			force:         false,
			mergeStatus:   MergeStatusMergeFailed,
			expectedClean: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			tmpDir := t.TempDir()
			stateDir := filepath.Join(tmpDir, ".autospec", "state", "dag-runs")
			require.NoError(t, os.MkdirAll(stateDir, 0o755))

			// Create worktree directory
			wtDir := filepath.Join(tmpDir, "wt-spec-1")
			require.NoError(t, os.MkdirAll(wtDir, 0o755))

			run := &DAGRun{
				RunID:     "test-run-456",
				DAGFile:   "dag.yaml",
				Status:    RunStatusCompleted,
				StartedAt: time.Now(),
				Specs: map[string]*SpecState{
					"spec-1": {
						SpecID:       "spec-1",
						WorktreePath: wtDir,
						Status:       SpecStatusCompleted,
						Merge: &MergeState{
							Status: tt.mergeStatus,
						},
					},
				},
			}
			require.NoError(t, SaveState(stateDir, run))

			mock := &mockCleanupManager{}
			var stdout bytes.Buffer
			exec := NewCleanupExecutor(
				stateDir,
				mock,
				WithCleanupStdout(&stdout),
				WithCleanupForce(tt.force),
			)

			result, err := exec.CleanupRun("test-run-456")
			require.NoError(t, err)

			if tt.expectedClean {
				assert.Contains(t, result.Cleaned, "spec-1")
				assert.NotContains(t, result.Kept, "spec-1")
			} else {
				assert.NotContains(t, result.Cleaned, "spec-1")
				assert.Contains(t, result.Kept, "spec-1")
			}
		})
	}
}

func TestCleanupExecutor_CleanupRun_HandlesNonexistentWorktree(t *testing.T) {
	tmpDir := t.TempDir()
	stateDir := filepath.Join(tmpDir, ".autospec", "state", "dag-runs")
	require.NoError(t, os.MkdirAll(stateDir, 0o755))

	run := &DAGRun{
		RunID:     "test-run-789",
		DAGFile:   "dag.yaml",
		Status:    RunStatusCompleted,
		StartedAt: time.Now(),
		Specs: map[string]*SpecState{
			"spec-1": {
				SpecID:       "spec-1",
				WorktreePath: "/nonexistent/path/that/does/not/exist",
				Status:       SpecStatusCompleted,
				Merge: &MergeState{
					Status: MergeStatusMerged,
				},
			},
		},
	}
	require.NoError(t, SaveState(stateDir, run))

	mock := &mockCleanupManager{}
	var stdout bytes.Buffer
	exec := NewCleanupExecutor(
		stateDir,
		mock,
		WithCleanupStdout(&stdout),
	)

	result, err := exec.CleanupRun("test-run-789")
	require.NoError(t, err)

	// Should have a warning about nonexistent worktree
	assert.Len(t, result.Warnings, 1)
	assert.Contains(t, result.Warnings[0], "no longer exists")
}

func TestCleanupExecutor_CleanupRun_TracksRemoveErrors(t *testing.T) {
	tmpDir := t.TempDir()
	stateDir := filepath.Join(tmpDir, ".autospec", "state", "dag-runs")
	require.NoError(t, os.MkdirAll(stateDir, 0o755))

	// Create worktree directory
	wtDir := filepath.Join(tmpDir, "wt-spec-error")
	require.NoError(t, os.MkdirAll(wtDir, 0o755))

	run := &DAGRun{
		RunID:     "test-run-error",
		DAGFile:   "dag.yaml",
		Status:    RunStatusCompleted,
		StartedAt: time.Now(),
		Specs: map[string]*SpecState{
			"spec-error": {
				SpecID:       "spec-error",
				WorktreePath: wtDir,
				Status:       SpecStatusCompleted,
				Merge: &MergeState{
					Status: MergeStatusMerged,
				},
			},
		},
	}
	require.NoError(t, SaveState(stateDir, run))

	// Mock manager that returns an error
	mock := &mockCleanupManager{
		removeFunc: func(name string, force bool) error {
			return assert.AnError
		},
	}

	var stdout bytes.Buffer
	exec := NewCleanupExecutor(
		stateDir,
		mock,
		WithCleanupStdout(&stdout),
	)

	result, err := exec.CleanupRun("test-run-error")
	require.NoError(t, err)

	assert.Len(t, result.Errors, 1)
	assert.Contains(t, result.Errors, "spec-error")
}

func TestCleanupExecutor_CleanupAllRuns_SkipsRunning(t *testing.T) {
	tmpDir := t.TempDir()
	stateDir := filepath.Join(tmpDir, ".autospec", "state", "dag-runs")
	require.NoError(t, os.MkdirAll(stateDir, 0o755))

	// Create completed run
	completedRun := &DAGRun{
		RunID:     "completed-run",
		DAGFile:   "dag.yaml",
		Status:    RunStatusCompleted,
		StartedAt: time.Now(),
		Specs:     map[string]*SpecState{},
	}
	require.NoError(t, SaveState(stateDir, completedRun))

	// Create running run
	runningRun := &DAGRun{
		RunID:     "running-run",
		DAGFile:   "dag.yaml",
		Status:    RunStatusRunning,
		StartedAt: time.Now(),
		Specs:     map[string]*SpecState{},
	}
	require.NoError(t, SaveState(stateDir, runningRun))

	mock := &mockCleanupManager{}
	var stdout bytes.Buffer
	exec := NewCleanupExecutor(
		stateDir,
		mock,
		WithCleanupStdout(&stdout),
	)

	results, err := exec.CleanupAllRuns()
	require.NoError(t, err)

	// Should only process the completed run
	assert.Len(t, results, 1)
}

func TestCleanupResult_HasSummary(t *testing.T) {
	tests := map[string]struct {
		result   *CleanupResult
		expected bool
	}{
		"empty result has no summary": {
			result: &CleanupResult{
				Cleaned: []string{},
				Kept:    []string{},
				Errors:  map[string]string{},
			},
			expected: false,
		},
		"result with cleaned has summary": {
			result: &CleanupResult{
				Cleaned: []string{"spec-1"},
				Kept:    []string{},
				Errors:  map[string]string{},
			},
			expected: true,
		},
		"result with kept has summary": {
			result: &CleanupResult{
				Cleaned: []string{},
				Kept:    []string{"spec-1"},
				Errors:  map[string]string{},
			},
			expected: true,
		},
		"result with errors has summary": {
			result: &CleanupResult{
				Cleaned: []string{},
				Kept:    []string{},
				Errors:  map[string]string{"spec-1": "error"},
			},
			expected: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.result.HasSummary())
		})
	}
}

func TestCleanupResult_TotalProcessed(t *testing.T) {
	tests := map[string]struct {
		result   *CleanupResult
		expected int
	}{
		"empty result": {
			result: &CleanupResult{
				Cleaned: []string{},
				Kept:    []string{},
				Errors:  map[string]string{},
			},
			expected: 0,
		},
		"mixed results": {
			result: &CleanupResult{
				Cleaned: []string{"a", "b"},
				Kept:    []string{"c"},
				Errors:  map[string]string{"d": "error"},
			},
			expected: 4,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.result.TotalProcessed())
		})
	}
}

func TestCleanupExecutor_CleanupRun_SkipsSpecsWithoutWorktreePath(t *testing.T) {
	tmpDir := t.TempDir()
	stateDir := filepath.Join(tmpDir, ".autospec", "state", "dag-runs")
	require.NoError(t, os.MkdirAll(stateDir, 0o755))

	run := &DAGRun{
		RunID:     "test-run-no-wt",
		DAGFile:   "dag.yaml",
		Status:    RunStatusCompleted,
		StartedAt: time.Now(),
		Specs: map[string]*SpecState{
			"spec-no-wt": {
				SpecID:       "spec-no-wt",
				WorktreePath: "", // Empty path
				Status:       SpecStatusCompleted,
				Merge: &MergeState{
					Status: MergeStatusMerged,
				},
			},
		},
	}
	require.NoError(t, SaveState(stateDir, run))

	mock := &mockCleanupManager{}
	var stdout bytes.Buffer
	exec := NewCleanupExecutor(
		stateDir,
		mock,
		WithCleanupStdout(&stdout),
	)

	result, err := exec.CleanupRun("test-run-no-wt")
	require.NoError(t, err)

	// Spec without worktree path should be skipped silently
	assert.Empty(t, result.Cleaned)
	assert.Empty(t, result.Kept)
	assert.Empty(t, result.Errors)
}

// Tests for CleanupByInlineState

func TestCleanupExecutor_CleanupByInlineState_RemovesMergedWorktrees(t *testing.T) {
	tests := map[string]struct {
		specs         map[string]*InlineSpecState
		expectedClean []string
		expectedKept  []string
	}{
		"removes merged spec worktree": {
			specs: map[string]*InlineSpecState{
				"spec-1": {
					Status:   InlineSpecStatusCompleted,
					Worktree: "/tmp/wt-spec-1",
					Merge: &MergeState{
						Status: MergeStatusMerged,
					},
				},
			},
			expectedClean: []string{"spec-1"},
			expectedKept:  []string{},
		},
		"preserves unmerged spec worktree": {
			specs: map[string]*InlineSpecState{
				"spec-1": {
					Status:   InlineSpecStatusCompleted,
					Worktree: "/tmp/wt-spec-1",
					Merge:    nil, // Not merged yet
				},
			},
			expectedClean: []string{},
			expectedKept:  []string{"spec-1"},
		},
		"preserves merge_failed spec worktree": {
			specs: map[string]*InlineSpecState{
				"spec-1": {
					Status:   InlineSpecStatusCompleted,
					Worktree: "/tmp/wt-spec-1",
					Merge: &MergeState{
						Status: MergeStatusMergeFailed,
					},
				},
			},
			expectedClean: []string{},
			expectedKept:  []string{"spec-1"},
		},
		"mixed merged and unmerged specs": {
			specs: map[string]*InlineSpecState{
				"spec-merged": {
					Status:   InlineSpecStatusCompleted,
					Worktree: "/tmp/wt-spec-merged",
					Merge: &MergeState{
						Status: MergeStatusMerged,
					},
				},
				"spec-pending": {
					Status:   InlineSpecStatusCompleted,
					Worktree: "/tmp/wt-spec-pending",
					Merge: &MergeState{
						Status: MergeStatusPending,
					},
				},
			},
			expectedClean: []string{"spec-merged"},
			expectedKept:  []string{"spec-pending"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			tmpDir := t.TempDir()
			stateDir := filepath.Join(tmpDir, ".autospec", "state", "dag-runs")
			require.NoError(t, os.MkdirAll(stateDir, 0o755))

			// Create worktree directories that exist
			for specID, spec := range tt.specs {
				if spec.Worktree != "" {
					wtDir := filepath.Join(tmpDir, "wt-"+specID)
					require.NoError(t, os.MkdirAll(wtDir, 0o755))
					spec.Worktree = wtDir
				}
			}

			// Create DAGConfig with inline state
			config := &DAGConfig{
				SchemaVersion: "1.0",
				DAG:           DAGMetadata{Name: "test-dag"},
				Layers:        []Layer{{ID: "L0", Features: []Feature{{ID: "spec-1"}}}},
				Run:           &InlineRunState{Status: InlineRunStatusCompleted},
				Specs:         tt.specs,
			}

			mock := &mockCleanupManager{}
			var stdout bytes.Buffer
			exec := NewCleanupExecutor(
				stateDir,
				mock,
				WithCleanupStdout(&stdout),
			)

			result, err := exec.CleanupByInlineState(config)
			require.NoError(t, err)
			require.NotNil(t, result)

			assert.ElementsMatch(t, tt.expectedClean, result.Cleaned)
			assert.ElementsMatch(t, tt.expectedKept, result.Kept)
		})
	}
}

func TestCleanupExecutor_CleanupByInlineState_ForceBypassesChecks(t *testing.T) {
	tests := map[string]struct {
		force         bool
		mergeStatus   MergeStatus
		expectedClean bool
	}{
		"force cleans unmerged specs": {
			force:         true,
			mergeStatus:   MergeStatusPending,
			expectedClean: true,
		},
		"force cleans failed specs": {
			force:         true,
			mergeStatus:   MergeStatusMergeFailed,
			expectedClean: true,
		},
		"no force preserves unmerged": {
			force:         false,
			mergeStatus:   MergeStatusPending,
			expectedClean: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			tmpDir := t.TempDir()
			stateDir := filepath.Join(tmpDir, ".autospec", "state", "dag-runs")
			require.NoError(t, os.MkdirAll(stateDir, 0o755))

			wtDir := filepath.Join(tmpDir, "wt-spec-1")
			require.NoError(t, os.MkdirAll(wtDir, 0o755))

			config := &DAGConfig{
				SchemaVersion: "1.0",
				DAG:           DAGMetadata{Name: "test-dag"},
				Layers:        []Layer{{ID: "L0", Features: []Feature{{ID: "spec-1"}}}},
				Run:           &InlineRunState{Status: InlineRunStatusCompleted},
				Specs: map[string]*InlineSpecState{
					"spec-1": {
						Status:   InlineSpecStatusCompleted,
						Worktree: wtDir,
						Merge: &MergeState{
							Status: tt.mergeStatus,
						},
					},
				},
			}

			mock := &mockCleanupManager{}
			var stdout bytes.Buffer
			exec := NewCleanupExecutor(
				stateDir,
				mock,
				WithCleanupStdout(&stdout),
				WithCleanupForce(tt.force),
			)

			result, err := exec.CleanupByInlineState(config)
			require.NoError(t, err)

			if tt.expectedClean {
				assert.Contains(t, result.Cleaned, "spec-1")
				assert.Empty(t, result.Kept)
			} else {
				assert.Empty(t, result.Cleaned)
				assert.Contains(t, result.Kept, "spec-1")
			}
		})
	}
}

func TestCleanupExecutor_CleanupByInlineState_NilConfig(t *testing.T) {
	tmpDir := t.TempDir()
	stateDir := filepath.Join(tmpDir, ".autospec", "state", "dag-runs")
	require.NoError(t, os.MkdirAll(stateDir, 0o755))

	mock := &mockCleanupManager{}
	var stdout bytes.Buffer
	exec := NewCleanupExecutor(
		stateDir,
		mock,
		WithCleanupStdout(&stdout),
	)

	result, err := exec.CleanupByInlineState(nil)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "config is nil")
}
