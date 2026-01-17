# DAG Commit Verification

> **⚠️ Experimental Feature (Dev Builds Only)**
>
> DAG commit verification is an experimental feature available only in development builds. It is not included in production releases. Install a dev build to use this feature.

Ensure all DAG worktrees have committed code before merging.

## Overview

DAG commit verification solves a common problem: `autospec dag merge` would report success even when worktrees had uncommitted code. This happened because agents don't always follow auto-commit instructions, and git considers merging a branch with no commits ahead as successful ("Already up to date").

The solution has three components:

1. **Post-execution commit flow**: Automatically verify and ensure commits after each spec completes
2. **`dag commit` command**: Manually trigger commit flow for recovery
3. **Merge pre-flight verification**: Block merges when no commits exist

## How It Works

### Post-Execution Commit Flow

After each spec's `autospec run` completes successfully:

1. Check for uncommitted changes using `git status --porcelain`
2. If uncommitted changes exist and autocommit is enabled:
   - Execute custom commit command OR spawn agent commit session
   - Retry up to configured number of times
   - Update spec state with commit status
3. If no uncommitted changes:
   - Verify commits exist ahead of target branch
   - Mark spec as committed with SHA

```
autospec run completes
        │
        ▼
┌─────────────────────┐
│ Check uncommitted   │
│ changes (git status)│
└────────┬────────────┘
         │
    ┌────┴────┐
    │         │
    ▼         ▼
Has changes  No changes
    │              │
    ▼              ▼
Retry commit   Check commits
flow           ahead
    │              │
    ▼              ▼
Mark status    Mark committed
```

### Merge Pre-flight Verification

Before merging any specs, `dag merge` runs verification on all completed specs:

1. Check each spec has commits ahead of target branch (`git rev-list --count`)
2. Check each spec has no uncommitted changes
3. If issues found:
   - Print detailed report with file lists
   - Fail with actionable error message
   - Suggest `dag commit` or `--skip-no-commits` flag

## Commands

### dag commit

Commit uncommitted changes in DAG worktrees.

**Syntax**: `autospec dag commit <workflow-file> [flags]`

**Flags**:
- `--only <spec-id>`: Commit only the specified spec
- `--dry-run`: Preview what would be committed without making changes
- `--cmd <command>`: Custom commit command (overrides config)

**Examples**:
```bash
# Commit all uncommitted changes
autospec dag commit .autospec/dags/my-workflow.yaml

# Commit a single spec
autospec dag commit .autospec/dags/my-workflow.yaml --only 050-auth-core

# Preview what would be committed
autospec dag commit .autospec/dags/my-workflow.yaml --dry-run

# Use custom commit command
autospec dag commit .autospec/dags/my-workflow.yaml \
  --cmd "git add . && git commit -m 'feat: implement feature'"
```

### dag run (new flags)

**New Flags**:
- `--autocommit`: Force enable autocommit verification (overrides config)
- `--no-autocommit`: Force disable autocommit verification
- `--merge`: Auto-merge after successful completion (for CI)
- `--no-merge-prompt`: Skip the post-run merge prompt

**Examples**:
```bash
# Force autocommit even if disabled in config
autospec dag run .autospec/dags/workflow.yaml --autocommit

# Disable autocommit for this run
autospec dag run .autospec/dags/workflow.yaml --no-autocommit

# Auto-merge without prompting (CI mode)
autospec dag run .autospec/dags/workflow.yaml --merge

# Skip merge prompt entirely
autospec dag run .autospec/dags/workflow.yaml --no-merge-prompt
```

### dag merge (new flags)

**New Flags**:
- `--skip-no-commits`: Skip specs with no commits ahead of target branch
- `--force`: Bypass pre-flight verification (not recommended)

**Pre-flight Verification**:

The merge command now verifies all specs before any merge:

```bash
$ autospec dag merge .autospec/dags/workflow.yaml

=== Pre-flight Verification ===
Checking 5 specs for merge readiness...

✗ Verification failed for 2 specs:

  050-auth-core:
    Issue: Uncommitted changes
    Files:
      - src/auth/login.go
      - src/auth/logout.go
    Action: Run 'autospec dag commit' or commit manually

  051-user-model:
    Issue: No commits ahead of main
    Action: Run 'autospec dag run --only 051-user-model' to implement

Merge aborted. Fix issues and retry, or use --skip-no-commits.
```

**Examples**:
```bash
# Skip specs with no commits (merge only those with work)
autospec dag merge .autospec/dags/workflow.yaml --skip-no-commits

# Bypass verification (use with caution)
autospec dag merge .autospec/dags/workflow.yaml --force
```

