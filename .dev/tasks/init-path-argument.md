# Add Path Argument to `autospec init`

**Status:** Planned
**Priority:** Low
**Rationale:** Consistency with `specify init` from speckit

## Current Behavior

```bash
autospec init                    # Operates on cwd only
cd /path/to/project && autospec init  # Workaround
```

## Proposed Behavior

```bash
autospec init                      # Current directory (unchanged)
autospec init .                    # Current directory (explicit)
autospec init my-project           # Create and init new directory
autospec init ~/repos/my-project   # Absolute path
autospec init --here               # Alternative for current directory
```

## Implementation

### 1. Update Command Definition

```go
// internal/cli/config/init_cmd.go
var initCmd = &cobra.Command{
    Use:   "init [path]",
    Args:  cobra.MaximumNArgs(1),
    // ...
}

initCmd.Flags().Bool("here", false, "Initialize in current directory (same as '.')")
```

### 2. Path Resolution Logic

```go
func resolvePath(args []string, here bool) (string, error) {
    switch {
    case here || (len(args) == 1 && args[0] == "."):
        return ".", nil
    case len(args) == 1:
        path := args[0]
        if !filepath.IsAbs(path) {
            path = filepath.Join(".", path)
        }
        // Create directory if doesn't exist
        if err := os.MkdirAll(path, 0755); err != nil {
            return "", fmt.Errorf("creating directory: %w", err)
        }
        return path, nil
    default:
        return ".", nil
    }
}
```

### 3. Update File Operations

Functions that need path parameter:
- `installCommandTemplates()` → already removed, handled by agent config
- `cliagent.Configure(agent, projectDir, specsDir)` → already takes projectDir ✓
- `handleConstitution()` → needs update
- `addAutospecToGitignore()` → needs update
- `gitignoreNeedsUpdate()` → needs update
- Constitution/worktree runners → need cwd change

### 4. Change Working Directory for Subprocesses

Constitution workflow runs Claude as subprocess. Options:
- A) `os.Chdir(path)` before running (affects whole process)
- B) Pass path to orchestrator, let it handle cwd for subprocess
- C) Run init file operations with path, but require user to cd for constitution

**Recommendation:** Option A - simplest, matches user expectation

### 5. Update Config Path Logic

When path is specified:
- `--project` → `<path>/.autospec/config.yml`
- No `--project` → `~/.config/autospec/config.yml` (unchanged)

## Edge Cases

| Scenario | Behavior |
|----------|----------|
| Path exists, is file | Error: "path exists and is not a directory" |
| Path exists, not empty | Proceed (like current behavior) |
| Path doesn't exist | Create directory, then init |
| Path + `--project` | Project config at `<path>/.autospec/config.yml` |
| Relative path | Resolve relative to cwd |

## Testing

- [ ] `autospec init .` works same as `autospec init`
- [ ] `autospec init new-project` creates dir and inits
- [ ] `autospec init ~/abs/path` works with absolute paths
- [ ] `autospec init --here` works same as `.`
- [ ] `autospec init path --project` creates config in path
- [ ] Constitution runs in correct directory
- [ ] Agent commands installed in correct directory

## Not Implementing (differs from specify)

- `--no-git` - autospec doesn't init git repos
- `--force` - autospec already overwrites with `--force` flag
- `--skip-tls` - not applicable
- `--github-token` - not applicable (no template download)
