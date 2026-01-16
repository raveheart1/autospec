# Quickstart: Workflow Progress Indicators

**Feature**: 004-workflow-progress-indicators
**Audience**: Developers implementing this feature
**Time to Complete**: 5-10 minutes to understand, 4-6 hours to implement with tests

## Overview

This quickstart guides you through implementing progress indicators for autospec CLI workflows. You'll add phase counters ([1/3], [2/3]), animated spinners, and completion checkmarks to improve user experience during long-running operations.

---

## Prerequisites

Before starting implementation:

1. **Read design documents** (10 minutes):
   - [spec.md](./spec.md) - User stories and requirements
   - [research.md](./research.md) - Technology decisions and rationale
   - [data-model.md](./data-model.md) - Entity definitions
   - [contracts/progress_package_api.md](./contracts/progress_package_api.md) - API contract

2. **Understand existing codebase** (15 minutes):
   - Review `internal/workflow/executor.go` - Phase execution logic
   - Review `internal/cli/full.go` and `workflow.go` - CLI command structure
   - Understand how `PhaseSpecify`, `PhasePlan`, etc. are currently executed

3. **Set up development environment**:
   ```bash
   cd /home/ari/repos/autospec
   go mod download
   make test  # Ensure all existing tests pass
   ```

---

## Implementation Roadmap (Test-First)

**Constitution Principle III**: Tests MUST be written before implementation.

### Phase 1: Package Structure & Types (1 hour)

#### Step 1.1: Create package skeleton (10 min)

```bash
mkdir -p internal/progress
touch internal/progress/types.go
touch internal/progress/types_test.go
touch internal/progress/terminal.go
touch internal/progress/terminal_test.go
touch internal/progress/display.go
touch internal/progress/display_test.go
```

#### Step 1.2: Define types in `types.go` (20 min)

Copy type definitions from [data-model.md](./data-model.md):
- `PhaseStatus` enum
- `PhaseInfo` struct with `Validate()` method
- `TerminalCapabilities` struct
- `ProgressSymbols` struct

#### Step 1.3: Write unit tests FIRST in `types_test.go` (30 min)

**Test-First Approach**: Write failing tests before implementation.

```go
package progress_test

import (
    "testing"
    "github.com/ariel-frischer/autospec/internal/progress"
)

func TestPhaseInfo_Validate(t *testing.T) {
    tests := []struct {
        name    string
        phase   progress.PhaseInfo
        wantErr bool
        errMsg  string
    }{
        {
            name: "valid phase",
            phase: progress.PhaseInfo{
                Name: "specify", Number: 1, TotalPhases: 3,
                Status: progress.PhaseInProgress, RetryCount: 0, MaxRetries: 3,
            },
            wantErr: false,
        },
        {
            name: "empty name",
            phase: progress.PhaseInfo{Name: "", Number: 1, TotalPhases: 3},
            wantErr: true,
            errMsg: "phase name cannot be empty",
        },
        {
            name: "number exceeds total",
            phase: progress.PhaseInfo{Name: "test", Number: 4, TotalPhases: 3},
            wantErr: true,
            errMsg: "phase number cannot exceed total phases",
        },
        // Add more test cases covering all validation rules...
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := tt.phase.Validate()
            if tt.wantErr {
                if err == nil {
                    t.Errorf("Validate() error = nil, want error containing %q", tt.errMsg)
                } else if !strings.Contains(err.Error(), tt.errMsg) {
                    t.Errorf("Validate() error = %v, want error containing %q", err, tt.errMsg)
                }
            } else {
                if err != nil {
                    t.Errorf("Validate() unexpected error = %v", err)
                }
            }
        })
    }
}
```

**Run tests** (they should fail):
```bash
go test ./internal/progress/
# Expected: FAIL (Validate method not implemented yet)
```

#### Step 1.4: Implement `Validate()` to pass tests (10 min)

Now implement the validation logic in `types.go` to make tests pass.

---

### Phase 2: Terminal Detection (1 hour)

#### Step 2.1: Write tests for `DetectTerminalCapabilities()` (30 min)

In `terminal_test.go`:

