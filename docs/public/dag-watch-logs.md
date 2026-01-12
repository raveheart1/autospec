# DAG Watch & Logs

Real-time monitoring of DAG runs with easy access to per-spec logs.

## Overview

Monitor multi-spec workflows in real-time with:
- **dag watch**: Live-updating status table showing all specs in a run
- **dag logs**: Stream or view log output for a specific spec
- **dag list**: View all DAG runs with their IDs and status

Key features:
- Auto-refresh status table (configurable interval)
- Tail -f style log streaming with timestamps
- Log size management with automatic truncation
- Latest run shortcut for quick access

## Quick Start

```bash
# Watch the current active run
autospec dag watch

# Watch a specific run by ID
autospec dag watch 20260110_143022_abc12345

# Stream logs for a spec in the latest run
autospec dag logs --latest 051-retry-backoff

# View full log file without streaming
autospec dag logs --latest 051-retry-backoff --no-follow

# List all DAG runs
autospec dag list
```

## Commands

### dag watch

Display a live-updating status table for all specs in a DAG run.

```bash
autospec dag watch [run-id] [flags]
```

The watch command provides real-time visibility into DAG execution with:
- Spec ID, status, progress, duration columns
- Automatic refresh at configurable intervals
- Color-coded status indicators

**Arguments:**

| Argument | Description | Default |
|----------|-------------|---------|
| `run-id` | Specific run to watch | Most recent active run |

**Flags:**

| Flag | Description | Default |
|------|-------------|---------|
| `--interval N` | Refresh interval in seconds | `2` |

**Example output:**

```
Run: 20260110_143022_abc12345     Status: Running     Started: 5 min ago

SPEC               STATUS      PROGRESS    DURATION    LAST UPDATE
051-retry-backoff  running     8/12        3m45s       2s ago
052-error-handling pending     -           -           -
053-notifications  completed   10/10       5m12s       2m ago

Press 'q' or Ctrl+C to exit
```

**Exit:**
- Press `q` to exit cleanly
- Press `Ctrl+C` to interrupt

### dag logs

Stream or view the log output for a specific spec execution.

```bash
autospec dag logs <run-id> <spec-id> [flags]
autospec dag logs --latest <spec-id> [flags]
```

Logs are written to `.autospec/state/dag-runs/<run-id>/logs/<spec-id>.log` with timestamps.

**Arguments:**

| Argument | Description | Required |
|----------|-------------|----------|
| `run-id` | Run identifier (or use `--latest`) | Yes* |
| `spec-id` | Spec identifier (e.g., `051-retry-backoff`) | Yes |

**Flags:**

| Flag | Description | Default |
|------|-------------|---------|
| `--latest` | Use the most recent run | `false` |
| `--no-follow` | Print entire log and exit (no streaming) | `false` |

**Examples:**

```bash
# Stream logs for a running spec
autospec dag logs 20260110_143022_abc12345 051-retry-backoff

# Stream logs from latest run
autospec dag logs --latest 051-retry-backoff

# View complete log without streaming
autospec dag logs --latest 051-retry-backoff --no-follow
```

**Example output:**

```
Log: /home/user/.autospec/state/dag-runs/20260110_143022_abc12345/logs/051-retry-backoff.log

[09:45:12] Running: autospec run -spti
[09:45:13] Starting specify stage...
[09:45:45] Spec generated: specs/051-retry-backoff/spec.yaml
[09:46:01] Starting plan stage...
```

### dag list

Show all DAG runs with their IDs, status, and timing.

```bash
autospec dag list [flags]
```

**Flags:**

| Flag | Description | Default |
|------|-------------|---------|
| `--limit N` | Maximum number of runs to display | `10` |

**Example output:**

```
RUN-ID                         STATUS      SPECS       STARTED
20260110_143022_abc12345       running     3/5         5 min ago
20260109_091530_def67890       completed   5/5         1 day ago
20260108_163045_ghi11223       failed      2/5         2 days ago
```

