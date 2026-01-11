# Spec 7: DAG Retry Command

## Context

Part of **DAG Multi-Spec Orchestration** - a meta-orchestrator that runs multiple `autospec run` workflows in parallel across worktrees with dependency management. See [00-summary.md](00-summary.md) for full vision.

## Scope

Smart retry for failed specs within a DAG run.

## Commands

- `autospec dag retry <run-id> <spec-id>` - Retry failed spec
- `autospec dag retry <run-id> <spec-id> --clean` - Clean restart

## Key Deliverables

- Identify failure stage from run state (specify/plan/tasks/implement)
- Without `--clean`: resume from failure point using existing retry logic
- With `--clean`: remove spec artifacts, delete worktree, restart from scratch
- Update DAG run state after retry
- Support retrying multiple specs: `autospec dag retry <run-id> spec1 spec2`
- `autospec dag retry <run-id> --all-failed` - retry all failed specs

## Retry Limits

**DAG-specific retry config** (separate from stage `max_retries`):

```yaml
# .autospec/config.yml
dag:
  max_spec_retries: 0    # Max auto-retry attempts per spec (default: 0 = manual only)
```

- Tracks retry count per spec in run state
- After limit reached, spec marked `exhausted` (not retryable)
- `--force` flag bypasses limit for manual intervention

## Poison Pill Detection

Specs that consistently fail are flagged:

- If spec fails at same stage 3 times consecutively → mark as `poison`
- Poison specs skipped in `dag resume` (require explicit `dag retry --force`)
- Warning: "Spec 051-retry has failed 3 times at 'implement' stage. Use --force to retry."

## Dependency Validation

Before retrying:
- Check all dependencies are still `completed`
- If dependency was reset/changed, warn user
- `--ignore-deps` to bypass check (for manual debugging)

## Behavior

```bash
# Resume from where it failed
autospec dag retry dag-20260110 051-retry-backoff
# Detects: failed at implement stage, task 8
# Resumes: autospec implement --from-task 8

# Clean restart
autospec dag retry dag-20260110 051-retry-backoff --clean
# Removes: specs/051-retry-backoff/*.yaml artifacts
# Deletes: worktree
# Restarts: full autospec run -spti
```

## NOT Included

- No automatic retry policies (manual trigger only)

## Run

```bash
autospec run -spti .dev/tasks/dag-orchestration/spec-7-dag-retry.md
```
