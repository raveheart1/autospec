package dag

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// setupTestRepo creates a temporary git repository for testing.
// Returns the path to the repo and a cleanup function.
func setupTestRepo(t *testing.T) (string, func()) {
	t.Helper()

	dir, err := os.MkdirTemp("", "githelpers-test-*")
	if err != nil {
		t.Fatalf("creating temp dir: %v", err)
	}

	cleanup := func() {
		os.RemoveAll(dir)
	}

	// Initialize git repo
	if err := runGit(dir, "init"); err != nil {
		cleanup()
		t.Fatalf("git init: %v", err)
	}

	// Configure git user for commits
	if err := runGit(dir, "config", "user.email", "test@example.com"); err != nil {
		cleanup()
		t.Fatalf("git config email: %v", err)
	}
	if err := runGit(dir, "config", "user.name", "Test User"); err != nil {
		cleanup()
		t.Fatalf("git config name: %v", err)
	}

	return dir, cleanup
}

// runGit executes a git command in the given directory.
func runGit(dir string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	return cmd.Run()
}

// createFile creates a file with the given content in the repo.
func createFile(t *testing.T, repoPath, filename, content string) {
	t.Helper()
	path := filepath.Join(repoPath, filename)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("creating file %s: %v", filename, err)
	}
}

func TestHasUncommittedChanges(t *testing.T) {
	tests := map[string]struct {
		setup    func(t *testing.T, repo string)
		expected bool
	}{
		"clean repo after commit": {
			setup: func(t *testing.T, repo string) {
				createFile(t, repo, "file.txt", "content")
				runGit(repo, "add", "file.txt")
				runGit(repo, "commit", "-m", "initial")
			},
			expected: false,
		},
		"modified tracked file": {
			setup: func(t *testing.T, repo string) {
				createFile(t, repo, "file.txt", "content")
				runGit(repo, "add", "file.txt")
				runGit(repo, "commit", "-m", "initial")
				createFile(t, repo, "file.txt", "modified content")
			},
			expected: true,
		},
		"untracked file": {
			setup: func(t *testing.T, repo string) {
				createFile(t, repo, "file.txt", "content")
				runGit(repo, "add", "file.txt")
				runGit(repo, "commit", "-m", "initial")
				createFile(t, repo, "untracked.txt", "untracked")
			},
			expected: true,
		},
		"staged changes": {
			setup: func(t *testing.T, repo string) {
				createFile(t, repo, "file.txt", "content")
				runGit(repo, "add", "file.txt")
				runGit(repo, "commit", "-m", "initial")
				createFile(t, repo, "new.txt", "new content")
				runGit(repo, "add", "new.txt")
			},
			expected: true,
		},
		"deleted file": {
			setup: func(t *testing.T, repo string) {
				createFile(t, repo, "file.txt", "content")
				runGit(repo, "add", "file.txt")
				runGit(repo, "commit", "-m", "initial")
				os.Remove(filepath.Join(repo, "file.txt"))
			},
			expected: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			repo, cleanup := setupTestRepo(t)
			defer cleanup()

			tt.setup(t, repo)

			result, err := HasUncommittedChanges(repo)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("got %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGetUncommittedFiles(t *testing.T) {
	tests := map[string]struct {
		setup         func(t *testing.T, repo string)
		expectedCount int
		containsFile  string
	}{
		"clean repo": {
			setup: func(t *testing.T, repo string) {
				createFile(t, repo, "file.txt", "content")
				runGit(repo, "add", "file.txt")
				runGit(repo, "commit", "-m", "initial")
			},
			expectedCount: 0,
			containsFile:  "",
		},
		"modified file": {
			setup: func(t *testing.T, repo string) {
				createFile(t, repo, "file.txt", "content")
				runGit(repo, "add", "file.txt")
				runGit(repo, "commit", "-m", "initial")
				createFile(t, repo, "file.txt", "modified")
			},
			expectedCount: 1,
			containsFile:  "file.txt",
		},
		"multiple files": {
			setup: func(t *testing.T, repo string) {
				createFile(t, repo, "file.txt", "content")
				runGit(repo, "add", "file.txt")
				runGit(repo, "commit", "-m", "initial")
				createFile(t, repo, "file.txt", "modified")
				createFile(t, repo, "new.txt", "new content")
			},
			expectedCount: 2,
			containsFile:  "file.txt",
		},
		"untracked file": {
			setup: func(t *testing.T, repo string) {
				createFile(t, repo, "file.txt", "content")
				runGit(repo, "add", "file.txt")
				runGit(repo, "commit", "-m", "initial")
				createFile(t, repo, "untracked.txt", "untracked")
			},
			expectedCount: 1,
			containsFile:  "untracked.txt",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			repo, cleanup := setupTestRepo(t)
			defer cleanup()

			tt.setup(t, repo)

			files, err := GetUncommittedFiles(repo)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(files) != tt.expectedCount {
				t.Errorf("file count: got %d, want %d", len(files), tt.expectedCount)
			}
			if tt.containsFile != "" && !stringSliceContains(files, tt.containsFile) {
				t.Errorf("expected files to contain %q, got %v", tt.containsFile, files)
			}
		})
	}
}

func TestGetCommitsAhead(t *testing.T) {
	tests := map[string]struct {
		setup        func(t *testing.T, repo string) string
		targetBranch string
		expected     int
		wantErr      bool
	}{
		"no commits ahead": {
			setup: func(t *testing.T, repo string) string {
				createFile(t, repo, "file.txt", "content")
				runGit(repo, "add", "file.txt")
				runGit(repo, "commit", "-m", "initial")
				runGit(repo, "branch", "target")
				return "target"
			},
			expected: 0,
			wantErr:  false,
		},
		"one commit ahead": {
			setup: func(t *testing.T, repo string) string {
				createFile(t, repo, "file.txt", "content")
				runGit(repo, "add", "file.txt")
				runGit(repo, "commit", "-m", "initial")
				runGit(repo, "branch", "target")
				createFile(t, repo, "new.txt", "new content")
				runGit(repo, "add", "new.txt")
				runGit(repo, "commit", "-m", "second commit")
				return "target"
			},
			expected: 1,
			wantErr:  false,
		},
		"multiple commits ahead": {
			setup: func(t *testing.T, repo string) string {
				createFile(t, repo, "file.txt", "content")
				runGit(repo, "add", "file.txt")
				runGit(repo, "commit", "-m", "initial")
				runGit(repo, "branch", "target")
				createFile(t, repo, "a.txt", "a")
				runGit(repo, "add", "a.txt")
				runGit(repo, "commit", "-m", "commit a")
				createFile(t, repo, "b.txt", "b")
				runGit(repo, "add", "b.txt")
				runGit(repo, "commit", "-m", "commit b")
				createFile(t, repo, "c.txt", "c")
				runGit(repo, "add", "c.txt")
				runGit(repo, "commit", "-m", "commit c")
				return "target"
			},
			expected: 3,
			wantErr:  false,
		},
		"empty target branch": {
			setup: func(t *testing.T, repo string) string {
				createFile(t, repo, "file.txt", "content")
				runGit(repo, "add", "file.txt")
				runGit(repo, "commit", "-m", "initial")
				return ""
			},
			expected: 0,
			wantErr:  true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			repo, cleanup := setupTestRepo(t)
			defer cleanup()

			targetBranch := tt.setup(t, repo)
			if tt.targetBranch != "" {
				targetBranch = tt.targetBranch
			}

			count, err := GetCommitsAhead(repo, targetBranch)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if count != tt.expected {
				t.Errorf("got %d commits ahead, want %d", count, tt.expected)
			}
		})
	}
}

func TestGetCommitSHA(t *testing.T) {
	tests := map[string]struct {
		setup   func(t *testing.T, repo string)
		wantErr bool
	}{
		"valid commit": {
			setup: func(t *testing.T, repo string) {
				createFile(t, repo, "file.txt", "content")
				runGit(repo, "add", "file.txt")
				runGit(repo, "commit", "-m", "initial")
			},
			wantErr: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			repo, cleanup := setupTestRepo(t)
			defer cleanup()

			tt.setup(t, repo)

			sha, err := GetCommitSHA(repo)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			// SHA should be 40 hex characters
			if len(sha) != 40 {
				t.Errorf("SHA length: got %d, want 40", len(sha))
			}
		})
	}
}

