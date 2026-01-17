package worktree

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// Manager defines the interface for worktree CRUD operations.
type Manager interface {
	// Create creates a new worktree with the given name and branch.
	Create(name, branch, customPath string) (*Worktree, error)
	// CreateWithOptions creates a new worktree with custom creation options.
	CreateWithOptions(name, branch, customPath string, opts CreateOptions) (*Worktree, error)
	// List returns all tracked worktrees.
	List() ([]Worktree, error)
	// Get returns a worktree by name.
	Get(name string) (*Worktree, error)
	// Remove removes a worktree by name.
	Remove(name string, force bool) error
	// Setup runs setup on an existing worktree path.
	Setup(path string, addToState bool) (*Worktree, error)
	// Prune removes stale worktree entries.
	Prune() (int, error)
	// UpdateStatus updates the status of a worktree.
	UpdateStatus(name string, status WorktreeStatus) error
}

// DefaultManager implements the Manager interface.
type DefaultManager struct {
	config     *WorktreeConfig
	stateDir   string
	repoRoot   string
	stdout     io.Writer
	gitOps     GitOperations
	copyFn     CopyFunc
	runSetupFn SetupFunc
	validateFn ValidateFunc
}

// GitOperations defines the git operations used by the manager.
// This interface enables testing with mocks.
type GitOperations interface {
	Add(repoPath, worktreePath, branch, startPoint string) error
	Remove(repoPath, worktreePath string, force bool) error
	List(repoPath string) ([]GitWorktreeEntry, error)
	HasUncommittedChanges(path string) (bool, error)
	HasUnpushedCommits(path string) (bool, error)
}

// CopyFunc is the function signature for directory copying.
type CopyFunc func(srcRoot, dstRoot string, dirs []string) ([]string, error)

// SetupFunc is the function signature for running setup scripts.
type SetupFunc func(scriptPath, worktreePath, name, branch, sourceRepo string, stdout io.Writer) *SetupResult

// ValidateFunc is the function signature for validating worktrees after setup.
type ValidateFunc func(worktreePath, sourceRepoPath string) (*ValidationResult, error)

// defaultGitOps implements GitOperations using the real git commands.
type defaultGitOps struct{}

func (g *defaultGitOps) Add(repoPath, worktreePath, branch, startPoint string) error {
	return GitWorktreeAdd(repoPath, worktreePath, branch, startPoint)
}

func (g *defaultGitOps) Remove(repoPath, worktreePath string, force bool) error {
	return GitWorktreeRemove(repoPath, worktreePath, force)
}

func (g *defaultGitOps) List(repoPath string) ([]GitWorktreeEntry, error) {
	return GitWorktreeList(repoPath)
}

func (g *defaultGitOps) HasUncommittedChanges(path string) (bool, error) {
	return HasUncommittedChanges(path)
}

func (g *defaultGitOps) HasUnpushedCommits(path string) (bool, error) {
	return HasUnpushedCommits(path)
}

// ManagerOption configures a DefaultManager.
type ManagerOption func(*DefaultManager)

// WithStdout sets the stdout writer for manager output.
func WithStdout(w io.Writer) ManagerOption {
	return func(m *DefaultManager) {
		m.stdout = w
	}
}

// WithGitOps sets custom git operations (for testing).
func WithGitOps(ops GitOperations) ManagerOption {
	return func(m *DefaultManager) {
		m.gitOps = ops
	}
}

// WithCopyFunc sets a custom copy function (for testing).
func WithCopyFunc(fn CopyFunc) ManagerOption {
	return func(m *DefaultManager) {
		m.copyFn = fn
	}
}

// WithSetupFunc sets a custom setup function (for testing).
func WithSetupFunc(fn SetupFunc) ManagerOption {
	return func(m *DefaultManager) {
		m.runSetupFn = fn
	}
}

// WithValidateFunc sets a custom validation function (for testing).
func WithValidateFunc(fn ValidateFunc) ManagerOption {
	return func(m *DefaultManager) {
		m.validateFn = fn
	}
}

// NewManager creates a new DefaultManager.
func NewManager(config *WorktreeConfig, stateDir, repoRoot string, opts ...ManagerOption) *DefaultManager {
	if config == nil {
		config = DefaultConfig()
	}

	m := &DefaultManager{
		config:     config,
		stateDir:   stateDir,
		repoRoot:   repoRoot,
		stdout:     os.Stdout,
		gitOps:     &defaultGitOps{},
		copyFn:     CopyDirs,
		runSetupFn: RunSetupScript,
		validateFn: ValidateWorktree,
	}

	for _, opt := range opts {
		opt(m)
	}

	return m
}