```go
func TestDetectTerminalCapabilities(t *testing.T) {
    tests := []struct {
        name       string
        setupEnv   func()
        cleanupEnv func()
        wantTTY    bool // Hard to mock, test behavior instead
    }{
        {
            name: "NO_COLOR disables color",
            setupEnv: func() {
                os.Setenv("NO_COLOR", "1")
            },
            cleanupEnv: func() {
                os.Unsetenv("NO_COLOR")
            },
            // Verify SupportsColor = false in actual test
        },
        {
            name: "AUTOSPEC_ASCII forces ASCII",
            setupEnv: func() {
                os.Setenv("AUTOSPEC_ASCII", "1")
            },
            cleanupEnv: func() {
                os.Unsetenv("AUTOSPEC_ASCII")
            },
            // Verify SupportsUnicode = false
        },
        // More test cases...
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            if tt.setupEnv != nil {
                tt.setupEnv()
                defer tt.cleanupEnv()
            }

            caps := progress.DetectTerminalCapabilities()

            // Assertions based on environment setup
            // ...
        })
    }
}
```

#### Step 2.2: Add dependencies (5 min)

```bash
go get golang.org/x/term@latest
go get github.com/briandowns/spinner@latest
go mod tidy
```

#### Step 2.3: Implement `DetectTerminalCapabilities()` (15 min)

In `terminal.go`, implement the function using `term.IsTerminal()` and env var checks.

#### Step 2.4: Implement `selectSymbols()` helper (10 min)

Function to return `ProgressSymbols` based on `TerminalCapabilities`.

---

### Phase 3: ProgressDisplay Core Logic (2 hours)

#### Step 3.1: Write tests for display methods (1 hour)

In `display_test.go`:

```go
func TestProgressDisplay_StartPhase(t *testing.T) {
    tests := []struct {
        name         string
        capabilities progress.TerminalCapabilities
        phase        progress.PhaseInfo
        wantContains []string // Strings expected in output
        wantErr      bool
    }{
        {
            name: "TTY mode with Unicode",
            capabilities: progress.TerminalCapabilities{
                IsTTY: true, SupportsUnicode: true,
            },
            phase: progress.PhaseInfo{
                Name: "specify", Number: 1, TotalPhases: 3,
            },
            wantContains: []string{"[1/3]", "Running specify phase"},
            wantErr: false,
        },
        {
            name: "non-TTY mode",
            capabilities: progress.TerminalCapabilities{
                IsTTY: false,
            },
            phase: progress.PhaseInfo{
                Name: "plan", Number: 2, TotalPhases: 3,
            },
            wantContains: []string{"[2/3]", "Running plan phase"},
            wantErr: false,
        },
        {
            name: "retry attempt",
            capabilities: progress.TerminalCapabilities{
                IsTTY: true,
            },
            phase: progress.PhaseInfo{
                Name: "tasks", Number: 3, TotalPhases: 3,
                RetryCount: 1, MaxRetries: 3,
            },
            wantContains: []string{"[3/3]", "Running tasks phase", "(retry 2/3)"},
            wantErr: false,
        },
        // Invalid phase should return error
        {
            name: "invalid phase",
            capabilities: progress.TerminalCapabilities{IsTTY: true},
            phase: progress.PhaseInfo{Name: ""}, // Empty name
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Capture stdout
            old := os.Stdout
            r, w, _ := os.Pipe()
            os.Stdout = w

            display := progress.NewProgressDisplay(tt.capabilities)
            err := display.StartPhase(tt.phase)

            // Restore stdout and read output
            w.Close()
            os.Stdout = old
            var buf bytes.Buffer
            io.Copy(&buf, r)
            output := buf.String()

            // Assertions
            if tt.wantErr {
                if err == nil {
                    t.Errorf("StartPhase() error = nil, want error")
                }
            } else {
                if err != nil {
                    t.Errorf("StartPhase() unexpected error = %v", err)
                }
                for _, want := range tt.wantContains {
                    if !strings.Contains(output, want) {
                        t.Errorf("StartPhase() output = %q, want to contain %q", output, want)
                    }
                }
            }
        })
    }
}

// Similar tests for CompletePhase, FailPhase, UpdateRetry
```

**Run tests** (they should fail):
```bash
go test ./internal/progress/
# Expected: FAIL (methods not implemented)
```

#### Step 3.2: Implement `NewProgressDisplay()` (15 min)

