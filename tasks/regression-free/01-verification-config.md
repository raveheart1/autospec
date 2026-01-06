# Task: Verification Level Configuration

> Part of: [Regression-Free Integration](./00-overview.md)
> Reference: `.dev/tasks/SPECKIT_REGRESSION_FREE_INTEGRATION.md` (Layer 1: Constitution)

## Summary

Add a `verification` configuration block that controls the depth of automated validation. This is the foundation for all regression-free features—users opt into enhanced verification through configuration.

## Motivation

Different projects have different needs. A quick prototype doesn't need mutation testing, but a production payment system does. Configuration tiers let users choose their tradeoff between speed and rigor.

## Design

### Configuration Schema

Add to both user config (`~/.config/autospec/config.yml`) and project config (`.autospec/config.yml`):

```yaml
verification:
  level: basic  # basic | enhanced | full

  # Optional overrides (only when level != basic)
  mutation_threshold: 0.8
  coverage_threshold: 0.85
  complexity_max: 10
```

### Verification Levels

| Level | What It Enables |
|-------|-----------------|
| `basic` | Current behavior. No additional verification. |
| `enhanced` | EARS validation, verification criteria checking, structured feedback |
| `full` | All enhanced features plus mutation testing, property tests, architecture checks |

### Behavior

- Default is `basic` for backwards compatibility
- Level cascades: `full` implies all `enhanced` features
- Individual thresholds can override level defaults
- Config validation ensures thresholds are within valid ranges

## Implementation Notes

### Config Package Changes

Extend `internal/config/config.go` with:

- New `Verification` struct with level and threshold fields
- Validation that level is one of the allowed values
- Default values that match current behavior

### Integration Points

- Spec validation uses level to decide which checks to run
- Task completion uses level to determine verification depth
- Verify command respects level for its check suite

## Acceptance Criteria

1. New config fields are recognized and parsed correctly
2. Invalid levels produce clear error messages
3. Existing configs without `verification` block work unchanged (default to `basic`)
4. `autospec config show` displays verification settings
5. Level is accessible throughout the codebase via config

## Dependencies

None. This is foundational and should be implemented first.

## Estimated Scope

Small. Primarily config schema extension and validation logic.
