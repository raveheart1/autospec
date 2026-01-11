package worktree

import (
	"bytes"
	"fmt"
	"io"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockGitOps implements GitOperations for testing
type mockGitOps struct {
	addCalled      bool
	addErr         error
	removeCalled   bool
	removeErr      error
	listResult     []GitWorktreeEntry
	listErr        error
	uncommitted    bool
	uncommittedErr error
	unpushed       bool
	unpushedErr    error
}

func (m *mockGitOps) Add(repoPath, worktreePath, branch string) error {
	m.addCalled = true
	return m.addErr
}

func (m *mockGitOps) Remove(repoPath, worktreePath string, force bool) error {
	m.removeCalled = true
	return m.removeErr
}

func (m *mockGitOps) List(repoPath string) ([]GitWorktreeEntry, error) {
	return m.listResult, m.listErr
}

func (m *mockGitOps) HasUncommittedChanges(path string) (bool, error) {
	return m.uncommitted, m.uncommittedErr
}

func (m *mockGitOps) HasUnpushedCommits(path string) (bool, error) {
	return m.unpushed, m.unpushedErr
}

func TestNewManager_DefaultConfig(t *testing.T) {
	t.Parallel()

	manager := NewManager(nil, "/state", "/repo")

	assert.NotNil(t, manager)
	assert.NotNil(t, manager.config)
	assert.True(t, manager.config.AutoSetup)
	assert.True(t, manager.config.TrackStatus)
}

func TestNewManager_WithOptions(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	mockOps := &mockGitOps{}
	copyCalled := false
	setupCalled := false

	manager := NewManager(
		DefaultConfig(),
		"/state",
		"/repo",
		WithStdout(&buf),
		WithGitOps(mockOps),
		WithCopyFunc(func(src, dst string, dirs []string) ([]string, error) {
			copyCalled = true
			return dirs, nil
		}),
		WithSetupFunc(func(script, path, name, branch, repo string, w io.Writer) *SetupResult {
			setupCalled = true
			return &SetupResult{Executed: false}
		}),
	)

	assert.NotNil(t, manager)
	// Verify options were set by triggering behaviors
	_, _ = manager.copyFn("", "", nil)
	assert.True(t, copyCalled)
	_ = manager.runSetupFn("", "", "", "", "", nil)
	assert.True(t, setupCalled)
}

func TestManager_Create_Success(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()
	repoRoot := t.TempDir()
	baseDir := t.TempDir()

	mockOps := &mockGitOps{}
	var buf bytes.Buffer

	cfg := &WorktreeConfig{
		BaseDir:     baseDir,
		Prefix:      "wt-",
		AutoSetup:   false,
		TrackStatus: true,
		CopyDirs:    []string{},
	}

	manager := NewManager(cfg, stateDir, repoRoot,
		WithStdout(&buf),
		WithGitOps(mockOps),
		WithCopyFunc(func(src, dst string, dirs []string) ([]string, error) {
			return nil, nil
		}),
	)

	wt, err := manager.Create("test", "feature/test", "")
	require.NoError(t, err)
	assert.True(t, mockOps.addCalled)
	assert.Equal(t, "test", wt.Name)
	assert.Equal(t, "feature/test", wt.Branch)
	assert.Equal(t, StatusActive, wt.Status)
	assert.Equal(t, filepath.Join(baseDir, "wt-test"), wt.Path)
}

func TestManager_Create_DuplicateName(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()

	// Pre-populate state
	state := &WorktreeState{
		Version:   StateVersion,
		Worktrees: []Worktree{{Name: "existing"}},
	}
	require.NoError(t, SaveState(stateDir, state))

	manager := NewManager(DefaultConfig(), stateDir, "/repo")

	_, err := manager.Create("existing", "branch", "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestManager_Create_CustomPath(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()
	repoRoot := t.TempDir()
	customPath := filepath.Join(t.TempDir(), "custom-location")

	mockOps := &mockGitOps{}
	var buf bytes.Buffer

	cfg := &WorktreeConfig{
		AutoSetup:   false,
		TrackStatus: true,
		CopyDirs:    []string{},
	}

	manager := NewManager(cfg, stateDir, repoRoot,
		WithStdout(&buf),
		WithGitOps(mockOps),
		WithCopyFunc(func(src, dst string, dirs []string) ([]string, error) {
			return nil, nil
		}),
	)

	wt, err := manager.Create("test", "branch", customPath)
	require.NoError(t, err)
	assert.Equal(t, customPath, wt.Path)
}

func TestManager_List_Empty(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()
	manager := NewManager(DefaultConfig(), stateDir, "/repo")

	worktrees, err := manager.List()
	require.NoError(t, err)
	assert.Empty(t, worktrees)
}

func TestManager_List_WithWorktrees(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()
	existingPath := t.TempDir() // This path exists

	state := &WorktreeState{
		Version: StateVersion,
		Worktrees: []Worktree{
			{Name: "wt1", Path: existingPath, Status: StatusActive},
			{Name: "wt2", Path: "/nonexistent/path", Status: StatusActive},
		},
	}
	require.NoError(t, SaveState(stateDir, state))

	manager := NewManager(DefaultConfig(), stateDir, "/repo")

	worktrees, err := manager.List()
	require.NoError(t, err)
	require.Len(t, worktrees, 2)

	// Existing path keeps status
	assert.Equal(t, StatusActive, worktrees[0].Status)
	// Non-existing path becomes stale
	assert.Equal(t, StatusStale, worktrees[1].Status)
}

func TestManager_Get_Found(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()
	state := &WorktreeState{
		Version:   StateVersion,
		Worktrees: []Worktree{{Name: "test", Path: "/path"}},
	}
	require.NoError(t, SaveState(stateDir, state))

	manager := NewManager(DefaultConfig(), stateDir, "/repo")

	wt, err := manager.Get("test")
	require.NoError(t, err)
	assert.Equal(t, "test", wt.Name)
}

func TestManager_Get_NotFound(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()
	manager := NewManager(DefaultConfig(), stateDir, "/repo")

	_, err := manager.Get("missing")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestManager_Remove_Success(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()
	wtPath := t.TempDir() // Existing path

	state := &WorktreeState{
		Version:   StateVersion,
		Worktrees: []Worktree{{Name: "test", Path: wtPath}},
	}
	require.NoError(t, SaveState(stateDir, state))

	mockOps := &mockGitOps{
		uncommitted: false,
		unpushed:    false,
	}
	var buf bytes.Buffer

	manager := NewManager(DefaultConfig(), stateDir, "/repo",
		WithStdout(&buf),
		WithGitOps(mockOps),
	)

	err := manager.Remove("test", false)
	require.NoError(t, err)
	assert.True(t, mockOps.removeCalled)

	// Verify removed from state
	loaded, _ := LoadState(stateDir)
	assert.Nil(t, loaded.FindWorktree("test"))
}

func TestManager_Remove_WithUncommittedChanges(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()
	wtPath := t.TempDir()

	state := &WorktreeState{
		Version:   StateVersion,
		Worktrees: []Worktree{{Name: "test", Path: wtPath}},
	}
	require.NoError(t, SaveState(stateDir, state))

	mockOps := &mockGitOps{
		uncommitted: true,
	}

	manager := NewManager(DefaultConfig(), stateDir, "/repo",
		WithGitOps(mockOps),
	)

	err := manager.Remove("test", false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "uncommitted changes")
}

func TestManager_Remove_ForceBypassesChecks(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()
	wtPath := t.TempDir()

	state := &WorktreeState{
		Version:   StateVersion,
		Worktrees: []Worktree{{Name: "test", Path: wtPath}},
	}
	require.NoError(t, SaveState(stateDir, state))

	mockOps := &mockGitOps{
		uncommitted: true,
		unpushed:    true,
	}
	var buf bytes.Buffer

	manager := NewManager(DefaultConfig(), stateDir, "/repo",
		WithStdout(&buf),
		WithGitOps(mockOps),
	)

	err := manager.Remove("test", true)
	require.NoError(t, err)
	assert.True(t, mockOps.removeCalled)
}

func TestManager_Prune_RemovesStale(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()
	existingPath := t.TempDir()

	state := &WorktreeState{
		Version: StateVersion,
		Worktrees: []Worktree{
			{Name: "exists", Path: existingPath},
			{Name: "stale1", Path: "/nonexistent1"},
			{Name: "stale2", Path: "/nonexistent2"},
		},
	}
	require.NoError(t, SaveState(stateDir, state))

	manager := NewManager(DefaultConfig(), stateDir, "/repo")

	pruned, err := manager.Prune()
	require.NoError(t, err)
	assert.Equal(t, 2, pruned)

	// Verify only existing remains
	loaded, _ := LoadState(stateDir)
	assert.Len(t, loaded.Worktrees, 1)
	assert.Equal(t, "exists", loaded.Worktrees[0].Name)
}

func TestManager_Prune_NoneToRemove(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()
	existingPath := t.TempDir()

	state := &WorktreeState{
		Version:   StateVersion,
		Worktrees: []Worktree{{Name: "exists", Path: existingPath}},
	}
	require.NoError(t, SaveState(stateDir, state))

	manager := NewManager(DefaultConfig(), stateDir, "/repo")

	pruned, err := manager.Prune()
	require.NoError(t, err)
	assert.Equal(t, 0, pruned)
}

func TestManager_UpdateStatus(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		status  WorktreeStatus
		wantErr bool
	}{
		"update to merged": {
			status:  StatusMerged,
			wantErr: false,
		},
		"update to abandoned": {
			status:  StatusAbandoned,
			wantErr: false,
		},
		"invalid status": {
			status:  WorktreeStatus("invalid"),
			wantErr: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			stateDir := t.TempDir()
			state := &WorktreeState{
				Version:   StateVersion,
				Worktrees: []Worktree{{Name: "test", Status: StatusActive}},
			}
			require.NoError(t, SaveState(stateDir, state))

			manager := NewManager(DefaultConfig(), stateDir, "/repo")

			err := manager.UpdateStatus("test", tt.status)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				loaded, _ := LoadState(stateDir)
				assert.Equal(t, tt.status, loaded.Worktrees[0].Status)
			}
		})
	}
}

