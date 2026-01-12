# Spec 10: DAG Commit Verification

## Context

Part of **DAG Multi-Spec Orchestration** - a meta-orchestrator that runs multiple `autospec run` workflows in parallel across worktrees with dependency management. See [00-summary.md](00-summary.md) for full vision.

## Problem

`autospec dag merge` reports success even when worktrees have uncommitted code:

```bash
$ autospec dag merge 20260111_112034_ba7166b1
✓ Merged 200-repo-reader
✓ Merged 201-cli-setup
# ... all specs report success ...
✓ Merge Complete

$ ls
docs  README.md   # <-- No actual code!
```

**Root cause:** Agents don't always follow auto-commit instructions. The spec is marked "completed" when `autospec run` exits 0, regardless of whether code was committed. When `dag merge` runs, it merges branches with no commits ahead of main, which git considers a success ("Already up to date").

**Evidence from production failure:**
- 10 specs "merged" successfully
- Worktree branches at or behind main (no commits ahead)
- All implementation code shows as `??` (untracked) in worktrees
- Binary doesn't exist because code was never committed/merged

## Solution

Three-pronged approach:

1. **Post-execution commit flow**: After `autospec run` succeeds, verify and ensure commits are made
2. **Dedicated `dag commit` command**: Manual trigger for commit flow
3. **Merge pre-flight verification**: Block merges when no commits exist

## Key Deliverables

### 1. Configuration Schema

**In `.autospec/config.yml`:**
```yaml
dag:
  autocommit: true              # Enable post-run commit verification (default: true)
  autocommit_cmd: ""            # Custom commit command (optional, replaces default flow)
  autocommit_retries: 1         # Retry count if commit fails (default: 1)
```

> **Note:** When `autocommit: true` (default), the agent_preset already injects auto-commit instructions. The DAG executor just verifies the commit happened and retries if needed.

**In `dag.yaml` (per-DAG override):**
```yaml
execution:
  autocommit: true              # Override config for this DAG
  autocommit_cmd: "./scripts/dag-commit.sh {{spec_id}}"
```

### 2. Post-Execution Commit Verification (Per-Worktree)

The agent_preset already injects auto-commit instructions into `autospec run`. The DAG executor's job is to **verify** the commit happened and **retry** if needed.

**This happens immediately after EACH worktree completes**, not as a batch step at the end:

```
dag run workflow.yaml
    │
    ├─→ [200-repo-reader] autospec run → verify commit → ✓ (or retry)
    │
    ├─→ [201-cli-setup] autospec run → verify commit → ✓ (or retry)
    │
    └─→ [202-author-stats] autospec run → verify commit → ✓ (or retry)
```

**Integration point in DAG Executor:**

```go
func (e *Executor) executeSpec(ctx context.Context, feature Feature) error {
    // ... existing setup code ...

    // Run autospec in worktree
    exitCode, err := e.runAutospec(ctx, worktreePath, specID)
    if err != nil || exitCode != 0 {
        return e.markSpecFailed(specID, "implement", err)
    }

    // IMMEDIATELY verify/retry commit after successful run
    if e.config.Autocommit {
        if err := e.postExecutionCommitFlow(specID, worktreePath); err != nil {
            return e.markSpecFailed(specID, "commit", err)
        }
    }

    return e.markSpecCompleted(specID)
}
```

After `autospec run` exits with success in a worktree:

