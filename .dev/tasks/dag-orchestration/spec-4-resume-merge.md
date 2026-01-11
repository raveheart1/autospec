# Spec 4: DAG Resume & Merge

## Context

Part of **DAG Multi-Spec Orchestration** - a meta-orchestrator that runs multiple `autospec run` workflows in parallel across worktrees with dependency management. See [00-summary.md](00-summary.md) for full vision.

## Scope

Resume failed/interrupted runs and auto-merge completed specs with AI-assisted conflict resolution.

## Commands

- `autospec dag resume <run-id>` - Resume paused/failed run
- `autospec dag merge <run-id> [--branch <target>]` - Merge completed specs to target branch
- `autospec dag merge --continue` - Continue merge after manual conflict resolution

## Key Deliverables

**Schema additions (to dag.yaml):**
- Add `execution.base_branch` - worktrees created from here, default merge target
- Add `execution.on_conflict` - `manual` or `agent` for merge conflict handling

**Resume:**
- Load run state from `.autospec/state/dag-runs/<run-id>.yaml`
- Skip completed specs
- Detect stale processes (PID no longer exists → mark interrupted)
- Retry failed/interrupted specs from failure point
- Continue pending specs

**Merge:**
- Automatic merge of all completed specs to target branch
- Uses `base_branch` from dag.yaml as default target
- `--branch` flag to override target at runtime
- Configurable conflict handling via `on_conflict`
- Update run state with merge status

## Conflict Handling Config

```yaml
# dag.yaml (takes precedence)
execution:
  on_conflict: "agent"    # agent | manual

# Or .autospec/config.yml (default)
dag:
  on_conflict: "manual"   # default: manual
```

**`manual` mode:**
- Pause on conflict
- Output single copy-pastable block with full context (file, diff, spec description)
- User can paste to their preferred agent or resolve manually
- Run `dag merge --continue` after resolving

**`agent` mode:**
- Spawn agent with concise context
- Agent resolves conflict
- Continue automatically

## Agent Conflict Resolution

When `on_conflict: agent` and merge fails:

1. Identify conflicted files
2. Extract conflict sections only (not whole files)
3. Build context with full spec info:
   - File path(s)
   - Conflict diff (`<<<<` `====` `>>>>` markers)
   - Spec ID, name, and description from dag.yaml
   - Branch names (source and target)
4. Spawn agent to resolve
5. Agent edits files, stages changes
6. Complete merge, continue to next spec

**Context template (includes full spec info):**
```
Merge conflict in: src/retry/backoff.go

Spec ID: 051-retry
Spec Name: Retry with Exponential Backoff
Description: Add retry mechanism with exponential backoff
  to handle transient failures gracefully. Includes
  configurable max attempts and initial delay.

Merging branch 'feat/051-retry' into 'main'

Conflicts:
<<<<<<< HEAD
func Retry(attempts int) {
    // existing implementation
=======
func Retry(attempts int, backoff time.Duration) {
    // new implementation with backoff
>>>>>>> feat/051-retry

Resolve this conflict preserving the intent of the spec.
```

## Merge Flow

```bash
$ autospec dag merge dag-20260110-143022 --branch main

Merging 3 completed specs to main...
  [050-error-handling] ✓
  [051-retry] CONFLICT in src/retry/backoff.go
    → Spawning agent to resolve...
    → Resolved ✓
  [052-caching] ✓

All 3 specs merged to main
```

## Usage

```bash
# Resume failed run
autospec dag resume dag-20260110-143022

# Merge to main (default)
autospec dag merge dag-20260110-143022

# Merge to different branch
autospec dag merge dag-20260110-143022 --branch develop
```

## NOT Included

- No pre-merge testing (repo-agnostic)
- No octopus merge strategy
- No manual conflict resolution mode (always uses agent)

## Run

```bash
autospec run -spti .dev/tasks/dag-orchestration/spec-4-resume-merge.md
```