// Create creates a new worktree with the given name and branch.
// Delegates to CreateWithOptions with default options.
func (m *DefaultManager) Create(name, branch, customPath string) (*Worktree, error) {
	return m.CreateWithOptions(name, branch, customPath, CreateOptions{})
}

// CreateWithOptions creates a new worktree with custom creation options.
// Supports skipping directory copying, skipping setup, and controlling rollback behavior.
func (m *DefaultManager) CreateWithOptions(
	name, branch, customPath string,
	opts CreateOptions,
) (*Worktree, error) {
	state, err := LoadState(m.stateDir)
	if err != nil {
		return nil, fmt.Errorf("loading state: %w", err)
	}

	if state.FindWorktree(name) != nil {
		return nil, fmt.Errorf("worktree %q already exists", name)
	}

	worktreePath := m.resolveWorktreePath(name, customPath)

	if err := m.gitOps.Add(m.repoRoot, worktreePath, branch, opts.StartPoint); err != nil {
		return nil, fmt.Errorf("creating git worktree: %w", err)
	}

	// Copy directories unless skipped
	if !opts.SkipCopy {
		m.copyDirsToWorktree(worktreePath)
	}

	// Run setup and validation, handling failures with rollback
	outcome := m.runSetupWithRollback(worktreePath, name, branch, opts)
	if outcome.Error != nil {
		return nil, fmt.Errorf("setup failed: %w", outcome.Error)
	}

	return m.saveWorktreeToState(state, name, worktreePath, branch, outcome)
}

// copyDirsToWorktree copies configured directories to the worktree.
func (m *DefaultManager) copyDirsToWorktree(worktreePath string) {
	copied, err := m.copyFn(m.repoRoot, worktreePath, m.config.CopyDirs)
	if err != nil {
		fmt.Fprintf(m.stdout, "Warning: failed to copy directories: %v\n", err)
	} else if len(copied) > 0 {
		fmt.Fprintf(m.stdout, "Copied directories: %v\n", copied)
	}
}

// runSetupWithRollback runs setup and handles rollback on failure.
func (m *DefaultManager) runSetupWithRollback(
	worktreePath, name, branch string,
	opts CreateOptions,
) SetupOutcome {
	if opts.SkipSetup {
		fmt.Fprintf(m.stdout, "Skipping setup script (--skip-setup)\n")
		return SetupOutcome{SetupCompleted: false}
	}

	outcome := m.runSetupIfConfigured(worktreePath, name, branch)

	if outcome.Error != nil {
		m.handleSetupFailure(worktreePath, opts.NoRollback)
	}

	return outcome
}

// handleSetupFailure performs rollback or logs warning based on NoRollback option.
func (m *DefaultManager) handleSetupFailure(worktreePath string, noRollback bool) {
	if noRollback {
		fmt.Fprintf(m.stdout, "Warning: worktree preserved in broken state (--no-rollback)\n")
		fmt.Fprintf(m.stdout, "Manual cleanup may be required: %s\n", worktreePath)
		return
	}
	_ = m.rollbackWorktree(worktreePath)
}

// saveWorktreeToState saves the worktree to state if tracking is enabled.
func (m *DefaultManager) saveWorktreeToState(
	state *WorktreeState,
	name, worktreePath, branch string,
	outcome SetupOutcome,
) (*Worktree, error) {
	wt := Worktree{
		Name:           name,
		Path:           worktreePath,
		Branch:         branch,
		Status:         StatusActive,
		CreatedAt:      time.Now(),
		SetupCompleted: outcome.SetupCompleted,
		LastAccessed:   time.Now(),
	}

	if m.config.TrackStatus {
		if err := state.AddWorktree(wt); err != nil {
			return nil, fmt.Errorf("adding to state: %w", err)
		}
		if err := SaveState(m.stateDir, state); err != nil {
			return nil, fmt.Errorf("saving state: %w", err)
		}
	}

	return &wt, nil
}

// resolveWorktreePath determines the path for a new worktree.
func (m *DefaultManager) resolveWorktreePath(name, customPath string) string {
	if customPath != "" {
		if filepath.IsAbs(customPath) {
			return customPath
		}
		return filepath.Join(m.repoRoot, customPath)
	}

	baseDir := m.config.BaseDir
	if baseDir == "" {
		baseDir = filepath.Dir(m.repoRoot)
	}

	dirName := m.config.Prefix + name
	return filepath.Join(baseDir, dirName)
}

// SetupOutcome contains the result of running setup and validation.
type SetupOutcome struct {
	// SetupCompleted indicates if setup completed successfully.
	SetupCompleted bool
	// ValidationResult contains validation results (only for custom scripts).
	ValidationResult *ValidationResult
	// Error contains any error during setup or validation.
	Error error
}