```go
func (e *Executor) postExecutionCommitFlow(specID string, worktreePath string) error {
    // 1. Check for uncommitted changes
    hasChanges, err := hasUncommittedChanges(worktreePath)
    if err != nil {
        return fmt.Errorf("checking uncommitted changes: %w", err)
    }
    if !hasChanges {
        // Agent followed auto-commit instructions, success
        return nil
    }

    // 2. Check if autocommit verification is enabled
    if !e.config.Autocommit {
        // User disabled verification - log warning but don't fail
        fmt.Fprintf(e.stdout, "[%s] Warning: uncommitted changes exist, autocommit disabled\n", specID)
        return nil
    }

    // 3. Agent didn't commit - retry with custom cmd or re-run agent
    for attempt := 0; attempt <= e.config.AutocommitRetries; attempt++ {
        fmt.Fprintf(e.stdout, "[%s] Uncommitted changes detected, retry %d/%d\n",
            specID, attempt+1, e.config.AutocommitRetries+1)

        if e.config.AutocommitCmd != "" {
            // User-provided custom commit command
            err = e.runCustomCommitCmd(specID, worktreePath)
        } else {
            // Re-run agent with commit-only prompt
            err = e.runAgentCommitSession(specID, worktreePath)
        }

        // Verify commit was made
        hasChanges, _ = hasUncommittedChanges(worktreePath)
        commitsAhead, _ := getCommitsAhead(worktreePath, e.baseBranch)

        if !hasChanges && commitsAhead > 0 {
            return nil // Success
        }
    }

    // 4. All retries exhausted
    return fmt.Errorf("failed to commit changes after %d attempts", e.config.AutocommitRetries+1)
}
```

### 3. Custom Commit Command

Template variables available:
- `{{spec_id}}` - spec ID (e.g., "200-repo-reader")
- `{{worktree}}` - absolute path to worktree
- `{{branch}}` - current branch name
- `{{base_branch}}` - target/base branch name
- `{{dag_id}}` - DAG identifier

**Example custom command:**
```bash
#!/bin/bash
# scripts/dag-commit.sh
cd "$PWD"
make fmt
git add -A
git commit -m "feat($1): implement spec"
```

**Config usage:**
```yaml
dag:
  autocommit: true
  autocommit_cmd: "./scripts/dag-commit.sh {{spec_id}}"
```

### 4. Agent Commit Session

When `autocommit_cmd` is not set, run a dedicated agent session:

```go
func (e *Executor) runAgentCommitSession(specID string, worktreePath string) error {
    prompt := fmt.Sprintf(`Commit the implementation changes for spec %s.

Current status:
%s

Instructions:
1. Review uncommitted changes with 'git status --short'
2. Add build artifacts, dependencies, cache to .gitignore if needed
3. Stage appropriate files: 'git add -A' (after .gitignore updates)
4. Create commit: 'git commit -m "feat(%s): implement spec"'

Important:
- Do NOT commit files that should be ignored (node_modules, __pycache__, .tmp, etc.)
- Update .gitignore BEFORE staging files
- Use conventional commit format`, specID, gitStatusOutput, specID)

    return e.cmdRunner.Run(ctx, worktreePath, e.stdout, e.stderr, "autospec", "run", "-p", prompt, "--no-auto-commit")
}
```

### 5. `dag commit` Command

New command to manually trigger commit flow:

```bash
# Commit changes in all specs with uncommitted changes
autospec dag commit workflow.yaml

# Commit specific spec only
autospec dag commit workflow.yaml --only 200-repo-reader

# Show what would be committed without committing
autospec dag commit workflow.yaml --dry-run

# Use custom command for this invocation
autospec dag commit workflow.yaml --cmd "./scripts/commit.sh"
```

**Implementation:**
```go
func runDagCommit(cmd *cobra.Command, args []string) error {
    // Load run state
    run, err := loadRunState(stateDir, workflowPath)

    for specID, specState := range run.Specs {
        if specState.Status != SpecStatusCompleted {
            continue // Only process completed specs
        }

        hasChanges, _ := hasUncommittedChanges(specState.WorktreePath)
        if !hasChanges {
            fmt.Printf("[%s] No uncommitted changes\n", specID)
            continue
        }

        if dryRun {
            fmt.Printf("[%s] Would commit changes:\n%s\n", specID, gitStatus)
            continue
        }

        // Run commit flow
        if err := commitFlow(specID, specState.WorktreePath); err != nil {
            return fmt.Errorf("committing %s: %w", specID, err)
        }
        fmt.Printf("[%s] Committed\n", specID)
    }
    return nil
}
```

