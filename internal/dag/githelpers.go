package dag

import (
	"bytes"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// HasUncommittedChanges checks if a worktree has uncommitted changes.
// Uses 'git status --porcelain' which returns output only when changes exist.
// Returns true if any tracked files are modified, staged, or untracked files exist.
func HasUncommittedChanges(worktreePath string) (bool, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = worktreePath

	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("checking uncommitted changes: %w", err)
	}

	return len(bytes.TrimSpace(output)) > 0, nil
}

// GetUncommittedFiles returns a list of files with uncommitted changes.
// Uses 'git status --porcelain' and parses the output to extract file paths.
// Returns empty slice if no uncommitted changes exist.
func GetUncommittedFiles(worktreePath string) ([]string, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = worktreePath

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("getting uncommitted files: %w", err)
	}

	return parseStatusOutput(output), nil
}

// parseStatusOutput extracts file paths from git status --porcelain output.
// Each line is in format "XY filename" where XY is a 2-character status code.
// The format is exactly: 2 status chars + 1 space + filename.
func parseStatusOutput(output []byte) []string {
	lines := strings.Split(string(output), "\n")
	var files []string

	for _, line := range lines {
		// Don't trim leading spaces - they're part of the status code
		if len(line) < 4 {
			continue
		}
		// Status codes are first 2 chars, space at position 2, filename starts at 3
		// Format: "XY filename" where XY is the status (may include space)
		filename := extractFilename(line[3:])
		if filename != "" {
			files = append(files, filename)
		}
	}

	return files
}

// extractFilename handles both regular filenames and rename format.
func extractFilename(raw string) string {
	// Handle rename format: "old -> new"
	if _, after, found := strings.Cut(raw, " -> "); found {
		return strings.TrimSpace(after)
	}
	return strings.TrimSpace(raw)
}

// GetCommitsAhead returns the number of commits ahead of the target branch.
// Uses 'git rev-list --count <target>..HEAD' to count commits.
// Returns 0 if the branch is at or behind the target.
func GetCommitsAhead(worktreePath, targetBranch string) (int, error) {
	if targetBranch == "" {
		return 0, fmt.Errorf("target branch is required")
	}

	cmd := exec.Command("git", "rev-list", "--count", targetBranch+"..HEAD")
	cmd.Dir = worktreePath

	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("counting commits ahead of %s: %w", targetBranch, err)
	}

	count, err := strconv.Atoi(strings.TrimSpace(string(output)))
	if err != nil {
		return 0, fmt.Errorf("parsing commit count: %w", err)
	}

	return count, nil
}

// GetCommitSHA returns the SHA of the current HEAD commit in the worktree.
// Uses 'git rev-parse HEAD' to get the full 40-character SHA.
func GetCommitSHA(worktreePath string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = worktreePath

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("getting HEAD commit SHA: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

// GetCurrentBranch returns the name of the current branch in the worktree.
// Returns empty string with error if in detached HEAD state.
func GetCurrentBranch(worktreePath string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = worktreePath

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("getting current branch: %w", err)
	}

	branch := strings.TrimSpace(string(output))
	if branch == "HEAD" {
		return "", fmt.Errorf("detached HEAD state")
	}

	return branch, nil
}
