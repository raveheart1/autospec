# DAG Idempotent Runs, Merge, and Cleanup

Idempotent DAG runs that automatically resume from where they left off, merge completed specs to a target branch with AI-assisted conflict resolution, and clean up worktrees after completion.

## Overview

The DAG orchestration system uses workflow file paths as the primary identifier for runs, enabling idempotent operations:

- **Idempotent runs**: Running `dag run workflow.yaml` again automatically resumes from existing state
- **No run-ids needed**: All commands now accept workflow file path instead of opaque run-ids
- **Merge completed specs** to a target branch in dependency order
- **Clean up worktrees** to free disk space

Key features:
- Workflow-path based state identification (human-readable, no opaque run-ids)
- Automatic resume behavior for interrupted runs
- Dependency-ordered merging (dependencies merge before dependents)
- AI-assisted conflict resolution with fallback to manual mode
- Safety checks before worktree cleanup

## Quick Start

```bash
# Run a workflow (automatically resumes if interrupted)
autospec dag run .autospec/dags/my-workflow.yaml

# Force a fresh start, discarding previous state
autospec dag run .autospec/dags/my-workflow.yaml --fresh

# Run only specific specs
autospec dag run .autospec/dags/my-workflow.yaml --only spec1,spec2

# Clean and restart specific specs
autospec dag run .autospec/dags/my-workflow.yaml --only spec1 --clean

# Merge completed specs to main branch
autospec dag merge .autospec/dags/my-workflow.yaml

# Merge to a specific branch
autospec dag merge .autospec/dags/my-workflow.yaml --branch develop

# Clean up worktrees after successful merge
autospec dag cleanup .autospec/dags/my-workflow.yaml
```

## Commands

### dag resume

Resume a previously interrupted or failed DAG run from where it left off.

```bash
autospec dag resume <run-id> [flags]
```

**Flags:**

| Flag | Description | Default |
|------|-------------|---------|
| `--force` | Force recreate failed/interrupted worktrees | `false` |
| `--parallel` | Execute specs concurrently | `false` |
| `--max-parallel N` | Maximum concurrent specs (requires `--parallel`) | `4` |
| `--fail-fast` | Stop all specs on first failure | `false` |

**Examples:**

```bash
# Resume an interrupted run
autospec dag resume 20240115_120000_abc12345

# Resume with force to recreate failed worktrees
autospec dag resume 20240115_120000_abc12345 --force

# Resume with parallel execution
autospec dag resume 20240115_120000_abc12345 --parallel --max-parallel 2
```

**What it does:**

1. Loads the run state from `.autospec/state/dag-runs/<run-id>.yaml`
2. Detects stale processes via lock file heartbeat mechanism
3. Skips specs that are already completed
4. Re-executes failed, interrupted, or pending specs
5. Acquires locks for specs before resuming execution

**Exit codes:**

| Code | Meaning |
|------|---------|
| `0` | All remaining specs completed successfully |
| `1` | One or more specs failed |
| `3` | Invalid run ID or state file not found |

### dag merge

Merge all completed specs from a DAG run to a target branch in dependency order.

```bash
autospec dag merge <run-id> [flags]
```

**Flags:**

| Flag | Description | Default |
|------|-------------|---------|
| `--branch <name>` | Target branch for merging | `main` |
| `--continue` | Continue merge after manual conflict resolution | `false` |
| `--skip-failed` | Skip specs that failed to merge and continue | `false` |
| `--cleanup` | Remove worktrees after successful merge | `false` |

**Examples:**

```bash
# Merge completed specs to default branch (main)
autospec dag merge 20240115_120000_abc12345

# Merge to a specific branch
autospec dag merge 20240115_120000_abc12345 --branch develop

# Continue merge after manual conflict resolution
autospec dag merge 20240115_120000_abc12345 --continue

# Skip failed specs and continue with others
autospec dag merge 20240115_120000_abc12345 --skip-failed

# Cleanup worktrees after successful merge
autospec dag merge 20240115_120000_abc12345 --cleanup
```

**What it does:**

1. Loads the run state from `.autospec/state/dag-runs/<run-id>.yaml`
2. Computes merge order based on spec dependencies
3. Merges specs in dependency order (dependencies first)
4. Handles conflicts based on configured strategy
5. Updates merge status for each spec in the run state

**Merge behavior:**

- Only specs with `completed` status are merged
- Dependencies are always merged before their dependents
- Conflicts pause the merge for resolution (agent or manual)

