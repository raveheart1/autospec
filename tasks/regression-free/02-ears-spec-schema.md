# Task: EARS-Formatted Requirements in Spec Schema

> Part of: [Regression-Free Integration](./00-overview.md)
> Reference: `.dev/tasks/SPECKIT_REGRESSION_FREE_INTEGRATION.md` (Layer 2: Specifications)

## Summary

Extend the spec.yaml schema to support EARS (Easy Approach to Requirements Syntax) formatted requirements. This provides machine-parseable requirements that map directly to test types.

## Motivation

Traditional user stories and acceptance criteria are prose—open to interpretation. EARS provides five sentence patterns that force explicit triggers, conditions, and states. An AI agent can unambiguously understand what to implement and how to test it.

## EARS Patterns

| Pattern | Template | Maps To |
|---------|----------|---------|
| **Ubiquitous** | The [system] shall [action] | Invariant/property test |
| **Event-Driven** | When [trigger], the [system] shall [action] | Function test |
| **State-Driven** | While [state], the [system] shall [action] | State machine test |
| **Unwanted Behavior** | If [condition], then the [system] shall [action] | Exception test |
| **Optional** | Where [feature], the [system] shall [action] | Feature flag test |

## Design

### Schema Extension

New optional block in spec.yaml alongside existing `functional` requirements:

```yaml
functional:
  # Existing format continues to work
  - id: "FR-001"
    description: "User can add items to cart"

ears_requirements:
  # New EARS-formatted requirements
  - id: "EARS-001"
    pattern: ubiquitous
    text: "The shopping cart shall maintain a non-negative total."
    test_type: invariant

  - id: "EARS-002"
    pattern: event-driven
    text: "When the user adds an item, the system shall increase the cart count by one."
    trigger: "user adds item"
    expected: "cart count increases by 1"
    test_type: property
```

### Validation Rules

When `verification.level` is `enhanced` or `full`:

1. EARS text must match the pattern template structure
2. Event-driven patterns require `trigger` and `expected` fields
3. State-driven patterns require `state` field
4. Unwanted-behavior patterns require `condition` field
5. IDs must be unique across both `functional` and `ears_requirements`

### Slash Command Updates

The `/autospec.specify` command should:

1. Offer EARS template suggestions when verification level is enhanced+
2. Validate EARS syntax before saving
3. Auto-generate test type hints from pattern

## Implementation Notes

### Schema Package

Extend `internal/validation/` with:

- EARS requirement struct with pattern-specific fields
- Pattern validation (text matches template)
- Cross-reference validation between EARS and functional requirements

### Backwards Compatibility

- `ears_requirements` block is entirely optional
- Specs without it continue to work exactly as before
- Only validated when present AND verification level is enhanced+

## Acceptance Criteria

1. Existing specs without EARS block parse and validate correctly
2. EARS requirements validate pattern-specific fields
3. Malformed EARS text produces helpful error with template example
4. Spec validation reports EARS coverage (how many FRs have corresponding EARS)
5. Documentation updated with EARS examples

## References

- Alistair Mavin et al., "Easy Approach to Requirements Syntax (EARS)"
- Rolls-Royce requirements engineering methodology

## Dependencies

- `01-verification-config.md` (uses verification level to decide validation depth)

## Estimated Scope

Medium. Schema extension, validation logic, and slash command updates.
