# Spec 2: DAG Run Sequential

## Context

Part of **DAG Multi-Spec Orchestration** - a meta-orchestrator that runs multiple `autospec run` workflows in parallel across worktrees with dependency management. See [00-summary.md](00-summary.md) for full vision.

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
autospec run -spti .dev/tasks/dag-orchestration/spec-2-run-sequential.md
```
