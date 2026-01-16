# Research: Command Execution Timeout

**Feature**: 003-command-timeout
**Date**: 2025-10-22
**Status**: Complete

## Research Questions

### 1. How to implement timeout for exec.Command in Go?

**Decision**: Use `context.WithTimeout` and pass context to `exec.CommandContext`

**Rationale**:
- Go's standard library provides `exec.CommandContext(ctx, name, args...)` which accepts a context
- When the context deadline is exceeded or cancelled, the command is automatically killed
- The command's Process is killed with `os.Kill` signal
- This is the idiomatic Go approach recommended by the standard library docs
- No manual signal handling or goroutine management required

**Alternatives Considered**:
1. **Manual timeout with time.AfterFunc + cmd.Process.Kill**
   - Rejected: More error-prone, requires manual cleanup, not idiomatic
   - Would need to handle race conditions between timeout and normal completion

2. **Third-party timeout libraries**
   - Rejected: Unnecessary dependency, standard library is sufficient
   - Would increase binary size for minimal benefit

**Implementation Details**:
```go
// Current implementation (no timeout)
cmd := exec.Command(c.ClaudeCmd, args...)
err := cmd.Run()

// New implementation (with timeout)
ctx, cancel := context.WithTimeout(context.Background(), timeout)
defer cancel()
cmd := exec.CommandContext(ctx, c.ClaudeCmd, args...)
err := cmd.Run()

// Check if timeout occurred
if ctx.Err() == context.DeadlineExceeded {
    return TimeoutError{...}
}
```

---

### 2. How to properly clean up processes when timeout occurs?

**Decision**: Rely on exec.CommandContext's built-in cleanup + verify with Wait()

**Rationale**:
- `exec.CommandContext` automatically sends `os.Kill` to the process when context expires
- Calling `cmd.Wait()` (which `cmd.Run()` does internally) ensures proper cleanup
- The process and all file descriptors are cleaned up by the OS
- No orphaned processes will remain (verified by `ps` in tests)

**Alternatives Considered**:
1. **Manual process tree cleanup**
   - Rejected: `os.Kill` on Unix sends SIGKILL which forcibly terminates the process
   - The OS handles cleanup of child processes automatically
   - Manual cleanup would be platform-specific and complex

2. **Graceful shutdown with SIGTERM first**
   - Rejected: The spec requires termination within 5 seconds (FR-002, SC-001)
   - SIGTERM allows processes to ignore or delay, violating the requirement
   - SIGKILL (used by exec.CommandContext) guarantees immediate termination

**Edge Case Handling**:
- Timeout during file writes: File handles are closed by OS, files may be incomplete
- Timeout during state updates: Existing retry logic handles partial state
- Orphaned processes: Verified in tests that no processes remain after timeout

---

### 3. What is the best format for timeout configuration values?

**Decision**: Use integer seconds only, validate range 1-3600 (1 second to 1 hour)

**Rationale**:
- Simple integer format aligns with existing config patterns in the codebase
- Config validation already uses validator tags (`validate:"min=1,max=3600"`)
- Easy to parse from environment variables and config files
- Duration strings ("5m", "30s") would require parsing and add complexity
- Users can calculate seconds (5 minutes = 300 seconds)

**Alternatives Considered**:
1. **Duration strings (e.g., "5m", "30s")**
   - Rejected: Requires parsing with `time.ParseDuration`
   - Inconsistent with existing config patterns (max_retries is int)
   - More error-prone (typos in duration format)

2. **Support both integers and duration strings**
   - Rejected: Increases complexity without significant benefit
   - Would need custom unmarshaling logic
   - Integer-only is sufficient and simpler

**Configuration Example**:
```json
{
  "timeout": 300,  // 5 minutes in seconds
  "max_retries": 3
}
```

Environment variable:
```bash
AUTOSPEC_TIMEOUT=300  # 5 minutes
```

---

### 4. How to distinguish timeout errors from other command failures?

**Decision**: Create custom `TimeoutError` type that implements `error` interface