func TestManager_UpdateStatus_NotFound(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()
	manager := NewManager(DefaultConfig(), stateDir, "/repo")

	err := manager.UpdateStatus("missing", StatusMerged)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// TestManager_Setup_ExistingPath requires an actual git worktree to test,
// which is difficult to set up in isolation. The Setup method calls IsWorktree
// which executes real git commands. Integration tests would cover this scenario.
// For unit testing, we verify the error case works correctly.

func TestManager_Setup_PathNotExist(t *testing.T) {
	t.Parallel()

	manager := NewManager(DefaultConfig(), t.TempDir(), "/repo")

	_, err := manager.Setup("/nonexistent/path", false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
}

// TestManager_Create_ValidationIntegration tests that validation runs after custom setup scripts.
func TestManager_Create_ValidationIntegration(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		setupScript    string
		setupFn        SetupFunc
		validateFn     ValidateFunc
		wantErr        bool
		errContains    string
		validateCalled bool
		description    string
	}{
		"valid custom script completes successfully": {
			setupScript: "custom-setup.sh",
			setupFn: func(script, path, name, branch, repo string, w io.Writer) *SetupResult {
				return &SetupResult{Executed: true, Error: nil}
			},
			validateFn: func(worktreePath, sourceRepoPath string) (*ValidationResult, error) {
				return &ValidationResult{
					PathExists:            true,
					PathDiffersFromSource: true,
					InGitWorktreeList:     true,
					Errors:                nil,
				}, nil
			},
			wantErr:        false,
			validateCalled: true,
			description:    "Custom setup succeeds and validation passes",
		},
		"default setup (no custom script) skips validation": {
			setupScript: "", // No custom script
			setupFn: func(script, path, name, branch, repo string, w io.Writer) *SetupResult {
				return &SetupResult{Executed: false}
			},
			validateFn: func(worktreePath, sourceRepoPath string) (*ValidationResult, error) {
				// Should not be called
				t.Error("validation should not be called for default setup")
				return nil, fmt.Errorf("unexpected call")
			},
			wantErr:        false,
			validateCalled: false,
			description:    "No custom script means no validation",
		},
		"custom script fails triggers error": {
			setupScript: "failing-setup.sh",
			setupFn: func(script, path, name, branch, repo string, w io.Writer) *SetupResult {
				return &SetupResult{Executed: true, Error: fmt.Errorf("script exited with code 1")}
			},
			validateFn: func(worktreePath, sourceRepoPath string) (*ValidationResult, error) {
				// Should not be called - script failed
				t.Error("validation should not be called when script fails")
				return nil, fmt.Errorf("unexpected call")
			},
			wantErr:        true,
			errContains:    "script exited with code 1",
			validateCalled: false,
			description:    "Failing script returns error before validation",
		},
		"custom script breaks worktree triggers validation failure": {
			setupScript: "breaking-setup.sh",
			setupFn: func(script, path, name, branch, repo string, w io.Writer) *SetupResult {
				return &SetupResult{Executed: true, Error: nil}
			},
			validateFn: func(worktreePath, sourceRepoPath string) (*ValidationResult, error) {
				return &ValidationResult{
					PathExists:            false,
					PathDiffersFromSource: true,
					InGitWorktreeList:     false,
					Errors: []string{
						"worktree path does not exist: " + worktreePath,
						"worktree not found in git worktree list",
					},
				}, nil
			},
			wantErr:        true,
			errContains:    "validation failed",
			validateCalled: true,
			description:    "Custom script that breaks worktree triggers validation failure",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			stateDir := t.TempDir()
			repoRoot := t.TempDir()
			baseDir := t.TempDir()

			mockOps := &mockGitOps{}
			var buf bytes.Buffer

			cfg := &WorktreeConfig{
				BaseDir:     baseDir,
				Prefix:      "wt-",
				AutoSetup:   true, // Enable auto setup to trigger validation path
				SetupScript: tt.setupScript,
				TrackStatus: true,
				CopyDirs:    []string{},
			}

			manager := NewManager(cfg, stateDir, repoRoot,
				WithStdout(&buf),
				WithGitOps(mockOps),
				WithCopyFunc(func(src, dst string, dirs []string) ([]string, error) {
					return nil, nil
				}),
				WithSetupFunc(tt.setupFn),
				WithValidateFunc(tt.validateFn),
			)

			wt, err := manager.Create("test", "test-branch", "")

			if tt.wantErr {
				require.Error(t, err, tt.description)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains, tt.description)
				}
				assert.Nil(t, wt)
			} else {
				require.NoError(t, err, tt.description)
				assert.NotNil(t, wt)
				assert.Equal(t, "test", wt.Name)
			}
		})
	}
}

