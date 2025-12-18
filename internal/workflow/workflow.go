// autospec - Spec-Driven Development Automation
// Author: Ariel Frischer
// Source: https://github.com/ariel-frischer/autospec

// Package workflow provides workflow orchestration for the autospec CLI.
// This file contains the WorkflowOrchestrator which coordinates between specialized
// executor components (StageExecutor, PhaseExecutor, TaskExecutor) for different workflow stages.
// Related: internal/workflow/stage_executor.go, internal/workflow/phase_executor.go, internal/workflow/task_executor.go
// Tags: workflow, orchestrator, coordination, delegation
package workflow

import (
	"fmt"
	"path/filepath"

	"github.com/ariel-frischer/autospec/internal/config"
	"github.com/ariel-frischer/autospec/internal/spec"
	"github.com/ariel-frischer/autospec/internal/validation"
)

// WorkflowOrchestrator manages the complete specify → plan → tasks workflow.
// It coordinates between specialized executor components for different workflow stages.
// The orchestrator contains only coordination and delegation logic - all execution
// logic is delegated to the injected executor interfaces.
//
// Design: The orchestrator follows the Strategy pattern, delegating execution to
// specialized executor types. This enables:
// - Isolated unit testing with mock executors
// - Single responsibility (coordination only)
// - Easy extension for new execution strategies
type WorkflowOrchestrator struct {
	// Executor is the underlying command executor for Claude CLI invocations.
	Executor *Executor
	// Config holds the application configuration.
	Config *config.Configuration
	// SpecsDir is the base directory for spec storage (e.g., "specs/").
	SpecsDir string
	// SkipPreflight disables pre-flight checks when true.
	SkipPreflight bool
	// Debug enables debug logging when true.
	Debug bool
	// PreflightChecker is injectable for testing (nil uses default).
	PreflightChecker PreflightChecker

	// Executor interfaces for dependency injection.
	// These are always set by constructors - never nil during normal operation.
	stageExecutor StageExecutorInterface // Handles specify, plan, tasks stages
	phaseExecutor PhaseExecutorInterface // Handles phase-based implementation
	taskExecutor  TaskExecutorInterface  // Handles task-level implementation
}

// debugLog prints a debug message if debug mode is enabled
func (w *WorkflowOrchestrator) debugLog(format string, args ...interface{}) {
	if w.Debug {
		fmt.Printf("[DEBUG] "+format+"\n", args...)
	}
}

// NewWorkflowOrchestrator creates a new workflow orchestrator from configuration.
// This constructor creates default implementations for all executor interfaces,
// ensuring the orchestrator always delegates to specialized executors.
// For dependency injection (testing), use NewWorkflowOrchestratorWithExecutors.
func NewWorkflowOrchestrator(cfg *config.Configuration) *WorkflowOrchestrator {
	claude := &ClaudeExecutor{
		ClaudeCmd:       cfg.ClaudeCmd,
		ClaudeArgs:      cfg.ClaudeArgs,
		CustomClaudeCmd: cfg.CustomClaudeCmd,
		Timeout:         cfg.Timeout,
	}

	executor := &Executor{
		Claude:      claude,
		StateDir:    cfg.StateDir,
		SpecsDir:    cfg.SpecsDir,
		MaxRetries:  cfg.MaxRetries,
		TotalStages: 3,     // Default to 3 stages (specify, plan, tasks)
		Debug:       false, // Will be set by CLI command
	}

	// Create default executor implementations
	stageExec := NewStageExecutor(executor, cfg.SpecsDir, false)
	phaseExec := NewPhaseExecutor(executor, cfg.SpecsDir, false)
	taskExec := NewTaskExecutor(executor, cfg.SpecsDir, false)

	return &WorkflowOrchestrator{
		Executor:      executor,
		Config:        cfg,
		SpecsDir:      cfg.SpecsDir,
		SkipPreflight: cfg.SkipPreflight,
		stageExecutor: stageExec,
		phaseExecutor: phaseExec,
		taskExecutor:  taskExec,
	}
}