Constructor that initializes display based on capabilities.

#### Step 3.3: Implement `StartPhase()` (20 min)

Method to start displaying phase progress with spinner (if TTY).

#### Step 3.4: Implement `CompletePhase()` and `FailPhase()` (20 min)

Methods to stop spinner and show completion status.

#### Step 3.5: Implement `UpdateRetry()` (5 min)

Simple wrapper around `StartPhase()` with updated phase info.

**Run tests again**:
```bash
go test ./internal/progress/
# Expected: PASS (all tests should pass now)
```

---

### Phase 4: Integration with workflow.Executor (1 hour)

#### Step 4.1: Modify `Executor` struct (10 min)

In `internal/workflow/executor.go`:

```go
type Executor struct {
    claudeExecutor *ClaudeExecutor
    retryManager   *retry.RetryManager
    progress       *progress.Display // NEW: optional progress display
}

// Update constructor
func NewExecutor(
    claudeExecutor *ClaudeExecutor,
    retryManager *retry.RetryManager,
    progressDisplay *progress.Display, // NEW: can be nil
) *Executor {
    return &Executor{
        claudeExecutor: claudeExecutor,
        retryManager:   retryManager,
        progress:       progressDisplay,
    }
}
```

#### Step 4.2: Add progress callbacks to `ExecutePhase()` (30 min)

```go
func (e *Executor) ExecutePhase(
    specName string,
    phase Phase,
    slashCommand string,
    validationFunc func(string) error,
) error {
    // Build PhaseInfo from Phase enum
    phaseInfo := progress.PhaseInfo{
        Name:        phase.String(), // Assuming Phase has String() method
        Number:      e.getPhaseNumber(phase),
        TotalPhases: e.getTotalPhases(),
        Status:      progress.PhaseInProgress,
        RetryCount:  e.retryManager.GetRetryCount(specName, phase),
        MaxRetries:  e.retryManager.GetMaxRetries(),
    }

    // Start progress display
    if e.progress != nil {
        if err := e.progress.StartPhase(phaseInfo); err != nil {
            // Log warning but don't fail workflow
            log.Printf("Warning: progress display error: %v", err)
        }
    }

    // Execute phase (existing logic)
    err := e.claudeExecutor.Execute(slashCommand)
    if err != nil {
        if e.progress != nil {
            e.progress.FailPhase(phaseInfo, err)
        }
        return err
    }

    // Validate (existing logic)
    if err := validationFunc(specName); err != nil {
        if e.progress != nil {
            e.progress.FailPhase(phaseInfo, err)
        }
        return err
    }

    // Success
    if e.progress != nil {
        e.progress.CompletePhase(phaseInfo)
    }
    return nil
}
```

#### Step 4.3: Add helper methods (10 min)

```go
func (e *Executor) getPhaseNumber(phase Phase) int {
    // Map Phase enum to sequential numbers
    switch phase {
    case PhaseSpecify: return 1
    case PhasePlan: return 2
    case PhaseTasks: return 3
    case PhaseImplement: return 4
    default: return 0
    }
}

func (e *Executor) getTotalPhases() int {
    // Determine from workflow type (set during construction)
    return e.totalPhases
}
```

#### Step 4.4: Update tests (10 min)

Update existing `executor_test.go` to pass `nil` for `progressDisplay` parameter (backward compatibility).

---

### Phase 5: Integration with CLI Commands (1 hour)

#### Step 5.1: Modify `workflow.go` command (20 min)

In `internal/cli/workflow.go`:

```go
func (c *WorkflowCommand) RunE(cmd *cobra.Command, args []string) error {
    // ... existing setup ...

    // NEW: Create progress display
    caps := progress.DetectTerminalCapabilities()
    var display *progress.Display
    if caps.IsTTY {
        display = progress.NewProgressDisplay(caps)
    }

    // Pass progress display to executor
    executor := workflow.NewExecutor(claudeExecutor, retryManager, display)

    // ... rest of existing code ...
}
```

#### Step 5.2: Apply same pattern to other commands (30 min)

Update `full.go`, `specify.go`, `plan.go`, `tasks.go`, `implement.go` with same pattern.

#### Step 5.3: Manual testing (10 min)

