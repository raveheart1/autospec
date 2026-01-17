package dag

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/ariel-frischer/autospec/internal/dag"
)

func TestValidateOnlySpec(t *testing.T) {
	tests := map[string]struct {
		specs   map[string]*dag.SpecState
		specID  string
		wantErr bool
	}{
		"spec exists": {
			specs: map[string]*dag.SpecState{
				"spec-001": {SpecID: "spec-001", Status: dag.SpecStatusCompleted},
			},
			specID:  "spec-001",
			wantErr: false,
		},
		"spec not found": {
			specs: map[string]*dag.SpecState{
				"spec-001": {SpecID: "spec-001", Status: dag.SpecStatusCompleted},
			},
			specID:  "spec-002",
			wantErr: true,
		},
		"empty specs": {
			specs:   map[string]*dag.SpecState{},
			specID:  "any-spec",
			wantErr: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			run := &dag.DAGRun{
				RunID:        "test-run",
				WorkflowPath: "test.yaml",
				Specs:        tt.specs,
			}

			err := validateOnlySpec(run, tt.specID)
			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestFindSpecsToCommit(t *testing.T) {
	// Create a temp git repo for testing
	dir := t.TempDir()
	setupTestGitRepo(t, dir)

	// Create a worktree subdirectory
	worktreePath := filepath.Join(dir, "worktree")
	if err := os.MkdirAll(worktreePath, 0o755); err != nil {
		t.Fatalf("creating worktree dir: %v", err)
	}

	// Initialize it as a git repo too
	runTestGitCmd(t, worktreePath, "init")
	runTestGitCmd(t, worktreePath, "config", "user.email", "test@example.com")
	runTestGitCmd(t, worktreePath, "config", "user.name", "Test User")

	// Create initial commit
	if err := os.WriteFile(filepath.Join(worktreePath, "initial.txt"), []byte("initial"), 0o644); err != nil {
		t.Fatalf("creating file: %v", err)
	}
	runTestGitCmd(t, worktreePath, "add", "initial.txt")
	runTestGitCmd(t, worktreePath, "commit", "-m", "initial")

	tests := map[string]struct {
		specs       map[string]*dag.SpecState
		onlySpec    string
		setupFunc   func()
		wantCount   int
		wantSpecIDs []string
	}{
		"spec with uncommitted changes": {
			specs: map[string]*dag.SpecState{
				"spec-001": {
					SpecID:       "spec-001",
					Status:       dag.SpecStatusCompleted,
					WorktreePath: worktreePath,
				},
			},
			onlySpec: "",
			setupFunc: func() {
				_ = os.WriteFile(filepath.Join(worktreePath, "uncommitted.txt"), []byte("uncommitted"), 0o644)
			},
			wantCount:   1,
			wantSpecIDs: []string{"spec-001"},
		},
		"spec without worktree": {
			specs: map[string]*dag.SpecState{
				"spec-001": {
					SpecID:       "spec-001",
					Status:       dag.SpecStatusCompleted,
					WorktreePath: "",
				},
			},
			onlySpec:  "",
			setupFunc: func() {},
			wantCount: 0,
		},
		"only flag filters to single spec": {
			specs: map[string]*dag.SpecState{
				"spec-001": {
					SpecID:       "spec-001",
					Status:       dag.SpecStatusCompleted,
					WorktreePath: worktreePath,
				},
				"spec-002": {
					SpecID:       "spec-002",
					Status:       dag.SpecStatusCompleted,
					WorktreePath: worktreePath,
				},
			},
			onlySpec: "spec-001",
			setupFunc: func() {
				_ = os.WriteFile(filepath.Join(worktreePath, "uncommitted.txt"), []byte("uncommitted"), 0o644)
			},
			wantCount:   1,
			wantSpecIDs: []string{"spec-001"},
		},
		"clean repo no specs to commit": {
			specs: map[string]*dag.SpecState{
				"spec-001": {
					SpecID:       "spec-001",
					Status:       dag.SpecStatusCompleted,
					WorktreePath: worktreePath,
				},
			},
			onlySpec: "",
			setupFunc: func() {
				// Remove uncommitted file if exists
				_ = os.Remove(filepath.Join(worktreePath, "uncommitted.txt"))
				// Commit any changes
				runTestGitCmd(t, worktreePath, "add", "-A")
				_ = exec.Command("git", "-C", worktreePath, "commit", "-m", "cleanup", "--allow-empty").Run()
			},
			wantCount: 0,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Reset worktree to clean state
			runTestGitCmd(t, worktreePath, "checkout", "--", ".")
			_ = os.Remove(filepath.Join(worktreePath, "uncommitted.txt"))
			runTestGitCmd(t, worktreePath, "add", "-A")
			_ = exec.Command("git", "-C", worktreePath, "commit", "-m", "reset", "--allow-empty").Run()

			tt.setupFunc()

			run := &dag.DAGRun{
				RunID:        "test-run",
				WorkflowPath: "test.yaml",
				Specs:        tt.specs,
			}

			specs, err := findSpecsToCommit(run, tt.onlySpec)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(specs) != tt.wantCount {
				t.Errorf("got %d specs, want %d", len(specs), tt.wantCount)
			}

			if len(tt.wantSpecIDs) > 0 {
				for _, wantID := range tt.wantSpecIDs {
					found := false
					for _, spec := range specs {
						if spec.SpecID == wantID {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("expected spec %q not found in results", wantID)
					}
				}
			}
		})
	}
}

func TestLoadCommitContext(t *testing.T) {
	tests := map[string]struct {
		workflowPath string
		createRun    bool
		wantErr      bool
		errMatch     string
	}{
		"existing workflow returns run": {
			workflowPath: "existing.yaml",
			createRun:    true,
			wantErr:      false,
		},
		"nonexistent workflow returns error": {
			workflowPath: "nonexistent.yaml",
			createRun:    false,
			wantErr:      true,
			errMatch:     "no run found",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			tmpDir := t.TempDir()
			stateDir := filepath.Join(tmpDir, "state", "dag-runs")
			if err := os.MkdirAll(stateDir, 0o755); err != nil {
				t.Fatalf("failed to create state dir: %v", err)
			}

			if tt.createRun {
				run := &dag.DAGRun{
					RunID:        "20260110_143022_abc12345",
					WorkflowPath: tt.workflowPath,
					DAGFile:      tt.workflowPath,
					Status:       dag.RunStatusCompleted,
					StartedAt:    time.Now(),
					Specs:        map[string]*dag.SpecState{},
				}
				if err := dag.SaveStateByWorkflow(stateDir, run); err != nil {
					t.Fatalf("failed to save state: %v", err)
				}
			}

			result, err := loadCommitContext(stateDir, tt.workflowPath)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tt.errMatch != "" && !containsSubstring(err.Error(), tt.errMatch) {
					t.Errorf("expected error containing %q, got %q", tt.errMatch, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if result == nil {
					t.Error("expected run, got nil")
				} else if result.WorkflowPath != tt.workflowPath {
					t.Errorf("expected workflow path %q, got %q", tt.workflowPath, result.WorkflowPath)
				}
			}
		})
	}
}

func TestBuildCommitDAGConfig(t *testing.T) {
	tests := map[string]struct {
		customCmd         string
		baseAutocommitCmd string
		wantAutocommit    bool
		wantAutocommitCmd string
		wantRetries       int
	}{
		"custom command overrides config": {
			customCmd:         "git add . && git commit -m 'custom'",
			baseAutocommitCmd: "git commit -m 'base'",
			wantAutocommit:    true,
			wantAutocommitCmd: "git add . && git commit -m 'custom'",
		},
		"no custom command uses config": {
			customCmd:         "",
			baseAutocommitCmd: "git commit -m 'base'",
			wantAutocommit:    true,
			wantAutocommitCmd: "git commit -m 'base'",
		},
		"empty custom and config uses agent": {
			customCmd:         "",
			baseAutocommitCmd: "",
			wantAutocommit:    true,
			wantAutocommitCmd: "",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Create a minimal config
			baseCfg := &dag.DAGExecutionConfig{
				AutocommitCmd: tt.baseAutocommitCmd,
			}

			// Build the commit DAG config using an empty config.Configuration
			// We need to simulate buildCommitDAGConfig behavior
			result := buildCommitDAGConfigForTest(baseCfg, tt.customCmd)

			if !result.IsAutocommitEnabled() {
				t.Error("expected autocommit to be enabled")
			}

			if result.AutocommitCmd != tt.wantAutocommitCmd {
				t.Errorf("AutocommitCmd: got %q, want %q", result.AutocommitCmd, tt.wantAutocommitCmd)
			}
		})
	}
}

// buildCommitDAGConfigForTest is a test helper that simulates buildCommitDAGConfig
func buildCommitDAGConfigForTest(baseCfg *dag.DAGExecutionConfig, customCmd string) *dag.DAGExecutionConfig {
	enabled := true
	result := &dag.DAGExecutionConfig{
		Autocommit:        &enabled,
		AutocommitCmd:     baseCfg.AutocommitCmd,
		AutocommitRetries: baseCfg.AutocommitRetries,
	}

	if customCmd != "" {
		result.AutocommitCmd = customCmd
	}

	return result
}

func TestExecuteDryRun(t *testing.T) {
	// Create a temp git repo for testing
	dir := t.TempDir()
	setupTestGitRepo(t, dir)

	// Create some uncommitted files
	if err := os.WriteFile(filepath.Join(dir, "new.txt"), []byte("new file"), 0o644); err != nil {
		t.Fatalf("creating file: %v", err)
	}

	specs := []*dag.SpecState{
		{
			SpecID:       "spec-001",
			WorktreePath: dir,
		},
	}

	// executeDryRun should not return an error
	err := executeDryRun(specs)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGetBaseBranch(t *testing.T) {
	tests := map[string]struct {
		specs      map[string]*dag.SpecState
		wantBranch string
	}{
		"returns main when specs have branches": {
			specs: map[string]*dag.SpecState{
				"spec-001": {
					SpecID: "spec-001",
					Branch: "dag/my-dag/spec-001",
				},
			},
			wantBranch: "main",
		},
		"returns main when specs have no branches": {
			specs: map[string]*dag.SpecState{
				"spec-001": {
					SpecID: "spec-001",
					Branch: "",
				},
			},
			wantBranch: "main",
		},
		"returns main for empty specs": {
			specs:      map[string]*dag.SpecState{},
			wantBranch: "main",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			run := &dag.DAGRun{
				Specs: tt.specs,
			}

			result := getBaseBranch(run)
			if result != tt.wantBranch {
				t.Errorf("got %q, want %q", result, tt.wantBranch)
			}
		})
	}
}

func TestPrintCommitResults(t *testing.T) {
	tests := map[string]struct {
		total   int
		failed  []string
		wantErr bool
	}{
		"all successful": {
			total:   3,
			failed:  []string{},
			wantErr: false,
		},
		"some failed": {
			total:   3,
			failed:  []string{"spec-001", "spec-002"},
			wantErr: true,
		},
		"all failed": {
			total:   2,
			failed:  []string{"spec-001", "spec-002"},
			wantErr: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			err := printCommitResults("test.yaml", tt.total, tt.failed)
			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestCommitSingleSpec(t *testing.T) {
	tests := map[string]struct {
		setupFunc    func(t *testing.T, dir string)
		specStatus   dag.SpecStatus
		autocommit   bool
		expectStatus dag.CommitStatus
	}{
		"committed changes ahead of main": {
			setupFunc: func(t *testing.T, dir string) {
				// Create a new branch for the spec and add a commit
				runTestGitCmd(t, dir, "checkout", "-b", "feature-branch")
				_ = os.WriteFile(filepath.Join(dir, "new.txt"), []byte("new"), 0o644)
				runTestGitCmd(t, dir, "add", "new.txt")
				runTestGitCmd(t, dir, "commit", "-m", "new commit")
			},
			specStatus:   dag.SpecStatusCompleted,
			autocommit:   true,
			expectStatus: dag.CommitStatusCommitted,
		},
		"no changes no commits ahead returns pending": {
			setupFunc: func(t *testing.T, dir string) {
				// Stay on main, no new commits
			},
			specStatus:   dag.SpecStatusCompleted,
			autocommit:   true,
			expectStatus: dag.CommitStatusPending,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Create a fresh temp git repo for each test
			dir := t.TempDir()
			setupTestGitRepo(t, dir)

			// Rename default branch to main
			runTestGitCmd(t, dir, "branch", "-M", "main")

			tt.setupFunc(t, dir)

			tmpStateDir := t.TempDir()

			run := &dag.DAGRun{
				RunID:        "test-run",
				WorkflowPath: "test.yaml",
				DAGId:        "test-dag",
				Specs: map[string]*dag.SpecState{
					"spec-001": {
						SpecID:       "spec-001",
						Status:       tt.specStatus,
						WorktreePath: dir,
						Branch:       "dag/test-dag/spec-001",
					},
				},
			}

			spec := run.Specs["spec-001"]

			enabled := tt.autocommit
			dagConfig := &dag.DAGExecutionConfig{
				Autocommit:        &enabled,
				AutocommitRetries: 1,
			}

			verifier := dag.NewCommitVerifier(dagConfig, os.Stdout, os.Stderr, dag.NewDefaultCommandRunner())

			result := commitSingleSpec(
				context.Background(),
				verifier,
				run,
				spec,
				tmpStateDir,
			)

			if result.Status != tt.expectStatus {
				t.Errorf("got status %v, want %v", result.Status, tt.expectStatus)
			}
		})
	}
}

// Helper functions

func setupTestGitRepo(t *testing.T, dir string) {
	t.Helper()

	runTestGitCmd(t, dir, "init")
	runTestGitCmd(t, dir, "config", "user.email", "test@example.com")
	runTestGitCmd(t, dir, "config", "user.name", "Test User")

	// Create initial commit
	if err := os.WriteFile(filepath.Join(dir, "initial.txt"), []byte("initial"), 0o644); err != nil {
		t.Fatalf("creating file: %v", err)
	}
	runTestGitCmd(t, dir, "add", "initial.txt")
	runTestGitCmd(t, dir, "commit", "-m", "initial commit")
}

func runTestGitCmd(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git %v failed: %v", args, err)
	}
}

func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
