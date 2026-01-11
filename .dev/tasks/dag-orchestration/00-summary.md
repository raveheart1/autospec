# DAG Multi-Spec Orchestration: Vision

## Problem

Running multiple features in parallel with autospec is currently manual and error-prone:

```bash
# Terminal 1                    # Terminal 2                    # Terminal 3
autospec run -spti "feat A"     autospec run -spti "feat B"     autospec run -spti "feat C"
```

**Pain points:**
- Manually create worktrees for each feature
- Open multiple terminals
- Track which features depend on which
- No visibility into overall progress
- Manually merge completed features
- No recovery if one fails

## Solution

A **meta-orchestrator** that runs multiple `autospec run` workflows in parallel across git worktrees, respecting dependencies.

```bash
# Define dependencies in a DAG file
autospec dag run workflow.yaml --parallel --max-parallel 4
```

## Core Concepts

### DAG File

Defines specs (features) organized in layers with dependencies:

```yaml
layers:
  - id: "L0"
    name: "Foundation"
    features:
      - id: "050-error-handling"    # References specs/050-error-handling/
      - id: "051-retry"
        depends_on: ["050-error-handling"]

  - id: "L1"
    depends_on: ["L0"]              # Layer-level dependency
    features:
      - id: "052-caching"
```

### Execution Model

1. Parse DAG, validate specs exist in `specs/` directory
2. Create worktree per spec
3. Run specs respecting dependencies (layer + feature level)
4. Track state for resume capability
5. Merge completed specs back to base branch

### Worktree Isolation

Each spec runs in its own git worktree:
```
main repo/
../wt-050-error-handling/    ← autospec run -spti here
../wt-051-retry/             ← autospec run -spti here
../wt-052-caching/           ← autospec run -spti here
```

## Commands Overview

| Command | Purpose |
|---------|---------|
| `autospec dag validate <file>` | Check DAG structure, verify specs exist |
| `autospec dag visualize <file>` | ASCII diagram of dependencies |
| `autospec dag run <file>` | Execute specs (sequential by default) |
| `autospec dag run --parallel` | Execute specs in parallel |
| `autospec dag status [run-id]` | Unified progress view |
| `autospec dag list` | List all DAG runs |
| `autospec dag resume <run-id>` | Continue failed/interrupted run |
| `autospec dag merge <run-id>` | Merge completed specs to base |
| `autospec dag retry <run-id> <spec>` | Retry failed spec |

## Execution Flow

```
dag.yaml
    │
    ▼
┌─────────────────┐
│  Parse & Validate│ ← Verify specs exist in specs/
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Create Worktrees │ ← One per spec using worktree.Manager
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Execute Specs   │ ← Run autospec run -spti in each worktree
│ (parallel/seq)  │   Respect dependencies, track state
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Merge & Cleanup │ ← Merge to base branch, cleanup worktrees
└─────────────────┘
```

## State Tracking

Persistent state in `.autospec/state/dag-runs/<run-id>.yaml`:
- Per-spec status (pending/running/completed/failed)
- Worktree paths
- Timestamps and durations
- Current stage (specify/plan/tasks/implement)
- Dependencies and blockers

Enables resume after interruption or failure.

## Success Criteria

1. **Run 5 specs in parallel** with single command
2. **Dependencies respected** - spec B waits for spec A if it depends on A
3. **Unified status** - see all specs' progress in one view
4. **Recoverable** - resume from where it failed
5. **Auto-merge** - completed specs merge to base branch

## What This Is NOT

- Not task-level parallelization (that's `autospec implement --parallel`, already implemented)
- Not distributed execution across machines
- Not a replacement for CI/CD