### 6. Merge Pre-flight Verification

The merge command MUST detect worktrees with no commits ahead of target before attempting any merges. This prevents silent "success" when nothing is actually merged.

**Verification runs BEFORE any merges happen:**

```go
func (me *MergeExecutor) Merge(ctx context.Context, runID string, dag *DAGConfig) error {
    // Phase 1: Pre-flight verification of ALL specs
    issues := me.verifyAllSpecs(run, targetBranch)

    if len(issues) > 0 && !me.skipNoCommits {
        me.printVerificationReport(issues)
        return fmt.Errorf("pre-merge verification failed: %d spec(s) need attention", len(issues))
    }

    // Phase 2: Merge specs that passed verification (skip others if --skip-no-commits)
    for _, specID := range mergeOrder {
        if issues[specID] != nil && me.skipNoCommits {
            fmt.Fprintf(me.stdout, "[%s] Skipped: %s\n", specID, issues[specID].Reason)
            continue
        }
        // ... perform merge ...
    }
}

type VerificationIssue struct {
    SpecID         string
    Reason         string   // "no_commits" | "uncommitted_changes"
    CommitsAhead   int
    UncommittedFiles []string
}

func (me *MergeExecutor) verifyAllSpecs(run *DAGRun, targetBranch string) map[string]*VerificationIssue {
    issues := make(map[string]*VerificationIssue)

    for specID, specState := range run.Specs {
        if specState.Status != SpecStatusCompleted {
            continue
        }

        worktreePath := specState.WorktreePath

        // Check 1: Uncommitted changes
        files := getUncommittedFiles(worktreePath)
        if len(files) > 0 {
            issues[specID] = &VerificationIssue{
                SpecID:           specID,
                Reason:           "uncommitted_changes",
                UncommittedFiles: files,
            }
            continue
        }

        // Check 2: Commits ahead of target
        commitsAhead, _ := getCommitsAhead(worktreePath, targetBranch)
        if commitsAhead == 0 {
            issues[specID] = &VerificationIssue{
                SpecID:       specID,
                Reason:       "no_commits",
                CommitsAhead: 0,
            }
        }
    }

    return issues
}
```

**CLI flags:**
```bash
# Default: fail if any spec has no commits or uncommitted changes
autospec dag merge workflow.yaml

# Skip specs with no commits (merge only those with actual changes)
autospec dag merge workflow.yaml --skip-no-commits

# Force merge even with issues (dangerous)
autospec dag merge workflow.yaml --force
```

**Verification output (default behavior - FAILS before any merge):**
```bash
$ autospec dag merge workflow.yaml

=== Pre-merge Verification ===
Checking 10 completed specs against main...

✓ 200-repo-reader: 3 commits ahead
✓ 201-cli-setup: 2 commits ahead
✗ 202-author-stats: no commits ahead of main
✗ 203-file-stats: no commits ahead of main
✗ 204-commit-patterns: uncommitted changes (5 files)
  M internal/patterns.go
  M internal/utils.go
  ?? go.sum

=== Summary ===
Ready to merge: 2 specs
No commits: 2 specs (need 'dag commit' first)
Uncommitted changes: 1 spec (need 'dag commit' first)

ERROR: 3 spec(s) cannot be merged

Options:
  1. Run 'autospec dag commit workflow.yaml' to commit pending changes
  2. Run 'autospec dag merge workflow.yaml --skip-no-commits' to merge only ready specs
```

**With --skip-no-commits:**
```bash
$ autospec dag merge workflow.yaml --skip-no-commits

=== Pre-merge Verification ===
Checking 10 completed specs against main...

✓ 200-repo-reader: 3 commits ahead
✓ 201-cli-setup: 2 commits ahead
⊘ 202-author-stats: skipped (no commits)
⊘ 203-file-stats: skipped (no commits)
✗ 204-commit-patterns: uncommitted changes (5 files)

=== Merging 2 specs ===
Merging 200-repo-reader...
✓ Merged 200-repo-reader
Merging 201-cli-setup...
✓ Merged 201-cli-setup

=== Summary ===
Merged: 2 specs
Skipped (no commits): 2 specs
Failed (uncommitted): 1 spec

Note: Run 'autospec dag commit' to commit changes, then 'dag merge' again
```