```bash
# Build and test
make build

# Test with TTY (should show spinners)
./autospec workflow "test feature"

# Test with pipe (should show simple output)
./autospec workflow "test feature" | cat

# Test with NO_COLOR
NO_COLOR=1 ./autospec workflow "test feature"

# Test with ASCII mode
AUTOSPEC_ASCII=1 ./autospec workflow "test feature"
```

---

### Phase 6: Benchmark Tests & Performance Validation (30 min)

#### Step 6.1: Create benchmark tests (20 min)

In `progress_bench_test.go`:

```go
func BenchmarkProgressDisplay_StartPhase(b *testing.B) {
    caps := progress.TerminalCapabilities{IsTTY: false} // Avoid spinner overhead
    display := progress.NewProgressDisplay(caps)
    phase := progress.PhaseInfo{
        Name: "test", Number: 1, TotalPhases: 3,
    }

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        display.StartPhase(phase)
    }
}

func BenchmarkProgressDisplay_CompletePhase(b *testing.B) {
    caps := progress.TerminalCapabilities{IsTTY: false}
    display := progress.NewProgressDisplay(caps)
    phase := progress.PhaseInfo{
        Name: "test", Number: 1, TotalPhases: 3,
    }

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        display.CompletePhase(phase)
    }
}

// Spinner animation CPU test
func BenchmarkSpinnerAnimation(b *testing.B) {
    s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
    s.Start()
    defer s.Stop()

    time.Sleep(5 * time.Second) // Run for 5 seconds
    // Use `go test -bench=BenchmarkSpinnerAnimation -cpuprofile=cpu.prof`
    // Then analyze with `go tool pprof cpu.prof` to verify <0.1% CPU
}
```

#### Step 6.2: Run benchmarks and validate (10 min)

```bash
go test -bench=. ./internal/progress/

# Expected results (validate against NFR-001, NFR-002):
# BenchmarkProgressDisplay_StartPhase: <10ms per operation
# BenchmarkProgressDisplay_CompletePhase: <10ms per operation
```

---

## Verification Checklist

After implementation, verify all requirements:

### Functional Requirements (FR)

- [ ] FR-001: Phase counter displayed in [current/total] format ✅
- [ ] FR-002: Terminal capabilities detected (TTY, color, Unicode) ✅
- [ ] FR-003: Spinner shown for operations >2s ✅
- [ ] FR-004: Checkmark (✓) shown on phase completion ✅
- [ ] FR-005: Failure indicator (✗) shown on phase failure ✅
- [ ] FR-006: Progress persists in terminal scrollback ✅
- [ ] FR-007: Terminal resize handled gracefully ✅
- [ ] FR-008: Animations disabled when not TTY ✅
- [ ] FR-009: Phase counter reflects actual phases executed ✅
- [ ] FR-010: Progress doesn't interfere with command output ✅
- [ ] FR-011: Spinner doesn't cause excessive CPU usage ✅
- [ ] FR-012: Current phase always visible ✅
- [ ] FR-013: Visual distinction between in-progress/completed/failed ✅

### Non-Functional Requirements (NFR)

- [ ] NFR-001: Progress overhead <100ms per update (benchmark verified) ✅
- [ ] NFR-002: Spinner 4-10 fps (configured at 10 fps / 100ms) ✅
- [ ] NFR-003: Compatible with target terminals (manual testing) ✅
- [ ] NFR-004: Graceful degradation in limited terminals ✅

### Success Criteria (SC)

- [ ] SC-001: Current phase identifiable within 1 second ✅
- [ ] SC-002: Completion percentage visible during execution ✅
- [ ] SC-005: Workflow time increase <5% (benchmark verified) ✅
- [ ] SC-007: Spinner confirms system responsiveness within 2s ✅

### Constitution Compliance

- [ ] Test-First Development: All tests written before implementation ✅
- [ ] Performance Standards: Sub-100ms validation (<10ms achieved) ✅
- [ ] Code Quality: Passes `go vet`, `go fmt`, `go test` ✅

---

## Common Pitfalls & Solutions

### Pitfall 1: Spinner not stopping on error

**Problem**: Spinner keeps running after phase fails, cursor hidden

**Solution**: Always call `spinner.Stop()` in error paths:
```go
if err != nil {
    if e.progress != nil {
        e.progress.FailPhase(phaseInfo, err) // This stops spinner
    }
    return err
}
```

