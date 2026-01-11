# Spec 5: Watch & Monitor Commands

## Context

Part of **DAG Multi-Spec Orchestration** - a meta-orchestrator that runs multiple `autospec run` workflows in parallel across worktrees with dependency management. See [00-summary.md](00-summary.md) for full vision.

## Scope

Real-time monitoring of single spec and multi-worktree dashboard.

## Commands

- `autospec watch [spec-name]` - Real-time monitoring of current spec
- `autospec watch --all` - Monitor all active worktrees

## Key Deliverables

**Single spec watch:**
- Stream task progress with timestamps
- Show tool calls (Edit, Bash, Read) as they happen
- Display test output in real-time

**Multi-worktree watch:**
- Table view of all active worktrees
- Show: name, status (specify/plan/impl), progress (X/Y tasks), last update
- Auto-refresh with configurable interval (default 2s)

## Output Examples

```
# Single spec
[22:45:01] Phase 7/9 - Task 22/30
[22:45:15] | Edit     src/handlers/auth.go
[22:45:18] | Bash     go test ./...
[22:45:20] PASS (26ms)

# All worktrees
NAME                 STATUS    PROGRESS      LAST UPDATE
050-error-handling   impl      22/30 (73%)   2s ago
051-retry-backoff    plan      -             15s ago
052-caching          done      12/12 (100%)  merged
```

## NOT Included

- No integration with DAG runs (standalone monitoring)

## Run

```bash
autospec run -spti .dev/tasks/dag-orchestration/spec-5-watch-monitor.md
```
