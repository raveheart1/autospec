# Plan: Migrate from git CLI to go-git library

## Goal

Replace `exec.Command("git", ...)` calls with go-git library where possible to reduce dependency on git CLI binary.

## Benefits

- Reduced dependency on git CLI for core operations
- No shell/exec overhead for common operations
- Cross-platform consistency
- Pure Go solution for most operations

## Complete Audit of git CLI Usage

### internal/git/git.go (7 commands) - **CAN MIGRATE**

| Line | Command | go-git Equivalent | Status |
|------|---------|-------------------|--------|
| 16 | `git rev-parse --abbrev-ref HEAD` | `repo.Head().Name().Short()` | ✅ Supported |
| 26 | `git rev-parse --show-toplevel` | `worktree.Filesystem.Root()` | ✅ Supported |
| 36 | `git rev-parse --git-dir` | `PlainOpenWithOptions` with `DetectDotGit` | ✅ Supported |
| 71 | `git branch -a --format=...` | `repo.Branches()` + `repo.References().IsRemote()` | ✅ Supported |
| 196 | `git checkout -b <name>` | `worktree.Checkout(&CheckoutOptions{Create: true})` | ✅ Supported |
| 215 | `git remote` | `repo.Remotes()` | ✅ Supported |
| 234 | `git fetch --prune <remote>` | `repo.Fetch(&FetchOptions{Prune: true})` | ✅ Supported |

### internal/workflow/preflight.go (1 command) - **CAN MIGRATE**

| Line | Command | go-git Equivalent | Status |
|------|---------|-------------------|--------|
| 117 | `git rev-parse --show-toplevel` | Use `internal/git.GetRepositoryRoot()` | ✅ Reuse |

### internal/worktree/git.go (11 commands) - **CANNOT MIGRATE**

| Line | Command | go-git Support | Status |
|------|---------|----------------|--------|
| 22 | `git worktree add -b <branch> <path>` | Not supported | ❌ Keep CLI |
| 39 | `git worktree add <path> <branch>` | Not supported | ❌ Keep CLI |
| 58 | `git worktree remove/prune` | Not supported | ❌ Keep CLI |
| 71 | `git worktree list --porcelain` | Not supported | ❌ Keep CLI |
| 125 | `git status --porcelain` | `worktree.Status()` | ⚠️ Possible but has known issues |
| 139 | `git rev-parse --abbrev-ref HEAD` | `repo.Head()` | ✅ Could migrate |
| 154 | `git rev-parse --abbrev-ref @{upstream}` | Need to read `.git/config` manually | ⚠️ Complex |
| 163 | `git rev-list --count upstream..HEAD` | Need to iterate commits manually | ⚠️ Complex |
| 177 | `git rev-parse --show-toplevel` | `worktree.Filesystem.Root()` | ✅ Could migrate |
| 190 | `git rev-parse --is-inside-work-tree` | `PlainOpenWithOptions` | ✅ Could migrate |
| 203 | `git rev-parse --git-common-dir` | `PlainOpenOptions.EnableDotGitCommonDir` | ⚠️ Partial |

### internal/testutil/git_isolation.go - **KEEP CLI** (test utilities)

Test utilities intentionally use git CLI for setting up test fixtures.

## go-git Research Summary

### Fully Supported Operations

```go
import "github.com/go-git/go-git/v5"

// Open repo from any subdirectory
repo, _ := git.PlainOpenWithOptions(".", &git.PlainOpenOptions{
    DetectDotGit: true,
})

// Get current branch
head, _ := repo.Head()
branchName := head.Name().Short()  // "main"

// Get repo root
wt, _ := repo.Worktree()
root := wt.Filesystem.Root()  // "/home/user/project"

// List local branches
branches, _ := repo.Branches()
branches.ForEach(func(ref *plumbing.Reference) error {
    fmt.Println(ref.Name().Short())  // "main", "feature-x"
    return nil
})

// List remote branches
refs, _ := repo.References()
refs.ForEach(func(ref *plumbing.Reference) error {
    if ref.Name().IsRemote() {
        fmt.Println(ref.Name().Short())  // "origin/main"
    }
    return nil
})

// List remotes
remotes, _ := repo.Remotes()
for _, r := range remotes {
    fmt.Println(r.Config().Name)  // "origin"
}

// Fetch with prune
repo.Fetch(&git.FetchOptions{
    RemoteName: "origin",
    Prune:      true,
})

// Create and checkout branch
wt.Checkout(&git.CheckoutOptions{
    Branch: plumbing.NewBranchReferenceName("new-branch"),
    Create: true,
})
```

### NOT Supported by go-git

1. **Multi-worktree operations** - `git worktree add/remove/list/prune`
   - See: https://github.com/go-git/go-git/issues/88
   - Must keep using git CLI for `internal/worktree/` package

2. **Upstream tracking shorthand** - `@{upstream}` syntax
   - Need to manually read `.git/config` to find tracking branch

3. **Rev-list counting** - `git rev-list --count A..B`
   - Need to iterate commits manually with `repo.Log()`

## Implementation Plan

### Phase 1: Migrate internal/git/git.go

1. Add direct go-git import:
   ```go
   import git "github.com/go-git/go-git/v5"
   ```

2. Create shared repo opener helper:
   ```go
   func openRepo() (*git.Repository, error) {
       return git.PlainOpenWithOptions(".", &git.PlainOpenOptions{
           DetectDotGit: true,
       })
   }
   ```

3. Migrate functions:
   - `IsGitRepository()`
   - `GetRepositoryRoot()`
   - `GetCurrentBranch()`
   - `GetAllBranches()`
   - `CreateBranch()`
   - `FetchAllRemotes()` (note: go-git fetch is per-remote, need loop)

### Phase 2: Update internal/workflow/preflight.go

Replace inline `exec.Command` with call to `git.GetRepositoryRoot()`.

### Phase 3: Remove git CLI check from doctor

Remove `CheckGit()` from `internal/health/health.go` entirely:

1. Delete `CheckGit()` function
2. Remove git check from `RunHealthChecks()`
3. Update `health_test.go` to remove git-related test cases

**Rationale:** Core autospec operations will use go-git library, so git CLI is no longer a dependency. The worktree feature (which still needs git CLI) is optional/dev-only and users of that feature can troubleshoot git CLI issues themselves.

### Phase 4: Update go.mod

Change go-git from indirect to direct dependency:
```
go get github.com/go-git/go-git/v5
```

## What Stays on git CLI

| Package | Reason |
|---------|--------|
| `internal/worktree/` | Worktree operations not supported by go-git |
| `internal/testutil/` | Test fixtures intentionally use CLI |

## Risks & Mitigations

| Risk | Mitigation |
|------|------------|
| go-git `Status()` has known performance issues | Only used in worktree package which stays on CLI |
| Behavioral differences from git CLI | Comprehensive test coverage |
| Detached HEAD handling | Test explicitly |

## Testing Strategy

1. Existing tests should pass (same behavior)
2. Test in various states: clean, dirty, detached HEAD, no commits, bare repo
3. Test `DetectDotGit` from nested subdirectories
4. Performance comparison (should be faster)

## Decision

**Recommended approach**: Migrate `internal/git/git.go` and `internal/workflow/preflight.go` to go-git. Keep `internal/worktree/` on git CLI due to lack of worktree support.

This eliminates git CLI dependency for **core autospec operations** while accepting CLI dependency for the optional worktree feature.
