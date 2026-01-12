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
- dag.yaml = definition, execution config, AND runtime state (inline)

The workflow file is the run identifier. State is embedded directly in dag.yaml (see [spec-12-inline-state.md](spec-12-inline-state.md)):

```yaml
# Definition at top...
layers:
  - id: "L0"
    features:
      - id: "050-feature"
        description: "..."

# ====== RUNTIME STATE (auto-managed) ======
run:
  status: running
  started_at: 2026-01-11T11:20:34

specs:
  050-feature:
    status: completed
    worktree: /path/to/worktree
    commit_sha: abc123
```

Agent/model come from autospec config, not dag.yaml.

## DAG Configuration

DAG-specific settings in `.autospec/config.yml` (separate from dag.yaml):

```yaml
# .autospec/config.yml
dag:
  on_conflict: "manual"      # Default conflict handling (manual | agent)
  max_spec_retries: 0        # Max auto-retry attempts (0 = manual only)
  max_log_size: "50MB"       # Max log file size per spec
  autocommit: true           # Verify/retry commit after implementation (default: true)
  autocommit_cmd: ""         # Custom commit command (optional)
  autocommit_retries: 1      # Retry count if commit fails

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
| `autospec dag commit <file>` | Commit uncommitted changes in worktrees |
| `autospec dag merge <file>` | Merge completed specs to base |
| `autospec dag cleanup <file>` | Remove worktrees for a run |

> **Note:** `dag run` is idempotent - the workflow file is the run identifier. No separate `dag resume` or `dag retry` commands needed. See [spec-7-idempotent-run.md](spec-7-idempotent-run.md).

## Execution Flow

```
dag.yaml (definition + state)
    │
    ▼
┌─────────────────┐
│  Parse & Validate│ ← Load definition, resume from inline state if exists
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Create Worktrees │ ← One per spec, from staging branch (see spec-11)
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Execute Specs   │ ← Run autospec run -spti in each worktree
│ (parallel/seq)  │   Update state in dag.yaml after each step
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Commit & Stage  │ ← Verify commits, merge to layer staging branch
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Final Merge     │ ← Merge final staging branch to base
└─────────────────┘
```

## State Tracking

State is embedded inline in the dag.yaml file itself (no separate state directory):

```yaml
# Runtime state fields (appended to dag.yaml during execution)
run:
  status: running|completed|failed
  started_at: <timestamp>
  completed_at: <timestamp>

specs:
  <spec-id>:
    status: pending|running|completed|failed
    worktree: /path/to/worktree
    started_at: <timestamp>
    completed_at: <timestamp>
    current_stage: specify|plan|tasks|implement
    commit_sha: <sha>
    commit_status: pending|committed|failed
    failure_reason: <error message>

staging:
  <layer-id>:
    branch: dag/<dag-id>/stage-<layer-id>
    specs_merged: [spec1, spec2]
```

**Design rationale** (see [spec-12-inline-state.md](spec-12-inline-state.md)):
- Workflow file IS the run identifier (no generated run_id)
- Single source of truth (definition + state in one file)
- No data redundancy (spec_id, layer_id derived from definition)
- `--fresh` flag clears state sections to restart

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

## Specs

| Spec | Description |
|------|-------------|
| [spec-1-schema-validation.md](spec-1-schema-validation.md) | DAG file parsing and validation |
| [spec-2-run-sequential.md](spec-2-run-sequential.md) | Sequential execution |
| [spec-3-parallel-status.md](spec-3-parallel-status.md) | Parallel execution and status |
| [spec-4-resume-merge.md](spec-4-resume-merge.md) | Resume and merge |
| [spec-5-watch-monitor.md](spec-5-watch-monitor.md) | Live watch/monitor |
| [spec-6-worktree-copy-files.md](spec-6-worktree-copy-files.md) | Worktree config copying |
| [spec-7-idempotent-run.md](spec-7-idempotent-run.md) | Idempotent run semantics |
| [spec-8-dag-name-branches.md](spec-8-dag-name-branches.md) | DAG-based branch naming |
| [spec-9-log-storage.md](spec-9-log-storage.md) | Log file management |
| [spec-10-commit-verification.md](spec-10-commit-verification.md) | Commit verification and autocommit |
| [spec-11-layer-merge-propagation.md](spec-11-layer-merge-propagation.md) | Layer staging branches for dependency propagation |
| [spec-12-inline-state.md](spec-12-inline-state.md) | Inline state in dag.yaml (eliminate separate state files) |
