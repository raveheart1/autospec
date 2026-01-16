# Executor API Contract: Timeout Enforcement

**Package**: `internal/workflow`
**Version**: 1.0.0
**Feature**: 003-command-timeout

## Overview

This contract defines the API for timeout enforcement in command execution. The ClaudeExecutor is enhanced to support optional timeout with automatic process termination and custom error handling.

---

## Types

### ClaudeExecutor

**Package**: `internal/workflow`
**Type**: `struct`

```go
type ClaudeExecutor struct {
    ClaudeCmd       string   // Path to claude CLI binary
    ClaudeArgs      []string // Additional arguments for claude command
    UseAPIKey       bool     // Whether to use ANTHROPIC_API_KEY
    CustomClaudeCmd string   // Custom command template with {{PROMPT}} placeholder
    Timeout         int      // NEW: Timeout in seconds (0 = no timeout)
}
```

**New Field**:

| Field | Type | Description | Valid Values |
|-------|------|-------------|--------------|
| Timeout | int | Command timeout in seconds | 0 (no timeout) or 1-3600 |

---

### TimeoutError

**Package**: `internal/workflow`
**Type**: `struct`

```go
type TimeoutError struct {
    Timeout time.Duration  // The timeout duration that was exceeded
    Command string         // The command that timed out
    Err     error          // Underlying context error (context.DeadlineExceeded)
}
```

**Methods**:

```go
// Error returns formatted error message with timeout details
func (e *TimeoutError) Error() string

// Unwrap returns underlying error for errors.Is/As compatibility
func (e *TimeoutError) Unwrap() error
```

**Error Message Format**:
```
command timed out after <duration>: <command> (hint: increase timeout in config)
```

**Example**:
```
command timed out after 5m0s: claude /speckit.implement (hint: increase timeout in config)
```

---

## Public API

### Execute

**Function**: `Execute(prompt string) error`
**Receiver**: `*ClaudeExecutor`

**Purpose**: Execute a Claude command with the given prompt, with optional timeout enforcement.

**Parameters**:
- `prompt`: The command or prompt to pass to Claude CLI (e.g., "/speckit.plan")

**Returns**:
- `error`: `nil` on success, `*TimeoutError` on timeout, or standard error on other failures

**Behavior**:

1. **No Timeout** (when `c.Timeout == 0`):
   ```go
   ctx := context.Background()
   cmd := exec.CommandContext(ctx, c.ClaudeCmd, args...)
   err := cmd.Run()
   return err  // Standard error handling
   ```

2. **With Timeout** (when `c.Timeout > 0`):
   ```go
   ctx, cancel := context.WithTimeout(context.Background(), time.Duration(c.Timeout)*time.Second)
   defer cancel()
   cmd := exec.CommandContext(ctx, c.ClaudeCmd, args...)
   err := cmd.Run()

   // Check for timeout
   if ctx.Err() == context.DeadlineExceeded {
       return &TimeoutError{
           Timeout: time.Duration(c.Timeout) * time.Second,
           Command: fmt.Sprintf("%s %s", c.ClaudeCmd, prompt),
           Err:     ctx.Err(),
       }
   }
   return err
   ```

**Process Termination**:
- `exec.CommandContext` automatically sends `os.Kill` (SIGKILL) when context deadline is exceeded
- Process termination is guaranteed (cannot be ignored)
- All child processes and file descriptors are cleaned up by OS

**Output Streaming**:
- `cmd.Stdout` and `cmd.Stderr` are connected to `os.Stdout` and `os.Stderr`
- Output is streamed in real-time until timeout or completion
- Partial output is visible to user before timeout

**Example Usage**:

```go
executor := &ClaudeExecutor{
    ClaudeCmd: "claude",
    Timeout:   300,  // 5 minutes
}

err := executor.Execute("/speckit.plan")
if err != nil {
    var timeoutErr *TimeoutError
    if errors.As(err, &timeoutErr) {
        fmt.Fprintf(os.Stderr, "Error: %v\n", timeoutErr)
        os.Exit(5)  // Timeout-specific exit code
    }
    return err  // Other error
}
```

---

### ExecuteSpecKitCommand

**Function**: `ExecuteSpecKitCommand(command string) error`
**Receiver**: `*ClaudeExecutor`

**Purpose**: Convenience wrapper for executing SpecKit slash commands with timeout.

**Parameters**:
- `command`: SpecKit slash command (e.g., "/speckit.specify", "/speckit.plan")

**Returns**:
- `error`: Same as `Execute()`

**Behavior**:
- Delegates to `Execute(command)`
- Subject to same timeout enforcement

