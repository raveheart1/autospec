// Package testutil provides test utilities and helpers for autospec tests.
package testutil

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// GitIsolation provides git repository isolation for tests.
// It creates a temporary git repository and ensures the original
// repository state is not modified during tests.
type GitIsolation struct {
	t              *testing.T
	origDir        string
	origBranch     string
	tempDir        string
	tempRepoDir    string
	cleanedUp      bool
	branchVerified bool
}

// WithIsolatedGitRepo creates an isolated git repository for testing.
// It captures the current directory and branch, creates a temp git repo,
// and returns a cleanup function that restores the original state.
//
// Usage:
//
//	func TestSomething(t *testing.T) {
//	    cleanup := testutil.WithIsolatedGitRepo(t)
//	    defer cleanup()
//	    // Test code runs in isolated temp git repo
//	}
func WithIsolatedGitRepo(t *testing.T) func() {
	t.Helper()

	gi := &GitIsolation{t: t}
	gi.setup()

	return gi.Cleanup
}

// NewGitIsolation creates a new GitIsolation instance with more control.
// Use this when you need access to the isolation object for additional
// verifications or custom cleanup logic.
func NewGitIsolation(t *testing.T) *GitIsolation {
	t.Helper()

	gi := &GitIsolation{t: t}
	gi.setup()
	t.Cleanup(gi.Cleanup)

	return gi
}

func (gi *GitIsolation) setup() {
	gi.t.Helper()

	// Capture original directory
	origDir, err := os.Getwd()
	if err != nil {
		gi.t.Fatalf("failed to get current directory: %v", err)
	}
	gi.origDir = origDir

	// Capture original branch (if in a git repo)
	gi.origBranch = gi.getCurrentBranch()

	// Create temp directory
	tempDir, err := os.MkdirTemp("", "git-isolation-*")
	if err != nil {
		gi.t.Fatalf("failed to create temp directory: %v", err)
	}
	gi.tempDir = tempDir

	// Create temp git repo
	gi.tempRepoDir = filepath.Join(tempDir, "repo")
	if err := os.MkdirAll(gi.tempRepoDir, 0755); err != nil {
		gi.t.Fatalf("failed to create temp repo directory: %v", err)
	}

	// Initialize git repo
	gi.runGitCommand(gi.tempRepoDir, "init")
	gi.runGitCommand(gi.tempRepoDir, "config", "user.email", "test@test.com")
	gi.runGitCommand(gi.tempRepoDir, "config", "user.name", "Test")

	// Create initial commit
	dummyFile := filepath.Join(gi.tempRepoDir, "README.md")
	if err := os.WriteFile(dummyFile, []byte("# Test Repo"), 0644); err != nil {
		gi.t.Fatalf("failed to create dummy file: %v", err)
	}
	gi.runGitCommand(gi.tempRepoDir, "add", ".")
	gi.runGitCommand(gi.tempRepoDir, "commit", "-m", "Initial commit")

	// Change to temp repo
	if err := os.Chdir(gi.tempRepoDir); err != nil {
		gi.t.Fatalf("failed to change to temp repo: %v", err)
	}
}

// Cleanup restores the original directory and removes the temp directory.
// It also verifies that the original branch was not modified.
func (gi *GitIsolation) Cleanup() {
	if gi.cleanedUp {
		return
	}
	gi.cleanedUp = true

	// Change back to original directory
	if err := os.Chdir(gi.origDir); err != nil {
		gi.t.Errorf("failed to restore original directory: %v", err)
	}

	// Verify original branch is unchanged
	if gi.origBranch != "" && !gi.branchVerified {
		gi.VerifyNoBranchPollution()
	}

	// Remove temp directory
	if gi.tempDir != "" {
		if err := os.RemoveAll(gi.tempDir); err != nil {
			gi.t.Errorf("failed to remove temp directory: %v", err)
		}
	}
}

// TempRepoDir returns the path to the temporary git repository.
func (gi *GitIsolation) TempRepoDir() string {
	return gi.tempRepoDir
}

// OriginalDir returns the original working directory.
func (gi *GitIsolation) OriginalDir() string {
	return gi.origDir
}

// OriginalBranch returns the original git branch name.
func (gi *GitIsolation) OriginalBranch() string {
	return gi.origBranch
}

// CreateBranch creates a new branch in the temp repo and optionally checks it out.
func (gi *GitIsolation) CreateBranch(name string, checkout bool) {
	gi.t.Helper()

	if checkout {
		gi.runGitCommand(gi.tempRepoDir, "checkout", "-b", name)
	} else {
		gi.runGitCommand(gi.tempRepoDir, "branch", name)
	}
}

