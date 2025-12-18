# internal/cli Test Coverage Improvement Plan

**Current Coverage:** 43.2%
**Target Coverage:** 85-90%
**Functions with 0% coverage:** ~100

## Executive Summary

The `internal/cli` package has low test coverage primarily due to **tightly coupled dependencies** in command `RunE` functions. The existing tests focus on flag registration, argument parsing, and source code pattern verification—but the actual command execution logic remains untested.

---

## Current Coverage Analysis

### What IS Tested (High Coverage Functions)

| Category | Pattern | Example |
|----------|---------|---------|
| `init()` functions | 100% | All command registrations |
| Flag parsing | 90%+ | `parseArtifactArgs`, `resolveArtifactPath` |
| Source pattern verification | Via string matching | `TestAllCommandsHaveNotificationSupport` |
| Argument validation | Logic extracted to tests | `TestImplementArgParsing` |
| Pure utility functions | No dependencies | `formatStatus`, `filterEntries` |

### What is NOT Tested (0% Coverage)

```
run.go:          executeStages, executeStage, executeSpecify, executePlan, etc.
run.go:          printDryRunPreview, printWorkflowSummary
implement.go:    Full RunE function (uses workflow orchestrator)
prereqs.go:      runPrereqs, detectCurrentFeature
doctor.go:       Full Run function (uses health.RunHealthChecks)
version.go:      printPrettyVersion, getTerminalWidth
status.go:       Full RunE function
setup_plan.go:   runSetupPlan
update_task.go:  runUpdateTask
uninstall.go:    executeUninstall, displayRemovalResults
```

---

## Root Causes of Low Coverage

### 1. Hardcoded Dependency Construction (CRITICAL)

Every workflow command directly constructs its dependencies inside `RunE`:

```go
// implement.go:141 - Config is loaded directly
cfg, err := config.Load(configPath)

// implement.go:216 - Orchestrator is constructed inline
orch := workflow.NewWorkflowOrchestrator(cfg)

// implement.go:214 - Lifecycle wrapper is called directly
return lifecycle.RunWithHistoryContext(cmd.Context(), notifHandler, historyLogger, ...)
```

This pattern makes unit testing impossible because:
- `config.Load()` reads from the filesystem
- `NewWorkflowOrchestrator()` creates a real Claude executor
- The orchestrator methods call external processes (`claude` CLI)

### 2. Global State via Cobra Commands

Commands are registered as global variables with `init()`:

```go
var implementCmd = &cobra.Command{
    ...
    RunE: func(cmd *cobra.Command, args []string) error {
        // All logic is here, untestable
    },
}

func init() {
    rootCmd.AddCommand(implementCmd)
}
```

This prevents:
- Injecting mock dependencies
- Testing `RunE` in isolation
- Parallelizing tests (shared global state)

### 3. Side Effects in Core Logic

Commands produce filesystem side effects that are hard to verify:
- `spec.DetectCurrentSpec()` - reads git and filesystem
- `workflow.CheckConstitutionExists()` - checks file existence
- `workflow.ValidateStagePrerequisites()` - validates directories
- `history.NewWriter()` - writes to state files

### 4. External Process Dependencies

Several commands shell out to external tools:
- `claude` CLI execution via `workflow.Executor`
- `git` operations via `git` package
- Health checks via `health.RunHealthChecks()`

---

## Mocking Strategy Recommendations

### Strategy 1: Interface Extraction + Dependency Injection

**Goal:** Extract interfaces for all external dependencies and inject them.

#### Required Interfaces