// ExecutorOptions holds optional executor interfaces for dependency injection.
// All fields are optional; nil values cause the orchestrator to use default implementations.
type ExecutorOptions struct {
	StageExecutor StageExecutorInterface
	PhaseExecutor PhaseExecutorInterface
	TaskExecutor  TaskExecutorInterface
}

// NewWorkflowOrchestratorWithExecutors creates a workflow orchestrator with injected executors.
// This constructor enables dependency injection for testing and modular composition.
// Pass nil for any executor to use the default implementation created by NewWorkflowOrchestrator.
//
// Example usage for testing:
//
//	mockStage := &MockStageExecutor{}
//	orch := NewWorkflowOrchestratorWithExecutors(cfg, ExecutorOptions{
//	    StageExecutor: mockStage,
//	})
func NewWorkflowOrchestratorWithExecutors(cfg *config.Configuration, opts ExecutorOptions) *WorkflowOrchestrator {
	orch := NewWorkflowOrchestrator(cfg)
	// Override with provided executors, keeping defaults for nil values
	if opts.StageExecutor != nil {
		orch.stageExecutor = opts.StageExecutor
	}
	if opts.PhaseExecutor != nil {
		orch.phaseExecutor = opts.PhaseExecutor
	}
	if opts.TaskExecutor != nil {
		orch.taskExecutor = opts.TaskExecutor
	}
	return orch
}

// RunCompleteWorkflow executes the full specify → plan → tasks workflow
func (w *WorkflowOrchestrator) RunCompleteWorkflow(featureDescription string) error {
	if err := w.runPreflightIfNeeded(); err != nil {
		return fmt.Errorf("preflight checks failed: %w", err)
	}

	specName, err := w.executeSpecifyPlanTasks(featureDescription, 3)
	if err != nil {
		return fmt.Errorf("executing specify-plan-tasks workflow: %w", err)
	}

	fmt.Println("Workflow completed successfully!")
	fmt.Printf("Spec: specs/%s/\n", specName)
	fmt.Println("Next: autospec implement")

	return nil
}

// RunFullWorkflow executes the complete specify → plan → tasks → implement workflow
func (w *WorkflowOrchestrator) RunFullWorkflow(featureDescription string, resume bool) error {
	// Set total stages for full workflow
	w.Executor.TotalStages = 4

	if err := w.runPreflightIfNeeded(); err != nil {
		return fmt.Errorf("preflight checks failed: %w", err)
	}

	// Execute specify → plan → tasks stages
	specName, err := w.executeSpecifyPlanTasks(featureDescription, 4)
	if err != nil {
		return fmt.Errorf("executing specify-plan-tasks workflow: %w", err)
	}

	// Execute implement stage
	if err := w.executeImplementStage(specName, featureDescription, resume); err != nil {
		return fmt.Errorf("executing implement stage: %w", err)
	}

	// Print success summary
	w.printFullWorkflowSummary(specName)
	return nil
}

// runPreflightIfNeeded runs preflight checks if enabled
func (w *WorkflowOrchestrator) runPreflightIfNeeded() error {
	if ShouldRunPreflightChecks(w.SkipPreflight) {
		return w.runPreflightChecks()
	}
	return nil
}

// executeSpecifyPlanTasks runs specify, plan, and tasks stages sequentially.
// Delegates to StageExecutor for all stage execution.
func (w *WorkflowOrchestrator) executeSpecifyPlanTasks(featureDescription string, totalStages int) (string, error) {
	// Stage 1: Specify
	fmt.Printf("[Stage 1/%d] Specify...\n", totalStages)
	fmt.Printf("Executing: /autospec.specify \"%s\"\n", featureDescription)

	specName, err := w.stageExecutor.ExecuteSpecify(featureDescription)
	if err != nil {
		return "", fmt.Errorf("specify stage failed: %w", err)
	}
	fmt.Printf("✓ Created specs/%s/spec.yaml\n\n", specName)

	// Stage 2: Plan
	fmt.Printf("[Stage 2/%d] Plan...\n", totalStages)
	fmt.Println("Executing: /autospec.plan")

	if err := w.stageExecutor.ExecutePlan(specName, ""); err != nil {
		return "", fmt.Errorf("plan stage failed: %w", err)
	}
	fmt.Printf("✓ Created specs/%s/plan.yaml\n\n", specName)

	// Stage 3: Tasks
	fmt.Printf("[Stage 3/%d] Tasks...\n", totalStages)
	fmt.Println("Executing: /autospec.tasks")

	if err := w.stageExecutor.ExecuteTasks(specName, ""); err != nil {
		return "", fmt.Errorf("tasks stage failed: %w", err)
	}
	fmt.Printf("✓ Created specs/%s/tasks.yaml\n\n", specName)

	return specName, nil
}

