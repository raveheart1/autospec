# Data Model: Workflow Progress Indicators

**Feature**: 004-workflow-progress-indicators
**Date**: 2025-10-23
**Status**: Complete

## Overview

This document defines the core entities and data structures for the progress indicator system. The model focuses on tracking phase execution state, terminal capabilities, and display configuration.

---

## Entity 1: PhaseInfo

**Purpose**: Represents metadata about a workflow phase for progress display

### Fields

| Field | Type | Description | Validation Rules |
|-------|------|-------------|-----------------|
| `Name` | `string` | Human-readable phase name (e.g., "specify", "plan", "tasks", "implement") | Required, non-empty |
| `Number` | `int` | Current phase number (1-based index) | Must be > 0 and ≤ TotalPhases |
| `TotalPhases` | `int` | Total number of phases in workflow | Must be > 0, typically 3 or 4 |
| `Status` | `PhaseStatus` | Current execution status | Must be one of: Pending, InProgress, Completed, Failed |
| `RetryCount` | `int` | Number of retry attempts (0 if first attempt) | Must be ≥ 0 |
| `MaxRetries` | `int` | Maximum retry attempts allowed | Must be ≥ 0 |

### Relationships

- **PhaseInfo → ProgressDisplay**: PhaseInfo instances are passed to ProgressDisplay methods to update the display
- **PhaseInfo → Executor**: Created by workflow.Executor when executing phases

### State Transitions

```
Pending → InProgress → Completed
              ↓
           Failed (can retry if RetryCount < MaxRetries)
              ↓
           InProgress (retry attempt)
```

### Example

```go
type PhaseStatus int

const (
    PhasePending PhaseStatus = iota
    PhaseInProgress
    PhaseCompleted
    PhaseFailed
)

type PhaseInfo struct {
    Name        string
    Number      int
    TotalPhases int
    Status      PhaseStatus
    RetryCount  int
    MaxRetries  int
}

// Example usage
phase := PhaseInfo{
    Name:        "specify",
    Number:      1,
    TotalPhases: 3,
    Status:      PhaseInProgress,
    RetryCount:  0,
    MaxRetries:  3,
}
```

---

## Entity 2: TerminalCapabilities

**Purpose**: Encapsulates detected terminal features to determine display modes

### Fields

| Field | Type | Description | Validation Rules |
|-------|------|-------------|-----------------|
| `IsTTY` | `bool` | Whether stdout is a terminal (vs pipe/redirect) | Derived from term.IsTerminal() |
| `SupportsColor` | `bool` | Whether terminal supports ANSI color codes | false if NO_COLOR env var set or !IsTTY |
| `SupportsUnicode` | `bool` | Whether terminal supports Unicode characters | false if AUTOSPEC_ASCII=1 or !IsTTY |
| `Width` | `int` | Terminal width in columns (0 if unknown/pipe) | Must be ≥ 0 |

### Relationships

- **TerminalCapabilities → ProgressDisplay**: Injected into ProgressDisplay at construction time to configure display mode

### Example

```go
type TerminalCapabilities struct {
    IsTTY           bool
    SupportsColor   bool
    SupportsUnicode bool
    Width           int
}

// Factory function
func DetectTerminalCapabilities() TerminalCapabilities {
    isTTY := term.IsTerminal(int(os.Stdout.Fd()))
    noColor := os.Getenv("NO_COLOR") != ""
    forceASCII := os.Getenv("AUTOSPEC_ASCII") == "1"

    width, _, err := term.GetSize(int(os.Stdout.Fd()))
    if err != nil {
        width = 0 // Unknown/unavailable
    }

    return TerminalCapabilities{
        IsTTY:           isTTY,
        SupportsColor:   isTTY && !noColor,
        SupportsUnicode: isTTY && !forceASCII,
        Width:           width,
    }
}
```

---

## Entity 3: ProgressDisplay

**Purpose**: Main orchestrator for displaying progress indicators, spinners, and status

### Fields

| Field | Type | Description | Validation Rules |
|-------|------|-------------|-----------------|
| `capabilities` | `TerminalCapabilities` | Terminal feature flags | Required, set at construction |
| `currentPhase` | `*PhaseInfo` | Currently executing phase (nil if none) | Optional |
| `spinner` | `*spinner.Spinner` | Active spinner instance (nil when stopped) | Optional |
| `symbols` | `ProgressSymbols` | Character set for checkmarks, failures | Required, set based on capabilities |