**Exit codes:**

| Code | Meaning |
|------|---------|
| `0` | All specs merged successfully |
| `1` | One or more specs failed to merge |
| `3` | Invalid run ID or state file not found |

### dag cleanup

Remove worktrees for a completed DAG run.

```bash
autospec dag cleanup <run-id> [flags]
autospec dag cleanup --all
```

**Flags:**

| Flag | Description | Default |
|------|-------------|---------|
| `--force` | Force cleanup, bypassing safety checks | `false` |
| `--all` | Clean up all completed runs | `false` |

**Examples:**

```bash
# Clean up worktrees for a completed run
autospec dag cleanup 20240115_120000_abc12345

# Force cleanup even with uncommitted changes
autospec dag cleanup 20240115_120000_abc12345 --force

# Clean up all old runs
autospec dag cleanup --all
```

**What it does:**

1. Loads the run state from `.autospec/state/dag-runs/<run-id>.yaml`
2. Removes worktrees for specs with merge status `merged`
3. Preserves worktrees for failed or unmerged specs
4. Checks for uncommitted changes before deleting

**Safety checks:**

| Condition | Default Behavior | With `--force` |
|-----------|------------------|----------------|
| Uncommitted changes | Preserved | Deleted |
| Unpushed commits | Preserved | Deleted |
| Unmerged specs | Always preserved | Deleted |

**Exit codes:**

| Code | Meaning |
|------|---------|
| `0` | Cleanup completed successfully |
| `1` | One or more worktrees could not be cleaned |
| `3` | Invalid run ID or state file not found |

## Conflict Resolution

When merge conflicts occur, autospec supports two resolution strategies:

### Agent Mode (AI-Assisted)

When configured for agent mode, autospec:
1. Spawns an AI agent with the conflict context
2. The agent attempts to resolve the conflict automatically
3. If resolution fails, retries up to 3 times
4. Falls back to manual mode after 3 failed attempts

The agent receives full context including:
- File path and conflict markers
- Spec ID and description
- Source branch (being merged) and target branch

### Manual Mode

When in manual mode or after agent fallback, autospec:
1. Outputs a copy-pastable context block
2. Pauses the merge operation
3. Waits for manual resolution

**Example manual context output:**