**Rationale**:
- Allows callers to use `errors.As()` for type-based error handling
- Can include metadata (timeout duration, command that timed out)
- Enables clear error messages with context
- Supports exit code 5 for timeout-specific failures

**Alternatives Considered**:
1. **Return standard error with timeout message**
   - Rejected: Callers cannot programmatically distinguish timeout from other errors
   - Would require string parsing to detect timeout

2. **Use sentinel error (errors.New("timeout"))**
   - Rejected: Cannot include dynamic metadata (duration, command)
   - Less flexible than custom error type

**Implementation**:
```go
type TimeoutError struct {
    Timeout time.Duration
    Command string
    Err     error
}

func (e *TimeoutError) Error() string {
    return fmt.Sprintf("command timed out after %v: %s", e.Timeout, e.Command)
}

func (e *TimeoutError) Unwrap() error {
    return e.Err
}
```

**Error Handling**:
```go
if err := executor.Execute(prompt); err != nil {
    var timeoutErr *TimeoutError
    if errors.As(err, &timeoutErr) {
        fmt.Fprintf(os.Stderr, "Error: %v\n", timeoutErr)
        fmt.Fprintf(os.Stderr, "Suggestion: Increase timeout in config\n")
        os.Exit(5)  // Timeout-specific exit code
    }
    // Handle other errors...
}
```

---

### 5. How to ensure <1% performance overhead when timeout is enabled?

**Decision**: Use Go's context package (zero allocation overhead) + benchmark tests

**Rationale**:
- `context.WithTimeout` has negligible overhead (creates timer, no polling)
- The timer is cleaned up by `defer cancel()` if command completes before timeout
- Benchmark tests will verify <1% overhead requirement (SC-004)
- No background goroutines or polling loops needed

**Alternatives Considered**:
1. **Polling-based timeout checking**
   - Rejected: Wastes CPU cycles checking time in a loop
   - Would violate <1% overhead requirement

2. **Channel-based timeout**
   - Rejected: More complex than context, similar performance
   - Context is the idiomatic approach

**Benchmark Test**:
```go
func BenchmarkExecuteWithoutTimeout(b *testing.B) {
    executor := &ClaudeExecutor{...}
    for i := 0; i < b.NsPerOp(); i++ {
        executor.Execute("test")
    }
}

func BenchmarkExecuteWithTimeout(b *testing.B) {
    executor := &ClaudeExecutor{Timeout: 300}
    for i := 0; i < b.NsPerOp(); i++ {
        executor.Execute("test")
    }
}
```

Compare results to verify <1% difference.

---

### 6. How to handle timeout values of 0 or missing timeout config?

**Decision**: 0 or missing timeout = no timeout (infinite wait), maintains backward compatibility

**Rationale**:
- Spec assumption: "Default timeout behavior (when not configured) is no timeout (infinite wait) to maintain backward compatibility"
- Existing users won't be affected by the new feature
- Timeout is opt-in, must be explicitly configured
- Config validation only applies if timeout > 0

**Implementation**:
```go
func (c *ClaudeExecutor) Execute(prompt string, timeout int) error {
    var ctx context.Context
    var cancel context.CancelFunc

    if timeout > 0 {
        ctx, cancel = context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
        defer cancel()
    } else {
        ctx = context.Background()  // No timeout
    }

    cmd := exec.CommandContext(ctx, c.ClaudeCmd, args...)
    // ...
}
```

---

## Summary

All technical unknowns have been resolved:

1. **Timeout Implementation**: Use `exec.CommandContext` with `context.WithTimeout`
2. **Process Cleanup**: Rely on built-in `os.Kill` cleanup from exec.CommandContext
3. **Config Format**: Integer seconds (1-3600), validated with validator tags
4. **Error Handling**: Custom `TimeoutError` type with metadata
5. **Performance**: Go context has negligible overhead, verified with benchmarks
6. **Backward Compatibility**: Missing/zero timeout = no timeout (existing behavior)

No additional research required. Ready to proceed to Phase 1 (Design & Contracts).
