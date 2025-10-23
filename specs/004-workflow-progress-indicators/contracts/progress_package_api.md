# API Contract: internal/progress Package

**Feature**: 004-workflow-progress-indicators
**Date**: 2025-10-23
**Package**: `internal/progress`

## Overview

This contract defines the public API of the `internal/progress` package, which provides progress indicator functionality for autospec CLI workflows. The package is consumed by `internal/workflow` and `internal/cli` packages.

---

## Package-Level Constants

### PhaseStatus Enumeration

```go
type PhaseStatus int

const (
    PhasePending PhaseStatus = iota  // Phase has not started
    PhaseInProgress                  // Phase is currently executing
    PhaseCompleted                   // Phase completed successfully
    PhaseFailed                      // Phase failed validation
)
```

**Usage**: Used to track the current state of a workflow phase.

**String Representation**:
```go
func (s PhaseStatus) String() string
```
Returns: "pending", "in_progress", "completed", or "failed"

---

## Type: PhaseInfo

**Purpose**: Metadata about a workflow phase for display purposes

```go
type PhaseInfo struct {
    Name        string      // Phase name (e.g., "specify", "plan", "tasks", "implement")
    Number      int         // 1-based phase number in workflow
    TotalPhases int         // Total phases in workflow (3 for workflow, 4 for full)
    Status      PhaseStatus // Current phase status
    RetryCount  int         // Number of retry attempts (0 = first attempt)
    MaxRetries  int         // Maximum allowed retries
}
```

### Methods

#### Validate

```go
func (p PhaseInfo) Validate() error
```

**Description**: Validates PhaseInfo fields for consistency

**Returns**:
- `nil` if valid
- `error` if any validation rule fails

**Validation Rules**:
- `Name` must not be empty
- `Number` must be > 0 and ≤ `TotalPhases`
- `TotalPhases` must be > 0
- `RetryCount` must be ≥ 0
- `MaxRetries` must be ≥ 0

**Example**:
```go
phase := PhaseInfo{Name: "specify", Number: 1, TotalPhases: 3}
if err := phase.Validate(); err != nil {
    log.Fatal(err)
}
```

---

## Type: TerminalCapabilities

**Purpose**: Encapsulates detected terminal features

```go
type TerminalCapabilities struct {
    IsTTY           bool // stdout is a terminal (not pipe/redirect)
    SupportsColor   bool // Terminal supports ANSI color codes
    SupportsUnicode bool // Terminal supports Unicode characters
    Width           int  // Terminal width in columns (0 if unknown)
}
```

### Factory Function

#### DetectTerminalCapabilities

```go
func DetectTerminalCapabilities() TerminalCapabilities
```

**Description**: Detects current terminal capabilities using environment and system calls

**Returns**: `TerminalCapabilities` with detected features

**Detection Logic**:
- `IsTTY`: Uses `term.IsTerminal(os.Stdout.Fd())`
- `SupportsColor`: `IsTTY && NO_COLOR env var not set`
- `SupportsUnicode`: `IsTTY && AUTOSPEC_ASCII env var != "1"`
- `Width`: Uses `term.GetSize(os.Stdout.Fd())`, 0 if unavailable

**Example**:
```go
caps := progress.DetectTerminalCapabilities()
if caps.IsTTY {
    fmt.Println("Running in interactive terminal")
}
```

---

## Type: ProgressDisplay

**Purpose**: Main API for displaying workflow progress

```go
type ProgressDisplay struct {
    // Unexported fields: capabilities, currentPhase, spinner, symbols
}
```

### Constructor

#### NewProgressDisplay

```go
func NewProgressDisplay(capabilities TerminalCapabilities) *ProgressDisplay
```

**Description**: Creates a new progress display configured for the detected terminal

**Parameters**:
- `capabilities`: Terminal feature flags (from `DetectTerminalCapabilities()`)

**Returns**: Configured `*ProgressDisplay` ready for use

**Example**:
```go
caps := progress.DetectTerminalCapabilities()
display := progress.NewProgressDisplay(caps)
```

---

### Methods

#### StartPhase

```go
func (p *ProgressDisplay) StartPhase(phase PhaseInfo) error
```

**Description**: Begins displaying progress for a workflow phase

**Parameters**:
- `phase`: Phase metadata (must pass `Validate()`)

**Behavior**:
- If TTY: Starts animated spinner with phase message
- If non-TTY: Prints static phase message
- Message format: `[N/Total] Running <name> phase`
- If retrying: Appends `(retry X/MaxRetries)` to message

**Returns**:
- `nil` on success
- `error` if phase validation fails

**Side Effects**:
- Sets internal `currentPhase` field
- Creates and starts spinner goroutine (if TTY)
- Writes to `os.Stdout`

**Example**:
```go
phase := PhaseInfo{Name: "specify", Number: 1, TotalPhases: 3}
if err := display.StartPhase(phase); err != nil {
    return err
}
// Output (TTY): [1/3] ⠋ Running specify phase (spinner animating)
// Output (non-TTY): [1/3] Running specify phase
```

---

#### CompletePhase

