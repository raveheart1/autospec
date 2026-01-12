package dag

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"text/template"
	"time"

	"github.com/ariel-frischer/autospec/internal/cliagent"
)

// TemplateVars holds the variables available for template expansion in autocommit_cmd.
type TemplateVars struct {
	// SpecID is the spec identifier.
	SpecID string
	// Worktree is the absolute path to the worktree directory.
	Worktree string
	// Branch is the current branch name.
	Branch string
	// BaseBranch is the target branch for merging.
	BaseBranch string
	// DagID is the DAG identifier.
	DagID string
}

// ExpandTemplateVars expands template variables in a command string.
// Uses Go's text/template package with {{.FieldName}} syntax.
// Returns an error if the template is invalid or execution fails.
func ExpandTemplateVars(cmdTemplate string, vars TemplateVars) (string, error) {
	if cmdTemplate == "" {
		return "", nil
	}

	tmpl, err := template.New("cmd").Parse(cmdTemplate)
	if err != nil {
		return "", fmt.Errorf("parsing command template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, vars); err != nil {
		return "", fmt.Errorf("executing command template: %w", err)
	}

	return buf.String(), nil
}

// ExpandAutocommitCmd expands the autocommit command using provided parameters.
// This is a convenience wrapper that constructs TemplateVars from individual values.
func ExpandAutocommitCmd(cmdTemplate, specID, worktree, branch, baseBranch, dagID string) (string, error) {
	vars := TemplateVars{
		SpecID:     specID,
		Worktree:   worktree,
		Branch:     branch,
		BaseBranch: baseBranch,
		DagID:      dagID,
	}
	return ExpandTemplateVars(cmdTemplate, vars)
}

// CommitVerifier handles post-execution commit verification and retry logic.
// It verifies that commits were made after spec execution and retries if needed.
type CommitVerifier struct {
	// config holds DAG execution configuration including autocommit settings.
	config *DAGExecutionConfig
	// stdout is the output destination for status messages.
	stdout io.Writer
	// stderr is the output destination for error messages.
	stderr io.Writer
	// cmdRunner is the command runner for executing commit commands.
	cmdRunner CommandRunner
	// agent is the CLI agent used for commit sessions (optional, defaults to claude).
	agent cliagent.Agent
}

// CommitVerifierOption is a functional option for configuring CommitVerifier.
type CommitVerifierOption func(*CommitVerifier)

// WithCommitAgent sets the CLI agent for commit sessions.
func WithCommitAgent(agent cliagent.Agent) CommitVerifierOption {
	return func(cv *CommitVerifier) {
		cv.agent = agent
	}
}

// NewCommitVerifier creates a new CommitVerifier with the given configuration.
func NewCommitVerifier(
	config *DAGExecutionConfig,
	stdout, stderr io.Writer,
	cmdRunner CommandRunner,
	opts ...CommitVerifierOption,
) *CommitVerifier {
	cv := &CommitVerifier{
		config:    config,
		stdout:    stdout,
		stderr:    stderr,
		cmdRunner: cmdRunner,
	}
	for _, opt := range opts {
		opt(cv)
	}
	return cv
}

// CommitResult contains the outcome of a commit verification flow.
type CommitResult struct {
	// Status is the final commit status (committed, failed, pending).
	Status CommitStatus
	// CommitSHA is the SHA of the commit if successful.
	CommitSHA string
	// Attempts is the number of commit attempts made.
	Attempts int
	// Error contains the error if the commit flow failed.
	Error error
}

// PostExecutionCommitFlow verifies commits after spec execution and retries if needed.
// Returns immediately if no uncommitted changes exist.
// If autocommit is disabled, logs a warning and returns pending status.
// Otherwise retries commit up to AutocommitRetries times.
func (cv *CommitVerifier) PostExecutionCommitFlow(
	ctx context.Context,
	specID, worktreePath, branch, baseBranch, dagID string,
) CommitResult {
	// Check for uncommitted changes first
	hasChanges, err := HasUncommittedChanges(worktreePath)
	if err != nil {
		return CommitResult{Status: CommitStatusFailed, Error: fmt.Errorf("checking uncommitted changes: %w", err)}
	}

	// No uncommitted changes - verify commits exist and return success
	if !hasChanges {
		return cv.verifyCommitsExist(worktreePath, baseBranch)
	}

	// Uncommitted changes exist - check if autocommit is enabled
	if !cv.config.IsAutocommitEnabled() {
		fmt.Fprintf(cv.stdout, "[%s] Warning: uncommitted changes exist but autocommit is disabled\n", specID)
		return CommitResult{Status: CommitStatusPending}
	}

	// Retry commit flow
	return cv.retryCommitFlow(ctx, specID, worktreePath, branch, baseBranch, dagID)
}

// verifyCommitsExist checks that commits exist ahead of the base branch.
func (cv *CommitVerifier) verifyCommitsExist(worktreePath, baseBranch string) CommitResult {
	commitsAhead, err := GetCommitsAhead(worktreePath, baseBranch)
	if err != nil {
		return CommitResult{Status: CommitStatusFailed, Error: fmt.Errorf("checking commits ahead: %w", err)}
	}

	if commitsAhead == 0 {
		return CommitResult{Status: CommitStatusPending, Error: fmt.Errorf("no commits ahead of %s", baseBranch)}
	}

	sha, err := GetCommitSHA(worktreePath)
	if err != nil {
		return CommitResult{Status: CommitStatusFailed, Error: fmt.Errorf("getting commit SHA: %w", err)}
	}

	return CommitResult{Status: CommitStatusCommitted, CommitSHA: sha}
}

// retryCommitFlow attempts to commit changes up to the configured retry count.
func (cv *CommitVerifier) retryCommitFlow(
	ctx context.Context,
	specID, worktreePath, branch, baseBranch, dagID string,
) CommitResult {
	maxRetries := cv.config.GetAutocommitRetries()
	var lastErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		fmt.Fprintf(cv.stdout, "[%s] Commit attempt %d/%d\n", specID, attempt, maxRetries)

		if err := cv.attemptCommit(ctx, specID, worktreePath, branch, baseBranch, dagID); err != nil {
			lastErr = err
			fmt.Fprintf(cv.stderr, "[%s] Commit attempt %d failed: %v\n", specID, attempt, err)
			continue
		}

		// Verify commit was successful
		result := cv.verifyCommitSuccess(worktreePath, baseBranch, attempt)
		if result.Status == CommitStatusCommitted {
			return result
		}
		lastErr = result.Error
	}

	return CommitResult{Status: CommitStatusFailed, Attempts: maxRetries, Error: lastErr}
}

