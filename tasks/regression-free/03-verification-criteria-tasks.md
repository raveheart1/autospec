# Task: Verification Criteria in Tasks Schema

> Part of: [Regression-Free Integration](./00-overview.md)
> Reference: `.dev/tasks/SPECKIT_REGRESSION_FREE_INTEGRATION.md` (Layer 4: Tasks)

## Summary

Extend tasks.yaml schema to include machine-verifiable completion criteria for each task. Instead of prose acceptance criteria that require human judgment, tasks declare specific checks that must pass.

## Motivation

Current task acceptance criteria are interpreted by the AI agent and validated by human review. This creates a bottleneck. Machine-verifiable criteria allow automated validation: did the types pass? Did the tests pass? Is complexity within budget?

## Design

### Configuration Toggle

Add `verification_criteria` toggle to the verification config block in `01-verification-config.md`:

```yaml
verification:
  level: enhanced
  verification_criteria: true  # Enable task verification criteria (default: follows level)
```

This toggle controls whether:
1. Tasks can include `verification` blocks
2. Verification blocks are validated during task parsing
3. Task generation includes verification criteria at enhanced+ level

### Schema Extension

New optional `verification` block per task:

```yaml
tasks:
  - id: "TASK-001"
    title: "Implement CartEntity"
    description: "Create the Cart entity with proper invariants"

    # Existing acceptance criteria (prose)
    acceptance_criteria:
      - "Cart stores items correctly"
      - "Total calculation is accurate"

    # NEW: Machine-verifiable criteria
    verification:
      files:
        - "internal/domain/cart.go"
        - "internal/domain/cart_test.go"

      checks:
        types:
          command: "go build ./..."
          must_pass: true

        tests:
          command: "go test ./internal/domain/..."
          must_pass: true
          coverage_min: 80  # optional threshold

        lint:
          command: "golangci-lint run ./internal/domain/..."
          must_pass: true

        complexity:
          max_cyclomatic: 10
          max_lines_per_function: 40

      # Links to EARS requirements this task addresses
      addresses_requirements:
        - "EARS-001"
        - "EARS-002"
```

### Verification Check Types

| Check | Description | Failure Meaning |
|-------|-------------|-----------------|
| `types` | Type/compile check | Code doesn't compile |
| `tests` | Test suite | Behavior doesn't match spec |
| `lint` | Static analysis | Code quality issues |
| `complexity` | Cyclomatic/line limits | Code too complex |
| `coverage` | Test coverage | Insufficient test depth |

### Validation Rules

When `verification.level` is `enhanced` or `full`:

1. Tasks with `verification` block have all checks validated
2. `must_pass: true` checks are required to pass
3. Threshold checks (coverage, complexity) compared against values
4. `addresses_requirements` must reference valid EARS IDs from spec

### Execution Model

The verification block is **declarative**, not automatically executed. It declares what should be checked. The `autospec verify` command (separate task) runs these checks.

## Implementation Notes

### Schema Package

Extend task schema in `internal/validation/`:

- VerificationBlock struct with check definitions
- Check struct with command, must_pass, thresholds
- Cross-reference validation for addresses_requirements

### Task Generation

Update `/autospec.tasks` command to:

1. Generate verification blocks when level is enhanced+
2. Infer checks from file types (Go → go build, go test, golangci-lint)
3. Link tasks to EARS requirements from spec

### Backwards Compatibility

- `verification` block is entirely optional
- Tasks without it work exactly as before
- Only validated when present AND verification level is enhanced+

## Acceptance Criteria

1. `verification_criteria` toggle added to verification config schema
2. Toggle respects level presets (enabled at enhanced+) with explicit override support
3. Existing tasks.yaml without verification blocks parse correctly
4. Verification blocks validate check structure
5. Invalid requirement references produce helpful errors
6. Task generation includes verification blocks when toggle is enabled
7. Verification block is pure data—no execution at parse time

## Dependencies

- `01-verification-config.md` (uses verification level)
- `02-ears-spec-schema.md` (for addresses_requirements cross-references)

## Estimated Scope

Medium. Schema extension, validation logic, and task generation updates.
