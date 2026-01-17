# Task: Session Tail Injection for Retry Context

> Part of: [Regression-Free Integration](./00-overview.md)
> Reference: `.dev/tasks/SPECKIT_REGRESSION_FREE_INTEGRATION.md` (AI Feedback Format)

## Summary

Capture the last N bytes of agent session output and inject it into retry context. This provides the agent with crucial information about what happened in the previous failed attempt—error messages, warnings, and final state—without the token cost of full conversation history.

## Motivation

The current retry mechanism only injects **schema validation errors** (missing fields, wrong types, invalid enums). It does NOT capture:

1. **Agent's reasoning** about what it did
2. **Error messages** the agent encountered during execution (build failures, test failures, lint errors)
3. **Agent's final conclusions** before the session ended

When a phase/task fails validation, the retry attempt has no context about what went wrong in the *execution* itself—only schema-level issues. The agent's **last output** typically contains the most valuable debugging information.

### Why "Last Message" Instead of Full History?

| Approach | Token Cost | Signal Quality | 
|----------|------------|----------------|
| Full conversation history | HIGH (100k+ tokens) | Diluted with noise |
| Last ~4KB of output | LOW (~500-2000 tokens) | HIGH - conclusions, errors, state |
| Validation errors only (current) | VERY LOW (~200 tokens) | INCOMPLETE - no execution context |

The last portion of output is information-dense: it contains what the agent concluded, what errors it saw, and what state things were left in.

## Design

### Agent-Agnostic Capture

Both Claude Code and OpenCode produce human-readable output to stdout:

- **Claude Code**: Outputs stream-json, which is transformed to plain text by the existing `FormatterWriter`
- **OpenCode**: Outputs plain text directly

The tail capture occurs **after** any formatting transformation, making it agent-agnostic. The same mechanism works for both agents without special handling.

### Capture Strategy

A ring buffer captures the last N bytes of formatted session output. On session failure:

1. The buffer contents are preserved
2. ANSI escape codes (colors, cursor control) are stripped
3. The cleaned text is stored for injection into the next retry attempt

### Retry Context Enhancement

The existing retry context format is extended to include the session tail:

```
RETRY 2/3

Previous session output (tail):
───────────────────────────────
[last ~4KB of agent output, showing errors, warnings, conclusions]
───────────────────────────────

Schema validation failed:
- missing required field: feature.name
- invalid enum value for status

## Retry Instructions
[existing retry guidance]
```

This gives the agent both:
- **What went wrong structurally** (validation errors)
- **What went wrong during execution** (session tail)

### Configuration

New configuration options under the `retry` section:

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `include_session_tail` | bool | `false` | Enable session tail capture for retry context |
| `session_tail_size` | int | `4096` | Bytes of output to capture (tail) |

Example:
```yaml
retry:
  max_retries: 3
  include_session_tail: true
  session_tail_size: 4096
```

### Example Captured Output

**Claude Code** (typical session end):
```
I've created the spec.yaml file with the feature specification.

However, I noticed some issues:
- The API endpoint /users/{id} returns 404 for the test user  
- Build failed with: undefined symbol 'validateAuth'

The file was saved but may need manual fixes for the API integration.
```

**OpenCode** (typical session end):
```
✓ Created specs/001-user-auth/spec.yaml
✓ Updated .autospec/config.yml

⚠ Warning: Could not verify API connectivity
  Error: Connection refused to localhost:3000

Task completed with warnings. Review the generated spec for API-related fields.
```

## Integration Points

### ClaudeExecutor

The executor wraps stdout with the tail buffer during non-interactive execution. On failure, the captured tail is made available for retry context building.

### Executor Retry Loop

The existing `FormatRetryContext()` function is extended to accept and format the session tail alongside validation errors.

### Config Loading

New fields added to the retry configuration struct with appropriate defaults and schema validation.

## Acceptance Criteria

1. Session tail is captured during non-interactive agent execution
2. Capture works identically for Claude Code and OpenCode agents
3. ANSI escape codes are stripped from captured output
4. Retry context includes session tail when available and configured
5. Feature is opt-in (`include_session_tail: false` by default)
6. Configurable capture size via `session_tail_size`
7. No performance impact when disabled
8. Empty output results in tail section being omitted from retry context

## Dependencies

None. This task is independent of:
- `02-ears-spec-schema.md` (EARS requirements are optional)
- `03-verification-criteria-tasks.md` (SKIPPED)
- `04-verify-command.md` (SKIPPED)

## Estimated Scope

Small. Ring buffer utility, executor integration, config fields, retry context formatting.

## Historical Note

This task was originally titled "Structured Feedback for AI Retry Loops" with broader scope:
- Per-task verification command output capture
- EARS requirement linking in error context
- Complex JSON feedback format with suggestion generation

These were dropped after 03-* (per-task verification) and 04-* (verify command) were skipped. The revised scope focuses on the highest-value, lowest-complexity improvement: capturing the agent's final output to provide execution context alongside existing schema validation errors.
