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

  # Individual feature toggles (override level presets)
  ears_validation: false       # 02-ears-spec-schema.md
  # verification_criteria: SKIPPED - constitution already defines quality gates
  verify_command: false        # 04-verify-command.md
  structured_feedback: false   # 05-structured-feedback.md
  adversarial_review: false    # 07-adversarial-review.md
  contracts: false             # 08-contracts-design-by-contract.md
  property_tests: false        # 09-property-based-testing.md
  metamorphic_tests: false     # 10-metamorphic-testing.md

  # Threshold overrides
  mutation_threshold: 0.8
  coverage_threshold: 0.85
  complexity_max: 10
```

### Verification Levels (Presets)

| Level | What It Enables |
|-------|-----------------|
| `basic` | Current behavior. No additional verification. |
| `enhanced` | EARS validation, verification criteria, structured feedback, contracts |
| `full` | All enhanced + adversarial review, property tests, metamorphic tests |

### Level → Feature Mapping

| Feature | Toggle | `basic` | `enhanced` | `full` |
|---------|--------|---------|------------|--------|
| EARS validation | `ears_validation` | - | ✓ | ✓ |
| Verification criteria | `verification_criteria` | - | ✓ | ✓ |
| Verify command | `verify_command` | - | ✓ | ✓ |
| Structured feedback | `structured_feedback` | - | ✓ | ✓ |
| Contracts | `contracts` | - | ✓ | ✓ |
| Adversarial review | `adversarial_review` | - | - | ✓ |
| Property tests | `property_tests` | - | - | ✓ |
| Metamorphic tests | `metamorphic_tests` | - | - | ✓ |

### Individual Feature Toggles

Each feature can be explicitly enabled/disabled regardless of level:

```yaml
verification:
  level: enhanced
  # Override: enable property tests even at enhanced level
  property_tests: true
  # Override: disable contracts even though enhanced enables it
  contracts: false
```

Resolution order: **explicit toggle > level preset > default (false)**

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