// CheckoutBranch switches to the specified branch.
func (gi *GitIsolation) CheckoutBranch(name string) {
	gi.t.Helper()
	gi.runGitCommand(gi.tempRepoDir, "checkout", name)
}

// CurrentBranch returns the current branch in the temp repo.
func (gi *GitIsolation) CurrentBranch() string {
	return gi.getCurrentBranchInDir(gi.tempRepoDir)
}

// VerifyNoBranchPollution checks that the original repository branch
// has not been modified during testing.
func (gi *GitIsolation) VerifyNoBranchPollution() {
	gi.t.Helper()
	gi.branchVerified = true

	// Change back to original dir temporarily
	currentDir, _ := os.Getwd()
	if err := os.Chdir(gi.origDir); err != nil {
		gi.t.Errorf("failed to change to original dir for verification: %v", err)
		return
	}
	defer func() {
		if err := os.Chdir(currentDir); err != nil {
			gi.t.Errorf("failed to restore directory: %v", err)
		}
	}()

	currentBranch := gi.getCurrentBranch()
	if currentBranch != gi.origBranch {
		gi.t.Errorf("branch pollution detected: expected %q, got %q", gi.origBranch, currentBranch)
	}
}

// AddFile adds a file to the temp repo with the given content.
func (gi *GitIsolation) AddFile(relativePath, content string) string {
	gi.t.Helper()

	fullPath := filepath.Join(gi.tempRepoDir, relativePath)
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		gi.t.Fatalf("failed to create directory %s: %v", dir, err)
	}

	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		gi.t.Fatalf("failed to write file %s: %v", fullPath, err)
	}

	return fullPath
}

// CommitAll stages and commits all changes in the temp repo.
func (gi *GitIsolation) CommitAll(message string) {
	gi.t.Helper()

	gi.runGitCommand(gi.tempRepoDir, "add", ".")
	gi.runGitCommand(gi.tempRepoDir, "commit", "-m", message)
}

func (gi *GitIsolation) getCurrentBranch() string {
	return gi.getCurrentBranchInDir(gi.origDir)
}

func (gi *GitIsolation) getCurrentBranchInDir(dir string) string {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		// Not in a git repo, or git not available
		return ""
	}
	return strings.TrimSpace(string(output))
}

func (gi *GitIsolation) runGitCommand(dir string, args ...string) {
	gi.t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_DATE=2025-01-01T00:00:00Z",
		"GIT_COMMITTER_DATE=2025-01-01T00:00:00Z",
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		gi.t.Fatalf("git %s failed: %v\nOutput: %s", strings.Join(args, " "), err, output)
	}
}

// SetupSpecsDir creates a specs directory structure in the temp repo.
// Returns the path to the specs directory.
func (gi *GitIsolation) SetupSpecsDir(specName string) string {
	gi.t.Helper()

	specsDir := filepath.Join(gi.tempRepoDir, "specs", specName)
	if err := os.MkdirAll(specsDir, 0755); err != nil {
		gi.t.Fatalf("failed to create specs directory: %v", err)
	}

	return specsDir
}

// WriteSpec writes a spec.yaml file to the given directory.
func (gi *GitIsolation) WriteSpec(specsDir string) string {
	gi.t.Helper()

	specContent := fmt.Sprintf(`feature:
  branch: "test-feature"
  created: "2025-01-01"
  status: "Draft"
  input: "test feature"

user_stories:
  - id: "US-001"
    title: "Test user story"
    priority: "P1"
    as_a: "developer"
    i_want: "to test"
    so_that: "tests pass"
    why_this_priority: "Testing"
    independent_test: "Run tests"
    acceptance_scenarios:
      - given: "a test"
        when: "running"
        then: "it passes"

requirements:
  functional:
    - id: "FR-001"
      description: "Test requirement"
      testable: true
      acceptance_criteria: "Test passes"
  non_functional:
    - id: "NFR-001"
      category: "code_quality"
      description: "Code quality"
      measurable_target: "High quality"

success_criteria:
  measurable_outcomes:
    - id: "SC-001"
      description: "Test passes"
      metric: "Pass rate"
      target: "100%%"

key_entities: []
edge_cases: []
assumptions: []
constraints: []
out_of_scope: []

_meta:
  version: "1.0.0"
  generator: "autospec"
  generator_version: "test"
  created: "2025-01-01T00:00:00Z"
  artifact_type: "spec"
`)

	specPath := filepath.Join(specsDir, "spec.yaml")
	if err := os.WriteFile(specPath, []byte(specContent), 0644); err != nil {
		gi.t.Fatalf("failed to write spec.yaml: %v", err)
	}

	return specPath
}