// TestManager_Create_ValidationErrorMessages tests that validation errors are clear and actionable.
func TestManager_Create_ValidationErrorMessages(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		setupFn        SetupFunc
		validateFn     ValidateFunc
		errContains    []string
		stdoutContains []string
		description    string
	}{
		"worktree not in git list triggers actionable error": {
			setupFn: func(script, path, name, branch, repo string, w io.Writer) *SetupResult {
				return &SetupResult{Executed: true, Error: nil}
			},
			validateFn: func(worktreePath, sourceRepoPath string) (*ValidationResult, error) {
				return &ValidationResult{
					PathExists:            true,
					PathDiffersFromSource: true,
					InGitWorktreeList:     false,
					Errors: []string{
						"worktree not found in git worktree list: " + worktreePath + " (run 'git worktree list' to verify)",
					},
				}, nil
			},
			errContains:    []string{"validation failed", "worktree not found in git worktree list"},
			stdoutContains: []string{"validation failed", "git worktree list"},
			description:    "Missing from git worktree list should have actionable message",
		},
		"path does not exist has suggestion": {
			setupFn: func(script, path, name, branch, repo string, w io.Writer) *SetupResult {
				return &SetupResult{Executed: true, Error: nil}
			},
			validateFn: func(worktreePath, sourceRepoPath string) (*ValidationResult, error) {
				return &ValidationResult{
					PathExists:            false,
					PathDiffersFromSource: true,
					InGitWorktreeList:     true,
					Errors: []string{
						"worktree path does not exist: " + worktreePath + " (ensure setup script creates the directory)",
					},
				}, nil
			},
			errContains:    []string{"validation failed", "does not exist"},
			stdoutContains: []string{"validation failed", "ensure setup script"},
			description:    "Path does not exist should suggest checking setup script",
		},
		"path same as source has suggestion": {
			setupFn: func(script, path, name, branch, repo string, w io.Writer) *SetupResult {
				return &SetupResult{Executed: true, Error: nil}
			},
			validateFn: func(worktreePath, sourceRepoPath string) (*ValidationResult, error) {
				return &ValidationResult{
					PathExists:            true,
					PathDiffersFromSource: false,
					InGitWorktreeList:     true,
					Errors: []string{
						"worktree path same as source repo: " + worktreePath + " (setup script may have changed directory)",
					},
				}, nil
			},
			errContains:    []string{"validation failed", "same as source"},
			stdoutContains: []string{"validation failed", "changed directory"},
			description:    "Path same as source should suggest script cd'd back",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			stateDir := t.TempDir()
			repoRoot := t.TempDir()
			baseDir := t.TempDir()

			mockOps := &mockGitOps{}
			var buf bytes.Buffer

			cfg := &WorktreeConfig{
				BaseDir:     baseDir,
				Prefix:      "wt-",
				AutoSetup:   true,
				SetupScript: "custom-setup.sh",
				TrackStatus: true,
				CopyDirs:    []string{},
			}

			manager := NewManager(cfg, stateDir, repoRoot,
				WithStdout(&buf),
				WithGitOps(mockOps),
				WithCopyFunc(func(src, dst string, dirs []string) ([]string, error) {
					return nil, nil
				}),
				WithSetupFunc(tt.setupFn),
				WithValidateFunc(tt.validateFn),
			)

			_, err := manager.Create("test", "test-branch", "")

			require.Error(t, err, tt.description)
			for _, s := range tt.errContains {
				assert.Contains(t, err.Error(), s, tt.description)
			}

			// Also check stdout for user-facing validation messages
			output := buf.String()
			for _, s := range tt.stdoutContains {
				assert.Contains(t, output, s, "Should print validation errors to stdout: "+tt.description)
			}
		})
	}
}