// executeImplementStage runs the implement stage with resume support
func (w *WorkflowOrchestrator) executeImplementStage(specName, featureDescription string, resume bool) error {
	fmt.Println("[Stage 4/4] Implement...")
	fmt.Println("Executing: /autospec.implement")
	w.debugLog("Starting implement stage for spec: %s", specName)

	command := w.buildImplementCommand(resume)
	result, err := w.Executor.ExecuteStage(specName, StageImplement, command, w.validateTasksCompleteFunc)

	w.debugLog("ExecuteStage returned - result: %+v, err: %v", result, err)

	if err != nil {
		return w.handleImplementError(result, featureDescription, err)
	}

	w.debugLog("Implement stage completed successfully")
	return nil
}

// buildImplementCommand constructs the implement command with optional resume flag
func (w *WorkflowOrchestrator) buildImplementCommand(resume bool) string {
	command := "/autospec.implement"
	if resume {
		command += " --resume"
		w.debugLog("Resume flag enabled")
	}
	w.debugLog("Calling ExecuteStage with command: %s", command)
	return command
}

// validateTasksCompleteFunc is a validation function for implement stage
func (w *WorkflowOrchestrator) validateTasksCompleteFunc(specDir string) error {
	w.debugLog("Running validation function for spec dir: %s", specDir)
	tasksPath := validation.GetTasksFilePath(specDir)
	w.debugLog("Validating tasks at: %s", tasksPath)
	validationErr := w.Executor.ValidateTasksComplete(tasksPath)
	w.debugLog("Validation result: %v", validationErr)
	return validationErr
}

// handleImplementError handles implement stage errors including retry exhaustion
func (w *WorkflowOrchestrator) handleImplementError(result *StageResult, featureDescription string, err error) error {
	w.debugLog("Implement stage failed with error: %v", err)
	if result.Exhausted {
		w.debugLog("Retries exhausted")
		fmt.Println("\nImplementation paused.")
		fmt.Printf("To resume: autospec full \"%s\" --resume\n", featureDescription)
		return fmt.Errorf("implementation stage exhausted retries: %w", err)
	}
	return fmt.Errorf("implementation failed: %w", err)
}

// printFullWorkflowSummary prints the completion summary for full workflow
func (w *WorkflowOrchestrator) printFullWorkflowSummary(specName string) {
	fmt.Println("\n✓ All tasks completed!")
	fmt.Println()

	specDir := filepath.Join(w.SpecsDir, specName)
	tasksPath := validation.GetTasksFilePath(specDir)
	stats, statsErr := validation.GetTaskStats(tasksPath)
	if statsErr == nil && stats.TotalTasks > 0 {
		fmt.Println("Task Summary:")
		fmt.Print(validation.FormatTaskSummary(stats))
		fmt.Println()
	}

	// Mark spec as completed
	markSpecCompletedAndPrint(specDir)

	fmt.Println("Completed 4 workflow stage(s): specify → plan → tasks → implement")
	fmt.Printf("Spec: specs/%s/\n", specName)
	w.debugLog("RunFullWorkflow exiting normally")
}