// runSetupIfConfigured runs the setup script if configured.
func (m *DefaultManager) runSetupIfConfigured(worktreePath, name, branch string) SetupOutcome {
	outcome := SetupOutcome{SetupCompleted: true}

	if !m.config.AutoSetup || m.config.SetupScript == "" {
		return outcome // No setup needed, consider completed
	}

	result := m.runSetupFn(m.config.SetupScript, worktreePath, name, branch, m.repoRoot, m.stdout)

	if !result.Executed {
		return outcome // Script didn't exist, consider completed
	}

	if result.Error != nil {
		outcome.SetupCompleted = false
		outcome.Error = result.Error
		fmt.Fprintf(m.stdout, "Warning: setup script failed: %v\n", result.Error)
		return outcome
	}

	// Validate worktree after custom setup script completes successfully
	outcome = m.validateAfterSetup(worktreePath, outcome)
	return outcome
}

// validateAfterSetup runs validation checks after a custom setup script.
// Only runs when a custom setup_script is configured (not for default setup).
func (m *DefaultManager) validateAfterSetup(worktreePath string, outcome SetupOutcome) SetupOutcome {
	validationResult, err := m.validateFn(worktreePath, m.repoRoot)
	if err != nil {
		outcome.SetupCompleted = false
		outcome.Error = fmt.Errorf("validating worktree: %w", err)
		return outcome
	}

	outcome.ValidationResult = validationResult

	if !validationResult.IsValid() {
		outcome.SetupCompleted = false
		outcome.Error = fmt.Errorf("worktree validation failed: %v", validationResult.Errors)
		fmt.Fprintf(m.stdout, "Worktree validation failed:\n")
		for _, errMsg := range validationResult.Errors {
			fmt.Fprintf(m.stdout, "  - %s\n", errMsg)
		}
	}

	return outcome
}

// List returns all tracked worktrees.
func (m *DefaultManager) List() ([]Worktree, error) {
	state, err := LoadState(m.stateDir)
	if err != nil {
		return nil, fmt.Errorf("loading state: %w", err)
	}

	// Update stale status for worktrees where path doesn't exist
	for i := range state.Worktrees {
		if _, err := os.Stat(state.Worktrees[i].Path); os.IsNotExist(err) {
			state.Worktrees[i].Status = StatusStale
		}
	}

	return state.Worktrees, nil
}

// Get returns a worktree by name.
func (m *DefaultManager) Get(name string) (*Worktree, error) {
	state, err := LoadState(m.stateDir)
	if err != nil {
		return nil, fmt.Errorf("loading state: %w", err)
	}

	wt := state.FindWorktree(name)
	if wt == nil {
		return nil, fmt.Errorf("worktree %q not found", name)
	}

	return wt, nil
}

// Remove removes a worktree by name.
func (m *DefaultManager) Remove(name string, force bool) error {
	state, err := LoadState(m.stateDir)
	if err != nil {
		return fmt.Errorf("loading state: %w", err)
	}

	wt := state.FindWorktree(name)
	if wt == nil {
		return fmt.Errorf("worktree %q not found", name)
	}

	if !force {
		if err := m.checkSafeToRemove(wt.Path); err != nil {
			return err
		}
	}

	if err := m.gitOps.Remove(m.repoRoot, wt.Path, force); err != nil {
		return fmt.Errorf("removing git worktree: %w", err)
	}

	state.RemoveWorktree(name)

	if err := SaveState(m.stateDir, state); err != nil {
		return fmt.Errorf("saving state: %w", err)
	}

	return nil
}

// checkSafeToRemove checks if it's safe to remove a worktree.
func (m *DefaultManager) checkSafeToRemove(path string) error {
	// Skip check if path doesn't exist
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	}

	hasChanges, err := m.gitOps.HasUncommittedChanges(path)
	if err != nil {
		return fmt.Errorf("checking uncommitted changes: %w", err)
	}
	if hasChanges {
		return fmt.Errorf("worktree has uncommitted changes (use --force to override)")
	}

	hasUnpushed, err := m.gitOps.HasUnpushedCommits(path)
	if err != nil {
		return fmt.Errorf("checking unpushed commits: %w", err)
	}
	if hasUnpushed {
		return fmt.Errorf("worktree has unpushed commits (use --force to override)")
	}

	return nil
}

