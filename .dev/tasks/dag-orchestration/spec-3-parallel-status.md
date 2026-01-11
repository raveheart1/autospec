# Spec 3: DAG Parallel & Status

## Context

Part of **DAG Multi-Spec Orchestration** - a meta-orchestrator that runs multiple `autospec run` workflows in parallel across worktrees with dependency management. See [00-summary.md](00-summary.md) for full vision.

## Scope

Parallel execution with concurrency control and unified status view.

## Commands

- `autospec dag run <file> --parallel` - Execute with parallelization
- `autospec dag run --max-parallel N` - Limit concurrent specs
- `autospec dag status [run-id]` - Show unified status across specs

## Key Deliverables

- Parallel process management using `errgroup` with `SetLimit(maxParallel)`
- Output multiplexing with `[spec-id]` prefixes
- Progress tracking (X/Y specs complete)
- Graceful Ctrl-C handling (save state before exit)
- `dag status` showing: completed, running, pending, failed specs with progress

## Execution Model

Same algorithm as Spec 2, but with concurrency:
1. Find specs with all dependencies satisfied
2. Run up to `max-parallel` concurrently (errgroup)
3. As each completes, check for newly unblocked specs
4. Repeat until done

Sequential (Spec 2) = `max-parallel=1`
Parallel (Spec 3) = `max-parallel=N`

## Status Output Example

```
Run: dag-20260110-143022 (V1 Feature Set)
Status: running | Progress: 2/5 specs

Completed:
  ✓ 050-error-handling  [14m32s]

Running:
  ~ 051-retry-backoff   [implement 8/12 tasks]
  ~ 052-caching         [plan stage]

Pending:
  ○ 053-logging         blocked by: 051-retry-backoff
```

## NOT Included

- No resume capability (Spec 4)
- No merge automation (Spec 4)

## Run

```bash
autospec run -spti .dev/tasks/dag-orchestration/spec-3-parallel-status.md
```
