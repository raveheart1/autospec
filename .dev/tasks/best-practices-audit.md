# Go Best Practices Audit

This document lists critical issues found in the autospec codebase based on the standards defined in `docs/go-best-practices.md`.

**Audit Date**: 2025-12-16
**Auditor**: Claude Code

---

## Summary

| Category | Issues Found | Severity |
|----------|-------------|----------|
| Long Functions (>40 lines) | 69 functions | HIGH |
| Missing t.Parallel() | 198 test cases in 44 files | MEDIUM |
| Slice-based tests (not map-based) | 25 test files | LOW |
| Unwrapped errors | 29+ instances | MEDIUM |
| Missing benchmark tests | 19 validation functions | MEDIUM |
| Missing package docs | 0 packages | OK |
| errors.New() instead of internal/errors | 6 instances | LOW |
| panic() in library code | 0 | OK |

---

## Critical Issues

### 1. Long Functions (>40 Lines) - PARTIAL DONE

**Best Practice**: "Keep functions short and focused (generally <40 lines)"

Found **69 functions** exceeding the 40-line guideline. Most critical violations:

#### Critical (>100 lines)- DONE

| File | Function | Lines |
|------|----------|-------|
| `internal/workflow/workflow.go` | `ExecuteImplementWithTasks` | 125 |
| `internal/workflow/executor.go` | `ExecutePhase` | 117 |
| `internal/cli/new_feature.go` | `runNewFeature` | 108 |
| `internal/cli/uninstall.go` | `runUninstall` | 110 |
| `internal/config/config.go` | `LoadWithOptions` | 105 |
| `internal/workflow/workflow.go` | `RunFullWorkflow` | 102 |

#### Serious (80-100 lines) - DONE

| File | Function | Lines |
|------|----------|-------|
| `internal/workflow/workflow.go` | `ExecuteImplementFromPhase` | 90 |
| `internal/cli/config.go` | `runConfigMigrate` | 90 |
| `internal/cli/update_agent_context.go` | `runUpdateAgentContext` | 87 |
| `internal/workflow/workflow.go` | `ExecuteImplementWithPhases` | 81 |
| `internal/git/git.go` | `GetAllBranches` | 80 |
| `internal/cli/init.go` | `runInit` | 79 |
| `internal/cli/run.go` | `executePhases` | 78 |

#### Significant (60-80 lines)

- `internal/cli/clean.go:runClean` (155 lines - most severe CLI violation)
- `internal/cli/prereqs.go:runPrereqs` (138 lines)
- `internal/cli/setup_plan.go:runSetupPlan` (94 lines)
- `internal/validation/tasks.go:ParseTasksByPhase` (75 lines)
- `internal/cli/update_task.go:runUpdateTask` (74 lines)
- `internal/config/validate.go:ValidateConfigValues` (72 lines)
- `internal/cli/version.go:printPrettyVersion` (72 lines)
- `internal/spec/spec.go:DetectCurrentSpec` (71 lines)
- `internal/agent/agent.go:updateRecentChanges` (69 lines)
- `internal/validation/autofix.go:FixArtifact` (69 lines)
- `internal/validation/artifact_plan.go:Validate` (65 lines)

**Recommendation**: Refactor into smaller helper functions. Priority targets:
1. `workflow.go` - Extract orchestration logic into focused helpers
2. `executor.go:ExecutePhase` - Break into pre-execute, execute, post-execute
3. CLI commands - Extract validation, execution, and output into separate functions

---

### 2. Missing t.Parallel() in Tests - DONE!

**Best Practice**: "Enable parallel execution" with `t.Parallel()` in test functions

Found **198 t.Run() calls** across **44 test files** missing `t.Parallel()`.

#### Most Affected Files

| File | Missing Calls |
|------|---------------|
| `internal/workflow/phase_context_test.go` | 21 |
| `internal/cli/init_test.go` | 12 |
| `internal/retry/retry_test.go` | 12 |
| `internal/workflow/executor_test.go` | 11 |
| `internal/errors/format_test.go` | 10 |
| `internal/workflow/claude_test.go` | 9 |
| `internal/validation/tasks_yaml_test.go` | 9 |
| `internal/errors/errors_test.go` | 8 |
| `internal/workflow/phase_config_test.go` | 8 |
| `internal/workflow/workflow_test.go` | 8 |

