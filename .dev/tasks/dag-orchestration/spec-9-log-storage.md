# Spec 9: DAG Log Storage (User Cache)

## Context

Part of **DAG Multi-Spec Orchestration** - a meta-orchestrator that runs multiple `autospec run` workflows in parallel across worktrees with dependency management. See [00-summary.md](00-summary.md) for full vision.

## Problem

Current storage puts both state and logs in the project directory:

```
.autospec/state/dag-runs/
├── workflow.yaml.state          # Small (~1KB), useful to version
└── <run-id>/
    └── logs/
        └── <spec-id>.log        # Large (up to 50MB each!)
```

**Issues:**
- Logs can be 50MB per spec × many specs = hundreds of MB
- Users accidentally commit logs to git
- Prompting to add `.gitignore` entries is intrusive
- Logs pollute project directory with ephemeral data
- State and logs have different lifecycles (state = persistent, logs = ephemeral)

## Solution

Split storage: state stays project-local (versionable), logs move to user cache (ephemeral).

**State (project-local):**
```
.autospec/state/dag-runs/
└── workflow.yaml.state           # Small, can be versioned
```

**Logs (user cache, XDG compliant):**
```
~/.cache/autospec/dag-logs/
└── <project-id>/
    └── <dag-id>/
        └── <spec-id>.log
```

## Key Deliverables

### 1. Log Directory Location

Use XDG cache directory for logs:

```go
func GetLogDir(projectID, dagID string) string {
    cacheDir := os.Getenv("XDG_CACHE_HOME")
    if cacheDir == "" {
        cacheDir = filepath.Join(os.Getenv("HOME"), ".cache")
    }
    return filepath.Join(cacheDir, "autospec", "dag-logs", projectID, dagID)
}
```

### 2. Project ID Generation

Identify the project for log storage:

```go
func GetProjectID() string {
    // Prefer git remote URL (stable across clones)
    if remote := getGitRemoteURL(); remote != "" {
        return slugifyURL(remote)  // "github-com-user-repo"
    }
    // Fallback to path hash (local-only repos)
    absPath, _ := filepath.Abs(".")
    return hash(absPath)[:12]  // "a1b2c3d4e5f6"
}

func slugifyURL(url string) string {
    // git@github.com:user/repo.git → github-com-user-repo
    // https://github.com/user/repo → github-com-user-repo
    // Remove protocol, convert special chars to hyphens, truncate
}
```

### 3. Log Path Structure

Full path example:
```
~/.cache/autospec/dag-logs/github-com-user-myproject/gitstats-cli-v1/200-repo-reader.log
                           └─────────────────────────┘└─────────────┘└─────────────────┘
                                  project-id            dag-id          spec-id
```

Uses DAG ID from spec-8 for organization.

### 4. State File References

State file stores relative log references:

```yaml
# .autospec/state/dag-runs/workflow.yaml.state
dag_id: "gitstats-cli-v1"
dag_name: "GitStats CLI v1"
project_id: "github-com-user-myproject"
log_base: "~/.cache/autospec/dag-logs/github-com-user-myproject/gitstats-cli-v1"
specs:
  200-repo-reader:
    branch: "dag/gitstats-cli-v1/200-repo-reader"
    worktree: "../dag-gitstats-cli-v1-200-repo-reader"
    status: completed
    log_file: "200-repo-reader.log"  # Relative to log_base
```

### 5. Log Commands Update

Update `dag logs` command to use new location:

```bash
# Tail logs for a spec
❯ autospec dag logs workflow.yaml 200-repo-reader

# Shows logs from:
# ~/.cache/autospec/dag-logs/github-com-user-repo/gitstats-cli-v1/200-repo-reader.log
```

### 6. Log Cleanup

Integrate log cleanup into existing `dag cleanup` command with interactive prompt:

```bash
❯ autospec dag cleanup workflow.yaml

Removing worktrees...
  ✓ Removed dag-gitstats-cli-v1-200-repo-reader
  ✓ Removed dag-gitstats-cli-v1-201-commit-analyzer

Logs found: 127MB in ~/.cache/autospec/dag-logs/.../gitstats-cli-v1/
Delete logs? [y/N]: y
  ✓ Deleted 127MB of logs

Done.
```

**Flags to skip prompt:**

```bash
# Always delete logs (no prompt)
❯ autospec dag cleanup workflow.yaml --logs

# Never delete logs (no prompt)
❯ autospec dag cleanup workflow.yaml --no-logs

# Delete ONLY logs (keep worktrees, state)
❯ autospec dag cleanup workflow.yaml --logs-only
```

**Standalone log cleanup (all projects):**

```bash
# Clean logs for all DAGs in current project
❯ autospec dag clean-logs

Found logs for 3 DAGs (482MB total):
  - gitstats-cli-v1: 127MB
  - auth-features: 312MB
  - payments-v2: 43MB

Delete all? [y/N]: y
Cleaned 482MB of logs.

# Clean all logs across all projects
❯ autospec dag clean-logs --all

Found logs for 7 projects (1.8GB total):
  - github-com-user-repo1: 482MB
  - github-com-user-repo2: 1.1GB
  - local-a1b2c3d4: 203MB

Delete all? [y/N]: y
Cleaned 1.8GB of logs.
```

## Storage Summary

| Data | Location | Lifecycle | Git tracked? |
|------|----------|-----------|--------------|
| State | `.autospec/state/dag-runs/*.state` | Persistent | Optional (useful) |
| Logs | `~/.cache/autospec/dag-logs/` | Ephemeral | Never |
| Config | `.autospec/config.yml` | Persistent | Yes |

## Behavior Details

### First Run

1. Resolve project ID (git remote or path hash)
2. Resolve DAG ID (from spec-8)
3. Create log directory: `~/.cache/autospec/dag-logs/<project-id>/<dag-id>/`
4. Store `log_base` in state file
5. Write logs to `<log_base>/<spec-id>.log`

### Resume Run

1. Load state file
2. Use stored `log_base` for log location
3. Append to existing logs or create new

### Cross-Machine Behavior

Logs don't transfer across machines (they're in user cache). This is expected:
- State file transfers (if versioned)
- Logs are regenerated on resume
- No stale log references (path is machine-specific)

## Configuration

Optional config for non-standard setups:

```yaml
# .autospec/config.yml
dag:
  log_dir: "/custom/path/dag-logs"  # Override XDG cache
  max_log_size: "50MB"              # Existing setting
```

Environment variable override:
```bash
export AUTOSPEC_DAG_LOG_DIR="/custom/path"
```

## Migration

For existing runs with logs in project directory:
1. On next `dag run`, detect old log location
2. Move logs to new cache location
3. Update state file with new `log_base`
4. Optionally: `dag migrate-logs` command for manual migration

## NOT Included

- Automatic log compression (future enhancement)
- Log rotation within a run (truncation at max_log_size is sufficient)
- Remote log storage (out of scope)

## Run

```bash
autospec run -spti .dev/tasks/dag-orchestration/spec-9-log-storage.md
```
