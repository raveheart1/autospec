# Spec 3: DAG Parallel & Status

## Scope

Parallel execution with concurrency control and unified status view.

## Commands

- `autospec dag run <file> --parallel` - Execute with parallelization
- `autospec dag run --max-parallel N` - Limit concurrent specs
- `autospec dag status [run-id]` - Show unified status across specs

## Key Deliverables

- Parallel process management with semaphore
- Output multiplexing with `[spec-id]` prefixes
- Progress tracking (X/Y specs complete)
- Graceful Ctrl-C handling (save state before exit)
- `dag status` showing: completed, running, pending, failed specs with progress

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
autospec run -spti "Add parallel execution to autospec dag run. Add --parallel and --max-parallel flags. Use semaphore for concurrency control. Multiplex output with [spec-id] prefixes. Handle Ctrl-C by saving state. Add autospec dag status showing unified view of all specs: completed, running, pending, failed with progress."
```