### 7. Helper Functions

```go
// hasUncommittedChanges returns true if the worktree has any uncommitted changes.
func hasUncommittedChanges(worktreePath string) (bool, error) {
    cmd := exec.Command("git", "status", "--porcelain")
    cmd.Dir = worktreePath
    output, err := cmd.Output()
    if err != nil {
        return false, fmt.Errorf("git status: %w", err)
    }
    return len(strings.TrimSpace(string(output))) > 0, nil
}

// getUncommittedFiles returns list of uncommitted files.
func getUncommittedFiles(worktreePath string) []string {
    cmd := exec.Command("git", "status", "--porcelain")
    cmd.Dir = worktreePath
    output, _ := cmd.Output()

    var files []string
    for _, line := range strings.Split(string(output), "\n") {
        line = strings.TrimSpace(line)
        if len(line) > 3 {
            files = append(files, strings.TrimSpace(line[2:]))
        }
    }
    return files
}

// getCommitsAhead returns the number of commits in worktree branch ahead of target.
func getCommitsAhead(worktreePath, targetBranch string) (int, error) {
    // Get current branch
    cmd := exec.Command("git", "rev-list", "--count", targetBranch+"..HEAD")
    cmd.Dir = worktreePath
    output, err := cmd.Output()
    if err != nil {
        return 0, fmt.Errorf("git rev-list: %w", err)
    }

    count := strings.TrimSpace(string(output))
    return strconv.Atoi(count)
}
```

## State Updates

Add commit tracking to spec state:

```yaml
# In dag-runs/<run-id>.yaml
specs:
  200-repo-reader:
    spec_id: 200-repo-reader
    status: completed
    commit_status: committed    # NEW: pending | committed | failed
    commit_sha: "abc123"        # NEW: SHA of the implementation commit
    commit_attempts: 1          # NEW: Number of commit attempts made
```

## Execution Flow

**Per-worktree flow (happens for EACH spec during `dag run`):**

```
dag run workflow.yaml
         │
         ▼
    ┌────────────────────────────────────────────────────┐
    │  FOR EACH SPEC (respecting dependencies):          │
    │                                                    │
    │  ┌─────────────────┐                               │
    │  │ autospec run    │ ← agent_preset has auto-commit│
    │  │ in worktree     │   instructions injected       │
    │  └────────┬────────┘                               │
    │           │                                        │
    │           ▼                                        │
    │  ┌─────────────────┐                               │
    │  │ Check uncommitted│                              │
    │  │ changes?         │                              │
    │  └────────┬────────┘                               │
    │           │                                        │
    │     No    │    Yes                                 │
    │     ▼     │    ▼                                   │
    │  ┌─────┐  │  ┌──────────────┐                      │
    │  │Done │  │  │autocommit    │                      │
    │  │     │  │  │enabled?      │                      │
    │  └─────┘  │  └──────┬───────┘                      │
    │           │         │                              │
    │           │   Yes   │   No                         │
    │           │   ▼     │   ▼                          │
    │           │ ┌─────┐ │ ┌─────────┐                  │
    │           │ │Retry│ │ │Warn,    │                  │
    │           │ │flow │ │ │continue │                  │
    │           │ └──┬──┘ │ └─────────┘                  │
    │           │    │    │                              │
    │           │    ▼    │                              │
    │           │ ┌───────────────┐                      │
    │           │ │Verify commit  │                      │
    │           │ │made?          │                      │
    │           │ └───────┬───────┘                      │
    │           │         │                              │
    │           │   Yes   │   No (retries exhausted)     │
    │           │   ▼     │   ▼                          │
    │           │ ┌─────┐ │ ┌──────────┐                 │
    │           │ │Mark │ │ │Mark spec │                 │
    │           │ │done │ │ │FAILED    │                 │
    │           │ └─────┘ │ └──────────┘                 │
    │                                                    │
    └────────────────────────────────────────────────────┘
         │
         ▼
    All specs done → ready for `dag merge`
```

