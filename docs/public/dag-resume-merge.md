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

### dag run (Idempotent)

Execute a DAG workflow. Automatically resumes from existing state if the workflow was previously interrupted.

```bash
autospec dag run <workflow-file> [flags]
```

**Flags:**

| Flag | Description | Default |
|------|-------------|---------|
| `--fresh` | Discard existing state and start fresh | `false` |
| `--only <specs>` | Run only specified specs (comma-separated) | `""` |
| `--clean` | Clean artifacts and reset state for `--only` specs | `false` |
| `--force` | Force recreate failed/interrupted worktrees | `false` |
| `--parallel` | Execute specs concurrently | `false` |
| `--max-parallel N` | Maximum concurrent specs (requires `--parallel`) | `4` |
| `--fail-fast` | Stop all specs on first failure | `false` |
| `--dry-run` | Preview execution plan without running | `false` |

**Examples:**

```bash
# Run a workflow (resumes automatically if interrupted)
autospec dag run .autospec/dags/my-workflow.yaml

# Force a fresh start
autospec dag run .autospec/dags/my-workflow.yaml --fresh

# Run only specific specs
autospec dag run .autospec/dags/my-workflow.yaml --only spec1,spec2

# Clean and restart specific specs
autospec dag run .autospec/dags/my-workflow.yaml --only spec1 --clean

# Run with parallel execution
autospec dag run .autospec/dags/my-workflow.yaml --parallel --max-parallel 2
```

**What it does:**

1. Checks for existing state in `.autospec/state/dag-runs/<workflow-name>.state`
2. If state exists, resumes from where it left off (skipping completed specs)
3. If no state exists, starts fresh execution
4. Creates worktrees for specs on-demand
5. Executes specs in dependency order (sequential or parallel)
6. Updates state after each spec completes

**Idempotent behavior:**

- Running the same command twice is safe
- Completed specs are skipped
- Failed/interrupted specs are retried
- Use `--fresh` to force a complete restart

**Exit codes:**

| Code | Meaning |
|------|---------|
| `0` | All specs completed successfully |
| `1` | One or more specs failed |
| `3` | Invalid workflow file or arguments |

### dag merge

Merge all completed specs from a DAG workflow to a target branch in dependency order.

```bash
autospec dag merge <workflow-file> [flags]
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
autospec dag merge .autospec/dags/my-workflow.yaml

# Merge to a specific branch
autospec dag merge .autospec/dags/my-workflow.yaml --branch develop

# Continue merge after manual conflict resolution
autospec dag merge .autospec/dags/my-workflow.yaml --continue

# Skip failed specs and continue with others
autospec dag merge .autospec/dags/my-workflow.yaml --skip-failed

# Cleanup worktrees after successful merge
autospec dag merge .autospec/dags/my-workflow.yaml --cleanup
```

**What it does:**

1. Loads the run state using the workflow file path
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
| `3` | Invalid workflow file or state not found |

### dag cleanup

Remove worktrees for a completed DAG workflow.

```bash
autospec dag cleanup <workflow-file> [flags]
autospec dag cleanup --all
```

**Flags:**

| Flag | Description | Default |
|------|-------------|---------|
| `--force` | Force cleanup, bypassing safety checks | `false` |
| `--all` | Clean up all completed runs | `false` |

**Examples:**

```bash
# Clean up worktrees for a completed workflow
autospec dag cleanup .autospec/dags/my-workflow.yaml

# Force cleanup even with uncommitted changes
autospec dag cleanup .autospec/dags/my-workflow.yaml --force

# Clean up all old runs
autospec dag cleanup --all
```

**What it does:**

1. Loads the run state using the workflow file path
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
| `3` | Invalid workflow file or state not found |

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

# 2. (If interrupted) Just run again - it resumes automatically
autospec dag run .autospec/dags/my-workflow.yaml --parallel

# 3. Merge completed specs to main
autospec dag merge .autospec/dags/my-workflow.yaml

# 4. (If conflicts) Resolve and continue
# ... resolve conflicts manually ...
git add <resolved-files>
autospec dag merge .autospec/dags/my-workflow.yaml --continue

# 5. Clean up worktrees
autospec dag cleanup .autospec/dags/my-workflow.yaml
```

### Handling Multiple Conflicts

When multiple specs have conflicts:

```bash
# Merge with --skip-failed to continue past problematic specs
autospec dag merge .autospec/dags/my-workflow.yaml --skip-failed

# Review which specs were skipped
autospec dag status .autospec/dags/my-workflow.yaml

# Resolve the skipped specs individually later
```

### Restarting Specific Specs

If you need to redo specific specs:

```bash
# Clean and restart a single spec
autospec dag run .autospec/dags/my-workflow.yaml --only spec1 --clean

# Clean and restart multiple specs
autospec dag run .autospec/dags/my-workflow.yaml --only spec1,spec2 --clean
```

### Force Cleanup After Issues

If you need to clean up despite uncommitted changes:

```bash
# Warning: This will lose uncommitted work!
autospec dag cleanup .autospec/dags/my-workflow.yaml --force
```

## State Files

### Run State

Run state is stored in `.autospec/state/dag-runs/<workflow-name>.state`:

For example, `.autospec/dags/my-workflow.yaml` stores state as `.autospec/state/dag-runs/.autospec-dags-my-workflow.yaml.state`.

```yaml
workflow_path: .autospec/dags/my-workflow.yaml
run_id: 20240115_120000_abc12345  # Legacy field for history
dag_file: .autospec/dags/my-workflow.yaml
status: running
started_at: 2024-01-15T12:00:00Z
updated_at: 2024-01-15T12:16:30Z
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