// runPreflightChecks runs pre-flight validation and handles user interaction.
// Uses the injected PreflightChecker if present, otherwise uses the default implementation.
func (w *WorkflowOrchestrator) runPreflightChecks() error {
	fmt.Println("Running pre-flight checks...")

	// Use injected checker or default
	checker := w.getPreflightChecker()

	result, err := checker.RunChecks()
	if err != nil {
		return fmt.Errorf("pre-flight checks failed: %w", err)
	}

	if !result.Passed {
		if len(result.FailedChecks) > 0 {
			for _, check := range result.FailedChecks {
				fmt.Printf("✗ %s\n", check)
			}
		}

		if result.WarningMessage != "" {
			// Prompt user to continue
			shouldContinue, err := checker.PromptUser(result.WarningMessage)
			if err != nil {
				return fmt.Errorf("prompting user to continue: %w", err)
			}
			if !shouldContinue {
				return fmt.Errorf("pre-flight checks failed, user aborted")
			}
		} else {
			// Critical failures (missing CLI tools)
			return fmt.Errorf("pre-flight checks failed")
		}
	} else {
		fmt.Println("✓ claude CLI found")
		fmt.Println("✓ specify CLI found")
		fmt.Println("✓ .claude/commands/ directory exists")
		fmt.Println("✓ .autospec/ directory exists")
	}

	fmt.Println()
	return nil
}

// getPreflightChecker returns the injected PreflightChecker or a default one.
// This ensures nil-safety: existing code works unchanged with nil checker.
func (w *WorkflowOrchestrator) getPreflightChecker() PreflightChecker {
	if w.PreflightChecker != nil {
		return w.PreflightChecker
	}
	return NewDefaultPreflightChecker()
}

// resolveSpecName resolves the spec name from argument or auto-detection.
func (w *WorkflowOrchestrator) resolveSpecName(specNameArg string) (string, error) {
	if specNameArg != "" {
		return specNameArg, nil
	}

	// Auto-detect current spec
	metadata, err := spec.DetectCurrentSpec(w.SpecsDir)
	if err != nil {
		return "", fmt.Errorf("detecting current spec: %w", err)
	}

	return fmt.Sprintf("%s-%s", metadata.Number, metadata.Name), nil
}

// buildCommand constructs a command with optional prompt.
func (w *WorkflowOrchestrator) buildCommand(baseCmd, prompt string) string {
	if prompt != "" {
		return fmt.Sprintf("%s \"%s\"", baseCmd, prompt)
	}
	return baseCmd
}

// printExecuting prints the executing message for a command.
func (w *WorkflowOrchestrator) printExecuting(baseCmd, prompt string) {
	if prompt != "" {
		fmt.Printf("Executing: %s \"%s\"\n", baseCmd, prompt)
	} else {
		fmt.Printf("Executing: %s\n", baseCmd)
	}
}


// ExecuteSpecify runs only the specify stage.
// Delegates to the StageExecutor for execution.
func (w *WorkflowOrchestrator) ExecuteSpecify(featureDescription string) (string, error) {
	fmt.Printf("Executing: /autospec.specify \"%s\"\n", featureDescription)

	specName, err := w.stageExecutor.ExecuteSpecify(featureDescription)
	if err != nil {
		return "", err
	}

	fmt.Printf("✓ Created specs/%s/spec.yaml\n\n", specName)
	fmt.Println("Next: autospec plan")

	return specName, nil
}

// ExecutePlan runs only the plan stage for a detected or specified spec.
// Delegates to the StageExecutor for execution.
func (w *WorkflowOrchestrator) ExecutePlan(specNameArg string, prompt string) error {
	specName, err := w.resolveSpecName(specNameArg)
	if err != nil {
		return fmt.Errorf("resolving spec name: %w", err)
	}

	if prompt != "" {
		fmt.Printf("Executing: /autospec.plan \"%s\"\n", prompt)
	} else {
		fmt.Println("Executing: /autospec.plan")
	}

	if err := w.stageExecutor.ExecutePlan(specName, prompt); err != nil {
		return fmt.Errorf("executing plan stage: %w", err)
	}

	fmt.Printf("✓ Created specs/%s/plan.yaml\n\n", specName)
	fmt.Println("Next: autospec tasks")

	return nil
}

// ExecuteTasks runs only the tasks stage for a detected or specified spec.
// Delegates to the StageExecutor for execution.
func (w *WorkflowOrchestrator) ExecuteTasks(specNameArg string, prompt string) error {
	specName, err := w.resolveSpecName(specNameArg)
	if err != nil {
		return fmt.Errorf("resolving spec name: %w", err)
	}

	if prompt != "" {
		fmt.Printf("Executing: /autospec.tasks \"%s\"\n", prompt)
	} else {
		fmt.Println("Executing: /autospec.tasks")
	}

	if err := w.stageExecutor.ExecuteTasks(specName, prompt); err != nil {
		return fmt.Errorf("executing tasks stage: %w", err)
	}

	fmt.Printf("✓ Created specs/%s/tasks.yaml\n\n", specName)
	fmt.Println("Next: autospec implement")

	return nil
}