### Methods

| Method | Parameters | Returns | Description |
|--------|-----------|---------|-------------|
| `NewProgressDisplay` | `capabilities TerminalCapabilities` | `*ProgressDisplay` | Constructor, initializes display based on terminal capabilities |
| `StartPhase` | `phase PhaseInfo` | `error` | Begins displaying progress for a phase, starts spinner if TTY |
| `CompletePhase` | `phase PhaseInfo` | `error` | Stops spinner, displays checkmark and completion message |
| `FailPhase` | `phase PhaseInfo, err error` | `error` | Stops spinner, displays failure indicator and error |
| `UpdateRetry` | `phase PhaseInfo` | `error` | Updates display with retry count information |

### Relationships

- **ProgressDisplay → Spinner**: Owns and manages spinner.Spinner lifecycle
- **ProgressDisplay → TerminalCapabilities**: Uses capabilities to determine display mode
- **ProgressDisplay → PhaseInfo**: Receives phase information to display

### State Management

- Display is stateless between phases (no persistent state file)
- Current phase tracked only during execution for display purposes
- Spinner lifecycle tied to phase execution (start → stop)

### Example

```go
type ProgressDisplay struct {
    capabilities TerminalCapabilities
    currentPhase *PhaseInfo
    spinner      *spinner.Spinner
    symbols      ProgressSymbols
}

func NewProgressDisplay(caps TerminalCapabilities) *ProgressDisplay {
    return &ProgressDisplay{
        capabilities: caps,
        symbols:      selectSymbols(caps),
    }
}

func (p *ProgressDisplay) StartPhase(phase PhaseInfo) error {
    p.currentPhase = &phase

    // Format: [1/3] Running specify phase
    msg := fmt.Sprintf("[%d/%d] Running %s phase",
        phase.Number, phase.TotalPhases, phase.Name)

    if phase.RetryCount > 0 {
        msg += fmt.Sprintf(" (retry %d/%d)",
            phase.RetryCount+1, phase.MaxRetries)
    }

    if p.capabilities.IsTTY {
        // Start spinner animation
        p.spinner = spinner.New(
            spinner.CharSets[p.symbols.SpinnerSet],
            100*time.Millisecond,
        )
        p.spinner.Suffix = " " + msg
        p.spinner.Start()
    } else {
        // Non-interactive: just print message
        fmt.Println(msg)
    }

    return nil
}

func (p *ProgressDisplay) CompletePhase(phase PhaseInfo) error {
    if p.spinner != nil {
        p.spinner.Stop()
        p.spinner = nil
    }

    // Format: ✓ [1/3] Specify phase complete
    checkmark := p.symbols.Checkmark
    if p.capabilities.SupportsColor {
        checkmark = "\033[32m" + checkmark + "\033[0m" // Green
    }

    fmt.Printf("%s [%d/%d] %s phase complete\n",
        checkmark, phase.Number, phase.TotalPhases,
        capitalize(phase.Name))

    p.currentPhase = nil
    return nil
}

func (p *ProgressDisplay) FailPhase(phase PhaseInfo, err error) error {
    if p.spinner != nil {
        p.spinner.Stop()
        p.spinner = nil
    }

    // Format: ✗ [1/3] Specify phase failed: <error>
    failMark := p.symbols.Failure
    if p.capabilities.SupportsColor {
        failMark = "\033[31m" + failMark + "\033[0m" // Red
    }

    fmt.Printf("%s [%d/%d] %s phase failed: %v\n",
        failMark, phase.Number, phase.TotalPhases,
        capitalize(phase.Name), err)

    p.currentPhase = nil
    return nil
}
```

---

## Entity 4: ProgressSymbols

**Purpose**: Character set configuration for visual indicators

### Fields

| Field | Type | Description | Validation Rules |
|-------|------|-------------|-----------------|
| `Checkmark` | `string` | Success indicator | "✓" (Unicode) or "[OK]" (ASCII) |
| `Failure` | `string` | Failure indicator | "✗" (Unicode) or "[FAIL]" (ASCII) |
| `SpinnerSet` | `int` | Index into spinner.CharSets | Must be valid index (0-90) |

### Example

