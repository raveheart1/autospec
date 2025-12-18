# Architecture Decoupling Analysis

**Date:** 2025-12-18
**Scope:** Full codebase architectural review for decoupling opportunities

## Overview

| Metric | Value |
|--------|-------|
| Non-test LOC | 21,390 |
| Test LOC | 45,788 (2:1 ratio) |
| Internal packages | 22 |
| CLI commands | 47 |

**Overall Assessment:** Good foundational architecture with clean dependencies, but suffers from two primary "god objects" that should be refactored for long-term maintainability.

---

## High Priority Issues

### 1. WorkflowOrchestrator God Object

**Location:** `internal/workflow/workflow.go` (1,285 LOC, 38 methods)

**Current Structure:**
```go
type WorkflowOrchestrator struct {
    Executor         *Executor
    Config           *config.Configuration
    SpecsDir         string
    SkipPreflight    bool
    Debug            bool
    PreflightChecker PreflightChecker
}
```

**Problems:**
- Single Responsibility Principle violated
- Orchestrates ALL workflow stages (specify → plan → tasks → implement)
- Handles phase execution, task execution, session management
- Manages preflight checks, validation, error handling
- Direct dependencies on: config, retry, spec, validation
- 10+ Execute variants for different execution modes

**Methods Span Multiple Concerns:**
```
Workflow orchestration:  RunCompleteWorkflow, RunFullWorkflow
Stage execution:         executeSpecifyPlanTasks, executeImplementStage
Phase management:        ExecuteImplementWithPhases, executePhaseLoop
Task management:         ExecuteImplementWithTasks, executeTaskLoop
Error handling:          handleImplementError, validateTasksCompleteFunc
Output/printing:         printFullWorkflowSummary, printPhaseCompletion
```

**Recommended Refactor:**

Split into focused components:
```go
// 1. Orchestrator - coordination only
type WorkflowOrchestrator struct {
    stageExecutor StageExecutor
    phaseExecutor PhaseExecutor
    taskExecutor  TaskExecutor
    config        Configuration // interface
}

// 2. StageExecutor - specify/plan/tasks stages
type StageExecutor struct { ... }

// 3. PhaseExecutor - phase-based implementation
type PhaseExecutor struct { ... }

// 4. TaskExecutor - task-level execution
type TaskExecutor struct { ... }
```

**Impact:** HIGH - Core of the application, affects testability and maintainability

---

### 2. Executor Class Overloaded

**Location:** `internal/workflow/executor.go` (451 LOC, 18 methods)

**Mixed Concerns:**
- Execution (Claude commands)
- Display (progress, spinner)
- Notifications (notification handler calls)
- Retry logic (retry state management)

**Recommended Refactor:**

Extract into 3-4 separate types:
```go
// 1. Executor - pure execution
type Executor struct {
    claude ClaudeRunner // interface
}

// 2. ProgressController - display concerns
type ProgressController struct {
    display *progress.ProgressDisplay
    spinner *spinner.Spinner
}

// 3. NotifyDispatcher - notification routing
type NotifyDispatcher struct {
    handler NotificationHandler // interface
}
```

**Impact:** HIGH - Improves testability and separation of concerns

---

### 3. CLI Package Size

**Location:** `internal/cli/` (6,463 LOC, 47 files)

Currently monolithic despite being split into 47 files. Commands are loosely organized.

**Recommended Subpackage Structure:**
```
cli/
├── stages/        # specify.go, plan.go, tasks.go, implement.go
├── config/        # init.go, config.go, migrate.go, doctor.go
├── util/          # status.go, history.go, version.go, clean.go
├── admin/         # commands*, completion*, uninstall.go
└── root.go, execute.go, all.go, prep.go, run.go (orchestration)
```

**Impact:** HIGH - Reduces cognitive load, improves navigation

---

## Medium Priority Issues

### 4. Incomplete Dependency Injection

**Current State:**
```go
// Some components accept interfaces (good)
type Executor struct {
    NotificationHandler *notify.Handler    // Optional, injectable
    ProgressDisplay     *progress.ProgressDisplay
}

// Others use concrete types (poor)
type WorkflowOrchestrator struct {
    Executor    *Executor              // Concrete, not injectable
    Config      *config.Configuration  // Concrete, not injectable
}
```

**Recommended Pattern:**
```go
type WorkflowOrchestratorDeps struct {
    Executor   Executor      // interface
    Config     Configuration // interface
    Validator  Validator     // interface
    RetryStore RetryStore    // interface
}

func NewWorkflowOrchestrator(deps WorkflowOrchestratorDeps) *WorkflowOrchestrator
```

**Impact:** MEDIUM - Significantly improves testability

---

### 5. Strategy Pattern for Execute Methods

