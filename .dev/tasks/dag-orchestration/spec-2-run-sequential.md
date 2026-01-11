# Spec 2: DAG Run Sequential

## Scope

Core execution engine - create worktrees, run specs sequentially, track state.

## Commands

- `autospec dag run <file>` - Execute DAG workflow (sequential mode)
- `autospec dag list` - List all DAG runs

## Key Deliverables

- Parse dag.yaml using Spec 1's parser
- Create worktree per spec using `worktree.Manager`
- Run `autospec run -spti` in each worktree sequentially
- Respect layer dependencies (L0 completes before L1 starts)
- Run state persistence in `.autospec/state/dag-runs/<run-id>.yaml`
- Terminal output with `[spec-id]` prefixes
- Per-spec log files in `.autospec/logs/dag-runs/<run-id>/`

## State Tracking

Track per-spec: status (pending/running/completed/failed), worktree path, timestamps, current stage, blocked_by list.

## NOT Included

- No parallel execution (Spec 3)
- No `--parallel` flag (Spec 3)
- No `dag status` command (Spec 3)
- No resume capability (Spec 4)
- No merge automation (Spec 4)

## Run

```bash
autospec run -spti "Add autospec dag run for sequential multi-spec execution. Parse dag.yaml, create worktree per spec using worktree.Manager, run autospec run -spti in each worktree sequentially respecting layer dependencies. Track state in .autospec/state/dag-runs/. Prefix output with [spec-id]. Create per-spec log files. Add autospec dag list showing all runs with progress. Sequential only."
```
