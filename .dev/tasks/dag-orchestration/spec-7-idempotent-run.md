# Spec 7: Idempotent DAG Run (Remove dag resume)

## Context

Part of **DAG Multi-Spec Orchestration** - a meta-orchestrator that runs multiple `autospec run` workflows in parallel across worktrees with dependency management. See [00-summary.md](00-summary.md) for full vision.

## Problem

The current design has two overlapping commands:
- `dag run workflow.yaml` - Start execution
- `dag resume <run-id>` - Continue failed/interrupted execution

This creates unnecessary complexity:
1. Users must remember opaque run-ids
2. Two commands for what should be one operation
3. `dag list` required to find run-ids

## Solution

Make `dag run` idempotent. Running the same workflow file automatically resumes if state exists.

## Commands

**Before (Spec 2-4 design):**
```bash
dag run workflow.yaml              # Creates run-id "dag-20260110-143022"
dag resume dag-20260110-143022     # Resume using opaque ID
dag list                           # Find the run-id
dag retry dag-20260110 spec1       # Retry specific spec
```

**After (this spec):**
```bash
dag run workflow.yaml                      # Idempotent: resume if state exists, else start
dag run workflow.yaml --fresh              # Force fresh start, ignore existing state
dag run workflow.yaml --only spec1         # Resume/run only specific spec(s)
dag run workflow.yaml --only spec1 --clean # Clean restart specific spec
```

## Key Deliverables

**State keying change:**
- Before: `.autospec/state/dag-runs/<run-id>.yaml` (opaque ID)
- After: `.autospec/state/dag-runs/<workflow-filename>.yaml` (human-readable)

Example: `dag run features/v1.yaml` stores state in `.autospec/state/dag-runs/features-v1.yaml.state`

**New flags for `dag run`:**
- `--fresh` - Ignore existing state, start from scratch
- `--only <spec-id>` - Run/resume only specified spec(s). Comma-separated: `--only spec1,spec2`
- `--clean` - When used with `--only`, clean restart: remove artifacts, delete worktree, run full `autospec run -spti`

**Idempotent behavior:**
```bash
# First run - starts fresh
$ dag run workflow.yaml
Starting new DAG run for workflow.yaml...

# Second run - resumes automatically
$ dag run workflow.yaml
Resuming DAG run for workflow.yaml (3/5 specs completed)...

# Force fresh start
$ dag run workflow.yaml --fresh
Starting fresh DAG run for workflow.yaml (discarding previous state)...
```

**Remove commands:**
- `dag resume` - Functionality absorbed into `dag run`
- `dag retry` - Functionality absorbed into `dag run --only --clean`

**Update other commands to use workflow file instead of run-id:**
- `dag status workflow.yaml` (or infer from current directory)
- `dag merge workflow.yaml`
- `dag cleanup workflow.yaml`
- `dag logs workflow.yaml <spec>`

## State File Location

Workflow path is normalized to create state filename:
- `workflow.yaml` → `.autospec/state/dag-runs/workflow.yaml.state`
- `features/v1.yaml` → `.autospec/state/dag-runs/features-v1.yaml.state`
- `/abs/path/dag.yaml` → `.autospec/state/dag-runs/dag.yaml.state` (basename only for absolute paths)

## Behavior Details

**`dag run workflow.yaml` (no flags):**
1. Check for existing state file
2. If no state: start fresh execution
3. If state exists:
   - Skip completed specs
   - Resume failed/interrupted specs from failure point
   - Continue pending specs

**`dag run workflow.yaml --fresh`:**
1. Delete existing state file if present
2. Delete associated worktrees
3. Start fresh execution

**`dag run workflow.yaml --only spec1,spec2`:**
1. Load existing state (error if none exists)
2. Run only specified specs
3. Respect dependencies (if spec1 depends on spec0, spec0 must be completed)

**`dag run workflow.yaml --only spec1 --clean`:**
1. For specified spec(s):
   - Remove `specs/<spec-id>/*.yaml` artifacts
   - Delete worktree
   - Reset spec state to `pending`
2. Run specified spec(s) fresh

## Updated Command Reference

| Command | Purpose |
|---------|---------|
| `dag validate workflow.yaml` | Check DAG structure |
| `dag visualize workflow.yaml` | ASCII dependency diagram |
| `dag run workflow.yaml` | Execute (idempotent) |
| `dag run workflow.yaml --fresh` | Force fresh start |
| `dag run workflow.yaml --parallel` | Execute in parallel |
| `dag run workflow.yaml --only <spec>` | Run specific spec(s) |
| `dag run workflow.yaml --only <spec> --clean` | Clean restart spec(s) |
| `dag status [workflow.yaml]` | Show progress |
| `dag watch [workflow.yaml]` | Live status |
| `dag logs workflow.yaml <spec>` | Tail spec logs |
| `dag merge workflow.yaml` | Merge completed specs |
| `dag cleanup workflow.yaml` | Remove worktrees |
| `dag list` | List all runs (historical) |

## Migration

Existing run-id based state files (from Spec 2-4) should be migrated:
- On first `dag run workflow.yaml`, check for matching run-id state
- If found, migrate to new filename format
- Delete old run-id state file

## NOT Included

- Multiple parallel runs of same workflow file (use different files if needed)
- Run-id as primary identifier (kept only for history/debugging in `dag list`)

## Run

```bash
autospec run -spti .dev/tasks/dag-orchestration/spec-7-idempotent-run.md
```
