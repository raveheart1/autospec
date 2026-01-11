# Spec 2: DAG Run Sequential

## Context

Part of **DAG Multi-Spec Orchestration** - a meta-orchestrator that runs multiple `autospec run` workflows in parallel across worktrees with dependency management. See [00-summary.md](00-summary.md) for full vision.

## Scope

Core execution engine - create worktrees, run specs sequentially, track state.

## Commands

- `autospec dag run <file>` - Execute DAG workflow (sequential mode)
- `autospec dag run <file> --dry-run` - Preview execution plan without running
- `autospec dag list` - List all DAG runs

## Key Deliverables

- Parse dag.yaml using Spec 1's parser
- Create worktree per spec using `worktree.Manager`
- Run `autospec run -spti` in each worktree sequentially
- Respect layer dependencies (L0 completes before L1 starts)
- Run state persistence in `.autospec/state/dag-runs/<run-id>.yaml`
- Terminal output with `[spec-id]` prefixes
- Per-spec log files in `.autospec/state/dag-runs/<run-id>/logs/`

## DAG Configuration

Add DAG-specific settings to `.autospec/config.yml`:

```yaml
# .autospec/config.yml
dag:
  on_conflict: "manual"      # Default merge conflict handling (manual | agent)
  max_spec_retries: 0        # Max auto-retry attempts per spec (0 = manual only)
  max_log_size: "50MB"       # Max log file size per spec

worktree:
  base_dir: ""               # Parent dir for worktrees (default: parent of repo)
  prefix: ""                 # Directory name prefix (e.g., "wt-")
  setup_script: ""           # Custom setup script path (relative to repo)
  setup_timeout: "5m"        # Setup script timeout
  copy_dirs: .autospec,.claude,.opencode  # Non-tracked dirs to copy to worktrees
```

- These are defaults; dag.yaml `execution` section can override per-DAG
- Config loading: env vars > project `.autospec/config.yml` > user `~/.config/autospec/config.yml` > defaults

## Worktree Strategy

**On-demand creation** (not upfront):
- Create worktree only when spec is about to execute
- Avoids disk exhaustion for large DAGs
- Branch naming: `dag/<run-id>/<spec-id>` to avoid collisions

**Existing worktree handling:**
- If worktree exists from previous run, check state file
- If state shows completed → skip spec
- If state shows failed/interrupted → prompt user or use `--force` to recreate
- Never silently overwrite existing worktree

## Run Locking

Prevent concurrent DAG runs on overlapping specs:
- Lock file: `.autospec/state/dag-runs/<run-id>.lock`
- Before starting each spec, check no other run has it locked
- Fail fast with clear error if collision detected
- Lock released on completion/failure

## Dry-Run Mode

`autospec dag run <file> --dry-run`:
- Parse and validate DAG
- Show execution order (respecting dependencies)
- List worktrees that would be created
- Show which specs exist vs would be created
- No actual execution, worktree creation, or state changes

```bash
$ autospec dag run workflow.yaml --dry-run
Dry-run: workflow.yaml (5 specs)

Execution order:
  1. 050-error-handling (create spec, new worktree)
  2. 051-retry          (create spec, new worktree)
  3. 052-caching        (existing spec, new worktree)
  4. 053-logging        (create spec, new worktree)
  5. 054-metrics        (create spec, new worktree)

Worktrees: ../wt-dag-20260110-143022-*
No changes made.
```

## Cleanup on Failure

When a spec fails:
- Mark spec as `failed` in state file
- Log failure details (stage, error message)
- Keep worktree for debugging (don't auto-delete)
- Stop execution (no `--continue-on-error` in Spec 2)
- Output: "Run 'dag resume <run-id>' to retry or 'dag retry <run-id> <spec> --clean' to restart"

## State Tracking

Track per-spec: status (pending/running/completed/failed), worktree path, timestamps, current stage, blocked_by list, failure_reason (if failed).

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
