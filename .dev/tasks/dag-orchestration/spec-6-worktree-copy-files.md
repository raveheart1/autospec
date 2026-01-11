# Spec 6: Worktree Copy Files Config

## Context

Part of **DAG Multi-Spec Orchestration** - a meta-orchestrator that runs multiple `autospec run` workflows in parallel across worktrees with dependency management. See [00-summary.md](00-summary.md) for full vision.

## Scope

Auto-copy essential untracked files to new worktrees.

## Commands

- Enhancement to `autospec worktree create`
- New flag: `--skip-copy` to bypass

## Key Deliverables

- `worktree.copy_files` config option
- Copy untracked but essential files before setup script runs
- Support directory and file paths
- Handle missing files gracefully (warn, don't fail)

## Config

```yaml
worktree:
  copy_files:
    - ".autospec/"
    - ".claude/"
    - ".opencode/"
    - "opencode.json"
  copy_on_create: true  # default: true
```

## Behavior

1. User runs `autospec worktree create feature-x`
2. Git worktree is created
3. Files in `copy_files` are copied from source repo
4. Setup script runs (if configured)

## NOT Included

- No glob pattern support (explicit paths only)
- No post-setup copying

## Run

```bash
autospec run -spti .dev/tasks/dag-orchestration/spec-6-worktree-copy-files.md
```