**Current Anti-Pattern:**
```go
// 10+ Execute variants on WorkflowOrchestrator
ExecuteSpecify()
ExecutePlan()
ExecuteTasks()
ExecuteImplement()
ExecuteImplementDefault()
ExecuteImplementWithPhases()
ExecuteImplementWithTasks()
ExecuteImplementSinglePhase()
ExecuteImplementFromPhase()
// etc.
```

**Recommended Refactor:**
```go
// Strategy interface
type ExecutionStrategy interface {
    Execute(ctx context.Context, spec SpecContext) error
}

// Implementations
type PhaseStrategy struct { ... }
type TaskStrategy struct { ... }
type SinglePhaseStrategy struct { ... }

// Orchestrator uses strategy
func (o *Orchestrator) Execute(strategy ExecutionStrategy, spec SpecContext) error {
    return strategy.Execute(ctx, spec)
}
```

**Impact:** MEDIUM - Reduces method count, improves maintainability

---

### 6. Validation Package Monolithic Schema

**Location:** `internal/validation/schema.go` (761 LOC)

All schema definitions consolidated in one file.

**Recommended Split:**
```
validation/
├── schemas/
│   ├── spec_schema.go
│   ├── plan_schema.go
│   ├── tasks_schema.go
│   ├── research_schema.go
│   └── ...
├── artifact.go        # interfaces
├── validator.go       # base validator
└── ...
```

**Impact:** MEDIUM - Improves code organization

---

### 7. String Enums

**Current:**
```go
type Stage string

const (
    StageSpecify   Stage = "specify"
    StagePlan      Stage = "plan"
    // ...
)
```

**Issue:** Prone to typos, no compiler validation for invalid values.

**Recommended:**
```go
type Stage int

const (
    StageUnknown Stage = iota
    StageSpecify
    StagePlan
    StageTasks
    StageImplement
)

func (s Stage) String() string { ... }
func ParseStage(s string) (Stage, error) { ... }
```

**Impact:** LOW - Improves type safety

---

## Low Priority Issues

### 8. Ad-hoc Logging

**Current:** Multiple types have `debugLog()` methods scattered throughout.

**Recommended:**
- Use `log/slog` structured logging
- Centralize logger configuration
- Consistent log levels across packages

**Impact:** LOW - Improves observability

---

### 9. Anemic Validator Objects

**Current:** Validator implementations mostly empty, all logic in baseValidator.

```go
type SpecValidator struct {
    *baseValidator
}
// Methods just call baseValidator
```

**Better Pattern:** Composition over inheritance
```go
type SpecValidator struct {
    schema    Schema
    validator ValidatorEngine
}
```

**Impact:** LOW - More idiomatic Go

---

## Strengths (Keep)

| Pattern | Description |
|---------|-------------|
| **Lifecycle Wrapper** | Clean, minimal implementation for notifications/history/timing |
| **Factory Pattern** | `NewArtifactValidator(artifactType)` - consistent across codebase |
| **No Circular Deps** | Clean acyclic dependency graph |
| **Test Ratio** | 2:1 test-to-code ratio with consistent patterns |
| **Interface Usage** | Good for optional components (NotificationHandler, PreflightChecker) |

---

## Dependency Graph (Current)

```
cmd/main.go
    ↓
cli/root.go
    ↓
cli/*.go (47 commands)
    ├→ workflow (execute stages)
    ├→ validation (validate artifacts)
    ├→ config (load settings)
    ├→ lifecycle (execution wrapper)
    └→ notify (notifications)

workflow/
    ├→ config (read configuration)
    ├→ retry (persistence)
    ├→ validation (validate output)
    ├→ lifecycle (wrap stage execution)
    ├→ notify (send notifications)
    └→ progress (display progress)

validation/
    ├→ yaml (YAML validation)
    └→ (minimal external coupling)
```

**No circular dependencies** - Clean acyclic architecture ✓

---

## Recommended Implementation Order

| Priority | Task | Effort | Impact |
|----------|------|--------|--------|
| 1 | Split WorkflowOrchestrator | HIGH | HIGH |
| 2 | Extract Executor concerns | MEDIUM | HIGH |
| 3 | Create CLI subpackages | LOW | MEDIUM |
| 4 | Add interfaces for DI | MEDIUM | MEDIUM |
| 5 | Strategy pattern for Execute* | MEDIUM | MEDIUM |
| 6 | Split validation schemas | LOW | LOW |
| 7 | Type-safe enums | LOW | LOW |

---

## Testability Improvements

**Current Hard-to-Test Areas:**
- WorkflowOrchestrator (38 methods, multiple concerns)
- Stage execution (external Claude process)
- CLI commands (each spawns orchestrator)
- Phase execution (complex state machine)

**With Recommended Changes:**
- Smaller, focused components are easier to unit test
- Interface-based DI allows mocking
- Strategy pattern enables isolated testing of execution modes
- Extracted concerns (progress, notifications) can be tested independently
