// Package testutil provides test utilities and helpers for autospec tests.
package testutil

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

var (
	// autospecBinaryPath caches the built autospec binary path.
	autospecBinaryPath string
	autospecBuildOnce  sync.Once
	autospecBuildErr   error
)

// E2EEnv provides an isolated environment for E2E testing.
// It manages PATH isolation, temp directories, and environment sanitization
// to ensure E2E tests never invoke the real Claude CLI.
type E2EEnv struct {
	t               *testing.T
	tempDir         string
	binDir          string
	specsDir        string
	originalEnv     map[string]string
	cleanedUp       bool
	mockExitCode    int
	mockExitCodeSet bool
}

// CommandResult captures the result of running an autospec command.
type CommandResult struct {
	ExitCode int
	Stdout   string
	Stderr   string
	Duration time.Duration
}

// NewE2EEnv creates a new E2E test environment with PATH isolation.
// The mock claude binary will be the only "claude" in PATH.
func NewE2EEnv(t *testing.T) *E2EEnv {
	t.Helper()

	env := &E2EEnv{
		t:           t,
		originalEnv: make(map[string]string),
	}

	env.setup()
	t.Cleanup(env.Cleanup)

	return env
}

func (e *E2EEnv) setup() {
	e.t.Helper()

	// Create temp directory for this test
	tempDir, err := os.MkdirTemp("", "e2e-test-*")
	if err != nil {
		e.t.Fatalf("creating temp directory: %v", err)
	}
	e.tempDir = tempDir

	// Create bin directory for mock binaries
	e.binDir = filepath.Join(tempDir, "bin")
	if err := os.MkdirAll(e.binDir, 0o755); err != nil {
		e.t.Fatalf("creating bin directory: %v", err)
	}

	// Create specs directory
	e.specsDir = filepath.Join(tempDir, "specs")
	if err := os.MkdirAll(e.specsDir, 0o755); err != nil {
		e.t.Fatalf("creating specs directory: %v", err)
	}

	e.setupMockClaude()
	e.buildAutospec()
	e.captureAndSanitizeEnv()
}

func (e *E2EEnv) setupMockClaude() {
	e.t.Helper()

	mockPath := e.findMockClaudePath()
	claudeLink := filepath.Join(e.binDir, "claude")

	// Copy the mock script as "claude" (symlink doesn't work for shebang scripts on all systems)
	content, err := os.ReadFile(mockPath)
	if err != nil {
		e.t.Fatalf("reading mock-claude.sh: %v", err)
	}

	if err := os.WriteFile(claudeLink, content, 0o755); err != nil {
		e.t.Fatalf("writing mock claude binary: %v", err)
	}
}

func (e *E2EEnv) findMockClaudePath() string {
	e.t.Helper()

	// Get the path to the current source file
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		e.t.Fatal("failed to determine current file location")
	}

	// Navigate from internal/testutil/ to repo root
	repoRoot := filepath.Join(filepath.Dir(currentFile), "..", "..")

	// Try the primary location
	mockPath := filepath.Join(repoRoot, "mocks", "scripts", "mock-claude.sh")
	if _, err := os.Stat(mockPath); err == nil {
		return mockPath
	}

	e.t.Fatalf("mock-claude.sh not found at %s", mockPath)
	return ""
}

func (e *E2EEnv) buildAutospec() {
	e.t.Helper()

	// Build autospec binary once per test session
	autospecBuildOnce.Do(func() {
		autospecBinaryPath, autospecBuildErr = e.doBuildAutospec()
	})

	if autospecBuildErr != nil {
		e.t.Fatalf("building autospec: %v", autospecBuildErr)
	}

	// Link autospec binary to our bin directory
	autospecLink := filepath.Join(e.binDir, "autospec")
	content, err := os.ReadFile(autospecBinaryPath)
	if err != nil {
		e.t.Fatalf("reading autospec binary: %v", err)
	}

	if err := os.WriteFile(autospecLink, content, 0o755); err != nil {
		e.t.Fatalf("writing autospec binary: %v", err)
	}
}

func (e *E2EEnv) doBuildAutospec() (string, error) {
	// Get repo root
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("determining current file location")
	}
	repoRoot := filepath.Join(filepath.Dir(currentFile), "..", "..")

	// Create a temp directory for the built binary
	tmpDir, err := os.MkdirTemp("", "autospec-build-*")
	if err != nil {
		return "", fmt.Errorf("creating temp dir for build: %w", err)
	}

	binaryPath := filepath.Join(tmpDir, "autospec")

	cmd := exec.Command("go", "build", "-o", binaryPath, "./cmd/autospec")
	cmd.Dir = repoRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("building autospec: %w\nOutput: %s", err, output)
	}

	return binaryPath, nil
}