**Example**:
```go
err := executor.ExecuteSpecKitCommand("/speckit.tasks")
```

---

### StreamCommand

**Function**: `StreamCommand(prompt string, stdout, stderr io.Writer) error`
**Receiver**: `*ClaudeExecutor`

**Purpose**: Execute command with custom output writers (useful for testing).

**Parameters**:
- `prompt`: Command or prompt to execute
- `stdout`: Writer for standard output
- `stderr`: Writer for standard error

**Returns**:
- `error`: Same as `Execute()`

**Behavior**:
- Same timeout enforcement as `Execute()`
- Output directed to provided writers instead of `os.Stdout`/`os.Stderr`

**Example** (testing):
```go
var stdout, stderr bytes.Buffer
err := executor.StreamCommand("/speckit.specify", &stdout, &stderr)
if err != nil {
    t.Fatalf("command failed: %v", err)
}
```

---

## Error Handling

### Error Type Hierarchy

```
error (interface)
  │
  ├─ *TimeoutError (custom)
  │   └─ Wraps: context.DeadlineExceeded
  │
  └─ *exec.ExitError (standard)
      └─ Non-zero exit code from command
```

### Error Detection

**Check for Timeout Error**:
```go
var timeoutErr *TimeoutError
if errors.As(err, &timeoutErr) {
    // Handle timeout specifically
    fmt.Fprintf(os.Stderr, "Command timed out after %v\n", timeoutErr.Timeout)
    fmt.Fprintf(os.Stderr, "Hint: Increase timeout in config\n")
    os.Exit(5)
}
```

**Check for Context Error**:
```go
if errors.Is(err, context.DeadlineExceeded) {
    // Timeout occurred (alternative check)
}
```

### Exit Codes

| Exit Code | Meaning | Error Type |
|-----------|---------|------------|
| 0 | Success | nil |
| 1 | Validation failed | Standard error |
| 2 | Retry limit exhausted | Standard error |
| 3 | Invalid arguments | Standard error |
| 4 | Missing dependencies | Standard error |
| 5 | **Timeout exceeded** | **TimeoutError** |

---

## Timeout Behavior

### Timeline

```
Time 0s          Timeout (e.g., 300s)
│                │
│ Command starts │
│────────────────│ Command completes → Success
│                │
│ Command starts │                Command still running
│────────────────│────────────────│
                 │ Timeout reached │
                 │ → Kill process  │
                 │ → TimeoutError  │
```

### Process Lifecycle

1. **Pre-execution**:
   - Create context with timeout (if `c.Timeout > 0`)
   - Create `exec.Cmd` with `exec.CommandContext(ctx, ...)`

2. **During execution**:
   - Command runs normally
   - Output streamed to stdout/stderr
   - Context timer counts down

3. **Timeout occurs**:
   - Context deadline exceeded
   - `exec.CommandContext` sends `os.Kill` to process
   - Process terminated immediately (SIGKILL)
   - `cmd.Wait()` returns (called by `cmd.Run()`)
   - `ctx.Err()` returns `context.DeadlineExceeded`

4. **Post-termination**:
   - Check `ctx.Err()` to detect timeout
   - Create and return `TimeoutError`
   - Cleanup handled by `defer cancel()`

---

## Integration Points

### Configuration Integration

**From**: `internal/config.Configuration`
**To**: `internal/workflow.ClaudeExecutor`

```go
// In workflow orchestrator or CLI command
cfg, err := config.Load(".autospec/config.json")
if err != nil {
    return err
}

executor := &ClaudeExecutor{
    ClaudeCmd:       cfg.ClaudeCmd,
    ClaudeArgs:      cfg.ClaudeArgs,
    UseAPIKey:       cfg.UseAPIKey,
    CustomClaudeCmd: cfg.CustomClaudeCmd,
    Timeout:         cfg.Timeout,  // NEW: Pass timeout to executor
}
```

### Workflow Orchestrator Integration

**Package**: `internal/workflow`
**Type**: `Executor` struct

**Modification**:
```go
type Executor struct {
    ClaudeExecutor  *ClaudeExecutor  // Already has Timeout set from config
    RetryManager    *retry.Manager
    // ...
}

func (e *Executor) ExecutePhase(specName, phase, command string, validationFunc ValidationFunc) error {
    // Execute command with timeout
    if err := e.ClaudeExecutor.Execute(command); err != nil {
        var timeoutErr *TimeoutError
        if errors.As(err, &timeoutErr) {
            // Timeout errors should be retried? Or fail immediately?
            // Decision: Fail immediately (timeout suggests fundamental issue)
            return fmt.Errorf("phase %s timed out: %w", phase, err)
        }
        // Handle other errors (retry logic)...
    }
    // ...
}
```

