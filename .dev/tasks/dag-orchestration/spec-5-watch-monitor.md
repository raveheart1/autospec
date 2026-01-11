# Spec 5: DAG Watch & Logs

## Context

Part of **DAG Multi-Spec Orchestration** - a meta-orchestrator that runs multiple `autospec run` workflows in parallel across worktrees with dependency management. See [00-summary.md](00-summary.md) for full vision.

## Scope

Real-time monitoring of DAG runs with easy access to per-spec logs.

## Commands

- `autospec dag watch [run-id]` - Live status table (auto-refresh)
- `autospec dag logs <run-id> <spec-id>` - Tail specific spec's log
- `autospec dag list` - List all runs with IDs (enhanced from Spec 2)

## Key Deliverables

**Log files:**
- Each spec writes output to `.autospec/state/dag-runs/<run-id>/logs/<spec-id>.log`
- Logs captured from autospec subprocess stdout/stderr
- Append-only during execution

**Watch command:**
- `dag watch` (no args) → watch most recent active run
- `dag watch <run-id>` → watch specific run
- Auto-refresh table (default 2s, configurable with `--interval`)
- Shows: spec ID, status, progress, duration, last update
- Exit with `q` or `Ctrl+C`

**Logs command:**
- `dag logs <run-id> <spec-id>` → tail -f style streaming
- `dag logs <run-id> <spec-id> --no-follow` → dump and exit
- `dag logs --latest <spec-id>` → use most recent run

**Easy run-id access:**
- `dag list` shows run-ids prominently (first column)
- `dag watch` without args picks latest active run
- `dag logs --latest` shortcut for most recent run
- Run-id format: `dag-YYYYMMDD-HHMMSS` (human-readable timestamps)

**Command transparency:**
- Both `dag watch` and `dag logs` print the underlying command first
- Users can copy/paste to customize (different interval, multiple terminals, etc.)

## Output Examples

```bash
$ autospec dag list
RUN-ID                  STATUS      SPECS   STARTED
dag-20260110-143022     running     3/5     10 min ago
dag-20260109-091500     completed   5/5     yesterday
dag-20260108-160000     failed      2/5     2 days ago

$ autospec dag watch
Running: watch -n 2 'autospec dag status dag-20260110-143022'

SPEC                   STATUS      PROGRESS    DURATION   LAST UPDATE
050-error-handling     completed   12/12       8m 22s     5 min ago
051-retry-backoff      running     18/25       6m 10s     2s ago
052-caching            pending     -           -          waiting on 050
053-logging            pending     -           -          waiting on 050

$ autospec dag logs dag-20260110-143022 051-retry-backoff
Running: tail -f .autospec/state/dag-runs/dag-20260110-143022/logs/051-retry-backoff.log

[22:45:01] Phase 7/9 - Task 18/25
[22:45:15] Edit src/retry/backoff.go
[22:45:18] Bash go test ./internal/retry/...
[22:45:20] PASS (26ms)
...
```

## NOT Included

- No TUI/split panes (use multiple terminals)
- No interleaved multi-spec streaming

## Run

```bash
autospec run -spti .dev/tasks/dag-orchestration/spec-5-watch-monitor.md
```
