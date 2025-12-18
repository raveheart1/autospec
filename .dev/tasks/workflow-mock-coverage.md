# Workflow Package Mock Coverage Enhancement

**Goal**: Reach 85% test coverage for `internal/workflow/` (currently at 79.4%)

## Key Finding: Existing Tests Don't Call Target Functions

The tests `TestExecuteSinglePhaseSession` and `TestExecuteSingleTaskSession` exist but **only test string formatting** - they never actually call the methods! This is why coverage is 0%.

**Current test pattern (WRONG):**
```go
func TestExecuteSingleTaskSession(t *testing.T) {
    // Only builds command string locally
    command := fmt.Sprintf("/autospec.implement --task %s", taskID)
    if command != wantCmd { ... }  // Never calls actual function!
}
```

**What's needed:**
```go
func TestExecuteSingleTaskSession(t *testing.T) {
    orch := newTestOrchestratorWithSpecName(t, "001-test")  // Uses mock-claude.sh
    err := orch.executeSingleTaskSession("001-test", "T001", "Test", "")  // Actually call it!
}
```

## Mock Infrastructure Already Available

### 1. `mock-claude.sh` (mocks/scripts/mock-claude.sh)
- Simulates Claude CLI without network calls
- **Generates artifacts** when `MOCK_ARTIFACT_DIR` is set:
  - `/autospec.specify` → creates `spec.yaml`
  - `/autospec.plan` → creates `plan.yaml`
  - `/autospec.tasks` → creates `tasks.yaml`
  - `/autospec.implement` → marks tasks as `Completed`
- Configurable via env vars: `MOCK_EXIT_CODE`, `MOCK_DELAY`, `MOCK_CALL_LOG`

### 2. `MockClaudeExecutor` (mocks_test.go)
- Go-level mock for unit testing without subprocess
- Records method calls, simulates errors/delays
- Good for testing executor logic in isolation

### 3. `newTestOrchestratorWithSpecName()` helper
- Creates orchestrator configured with `mock-claude.sh`
- Sets up temp directories and env vars
- Already used by `TestExecuteSpecify`, `TestExecutePlan`, etc.

---

## Coverage Gap Analysis

### Functions with 0% Coverage

| Function | File:Line | Current Coverage | Root Cause |
|----------|-----------|------------------|------------|
| `PromptUserToContinue` | preflight.go:117 | 0% | Reads from `os.Stdin` |
| `runPreflightChecks` | workflow.go:217 | 0% | Calls real system checks |
| `executeAndVerifyTask` | workflow.go:857 | 0% | **Tests don't call it** |
| `executeSingleTaskSession` | workflow.go:927 | 0% | **Tests don't call it** |

### Functions with < 60% Coverage

| Function | File:Line | Current Coverage | Root Cause |
|----------|-----------|------------------|------------|
| `startProgressDisplay` | executor.go:159 | 25% | Missing nil/error cases |
| `executeSinglePhaseSession` | workflow.go:973 | 32.7% | **Tests don't call it** |
| `completeStageSuccessNoNotify` | executor.go:237 | 54.5% | Missing error paths |
| `executeTaskLoop` | workflow.go:822 | 55.6% | Depends on task execution |
| `CleanupContextFile` | phase_context.go:239 | 60% | Missing edge cases |

---

## Mock Requirements by Category

## Quick Wins: Tests That Just Need to Call Functions

These functions already have mock infrastructure via `mock-claude.sh`. Tests just need to:
1. Use `newTestOrchestratorWithSpecName()`
2. Set up prerequisite artifacts
3. **Actually call the function**

### Quick Win 1: `executeSingleTaskSession` (+~3% coverage)

```go
func TestExecuteSingleTaskSession_Integration(t *testing.T) {
    orch, specsDir := newTestOrchestratorWithSpecName(t, "001-test")
    specDir := filepath.Join(specsDir, "001-test")

    // Create prerequisite artifacts (spec.yaml, plan.yaml, tasks.yaml with Pending task)
    createTestArtifacts(t, specDir)

    // mock-claude.sh will mark tasks as Completed when called
    err := orch.executeSingleTaskSession("001-test", "T001", "Test Task", "")
    if err != nil {
        t.Errorf("unexpected error: %v", err)
    }

    // Verify task was marked completed
    task := getTaskByID(t, specDir, "T001")
    if task.Status != "Completed" {
        t.Errorf("task status = %q, want Completed", task.Status)
    }
}
```

### Quick Win 2: `executeSinglePhaseSession` (+~3% coverage)

```go
func TestExecuteSinglePhaseSession_Integration(t *testing.T) {
    orch, specsDir := newTestOrchestratorWithSpecName(t, "001-test")
    specDir := filepath.Join(specsDir, "001-test")

    // Create all artifacts with phase 1 tasks
    createTestArtifacts(t, specDir)

    err := orch.executeSinglePhaseSession("001-test", 1, "")
    if err != nil {
        t.Errorf("unexpected error: %v", err)
    }
}
```

### Quick Win 3: `executeAndVerifyTask` (+~2% coverage)