## Configuration

### Config File Options

Add to `.autospec/config.yml`:

```yaml
dag:
  # Enable/disable autocommit verification (default: true)
  autocommit: true

  # Number of retry attempts for commit flow (default: 1)
  autocommit_retries: 1

  # Custom commit command (optional, uses agent session if empty)
  autocommit_cmd: ""
```

### DAG File Override

Override autocommit settings per-workflow in `dag.yaml`:

```yaml
schema_version: "1.0"

dag:
  name: "My Features"

execution:
  max_parallel: 4
  base_branch: "main"
  autocommit: true              # Override config
  autocommit_retries: 2         # Override config
  autocommit_cmd: "git add . && git commit -m 'auto: {{spec_id}}'"
```

### Priority Order

Settings are resolved in this order (highest to lowest priority):

1. CLI flags (`--autocommit`, `--no-autocommit`)
2. DAG file execution section
3. Config file (`dag.autocommit`)
4. Default (`true`)

### Custom Commit Command Template Variables

When using `autocommit_cmd`, these variables are available:

| Variable | Description | Example |
|----------|-------------|---------|
| `{{spec_id}}` | Spec identifier | `050-auth-core` |
| `{{worktree}}` | Worktree path | `/home/user/dag-mydag-050-auth-core` |
| `{{branch}}` | Spec branch | `dag/mydag/050-auth-core` |
| `{{base_branch}}` | Target branch | `main` |
| `{{dag_id}}` | DAG identifier | `mydag` |

**Example**:
```yaml
dag:
  autocommit_cmd: |
    git add . && git commit -m "feat({{spec_id}}): implement feature

    Branch: {{branch}}
    Worktree: {{worktree}}"
```

## State Tracking

Commit status is tracked in the run state file:

```yaml
specs:
  050-auth-core:
    status: completed
    commit_status: committed    # pending | committed | failed
    commit_sha: "abc123def..."  # SHA of commit
    commit_attempts: 1          # Number of attempts made
```

## Recovery Workflow

If specs have uncommitted changes:

```bash
# 1. Check status
autospec dag status .autospec/dags/workflow.yaml

# 2. Preview what needs committing
autospec dag commit .autospec/dags/workflow.yaml --dry-run

# 3. Commit all uncommitted changes
autospec dag commit .autospec/dags/workflow.yaml

# 4. Merge
autospec dag merge .autospec/dags/workflow.yaml
```

## Exit Codes

| Code | Command | Meaning |
|------|---------|---------|
| 0 | `dag commit` | All uncommitted changes committed |
| 1 | `dag commit` | One or more specs failed to commit |
| 0 | `dag merge` | All specs merged successfully |
| 1 | `dag merge` | Pre-flight verification failed or merge failed |
| 3 | Both | Invalid workflow file or state not found |

## Best Practices

1. **Keep autocommit enabled**: The default (`true`) ensures commits are made
2. **Use merge prompt**: Accept the post-run prompt for streamlined workflow
3. **Check dry-run first**: Use `dag commit --dry-run` before committing
4. **Don't bypass verification**: Avoid `--force` unless you understand the implications
5. **Use skip-no-commits sparingly**: Only when you intentionally have specs with no changes

## Troubleshooting

### "No commits ahead of main"

This means the spec's branch has no new commits compared to the target branch.

**Causes**:
- Agent didn't implement anything
- All changes were already in the target branch
- Implementation failed silently

**Solutions**:
```bash
# Re-run the spec
autospec dag run .autospec/dags/workflow.yaml --only 050-auth-core --clean

# Or skip during merge
autospec dag merge .autospec/dags/workflow.yaml --skip-no-commits
```

### "Uncommitted changes"

The spec has files that weren't committed.

**Solutions**:
```bash
# Use dag commit to commit them
autospec dag commit .autospec/dags/workflow.yaml --only 050-auth-core

# Or commit manually in the worktree
cd /path/to/worktree
git add .
git commit -m "feat: implement feature"
```

### Autocommit not triggering

Check your configuration:

```bash
# Show current config
autospec config show

# Verify dag.autocommit is true
autospec config get dag.autocommit
```

Override with CLI flag if needed:
```bash
autospec dag run .autospec/dags/workflow.yaml --autocommit
```

## See Also

- [DAG Orchestration](./dag-orchestration.md) - Multi-spec workflow overview
- [DAG Resume and Merge](./dag-resume-merge.md) - Merge workflow details
- [Worktree Management](./worktree.md) - Git worktree configuration