func (e *E2EEnv) captureAndSanitizeEnv() {
	e.t.Helper()

	// Capture original environment variables we'll modify
	sensitiveVars := []string{
		"ANTHROPIC_API_KEY",
		"CLAUDE_API_KEY",
		"PATH",
	}

	for _, key := range sensitiveVars {
		if val, ok := os.LookupEnv(key); ok {
			e.originalEnv[key] = val
		}
	}
}

// Run executes an autospec command in the isolated E2E environment.
func (e *E2EEnv) Run(args ...string) CommandResult {
	e.t.Helper()

	start := time.Now()

	cmd := exec.Command(filepath.Join(e.binDir, "autospec"), args...)
	cmd.Dir = e.tempDir
	cmd.Env = e.buildIsolatedEnv()

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	result := CommandResult{
		ExitCode: 0,
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		Duration: time.Since(start),
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.ExitCode = 1
		}
	}

	return result
}

func (e *E2EEnv) buildIsolatedEnv() []string {
	// Build PATH: our mock bin dir first (so mock claude takes precedence),
	// then system PATH for standard utilities (mkdir, cat, etc.)
	systemPath := os.Getenv("PATH")
	isolatedPath := e.binDir
	if systemPath != "" {
		isolatedPath = e.binDir + ":" + systemPath
	}

	env := []string{
		"PATH=" + isolatedPath,
		"HOME=" + e.tempDir,
		"MOCK_ARTIFACT_DIR=" + e.specsDir,
	}

	// Add mock exit code if configured
	if e.mockExitCodeSet {
		env = append(env, fmt.Sprintf("MOCK_EXIT_CODE=%d", e.mockExitCode))
	}

	// Add safe environment variables from original environment
	safeVars := []string{
		"TERM",
		"LANG",
		"LC_ALL",
		"TMPDIR",
		"TMP",
		"TEMP",
	}

	for _, key := range safeVars {
		if val, ok := os.LookupEnv(key); ok {
			env = append(env, key+"="+val)
		}
	}

	// Explicitly exclude API keys - they should NEVER be in the isolated env
	// This is verified by HasAPIKeyInEnv()

	return env
}

// TempDir returns the root temp directory for this test environment.
func (e *E2EEnv) TempDir() string {
	return e.tempDir
}

// SpecsDir returns the specs directory path.
func (e *E2EEnv) SpecsDir() string {
	return e.specsDir
}

// BinDir returns the bin directory containing mock and autospec binaries.
func (e *E2EEnv) BinDir() string {
	return e.binDir
}

// SetupConstitution creates a valid constitution.yaml in the test environment.
func (e *E2EEnv) SetupConstitution() {
	e.t.Helper()

	constitutionDir := filepath.Join(e.tempDir, ".autospec", "memory")
	if err := os.MkdirAll(constitutionDir, 0o755); err != nil {
		e.t.Fatalf("creating constitution directory: %v", err)
	}

	content := `constitution:
  project_name: "e2e-test-project"
  version: "1.0.0"
  ratified: "2025-01-01"
  last_amended: "2025-01-01"

preamble: "Test constitution for E2E testing."

principles:
  - name: "Test-First Development"
    id: "PRIN-001"
    category: "quality"
    priority: "NON-NEGOTIABLE"
    description: "All new code must have tests."
    rationale: "Ensures code quality"
    enforcement:
      - mechanism: "CI"
        description: "Tests run on commit"
    exceptions: []

sections:
  - name: "Code Quality"
    content: "All code must pass linting."

governance:
  amendment_process:
    - step: 1
      action: "Propose"
      requirements: "Include rationale"
  versioning_policy: "Semantic versioning"
  compliance_review:
    frequency: "quarterly"
    process: "Review"
  rules:
    - "Changes require review"

sync_impact:
  version_change: "1.0.0 -> 1.0.0"
  modified_principles: []
  added_sections: []
  removed_sections: []
  templates_requiring_updates: []
  follow_up_todos: []

_meta:
  version: "1.0.0"
  generator: "autospec"
  generator_version: "test"
  created: "2025-01-01T00:00:00Z"
  artifact_type: "constitution"
`
	path := filepath.Join(constitutionDir, "constitution.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		e.t.Fatalf("writing constitution: %v", err)
	}
}

// SetupSpec creates a valid spec.yaml in the test environment.
func (e *E2EEnv) SetupSpec(specName string) string {
	e.t.Helper()

	specDir := filepath.Join(e.specsDir, specName)
	if err := os.MkdirAll(specDir, 0o755); err != nil {
		e.t.Fatalf("creating spec directory: %v", err)
	}

	ArtifactGenerators.Spec(specDir)
	return specDir
}

