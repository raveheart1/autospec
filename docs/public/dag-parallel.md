# DAG Parallel Execution

Execute multiple feature specifications concurrently with configurable parallelism, progress tracking, and graceful interruption handling.

## Overview

The `autospec dag run` command with `--parallel` flag enables concurrent execution of specs within a DAG workflow. This provides significant time savings for large multi-spec projects where specs can run independently.

Key features:
- Concurrent spec execution with configurable parallelism limits
- Dependency-aware scheduling (specs wait for their dependencies)
- Real-time progress tracking
- Graceful SIGINT handling with state preservation
- Unified status view for monitoring runs

## Quick Start

```bash
# Run specs in parallel (default: up to 4 concurrent)
autospec dag run .autospec/dags/my-workflow.yaml --parallel

# Limit to 2 concurrent specs
autospec dag run .autospec/dags/my-workflow.yaml --parallel --max-parallel 2

# Check status of running or completed DAG run
autospec dag status
```

## Commands

### dag run (with --parallel)

Execute a DAG workflow with parallel spec execution.

```bash
autospec dag run <file> --parallel [flags]
```

**Flags:**

| Flag | Description | Default |
|------|-------------|---------|
| `--parallel` | Enable concurrent spec execution | `false` |
| `--max-parallel N` | Maximum concurrent specs (requires `--parallel`) | `4` |
| `--fail-fast` | Stop all specs on first failure (requires `--parallel`) | `false` |
| `--dry-run` | Preview execution plan without running | `false` |
| `--force` | Force recreate failed/interrupted worktrees | `false` |

**Examples:**

```bash
# Execute with default parallelism (4 concurrent specs)
autospec dag run .autospec/dags/my-workflow.yaml --parallel

# Execute with limited parallelism
autospec dag run .autospec/dags/my-workflow.yaml --parallel --max-parallel 2

# Stop everything on first failure
autospec dag run .autospec/dags/my-workflow.yaml --parallel --fail-fast

# Preview the execution plan
autospec dag run .autospec/dags/my-workflow.yaml --parallel --dry-run
```

### dag status

Show the execution status of a DAG run.

```bash
autospec dag status [run-id]
```

If no run-id is provided, shows the most recent run.

**Example output:**

```
Run ID: 20240115_143022_abc12345
DAG: .autospec/dags/my-workflow.yaml
Status: running
Started: 2024-01-15T14:30:22Z

Completed:
  ✓ 001-database-schema (2m30s)
  ✓ 002-auth-system (3m15s)

Running:
  ● 003-product-catalog [implement: task 5/12]
  ● 004-shopping-cart [plan]

Pending:
  ○ 005-payment-processing (waiting for: [004-shopping-cart])

---
Progress: 2/5 specs complete
```

**Status symbols:**

| Symbol | Meaning | Additional Info |
|--------|---------|-----------------|
| ✓ | Completed | Shows duration |
| ● | Running | Shows current stage/task |
| ○ | Pending | Shows blocking dependencies |
| ⊘ | Blocked | Shows failed dependencies |
| ✗ | Failed | Shows error message |

## How Parallel Execution Works

### Dependency-Aware Scheduling

Specs are scheduled respecting their declared dependencies:

1. **Ready specs** (no pending dependencies) are queued for execution
2. As specs complete, newly unblocked specs are queued
3. The process continues until all specs complete or are blocked

Example with dependencies:
```
A (no deps)     ──→ runs immediately
B (no deps)     ──→ runs immediately
C (depends: A)  ──→ waits for A
D (depends: B)  ──→ waits for B
E (depends: C,D)──→ waits for C and D
```

With `--max-parallel 2`:
```
Wave 1: A, B (parallel)
Wave 2: C, D (parallel, after A and B complete)
Wave 3: E (after C and D complete)
```

### Failure Handling

**Default behavior (continue on failure):**
- Failed specs mark their dependents as "blocked"
- Other independent specs continue running
- Run completes with partial success

**With `--fail-fast`:**
- First failure cancels all running specs
- State is saved immediately
- Run exits with failure status

### Output Multiplexing

During parallel execution, output from each spec is prefixed with its ID:

```
[001-database-schema] Running migrations...
[002-auth-system] Generating auth middleware...
[001-database-schema] Migration complete.
[002-auth-system] Auth middleware generated.
```

This ensures you can identify which spec produced each output line.

### Progress Tracking

Progress is displayed as specs complete:

```
Progress: 3/10 specs complete
Progress: 4/10 specs complete
```

## Graceful Interruption (Ctrl-C)

When you press Ctrl-C during a parallel run:

1. All running specs receive cancellation signal
2. Specs are given a brief window for cleanup
3. Current state is saved (within 2 seconds)
4. Worktrees are preserved for potential resume

After interruption, `dag status` shows which specs completed:

```
Run ID: 20240115_143022_abc12345
Status: interrupted

Completed:
  ✓ 001-database-schema (2m30s)

Failed:
  ✗ 002-auth-system
    Error: interrupted by signal
  ✗ 003-product-catalog
    Error: interrupted by signal

Pending:
  ○ 004-shopping-cart
```

## Best Practices

### Choosing max-parallel

| Scenario | Recommended max-parallel |
|----------|-------------------------|
| Local development | 2-4 |
| CI/CD pipeline | 4-8 |
| Resource-constrained | 1-2 |
| Many independent specs | Match CPU cores |

### Avoiding Conflicts

Parallel specs should avoid modifying shared files simultaneously:

- Each spec runs in its own git worktree
- Avoid specs that modify `go.mod`, `package.json`, or similar shared files
- If conflicts are unavoidable, use sequential mode or add dependencies

### Monitoring Long Runs

For long-running DAG executions:

```bash
# In another terminal, check status periodically
watch -n 10 autospec dag status

# Or use the run-id for a specific run
autospec dag status 20240115_143022_abc12345
```

## State Files

DAG run state is persisted to enable monitoring and recovery:

```
.autospec/state/dag-runs/
  └── 20240115_143022_abc12345.yaml  # Run state file
```

State includes:
- Run ID and DAG file path
- Overall run status
- Per-spec status (pending/running/completed/blocked/failed)
- Current stage and task for running specs
- Timestamps and durations
- Failure reasons

## Troubleshooting

### "max-parallel must be at least 1"

The `--max-parallel` flag requires a positive integer:
```bash
# Invalid
autospec dag run dag.yaml --parallel --max-parallel 0

# Valid
autospec dag run dag.yaml --parallel --max-parallel 1
```

### "--fail-fast requires --parallel flag"

The `--fail-fast` flag only works with parallel execution:
```bash
# Invalid
autospec dag run dag.yaml --fail-fast

# Valid
autospec dag run dag.yaml --parallel --fail-fast
```

### "no DAG runs found"

The `dag status` command requires at least one prior run:
```bash
# First, run a DAG
autospec dag run .autospec/dags/my-workflow.yaml --parallel

# Then check status
autospec dag status
```

### Specs appear blocked unexpectedly

If specs show as "blocked" when they shouldn't be:

1. Check for failed dependencies with `dag status`
2. Verify the DAG file has correct dependency declarations
3. Use `dag validate` to check for issues

### Worktree conflicts

If you see worktree-related errors:

```bash
# Force recreate worktrees
autospec dag run dag.yaml --parallel --force

# Or clean up manually
git worktree prune
```

## See Also

- [DAG Schema Validation](dag-validation.md) - DAG file format and validation
- [Parallel Execution Guide](parallel-execution.md) - Task-level parallelism within a single spec
- [Worktree Management](worktree.md) - Git worktree usage
