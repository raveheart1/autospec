package worktree

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// SetupResult contains the result of running a setup script.
type SetupResult struct {
	// Executed indicates whether the script was actually run.
	Executed bool
	// Output contains the combined stdout/stderr from the script.
	Output string
	// Error contains any error that occurred during execution.
	Error error
	// TimedOut indicates whether the script exceeded its timeout.
	TimedOut bool
}

// RunSetupScript executes a setup script with the given parameters.
// The script receives:
//   - Arguments: worktreePath, worktreeName, branchName
//   - Environment: WORKTREE_PATH, WORKTREE_NAME, WORKTREE_BRANCH, SOURCE_REPO
//
// Returns nil (with Executed=false) if the script doesn't exist.
func RunSetupScript(scriptPath, worktreePath, worktreeName, branchName, sourceRepo string, stdout io.Writer) *SetupResult {
	result := &SetupResult{Executed: false}

	if scriptPath == "" {
		return result
	}

	// Make script path absolute if relative
	if !filepath.IsAbs(scriptPath) {
		scriptPath = filepath.Join(sourceRepo, scriptPath)
	}

	// Check if script exists
	info, err := os.Stat(scriptPath)
	if err != nil {
		if os.IsNotExist(err) {
			return result
		}
		result.Error = fmt.Errorf("checking setup script: %w", err)
		return result
	}

	// Check if script is executable
	if info.Mode()&0o111 == 0 {
		result.Error = fmt.Errorf("setup script is not executable: %s", scriptPath)
		return result
	}

	result.Executed = true

	cmd := exec.Command(scriptPath, worktreePath, worktreeName, branchName)
	cmd.Dir = worktreePath
	cmd.Env = buildSetupEnv(worktreePath, worktreeName, branchName, sourceRepo)

	var outputBuf bytes.Buffer
	if stdout != nil {
		cmd.Stdout = io.MultiWriter(&outputBuf, stdout)
		cmd.Stderr = io.MultiWriter(&outputBuf, stdout)
	} else {
		cmd.Stdout = &outputBuf
		cmd.Stderr = &outputBuf
	}

	if err := cmd.Run(); err != nil {
		result.Output = outputBuf.String()
		result.Error = fmt.Errorf("running setup script: %w", err)
		return result
	}

	result.Output = outputBuf.String()
	return result
}

// DefaultSetupTimeout is the default timeout for setup script execution.
const DefaultSetupTimeout = 5 * time.Minute

// RunSetupScriptWithTimeout executes a setup script with timeout enforcement.
// If timeout is zero or negative, DefaultSetupTimeout (5 minutes) is used.
// Returns SetupResult with TimedOut=true if the script exceeds the timeout.
func RunSetupScriptWithTimeout(
	ctx context.Context,
	scriptPath, worktreePath, worktreeName, branchName, sourceRepo string,
	timeout time.Duration,
	stdout io.Writer,
) *SetupResult {
	if timeout <= 0 {
		log.Printf("Warning: invalid timeout %v, using default %v", timeout, DefaultSetupTimeout)
		timeout = DefaultSetupTimeout
	}

	result := prepareScript(scriptPath, sourceRepo)
	if result.Error != nil || !result.Executed {
		return result
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	return executeWithTimeout(timeoutCtx, scriptPath, worktreePath, worktreeName, branchName, sourceRepo, timeout, stdout)
}

// prepareScript validates the script exists and is executable.
func prepareScript(scriptPath, sourceRepo string) *SetupResult {
	result := &SetupResult{Executed: false}

	if scriptPath == "" {
		return result
	}

	if !filepath.IsAbs(scriptPath) {
		scriptPath = filepath.Join(sourceRepo, scriptPath)
	}

	info, err := os.Stat(scriptPath)
	if err != nil {
		if os.IsNotExist(err) {
			return result
		}
		result.Error = fmt.Errorf("checking setup script: %w", err)
		return result
	}

	if info.Mode()&0o111 == 0 {
		result.Error = fmt.Errorf("setup script is not executable: %s", scriptPath)
		return result
	}

	result.Executed = true
	return result
}

// executeWithTimeout runs the script with the given timeout context.
func executeWithTimeout(
	ctx context.Context,
	scriptPath, worktreePath, worktreeName, branchName, sourceRepo string,
	timeout time.Duration,
	stdout io.Writer,
) *SetupResult {
	result := &SetupResult{Executed: true}

	if !filepath.IsAbs(scriptPath) {
		scriptPath = filepath.Join(sourceRepo, scriptPath)
	}

	cmd := exec.CommandContext(ctx, scriptPath, worktreePath, worktreeName, branchName)
	cmd.Dir = worktreePath
	cmd.Env = buildSetupEnv(worktreePath, worktreeName, branchName, sourceRepo)

	var outputBuf bytes.Buffer
	if stdout != nil {
		cmd.Stdout = io.MultiWriter(&outputBuf, stdout)
		cmd.Stderr = io.MultiWriter(&outputBuf, stdout)
	} else {
		cmd.Stdout = &outputBuf
		cmd.Stderr = &outputBuf
	}

	if err := cmd.Run(); err != nil {
		result.Output = outputBuf.String()
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			result.TimedOut = true
			log.Printf("Warning: setup script timed out after %v", timeout)
			result.Error = fmt.Errorf("setup script timed out after %v", timeout)
		} else {
			result.Error = fmt.Errorf("running setup script: %w", err)
		}
		return result
	}

	result.Output = outputBuf.String()
	return result
}

// buildSetupEnv creates the environment for the setup script.
func buildSetupEnv(worktreePath, worktreeName, branchName, sourceRepo string) []string {
	env := os.Environ()
	env = append(env,
		"WORKTREE_PATH="+worktreePath,
		"WORKTREE_NAME="+worktreeName,
		"WORKTREE_BRANCH="+branchName,
		"SOURCE_REPO="+sourceRepo,
	)
	return env
}

// CreateDefaultSetupScript creates a template setup script at the given path.
func CreateDefaultSetupScript(path string) error {
	script := `#!/bin/bash
# Worktree setup script
# Arguments: $1 = worktree path, $2 = worktree name, $3 = branch name
# Environment: WORKTREE_PATH, WORKTREE_NAME, WORKTREE_BRANCH, SOURCE_REPO

set -e

WORKTREE_PATH="${1:-$WORKTREE_PATH}"
WORKTREE_NAME="${2:-$WORKTREE_NAME}"
WORKTREE_BRANCH="${3:-$WORKTREE_BRANCH}"

echo "Setting up worktree: $WORKTREE_NAME"
echo "Path: $WORKTREE_PATH"
echo "Branch: $WORKTREE_BRANCH"

cd "$WORKTREE_PATH"

# Add your project-specific setup commands below:
# Examples:
#   npm install
#   go mod download
#   pip install -r requirements.txt

echo "Setup complete!"
`

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating script directory: %w", err)
	}

	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		return fmt.Errorf("writing setup script: %w", err)
	}

	return nil
}
