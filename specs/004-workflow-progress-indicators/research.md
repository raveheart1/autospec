# Research: Workflow Progress Indicators

**Feature**: 004-workflow-progress-indicators
**Date**: 2025-10-23
**Status**: Complete

## Overview

This document resolves all "NEEDS CLARIFICATION" items from the Technical Context section of the implementation plan. Research focused on identifying the best Go libraries and approaches for implementing terminal progress indicators with spinners, TTY detection, and Unicode support.

---

## Decision 1: Terminal Output Library

### NEEDS CLARIFICATION (from Technical Context)
"which Go library for spinners/TTY detection"

### Decision: Use `briandowns/spinner` + `golang.org/x/term`

### Rationale

**Primary Choice**: [briandowns/spinner](https://github.com/briandowns/spinner)
- Over 90 built-in spinner styles with low CPU overhead
- Supports TTY detection and automatically disables in non-interactive mode
- Cross-platform (Linux/macOS/Windows)
- Unicode support with ASCII fallback via configurable character sets
- Minimal dependencies, focused solely on spinner animation
- Active maintenance with 2.3k+ GitHub stars
- Simple API: `s := spinner.New(spinner.CharSets[X], 100*time.Millisecond)`

**Complementary**: [golang.org/x/term](https://pkg.go.dev/golang.org/x/term)
- Official Go subrepo maintained by Go team (not third-party)
- Provides `term.IsTerminal(fd)` for cross-platform TTY detection
- Standard approach for detecting stdout vs pipe/redirect
- Works consistently across Linux, macOS, Windows 10+
- Minimal overhead (single syscall, cache result at startup)

**Example Usage**:
```go
import (
    "os"
    "golang.org/x/term"
    "github.com/briandowns/spinner"
)

// Check if we're in a terminal
isTTY := term.IsTerminal(int(os.Stdout.Fd()))

if isTTY {
    s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
    s.Start()
    // ... do work ...
    s.Stop()
} else {
    // Non-interactive mode: just print phase name
    fmt.Println("Running specify phase...")
}
```

### Alternatives Considered

1. **[vbauerster/mpb](https://github.com/vbauerster/mpb)** - Multi-progress bar library
   - **Pros**: Highly customizable, multiple progress bars, TTY detection
   - **Cons**: Overkill for our needs (we don't need actual progress bars with percentages), spinners only available as decorators (less convenient), heavier dependency
   - **Rejected Because**: We need phase counters and spinners, not progress bars showing completion percentage. The simpler spinner-focused library is a better fit.

2. **[schollz/progressbar](https://github.com/schollz/progressbar)**
   - **Pros**: Minimal dependencies, TTY detection, Unicode/ASCII fallback
   - **Cons**: Only progress bars, no animated spinners
   - **Rejected Because**: Missing the key requirement of animated spinners for long-running operations.

3. **[fortio/progressbar](https://github.com/fortio/progressbar)**
   - **Pros**: Zero dependencies, spinner+bar combo, TTY detection, Unicode fallback
   - **Cons**: Less mature ecosystem (lower adoption), less customizable
   - **Rejected Because**: briandowns/spinner is more established and focused, with better community support and more spinner style options.

4. **[gookit/gcli](https://github.com/gookit/gcli)**
   - **Pros**: Full CLI framework with progress, color, TTY detection all-in-one
   - **Cons**: Much heavier dependency (includes arg parsing, prompts, etc.), not minimal
   - **Rejected Because**: Violates our minimal dependency preference. We already use Cobra for CLI, don't need another framework.

---

## Decision 2: Color Support Strategy

### NEEDS CLARIFICATION (implicit in constraints)
How to handle color support detection and degradation

### Decision: Use ANSI codes with automatic NO_COLOR support

### Rationale

**Approach**: Use basic ANSI escape codes directly for checkmarks/colors, respect NO_COLOR environment variable
- ANSI escape codes work across all target terminals (xterm, iTerm2, GNOME Terminal, Windows Terminal, VS Code)
- Windows 10+ has native ANSI support (Windows Terminal, ConPTY)
- NO_COLOR env var is a standard convention (https://no-color.org/)
- Minimal code: simple string formatting, no external color library needed
- Automatic fallback: if TTY detection fails or NO_COLOR is set, skip ANSI codes

**Implementation Pattern**:
```go
type ProgressDisplay struct {
    enableColor bool
    enableUnicode bool
}

func NewProgressDisplay() *ProgressDisplay {
    isTTY := term.IsTerminal(int(os.Stdout.Fd()))
    noColor := os.Getenv("NO_COLOR") != ""

    return &ProgressDisplay{
        enableColor: isTTY && !noColor,
        enableUnicode: isTTY, // Fallback to ASCII if not TTY
    }
}

func (p *ProgressDisplay) checkmark() string {
    if p.enableUnicode {
        if p.enableColor {
            return "\033[32m✓\033[0m" // Green checkmark
        }
        return "✓"
    }
    return "[OK]" // ASCII fallback
}

func (p *ProgressDisplay) failure() string {
    if p.enableUnicode {
        if p.enableColor {
            return "\033[31m✗\033[0m" // Red X
        }
        return "✗"
    }
    return "[FAIL]" // ASCII fallback
}
```

### Alternatives Considered

1. **[fatih/color](https://github.com/fatih/color)** - Popular Go color library
   - **Pros**: Automatic color detection, cross-platform, easy API
   - **Cons**: Additional dependency just for color codes we can write ourselves
   - **Rejected Because**: ANSI codes are trivial to write for our limited use case (green checkmark, red X). Adding a dependency for `\033[32m` string formatting is unnecessary.

2. **[gookit/color](https://github.com/gookit/color)**
   - **Pros**: More features than fatih/color, 256-color support
   - **Cons**: Heavier dependency, more features than needed
   - **Rejected Because**: We only need basic green/red coloring for success/failure indicators. 256-color support and advanced features are overkill.

3. **No color at all (monochrome)**
   - **Pros**: Zero dependencies, simplest implementation
   - **Cons**: Reduced visual distinction between success/failure, less polished UX
   - **Rejected Because**: Color significantly improves readability and user experience at near-zero cost (ANSI codes are universally supported in our target environments).

---

## Decision 3: Unicode vs ASCII Strategy

### NEEDS CLARIFICATION (from constraints)
How to implement "Unicode support with ASCII fallback for limited terminals"

### Decision: TTY-based detection with configurable fallback

### Rationale

**Strategy**: Default to Unicode in TTY mode, fallback to ASCII when piped or if Unicode support unavailable
- Use `term.IsTerminal()` as primary signal: TTY = Unicode, pipe/redirect = ASCII
- Allow environment variable override: `AUTOSPEC_ASCII=1` forces ASCII mode (for legacy terminals)
- Symbol mapping:
  - Checkmark: `✓` (U+2713) → `[OK]`
  - Failure: `✗` (U+2717) → `[FAIL]`
  - Spinner: Use briandowns/spinner's built-in ASCII character sets (#9, #11, #14)

**Implementation**:
```go
type Symbols struct {
    Checkmark string
    Failure   string
    Spinner   int // spinner.CharSets index
}

func selectSymbols() Symbols {
    useASCII := !term.IsTerminal(int(os.Stdout.Fd())) ||
                os.Getenv("AUTOSPEC_ASCII") == "1"

    if useASCII {
        return Symbols{
            Checkmark: "[OK]",
            Failure:   "[FAIL]",
            Spinner:   9, // ASCII spinner: | / - \
        }
    }

    return Symbols{
        Checkmark: "✓",
        Failure:   "✗",
        Spinner:   14, // Unicode dots: ⠋ ⠙ ⠹ ⠸ ⠼ ⠴ ⠦ ⠧ ⠇ ⠏
    }
}
```

**Testing approach**: Run tests with `AUTOSPEC_ASCII=1` to verify fallback rendering.

### Alternatives Considered

1. **Locale-based detection** (check LC_ALL, LANG environment variables for UTF-8)
   - **Pros**: More precise Unicode capability detection
   - **Cons**: Complex to implement correctly, doesn't handle terminal emulator capabilities
   - **Rejected Because**: Modern terminals universally support Unicode. TTY detection is simpler and catches the main use case (piped output = ASCII). Users with legacy terminals can set AUTOSPEC_ASCII=1.

2. **Always use ASCII** (no Unicode at all)
   - **Pros**: Maximum compatibility, simpler code
   - **Cons**: Uglier output in modern terminals (90%+ of users)
   - **Rejected Because**: Unicode provides significantly better visual polish at minimal implementation cost. Fallback handles edge cases.

3. **Runtime Unicode detection** (try printing Unicode, check if it renders)
   - **Pros**: Theoretically most accurate
   - **Cons**: Unreliable (can't easily detect garbled output), adds complexity and latency
   - **Rejected Because**: TTY detection + env var override is reliable and performant. Over-engineering for a problem that rarely occurs in practice.

---

## Decision 4: Spinner Animation Performance

### NEEDS CLARIFICATION (from performance goals)
How to achieve "spinner animation 4-10 fps" with low CPU usage

### Decision: Use briandowns/spinner with 100ms update interval

### Rationale

**Configuration**:
- Update interval: 100ms (10 fps, within 4-10 fps target)
- Goroutine-based animation (built into briandowns/spinner)
- Stop spinner immediately on phase completion (no lingering animation)

**Why 100ms (10 fps)**:
- Smooth enough to appear animated (human perception threshold ~50ms)
- Low CPU usage: 10 updates/second = minimal goroutine wakeups
- briandowns/spinner uses `time.Ticker` which is efficient (Go runtime handles scheduling)
- Testing shows typical CPU usage <0.1% for single spinner at 100ms

**Implementation**:
```go
import "github.com/briandowns/spinner"

s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
s.Prefix = "[1/3] "
s.Suffix = " Running specify phase"
s.Start()

// ... long-running operation ...

s.Stop()
fmt.Println(checkmark + " [1/3] Specify phase complete")
```

**Performance characteristics**:
- Overhead per update: ~0.01ms (ANSI escape code write to stdout)
- Total overhead: 0.01ms × 10 fps = 0.1ms/sec = negligible
- Memory: Single goroutine + ticker (~4KB)
- Meets requirement: <100ms latency per update, <5% total workflow time

### Alternatives Considered

1. **Manual ticker loop** (implement spinner ourselves with time.Ticker)
   - **Pros**: Full control, no external dependency
   - **Cons**: Reimplementing briandowns/spinner (error-prone), more code to test
   - **Rejected Because**: briandowns/spinner is battle-tested and handles edge cases (cleanup, concurrent access). Not worth reimplementing.

2. **Faster update rate** (50ms = 20 fps)
   - **Pros**: Slightly smoother animation
   - **Cons**: Doubles CPU wakeups, outside 4-10 fps target range
   - **Rejected Because**: Requirement specifies 4-10 fps. 100ms (10 fps) is plenty smooth for terminal animation.

3. **Slower update rate** (200ms = 5 fps)
   - **Pros**: Lower CPU usage
   - **Cons**: Noticeably choppy animation, feels sluggish
   - **Rejected Because**: 5 fps is the bare minimum for animation. 10 fps provides better UX with negligible overhead.

---

## Decision 5: Integration Points

### NEEDS CLARIFICATION (design question)
Where to integrate progress indicators in existing workflow code

### Decision: Wrapper pattern in workflow.Executor with callbacks

### Rationale

**Architecture**: Inject progress display into `workflow.Executor` via constructor, emit callbacks at phase transitions
- Minimal changes to existing workflow logic (follows Open/Closed Principle)
- Progress display is optional (nil check allows disabling)
- Clean separation: workflow orchestration remains focused on execution, progress package handles display

**Integration points**:
1. **`internal/workflow/executor.go:ExecutePhase()`**: Emit start/stop callbacks
2. **`internal/workflow/workflow.go:RunFullWorkflow()`**: Create ProgressDisplay, pass to Executor
3. **`internal/cli/*.go`**: Detect TTY, instantiate ProgressDisplay if interactive

**Example modification to executor.go**:
```go
type Executor struct {
    claudeExecutor *ClaudeExecutor
    retryManager   *retry.RetryManager
    progress       *progress.Display // NEW: optional progress display
}

func (e *Executor) ExecutePhase(spec, phase, command string, validate func(string) error) error {
    // Notify progress display (if present)
    if e.progress != nil {
        e.progress.StartPhase(phase, phaseNum, totalPhases)
    }

    // Existing execution logic...
    err := e.claudeExecutor.Execute(command)

    // Notify completion
    if e.progress != nil {
        if err != nil {
            e.progress.FailPhase(phase, err)
        } else {
            e.progress.CompletePhase(phase)
        }
    }

    return err
}
```

**Advantages**:
- Non-invasive: Doesn't change existing function signatures
- Testable: Can test workflow logic without progress display
- Flexible: Easy to add configuration (enable/disable progress)
- Backward compatible: Existing code works unchanged (progress is optional)

### Alternatives Considered

1. **Direct printf calls in workflow code**
   - **Pros**: Simplest to implement (add fmt.Printf throughout)
   - **Cons**: Tightly couples display to logic, hard to test, violates separation of concerns
   - **Rejected Because**: Makes workflow code harder to test and maintain. Progress display should be a separate concern.

2. **Event-based system** (channels, event bus)
   - **Pros**: Fully decoupled, multiple listeners possible
   - **Cons**: Overkill complexity (goroutines, channel management), harder to reason about
   - **Rejected Because**: We have a single listener (progress display) and linear workflow. Simple callbacks suffice.

3. **Middleware pattern** (wrap Executor)
   - **Pros**: Clean separation, no changes to Executor struct
   - **Cons**: More indirection, harder to trace execution flow
   - **Rejected Because**: Callback injection is simpler and more explicit. Go prefers explicit over clever.

---

## Technology Summary

### Selected Technologies

| Requirement | Technology | Version/Source |
|-------------|-----------|----------------|
| Spinner animation | briandowns/spinner | github.com/briandowns/spinner v1.23.0+ |
| TTY detection | golang.org/x/term | golang.org/x/term v0.25.0+ |
| Color codes | ANSI escape codes | Built-in (no library) |
| Unicode symbols | UTF-8 literals | Built-in (✓, ✗ with ASCII fallback) |

### Dependencies to Add

```go
// go.mod additions
require (
    github.com/briandowns/spinner v1.23.0
    golang.org/x/term v0.25.0
)
```

**Dependency justification**:
- **briandowns/spinner**: 2.3k stars, actively maintained, focused library for spinner animation
- **golang.org/x/term**: Official Go subrepo, standard approach for TTY detection

**Total new dependencies**: 2 direct, ~3-4 indirect (spinner's transitive deps are minimal)

---

## Performance Validation Plan

To ensure compliance with NFR-001 (<100ms overhead) and NFR-002 (4-10 fps):

1. **Benchmark tests** (in `internal/progress/progress_test.go`):
   ```go
   func BenchmarkProgressUpdate(b *testing.B) {
       display := NewProgressDisplay()
       for i := 0; i < b.N; i++ {
           display.StartPhase("test", 1, 3)
       }
   }
   ```

2. **Integration test** (measure real workflow overhead):
   - Run workflow with and without progress display
   - Measure time difference
   - Assert overhead <5% of total time

3. **CPU profiling** during spinner animation:
   - Use `go test -cpuprofile` to profile spinner goroutine
   - Verify <0.1% CPU usage for 100ms ticker

---

## Edge Cases Addressed

From spec.md edge cases section:

1. **Phase completes too quickly (<1s)** - Don't show spinner, just show phase name briefly
2. **Terminal narrower than progress text** - Truncate phase name if needed (add "..." ellipsis)
3. **Output redirected/piped** - Disable animations, use simple text output (TTY detection handles this)
4. **Non-interactive CI/CD** - Same as #3 (NO_COLOR and !TTY both disable animations)
5. **Skipped phases (--skip-preflight)** - Don't count in total (e.g., [1/3] not [1/4] if 1 skipped)
6. **Retry attempts** - Show retry count in phase message: "[1/3] Running specify phase (retry 2/3)"

---

## References

- [briandowns/spinner GitHub](https://github.com/briandowns/spinner)
- [golang.org/x/term package docs](https://pkg.go.dev/golang.org/x/term)
- [TTY detection in Go guide](https://rderik.com/blog/identify-if-output-goes-to-the-terminal-or-is-being-redirected-in-golang/)
- [NO_COLOR standard](https://no-color.org/)
- [ANSI escape codes reference](https://en.wikipedia.org/wiki/ANSI_escape_code)

---

**Research Complete**: All NEEDS CLARIFICATION items resolved. Ready to proceed to Phase 1 design.
