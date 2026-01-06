# Task: Session Tail Injection for Retry Context

> Part of: [Regression-Free Integration](./00-overview.md)
> Reference: `.dev/tasks/SPECKIT_REGRESSION_FREE_INTEGRATION.md` (AI Feedback Format)

## Summary

Capture the last N bytes of agent session output and inject it into retry context. This provides Claude/OpenCode with crucial information about what happened in the previous failed attempt—error messages, warnings, and the agent's final state—without the token cost of full conversation history.

## Motivation

Current retry mechanism only injects **schema validation errors** (missing fields, wrong types). It does NOT capture:

1. **Agent's reasoning** about what it did
2. **Error messages** the agent saw during execution (build failures, test failures, lint errors)
3. **Agent's final conclusions** from the previous session

When a phase/task fails validation, the agent gets no context about what went wrong in the *execution* itself—only schema-level issues. The agent's **last message** typically contains the most valuable debugging information: summaries, error messages, blockers, and state.

### Token Cost Comparison

| Approach | Token Cost | Signal Quality | Complexity |
|----------|------------|----------------|------------|
| Full conversation history | HIGH (100k+) | Diluted (noise) | Medium |
| Last message only | LOW (~500-2000) | HIGH (conclusions) | Low |
| Validation errors only (current) | VERY LOW (~50-200) | INCOMPLETE | Done |

## Design

### Agent-Agnostic Architecture

Both agents output plain text to the terminal after processing:

```
┌─────────────────────────────────────────────────────────┐
│                   Agent.Execute()                        │
│                (Claude or OpenCode)                      │
└─────────────────────────────────────────────────────────┘
                          │
           ┌──────────────┴──────────────┐
           ▼                              ▼
  ┌─────────────────┐            ┌─────────────────┐
  │   Claude Code   │            │    OpenCode     │
  │   stream-json   │            │   plain text    │
  └─────────────────┘            └─────────────────┘
           │                              │
           ▼                              │
  ┌─────────────────┐                     │
  │ FormatterWriter │                     │
  │  (JSON→text)    │                     │
  └─────────────────┘                     │
           │                              │
           └──────────────┬───────────────┘
                          ▼
  ┌─────────────────────────────────────────────────────────┐
  │                TailBuffer (NEW)                         │
  │         Captures last N bytes of output                 │
  └─────────────────────────────────────────────────────────┘
                          │
                          ▼
  ┌─────────────────────────────────────────────────────────┐
  │                     os.Stdout                           │
  └─────────────────────────────────────────────────────────┘
```

The capture happens **after** formatting, so it captures human-readable output regardless of agent.

### Configuration

```yaml
# .autospec/config.yml
retry:
  include_session_tail: true  # Capture last output for retry context (default: false)
  session_tail_size: 4096     # Bytes to capture (default: 4096)
```

### TailBuffer Implementation

A simple ring buffer that keeps the last N bytes:

```go
// internal/workflow/tailbuffer.go

type TailBuffer struct {
    buf    []byte
    maxLen int
}

func NewTailBuffer(maxLen int) *TailBuffer {
    return &TailBuffer{maxLen: maxLen}
}

func (t *TailBuffer) Write(p []byte) (n int, err error) {
    t.buf = append(t.buf, p...)
    if len(t.buf) > t.maxLen {
        t.buf = t.buf[len(t.buf)-t.maxLen:]
    }
    return len(p), nil
}

func (t *TailBuffer) String() string {
    return stripANSI(string(t.buf))
}
```

### Integration Point

In `ClaudeExecutor.executeWithAgent()`:

```go
// Layer 1: Start with os.Stdout
var stdout io.Writer = os.Stdout

// Layer 2: Wrap with formatter (Claude stream-json → plain text)
if !interactive {
    stdout = c.getFormattedStdout(stdout)
}

// Layer 3: Wrap with tail capture (agent-agnostic)
var tailBuf *TailBuffer
if c.CaptureSessionTail && !interactive {
    tailBuf = NewTailBuffer(c.SessionTailSize)
    stdout = io.MultiWriter(stdout, tailBuf)
}

// Execute...

// On failure, capture tail for retry injection
if tailBuf != nil && err != nil {
    c.lastSessionTail = tailBuf.String()
}
```

### Retry Context Format

Extended `FormatRetryContext()` output:

```
RETRY 2/3

Previous session output (tail):
───────────────────────────────
... I encountered an error building: undefined reference to foo
... The spec.yaml was written but may be incomplete
───────────────────────────────

Schema validation failed:
- missing required field: feature.name
- invalid enum value for status

[Retry Instructions]
```

### What Gets Captured

Both agents typically end sessions with valuable context:

**Claude Code** (after FormatterWriter):
```
I've created the spec.yaml file with the feature specification.

However, I noticed some issues:
- The API endpoint /users/{id} returns 404 for the test user
- Build failed with: undefined symbol 'validateAuth'

The file was saved but may need manual fixes.
```

**OpenCode**:
```
✓ Created specs/001-user-auth/spec.yaml

⚠ Warning: Could not verify API connectivity
  Error: Connection refused to localhost:3000

Task completed with warnings.
```

## Implementation Notes

### Files to Modify

| File | Changes |
|------|---------|
| `internal/workflow/tailbuffer.go` | NEW: TailBuffer type (~30 LOC) |
| `internal/workflow/claude.go` | Add tail capture to executeWithAgent (~15 LOC) |
| `internal/workflow/executor.go` | Extend FormatRetryContext with tail (~20 LOC) |
| `internal/config/config.go` | Add retry config fields (~5 LOC) |
| `internal/config/schema.go` | Add schema entries (~5 LOC) |
| `internal/config/defaults.go` | Add defaults (~3 LOC) |

**Total**: ~80-100 LOC

### ANSI Stripping

The captured output may contain ANSI escape codes (colors, cursor movement). Strip these before injection:

```go
var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

func stripANSI(s string) string {
    return ansiRegex.ReplaceAllString(s, "")
}
```

### Edge Cases

| Case | Handling |
|------|----------|
| Output < tail size | Return entire output |
| Tail starts mid-line | Acceptable (context still useful) |
| Empty output | Skip tail section in retry context |
| Interactive mode | Skip capture (no retry loop) |

## Acceptance Criteria

1. `TailBuffer` correctly captures last N bytes with O(1) amortized writes
2. Tail capture integrates with both Claude and OpenCode agents
3. ANSI escape codes are stripped from captured output
4. Retry context includes tail section when available
5. Config options `retry.include_session_tail` and `retry.session_tail_size` work correctly
6. Opt-in by default (`include_session_tail: false`)
7. No performance impact when disabled

## Dependencies

None. This task is independent of:
- `02-ears-spec-schema.md` (EARS is optional)
- `03-verification-criteria-tasks.md` (SKIPPED)
- `04-verify-command.md` (SKIPPED)

## Estimated Scope

Small. ~80-100 LOC across 6 files. No new packages or complex parsing.

## Historical Note

This task was originally titled "Structured Feedback for AI Retry Loops" and proposed:
- Per-task verification command output
- EARS requirement linking
- Complex JSON feedback format
- Suggestion generation for failure patterns

These were dropped after 03-* and 04-* were skipped (no per-task verification commands exist). The current scope focuses on the highest-value, lowest-complexity approach: capturing the agent's final output to provide execution context alongside schema validation errors.