```
================================================================================
MERGE CONFLICT - Manual Resolution Required
================================================================================

## File: internal/api/handler.go

### Context
- Spec ID: 003-product-catalog
- Description: Product listing and search
- Source Branch: 003-product-catalog (being merged)
- Target Branch: main (merge destination)

### Conflict Markers
```
Line 42:
<<<<<<< HEAD
func handleProduct(w http.ResponseWriter, r *http.Request) {
=======
func HandleProduct(ctx context.Context, w http.ResponseWriter, r *http.Request) {
>>>>>>> 003-product-catalog
```

### Resolution Steps
1. Open the file and locate the conflict markers
2. Understand the intent of both changes
3. Edit to produce the correct merged result
4. Remove ALL conflict markers (<<<<<<< ======= >>>>>>>)
5. Stage the file: git add internal/api/handler.go

================================================================================
Copy the above context to your AI assistant for help resolving.
After resolving, run: autospec dag merge --continue
================================================================================
```

After manually resolving conflicts:
1. Edit the conflicted files
2. Remove all conflict markers
3. Stage the resolved files: `git add <file>`
4. Continue the merge: `autospec dag merge --continue`

## Stale Process Detection

The resume command uses heartbeat-based detection instead of PID-based detection because PIDs can be reused by the operating system.

### How It Works

1. Each running spec writes a lock file at `.autospec/state/dag-runs/<run-id>/<spec-id>.lock`
2. The lock file contains a `heartbeat` timestamp updated every 30 seconds
3. Lock files with heartbeats older than 2 minutes are considered stale
4. Stale specs are marked as `interrupted` and can be retried

### Benefits

| Approach | Weakness | Heartbeat Solution |
|----------|----------|-------------------|
| PID-based | PIDs can be reused by OS | Timestamp doesn't suffer from reuse |
| File locks (flock) | Platform-specific behavior | Cross-platform compatibility |
| Process table inspection | OS-specific APIs | Pure Go implementation |

## Workflow Examples

### Complete DAG Workflow

```bash
# 1. Start a DAG run
autospec dag run .autospec/dags/my-workflow.yaml --parallel

# 2. (If interrupted) Resume the run
autospec dag resume 20240115_120000_abc12345

# 3. Merge completed specs to main
autospec dag merge 20240115_120000_abc12345

# 4. (If conflicts) Resolve and continue
# ... resolve conflicts manually ...
git add <resolved-files>
autospec dag merge 20240115_120000_abc12345 --continue

# 5. Clean up worktrees
autospec dag cleanup 20240115_120000_abc12345
```

### Handling Multiple Conflicts

When multiple specs have conflicts:

```bash
# Merge with --skip-failed to continue past problematic specs
autospec dag merge 20240115_120000_abc12345 --skip-failed

# Review which specs were skipped
autospec dag status 20240115_120000_abc12345

# Resolve the skipped specs individually later
```

### Force Cleanup After Issues

If you need to clean up despite uncommitted changes:

```bash
# Warning: This will lose uncommitted work!
autospec dag cleanup 20240115_120000_abc12345 --force
```

## State Files

### Run State

Run state is stored in `.autospec/state/dag-runs/<run-id>.yaml`:

```yaml
run_id: 20240115_120000_abc12345
dag_file: .autospec/dags/my-workflow.yaml
status: running
started_at: 2024-01-15T12:00:00Z
specs:
  001-database-schema:
    status: completed
    started_at: 2024-01-15T12:00:00Z
    completed_at: 2024-01-15T12:05:00Z
    worktree_path: /home/user/repo/.autospec-worktrees/001-database-schema
    merge:
      status: merged
      merged_at: 2024-01-15T14:30:00Z
      resolution_method: none
  002-auth-system:
    status: completed
    started_at: 2024-01-15T12:05:00Z
    completed_at: 2024-01-15T12:12:00Z
    worktree_path: /home/user/repo/.autospec-worktrees/002-auth-system
    merge:
      status: pending
```

### Lock Files

Lock files are stored in `.autospec/state/dag-runs/<run-id>/<spec-id>.lock`:

```yaml
spec_id: 003-product-catalog
run_id: 20240115_120000_abc12345
pid: 12345
started_at: 2024-01-15T12:15:00Z
heartbeat: 2024-01-15T12:16:30Z
```

## Troubleshooting

### "run state not found"

The run ID doesn't exist or the state file was deleted:

```bash
# List available runs
ls .autospec/state/dag-runs/

# Check for the correct run ID format
# Format: YYYYMMDD_HHMMSS_xxxxxxxx
```

### "stale lock detected"

Another process was running but appears to have crashed:

```bash
# Resume will automatically detect and handle stale locks
autospec dag resume 20240115_120000_abc12345

# The stale spec will be marked as interrupted and retried
```

### "merge conflict in N file(s)"

Merge conflicts occurred during the merge operation:

1. Check the conflict context output
2. Resolve conflicts in the listed files
3. Stage resolved files: `git add <files>`
4. Continue: `autospec dag merge --continue`

Or skip the problematic spec:
```bash
autospec dag merge 20240115_120000_abc12345 --skip-failed
```

### "cleanup completed with N error(s)"

Some worktrees couldn't be cleaned up:

```bash
# Check which worktrees still exist
git worktree list

# Force cleanup if safe to do so
autospec dag cleanup 20240115_120000_abc12345 --force

# Or manually prune orphaned worktrees
git worktree prune
```

### "cannot specify run-id with --all flag"

The `--all` flag is mutually exclusive with specifying a run ID:

```bash
# Clean all runs (no run-id)
autospec dag cleanup --all

# Or clean a specific run (with run-id)
autospec dag cleanup 20240115_120000_abc12345
```

## Best Practices

### Before Running Large DAGs

1. **Save your work**: Commit or stash any uncommitted changes
2. **Check disk space**: Each spec creates a worktree
3. **Use --dry-run first**: Preview the execution plan

### During Execution

1. **Let heartbeats work**: Don't manually kill processes; use Ctrl-C for graceful shutdown
2. **Monitor progress**: Use `dag status` in another terminal

### After Completion

1. **Merge promptly**: Merge specs while the changes are fresh
2. **Clean up**: Remove worktrees to free disk space
3. **Review history**: Check the run state file for any issues

## See Also

- [DAG Schema Validation](dag-validation.md) - DAG file format and validation
- [DAG Parallel Execution](dag-parallel.md) - Concurrent spec execution
- [Worktree Management](worktree.md) - Git worktree usage
