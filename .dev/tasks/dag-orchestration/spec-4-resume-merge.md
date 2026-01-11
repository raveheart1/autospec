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
- Detect stale processes via lock file (not PID - PIDs get reused)
- Retry failed/interrupted specs from failure point
- Continue pending specs

**Stale process detection:**
- Each running spec holds `.autospec/state/dag-runs/<run-id>/<spec-id>.lock`
- Lock file contains: PID, start timestamp, heartbeat timestamp
- Heartbeat updated every 30s while running
- Stale = lock exists but heartbeat >2 min old
- On resume: stale locks → mark spec as `interrupted`

**Merge:**
- Automatic merge of all completed specs to target branch
- Uses `base_branch` from dag.yaml as default target
- `--branch` flag to override target at runtime
- Configurable conflict handling via `on_conflict`
- Update run state with merge status

**Merge order:**
- Merge in **dependency order** (specs with no dependencies first)
- This ensures each merge builds on previous merges correctly
- If A depends on B, merge B first, then A
- Parallel-completed specs merged in dependency order, not completion order

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

## Unresolvable Conflicts

When agent mode fails to resolve (3 attempts) or produces invalid code:

1. Mark spec as `merge_failed` in state
2. Output full context block (same as manual mode)
3. Pause merge process
4. User resolves manually, then runs `dag merge --continue`
5. If `--skip-failed` flag, skip this spec and continue with others

```bash
$ autospec dag merge dag-20260110-143022

[051-retry] CONFLICT in src/retry/backoff.go
  → Agent attempt 1/3... failed (syntax error in resolution)
  → Agent attempt 2/3... failed (test compilation error)
  → Agent attempt 3/3... failed

Agent could not resolve conflict. Manual intervention required:
[copy-pastable context block here]

Run 'dag merge --continue' after resolving, or 'dag merge --skip-failed' to skip.
```

## Worktree Cleanup

**Commands:**
- `autospec dag cleanup <run-id>` - Remove worktrees for completed run
- `autospec dag cleanup <run-id> --force` - Remove even if unmerged
- `autospec dag cleanup --all` - Remove all worktrees from old runs

**Automatic cleanup:**
- After successful merge of a spec, worktree is deleted
- Failed/unmerged specs keep worktrees for debugging
- `dag merge` with `--cleanup` flag deletes worktrees after merge

**Cleanup behavior:**
- Check for uncommitted changes before deleting
- Warn if worktree has unpushed commits
- `--force` bypasses all checks

```bash
$ autospec dag cleanup dag-20260110-143022
Cleaning up 5 worktrees...
  ✓ wt-050-error-handling (merged, deleted)
  ✓ wt-051-retry (merged, deleted)
  ⚠ wt-052-caching (unmerged, kept)
  ✓ wt-053-logging (merged, deleted)
  ✓ wt-054-metrics (merged, deleted)

4 worktrees removed, 1 kept (unmerged)
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
- No octopus merge strategy (sequential merges only)

## Run

```bash
autospec run -spti .dev/tasks/dag-orchestration/spec-4-resume-merge.md
```