// attemptCommit tries to commit using either custom command or agent session.
func (cv *CommitVerifier) attemptCommit(
	ctx context.Context,
	specID, worktreePath, branch, baseBranch, dagID string,
) error {
	if cv.config.AutocommitCmd != "" {
		return cv.RunCustomCommitCmd(ctx, specID, worktreePath, branch, baseBranch, dagID)
	}
	return cv.RunAgentCommitSession(ctx, specID, worktreePath)
}

// verifyCommitSuccess checks if commit was successful after an attempt.
func (cv *CommitVerifier) verifyCommitSuccess(worktreePath, baseBranch string, attempt int) CommitResult {
	hasChanges, err := HasUncommittedChanges(worktreePath)
	if err != nil {
		return CommitResult{Status: CommitStatusFailed, Attempts: attempt, Error: fmt.Errorf("checking uncommitted changes: %w", err)}
	}

	if hasChanges {
		return CommitResult{Status: CommitStatusFailed, Attempts: attempt, Error: fmt.Errorf("uncommitted changes still exist")}
	}

	result := cv.verifyCommitsExist(worktreePath, baseBranch)
	result.Attempts = attempt
	return result
}

// RunCustomCommitCmd executes the configured custom commit command.
// Expands template variables and runs the command in the worktree directory.
// Returns error if the command fails or times out (30 seconds default).
func (cv *CommitVerifier) RunCustomCommitCmd(
	ctx context.Context,
	specID, worktreePath, branch, baseBranch, dagID string,
) error {
	cmd, err := ExpandAutocommitCmd(cv.config.AutocommitCmd, specID, worktreePath, branch, baseBranch, dagID)
	if err != nil {
		return fmt.Errorf("expanding commit command: %w", err)
	}

	// Create context with 30 second timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Execute command via shell
	exitCode, err := cv.cmdRunner.Run(timeoutCtx, worktreePath, cv.stdout, cv.stderr, "sh", "-c", cmd)
	if err != nil {
		return fmt.Errorf("running commit command: %w", err)
	}
	if exitCode != 0 {
		return fmt.Errorf("commit command failed with exit code %d", exitCode)
	}

	return nil
}

// RunAgentCommitSession runs an agent session to commit changes.
// Constructs a commit-focused prompt with git status output.
// Uses the configured agent to execute the commit prompt in the worktree directory.
// The stdout writer should be a FormatterWriter for clean, colored stream-json output.
func (cv *CommitVerifier) RunAgentCommitSession(
	ctx context.Context,
	specID, worktreePath string,
) error {
	// Get git status for the prompt
	status, err := cv.getGitStatusOutput(worktreePath)
	if err != nil {
		return fmt.Errorf("getting git status: %w", err)
	}

	// Build commit-focused prompt
	prompt := cv.buildCommitPrompt(specID, status)

	// Use agent if configured, otherwise fall back to default claude agent
	agent := cv.agent
	if agent == nil {
		agent = cliagent.NewClaude()
	}

	// Show the command being executed (similar to workflow runs)
	fmt.Fprintf(cv.stdout, "Executing: %s -p \"<commit prompt>\"\n", agent.Name())

	// Execute the commit prompt via the agent, streaming output
	// Caller should wrap stdout with FormatterWriter for cclean formatting
	result, err := agent.Execute(ctx, prompt, cliagent.ExecOptions{
		Autonomous: true,
		WorkDir:    worktreePath,
		Stdout:     cv.stdout,
		Stderr:     cv.stderr,
	})
	if err != nil {
		return fmt.Errorf("running agent commit session: %w", err)
	}
	if result.ExitCode != 0 {
		return fmt.Errorf("agent commit session failed with exit code %d", result.ExitCode)
	}

	return nil
}

// getGitStatusOutput runs git status --porcelain and returns the output.
func (cv *CommitVerifier) getGitStatusOutput(worktreePath string) (string, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = worktreePath

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("running git status: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

// buildCommitPrompt constructs a prompt for the agent to commit changes.
func (cv *CommitVerifier) buildCommitPrompt(specID, statusOutput string) string {
	var sb strings.Builder
	sb.WriteString("Commit all uncommitted changes for spec ")
	sb.WriteString(specID)
	sb.WriteString(".\n\nCurrent git status:\n")
	sb.WriteString(statusOutput)
	sb.WriteString("\n\nInstructions:\n")
	sb.WriteString("1. Stage all changes with 'git add .'\n")
	sb.WriteString("2. Commit with a descriptive message summarizing the changes\n")
	sb.WriteString("3. Do not push to remote\n")
	return sb.String()
}
