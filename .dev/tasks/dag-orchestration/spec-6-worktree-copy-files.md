# Spec 6: Worktree Setup & Validation

## Context

Part of **DAG Multi-Spec Orchestration** - a meta-orchestrator that runs multiple `autospec run` workflows in parallel across worktrees with dependency management. See [00-summary.md](00-summary.md) for full vision.

## Scope

Enhance worktree creation with directory copying, custom setup scripts, and validation.

## Commands

- Enhancement to `autospec worktree create`
- New flag: `--skip-copy` to bypass directory copying
- New flag: `--skip-setup` to bypass setup script

## Key Deliverables

**Directory copying:**
- `worktree.copy_dirs` config option (comma-separated or list)
- Copy untracked but essential directories before setup script runs
- Handle missing dirs gracefully (warn, don't fail)

**Setup script support:**
- Default: autospec's built-in worktree setup
- Custom: user provides `worktree.setup_script` path
- Script runs after git worktree create and copy_dirs

**Custom script validation:**
- Validate worktree directory was actually created
- Validate worktree cwd differs from base repo cwd
- Validate worktree is a valid git worktree (`git worktree list` includes it)
- Fail with clear error if validation fails

## Config

```yaml
# .autospec/config.yml
worktree:
  base_dir: ""                    # Parent dir for worktrees (default: parent of repo)
  prefix: ""                      # Directory name prefix
  setup_script: ""                # Custom setup script (relative to repo)
  auto_setup: true                # Run setup automatically on create
  track_status: true              # Persist worktree state
  copy_dirs: .autospec,.claude,.opencode  # Non-tracked dirs to copy
```

## Behavior

**Default flow (no custom script):**
1. `autospec worktree create feature-x`
2. Git worktree created at `<base_dir>/<prefix>feature-x`
3. Directories in `copy_dirs` copied from source repo
4. Autospec's default setup runs (if `auto_setup: true`)

**Custom script flow:**
1. `autospec worktree create feature-x`
2. Git worktree created
3. Directories in `copy_dirs` copied
4. Custom `setup_script` executed
5. **Validation runs:**
   - Check worktree path exists and is directory
   - Check worktree path != source repo path
   - Check `git worktree list` includes new path
6. Fail with error if validation fails

## Validation Errors

```bash
$ autospec worktree create feature-x
Creating worktree: ../wt-feature-x
Copying: .autospec/, .claude/
Running setup script: ./scripts/worktree-setup.sh

ERROR: Worktree validation failed:
  - Worktree path does not exist: ../wt-feature-x
  - Run 'git worktree list' to check status

# Or:
ERROR: Worktree validation failed:
  - Worktree cwd same as source repo (script may have cd'd back)
  - Setup script must leave cwd in worktree directory
```

## NOT Included

- No glob pattern support (explicit paths only)
- No post-setup copying

## Run

```bash
autospec run -spti .dev/tasks/dag-orchestration/spec-6-worktree-copy-files.md
```