**Correct Pattern**:
```go
t.Run(name, func(t *testing.T) {
    t.Parallel()  // Add as first line
    // test code
})
```

**Reference**: `internal/validation/validation_test.go` shows correct implementation.

---

### 3. Slice-Based Test Cases Instead of Map-Based

**Best Practice**: "Use map[string]struct{} for named test cases"

Found **25 test files** using slice-based test cases (`tests := []struct`) instead of the project's standard map-based pattern.

#### Files to Convert

- `internal/cli/implement_test.go`
- `internal/cli/run_test.go`
- `internal/cli/new_feature_test.go`
- `internal/cli/artifact_test.go`
- `internal/cli/all_test.go`
- `internal/cli/alias_test.go`
- `internal/config/config_test.go`
- `internal/config/validate_test.go`
- `internal/workflow/workflow_test.go`
- `internal/workflow/phase_config_test.go`
- `internal/workflow/errors_test.go`
- `internal/workflow/claude_test.go`
- `internal/validation/schema_test.go`
- `internal/validation/autofix_test.go`
- `internal/yaml/validator_test.go`
- `internal/yaml/migrate_test.go`
- `internal/yaml/meta_test.go`
- `internal/errors/errors_test.go`
- `internal/health/health_test.go`
- `internal/spec/branch_test.go`
- `internal/progress/types_test.go`
- `internal/progress/terminal_test.go`
- `internal/progress/display_test.go`
- `internal/agent/types_test.go`
- `internal/agent/parse_test.go`

**Correct Pattern**:
```go
tests := map[string]struct {
    input   string
    wantErr bool
}{
    "valid input": {input: "foo", wantErr: false},
    "empty input": {input: "", wantErr: true},
}
```

---

### 4. Unwrapped Errors (Missing Context)

**Best Practice**: "Wrap errors with context at boundaries"

Found **29+ instances** of `return err` without wrapping.

#### Critical Locations

**Workflow Package** (most important - core orchestration):
- `workflow/workflow.go:75` - `RunCompleteWorkflow` returns preflight error unwrapped
- `workflow/workflow.go:126` - `RunFullWorkflow` returns preflight error unwrapped
- `workflow/workflow.go:393` - `executePlan` returns error unwrapped
- `workflow/workflow.go:427` - `executeTasks` returns error unwrapped
- `workflow/workflow.go:957,962,978` - `ExecuteImplementWithTasks` returns validation errors unwrapped
- `workflow/executor.go:262` - `ValidateTasksComplete` returns stats error unwrapped
- `workflow/claude.go:176` - `Execute` returns raw error at function end

**Retry Package**:
- `retry/retry.go:264` - `MarkPhaseComplete` returns load error unwrapped
- `retry/retry.go:407` - `MarkTaskComplete` returns load error unwrapped

**CLI Commands** (18 instances):
- `cli/implement.go:188`
- `cli/specify.go:82`
- `cli/all.go:93`
- `cli/plan.go:76`
- `cli/tasks.go:76`
- `cli/prep.go:74`
- `cli/analyze.go:101`
- `cli/clarify.go:86`
- `cli/checklist.go:85`
- `cli/prereqs.go:84`
- `cli/setup_plan.go:72,154,160,165`
- `cli/completion_install.go:231`
- `cli/artifact.go:254`
- `cli/update_agent_context.go:252`
- `cli/yaml_check.go:42`
- `cli/migrate_mdtoyaml.go:77`

**Correct Pattern**:
```go
if err != nil {
    return fmt.Errorf("loading phase state: %w", err)
}
```

---

### 5. Missing Benchmark Tests for Validation Functions

**Best Practice**: "Benchmark critical paths (validation functions must be <10ms)"

Found **19 validation functions** without benchmark tests despite the <10ms performance contract.

#### Functions Missing Benchmarks