```go
func TestExecuteAndVerifyTask_Integration(t *testing.T) {
    orch, specsDir := newTestOrchestratorWithSpecName(t, "001-test")
    specDir := filepath.Join(specsDir, "001-test")
    tasksPath := filepath.Join(specDir, "tasks.yaml")

    createTestArtifacts(t, specDir)
    tasks, _ := validation.GetAllTasks(tasksPath)

    err := orch.executeAndVerifyTask("001-test", tasksPath, tasks[0], "")
    // ...
}
```

### Quick Win 4: `executeTaskLoop` (+~2% coverage)

```go
func TestExecuteTaskLoop_Integration(t *testing.T) {
    orch, specsDir := newTestOrchestratorWithSpecName(t, "001-test")
    // ... setup tasks with various states (Pending, Completed, Blocked)

    err := orch.executeTaskLoop("001-test", tasksPath, orderedTasks, 0, 3, "")
    // Verify only Pending tasks were executed
}
```

---

## Functions Requiring Actual Refactoring

### Category 1: User Input Mocking (PromptUserToContinue)

**Problem**: Function reads from `os.Stdin` directly using `bufio.NewReader(os.Stdin)`.

**Required Mocking**:
```go
// Option A: Dependency injection for stdin reader
type StdinReader interface {
    ReadString(delim byte) (string, error)
}

// Option B: Function parameter injection
func PromptUserToContinueWithReader(warningMessage string, reader io.Reader) (bool, error)
```

**Refactoring Required**:
1. Extract `os.Stdin` to an interface or parameter
2. Create mock that simulates "y", "n", "yes", "no", and EOF inputs
3. Test error cases (read failure, unexpected input)

**Test Cases**:
- User enters "y" → returns (true, nil)
- User enters "yes" → returns (true, nil)
- User enters "n" → returns (false, nil)
- User enters "N" → returns (false, nil)
- User enters "" → returns (false, nil)
- Reader returns error → returns (false, error)

---

### Category 2: Preflight Check System Mocking (runPreflightChecks)

**Problem**: Calls `RunPreflightChecks()` which checks real system dependencies (claude CLI, git, project structure).

**Required Mocking**:
```go
// Interface for preflight system
type PreflightChecker interface {
    RunPreflightChecks() (*PreflightResult, error)
    PromptUserToContinue(msg string) (bool, error)
}

// Mock implementation
type MockPreflightChecker struct {
    Result      *PreflightResult
    Error       error
    UserContinue bool
    UserPromptErr error
}
```

**Refactoring Required**:
1. Create `PreflightChecker` interface
2. Inject `PreflightChecker` into `WorkflowOrchestrator`
3. Create mock that returns configurable `PreflightResult`

**Test Cases**:
- All checks pass → continues
- Checks fail with warning → prompts user
- User continues after warning → proceeds
- User aborts after warning → returns error
- Critical failure (missing CLI) → returns error immediately
- RunPreflightChecks returns error → wrapped error

---

### Category 3: Task Execution Session Mocking (executeSingleTaskSession, executeAndVerifyTask)

**Problem**: These functions:
1. Call `w.Executor.ExecuteStage()` which runs real claude commands
2. Call validation functions that read/parse tasks.yaml files
3. Have complex validation callbacks

**Required Mocking**:
```go
// Extended MockStageExecutor
type MockStageExecutor struct {
    *Executor
    ExecuteStageFunc func(specName string, stage Stage, command string, validator func(string) error) (*StageResult, error)

    // Task-specific behavior
    TaskResults      map[string]*StageResult  // taskID -> result
    TaskValidations  map[string]error         // taskID -> validation error
    SimulateTaskCompletion bool               // Auto-mark tasks as completed
}

// Mock that simulates task file updates
type MockTasksFileManager struct {
    Tasks []validation.TaskItem
    // UpdateTask simulates claude updating the task status
    UpdateTask(taskID, newStatus string) error
}
```

**Refactoring Required**:
1. Create `StageExecutor` interface extracted from `Executor`
2. Create `TasksFileReader` interface for validation functions
3. Inject both into `WorkflowOrchestrator`

**Test Cases for executeAndVerifyTask**:
- Task dependencies met → executes task
- Task dependencies not met → skips with message
- Task execution succeeds → verifies completion
- Task execution fails → returns error
- Task completes but verification fails → returns error

**Test Cases for executeSingleTaskSession**:
- Simple task execution → success
- Task with prompt → includes prompt in command
- Execution returns exhausted → prints resume instructions
- Validation callback fails → returns error

---

### Category 4: Phase Execution Session Mocking (executeSinglePhaseSession)

**Problem**: Similar to task execution but also:
1. Calls `BuildPhaseContext()` which reads spec, plan, tasks files
2. Calls `WriteContextFile()` which writes to filesystem
3. Has defer `CleanupContextFile()` for cleanup

**Required Mocking**:
```go
// Phase context builder interface
type PhaseContextBuilder interface {
    BuildPhaseContext(specDir string, phaseNumber, totalPhases int) (*PhaseContext, error)
    WriteContextFile(ctx *PhaseContext) (string, error)
    CleanupContextFile(path string) error
}

// Mock implementation
type MockPhaseContextBuilder struct {
    Context       *PhaseContext
    ContextPath   string
    BuildError    error
    WriteError    error
    CleanupCalled bool
}
```