## CLI Changes

**New flags for `dag run`:**
```bash
# Override config for this run
autospec dag run workflow.yaml --autocommit
autospec dag run workflow.yaml --no-autocommit
```

**New flags for `dag merge`:**
```bash
# Skip pre-merge verification (dangerous)
autospec dag merge workflow.yaml --force
```

**New flags for `dag run`:**
```bash
# Skip merge prompt after completion
autospec dag run workflow.yaml --no-merge-prompt

# Auto-yes to merge prompt (non-interactive)
autospec dag run workflow.yaml --merge
```

### 8. Post-Run Merge Prompt

After `dag run` completes (success or partial success), prompt the user to run merge:

**On full success (all specs completed):**
```bash
$ autospec dag run workflow.yaml

=== DAG Run Complete ===
✓ All 5 specs completed successfully

Run 'autospec dag merge' to merge into main? [Y/n]:
```

**On partial success (some specs completed):**
```bash
$ autospec dag run workflow.yaml

=== DAG Run Complete ===
✓ 3 specs completed
✗ 2 specs failed

Run 'autospec dag merge --skip-failed' to merge completed specs? [Y/n]:
```

**Implementation:**
```go
func (e *Executor) promptForMerge(run *DAGRun) error {
    completed, failed := countSpecStatuses(run)

    if completed == 0 {
        // No specs to merge, skip prompt
        return nil
    }

    // Build prompt message
    var prompt string
    if failed == 0 {
        fmt.Fprintf(e.stdout, "\n=== DAG Run Complete ===\n")
        fmt.Fprintf(e.stdout, "✓ All %d specs completed successfully\n\n", completed)
        prompt = "Run 'autospec dag merge' to merge into main? [Y/n]: "
    } else {
        fmt.Fprintf(e.stdout, "\n=== DAG Run Complete ===\n")
        fmt.Fprintf(e.stdout, "✓ %d specs completed\n", completed)
        fmt.Fprintf(e.stdout, "✗ %d specs failed\n\n", failed)
        prompt = "Run 'autospec dag merge --skip-failed' to merge completed specs? [Y/n]: "
    }

    // Prompt user (Y is default)
    if !e.noMergePrompt && isInteractive() {
        fmt.Fprint(e.stdout, prompt)
        response := readLine()

        if response == "" || strings.ToLower(response) == "y" {
            // Run merge
            return e.runMerge(failed > 0)
        }
    }

    return nil
}
```

**Behavior:**
- Default answer is **Y** (just press Enter to merge)
- On partial success, suggests `--skip-failed` flag
- Skipped if `--no-merge-prompt` flag or non-interactive terminal
- `--merge` flag auto-accepts (for CI/scripting)

## Error Messages

Clear, actionable error messages:

```
ERROR: Spec 200-repo-reader has uncommitted changes

Uncommitted files:
  M  internal/reader.go
  M  internal/utils.go
  ?? go.sum

Options:
  1. Run 'autospec dag commit workflow.yaml --only 200-repo-reader' to commit
  2. Enable autocommit: set 'dag.autocommit: true' in .autospec/config.yml
  3. Manually commit in worktree: cd /path/to/worktree && git add -A && git commit -m "..."
```

## NOT Included

- No automatic .gitignore pattern detection (agent decides)
- No commit message validation (agent/user decides format)
- No force-commit option (always verify uncommitted = 0)
- No squash/rebase options (simple merge only)

## Run

```bash
autospec run -spti .dev/tasks/dag-orchestration/spec-10-commit-verification.md
```
