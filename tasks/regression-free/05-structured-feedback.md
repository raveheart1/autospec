# Task: Structured Feedback for AI Retry Loops

> Part of: [Regression-Free Integration](./00-overview.md)
> Reference: `.dev/tasks/SPECKIT_REGRESSION_FREE_INTEGRATION.md` (AI Feedback Format)

## Summary

When verification fails, provide structured, actionable feedback that AI agents can parse and act upon. This enables automated retry loops where the agent understands exactly what failed and why.

## Motivation

Current failure output is human-oriented: stack traces, prose error messages, test output. An AI agent benefits from structured data: which check failed, which requirement it maps to, what the minimal failing case is, and suggestions for fixes.

## Design

### Structured Error Format

When a verification check fails, produce structured output alongside human-readable:

```json
{
  "verification_failed": true,
  "task_id": "TASK-001",
  "check": "tests",
  "command": "go test ./internal/domain/...",
  "exit_code": 1,

  "failure": {
    "type": "test_failure",
    "test_name": "TestAddToCart_EmptyItem",
    "file": "internal/domain/cart_test.go",
    "line": 42,
    "message": "expected cart.Count() == 1, got 0",
    "output_snippet": "--- FAIL: TestAddToCart_EmptyItem (0.00s)\n    cart_test.go:42: ..."
  },

  "context": {
    "addresses_requirements": ["EARS-002"],
    "ears_text": "When the user adds an item, the system shall increase the cart count by one.",
    "relevant_code": ["internal/domain/cart.go:Add"]
  },

  "suggestion": "The Add method may not be appending to the items slice. Check that items = append(items, item) is called.",

  "retry_info": {
    "attempt": 2,
    "max_retries": 5,
    "previous_failures": ["same test, different assertion"]
  }
}
```

### Feedback Components

| Component | Purpose |
|-----------|---------|
| `failure` | What exactly failed (test, lint rule, complexity limit) |
| `context` | Links to EARS requirements and relevant code |
| `suggestion` | AI-generated hint based on failure pattern |
| `retry_info` | Track retry attempts to detect loops |

### Integration Points

#### Verify Command

`autospec verify --format feedback` produces this format:

- One JSON object per failed check
- Includes full context for AI consumption
- Human summary still shown in stderr

#### Implement Command

When `--verify` flag is used with implement:

- Capture structured feedback on failure
- Pass to AI agent as context for retry
- Track retry count against max_retries config

### Suggestion Generation

Simple pattern matching for common failures:

| Pattern | Suggestion |
|---------|------------|
| Test assertion failed | "Check the implementation of [function] matches the expected behavior" |
| Nil pointer | "Ensure [variable] is initialized before use" |
| Complexity exceeded | "Extract helper functions to reduce cyclomatic complexity" |
| Coverage below threshold | "Add tests for [uncovered functions]" |

## Implementation Notes

### Feedback Package

New package `internal/feedback/`:

- StructuredError struct matching JSON format
- Pattern matchers for common failure types
- Suggestion generator based on failure patterns
- EARS requirement linker (looks up from spec)

### Output Modes

Extend verify command output handling:

- `--format human` (default): Current readable output
- `--format json`: Raw results as JSON
- `--format feedback`: AI-optimized structured feedback

### Retry State

Extend `internal/retry/` to track:

- Per-task failure history
- Previous failure types
- Retry count per check

## Acceptance Criteria

1. `autospec verify --format feedback` produces structured JSON on failure
2. Failures include links to EARS requirements when available
3. Suggestions are generated for common failure patterns
4. Retry info tracks attempt count accurately
5. Human-readable output still available in parallel
6. Feedback is actionable—AI can parse and respond

## Dependencies

- `02-ears-spec-schema.md` (for EARS requirement linking)
- `04-verify-command.md` (extends verify output)

## Estimated Scope

Medium. Feedback structure, pattern matching, output formatting.
