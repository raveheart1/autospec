# Data Model: Command Execution Timeout

**Feature**: 003-command-timeout
**Date**: 2025-10-22

## Entities

### 1. Configuration (Enhanced)

**Package**: `internal/config`
**File**: `config.go`

**Description**: Represents the autospec CLI tool configuration, enhanced to include timeout settings.

**Fields**:
```go
type Configuration struct {
    ClaudeCmd       string   `koanf:"claude_cmd" validate:"required"`
    ClaudeArgs      []string `koanf:"claude_args"`
    UseAPIKey       bool     `koanf:"use_api_key"`
    CustomClaudeCmd string   `koanf:"custom_claude_cmd"`
    SpecifyCmd      string   `koanf:"specify_cmd" validate:"required"`
    MaxRetries      int      `koanf:"max_retries" validate:"min=1,max=10"`
    SpecsDir        string   `koanf:"specs_dir" validate:"required"`
    StateDir        string   `koanf:"state_dir" validate:"required"`
    SkipPreflight   bool     `koanf:"skip_preflight"`
    Timeout         int      `koanf:"timeout" validate:"min=1,max=3600"`  // EXISTING FIELD (line 27)
}
```

**Validation Rules**:
- `Timeout`: Optional field
  - If present: Must be between 1 and 3600 seconds (1 second to 1 hour)
  - If missing or 0: No timeout (infinite wait) - backward compatible default
  - Validation enforced via `validator` tag at config load time

**Configuration Sources** (Priority order):
1. Environment variable: `AUTOSPEC_TIMEOUT=300`
2. Local config: `.autospec/config.json`
3. Global config: `~/.autospec/config.json`
4. Default: `0` (no timeout)

**Example Configuration**:
```json
{
  "claude_cmd": "claude",
  "specify_cmd": "specify",
  "max_retries": 3,
  "specs_dir": "./specs",
  "state_dir": "~/.autospec/state",
  "timeout": 300
}
```

**State Transitions**: N/A (configuration is immutable once loaded)

---

### 2. ClaudeExecutor (Enhanced)

**Package**: `internal/workflow`
**File**: `claude.go`

**Description**: Handles Claude CLI command execution with optional timeout enforcement.

**Fields**:
```go
type ClaudeExecutor struct {
    ClaudeCmd       string
    ClaudeArgs      []string
    UseAPIKey       bool
    CustomClaudeCmd string
    Timeout         int      // NEW FIELD: timeout in seconds (0 = no timeout)
}
```

**Methods**:
```go
// Execute runs a Claude command with the given prompt
// If Timeout > 0, the command is terminated after the timeout duration
func (c *ClaudeExecutor) Execute(prompt string) error

// ExecuteSpecKitCommand is a convenience function for SpecKit slash commands
func (c *ClaudeExecutor) ExecuteSpecKitCommand(command string) error

// StreamCommand executes a command and streams output to the provided writer
func (c *ClaudeExecutor) StreamCommand(prompt string, stdout, stderr io.Writer) error
```

**Behavior**:
- When `Timeout == 0`: No timeout, existing behavior (backward compatible)
- When `Timeout > 0`: Command execution wrapped in `context.WithTimeout`
- On timeout: Process killed with `os.Kill`, returns `TimeoutError`

**State Transitions**:
```
[Ready]
  → Execute(prompt) with Timeout > 0
  → [Executing with deadline]
     → Command completes before timeout → [Success]
     → Timeout exceeded → Kill process → [TimeoutError]
```

---

### 3. TimeoutError (New)

**Package**: `internal/workflow`
**File**: `claude.go` (or new `errors.go` if multiple custom errors)

**Description**: Custom error type representing a command timeout failure.

**Structure**:
```go
type TimeoutError struct {
    Timeout time.Duration  // The timeout duration that was exceeded
    Command string         // The command that timed out (e.g., "claude /speckit.plan")
    Err     error          // Underlying error (e.g., context.DeadlineExceeded)
}
```

**Methods**:
```go
// Error returns a human-readable error message
func (e *TimeoutError) Error() string {
    return fmt.Sprintf("command timed out after %v: %s (hint: increase timeout in config)",
        e.Timeout, e.Command)
}

// Unwrap returns the underlying error for errors.Is/As compatibility
func (e *TimeoutError) Unwrap() error {
    return e.Err
}
```

