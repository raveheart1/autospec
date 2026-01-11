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
schema_version: "1.0"

dag:
  name: "V1 Features"

execution:
  max_parallel: 4
  timeout: "2h"
  base_branch: "main"       # Worktree source, merge target
  on_conflict: "manual"     # manual | agent

layers:
  - id: "L0"
    features:
      - id: "050-error-handling"
        description: "Improve error handling with context wrapping"
      - id: "051-retry"
        description: "Add retry with exponential backoff"
        depends_on: ["050-error-handling"]

  - id: "L1"
    depends_on: ["L0"]
    features:
      - id: "052-caching"
        description: "Add caching layer for API responses"
```

### Dynamic Spec Creation

Specs are created **on-the-fly** during DAG execution:
- If `specs/<id>/` exists → resume from where it left off
- If `specs/<id>/` doesn't exist → create using `description` field

This means you can define a DAG of 10 features and run them all - specs are created as needed, each building on the context of previously completed specs.

### Execution Model

1. Parse DAG, validate descriptions exist
2. Create worktree per spec
3. For each spec (respecting dependencies):
   - Run `autospec run -spti "description"`
   - Autospec creates spec if needed, or resumes if partial
4. Track state for resume capability
5. Merge completed specs back to base branch

### Source of Truth

- `specs/` folders = spec content and completion status
- dag.yaml = which specs, dependencies, execution config
- `.autospec/state/dag-runs/` = run state (worktrees, progress)

Agent/model come from autospec config, not dag.yaml.

## DAG Configuration

DAG-specific settings in `.autospec/config.yml` (separate from dag.yaml):

```yaml
# .autospec/config.yml
dag:
  on_conflict: "manual"      # Default conflict handling (manual | agent)
  max_spec_retries: 0        # Max auto-retry attempts (0 = manual only)
  max_log_size: "50MB"       # Max log file size per spec

worktree:
  base_dir: ""               # Parent dir for worktrees
  prefix: ""                 # Directory name prefix
  setup_script: ""           # Custom setup script path
  setup_timeout: "5m"        # Setup script timeout
  copy_dirs: .autospec,.claude,.opencode  # Dirs to copy
```

These are defaults - dag.yaml `execution` section overrides per-DAG.

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
| `autospec dag run <file>` | Execute specs (idempotent: resumes if state exists) |
| `autospec dag run <file> --fresh` | Force fresh start, ignore existing state |
| `autospec dag run <file> --parallel` | Execute specs in parallel |
| `autospec dag run <file> --dry-run` | Preview execution plan |
| `autospec dag run <file> --only <spec>` | Run/resume specific spec(s) |
| `autospec dag run <file> --only <spec> --clean` | Clean restart specific spec(s) |
| `autospec dag status [<file>]` | Unified progress view |
| `autospec dag watch [<file>]` | Live status table (auto-refresh) |
| `autospec dag logs <file> <spec>` | Tail spec's log output |
| `autospec dag list` | List all DAG runs (historical) |
| `autospec dag merge <file>` | Merge completed specs to base |
| `autospec dag cleanup <file>` | Remove worktrees for a run |

> **Note:** `dag run` is idempotent - the workflow file is the run identifier. No separate `dag resume` or `dag retry` commands needed. See [spec-7-idempotent-run.md](spec-7-idempotent-run.md).

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

## Merge & Conflict Handling

Completed specs auto-merge to `base_branch`. When conflicts occur:

**`on_conflict: manual`** (default)
- Pause merge, output copy-pastable context block
- Block includes: file path, conflict diff, spec info (ID, name, description)
- User resolves manually or pastes to their preferred agent
- Run `dag merge --continue` after resolution

**`on_conflict: agent`**
- Spawn agent with conflict context
- Agent resolves, stages changes
- Merge continues automatically

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