// ExecuteImplement runs the implementation stage with optional prompt
func (w *WorkflowOrchestrator) ExecuteImplement(specNameArg string, prompt string, resume bool, phaseOpts PhaseExecutionOptions) error {
	var specName string
	var metadata *spec.Metadata
	var err error

	if specNameArg != "" {
		specName = specNameArg
		// Load metadata for this spec
		metadata, err = spec.GetSpecMetadata(w.SpecsDir, specName)
		if err != nil {
			return fmt.Errorf("failed to load spec metadata: %w", err)
		}
	} else {
		// Auto-detect current spec
		metadata, err = spec.DetectCurrentSpec(w.SpecsDir)
		if err != nil {
			return fmt.Errorf("failed to detect current spec: %w", err)
		}
		// Use full spec directory name (e.g., "003-command-timeout")
		specName = fmt.Sprintf("%s-%s", metadata.Number, metadata.Name)
	}

	// Dispatch to appropriate execution mode based on phase options
	switch phaseOpts.Mode() {
	case ModeAllTasks:
		return w.ExecuteImplementWithTasks(specName, metadata, prompt, phaseOpts.FromTask)
	case ModeAllPhases:
		return w.ExecuteImplementWithPhases(specName, metadata, prompt, resume)
	case ModeSinglePhase:
		return w.ExecuteImplementSinglePhase(specName, metadata, prompt, phaseOpts.SinglePhase)
	case ModeFromPhase:
		return w.ExecuteImplementFromPhase(specName, metadata, prompt, phaseOpts.FromPhase)
	default:
		// Default mode: single session (backward compatible)
		return w.executeImplementDefault(specName, metadata, prompt, resume)
	}
}

// executeImplementDefault executes implementation in a single Claude session (backward compatible)
func (w *WorkflowOrchestrator) executeImplementDefault(specName string, metadata *spec.Metadata, prompt string, resume bool) error {
	// Check progress
	fmt.Printf("Progress: checking tasks...\n\n")

	// Build command with optional prompt
	command := "/autospec.implement"
	if resume {
		command += " --resume"
	}
	if prompt != "" {
		command = fmt.Sprintf("/autospec.implement \"%s\"", prompt)
		if resume {
			// If both resume and prompt, append resume after prompt
			command = fmt.Sprintf("/autospec.implement --resume \"%s\"", prompt)
		}
	}

	if prompt != "" {
		fmt.Printf("Executing: /autospec.implement \"%s\"\n", prompt)
	} else {
		fmt.Println("Executing: /autospec.implement")
	}

	result, err := w.Executor.ExecuteStage(
		specName,
		StageImplement,
		command,
		func(specDir string) error {
			tasksPath := validation.GetTasksFilePath(specDir)
			return w.Executor.ValidateTasksComplete(tasksPath)
		},
	)

	if err != nil {
		if result.Exhausted {
			// Generate continuation prompt
			fmt.Println("\nImplementation paused.")
			fmt.Println("To resume: autospec implement --resume")
			return fmt.Errorf("implementation stage exhausted retries: %w", err)
		}
		return fmt.Errorf("implementation failed: %w", err)
	}

	// Show task completion stats
	fmt.Println("\n✓ All tasks completed!")
	fmt.Println()
	tasksPath := validation.GetTasksFilePath(metadata.Directory)
	stats, statsErr := validation.GetTaskStats(tasksPath)
	if statsErr == nil && stats.TotalTasks > 0 {
		fmt.Println("Task Summary:")
		fmt.Print(validation.FormatTaskSummary(stats))
	}

	return nil
}

