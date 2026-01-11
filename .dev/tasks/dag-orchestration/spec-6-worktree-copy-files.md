# Spec 6: Worktree Copy Files Config

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
autospec run -spti "Add worktree.copy_files config to auto-copy essential files when creating worktrees. Config is list of paths. Copy from source repo to worktree before setup script runs. Handle missing files gracefully with warning. Add --skip-copy flag to bypass. Default copy_on_create to true."
```