```go
func (p *ProgressDisplay) CompletePhase(phase PhaseInfo) error
```

**Description**: Marks a phase as successfully completed

**Parameters**:
- `phase`: Phase metadata (should match current phase)

**Behavior**:
- Stops spinner animation (if active)
- Prints completion message with checkmark
- Message format: `✓ [N/Total] <Name> phase complete`
- Checkmark colored green if color supported

**Returns**:
- `nil` on success
- `error` if phase validation fails

**Side Effects**:
- Stops and cleans up spinner goroutine
- Clears internal `currentPhase` field
- Writes to `os.Stdout`

**Example**:
```go
err := display.CompletePhase(phase)
// Output (TTY, color): \033[32m✓\033[0m [1/3] Specify phase complete
// Output (TTY, no color): ✓ [1/3] Specify phase complete
// Output (non-TTY): [OK] [1/3] Specify phase complete
```

---

#### FailPhase

```go
func (p *ProgressDisplay) FailPhase(phase PhaseInfo, err error) error
```

**Description**: Marks a phase as failed with error message

**Parameters**:
- `phase`: Phase metadata (should match current phase)
- `err`: Error that caused the failure (displayed in message)

**Behavior**:
- Stops spinner animation (if active)
- Prints failure message with failure indicator
- Message format: `✗ [N/Total] <Name> phase failed: <err>`
- Failure indicator colored red if color supported

**Returns**:
- `nil` on success
- `error` if phase validation fails

**Side Effects**:
- Stops and cleans up spinner goroutine
- Clears internal `currentPhase` field
- Writes to `os.Stdout`

**Example**:
```go
err := display.FailPhase(phase, errors.New("validation failed"))
// Output (TTY, color): \033[31m✗\033[0m [1/3] Specify phase failed: validation failed
// Output (TTY, no color): ✗ [1/3] Specify phase failed: validation failed
// Output (non-TTY): [FAIL] [1/3] Specify phase failed: validation failed
```

---

#### UpdateRetry

```go
func (p *ProgressDisplay) UpdateRetry(phase PhaseInfo) error
```

**Description**: Updates display to show retry attempt information

**Parameters**:
- `phase`: Phase metadata with updated `RetryCount`

**Behavior**:
- Restarts phase display with retry count in message
- Internally calls `StartPhase` with updated phase info
- Message includes `(retry X/MaxRetries)` suffix

**Returns**:
- `nil` on success
- `error` if phase validation fails

**Side Effects**:
- Stops existing spinner (if any)
- Starts new spinner with updated message
- Writes to `os.Stdout`

**Example**:
```go
phase.RetryCount = 1  // Second attempt
err := display.UpdateRetry(phase)
// Output: [1/3] ⠋ Running specify phase (retry 2/3)
```

---

## Integration Contract: workflow.Executor

**Modification Points**: The `workflow.Executor` will be modified to integrate progress display.

### Constructor Change

```go
// BEFORE (current)
func NewExecutor(claudeExecutor *ClaudeExecutor, retryManager *retry.RetryManager) *Executor

// AFTER (with progress)
func NewExecutor(
    claudeExecutor *ClaudeExecutor,
    retryManager *retry.RetryManager,
    progressDisplay *progress.ProgressDisplay, // NEW: optional (can be nil)
) *Executor
```

**Behavior**: If `progressDisplay` is `nil`, no progress indicators shown (backward compatible).

---

### ExecutePhase Integration

**Current signature** (no changes):
```go
func (e *Executor) ExecutePhase(
    specName string,
    phase Phase,
    slashCommand string,
    validationFunc func(string) error,
) error
```

**Integration pseudocode**:
```go
func (e *Executor) ExecutePhase(...) error {
    // Map Phase enum to PhaseInfo
    phaseInfo := e.buildPhaseInfo(phase)

    // Start progress display
    if e.progressDisplay != nil {
        e.progressDisplay.StartPhase(phaseInfo)
    }

    // Execute phase (existing logic)
    err := e.claudeExecutor.Execute(slashCommand)

    // Update progress based on result
    if e.progressDisplay != nil {
        if err != nil {
            e.progressDisplay.FailPhase(phaseInfo, err)
        } else {
            e.progressDisplay.CompletePhase(phaseInfo)
        }
    }

    return err
}
```

---

## Integration Contract: CLI Commands

**Modification Points**: CLI commands (`full.go`, `workflow.go`, etc.) will instantiate and pass `ProgressDisplay`.

### Example: autospec workflow command

```go
// In internal/cli/workflow.go RunE function

func (cmd *WorkflowCommand) RunE(cmd *cobra.Command, args []string) error {
    // ... existing setup ...

    // NEW: Detect terminal capabilities
    caps := progress.DetectTerminalCapabilities()
    var display *progress.ProgressDisplay
    if caps.IsTTY {
        display = progress.NewProgressDisplay(caps)
    }
    // If not TTY, display stays nil (no progress indicators)

    // Create executor with progress display
    executor := workflow.NewExecutor(claudeExecutor, retryManager, display)

    // Run workflow (existing code, now with progress)
    return orchestrator.RunCompleteWorkflow(specName, executor)
}
```