// ExecuteImplementWithPhases runs each phase in a separate Claude session.
// Delegates to PhaseExecutor for execution.
func (w *WorkflowOrchestrator) ExecuteImplementWithPhases(specName string, metadata *spec.Metadata, prompt string, resume bool) error {
	specDir := filepath.Join(w.SpecsDir, specName)
	tasksPath := validation.GetTasksFilePath(specDir)

	phases, err := validation.GetPhaseInfo(tasksPath)
	if err != nil {
		return fmt.Errorf("getting phase info: %w", err)
	}

	if len(phases) == 0 {
		return fmt.Errorf("no phases found in tasks.yaml")
	}

	firstIncomplete, _, err := validation.GetFirstIncompletePhase(tasksPath)
	if err != nil {
		return fmt.Errorf("checking phase completion: %w", err)
	}

	if firstIncomplete == 0 {
		fmt.Println("✓ All phases already complete!")
		return nil
	}

	if firstIncomplete > 1 {
		fmt.Printf("Phases 1-%d complete, starting from phase %d\n\n", firstIncomplete-1, firstIncomplete)
	}

	return w.phaseExecutor.ExecutePhaseLoop(specName, tasksPath, phases, firstIncomplete, len(phases), prompt)
}

// ExecuteImplementSinglePhase runs only a specific phase.
// Delegates to PhaseExecutor for execution.
func (w *WorkflowOrchestrator) ExecuteImplementSinglePhase(specName string, metadata *spec.Metadata, prompt string, phaseNumber int) error {
	specDir := filepath.Join(w.SpecsDir, specName)
	tasksPath := validation.GetTasksFilePath(specDir)

	totalPhases, err := validation.GetTotalPhases(tasksPath)
	if err != nil {
		return fmt.Errorf("getting total phases: %w", err)
	}

	if phaseNumber < 1 || phaseNumber > totalPhases {
		return fmt.Errorf("phase %d is out of range (valid: 1-%d)", phaseNumber, totalPhases)
	}

	return w.phaseExecutor.ExecuteSinglePhase(specName, phaseNumber, prompt)
}

// ExecuteImplementFromPhase runs phases starting from the specified phase.
// Delegates to PhaseExecutor for execution.
func (w *WorkflowOrchestrator) ExecuteImplementFromPhase(specName string, metadata *spec.Metadata, prompt string, startPhase int) error {
	specDir := filepath.Join(w.SpecsDir, specName)
	tasksPath := validation.GetTasksFilePath(specDir)

	totalPhases, err := validation.GetTotalPhases(tasksPath)
	if err != nil {
		return fmt.Errorf("getting total phases: %w", err)
	}

	if startPhase < 1 || startPhase > totalPhases {
		return fmt.Errorf("phase %d is out of range (valid: 1-%d)", startPhase, totalPhases)
	}

	phases, err := validation.GetPhaseInfo(tasksPath)
	if err != nil {
		return fmt.Errorf("getting phase info: %w", err)
	}

	fmt.Printf("Starting from phase %d of %d\n\n", startPhase, totalPhases)

	return w.phaseExecutor.ExecutePhaseLoop(specName, tasksPath, phases, startPhase, totalPhases, prompt)
}

// ExecuteImplementWithTasks runs each task in a separate Claude session.
// Delegates to TaskExecutor for execution.
func (w *WorkflowOrchestrator) ExecuteImplementWithTasks(specName string, metadata *spec.Metadata, prompt string, fromTask string) error {
	specDir := filepath.Join(w.SpecsDir, specName)
	tasksPath := validation.GetTasksFilePath(specDir)

	orderedTasks, startIdx, totalTasks, err := w.taskExecutor.PrepareTaskExecution(tasksPath, fromTask)
	if err != nil {
		return fmt.Errorf("preparing task execution: %w", err)
	}

	if startIdx > 0 {
		fmt.Printf("Starting from task %s (task %d of %d)\n\n", fromTask, startIdx+1, totalTasks)
	}

	return w.taskExecutor.ExecuteTaskLoop(specName, tasksPath, orderedTasks, startIdx, totalTasks, prompt)
}

// markSpecCompletedAndPrint marks the spec as completed and prints the result.
// This is a package-level function used by executors for consistent completion marking.
func markSpecCompletedAndPrint(specDir string) {
	result, err := spec.MarkSpecCompleted(specDir)
	if err != nil {
		fmt.Printf("Warning: could not update spec.yaml status: %v\n", err)
		return
	}

	if result.Updated {
		fmt.Printf("Updated spec.yaml: %s → %s\n", result.PreviousStatus, result.NewStatus)
	}
}

