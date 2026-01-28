# Bug: `autospec new-feature` Hangs When Run by Agent

## Issue Summary

When Claude (running inside autospec as an agent) tries to execute `autospec new-feature`, the command can hang indefinitely. This prevents the agent from creating new feature branches programmatically.

## Observed Behavior

```
┌─ TOOL: Bash
│ Input:
│   command: autospec new-feature --json --short-name "tui-skill-fzf" "..."
```
Command runs forever, never completing.

## Root Cause Analysis

### Location
- **File**: `internal/cli/new_feature.go:115-121`
- **Function**: `initGitForNewFeature()`

```go
func initGitForNewFeature() bool {
    hasGit := git.IsGitRepository()
    if hasGit {
        git.FetchAllRemotes() // <-- PROBLEM HERE
    }
    return hasGit
}
```

### Why It Hangs

1. `FetchAllRemotes()` calls `fetchRemote()` for each remote (`internal/git/git.go:369-397`)
2. `fetchRemote()` uses go-git's `repo.Fetch()` with no timeout (`git.go:409-414`)
3. For SSH URLs, it creates SSH agent auth (`git.go:427-434`)
4. In the agent sandbox environment:
   - `SSH_AUTH_SOCK` is not set
   - Network access may be restricted or firewalled
   - TCP connections can hang waiting for response (default OS timeout: 30-120 seconds)
5. go-git's `Fetch` operation has **no built-in timeout** - it waits indefinitely

### Evidence

When run locally with SSH agent available, the warnings appear but complete quickly:
```
[git] Warning: failed to fetch from remote 'origin': error creating SSH agent: "SSH agent requested but SSH_AUTH_SOCK not-specified"
```

In the agent sandbox, the TCP connection attempt may never fail (firewall drops packets silently), causing indefinite hang.

## Proposed Solutions

### Option 1: Add `--no-fetch` Flag (Minimal Change)

```go
var newFeatureNoFetch bool

func init() {
    newFeatureCmd.Flags().BoolVar(&newFeatureNoFetch, "no-fetch", false,
        "Skip fetching from remotes (useful in sandboxed environments)")
}

func initGitForNewFeature() bool {
    hasGit := git.IsGitRepository()
    if hasGit && !newFeatureNoFetch {
        git.FetchAllRemotes()
    }
    return hasGit
}
```

**Pros**: Simple, backwards compatible
**Cons**: Agent needs to know to use the flag

### Option 2: Auto-Detect Sandbox Environment

Skip fetch when running in known restricted environments:

```go
func initGitForNewFeature() bool {
    hasGit := git.IsGitRepository()
    if hasGit && !isRestrictedEnvironment() {
        git.FetchAllRemotes()
    }
    return hasGit
}

func isRestrictedEnvironment() bool {
    // Check for common sandbox indicators
    if os.Getenv("CLAUDE_CODE_ENTRY") != "" {
        return true  // Running inside Claude Code
    }
    if os.Getenv("SSH_AUTH_SOCK") == "" {
        // No SSH agent - fetch will likely fail anyway
        return true
    }
    return false
}
```

**Pros**: Works automatically
**Cons**: May miss some cases

### Option 3: Add Timeout to Fetch Operations (Best)

Wrap fetch in a context with timeout:

```go
func FetchAllRemotesWithTimeout(timeout time.Duration) (bool, error) {
    ctx, cancel := context.WithTimeout(context.Background(), timeout)
    defer cancel()

    // ... existing logic with ctx passed to fetch operations
}
```

**Pros**: Robust, never hangs indefinitely
**Cons**: go-git's Fetch doesn't support context directly - need wrapper

### Option 4: Skip Fetch for SSH URLs Without Auth (Recommended)

Don't even attempt fetch for SSH remotes when SSH agent isn't available:

```go
func fetchRemote(repo *git.Repository, remote *git.Remote) error {
    remoteConfig := remote.Config()
    if len(remoteConfig.URLs) == 0 {
        return nil
    }

    url := remoteConfig.URLs[0]

    // Skip SSH URLs if SSH agent unavailable
    if isSSHURL(url) && os.Getenv("SSH_AUTH_SOCK") == "" {
        logDebug("[git] skipping fetch for SSH remote %s (no SSH agent)", remoteConfig.Name)
        return nil
    }

    // ... rest of existing logic
}
```

**Pros**: Smart behavior, no new flags needed
**Cons**: Might surprise users who expect fetch attempt

## Recommended Fix

Implement **Option 4** (skip SSH fetch without agent) combined with **Option 1** (`--no-fetch` flag as escape hatch).

This ensures:
1. Command never hangs in sandbox environments
2. Users have explicit control when needed
3. Behavior is predictable and documented

## Test Plan

1. Run `autospec new-feature` normally - should still fetch
2. Run `unset SSH_AUTH_SOCK && autospec new-feature` - should skip SSH remotes
3. Run `autospec new-feature --no-fetch` - should skip all fetches
4. Run inside Claude agent sandbox - should complete quickly

## Files to Modify

- `internal/cli/new_feature.go` - Add `--no-fetch` flag
- `internal/git/git.go` - Skip SSH fetch when no agent available
- `internal/git/git_test.go` - Add tests for new behavior