// Setup runs setup on an existing worktree path.
func (m *DefaultManager) Setup(path string, addToState bool) (*Worktree, error) {
	absPath, err := m.validateWorktreePath(path)
	if err != nil {
		return nil, err
	}

	m.copyDirsToWorktree(absPath)

	name := filepath.Base(absPath)
	branch := m.getBranchForPath(absPath)

	outcome := m.runSetupIfConfigured(absPath, name, branch)
	if outcome.Error != nil {
		return nil, fmt.Errorf("setup failed: %w", outcome.Error)
	}

	wt := m.buildWorktree(name, absPath, branch, outcome.SetupCompleted)

	if addToState && m.config.TrackStatus {
		if err := m.persistWorktreeToState(wt); err != nil {
			return nil, err
		}
	}

	return &wt, nil
}

// validateWorktreePath validates and returns the absolute path.
func (m *DefaultManager) validateWorktreePath(path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolving path: %w", err)
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return "", fmt.Errorf("path does not exist: %s", absPath)
	}

	isWT, err := IsWorktree(absPath)
	if err != nil {
		return "", fmt.Errorf("checking if path is worktree: %w", err)
	}
	if !isWT {
		return "", fmt.Errorf("path is not a git worktree: %s", absPath)
	}

	return absPath, nil
}

// buildWorktree creates a Worktree struct with the given parameters.
func (m *DefaultManager) buildWorktree(name, path, branch string, setupCompleted bool) Worktree {
	return Worktree{
		Name:           name,
		Path:           path,
		Branch:         branch,
		Status:         StatusActive,
		CreatedAt:      time.Now(),
		SetupCompleted: setupCompleted,
		LastAccessed:   time.Now(),
	}
}

// persistWorktreeToState saves a worktree to the state file.
func (m *DefaultManager) persistWorktreeToState(wt Worktree) error {
	state, err := LoadState(m.stateDir)
	if err != nil {
		return fmt.Errorf("loading state: %w", err)
	}

	if err := state.AddWorktree(wt); err != nil {
		return fmt.Errorf("adding to state: %w", err)
	}

	if err := SaveState(m.stateDir, state); err != nil {
		return fmt.Errorf("saving state: %w", err)
	}

	return nil
}

// getBranchForPath gets the branch checked out in a worktree.
func (m *DefaultManager) getBranchForPath(path string) string {
	entries, err := m.gitOps.List(m.repoRoot)
	if err != nil {
		return "unknown"
	}

	for _, entry := range entries {
		if entry.Path == path {
			return entry.Branch
		}
	}

	return "unknown"
}

// Prune removes stale worktree entries.
func (m *DefaultManager) Prune() (int, error) {
	state, err := LoadState(m.stateDir)
	if err != nil {
		return 0, fmt.Errorf("loading state: %w", err)
	}

	var remaining []Worktree
	var pruned int

	for _, wt := range state.Worktrees {
		if _, err := os.Stat(wt.Path); os.IsNotExist(err) {
			pruned++
			continue
		}
		remaining = append(remaining, wt)
	}

	if pruned > 0 {
		state.Worktrees = remaining
		if err := SaveState(m.stateDir, state); err != nil {
			return 0, fmt.Errorf("saving state: %w", err)
		}
	}

	return pruned, nil
}

// rollbackWorktree removes a worktree directory after setup or validation failure.
// Uses git worktree remove --force for proper git state cleanup.
// Logs actions for debugging and handles partial cleanup failures gracefully.
func (m *DefaultManager) rollbackWorktree(worktreePath string) error {
	fmt.Fprintf(m.stdout, "Rolling back: removing worktree at %s\n", worktreePath)

	if err := m.gitOps.Remove(m.repoRoot, worktreePath, true); err != nil {
		// Log error but continue - best-effort cleanup
		fmt.Fprintf(m.stdout, "Warning: rollback cleanup failed: %v\n", err)
		fmt.Fprintf(m.stdout, "Manual cleanup may be required for: %s\n", worktreePath)
		return fmt.Errorf("rollback cleanup failed: %w", err)
	}

	fmt.Fprintf(m.stdout, "Rollback complete: worktree removed\n")
	return nil
}

// UpdateStatus updates the status of a worktree.
func (m *DefaultManager) UpdateStatus(name string, status WorktreeStatus) error {
	if !status.IsValid() {
		return fmt.Errorf("invalid status: %s", status)
	}

	state, err := LoadState(m.stateDir)
	if err != nil {
		return fmt.Errorf("loading state: %w", err)
	}

	wt := state.FindWorktree(name)
	if wt == nil {
		return fmt.Errorf("worktree %q not found", name)
	}

	wt.Status = status
	wt.LastAccessed = time.Now()

	if status == StatusMerged {
		now := time.Now()
		wt.MergedAt = &now
	}

	if err := state.UpdateWorktree(*wt); err != nil {
		return fmt.Errorf("updating worktree: %w", err)
	}

	if err := SaveState(m.stateDir, state); err != nil {
		return fmt.Errorf("saving state: %w", err)
	}

	return nil
}