**Refactoring Required**:
1. Create `PhaseContextBuilder` interface
2. Inject into `WorkflowOrchestrator`
3. Mock should track cleanup calls

**Test Cases**:
- Empty phase → skips execution
- All tasks completed → skips execution
- Phase execution succeeds → cleans up context
- Phase execution fails → still cleans up context (defer)
- Build context fails → returns error
- Write context fails → returns error
- Execution retries exhausted → prints resume instructions

---

### Category 5: Progress Display Mocking (startProgressDisplay, completeStageSuccessNoNotify)

**Problem**: Calls `e.ProgressDisplay.StartStage()` and `CompleteStage()` methods.

**Required Mocking**:
```go
// Already exists: progress.Display interface
// Need mock that tracks calls and can simulate errors

type MockProgressDisplay struct {
    StartCalls    []progress.StageInfo
    CompleteCalls []progress.StageInfo
    StartError    error
    CompleteError error
}
```

**Test Cases**:
- ProgressDisplay is nil → no panic
- StartStage returns error → logs warning, continues
- CompleteStage returns error → logs warning, continues
- Normal operation → tracks stage info correctly

---

### Category 6: Task Loop Mocking (executeTaskLoop)

**Problem**: Iterates through tasks calling `executeAndVerifyTask` for each.

**Test Cases (with Category 3 mocks)**:
- Empty task list → returns nil immediately
- All tasks completed → skips all
- Mix of completed/pending → only executes pending
- Task fails mid-loop → returns error, stops loop
- Blocked task → skips with message

---

## Implementation Order (Revised)

### Phase 1: Quick Wins - Just Call The Functions (~10% coverage gain)
**No refactoring needed - just write tests that actually call the methods**

| Priority | Function | Est. Coverage Gain |
|----------|----------|-------------------|
| 1 | `executeSingleTaskSession` | +3% |
| 2 | `executeSinglePhaseSession` | +3% |
| 3 | `executeAndVerifyTask` | +2% |
| 4 | `executeTaskLoop` | +2% |

**Pattern to follow:**
```go
func TestXxx_Integration(t *testing.T) {
    // Cannot use t.Parallel() - uses t.Setenv for mock-claude.sh
    orch, specsDir := newTestOrchestratorWithSpecName(t, "001-test")
    specDir := filepath.Join(specsDir, "001-test")

    // Use helper to create spec.yaml, plan.yaml, tasks.yaml
    createTestArtifacts(t, specDir)

    // ACTUALLY CALL THE FUNCTION
    err := orch.executeSingleTaskSession("001-test", "T001", "Test", "")

    // Verify result
}
```

### Phase 2: Minor Refactoring - Stdin/Preflight (~3% coverage gain)

| Priority | Function | Refactoring Needed |
|----------|----------|-------------------|
| 5 | `PromptUserToContinue` | Add `io.Reader` parameter |
| 6 | `runPreflightChecks` | Inject `PreflightChecker` interface |

### Phase 3: Edge Cases (~2% coverage gain)

| Priority | Function | What's Missing |
|----------|----------|----------------|
| 7 | `startProgressDisplay` | Error path when `StartStage` fails |
| 8 | `completeStageSuccessNoNotify` | Error paths for retry reset |
| 9 | `CleanupContextFile` | File not found / permission errors |

---

## Files to Create/Modify

### Phase 1 (No new files needed)
- `internal/workflow/workflow_test.go` - Add integration tests that call actual methods

### Phase 2 (Minor refactoring)
- `internal/workflow/preflight.go` - Add `PromptUserToContinueWithReader()` variant
- `internal/workflow/workflow.go` - Use injectable preflight checker

### Phase 3 (Edge cases)
- `internal/workflow/executor_test.go` - Add error path tests
- `internal/workflow/phase_context_test.go` - Add cleanup edge cases

---

## Acceptance Criteria

1. `go test -cover ./internal/workflow/` reports >= 85%
2. All new tests follow map-based table-driven pattern
3. All new code has wrapped errors with context
4. `make test`, `make fmt`, `make lint`, `make build` all pass
5. No breaking changes to existing API

---

## Summary

**Current**: 79.4% coverage
**Target**: 85% coverage
**Gap**: 5.6%

**Breakdown**:
- **Quick wins (Phase 1)**: ~10% potential gain - just write tests that call functions
- **Minor refactoring (Phase 2)**: ~3% potential gain - stdin/preflight mocking
- **Edge cases (Phase 3)**: ~2% potential gain - error paths

**Key Insight**: Most coverage can be gained WITHOUT complex mocking. The `mock-claude.sh` script already handles artifact generation. The main problem is existing tests only test string formatting instead of calling the actual methods.

---

## Risk Assessment

| Risk | Mitigation |
|------|------------|
| Interface changes break existing code | Ensure interfaces match current signatures exactly |
| Mocks become maintenance burden | Keep mocks focused, use builder pattern |
| Over-engineering | Only add interfaces needed for testing |
| Test flakiness | Avoid time-dependent tests, use channels for sync |