### Path Normalization

Workflow paths are normalized for state filenames:
- Path separators are replaced with dashes
- Absolute paths use basename only
- `.state` extension is appended

Examples:
- `my-workflow.yaml` → `my-workflow.yaml.state`
- `features/v1.yaml` → `features-v1.yaml.state`
- `/abs/path/workflow.yaml` → `workflow.yaml.state`

### Lock Files

Lock files are stored in `.autospec/state/dag-runs/<workflow-name>/<spec-id>.lock`:

```yaml
spec_id: 003-product-catalog
workflow_path: .autospec/dags/my-workflow.yaml
pid: 12345
started_at: 2024-01-15T12:15:00Z
heartbeat: 2024-01-15T12:16:30Z
```

## Troubleshooting

### "run state not found" or "no run found for workflow"

No state exists for this workflow file:

```bash
# List available state files
ls .autospec/state/dag-runs/

# Check if the workflow path is correct
# State filename format: <normalized-path>.state
```

For new workflows, this is expected. Just run the workflow with `dag run`.

### "stale lock detected"

Another process was running but appears to have crashed:

```bash
# Running the workflow again will automatically detect and handle stale locks
autospec dag run .autospec/dags/my-workflow.yaml

# The stale spec will be marked as interrupted and retried
```

### "merge conflict in N file(s)"

Merge conflicts occurred during the merge operation:

1. Check the conflict context output
2. Resolve conflicts in the listed files
3. Stage resolved files: `git add <files>`
4. Continue: `autospec dag merge .autospec/dags/my-workflow.yaml --continue`

Or skip the problematic spec:
```bash
autospec dag merge .autospec/dags/my-workflow.yaml --skip-failed
```

### "--only requires existing state"

The `--only` flag requires that a previous run exists:

```bash
# For new workflows, don't use --only - just run normally
autospec dag run .autospec/dags/my-workflow.yaml

# After a run exists, you can use --only to retry specific specs
autospec dag run .autospec/dags/my-workflow.yaml --only spec1
```

### "--clean requires --only"

The `--clean` flag cannot be used alone:

```bash
# Clean requires specifying which specs to clean
autospec dag run .autospec/dags/my-workflow.yaml --only spec1 --clean

# To restart everything fresh, use --fresh instead
autospec dag run .autospec/dags/my-workflow.yaml --fresh
```

### "cleanup completed with N error(s)"

Some worktrees couldn't be cleaned up:

```bash
# Check which worktrees still exist
git worktree list

# Force cleanup if safe to do so
autospec dag cleanup .autospec/dags/my-workflow.yaml --force

# Or manually prune orphaned worktrees
git worktree prune
```

### "cannot specify workflow file with --all flag"

The `--all` flag is mutually exclusive with specifying a workflow:

```bash
# Clean all runs (no workflow file)
autospec dag cleanup --all

# Or clean a specific workflow
autospec dag cleanup .autospec/dags/my-workflow.yaml
```

## Best Practices

### Before Running Large DAGs

1. **Save your work**: Commit or stash any uncommitted changes
2. **Check disk space**: Each spec creates a worktree
3. **Use --dry-run first**: Preview the execution plan

### During Execution

1. **Let heartbeats work**: Don't manually kill processes; use Ctrl-C for graceful shutdown
2. **Monitor progress**: Use `dag status workflow.yaml` in another terminal
3. **Resume is automatic**: Just run the same command again if interrupted

### After Completion

1. **Merge promptly**: Merge specs while the changes are fresh
2. **Clean up**: Remove worktrees to free disk space
3. **Review history**: Check the run state file for any issues

### Troubleshooting Runs

1. **Use --fresh to start over**: Discards all state and worktrees
2. **Use --only --clean for specific specs**: Restart individual specs without affecting others
3. **State files are human-readable**: Check `.autospec/state/dag-runs/` for debugging

## Migration from run-id Based Commands

If you have existing scripts using run-id based commands, update them:

| Old Command | New Command |
|-------------|-------------|
| `dag resume <run-id>` | `dag run <workflow-file>` (automatic resume) |
| `dag status <run-id>` | `dag status <workflow-file>` |
| `dag merge <run-id>` | `dag merge <workflow-file>` |
| `dag cleanup <run-id>` | `dag cleanup <workflow-file>` |
| `dag logs <run-id> <spec>` | `dag logs <workflow-file> <spec>` |

The new workflow-path based approach:
- Eliminates need to remember/copy run-ids
- Makes commands idempotent (safe to run multiple times)
- Uses human-readable identifiers

## See Also

- [DAG Schema Validation](dag-validation.md) - DAG file format and validation
- [DAG Parallel Execution](dag-parallel.md) - Concurrent spec execution
- [Worktree Management](worktree.md) - Git worktree usage