```go
type ProgressSymbols struct {
    Checkmark  string
    Failure    string
    SpinnerSet int
}

func selectSymbols(caps TerminalCapabilities) ProgressSymbols {
    if caps.SupportsUnicode {
        return ProgressSymbols{
            Checkmark:  "✓",
            Failure:    "✗",
            SpinnerSet: 14, // Unicode dots: ⠋ ⠙ ⠹ ⠸ ⠼ ⠴ ⠦ ⠧ ⠇ ⠏
        }
    }

    return ProgressSymbols{
        Checkmark:  "[OK]",
        Failure:    "[FAIL]",
        SpinnerSet: 9, // ASCII: | / - \
    }
}
```

---

## Data Flow

### Workflow Execution with Progress Display

```
1. CLI Command (e.g., autospec workflow)
   ↓
2. Detect terminal capabilities (DetectTerminalCapabilities)
   ↓
3. Create ProgressDisplay with capabilities
   ↓
4. Pass ProgressDisplay to workflow.Executor
   ↓
5. For each phase:
   a. Executor calls display.StartPhase(phase)
      → Displays "[1/3] Running specify phase" + spinner
   b. Executor executes phase (Claude CLI, validation)
   c. On success: display.CompletePhase(phase)
      → Displays "✓ [1/3] Specify phase complete"
   d. On failure: display.FailPhase(phase, err)
      → Displays "✗ [1/3] Specify phase failed: <error>"
   e. On retry: display.UpdateRetry(phase)
      → Updates to show retry count
   ↓
6. Workflow completes
```

### Spinner Lifecycle

```
StartPhase called
   ↓
Check if IsTTY
   ↓ yes
Create spinner.Spinner with 100ms interval
   ↓
spinner.Start() → goroutine begins animation
   ↓
... phase executes (1-5 minutes) ...
   ↓
CompletePhase or FailPhase called
   ↓
spinner.Stop() → goroutine stops, cleanup
   ↓
Print final status with checkmark/failure indicator
```

---

## Validation Rules Summary

### PhaseInfo Validation
```go
func (p PhaseInfo) Validate() error {
    if p.Name == "" {
        return errors.New("phase name cannot be empty")
    }
    if p.Number <= 0 {
        return errors.New("phase number must be > 0")
    }
    if p.Number > p.TotalPhases {
        return errors.New("phase number cannot exceed total phases")
    }
    if p.TotalPhases <= 0 {
        return errors.New("total phases must be > 0")
    }
    if p.RetryCount < 0 {
        return errors.New("retry count cannot be negative")
    }
    if p.MaxRetries < 0 {
        return errors.New("max retries cannot be negative")
    }
    return nil
}
```

### TerminalCapabilities Validation
```go
func (tc TerminalCapabilities) Validate() error {
    if tc.Width < 0 {
        return errors.New("terminal width cannot be negative")
    }
    // Color/Unicode support must be false if not TTY
    if !tc.IsTTY && (tc.SupportsColor || tc.SupportsUnicode) {
        return errors.New("color/unicode support requires TTY")
    }
    return nil
}
```

---

## Package Structure

All entities will live in `internal/progress/` package:

```
internal/progress/
├── types.go           # PhaseInfo, PhaseStatus, TerminalCapabilities, ProgressSymbols
├── types_test.go      # Validation tests for types
├── display.go         # ProgressDisplay struct and methods
├── display_test.go    # Unit tests for display logic
├── terminal.go        # DetectTerminalCapabilities, selectSymbols
└── terminal_test.go   # Tests for terminal detection
```

---

## Test Coverage Requirements

Per constitution principle III (Test-First Development):

1. **Unit tests for PhaseInfo validation** (test invalid field combinations)
2. **Unit tests for TerminalCapabilities detection** (mock env vars, TTY state)
3. **Unit tests for ProgressDisplay methods** (verify output formatting)
4. **Unit tests for symbol selection** (Unicode vs ASCII based on caps)
5. **Integration tests** (full workflow with mocked spinner)
6. **Benchmark tests** (verify <100ms overhead per phase transition)

---

## Dependencies

- `github.com/briandowns/spinner` - Spinner animation (embedded in ProgressDisplay)
- `golang.org/x/term` - TTY detection and terminal size (used in TerminalCapabilities)

---

**Data Model Complete**: All entities defined with fields, relationships, and validation rules. Ready to proceed to contract generation.