func TestGetCurrentBranch(t *testing.T) {
	tests := map[string]struct {
		setup    func(t *testing.T, repo string)
		expected string
		wantErr  bool
	}{
		"default branch": {
			setup: func(t *testing.T, repo string) {
				createFile(t, repo, "file.txt", "content")
				runGit(repo, "add", "file.txt")
				runGit(repo, "commit", "-m", "initial")
			},
			expected: "",
			wantErr:  false,
		},
		"named branch": {
			setup: func(t *testing.T, repo string) {
				createFile(t, repo, "file.txt", "content")
				runGit(repo, "add", "file.txt")
				runGit(repo, "commit", "-m", "initial")
				runGit(repo, "checkout", "-b", "feature-branch")
			},
			expected: "feature-branch",
			wantErr:  false,
		},
		"detached HEAD": {
			setup: func(t *testing.T, repo string) {
				createFile(t, repo, "file.txt", "content")
				runGit(repo, "add", "file.txt")
				runGit(repo, "commit", "-m", "initial")
				runGit(repo, "checkout", "--detach")
			},
			expected: "",
			wantErr:  true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			repo, cleanup := setupTestRepo(t)
			defer cleanup()

			tt.setup(t, repo)

			branch, err := GetCurrentBranch(repo)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			// For default branch test, we just check it's not empty
			if tt.expected != "" && branch != tt.expected {
				t.Errorf("got %q, want %q", branch, tt.expected)
			}
			if tt.expected == "" && branch == "" {
				t.Error("expected non-empty branch name")
			}
		})
	}
}

