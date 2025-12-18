# Orchestrator Schema Validation

**Priority:** P0 (HIGH)
**Status:** Open
**Effort:** Medium (2-3 days)

## Quick Start

```bash
autospec specify "$(cat .dev/tasks/orchestrator-schema-validation.md)"
```

## Problem Statement

The Go orchestrator only validates file existence after each stage completes, not schema compliance. This affects ALL stage execution paths:

- `autospec run -spti` / `autospec run -a`
- `autospec specify` / `autospec plan` / `autospec tasks`
- `autospec prep`
- `autospec implement`

Claude's slash commands include `autospec artifact` validation calls, but:
1. Claude might skip the validation step
2. Output might pass YAML syntax but fail schema validation
3. Orchestrator doesn't catch this before proceeding to next stage

## Current Behavior

```go
// internal/workflow/executor.go
func (e *Executor) ValidateSpec(specDir string) error {
    return validation.ValidateSpecFile(specDir)  // Only checks file EXISTS
}

// internal/validation/validation.go
func ValidateSpecFile(specDir string) error {
    yamlPath := filepath.Join(specDir, "spec.yaml")
    if _, err := os.Stat(yamlPath); err == nil {
        return nil  // File exists = valid. NO SCHEMA CHECK!
    }
    // ...
}

// internal/workflow/workflow.go - executePlan passes ValidatePlan as callback
result, err := w.Executor.ExecuteStage(
    specName,
    StagePlan,
    command,
    w.Executor.ValidatePlan,  // This only checks file exists!
)
```

## Gap

After Claude generates artifacts, the orchestrator doesn't verify:
- Required fields present (feature.branch, user_stories, requirements)
- Enum values valid (priority: P1, status: Draft, task status: Pending/InProgress/Completed/Blocked)
- Cross-references valid (in tasks.yaml: dependencies reference existing task IDs)
- Type correctness (arrays vs strings, nested structures)

## Solution

Replace file-existence checks with full schema validation using existing artifact validators.

### Affected Code Paths

| File | Function | Current | Proposed |
|------|----------|---------|----------|
| `executor.go` | `ValidateSpec()` | `ValidateSpecFile()` | `ValidateSpecSchema()` |
| `executor.go` | `ValidatePlan()` | `ValidatePlanFile()` | `ValidatePlanSchema()` |
| `executor.go` | `ValidateTasks()` | `ValidateTasksFile()` | `ValidateTasksSchema()` |
| `workflow.go` | `executePlan()` | passes `ValidatePlan` | passes `ValidatePlanSchema` |
| `workflow.go` | `executeTasks()` | passes `ValidateTasks` | passes `ValidateTasksSchema` |
| `run.go` | `executeStages()` | chains stages | same, but validators now check schema |

### New Validation Functions

```go
// internal/workflow/executor.go

// ValidateSpecSchema validates spec.yaml against full schema
func (e *Executor) ValidateSpecSchema(specDir string) error {
    specPath := filepath.Join(specDir, "spec.yaml")
    validator := validation.NewSpecValidator()
    result := validator.Validate(specPath)
    if !result.Valid {
        return fmt.Errorf("spec.yaml schema validation failed:\n%s", result.FormatErrors())
    }
    return nil
}

// ValidatePlanSchema validates plan.yaml against full schema
func (e *Executor) ValidatePlanSchema(specDir string) error {
    planPath := filepath.Join(specDir, "plan.yaml")
    validator := validation.NewPlanValidator()
    result := validator.Validate(planPath)
    if !result.Valid {
        return fmt.Errorf("plan.yaml schema validation failed:\n%s", result.FormatErrors())
    }
    return nil
}

// ValidateTasksSchema validates tasks.yaml against full schema
func (e *Executor) ValidateTasksSchema(specDir string) error {
    tasksPath := filepath.Join(specDir, "tasks.yaml")
    validator := validation.NewTasksValidator()
    result := validator.Validate(tasksPath)
    if !result.Valid {
        return fmt.Errorf("tasks.yaml schema validation failed:\n%s", result.FormatErrors())
    }
    return nil
}
```

