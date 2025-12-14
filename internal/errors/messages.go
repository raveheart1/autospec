package errors

import "fmt"

// Common error messages for the autospec CLI.
// These templates ensure consistent, actionable error messages.

// MissingFeatureDescription creates an error for missing feature description argument.
func MissingFeatureDescription() *CLIError {
	return NewArgumentErrorWithUsage(
		"feature description is required",
		"autospec specify \"<feature description>\"",
		"Provide a feature description in quotes",
		"Example: autospec specify \"Add user authentication\"",
	)
}

// MissingSpecFile creates an error for missing spec.yaml file.
func MissingSpecFile(specDir string) *CLIError {
	return NewPrerequisiteError(
		fmt.Sprintf("spec.yaml not found in %s", specDir),
		"Run 'autospec specify \"<description>\"' first to create spec.yaml",
		"Or check that you're in the correct feature branch",
	)
}

// MissingPlanFile creates an error for missing plan.yaml file.
func MissingPlanFile(specDir string) *CLIError {
	return NewPrerequisiteError(
		fmt.Sprintf("plan.yaml not found in %s", specDir),
		"Run 'autospec plan' first to create plan.yaml",
		"Make sure spec.yaml exists before running plan",
	)
}

// MissingTasksFile creates an error for missing tasks.yaml file.
func MissingTasksFile(specDir string) *CLIError {
	return NewPrerequisiteError(
		fmt.Sprintf("tasks.yaml not found in %s", specDir),
		"Run 'autospec tasks' first to create tasks.yaml",
		"Make sure plan.yaml exists before running tasks",
	)
}

// SpecNotDetected creates an error when no spec can be auto-detected.
func SpecNotDetected() *CLIError {
	return NewPrerequisiteError(
		"could not detect current spec",
		"Check out a feature branch (e.g., git checkout 001-feature-name)",
		"Or specify the spec name explicitly: autospec implement 001-feature-name",
		"Create a new spec with: autospec specify \"<description>\"",
	)
}

// InvalidSpecNameFormat creates an error for invalid spec name format.
func InvalidSpecNameFormat(provided string) *CLIError {
	return NewArgumentErrorWithUsage(
		fmt.Sprintf("invalid spec name format: %s", provided),
		"autospec implement <NNN-feature-name>",
		"Spec names must match pattern: NNN-feature-name (e.g., 001-user-auth)",
		"Check available specs with: ls specs/",
	)
}

// ClaudeCliNotFound creates an error when Claude CLI is not installed.
func ClaudeCliNotFound() *CLIError {
	return NewPrerequisiteError(
		"claude command not found",
		"Install Claude CLI: npm install -g @anthropic-ai/claude-cli",
		"Or check that claude is in your PATH",
		"Verify installation with: claude --version",
	)
}

// ClaudeCliError creates an error when Claude CLI command fails.
func ClaudeCliError(err error) *CLIError {
	return WrapWithMessage(err, Runtime,
		"Claude CLI command failed",
		"Check your network connection",
		"Verify your API key is set: echo $ANTHROPIC_API_KEY",
		"Run 'autospec doctor' to diagnose issues",
	)
}

// ConfigFileNotFound creates an error for missing config file.
func ConfigFileNotFound(path string) *CLIError {
	return NewConfigError(
		fmt.Sprintf("config file not found: %s", path),
		"Run 'autospec init' to create default configuration",
		"Or create the file manually with required settings",
	)
}

// ConfigParseError creates an error for invalid config file format.
func ConfigParseError(path string, err error) *CLIError {
	return WrapWithMessage(err, Configuration,
		fmt.Sprintf("failed to parse config file: %s", path),
		"Check the file for JSON syntax errors",
		"Validate with: cat "+path+" | jq .",
		"Reset to defaults with: autospec init --force",
	)
}

// InvalidFlagCombination creates an error for incompatible flag combinations.
func InvalidFlagCombination(flags string, reason string) *CLIError {
	return NewArgumentError(
		fmt.Sprintf("invalid flag combination: %s", flags),
		reason,
		"Use 'autospec <command> --help' to see valid options",
	)
}

// TimeoutError creates an error when a command times out.
func TimeoutError(duration string, command string) *CLIError {
	return NewRuntimeError(
		fmt.Sprintf("command timed out after %s: %s", duration, command),
		"Increase timeout in config: AUTOSPEC_TIMEOUT=600",
		"Or edit .autospec/config.json and set \"timeout\": 600",
		"Set timeout to 0 to disable timeout",
	)
}

// DirectoryNotFound creates an error for missing directory.
func DirectoryNotFound(path string) *CLIError {
	return NewPrerequisiteError(
		fmt.Sprintf("directory not found: %s", path),
		"Create the directory with: mkdir -p "+path,
		"Or check that the path is correct",
	)
}

// FileNotWritable creates an error when a file cannot be written.
func FileNotWritable(path string) *CLIError {
	return NewRuntimeError(
		fmt.Sprintf("cannot write to file: %s", path),
		"Check file permissions: ls -la "+path,
		"Ensure parent directory exists and is writable",
	)
}

// NoTasksPending creates an error when no tasks are pending for implementation.
func NoTasksPending() *CLIError {
	return NewPrerequisiteError(
		"no pending tasks found in tasks.yaml",
		"All tasks may already be completed",
		"Run 'autospec status' to check current progress",
		"Generate new tasks with: autospec tasks",
	)
}

// TaskNotFound creates an error when a specified task ID doesn't exist.
func TaskNotFound(taskID string) *CLIError {
	return NewArgumentError(
		fmt.Sprintf("task not found: %s", taskID),
		"Check available tasks with: autospec status",
		"Task IDs are case-sensitive (e.g., T001, T002)",
	)
}

// InvalidTaskStatus creates an error for invalid task status.
func InvalidTaskStatus(status string) *CLIError {
	return NewArgumentError(
		fmt.Sprintf("invalid task status: %s", status),
		"Valid statuses: Pending, InProgress, Completed, Blocked",
		"Example: autospec update-task T001 Completed",
	)
}

// GitNotRepository creates an error when not in a git repository.
func GitNotRepository() *CLIError {
	return NewPrerequisiteError(
		"not a git repository",
		"Initialize with: git init",
		"Or navigate to an existing repository",
	)
}