```go
// internal/cli/interfaces.go

// ConfigLoader loads configuration
type ConfigLoader interface {
    Load(path string) (*config.Configuration, error)
}

// WorkflowExecutor executes workflow stages
type WorkflowExecutor interface {
    ExecuteImplement(specName, prompt string, resume bool, opts PhaseExecutionOptions) error
    ExecuteSpecify(featureDescription string) (string, error)
    ExecutePlan(specName, prompt string) error
    ExecuteTasks(specName, prompt string) error
    ExecuteConstitution(prompt string) error
    ExecuteClarify(specName, prompt string) error
    ExecuteChecklist(specName, prompt string) error
    ExecuteAnalyze(specName, prompt string) error
}

// SpecDetector detects current spec from environment
type SpecDetector interface {
    DetectCurrentSpec(specsDir string) (*spec.Metadata, error)
    GetSpecMetadata(specsDir, specName string) (*spec.Metadata, error)
}

// PreflightValidator validates prerequisites
type PreflightValidator interface {
    CheckConstitutionExists() *workflow.ConstitutionCheckResult
    ValidateStagePrerequisites(stage workflow.Stage, specDir string) *workflow.PrereqResult
    CheckArtifactDependencies(config *workflow.StageConfig, specDir string) *workflow.ArtifactDependencyResult
}

// HistoryRecorder records command history
type HistoryRecorder interface {
    WriteStart(entry history.Entry) error
    UpdateComplete(entry history.Entry, err error) error
}

// NotificationDispatcher handles notifications
type NotificationDispatcher interface {
    OnCommandComplete(command, spec string, duration time.Duration, err error)
}
```

#### Command Context Pattern

```go
// internal/cli/command_context.go

// CommandContext holds injectable dependencies for CLI commands
type CommandContext struct {
    ConfigLoader       ConfigLoader
    WorkflowExecutor   WorkflowExecutor
    SpecDetector       SpecDetector
    PreflightValidator PreflightValidator
    HistoryRecorder    HistoryRecorder
    NotificationHandler NotificationDispatcher
    Stdout             io.Writer
    Stderr             io.Writer
}

// DefaultCommandContext returns production dependencies
func DefaultCommandContext() *CommandContext {
    return &CommandContext{
        ConfigLoader:       &defaultConfigLoader{},
        // ... other defaults
        Stdout:             os.Stdout,
        Stderr:             os.Stderr,
    }
}
```

#### Refactored Command Pattern

```go
// Before (untestable)
RunE: func(cmd *cobra.Command, args []string) error {
    cfg, err := config.Load(configPath)
    ...
    orch := workflow.NewWorkflowOrchestrator(cfg)
    return orch.ExecuteImplement(...)
}

// After (testable)
var implementCmdContext *CommandContext = nil // Allows injection for tests

RunE: func(cmd *cobra.Command, args []string) error {
    ctx := implementCmdContext
    if ctx == nil {
        ctx = DefaultCommandContext()
    }
    return runImplement(cmd, args, ctx)
}

func runImplement(cmd *cobra.Command, args []string, ctx *CommandContext) error {
    cfg, err := ctx.ConfigLoader.Load(configPath)
    if err != nil {
        return fmt.Errorf("loading config: %w", err)
    }

    return ctx.WorkflowExecutor.ExecuteImplement(specName, prompt, resume, phaseOpts)
}
```

### Strategy 2: Mock Package in internal/cli

Create `internal/cli/mocks_test.go`:

```go
package cli

// MockConfigLoader for testing
type MockConfigLoader struct {
    LoadFunc func(path string) (*config.Configuration, error)
    LoadCalls []string
}

func (m *MockConfigLoader) Load(path string) (*config.Configuration, error) {
    m.LoadCalls = append(m.LoadCalls, path)
    if m.LoadFunc != nil {
        return m.LoadFunc(path)
    }
    return &config.Configuration{
        SpecsDir:    "./specs",
        StateDir:    "/tmp/test-state",
        MaxRetries:  3,
    }, nil
}

// MockWorkflowExecutor for testing
type MockWorkflowExecutor struct {
    ExecuteImplementFunc func(specName, prompt string, resume bool, opts workflow.PhaseExecutionOptions) error
    ExecuteImplementCalls []ExecuteImplementCall
    // ... other methods
}

type ExecuteImplementCall struct {
    SpecName  string
    Prompt    string
    Resume    bool
    PhaseOpts workflow.PhaseExecutionOptions
}

func (m *MockWorkflowExecutor) ExecuteImplement(specName, prompt string, resume bool, opts workflow.PhaseExecutionOptions) error {
    m.ExecuteImplementCalls = append(m.ExecuteImplementCalls, ExecuteImplementCall{
        SpecName:  specName,
        Prompt:    prompt,
        Resume:    resume,
        PhaseOpts: opts,
    })
    if m.ExecuteImplementFunc != nil {
        return m.ExecuteImplementFunc(specName, prompt, resume, opts)
    }
    return nil
}
```

