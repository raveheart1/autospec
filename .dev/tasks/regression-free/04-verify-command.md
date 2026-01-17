# Task: Autospec Verify Command

> Part of: [Regression-Free Integration](./00-overview.md)
> Reference: `.dev/tasks/SPECKIT_REGRESSION_FREE_INTEGRATION.md` (Layer 5: Implementation)

## Status: SKIPPED (DO NOT IMPLEMENT)

### Decision Date: 2026-01-05

### Reason

This command's purpose was to execute per-task verification blocks from `03-verification-criteria-tasks.md`. Since that feature was skipped (constitution already defines quality gates), this command has no data to operate on.

**Quality gates are already executable via:**
- `make fmt && make lint && make test && make build`
- Constitution principles PRIN-007 and PRIN-011

A dedicated `autospec verify` command would just be a wrapper around these existing commands with no additional value.

### If Needed Later

If a verify command becomes useful, it should:
- Read `quality_gates` from constitution.yaml (not per-task verification blocks)
- Provide unified reporting across all gates
- But this is essentially what `make lint && make test && make build` already does

---

## Original Design (Preserved for Reference)

## Summary

Implement `autospec verify [spec-name]` command that runs verification checks defined in tasks.yaml and reports results. This is the runtime component that executes the declarative verification criteria.

## Motivation

Verification criteria in tasks.yaml are just data until something executes them. The verify command bridges declaration and execution, providing a single entry point to validate implementation against spec.

## Design

### Configuration Toggle

Add `verify_command` toggle to the verification config block in `01-verification-config.md`:

```yaml
verification:
  level: enhanced
  verify_command: true  # Enable autospec verify command (default: follows level)
```

This toggle controls whether:
1. The `autospec verify` command is available
2. Verification checks are executed when invoked
3. Integration with `autospec implement --verify` is enabled

### Command Interface

```bash
# Verify all tasks in a spec
autospec verify cart-feature

# Verify a specific task
autospec verify cart-feature --task TASK-001

# Verify with verbose output
autospec verify cart-feature --verbose

# Output as JSON (for CI/tooling)
autospec verify cart-feature --format json

# Dry-run (show what would be checked)
autospec verify cart-feature --dry-run
```

### Execution Flow

1. Load spec and tasks.yaml for given spec-name
2. For each task (or specified task):
   - Extract verification block
   - Execute each check command
   - Collect results (pass/fail, output, timing)
3. Aggregate results across all tasks
4. Report summary with pass/fail status

### Output Format

Human-readable (default):

```
Verifying: cart-feature

TASK-001: Implement CartEntity
  ✓ types (go build)          0.8s
  ✓ tests (go test)           2.1s
  ✓ lint (golangci-lint)      1.2s
  ✓ complexity                 -

TASK-002: Implement AddToCart
  ✓ types (go build)          0.3s
  ✗ tests (go test)           1.8s
    FAIL: TestAddToCart_EmptyItem
  ✓ lint (golangci-lint)      0.9s

Summary: 1 of 2 tasks passed
```

JSON format (for tooling):

```json
{
  "spec": "cart-feature",
  "passed": false,
  "tasks": [
    {
      "id": "TASK-001",
      "passed": true,
      "checks": [...]
    }
  ]
}
```

### Integration with Implement

The verify command is standalone but designed to integrate with implement:

- `autospec implement --verify` could run verify after each task
- Structured output enables AI retry loops with specific feedback
- Exit codes indicate pass (0), fail (1), or error (2)

## Implementation Notes

### CLI Package

New command in `internal/cli/stages/verify.go`:

- Cobra command with spec-name argument
- Flags: --task, --verbose, --format, --dry-run
- Uses existing spec detection logic

### Verification Runner

New package `internal/verification/`:

- Runner that executes check commands
- Result aggregation and reporting
- Timeout handling per check
- Parallel execution option for independent checks

### Check Execution

Each check type has execution logic:

- `types`: Run command, check exit code
- `tests`: Run command, parse output for failures
- `lint`: Run command, check exit code
- `complexity`: Analyze files, compare against thresholds

## Acceptance Criteria

1. `verify_command` toggle added to verification config schema
2. Toggle respects level presets (enabled at enhanced+) with explicit override support
3. `autospec verify spec-name` runs all verification checks
4. `--task` flag limits to specific task
5. `--format json` produces machine-parseable output
6. Exit code 0 only when all checks pass
7. Verbose mode shows command output
8. Timeout handling prevents hung commands
9. Works with specs that have no verification blocks (no-op with warning)

## Dependencies

- `01-verification-config.md` (respects verification level)
- ~~`03-verification-criteria-tasks.md`~~ **SKIPPED** - verification blocks not implemented; verify command uses constitution quality gates instead

## Estimated Scope

Medium-Large. New command, execution runner, output formatting.