**Usage**:
```go
if err := executor.Execute(prompt); err != nil {
    var timeoutErr *TimeoutError
    if errors.As(err, &timeoutErr) {
        fmt.Fprintf(os.Stderr, "Error: %v\n", timeoutErr)
        os.Exit(5)  // Timeout-specific exit code
    }
}
```

**Validation Rules**: N/A (error type, not validated data)

---

### 4. Command Execution Context (Implicit)

**Package**: Standard library `context`
**Type**: `context.Context`

**Description**: Go's standard context used to enforce timeout deadlines on command execution.

**Creation**:
```go
// No timeout
ctx := context.Background()

// With timeout
ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
defer cancel()
```

**Properties**:
- **Deadline**: Absolute time when the context expires
- **Done channel**: Closed when deadline is exceeded or context is cancelled
- **Err**: Returns `context.DeadlineExceeded` when timeout occurs

**Lifecycle**:
```
[Created with timeout]
  → Time progresses
  → Deadline not reached → Command completes → [Context cancelled by defer]
  → Deadline exceeded → Done channel closed → [Process killed]
```

**Integration with exec**:
```go
cmd := exec.CommandContext(ctx, c.ClaudeCmd, args...)
// When ctx deadline is exceeded:
// - cmd.Process is killed with os.Kill
// - cmd.Wait() returns error
// - ctx.Err() == context.DeadlineExceeded
```

---

## Entity Relationships

```
Configuration
  │
  ├─(loaded by)──> ClaudeExecutor
  │                   │
  │                   ├─(creates if Timeout > 0)──> Context with deadline
  │                   │                               │
  │                   │                               ├─(passed to)──> exec.CommandContext
  │                   │                               │
  │                   │                               └─(on deadline exceeded)──> Process killed
  │                   │
  │                   └─(on timeout)──> TimeoutError
  │
  └─(timeout value: 0 or missing)──> No timeout enforcement (backward compatible)
```

---

## Data Flow

### Scenario 1: Command Execution with Timeout (Success)

```
1. Load Configuration from file/env
   → Timeout = 300 (5 minutes)

2. Create ClaudeExecutor
   → executor.Timeout = 300

3. Execute command
   → Create context.WithTimeout(5 minutes)
   → exec.CommandContext(ctx, "claude", "/speckit.plan")
   → Command completes in 2 minutes
   → defer cancel() cleans up context
   → Return nil (success)
```

### Scenario 2: Command Execution with Timeout (Timeout)

```
1. Load Configuration
   → Timeout = 30 (30 seconds)

2. Create ClaudeExecutor
   → executor.Timeout = 30

3. Execute command
   → Create context.WithTimeout(30 seconds)
   → exec.CommandContext(ctx, "claude", "/speckit.implement")
   → Command runs for 35 seconds
   → Context deadline exceeded
   → Process killed with os.Kill
   → Check ctx.Err() == context.DeadlineExceeded
   → Return TimeoutError{Timeout: 30s, Command: "claude /speckit.implement"}

4. Handle error in CLI
   → Detect TimeoutError with errors.As
   → Print error message with hint
   → Exit with code 5
```

### Scenario 3: No Timeout Configured (Backward Compatible)

```
1. Load Configuration
   → Timeout field missing or = 0

2. Create ClaudeExecutor
   → executor.Timeout = 0

3. Execute command
   → Check: timeout == 0?
   → Use context.Background() (no deadline)
   → exec.CommandContext(ctx, "claude", "/speckit.specify")
   → Command runs indefinitely until completion
   → Return nil or standard error (existing behavior)
```

---

## Validation Summary

| Entity | Field | Validation Rule | Enforcement |
|--------|-------|----------------|-------------|
| Configuration | Timeout | `min=1,max=3600` | Config load time via validator |
| ClaudeExecutor | Timeout | 0 = no timeout, >0 = timeout in seconds | Runtime via context.WithTimeout |
| TimeoutError | - | N/A (error type) | N/A |
| Context | Deadline | Enforced by Go runtime | Automatic via exec.CommandContext |

---

## Performance Considerations

1. **Context Overhead**: ~50 nanoseconds per context creation (negligible)
2. **Timer Cleanup**: Handled by `defer cancel()` if command completes before timeout
3. **Benchmark Requirement**: Verify <1% overhead with timeout enabled (SC-004)

---

## Backward Compatibility

- **Missing timeout config**: Defaults to 0 (no timeout)
- **Timeout = 0**: No timeout enforcement (existing behavior)
- **Existing commands**: Unaffected unless user explicitly sets timeout
- **Configuration migration**: No migration needed (field is optional)