### Strategy 3: Test Helper Functions

For commands that only need filesystem isolation:

```go
// internal/cli/test_helpers_test.go

func setupTestSpec(t *testing.T, specName string) (string, func()) {
    t.Helper()

    tmpDir := t.TempDir()
    specsDir := filepath.Join(tmpDir, "specs")
    specDir := filepath.Join(specsDir, specName)

    os.MkdirAll(specDir, 0755)

    // Create minimal spec.yaml
    specYaml := `name: test-spec
status: draft
`
    os.WriteFile(filepath.Join(specDir, "spec.yaml"), []byte(specYaml), 0644)

    return tmpDir, func() { os.RemoveAll(tmpDir) }
}

func setupTestConfig(t *testing.T, tmpDir string) *config.Configuration {
    t.Helper()
    return &config.Configuration{
        SpecsDir:   filepath.Join(tmpDir, "specs"),
        StateDir:   filepath.Join(tmpDir, "state"),
        ClaudeCmd:  "echo", // Dummy command for testing
        MaxRetries: 1,
    }
}
```

---

## Implementation Priority

### Phase 1: Quick Wins (Effort: Low, Impact: +10-15%)

1. **Test pure functions** that currently have 0% coverage:
   - `joinStageNames()`
   - `truncateCommit()`
   - `centerText()`
   - `formatID()`
   - `isValidStatus()`

2. **Test error paths** in partially covered functions:
   - `runClean` error handling
   - `runConfigShow` edge cases
   - `migrateDirectory` error paths

### Phase 2: Interface Extraction (Effort: Medium, Impact: +20-25%)

3. **Create interfaces** for core dependencies:
   - `ConfigLoader`
   - `WorkflowExecutor`
   - `SpecDetector`

4. **Refactor commands** to accept context:
   - `implement`, `run`, `specify`, `plan`, `tasks`
   - These are the highest-impact commands

### Phase 3: Full Mock Coverage (Effort: High, Impact: +20-25%)

5. **Create comprehensive mocks** in `internal/cli/mocks_test.go`

6. **Test all command RunE functions** with mocked dependencies:
   - Success paths
   - Error handling (config load fails, orchestrator fails, etc.)
   - Flag combinations
   - Argument validation

7. **Test lifecycle/notification integration**:
   - Verify history is recorded
   - Verify notifications are dispatched

### Phase 4: Integration Tests (Effort: Medium, Impact: +5-10%)

8. **Create integration tests** with real filesystem (in `_test.go` files):
   - Use `t.TempDir()` for isolation
   - Mock only external processes (claude CLI)

---

## Specific Files Requiring Attention

### High Priority (Core Workflow Commands)

| File | Current Coverage | Functions Needing Tests |
|------|------------------|------------------------|
| `implement.go` | ~10% (init only) | RunE, all flag handling logic |
| `run.go` | ~10% (init only) | executeStages, executeStage, all stage handlers |
| `specify.go` | ~10% (init only) | RunE |
| `plan.go` | ~10% (init only) | RunE |
| `tasks.go` | ~10% (init only) | RunE |

### Medium Priority (Supporting Commands)

| File | Current Coverage | Functions Needing Tests |
|------|------------------|------------------------|
| `status.go` | ~10% | RunE, status display logic |
| `prereqs.go` | 0% | runPrereqs, detectCurrentFeature |
| `doctor.go` | ~10% | Run function |
| `setup_plan.go` | ~10% | runSetupPlan |

### Lower Priority (Utility Commands)

| File | Current Coverage | Functions Needing Tests |
|------|------------------|------------------------|
| `version.go` | ~10% | printPrettyVersion, formatters |
| `uninstall.go` | ~50% | executeUninstall, displayRemovalResults |
| `update_task.go` | ~60% | runUpdateTask |

---

## Code Examples

### Example Test for Implement Command