| Function | Contract | Status |
|----------|----------|--------|
| `ValidateYAMLFile` | <100ms | MISSING |
| `ValidateArtifactFile` | <100ms | MISSING |
| `FixArtifact` | - | MISSING |
| `GenerateContinuationPrompt` | - | MISSING |
| `GetAllTasks` | - | MISSING |
| `GetTaskByID` | - | MISSING |
| `GetTasksInDependencyOrder` | - | MISSING |
| `GetTaskStats` | - | MISSING |
| `ParseTasksYAML` | - | MISSING |
| `ValidateTaskDependenciesMet` | - | MISSING |
| `IsPhaseComplete` | - | MISSING |
| `GetPhaseInfo` | - | MISSING |
| `GetFirstIncompletePhase` | - | MISSING |
| `GetActionablePhases` | - | MISSING |
| `ListIncompletePhasesWithTasks` | - | MISSING |
| `GetTotalPhases` | - | MISSING |
| `InferArtifactTypeFromFilename` | - | MISSING |
| `NewArtifactValidator` | - | MISSING |
| `FormatFixes` | - | MISSING |

**Coverage**: ~46% of validation functions have benchmarks (16 of 35+).

**Existing Benchmarks** (good examples to follow):
- `validation/validation_bench_test.go`
- `validation/artifact_bench_test.go`

---

### 6. Missing Package Documentation - DONE

**Best Practice**: "All exported types, functions, and constants" need doc comments

All **16 packages** now have `// Package` documentation.

| Package | Status |
|---------|--------|
| `internal/agent` | ✓ |
| `internal/clean` | ✓ |
| `internal/cli` | ✓ |
| `internal/commands` | ✓ |
| `internal/completion` | ✓ |
| `internal/config` | ✓ |
| `internal/errors` | ✓ |
| `internal/git` | ✓ |
| `internal/health` | ✓ |
| `internal/progress` | ✓ |
| `internal/retry` | ✓ |
| `internal/spec` | ✓ |
| `internal/uninstall` | ✓ |
| `internal/validation` | ✓ |
| `internal/workflow` | ✓ |
| `internal/yaml` | ✓ |

---

### 7. Using errors.New() Instead of internal/errors Package

**Best Practice**: "Use project error types for structured errors (internal/errors/)"

Found **6 instances** in `internal/progress/types.go` using `errors.New()`:

```go
// Lines 54-69 in PhaseInfo.Validate()
errors.New("phase name cannot be empty")
errors.New("phase number must be > 0")
errors.New("total phases must be > 0")
errors.New("phase number cannot exceed total phases")
errors.New("retry count cannot be negative")
errors.New("max retries cannot be negative")
```

**Recommendation**: Use `errors.NewArgumentError()` for validation errors.

---

## Compliance Summary

### Passing

- **No panic() in library code** - All 16 internal packages comply
- **go.mod dependencies documented** - All dependencies have comments explaining purpose
- **No circular dependencies detected**

### Needs Improvement

| Area | Current | Target |
|------|---------|--------|
| Functions <40 lines | 69 violations | 0 |
| t.Parallel() usage | 54% coverage | 100% |
| Map-based test cases | 46% (21/46 files) | 100% |
| Error wrapping | ~70% | 100% |
| Benchmark coverage | 46% | >80% |
| Package documentation | 100% (16/16) | 100% |

---

## Priority Action Items

### P0 - Critical (Address Immediately)

1. **Refactor `workflow.go`** - 5 functions >80 lines, core orchestration logic
2. **Refactor `executor.go:ExecutePhase`** - 117 lines, critical path
3. **Add error wrapping in workflow package** - Core error handling

### P1 - High (Next Sprint)

1. Add `t.Parallel()` to all 198 missing test cases
2. Convert 25 slice-based tests to map-based pattern
3. Add missing benchmarks for validation functions with <10ms contract

### P2 - Medium (Backlog)

1. Refactor oversized CLI command functions (clean, prereqs, init, etc.)
2. Replace `errors.New()` with `internal/errors` types

### P3 - Low (Technical Debt)

1. Add benchmarks for remaining validation utility functions
2. Review and document any remaining non-wrapped errors