## Log File Format

Log files are stored in the **XDG cache directory** (not the project directory) to avoid accidentally committing large log files to git:

```
~/.cache/autospec/dag-logs/<project-id>/<dag-id>/<spec-id>.log
```

If `XDG_CACHE_HOME` is set, it uses `$XDG_CACHE_HOME/autospec/dag-logs/` instead.

**Project ID**: Derived from git remote URL (slugified) or path hash for local-only repos.

**Timestamp format:**

Each line is prefixed with `[HH:MM:SS]`:
```
[09:45:12] Starting autospec run...
[09:45:13] Loading specification...
```

**Log truncation:**

Logs are automatically truncated when they exceed the configured maximum size:
- Default: 50MB per spec
- Oldest 20% of the file is removed
- A `[TRUNCATED at HH:MM:SS]` marker is added at the beginning

## Log Cleanup

### dag cleanup (with logs)

The `dag cleanup` command can also clean up logs:

```bash
# Cleanup worktrees and prompt about logs
autospec dag cleanup .autospec/dags/my-workflow.yaml

# Cleanup worktrees AND logs (no prompt)
autospec dag cleanup .autospec/dags/my-workflow.yaml --logs

# Cleanup worktrees, keep logs
autospec dag cleanup .autospec/dags/my-workflow.yaml --no-logs

# Cleanup ONLY logs (keep worktrees and state)
autospec dag cleanup .autospec/dags/my-workflow.yaml --logs-only
```

### dag clean-logs

Standalone command for bulk log cleanup:

```bash
# Clean logs for current project
autospec dag clean-logs

# Clean logs for ALL projects
autospec dag clean-logs --all
```

This displays log sizes and prompts for confirmation before deletion

## Configuration

Add to `.autospec/config.yml` or `~/.config/autospec/config.yml`:

```yaml
# DAG execution settings
dag:
  on_conflict: manual           # Merge conflict handling: manual | agent
  base_branch: ""               # Target branch for merging (empty = repo default)
  max_spec_retries: 0           # Max auto-retry per spec (0 = manual only)
  max_log_size: "50MB"          # Max log file size per spec
```

**Configuration options:**

| Key | Description | Default | Env Var |
|-----|-------------|---------|---------|
| `dag.on_conflict` | Conflict resolution strategy | `manual` | `AUTOSPEC_DAG_ON_CONFLICT` |
| `dag.base_branch` | Target branch for merges | repo default | `AUTOSPEC_DAG_BASE_BRANCH` |
| `dag.max_spec_retries` | Auto-retry attempts | `0` | `AUTOSPEC_DAG_MAX_SPEC_RETRIES` |
| `dag.max_log_size` | Maximum log file size | `50MB` | `AUTOSPEC_DAG_MAX_LOG_SIZE` |

**Size format examples:**
- `50MB` - 50 megabytes
- `100MB` - 100 megabytes
- `1GB` - 1 gigabyte

## Error Handling

**Run not found:**
```
Error: run 'invalid-run-id' not found

Available runs:
  20260110_143022_abc12345
  20260109_091530_def67890

Use 'autospec dag list' to see all runs.
```

**Spec not found:**
```
Error: spec '999-nonexistent' not found in run '20260110_143022_abc12345'

Available specs:
  051-retry-backoff
  052-error-handling
  053-notifications
```

**No active runs:**
```
No active DAG runs found.

Start a new run with: autospec dag run dag.yaml
```

## Tips

1. **Quick monitoring**: Run `dag watch` without arguments to monitor the current run
2. **Log access**: Use `dag logs --latest <spec>` to avoid typing run IDs
3. **Log files**: Logs are plain text and compatible with standard tools (`tail -f`, `grep`, `less`)
4. **Terminal narrow**: Watch table columns truncate gracefully on narrow terminals
5. **Background logs**: Open multiple terminals to watch one spec while streaming logs from another