```go
func TestImplementCommand_Success(t *testing.T) {
    mockCtx := &CommandContext{
        ConfigLoader: &MockConfigLoader{
            LoadFunc: func(path string) (*config.Configuration, error) {
                return &config.Configuration{
                    SpecsDir:        "./specs",
                    ImplementMethod: "phases",
                }, nil
            },
        },
        SpecDetector: &MockSpecDetector{
            DetectFunc: func(specsDir string) (*spec.Metadata, error) {
                return &spec.Metadata{
                    Name:      "test-feature",
                    Number:    "001",
                    Directory: "./specs/001-test-feature",
                }, nil
            },
        },
        WorkflowExecutor: &MockWorkflowExecutor{},
        PreflightValidator: &MockPreflightValidator{
            CheckConstitutionFunc: func() *workflow.ConstitutionCheckResult {
                return &workflow.ConstitutionCheckResult{Exists: true}
            },
            ValidatePrereqsFunc: func(stage workflow.Stage, dir string) *workflow.PrereqResult {
                return &workflow.PrereqResult{Valid: true}
            },
        },
        Stdout: io.Discard,
        Stderr: io.Discard,
    }

    // Inject mock context
    implementCmdContext = mockCtx
    defer func() { implementCmdContext = nil }()

    cmd := implementCmd
    cmd.SetArgs([]string{})

    err := cmd.Execute()
    require.NoError(t, err)

    // Verify workflow executor was called correctly
    assert.Len(t, mockCtx.WorkflowExecutor.(*MockWorkflowExecutor).ExecuteImplementCalls, 1)
    call := mockCtx.WorkflowExecutor.(*MockWorkflowExecutor).ExecuteImplementCalls[0]
    assert.Equal(t, "001-test-feature", call.SpecName)
    assert.True(t, call.PhaseOpts.RunAllPhases) // Default from config
}

func TestImplementCommand_ConfigLoadError(t *testing.T) {
    mockCtx := &CommandContext{
        ConfigLoader: &MockConfigLoader{
            LoadFunc: func(path string) (*config.Configuration, error) {
                return nil, errors.New("config file not found")
            },
        },
        Stderr: &bytes.Buffer{},
    }

    implementCmdContext = mockCtx
    defer func() { implementCmdContext = nil }()

    cmd := implementCmd
    cmd.SetArgs([]string{})

    err := cmd.Execute()
    require.Error(t, err)
    assert.Contains(t, err.Error(), "config")
}
```

---

## Estimated Effort

| Phase | Estimated Time | Coverage Gain |
|-------|---------------|---------------|
| Phase 1: Quick Wins | 2-3 days | +10-15% |
| Phase 2: Interface Extraction | 3-4 days | +20-25% |
| Phase 3: Full Mock Coverage | 4-5 days | +20-25% |
| Phase 4: Integration Tests | 2-3 days | +5-10% |
| **Total** | **11-15 days** | **55-75%** → **85-90%** |

---

## Dependencies on Other Packages

The CLI package depends heavily on:

1. **`internal/workflow`** - Already has mock infrastructure (`MockClaudeExecutor`, `mockPreflightChecker`)
2. **`internal/config`** - Needs mock for `Load()`
3. **`internal/spec`** - Needs mock for `DetectCurrentSpec()`, `GetSpecMetadata()`
4. **`internal/validation`** - Used for artifact validation
5. **`internal/history`** - Needs mock for `NewWriter()`
6. **`internal/lifecycle`** - May need mock for `RunWithHistory*` functions
7. **`internal/notify`** - Needs mock for `NewHandler()`

The `internal/workflow` package provides a good template for mock patterns that can be adapted for CLI testing.

---

## Risks and Mitigations

| Risk | Mitigation |
|------|------------|
| Refactoring introduces bugs | Add tests for existing behavior before refactoring |
| Interface proliferation | Keep interfaces minimal; don't mock what doesn't need mocking |
| Test maintenance burden | Use table-driven tests; share mock setup helpers |
| Global state issues | Use `t.Cleanup()` to restore global state after tests |

---

## Conclusion

Achieving 85-90% coverage in `internal/cli` requires architectural changes to support dependency injection. The existing test patterns (flag checking, source verification) are necessary but insufficient. The recommended approach is:

1. Extract interfaces for external dependencies
2. Create a `CommandContext` pattern for dependency injection
3. Build comprehensive mocks following the `internal/workflow` patterns
4. Test all command paths including errors

This is a significant but achievable undertaking that will dramatically improve confidence in CLI behavior and prevent regressions during future development.