// ExecuteConstitution runs the constitution stage with optional prompt
// Constitution creates or updates the project constitution file
func (w *WorkflowOrchestrator) ExecuteConstitution(prompt string) error {
	// Build command with optional prompt
	command := "/autospec.constitution"
	if prompt != "" {
		command = fmt.Sprintf("/autospec.constitution \"%s\"", prompt)
	}

	if prompt != "" {
		fmt.Printf("Executing: /autospec.constitution \"%s\"\n", prompt)
	} else {
		fmt.Println("Executing: /autospec.constitution")
	}

	// Constitution stage doesn't require spec detection - it works at project level
	result, err := w.Executor.ExecuteStage(
		"", // No spec name needed for constitution
		StageConstitution,
		command,
		func(specDir string) error {
			// Constitution doesn't produce tracked artifacts
			// It modifies .autospec/memory/constitution.yaml
			return nil
		},
	)

	if err != nil {
		if result.Exhausted {
			return fmt.Errorf("constitution stage exhausted retries: %w", err)
		}
		return fmt.Errorf("constitution failed: %w", err)
	}

	fmt.Println("\n✓ Constitution updated!")
	return nil
}

// ExecuteClarify runs the clarify stage with optional prompt.
// Clarify refines the specification by asking targeted clarification questions.
func (w *WorkflowOrchestrator) ExecuteClarify(specNameArg string, prompt string) error {
	specName, err := w.resolveSpecName(specNameArg)
	if err != nil {
		return fmt.Errorf("resolving spec name: %w", err)
	}

	command := w.buildCommand("/autospec.clarify", prompt)
	w.printExecuting("/autospec.clarify", prompt)

	result, err := w.Executor.ExecuteStage(specName, StageClarify, command,
		func(specDir string) error { return validation.ValidateSpecFile(specDir) })

	if err != nil {
		if result.Exhausted {
			return fmt.Errorf("clarify stage exhausted retries: %w", err)
		}
		return fmt.Errorf("clarify failed: %w", err)
	}

	fmt.Printf("\n✓ Clarification complete for specs/%s/\n", specName)
	return nil
}

// ExecuteChecklist runs the checklist stage with optional prompt.
// Checklist generates a custom checklist for the current feature.
func (w *WorkflowOrchestrator) ExecuteChecklist(specNameArg string, prompt string) error {
	specName, err := w.resolveSpecName(specNameArg)
	if err != nil {
		return fmt.Errorf("resolving spec name: %w", err)
	}

	command := w.buildCommand("/autospec.checklist", prompt)
	w.printExecuting("/autospec.checklist", prompt)

	result, err := w.Executor.ExecuteStage(specName, StageChecklist, command,
		func(specDir string) error { return nil })

	if err != nil {
		if result.Exhausted {
			return fmt.Errorf("checklist stage exhausted retries: %w", err)
		}
		return fmt.Errorf("checklist failed: %w", err)
	}

	fmt.Printf("\n✓ Checklist generated for specs/%s/\n", specName)
	return nil
}

// ExecuteAnalyze runs the analyze stage with optional prompt.
// Analyze performs cross-artifact consistency and quality analysis.
func (w *WorkflowOrchestrator) ExecuteAnalyze(specNameArg string, prompt string) error {
	specName, err := w.resolveSpecName(specNameArg)
	if err != nil {
		return fmt.Errorf("resolving spec name: %w", err)
	}

	command := w.buildCommand("/autospec.analyze", prompt)
	w.printExecuting("/autospec.analyze", prompt)

	result, err := w.Executor.ExecuteStage(specName, StageAnalyze, command,
		func(specDir string) error { return nil })

	if err != nil {
		if result.Exhausted {
			return fmt.Errorf("analyze stage exhausted retries: %w", err)
		}
		return fmt.Errorf("analyze failed: %w", err)
	}

	fmt.Printf("\n✓ Analysis complete for specs/%s/\n", specName)
	return nil
}