---

## Error Handling Contract

### Error Propagation

- `ProgressDisplay` methods return errors for invalid inputs (failed validation)
- Display errors are **non-fatal**: if a display method fails, the workflow continues
- Workflow execution errors (from `Executor`) take precedence over display errors

### Error Recovery

```go
// Recommended pattern in Executor
if e.progressDisplay != nil {
    if err := e.progressDisplay.StartPhase(phaseInfo); err != nil {
        // Log but don't fail workflow
        log.Printf("Warning: progress display error: %v", err)
    }
}
```

---

## Performance Contract

**Requirements** (from NFR-001, NFR-002):
- Progress update overhead: <100ms per call
- Spinner animation: 4-10 fps (target: 10 fps = 100ms interval)
- Total workflow overhead: <5% of execution time

**Guarantees**:
- `StartPhase`: <10ms (spinner creation + stdout write)
- `CompletePhase`: <10ms (spinner stop + stdout write)
- `FailPhase`: <10ms (spinner stop + stdout write)
- Spinner animation: 100ms interval, <0.1% CPU usage

**Testing**: Benchmark tests in `progress_test.go` validate these constraints.

---

## Thread Safety

**Guarantees**:
- `ProgressDisplay` is **not thread-safe** (single-threaded use expected)
- Workflow phases execute sequentially (no concurrent phase execution)
- Spinner goroutine is managed internally (safe)

**Constraints**:
- Do not call `ProgressDisplay` methods from multiple goroutines
- Do not share `ProgressDisplay` across concurrent workflows

---

## Dependencies

### External Packages

```go
import (
    "github.com/briandowns/spinner"  // Spinner animation
    "golang.org/x/term"               // TTY detection
)
```

### Standard Library

```go
import (
    "errors"
    "fmt"
    "os"
    "strings"
    "time"
)
```

---

## Deprecation Policy

**Stability**: This is a new package (v1.0.0 when released)

**Breaking Changes**:
- Major version bump required for API signature changes
- Minor version bump for new methods (backward compatible)
- Patch version for bug fixes

**Backward Compatibility**:
- `ProgressDisplay` parameter in `Executor` is optional (nil = no progress)
- Existing workflows without progress display continue to work unchanged

---

## Testing Contract

### Unit Test Coverage (per constitution III)

1. `PhaseInfo.Validate()` - Test all validation rules
2. `DetectTerminalCapabilities()` - Mock env vars and TTY state
3. `NewProgressDisplay()` - Test symbol selection based on capabilities
4. `StartPhase()` - Verify output format (TTY and non-TTY modes)
5. `CompletePhase()` - Verify checkmark rendering (color and no-color)
6. `FailPhase()` - Verify failure indicator rendering
7. `UpdateRetry()` - Verify retry count in message

### Integration Test Coverage

1. Full workflow with progress display (mock spinner, verify output)
2. Non-TTY mode (verify no ANSI codes emitted)
3. Retry scenario (verify retry count updates)

### Benchmark Tests

1. `BenchmarkStartPhase` - Verify <10ms latency
2. `BenchmarkCompletePhase` - Verify <10ms latency
3. `BenchmarkSpinnerAnimation` - Verify low CPU usage

---

## Example: Complete Usage Flow

```go
package main

import (
    "github.com/anthropics/auto-claude-speckit/internal/progress"
    "github.com/anthropics/auto-claude-speckit/internal/workflow"
)

func main() {
    // 1. Detect terminal capabilities
    caps := progress.DetectTerminalCapabilities()

    // 2. Create progress display (only if TTY)
    var display *progress.ProgressDisplay
    if caps.IsTTY {
        display = progress.NewProgressDisplay(caps)
    }

    // 3. Create workflow executor with progress
    executor := workflow.NewExecutor(claudeExec, retryMgr, display)

    // 4. Execute phases (progress automatically displayed)
    phases := []progress.PhaseInfo{
        {Name: "specify", Number: 1, TotalPhases: 3},
        {Name: "plan", Number: 2, TotalPhases: 3},
        {Name: "tasks", Number: 3, TotalPhases: 3},
    }

    for _, phase := range phases {
        // Executor internally calls:
        // - display.StartPhase(phase)
        // - <execute phase logic>
        // - display.CompletePhase(phase) or display.FailPhase(phase, err)
        if err := executor.ExecutePhase(spec, phase, command, validate); err != nil {
            return err
        }
    }
}
```

**Output** (TTY mode with color):
```
[1/3] ⠋ Running specify phase
✓ [1/3] Specify phase complete
[2/3] ⠙ Running plan phase
✓ [2/3] Plan phase complete
[3/3] ⠹ Running tasks phase
✓ [3/3] Tasks phase complete
```

**Output** (non-TTY/piped):
```
[1/3] Running specify phase
[OK] [1/3] Specify phase complete
[2/3] Running plan phase
[OK] [2/3] Plan phase complete
[3/3] Running tasks phase
[OK] [3/3] Tasks phase complete
```

---

**Contract Version**: 1.0.0
**Status**: Ready for implementation
