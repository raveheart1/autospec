# Spec 4: DAG Resume & Merge

## Context

Part of **DAG Multi-Spec Orchestration** - a meta-orchestrator that runs multiple `autospec run` workflows in parallel across worktrees with dependency management. See [00-summary.md](00-summary.md) for full vision.

## Scope

Resume failed/interrupted runs and auto-merge completed specs.

## Commands

- `autospec dag resume <run-id>` - Resume paused/failed run
- `autospec dag merge <run-id>` - Merge completed specs to base branch

## Key Deliverables

**Resume:**
- Load run state, skip completed specs
- Detect stale processes (PID no longer exists)
- Retry failed specs from their failure point
- Continue pending specs

**Merge:**
- Sequential merge strategy (one spec at a time)
- Pre-merge test execution (configurable)
- Conflict detection with pause for user resolution
- Optional worktree cleanup after successful merge

## Config

```yaml
dag:
  merge:
    strategy: "sequential"  # sequential | manual
    run_tests_before_merge: true
    test_command: "make test"
    cleanup_after_merge: false
```

## NOT Included

- No octopus merge strategy
- No automatic conflict resolution

## Run

```bash
autospec run -spti .dev/tasks/dag-orchestration/spec-4-resume-merge.md
```