### Pitfall 2: ANSI codes in piped output

**Problem**: When output is piped/redirected, ANSI codes cause garbage characters

**Solution**: Use `term.IsTerminal()` to detect TTY and disable codes:
```go
caps := progress.DetectTerminalCapabilities()
// caps.IsTTY == false when piped → no ANSI codes emitted
```

### Pitfall 3: Tests failing due to stdout capture

**Problem**: Tests that capture stdout conflict with spinner output

**Solution**: Use non-TTY capabilities in tests:
```go
caps := progress.TerminalCapabilities{IsTTY: false} // No spinner
display := progress.NewProgressDisplay(caps)
```

### Pitfall 4: Phase number doesn't match workflow

**Problem**: Full workflow shows [1/3] instead of [1/4] for implement phase

**Solution**: Pass correct `TotalPhases` to `Executor`:
```go
// In workflow orchestrator
executor.SetTotalPhases(4) // for full workflow
executor.SetTotalPhases(3) // for workflow (no implement)
```

---

## Testing Strategy

### Unit Tests (internal/progress/)

```bash
go test -v ./internal/progress/
# Expected: ~20-30 tests covering all methods and edge cases
```

### Integration Tests (internal/workflow/, internal/cli/)

```bash
go test -v ./internal/workflow/
go test -v ./internal/cli/
# Verify existing tests still pass with progress display injected
```

### Manual Tests (cross-platform)

```bash
# Linux/macOS
./autospec workflow "test"
NO_COLOR=1 ./autospec workflow "test"
AUTOSPEC_ASCII=1 ./autospec workflow "test"
./autospec workflow "test" | tee output.txt  # Verify no ANSI codes

# Windows (if available)
autospec.exe workflow "test"
```

### Performance Tests

```bash
go test -bench=. -benchmem ./internal/progress/
# Verify:
# - StartPhase: <10ms, <1KB alloc
# - CompletePhase: <10ms, <1KB alloc
```

---

## Rollout Plan

### Step 1: Merge to feature branch
```bash
git checkout 004-workflow-progress-indicators
git add internal/progress/ internal/workflow/ internal/cli/
git commit -m "feat: add workflow progress indicators"
```

### Step 2: Update documentation
- Add progress indicator section to README.md
- Document NO_COLOR and AUTOSPEC_ASCII env vars

### Step 3: Validation
```bash
make test      # All tests pass
make lint      # No linting errors
make build     # Binary builds successfully
./autospec doctor  # Verify dependencies
```

### Step 4: Manual QA
- Test on Linux, macOS, Windows (if available)
- Test in various terminal emulators
- Test in CI/CD environment (GitHub Actions)

### Step 5: Create PR
```bash
git push origin 004-workflow-progress-indicators
# Open PR to main branch
# Include screenshots/GIFs of progress indicators in action
```

---

## Success Metrics

After implementation, measure:

1. **User Feedback**: Survey 5-10 users on progress indicator usefulness (target: 4/5 satisfaction)
2. **Performance**: Measure workflow execution time before/after (target: <5% increase)
3. **Bug Reports**: Track issues related to progress display (target: 0 critical bugs in first week)
4. **Terminal Compatibility**: Test on 5 major terminals (target: 100% compatibility)

---

## References

- [Feature Spec](./spec.md) - User stories and acceptance criteria
- [Research](./research.md) - Technology decisions
- [Data Model](./data-model.md) - Entity definitions
- [API Contract](./contracts/progress_package_api.md) - Detailed API documentation
- [briandowns/spinner docs](https://github.com/briandowns/spinner)
- [golang.org/x/term docs](https://pkg.go.dev/golang.org/x/term)

---

## Need Help?

- **Questions about design**: Refer to research.md for rationale
- **API usage questions**: See contracts/progress_package_api.md examples
- **Test failures**: Check display_test.go for reference implementations
- **Performance issues**: Run benchmarks with `-cpuprofile` and analyze

---

**Estimated Implementation Time**: 4-6 hours (with tests)
**Difficulty**: Moderate (straightforward API, main complexity is spinner lifecycle)
**Priority**: P1 (foundation for P2 and P3 user stories)
