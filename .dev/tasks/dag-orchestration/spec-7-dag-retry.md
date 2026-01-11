# Spec 7: DAG Retry Command

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
autospec run -spti "Add autospec dag retry to retry failed specs in a DAG run. Detect failure stage from run state. Without --clean: resume from failure point. With --clean: remove artifacts, delete worktree, restart from specify. Update run state after retry. Support multiple spec IDs."
```