func TestParseStatusOutput(t *testing.T) {
	// git status --porcelain format:
	// XY filename
	// where X = index status, Y = worktree status
	// Position 0-1: status codes, Position 2: space, Position 3+: filename
	tests := map[string]struct {
		input    string
		expected []string
	}{
		"empty output": {
			input:    "",
			expected: nil,
		},
		"single modified file": {
			// " M " = modified in worktree (space at 0, M at 1, space at 2)
			input:    " M file.txt\n",
			expected: []string{"file.txt"},
		},
		"staged file": {
			// "A  " = added to index
			input:    "A  new.txt\n",
			expected: []string{"new.txt"},
		},
		"untracked file": {
			// "?? " = untracked
			input:    "?? untracked.txt\n",
			expected: []string{"untracked.txt"},
		},
		"renamed file": {
			// "R  " = renamed, format: "R  old -> new"
			input:    "R  old.txt -> new.txt\n",
			expected: []string{"new.txt"},
		},
		"multiple files": {
			input:    " M modified.txt\nA  added.txt\n?? untracked.txt\n",
			expected: []string{"modified.txt", "added.txt", "untracked.txt"},
		},
		"deleted file": {
			// " D " = deleted in worktree
			input:    " D deleted.txt\n",
			expected: []string{"deleted.txt"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := parseStatusOutput([]byte(tt.input))
			if len(result) != len(tt.expected) {
				t.Errorf("length: got %d, want %d", len(result), len(tt.expected))
				return
			}
			for i, file := range tt.expected {
				if result[i] != file {
					t.Errorf("file[%d]: got %q, want %q", i, result[i], file)
				}
			}
		})
	}
}

// stringSliceContains checks if a slice contains a string.
func stringSliceContains(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}
