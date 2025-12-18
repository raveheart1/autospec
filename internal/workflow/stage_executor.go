// Package workflow provides stage execution functionality.
// StageExecutor handles specify, plan, and tasks stage execution.
// Related: internal/workflow/workflow.go (orchestrator), internal/workflow/interfaces.go (interface definition)
// Tags: workflow, stage-executor, specify, plan, tasks
package workflow

import (
	"fmt"
	"path/filepath"

	"github.com/ariel-frischer/autospec/internal/retry"
	"github.com/ariel-frischer/autospec/internal/spec"
)

// StageExecutor handles specify, plan, and tasks stage execution.
// It implements StageExecutorInterface to enable dependency injection and testing.
// Each stage transforms artifacts: specify creates spec.yaml, plan creates plan.yaml,
// tasks creates tasks.yaml.
type StageExecutor struct {
	executor *Executor // Underlying executor for Claude command execution
	specsDir string    // Base directory for spec storage (e.g., "specs/")
	debug    bool      // Enable debug logging
}

// NewStageExecutor creates a new StageExecutor with the given dependencies.
// executor: required, handles actual command execution with retry logic
// specsDir: required, base directory where spec directories are created
// debug: optional, enables verbose logging for troubleshooting
func NewStageExecutor(executor *Executor, specsDir string, debug bool) *StageExecutor {
	return &StageExecutor{
		executor: executor,
		specsDir: specsDir,
		debug:    debug,
	}
}

// debugLog prints a debug message if debug mode is enabled.
func (s *StageExecutor) debugLog(format string, args ...interface{}) {
	if s.debug {
		fmt.Printf("[DEBUG][StageExecutor] "+format+"\n", args...)
	}
}

// ExecuteSpecify runs the specify stage for a feature description.
// Returns the spec name (e.g., "003-command-timeout") on success.
// The spec name is derived from the newly created spec directory.
func (s *StageExecutor) ExecuteSpecify(featureDescription string) (string, error) {
	s.debugLog("ExecuteSpecify called with description: %s", featureDescription)

	// Reset retry state for specify stage - each specify run creates a NEW spec,
	// so retry state from previous specify runs should not persist.
	if err := retry.ResetRetryCount(s.executor.StateDir, "", string(StageSpecify)); err != nil {
		s.debugLog("Warning: failed to reset specify retry state: %v", err)
	}

	command := fmt.Sprintf("/autospec.specify \"%s\"", featureDescription)

	// Use validation with detection since spec name is not known until Claude creates it.
	validateFunc := MakeSpecSchemaValidatorWithDetection(s.specsDir)

	result, err := s.executor.ExecuteStage(
		"", // Spec name not known yet
		StageSpecify,
		command,
		validateFunc,
	)

	if err != nil {
		totalAttempts := result.RetryCount + 1
		return "", fmt.Errorf("specify failed after %d total attempts (%d retries): %w",
			totalAttempts, result.RetryCount, err)
	}

	// Detect the newly created spec
	metadata, err := spec.DetectCurrentSpec(s.specsDir)
	if err != nil {
		return "", fmt.Errorf("detecting created spec: %w", err)
	}

	// Validate spec.yaml exists
	if err := s.executor.ValidateSpec(metadata.Directory); err != nil {
		return "", fmt.Errorf("validating spec: %w", err)
	}

	// Return full spec directory name (e.g., "003-command-timeout")
	specName := fmt.Sprintf("%s-%s", metadata.Number, metadata.Name)
	s.debugLog("ExecuteSpecify completed successfully: %s", specName)
	return specName, nil
}

// ExecutePlan runs the plan stage for an existing spec.
// specNameArg: spec name or empty string to auto-detect from git branch
// prompt: optional custom prompt to pass to the plan command
func (s *StageExecutor) ExecutePlan(specNameArg string, prompt string) error {
	specName, err := s.resolveSpecName(specNameArg)
	if err != nil {
		return fmt.Errorf("resolving spec name: %w", err)
	}

	s.debugLog("ExecutePlan called for spec: %s, prompt: %s", specName, prompt)

	command := s.buildPlanCommand(prompt)
	specDir := filepath.Join(s.specsDir, specName)

	result, err := s.executor.ExecuteStage(
		specName,
		StagePlan,
		command,
		ValidatePlanSchema,
	)

	if err != nil {
		totalAttempts := result.RetryCount + 1
		if result.Exhausted {
			return fmt.Errorf("plan stage exhausted retries after %d total attempts: %w",
				totalAttempts, err)
		}
		return fmt.Errorf("plan failed after %d total attempts (%d retries): %w",
			totalAttempts, result.RetryCount, err)
	}

	// Check for research.md (optional but usually created)
	researchPath := filepath.Join(specDir, "research.md")
	if _, statErr := filepath.Glob(researchPath); statErr == nil {
		s.debugLog("Research file exists at: %s", researchPath)
	}

	s.debugLog("ExecutePlan completed successfully")
	return nil
}

// ExecuteTasks runs the tasks stage for an existing spec.
// specNameArg: spec name or empty string to auto-detect from git branch
// prompt: optional custom prompt to pass to the tasks command
func (s *StageExecutor) ExecuteTasks(specNameArg string, prompt string) error {
	specName, err := s.resolveSpecName(specNameArg)
	if err != nil {
		return fmt.Errorf("resolving spec name: %w", err)
	}

	s.debugLog("ExecuteTasks called for spec: %s, prompt: %s", specName, prompt)

	command := s.buildTasksCommand(prompt)

	result, err := s.executor.ExecuteStage(
		specName,
		StageTasks,
		command,
		ValidateTasksSchema,
	)

	if err != nil {
		totalAttempts := result.RetryCount + 1
		if result.Exhausted {
			return fmt.Errorf("tasks stage exhausted retries after %d total attempts: %w",
				totalAttempts, err)
		}
		return fmt.Errorf("tasks failed after %d total attempts (%d retries): %w",
			totalAttempts, result.RetryCount, err)
	}

	s.debugLog("ExecuteTasks completed successfully")
	return nil
}

// resolveSpecName resolves the spec name from argument or auto-detection.
func (s *StageExecutor) resolveSpecName(specNameArg string) (string, error) {
	if specNameArg != "" {
		return specNameArg, nil
	}

	// Auto-detect current spec
	metadata, err := spec.DetectCurrentSpec(s.specsDir)
	if err != nil {
		return "", fmt.Errorf("detecting current spec: %w", err)
	}

	return fmt.Sprintf("%s-%s", metadata.Number, metadata.Name), nil
}

// buildPlanCommand constructs the plan command with optional prompt.
func (s *StageExecutor) buildPlanCommand(prompt string) string {
	if prompt != "" {
		return fmt.Sprintf("/autospec.plan \"%s\"", prompt)
	}
	return "/autospec.plan"
}

// buildTasksCommand constructs the tasks command with optional prompt.
func (s *StageExecutor) buildTasksCommand(prompt string) string {
	if prompt != "" {
		return fmt.Sprintf("/autospec.tasks \"%s\"", prompt)
	}
	return "/autospec.tasks"
}

// Compile-time interface compliance check.
var _ StageExecutorInterface = (*StageExecutor)(nil)