### Wire into ExecuteStage

```go
// internal/workflow/workflow.go

func (w *WorkflowOrchestrator) executePlan(specName string, prompt string) error {
    command := "/autospec.plan"
    if prompt != "" {
        command = fmt.Sprintf("/autospec.plan \"%s\"", prompt)
    }

    result, err := w.Executor.ExecuteStage(
        specName,
        StagePlan,
        command,
        w.Executor.ValidatePlanSchema,  // CHANGED: Now validates schema
    )
    // ...
}
```

## Critical: Retry Behavior with Error Injection

On schema validation failure with retries remaining, the NEXT Claude invocation MUST include:
1. The full validation error message so Claude knows exactly what to fix
2. Retry number indicator (e.g., "RETRY 2/3")
3. Instruction to fix the schema errors

### Retry Prompt Injection

```go
// When retrying after schema validation failure:
func buildRetryCommand(stage Stage, retryNum, maxRetries int, validationErrors string) string {
    return fmt.Sprintf("/%s RETRY %d/%d\n\nPrevious attempt failed schema validation:\n%s\n\nFix these schema errors and regenerate the artifact.",
        stage.Command(), retryNum, maxRetries, validationErrors)
}

// Example output:
// /autospec.plan RETRY 2/3
//
// Previous attempt failed schema validation:
//   - line 5: missing required field 'feature.branch'
//   - line 8: invalid enum value 'status=pending' (expected: Draft|Review|Approved|Implemented)
//   - line 23: wrong type for 'user_stories' (expected: array, got: string)
//
// Fix these schema errors and regenerate the artifact.
```

### Slash Command Changes

Update `internal/commands/*.md` files to handle retry context:

```markdown
## Retry Context (if present)

$RETRY_INFO

$VALIDATION_ERRORS

If retry context is present above, focus on fixing the listed schema errors.
```

The Go code injects these variables when retrying:
- `$RETRY_INFO` = "RETRY 2/3" or empty on first attempt
- `$VALIDATION_ERRORS` = formatted error list or empty on first attempt

## Implementation Checklist

- [ ] Add `ValidateSpecSchema()` to executor.go
- [ ] Add `ValidatePlanSchema()` to executor.go
- [ ] Add `ValidateTasksSchema()` to executor.go
- [ ] Update `executePlan()` to use `ValidatePlanSchema`
- [ ] Update `executeTasks()` to use `ValidateTasksSchema`
- [ ] Update `executeSpecify()` to use `ValidateSpecSchema`
- [ ] Add `buildRetryCommand()` function for error injection
- [ ] Update `ExecuteStage()` to pass validation errors on retry
- [ ] Update `internal/commands/autospec.specify.md` with retry context section
- [ ] Update `internal/commands/autospec.plan.md` with retry context section
- [ ] Update `internal/commands/autospec.tasks.md` with retry context section
- [ ] Add tests for schema validation in executor
- [ ] Add tests for retry prompt injection
- [ ] Run `make test` and `make lint`

## Benefits

- Catches Claude's schema errors **programmatically** before proceeding to next stage
- Prevents cascading failures from malformed artifacts
- Same validation quality as `autospec artifact` CLI command
- **Claude gets actionable feedback** on retry instead of blind retry
- No more "plan failed because spec was invalid"

## Related Files

| File | Purpose |
|------|---------|
| `internal/workflow/executor.go` | Add new ValidateXxxSchema functions |
| `internal/workflow/workflow.go` | Wire schema validators into stage execution |
| `internal/validation/artifact_spec.go` | Existing SpecValidator |
| `internal/validation/artifact_plan.go` | Existing PlanValidator |
| `internal/validation/artifact_tasks.go` | Existing TasksValidator |
| `internal/commands/autospec.*.md` | Add retry context sections |
