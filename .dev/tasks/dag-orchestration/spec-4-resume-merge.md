# Spec 4: DAG Resume & Merge

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
autospec run -spti "Add autospec dag resume to continue failed/interrupted runs. Load state, skip completed specs, retry failed ones. Add autospec dag merge to merge completed specs to base branch sequentially. Run tests before merge if configured. Pause on conflicts for user resolution. Add merge config options. Optionally cleanup worktrees after merge."
```