// SetupPlan creates a valid plan.yaml in the test environment.
func (e *E2EEnv) SetupPlan(specName string) string {
	e.t.Helper()

	specDir := e.SetupSpec(specName) // Ensure spec exists first
	ArtifactGenerators.Plan(specDir)
	return specDir
}

// SetupTasks creates a valid tasks.yaml in the test environment.
func (e *E2EEnv) SetupTasks(specName string) string {
	e.t.Helper()

	specDir := e.SetupPlan(specName) // Ensure plan exists first
	ArtifactGenerators.Tasks(specDir)
	return specDir
}

// SetMockExitCode configures the mock claude to return a specific exit code.
// Note: This sets an internal field that will be used when Run is called.
func (e *E2EEnv) SetMockExitCode(code int) {
	e.t.Helper()
	// Store for use in buildIsolatedEnv - we'll add it there
	e.mockExitCode = code
	e.mockExitCodeSet = true
}

// HasAPIKeyInEnv returns true if ANTHROPIC_API_KEY is present in the environment.
// This is used to verify that E2E tests properly sanitize the environment.
func (e *E2EEnv) HasAPIKeyInEnv() bool {
	env := e.buildIsolatedEnv()
	for _, v := range env {
		if strings.HasPrefix(v, "ANTHROPIC_API_KEY=") {
			return true
		}
	}
	return false
}

// SpecExists checks if a spec artifact file exists.
func (e *E2EEnv) SpecExists(specName string) bool {
	path := filepath.Join(e.specsDir, specName, "spec.yaml")
	_, err := os.Stat(path)
	return err == nil
}

// PlanExists checks if a plan artifact file exists.
func (e *E2EEnv) PlanExists(specName string) bool {
	path := filepath.Join(e.specsDir, specName, "plan.yaml")
	_, err := os.Stat(path)
	return err == nil
}

// TasksExists checks if a tasks artifact file exists.
func (e *E2EEnv) TasksExists(specName string) bool {
	path := filepath.Join(e.specsDir, specName, "tasks.yaml")
	_, err := os.Stat(path)
	return err == nil
}

// Cleanup restores the original environment and removes temp files.
func (e *E2EEnv) Cleanup() {
	if e.cleanedUp {
		return
	}
	e.cleanedUp = true

	// Remove temp directory
	if e.tempDir != "" {
		if err := os.RemoveAll(e.tempDir); err != nil {
			e.t.Logf("note: could not remove temp directory: %v", err)
		}
	}
}

// InitGitRepo initializes a git repository in the temp directory.
func (e *E2EEnv) InitGitRepo() {
	e.t.Helper()

	cmd := exec.Command("git", "init")
	cmd.Dir = e.tempDir
	if output, err := cmd.CombinedOutput(); err != nil {
		e.t.Fatalf("git init failed: %v\nOutput: %s", err, output)
	}

	// Configure git user for commits
	cmd = exec.Command("git", "config", "user.email", "test@test.com")
	cmd.Dir = e.tempDir
	if output, err := cmd.CombinedOutput(); err != nil {
		e.t.Fatalf("git config email failed: %v\nOutput: %s", err, output)
	}

	cmd = exec.Command("git", "config", "user.name", "Test")
	cmd.Dir = e.tempDir
	if output, err := cmd.CombinedOutput(); err != nil {
		e.t.Fatalf("git config name failed: %v\nOutput: %s", err, output)
	}
}

// CreateBranch creates and checks out a new branch in the test git repo.
func (e *E2EEnv) CreateBranch(name string) {
	e.t.Helper()

	// Need an initial commit first
	readme := filepath.Join(e.tempDir, "README.md")
	if err := os.WriteFile(readme, []byte("# Test"), 0o644); err != nil {
		e.t.Fatalf("writing README: %v", err)
	}

	cmd := exec.Command("git", "add", ".")
	cmd.Dir = e.tempDir
	if output, err := cmd.CombinedOutput(); err != nil {
		e.t.Fatalf("git add failed: %v\nOutput: %s", err, output)
	}

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = e.tempDir
	if output, err := cmd.CombinedOutput(); err != nil {
		e.t.Fatalf("git commit failed: %v\nOutput: %s", err, output)
	}

	cmd = exec.Command("git", "checkout", "-b", name)
	cmd.Dir = e.tempDir
	if output, err := cmd.CombinedOutput(); err != nil {
		e.t.Fatalf("git checkout -b failed: %v\nOutput: %s", err, output)
	}
}
