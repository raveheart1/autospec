package worktree

import (
	"fmt"
	"os"
	"path/filepath"
)

// ValidationResult contains the results of worktree validation checks.
// All three checks must pass for the worktree to be considered valid.
type ValidationResult struct {
	// PathExists indicates whether the worktree path exists as a directory.
	PathExists bool
	// PathDiffersFromSource indicates whether the worktree path differs from source repo.
	PathDiffersFromSource bool
	// InGitWorktreeList indicates whether the worktree appears in git worktree list.
	InGitWorktreeList bool
	// Errors contains actionable error messages for any failed validation checks.
	Errors []string
}

// IsValid returns true if all validation checks passed and there are no errors.
func (v *ValidationResult) IsValid() bool {
	return v.PathExists && v.PathDiffersFromSource && v.InGitWorktreeList && len(v.Errors) == 0
}

// ValidateWorktree performs validation checks on a worktree after setup.
// It verifies path existence, path differs from source, and git worktree registration.
func ValidateWorktree(worktreePath, sourceRepoPath string) (*ValidationResult, error) {
	result := &ValidationResult{}

	absWorktreePath, err := filepath.Abs(worktreePath)
	if err != nil {
		return nil, fmt.Errorf("resolving worktree path: %w", err)
	}

	absSourcePath, err := filepath.Abs(sourceRepoPath)
	if err != nil {
		return nil, fmt.Errorf("resolving source path: %w", err)
	}

	result.PathExists = checkPathExists(absWorktreePath, result)
	result.PathDiffersFromSource = checkPathDiffers(absWorktreePath, absSourcePath, result)
	result.InGitWorktreeList = checkInWorktreeList(absWorktreePath, absSourcePath, result)

	return result, nil
}

// checkPathExists verifies the worktree directory exists.
func checkPathExists(path string, result *ValidationResult) bool {
	info, err := os.Stat(path)
	if err != nil {
		result.Errors = append(result.Errors,
			fmt.Sprintf("worktree path does not exist: %s (ensure setup script creates the directory)", path))
		return false
	}

	if !info.IsDir() {
		result.Errors = append(result.Errors,
			fmt.Sprintf("worktree path is not a directory: %s", path))
		return false
	}

	return true
}

// checkPathDiffers verifies worktree path differs from source repository.
func checkPathDiffers(worktreePath, sourcePath string, result *ValidationResult) bool {
	if worktreePath == sourcePath {
		result.Errors = append(result.Errors,
			fmt.Sprintf("worktree path same as source repo: %s (setup script may have changed directory)", worktreePath))
		return false
	}
	return true
}

// checkInWorktreeList verifies the worktree appears in git worktree list.
func checkInWorktreeList(worktreePath, repoPath string, result *ValidationResult) bool {
	entries, err := GitWorktreeList(repoPath)
	if err != nil {
		result.Errors = append(result.Errors,
			fmt.Sprintf("failed to list git worktrees: %v", err))
		return false
	}

	for _, entry := range entries {
		if entry.Path == worktreePath {
			return true
		}
	}

	result.Errors = append(result.Errors,
		fmt.Sprintf("worktree not found in git worktree list: %s (run 'git worktree list' to verify)", worktreePath))
	return false
}
