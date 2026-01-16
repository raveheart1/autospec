# Manual Test Plan: go-git Migration (Branch 100-go-git-migration)

## Overview

This feature migrates core git operations from `git` CLI to the `go-git` library for:
- Reduced external dependency
- Cross-platform consistency
- Pure Go solution for most operations

**Worktree operations remain on git CLI** (go-git v5 lacks support).

## Test Environment

- Binary: `bin/autospec` (built via `make build`)
- Test repo: Current autospec repo or any git repository

---

## 1. Prerequisites Command (IS_GIT_REPO)

### Test 1.1: JSON output includes IS_GIT_REPO (in git repo)

```bash
cd /home/ari/repos/autospec
./bin/autospec prereqs --json | jq '.IS_GIT_REPO'
```

**Expected**: `true`

### Test 1.2: IS_GIT_REPO outside git repo

```bash
cd /tmp
mkdir test-no-git && cd test-no-git
/home/ari/repos/autospec/bin/autospec prereqs --json --paths-only | jq '.IS_GIT_REPO'
```

**Expected**: `false`

### Test 1.3: Text output still works (no IS_GIT_REPO in text format)

```bash
./bin/autospec prereqs
```

**Expected**: Should not crash, shows FEATURE_DIR, FEATURE_SPEC, etc.

---

## 2. Doctor Command (Git CLI Check Removed)

### Test 2.1: Doctor no longer checks for git CLI

```bash
./bin/autospec doctor
```

**Expected**:
- Should NOT show "Git CLI: ..." line
- Should show: Claude CLI, Claude settings, CLI Agents

### Test 2.2: Doctor works even if git CLI is unavailable

```bash
# Rename git temporarily (requires root or PATH manipulation)
PATH="" ./bin/autospec doctor 2>&1 || true
```

**Expected**: Should not fail due to missing git (may fail for other reasons like missing Claude CLI)

---

## 3. Core Git Functions (via go-git)

### Test 3.1: GetCurrentBranch

```bash
./bin/autospec prereqs --json | jq -r '.FEATURE_DIR'
# Verifies branch detection works (100-go-git-migration detected)
```

**Expected**: `specs/100-go-git-migration` (detected from current branch)

### Test 3.2: GetRepositoryRoot from nested directory

```bash
cd /home/ari/repos/autospec/internal/git
/home/ari/repos/autospec/bin/autospec prereqs --json 2>/dev/null | jq -r '.IS_GIT_REPO'
```

**Expected**: `true` (DetectDotGit finds repo root)

### Test 3.3: GetAllBranches (implicit via spec detection)

```bash
# Branch detection relies on GetAllBranches to match branches to spec dirs
./bin/autospec prereqs 2>&1
```

**Expected**: Correctly detects `100-go-git-migration` feature from branch

### Test 3.4: Debug logging for git operations

```bash
./bin/autospec --debug prereqs --json 2>&1 | grep -i "\[git\]"
```

**Expected**: Debug output showing git operations like:
- `[git] opening repository at ...`
- `[git] GetCurrentBranch: ...`

---

## 4. Worktree Commands (Still on git CLI)

### Test 4.1: Worktree list works

```bash
./bin/autospec worktree list
```

**Expected**: Either lists worktrees or shows "no additional worktrees" message

### Test 4.2: Worktree create fails gracefully without name

```bash
./bin/autospec worktree create 2>&1
```

**Expected**: Error about missing worktree name/branch

---

## 5. Edge Cases

### Test 5.1: Empty repository handling

```bash
cd /tmp && mkdir empty-repo && cd empty-repo
git init
/home/ari/repos/autospec/bin/autospec prereqs --json --paths-only | jq '.IS_GIT_REPO'
```

**Expected**: `true` (empty repo is still a git repo)

### Test 5.2: Detached HEAD state

```bash
cd /home/ari/repos/autospec
git checkout --detach HEAD
./bin/autospec prereqs --json 2>&1 || echo "Expected: may fail spec detection"
git checkout 100-go-git-migration  # Return to branch
```

**Expected**: IS_GIT_REPO is `true`, but spec detection may fail (expected behavior)

### Test 5.3: Bare repository

```bash
cd /tmp && mkdir bare-repo.git && cd bare-repo.git
git init --bare
/home/ari/repos/autospec/bin/autospec prereqs --json --paths-only 2>&1
cd /home/ari/repos/autospec
```

**Expected**: May show IS_GIT_REPO false or error (bare repos have no worktree)

---

## 6. Integration Tests

### Test 6.1: Full workflow still works

```bash
./bin/autospec st  # Status command
```

**Expected**: Shows current spec status for 100-go-git-migration

### Test 6.2: Build passes

```bash
make build && make test
```

**Expected**: All pass

---

## Test Results

| Test | Status | Notes |
|------|--------|-------|
| 1.1 IS_GIT_REPO in JSON | | |
| 1.2 IS_GIT_REPO outside repo | | |
| 1.3 Text output works | | |
| 2.1 Doctor no git check | | |
| 2.2 Doctor without git CLI | | |
| 3.1 GetCurrentBranch | | |
| 3.2 GetRepositoryRoot nested | | |
| 3.3 GetAllBranches | | |
| 3.4 Debug logging | | |
| 4.1 Worktree list | | |
| 4.2 Worktree create error | | |
| 5.1 Empty repo | | |
| 5.2 Detached HEAD | | |
| 5.3 Bare repo | | |
| 6.1 Status command | | |
| 6.2 Build + tests | | |

---

## Cleanup Commands

```bash
# Remove test directories
rm -rf /tmp/test-no-git /tmp/empty-repo /tmp/bare-repo.git
```