---

## Testing Contract

### Unit Tests

**File**: `internal/workflow/claude_test.go`

#### Test Cases

1. **TestExecute_NoTimeout_Success**
   - Setup: `Timeout = 0`
   - Execute: Fast-completing command
   - Expect: No error

2. **TestExecute_WithTimeout_CompletesBeforeTimeout**
   - Setup: `Timeout = 60`
   - Execute: Command completes in 1 second
   - Expect: No error, no timeout

3. **TestExecute_WithTimeout_ExceedsTimeout**
   - Setup: `Timeout = 2`
   - Execute: Mock command that sleeps for 10 seconds
   - Expect: `*TimeoutError` returned within ~2 seconds

4. **TestExecute_TimeoutError_IncludesMetadata**
   - Setup: Timeout occurs
   - Expect: `TimeoutError.Timeout` matches configured timeout
   - Expect: `TimeoutError.Command` contains command string
   - Expect: `TimeoutError.Err == context.DeadlineExceeded`

5. **TestExecute_ProcessCleanup_NoOrphanedProcesses**
   - Setup: Timeout occurs
   - Verify: No child processes remain after timeout (use `ps`)

6. **TestTimeoutError_ErrorMessage_Format**
   - Setup: Create `TimeoutError`
   - Expect: Error message includes timeout duration and hint

7. **TestTimeoutError_Unwrap_ReturnsUnderlyingError**
   - Setup: Create `TimeoutError` with underlying `context.DeadlineExceeded`
   - Expect: `errors.Is(err, context.DeadlineExceeded)` returns true

---

### Benchmark Tests

**File**: `internal/workflow/claude_bench_test.go`

#### Benchmarks

1. **BenchmarkExecute_NoTimeout**
   - Measure execution time without timeout

2. **BenchmarkExecute_WithTimeout**
   - Measure execution time with timeout enabled

3. **Performance Requirement**:
   - Overhead with timeout < 1% of execution time without timeout
   - Satisfies success criterion SC-004

**Example**:
```go
func BenchmarkExecute_NoTimeout(b *testing.B) {
    executor := &ClaudeExecutor{Timeout: 0}
    for i := 0; i < b.N; i++ {
        executor.Execute("echo test")
    }
}

func BenchmarkExecute_WithTimeout(b *testing.B) {
    executor := &ClaudeExecutor{Timeout: 300}
    for i := 0; i < b.N; i++ {
        executor.Execute("echo test")
    }
}
```

---

### Integration Tests

**File**: `integration/claude_test.go`

#### Test Scenarios

1. **TestIntegration_TimeoutWithRealClaude**
   - Setup: Real claude CLI (if available)
   - Execute: Long-running command with short timeout
   - Verify: Timeout occurs, exit code 5

2. **TestIntegration_ConfigToExecutor_TimeoutPropagation**
   - Setup: Load config with timeout
   - Create executor from config
   - Verify: Executor.Timeout matches config.Timeout

---

## Performance Contract

### Requirements

| Metric | Target | Validation |
|--------|--------|------------|
| Context creation overhead | < 100ns | Benchmark test |
| Timeout enforcement accuracy | Within 5 seconds | Integration test |
| Overall execution overhead | < 1% | Benchmark comparison |
| Process cleanup time | < 1 second | Integration test |

### Measurement

```bash
# Run benchmarks
go test -bench=BenchmarkExecute -benchmem ./internal/workflow/

# Compare results
# BenchmarkExecute_NoTimeout:    1000000 ns/op
# BenchmarkExecute_WithTimeout:  1005000 ns/op
# Overhead: 0.5% ✅
```

---

## Backward Compatibility

### Existing Behavior Preserved

**Scenario**: Executor created with `Timeout = 0`

```go
executor := &ClaudeExecutor{
    ClaudeCmd: "claude",
    Timeout:   0,  // No timeout
}

err := executor.Execute("/speckit.plan")
// Behavior: Identical to pre-timeout implementation
// No context timeout, no TimeoutError possible
```

**Guarantee**:
- Default `Timeout = 0` maintains existing behavior
- No breaking changes to callers who don't set timeout
- Timeout is opt-in feature

---

## Summary

- **Primary API**: `Execute(prompt string) error`
- **New Error Type**: `*TimeoutError` for timeout failures
- **Configuration**: `Timeout int` field in `ClaudeExecutor`
- **Mechanism**: `context.WithTimeout` + `exec.CommandContext`
- **Exit Code**: 5 for timeout errors
- **Backward Compatibility**: `Timeout = 0` preserves existing behavior
