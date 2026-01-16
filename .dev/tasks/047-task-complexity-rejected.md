# Task Complexity Feature Analysis

**Status**: REJECTED
**Spec**: `specs/047-task-complexity/spec.yaml`
**Date**: 2025-12-17

## Summary

The proposed feature adds an optional `complexity` field (XS/S/M/L/XL) to tasks in `tasks.yaml`. After analysis, this feature should be **rejected** or significantly rescoped due to costs outweighing benefits.

## Costs (Substantial)

| Cost | Impact |
|------|--------|
| **Token overhead** | Claude must reason about complexity for EVERY task - extra output tokens on every run |
| **Cognitive distraction** | Risk of conflating "break down work" with "estimate work" - degrades PRIMARY job |
| **Accuracy problem** | Complexity is subjective and context-dependent. "M" for codebase author is "L" for newcomer |
| **Schema bloat** | TaskItem struct, validation, status display, history logging all grow |
| **YAGNI** | Calibration data sounds nice but: completion time ≠ effort (includes retries, validation, API latency) |

## Benefits (Marginal)

| Benefit | Reality Check |
|---------|---------------|
| Session isolation decision | Task TITLES already communicate this. "Implement OAuth2 flow" is obviously complex. |
| Progress tracking by weight | autospec is an orchestrator, not a PM tool |
| Calibration data | Would you actually use this? To do what? |
| At-a-glance summary | Task count + titles usually sufficient |

## The Core Problem

Claude estimating complexity is like estimating how long it takes **you** to run a mile without knowing if you're a marathon runner. It's fundamentally disconnected from the implementer.

Complexity estimation requires understanding the **implementer**, not just the task:
- Familiarity with codebase
- Debugging skills
- Existing patterns and test coverage
- Hidden complexity in existing code

## Better Alternatives

1. **On-demand**: Ask Claude in chat when you actually need estimates
2. **Manual-only**: Schema supports it, Claude doesn't generate it
3. **Opt-in flag**: `autospec tasks --with-complexity` only when explicitly requested
4. **Post-hoc**: Separate `autospec estimate` command on existing tasks.yaml

## Verdict

The spec tries to make complexity a **first-class concern** of task generation. This violates single-responsibility and risks degrading the primary output (good task breakdowns) for marginal PM-style benefits that aren't autospec's purpose.

**Recommendation**: Reject or rescope to manual/opt-in only.

## If Rescoping

If any complexity feature is desired, the minimal viable approach:

1. Add `complexity` field to TaskItem schema (optional, no validation warning)
2. Do NOT have Claude generate it during `/autospec.tasks`
3. Users manually add complexity if they want it
4. Status display shows breakdown only if values present

This preserves backward compatibility and user choice without degrading task generation quality.
