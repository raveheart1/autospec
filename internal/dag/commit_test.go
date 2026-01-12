package dag

import (
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestExpandTemplateVars(t *testing.T) {
	tests := map[string]struct {
		template string
		vars     TemplateVars
		expected string
		wantErr  bool
	}{
		"empty template": {
			template: "",
			vars:     TemplateVars{},
			expected: "",
			wantErr:  false,
		},
		"no variables": {
			template: "git add . && git commit -m 'auto'",
			vars:     TemplateVars{},
			expected: "git add . && git commit -m 'auto'",
			wantErr:  false,
		},
		"spec_id variable": {
			template: "autospec run {{.SpecID}}",
			vars:     TemplateVars{SpecID: "001-auth"},
			expected: "autospec run 001-auth",
			wantErr:  false,
		},
		"worktree variable": {
			template: "cd {{.Worktree}} && git status",
			vars:     TemplateVars{Worktree: "/tmp/worktree/001-auth"},
			expected: "cd /tmp/worktree/001-auth && git status",
			wantErr:  false,
		},
		"branch variable": {
			template: "git checkout {{.Branch}}",
			vars:     TemplateVars{Branch: "dag/my-dag/001-auth"},
			expected: "git checkout dag/my-dag/001-auth",
			wantErr:  false,
		},
		"base_branch variable": {
			template: "git rebase {{.BaseBranch}}",
			vars:     TemplateVars{BaseBranch: "main"},
			expected: "git rebase main",
			wantErr:  false,
		},
		"dag_id variable": {
			template: "echo 'DAG: {{.DagID}}'",
			vars:     TemplateVars{DagID: "my-workflow"},
			expected: "echo 'DAG: my-workflow'",
			wantErr:  false,
		},
		"all variables": {
			template: "cd {{.Worktree}} && git add . && git commit -m 'feat({{.SpecID}}): auto-commit for {{.DagID}} on {{.Branch}} targeting {{.BaseBranch}}'",
			vars: TemplateVars{
				SpecID:     "001-auth",
				Worktree:   "/tmp/wt",
				Branch:     "feature",
				BaseBranch: "main",
				DagID:      "my-dag",
			},
			expected: "cd /tmp/wt && git add . && git commit -m 'feat(001-auth): auto-commit for my-dag on feature targeting main'",
			wantErr:  false,
		},
		"invalid template syntax": {
			template: "git {{.Invalid",
			vars:     TemplateVars{},
			expected: "",
			wantErr:  true,
		},
		"undefined variable": {
			template: "git {{.UndefinedVar}}",
			vars:     TemplateVars{},
			expected: "",
			wantErr:  true, // text/template returns error for undefined fields
		},
		"special characters in values": {
			template: "git commit -m '{{.SpecID}}'",
			vars:     TemplateVars{SpecID: "feat: 'test' & \"more\""},
			expected: "git commit -m 'feat: 'test' & \"more\"'",
			wantErr:  false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result, err := ExpandTemplateVars(tt.template, tt.vars)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("got %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestExpandAutocommitCmd(t *testing.T) {
	tests := map[string]struct {
		template   string
		specID     string
		worktree   string
		branch     string
		baseBranch string
		dagID      string
		expected   string
		wantErr    bool
	}{
		"simple expansion": {
			template:   "autospec run {{.SpecID}} -d {{.Worktree}}",
			specID:     "001-auth",
			worktree:   "/tmp/wt",
			branch:     "feature",
			baseBranch: "main",
			dagID:      "my-dag",
			expected:   "autospec run 001-auth -d /tmp/wt",
			wantErr:    false,
		},
		"empty template": {
			template:   "",
			specID:     "test",
			worktree:   "/tmp",
			branch:     "br",
			baseBranch: "main",
			dagID:      "dag",
			expected:   "",
			wantErr:    false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result, err := ExpandAutocommitCmd(
				tt.template,
				tt.specID,
				tt.worktree,
				tt.branch,
				tt.baseBranch,
				tt.dagID,
			)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("got %q, want %q", result, tt.expected)
			}
		})
	}
}

// commitMockRunner is a mock implementation of CommandRunner for commit testing.
type commitMockRunner struct {
	// runFunc is called when Run is invoked.
	runFunc func(ctx context.Context, dir string, stdout, stderr io.Writer, name string, args ...string) (int, error)
}

func (m *commitMockRunner) Run(ctx context.Context, dir string, stdout, stderr io.Writer, name string, args ...string) (int, error) {
	if m.runFunc != nil {
		return m.runFunc(ctx, dir, stdout, stderr, name, args...)
	}
	return 0, nil
}

// setupCommitTestRepo creates a temp git repo with an initial commit.
func setupCommitTestRepo(t *testing.T) (string, func()) {
	t.Helper()

	dir, err := os.MkdirTemp("", "commit-test-*")
	if err != nil {
		t.Fatalf("creating temp dir: %v", err)
	}

	cleanup := func() {
		os.RemoveAll(dir)
	}

	// Initialize git repo
	if err := runGitCmd(dir, "init"); err != nil {
		cleanup()
		t.Fatalf("git init: %v", err)
	}

	// Configure git user
	if err := runGitCmd(dir, "config", "user.email", "test@example.com"); err != nil {
		cleanup()
		t.Fatalf("git config email: %v", err)
	}
	if err := runGitCmd(dir, "config", "user.name", "Test User"); err != nil {
		cleanup()
		t.Fatalf("git config name: %v", err)
	}

	// Create initial commit
	if err := os.WriteFile(filepath.Join(dir, "initial.txt"), []byte("initial"), 0644); err != nil {
		cleanup()
		t.Fatalf("creating file: %v", err)
	}
	if err := runGitCmd(dir, "add", "initial.txt"); err != nil {
		cleanup()
		t.Fatalf("git add: %v", err)
	}
	if err := runGitCmd(dir, "commit", "-m", "initial commit"); err != nil {
		cleanup()
		t.Fatalf("git commit: %v", err)
	}

	// Create target branch for comparison
	if err := runGitCmd(dir, "branch", "main"); err != nil {
		cleanup()
		t.Fatalf("git branch: %v", err)
	}

	return dir, cleanup
}

// runGitCmd executes a git command in the given directory.
func runGitCmd(dir string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	return cmd.Run()
}

func TestNewCommitVerifier(t *testing.T) {
	autocommit := true
	cfg := &DAGExecutionConfig{
		Autocommit:        &autocommit,
		AutocommitRetries: 3,
	}
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	runner := &commitMockRunner{}

	cv := NewCommitVerifier(cfg, stdout, stderr, runner)

	if cv == nil {
		t.Fatal("expected non-nil CommitVerifier")
	}
	if cv.config != cfg {
		t.Error("config not set correctly")
	}
	if cv.stdout != stdout {
		t.Error("stdout not set correctly")
	}
	if cv.stderr != stderr {
		t.Error("stderr not set correctly")
	}
	if cv.cmdRunner != runner {
		t.Error("cmdRunner not set correctly")
	}
}

func TestPostExecutionCommitFlow(t *testing.T) {
	tests := map[string]struct {
		setup          func(t *testing.T, repo string)
		autocommit     bool
		retries        int
		customCmd      string
		mockRunner     func(commitMade *bool) *commitMockRunner
		expectedStatus CommitStatus
		expectError    bool
	}{
		"no uncommitted changes with commits ahead": {
			setup: func(t *testing.T, repo string) {
				// Add a new commit ahead of main
				if err := os.WriteFile(filepath.Join(repo, "new.txt"), []byte("new"), 0644); err != nil {
					t.Fatal(err)
				}
				runGitCmd(repo, "add", "new.txt")
				runGitCmd(repo, "commit", "-m", "new commit")
			},
			autocommit:     true,
			retries:        1,
			mockRunner:     nil,
			expectedStatus: CommitStatusCommitted,
			expectError:    false,
		},
		"no uncommitted changes with no commits ahead": {
			setup:          func(t *testing.T, repo string) {},
			autocommit:     true,
			retries:        1,
			mockRunner:     nil,
			expectedStatus: CommitStatusPending,
			expectError:    true, // error: no commits ahead
		},
		"uncommitted changes autocommit disabled": {
			setup: func(t *testing.T, repo string) {
				if err := os.WriteFile(filepath.Join(repo, "uncommitted.txt"), []byte("uncommitted"), 0644); err != nil {
					t.Fatal(err)
				}
			},
			autocommit:     false,
			retries:        1,
			mockRunner:     nil,
			expectedStatus: CommitStatusPending,
			expectError:    false,
		},
		"uncommitted changes custom command succeeds": {
			setup: func(t *testing.T, repo string) {
				if err := os.WriteFile(filepath.Join(repo, "uncommitted.txt"), []byte("uncommitted"), 0644); err != nil {
					t.Fatal(err)
				}
			},
			autocommit: true,
			retries:    1,
			customCmd:  "git add . && git commit -m 'auto'",
			mockRunner: func(commitMade *bool) *commitMockRunner {
				return &commitMockRunner{
					runFunc: func(ctx context.Context, dir string, stdout, stderr io.Writer, name string, args ...string) (int, error) {
						// Simulate the commit being made
						runGitCmd(dir, "add", ".")
						runGitCmd(dir, "commit", "-m", "auto commit")
						*commitMade = true
						return 0, nil
					},
				}
			},
			expectedStatus: CommitStatusCommitted,
			expectError:    false,
		},
		"uncommitted changes custom command fails all retries": {
			setup: func(t *testing.T, repo string) {
				if err := os.WriteFile(filepath.Join(repo, "uncommitted.txt"), []byte("uncommitted"), 0644); err != nil {
					t.Fatal(err)
				}
			},
			autocommit: true,
			retries:    2,
			customCmd:  "false", // command that always fails
			mockRunner: func(commitMade *bool) *commitMockRunner {
				return &commitMockRunner{
					runFunc: func(ctx context.Context, dir string, stdout, stderr io.Writer, name string, args ...string) (int, error) {
						// Simulate command failure
						return 1, nil
					},
				}
			},
			expectedStatus: CommitStatusFailed,
			expectError:    true,
		},
		"retry succeeds on second attempt": {
			setup: func(t *testing.T, repo string) {
				if err := os.WriteFile(filepath.Join(repo, "uncommitted.txt"), []byte("uncommitted"), 0644); err != nil {
					t.Fatal(err)
				}
			},
			autocommit: true,
			retries:    3,
			customCmd:  "git add . && git commit -m 'auto'",
			mockRunner: func(commitMade *bool) *commitMockRunner {
				attemptCount := 0
				return &commitMockRunner{
					runFunc: func(ctx context.Context, dir string, stdout, stderr io.Writer, name string, args ...string) (int, error) {
						attemptCount++
						if attemptCount < 2 {
							// Fail first attempt
							return 1, nil
						}
						// Succeed on second attempt
						runGitCmd(dir, "add", ".")
						runGitCmd(dir, "commit", "-m", "auto commit")
						*commitMade = true
						return 0, nil
					},
				}
			},
			expectedStatus: CommitStatusCommitted,
			expectError:    false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			repo, cleanup := setupCommitTestRepo(t)
			defer cleanup()

			tt.setup(t, repo)

			cfg := &DAGExecutionConfig{
				Autocommit:        &tt.autocommit,
				AutocommitRetries: tt.retries,
				AutocommitCmd:     tt.customCmd,
				BaseBranch:        "main",
			}

			stdout := &bytes.Buffer{}
			stderr := &bytes.Buffer{}

			var runner CommandRunner
			commitMade := false
			if tt.mockRunner != nil {
				runner = tt.mockRunner(&commitMade)
			} else {
				runner = &mockCommandRunner{}
			}

			cv := NewCommitVerifier(cfg, stdout, stderr, runner)

			result := cv.PostExecutionCommitFlow(
				context.Background(),
				"test-spec",
				repo,
				"feature-branch",
				"main",
				"test-dag",
			)

			if result.Status != tt.expectedStatus {
				t.Errorf("status: got %v, want %v", result.Status, tt.expectedStatus)
			}

			if tt.expectError && result.Error == nil {
				t.Error("expected error, got nil")
			}

			if !tt.expectError && result.Error != nil {
				t.Errorf("unexpected error: %v", result.Error)
			}

			if result.Status == CommitStatusCommitted && result.CommitSHA == "" {
				t.Error("expected commit SHA for committed status")
			}
		})
	}
}

func TestRunCustomCommitCmd(t *testing.T) {
	tests := map[string]struct {
		customCmd     string
		mockExitCode  int
		expectError   bool
		errorContains string
	}{
		"successful command": {
			customCmd:    "echo '{{.SpecID}}'",
			mockExitCode: 0,
			expectError:  false,
		},
		"failed command": {
			customCmd:     "false",
			mockExitCode:  1,
			expectError:   true,
			errorContains: "exit code 1",
		},
		"template expansion": {
			customCmd:    "echo '{{.SpecID}} {{.Worktree}}'",
			mockExitCode: 0,
			expectError:  false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			cfg := &DAGExecutionConfig{
				AutocommitCmd: tt.customCmd,
			}

			stdout := &bytes.Buffer{}
			stderr := &bytes.Buffer{}

			runner := &commitMockRunner{
				runFunc: func(ctx context.Context, dir string, stdout, stderr io.Writer, name string, args ...string) (int, error) {
					return tt.mockExitCode, nil
				},
			}

			cv := NewCommitVerifier(cfg, stdout, stderr, runner)

			err := cv.RunCustomCommitCmd(
				context.Background(),
				"test-spec",
				"/tmp/worktree",
				"feature",
				"main",
				"test-dag",
			)

			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tt.errorContains != "" && !commitContainsString(err.Error(), tt.errorContains) {
					t.Errorf("error %q should contain %q", err.Error(), tt.errorContains)
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestRunAgentCommitSession(t *testing.T) {
	tests := map[string]struct {
		mockExitCode  int
		expectError   bool
		errorContains string
	}{
		"successful session": {
			mockExitCode: 0,
			expectError:  false,
		},
		"failed session": {
			mockExitCode:  1,
			expectError:   true,
			errorContains: "exit code 1",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			repo, cleanup := setupCommitTestRepo(t)
			defer cleanup()

			// Create an uncommitted file for git status output
			if err := os.WriteFile(filepath.Join(repo, "uncommitted.txt"), []byte("uncommitted"), 0644); err != nil {
				t.Fatal(err)
			}

			cfg := &DAGExecutionConfig{}

			stdout := &bytes.Buffer{}
			stderr := &bytes.Buffer{}

			runner := &commitMockRunner{
				runFunc: func(ctx context.Context, dir string, stdout, stderr io.Writer, name string, args ...string) (int, error) {
					return tt.mockExitCode, nil
				},
			}

			cv := NewCommitVerifier(cfg, stdout, stderr, runner)

			err := cv.RunAgentCommitSession(
				context.Background(),
				"test-spec",
				repo,
			)

			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tt.errorContains != "" && !commitContainsString(err.Error(), tt.errorContains) {
					t.Errorf("error %q should contain %q", err.Error(), tt.errorContains)
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestBuildCommitPrompt(t *testing.T) {
	tests := map[string]struct {
		specID       string
		statusOutput string
		contains     []string
	}{
		"basic prompt": {
			specID:       "001-auth",
			statusOutput: " M file.txt",
			contains: []string{
				"001-auth",
				" M file.txt",
				"git add",
				"Commit",
			},
		},
		"multiple files": {
			specID:       "002-api",
			statusOutput: " M api.go\n?? new.go",
			contains: []string{
				"002-api",
				" M api.go",
				"?? new.go",
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			cfg := &DAGExecutionConfig{}
			cv := NewCommitVerifier(cfg, nil, nil, nil)

			prompt := cv.buildCommitPrompt(tt.specID, tt.statusOutput)

			for _, expected := range tt.contains {
				if !commitContainsString(prompt, expected) {
					t.Errorf("prompt should contain %q, got: %s", expected, prompt)
				}
			}
		})
	}
}

// commitContainsString checks if s contains substr.
func commitContainsString(s, substr string) bool {
	return bytes.Contains([]byte(s), []byte(substr))
}
