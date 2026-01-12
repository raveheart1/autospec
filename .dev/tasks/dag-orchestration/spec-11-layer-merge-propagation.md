# Spec 11: DAG Layer Merge Propagation

## Context

Part of **DAG Multi-Spec Orchestration** - a meta-orchestrator that runs multiple `autospec run` workflows in parallel across worktrees with dependency management. See [00-summary.md](00-summary.md) for full vision.

## Problem

**All worktrees currently branch from `main`, causing merge conflicts and missing dependencies.**

Current behavior:
```
main ─────────────────────────────────────────────────────────>
      │
      ├── dag/id/200-repo-reader    (Layer 0, branches from main)
      ├── dag/id/201-cli-setup      (Layer 0, branches from main)
      │
      ├── dag/id/202-author-stats   (Layer 1, branches from main!) ← PROBLEM
      ├── dag/id/203-file-stats     (Layer 1, branches from main!) ← PROBLEM
```

**The issue:** Layer 1 specs (202, 203) depend on Layer 0 (200, 201), but they branch from `main` - they don't have Layer 0's code! When you try to merge:

1. Layer 0 specs each modify `main` independently
2. Layer 1 specs build on `main` without Layer 0's changes
3. Merging Layer 1 creates conflicts because files were modified in different ways

**Real-world impact (from gitstats DAG):**
- 10 specs "merged" successfully
- `git log` shows branches diverged from different points
- `main` only has docs, no actual implementation
- Worktrees have implementation but can't be merged cleanly

## Solution: Staging Branch Per Layer

After each layer completes, merge completed specs into a staging branch. Next layer branches from that staging branch.

```
main ──────────────────────────────────────────────────────────>
      │
      ├── dag/id/200-repo-reader    (L0, from main)
      ├── dag/id/201-cli-setup      (L0, from main)
      │
      └─┬─> dag/id/stage-L0 ← merge 200 + 201 here
        │
        ├── dag/id/202-author-stats   (L1, from stage-L0) ✓ HAS L0 CODE
        ├── dag/id/203-file-stats     (L1, from stage-L0) ✓ HAS L0 CODE
        │
        └─┬─> dag/id/stage-L1 ← merge 202 + 203 here
          │
          └── dag/id/210-ci-integration (L2, from stage-L1) ✓ HAS L0+L1 CODE
```

## Key Deliverables

### 1. Configuration Schema

**In `.autospec/config.yml`:**
```yaml
dag:
  autocommit: true              # Verify/retry commit after spec completes
  autocommit_cmd: ""            # Custom commit command (optional)
  autocommit_retries: 1         # Retry count for commits

  automerge: true               # NEW: Merge into staging after each spec commits
                                # REQUIRES: autocommit: true (can't merge uncommitted code)
```

**In `dag.yaml` (per-DAG override):**
```yaml
execution:
  automerge: true               # Override for this DAG
```

**Validation:**
```go
func (c *DAGExecutionConfig) Validate() error {
    // automerge requires autocommit
    if c.IsAutomergeEnabled() && !c.IsAutocommitEnabled() {
        return fmt.Errorf("automerge requires autocommit to be enabled")
    }
    return nil
}

func (c *DAGExecutionConfig) IsAutomergeEnabled() bool {
    if c.Automerge == nil {
        return true // Default enabled
    }
    return *c.Automerge
}
```

**Integration with spec completion flow:**
```
spec execution complete
    │
    ▼
┌─────────────────────┐
│ autocommit enabled? │
└─────────┬───────────┘
    Yes   │   No
    ▼     └──> Done (uncommitted changes remain)
┌─────────────────────┐
│ Verify/retry commit │
└─────────┬───────────┘
    Success
    ▼
┌─────────────────────┐
│ automerge enabled?  │
└─────────┬───────────┘
    Yes   │   No
    ▼     └──> Done (manual merge later)
┌─────────────────────┐
│ Merge into staging  │ ← Immediate merge into layer staging branch
└─────────────────────┘
```

### 2. Staging Branch Management

```go
// StageBranch returns the staging branch name for a layer.
// Format: dag/<dag-id>/stage-<layer-id>
func (e *Executor) stageBranchName(layerID string) string {
    return fmt.Sprintf("dag/%s/stage-%s", e.state.DAGId, layerID)
}

// CreateStagingBranch creates a staging branch from the given source.
// Used to accumulate completed specs from a layer before spawning next layer.
func (e *Executor) createStagingBranch(layerID, sourceBranch string) error {
    stageBranch := e.stageBranchName(layerID)

    // Create branch from source (main for L0, previous stage for L1+)
    cmd := exec.Command("git", "branch", stageBranch, sourceBranch)
    cmd.Dir = e.repoRoot
    if output, err := cmd.CombinedOutput(); err != nil {
        return fmt.Errorf("creating stage branch %s: %w: %s", stageBranch, err, output)
    }

    return nil
}

// MergeIntoStaging merges a spec branch into the staging branch.
// Uses --no-ff to preserve merge history.
func (e *Executor) mergeIntoStaging(stageBranch, specBranch, specID string) error {
    // Checkout staging branch
    if err := e.checkout(stageBranch); err != nil {
        return fmt.Errorf("checkout staging: %w", err)
    }

    // Merge spec branch
    cmd := exec.Command("git", "merge", "--no-ff", "-m",
        fmt.Sprintf("Merge %s into %s", specID, stageBranch), specBranch)
    cmd.Dir = e.repoRoot

    if output, err := cmd.CombinedOutput(); err != nil {
        // Check for conflicts
        if conflicts := DetectConflictedFiles(e.repoRoot); len(conflicts) > 0 {
            return &MergeConflictError{
                StageBranch: stageBranch,
                SpecBranch:  specBranch,
                SpecID:      specID,
                Conflicts:   conflicts,
            }
        }
        return fmt.Errorf("merge %s into staging: %w: %s", specID, err, output)
    }

    return nil
}
```

### 2. Modified Execution Flow

```
dag run workflow.yaml
    │
    ▼
┌───────────────────────────────────────────────────────────────────┐
│  LAYER 0                                                          │
│                                                                   │
│  1. Create staging branch: dag/<id>/stage-L0 from main           │
│  2. Create worktrees from main (or stage-L0 if resuming)         │
│  3. Execute L0 specs (parallel if enabled)                       │
│  4. For each completed spec:                                      │
│     - Verify commit exists                                        │
│     - Merge spec branch → stage-L0                               │
│  5. All L0 done? → proceed to L1                                  │
└───────────────────────────────────────────────────────────────────┘
    │
    ▼
┌───────────────────────────────────────────────────────────────────┐
│  LAYER 1                                                          │
│                                                                   │
│  1. Create staging branch: dag/<id>/stage-L1 from stage-L0       │
│  2. Create worktrees from stage-L0 (has all L0 code!)            │
│  3. Execute L1 specs (parallel if enabled)                       │
│  4. For each completed spec:                                      │
│     - Verify commit exists                                        │
│     - Merge spec branch → stage-L1                               │
│  5. All L1 done? → proceed to L2                                  │
└───────────────────────────────────────────────────────────────────┘
    │
    ▼
┌───────────────────────────────────────────────────────────────────┐
│  FINAL MERGE                                                      │
│                                                                   │
│  dag merge workflow.yaml                                          │
│  - Merge final staging branch (stage-LN) → main                  │
│  - Single merge commit with all changes                          │
└───────────────────────────────────────────────────────────────────┘
```

### 3. Worktree Creation Changes

```go
// getBaseBranchForLayer returns the branch to create worktrees from.
// L0 uses base_branch (typically main).
// L1+ use the previous layer's staging branch.
func (e *Executor) getBaseBranchForLayer(layerIndex int) string {
    if layerIndex == 0 {
        return e.config.BaseBranch // "main" by default
    }

    // Use previous layer's staging branch
    prevLayerID := e.dag.Layers[layerIndex-1].ID
    return e.stageBranchName(prevLayerID)
}

// createWorktree creates a new worktree for a spec.
// CHANGED: Branch from layer's base branch, not always main.
func (e *Executor) createWorktree(specID string, layerIndex int) (string, error) {
    name := e.worktreeName(specID)
    branch := e.branchName(specID)
    baseBranch := e.getBaseBranchForLayer(layerIndex)

    fmt.Fprintf(e.stdout, "[%s] Creating worktree: branch %s from %s\n",
        specID, branch, baseBranch)

    // git worktree add -b <branch> <path> <start-point>
    // The start-point is the base branch (staging from previous layer)
    wt, err := e.worktreeManager.CreateFrom(name, branch, baseBranch, "")
    if err != nil {
        return "", fmt.Errorf("creating worktree: %w", err)
    }

    return wt.Path, nil
}
```

### 4. Worktree Manager Extension

Add `CreateFrom` to specify the start point:

```go
// CreateFrom creates a worktree with a specific start point.
// Unlike Create which branches from HEAD, this branches from startPoint.
func (m *DefaultManager) CreateFrom(name, branch, startPoint, customPath string) (*Worktree, error) {
    return m.CreateWithOptions(name, branch, customPath, CreateOptions{
        StartPoint: startPoint,
    })
}

// CreateOptions configures worktree creation.
type CreateOptions struct {
    SkipSetup   bool   // Skip running setup script
    SkipCopy    bool   // Skip copying config directories
    StartPoint  string // Branch/commit to start from (empty = HEAD)
}

// In git.go:
func GitWorktreeAddFrom(repoPath, worktreePath, branch, startPoint string) error {
    args := []string{"worktree", "add", "-b", branch, worktreePath}
    if startPoint != "" {
        args = append(args, startPoint)
    }

    cmd := exec.Command("git", args...)
    cmd.Dir = repoPath
    // ... rest of implementation
}
```

### 5. Per-Spec Automerge (Immediate After Commit)

When `automerge: true`, merge happens immediately after each spec commits:

```go
// postSpecCompletion handles commit verification and automerge.
// Called immediately after autospec run exits 0 for a spec.
func (e *Executor) postSpecCompletion(ctx context.Context, specID string, layer Layer) error {
    specState := e.state.Specs[specID]

    // Step 1: Verify/retry commit (if autocommit enabled)
    if e.config.IsAutocommitEnabled() {
        result := e.commitVerifier.PostExecutionCommitFlow(
            ctx, specID, specState.WorktreePath, specState.Branch,
            e.getBaseBranchForLayer(layer), e.state.DAGId,
        )
        specState.CommitStatus = result.Status
        specState.CommitSHA = result.CommitSHA

        if result.Status != CommitStatusCommitted {
            return fmt.Errorf("commit failed: %v", result.Error)
        }
    }

    // Step 2: Merge into staging (if automerge enabled)
    // REQUIRES: autocommit must have succeeded (validation ensures this)
    if e.config.IsAutomergeEnabled() {
        stageBranch := e.stageBranchName(layer.ID)

        fmt.Fprintf(e.stdout, "[%s] Merging into %s...\n", specID, stageBranch)

        if err := e.mergeIntoStaging(stageBranch, specState.Branch, specID); err != nil {
            if conflictErr, ok := err.(*MergeConflictError); ok {
                return e.handleStagingConflict(ctx, conflictErr)
            }
            return fmt.Errorf("automerge failed: %w", err)
        }

        specState.MergedToStaging = true
        fmt.Fprintf(e.stdout, "[%s] ✓ Merged into staging\n", specID)
    }

    // Save state after each spec
    return SaveStateByWorkflow(e.stateDir, e.state)
}
```

### 6. Layer Completion (When Automerge Disabled)

When `automerge: false`, batch merge at layer end:

```go
// completeLayer handles batch merge when automerge is disabled.
// Only called if automerge=false, otherwise specs merge individually.
func (e *Executor) completeLayer(ctx context.Context, layer Layer) error {
    if e.config.IsAutomergeEnabled() {
        // Already merged individually, nothing to do
        return nil
    }

    stageBranch := e.stageBranchName(layer.ID)

    // Get completed specs that haven't been merged yet
    unmergedSpecs := e.getUnmergedSpecsInLayer(layer)
    if len(unmergedSpecs) == 0 {
        return nil // All merged or none completed
    }

    fmt.Fprintf(e.stdout, "\n=== Merging Layer %s into staging ===\n", layer.ID)

    for _, specID := range unmergedSpecs {
        specState := e.state.Specs[specID]
        if specState == nil || specState.Branch == "" {
            continue
        }

        // Verify commit exists before merging
        if specState.CommitStatus != CommitStatusCommitted {
            fmt.Fprintf(e.stdout, "[%s] Skipped: no commit\n", specID)
            continue
        }

        fmt.Fprintf(e.stdout, "Merging %s into %s...\n", specID, stageBranch)

        if err := e.mergeIntoStaging(stageBranch, specState.Branch, specID); err != nil {
            if conflictErr, ok := err.(*MergeConflictError); ok {
                return e.handleStagingConflict(ctx, conflictErr)
            }
            return fmt.Errorf("merging %s: %w", specID, err)
        }

        specState.MergedToStaging = true
    }

    if err := SaveStateByWorkflow(e.stateDir, e.state); err != nil {
        return fmt.Errorf("saving state: %w", err)
    }

    fmt.Fprintf(e.stdout, "✓ Layer %s merged into staging\n\n", layer.ID)
    return nil
}
```

### 7. State Updates

Track staging branch status:

```yaml
# In dag-runs/<workflow>.state
run_id: "..."
staging_branches:
  L0:
    branch: dag/id/stage-L0
    created_at: 2026-01-12T10:00:00Z
    specs_merged:
      - 200-repo-reader
      - 201-cli-setup
  L1:
    branch: dag/id/stage-L1
    created_at: 2026-01-12T11:00:00Z
    specs_merged:
      - 202-author-stats
      - 203-file-stats
specs:
  200-repo-reader:
    merged_to_staging: true  # NEW: track staging merge status
    # ...
```

### 8. Final Merge Simplification

`dag merge` becomes simpler - just merge the final staging branch:

```go
func (me *MergeExecutor) Merge(ctx context.Context, run *DAGRun, dag *DAGConfig) error {
    // Find the final staging branch
    finalLayer := dag.Layers[len(dag.Layers)-1]
    finalStageBranch := fmt.Sprintf("dag/%s/stage-%s", run.DAGId, finalLayer.ID)

    // Verify staging branch has all expected specs
    if err := me.verifyFinalStaging(run, finalStageBranch); err != nil {
        return fmt.Errorf("staging verification failed: %w", err)
    }

    // Single merge: staging → main
    fmt.Fprintf(me.stdout, "Merging %s into %s...\n", finalStageBranch, me.targetBranch)

    if err := me.performMerge(ctx, finalStageBranch); err != nil {
        return err
    }

    fmt.Fprintf(me.stdout, "✓ Merged all specs to %s\n", me.targetBranch)
    return nil
}
```

### 9. Conflict Handling at Layer Boundaries

When merging specs into staging causes conflicts:

```go
type MergeConflictError struct {
    StageBranch string
    SpecBranch  string
    SpecID      string
    Conflicts   []string
}

func (e *Executor) handleLayerMergeConflict(ctx context.Context, err *MergeConflictError) error {
    fmt.Fprintf(e.stderr, "\n=== Merge Conflict ===\n")
    fmt.Fprintf(e.stderr, "Conflict merging %s into %s\n", err.SpecID, err.StageBranch)
    fmt.Fprintf(e.stderr, "Conflicting files:\n")
    for _, f := range err.Conflicts {
        fmt.Fprintf(e.stderr, "  - %s\n", f)
    }

    if e.config.OnConflict == OnConflictAgent {
        // Let agent resolve
        return e.agentResolveConflict(ctx, err)
    }

    // Manual resolution required
    fmt.Fprintf(e.stderr, "\nTo resolve:\n")
    fmt.Fprintf(e.stderr, "  1. cd %s\n", e.repoRoot)
    fmt.Fprintf(e.stderr, "  2. Resolve conflicts in the files above\n")
    fmt.Fprintf(e.stderr, "  3. git add <resolved-files>\n")
    fmt.Fprintf(e.stderr, "  4. git commit\n")
    fmt.Fprintf(e.stderr, "  5. autospec dag run %s  # resume\n", e.dagFile)

    return fmt.Errorf("manual conflict resolution required")
}
```

### 10. Parallel Execution Within Layers

Specs within the same layer can still run in parallel (they share the same base):

```go
func (e *ParallelExecutor) ExecuteLayer(ctx context.Context, layer Layer, baseBranch string) error {
    // All specs in this layer branch from the same base
    // So they can run in parallel without conflicts

    var wg sync.WaitGroup
    errChan := make(chan error, len(layer.Features))

    for _, feature := range layer.Features {
        // Check dependencies within layer (some specs may depend on others in same layer)
        if !e.canStart(feature) {
            continue // Will be picked up in next wave
        }

        wg.Add(1)
        go func(f Feature) {
            defer wg.Done()
            if err := e.executeSpec(ctx, f, baseBranch); err != nil {
                errChan <- err
            }
        }(feature)
    }

    wg.Wait()
    close(errChan)

    // Collect errors
    var errs []error
    for err := range errChan {
        errs = append(errs, err)
    }

    if len(errs) > 0 {
        return fmt.Errorf("layer %s had %d failures", layer.ID, len(errs))
    }

    return nil
}
```

## Migration Path

For existing DAGs with worktrees already created from main:

1. **Detection**: Check if worktrees were created from main vs staging
2. **Warning**: Print warning about potential merge conflicts
3. **Recovery**: Offer `--rebase-worktrees` to rebase existing worktrees onto proper staging branch

```bash
autospec dag run workflow.yaml --rebase-worktrees
```

## CLI Changes

**New flags for `dag run`:**
```bash
# Enable/disable automerge for this run (overrides config)
autospec dag run workflow.yaml --automerge       # Force enable
autospec dag run workflow.yaml --no-automerge    # Force disable

# Disable layer staging entirely (legacy behavior, not recommended)
autospec dag run workflow.yaml --no-layer-staging

# Rebase existing worktrees onto correct staging branches
autospec dag run workflow.yaml --rebase-worktrees
```

**New flags for `dag status`:**
```bash
# Show staging branch status
autospec dag status workflow.yaml --staging
```

**New flags for `dag merge`:**
```bash
# Already has --reset from spec-10
# When layer staging is enabled, merges final staging branch only
autospec dag merge workflow.yaml
```

**New output during run (with automerge=true, default):**
```bash
$ autospec dag run workflow.yaml

=== Layer L0 ===
Creating staging branch: dag/mydag/stage-L0 from main
[200-repo-reader] Creating worktree from main...
[201-cli-setup] Creating worktree from main...
...
[200-repo-reader] ✓ Completed
[200-repo-reader] Verifying commit...
[200-repo-reader] ✓ Commit verified (abc1234)
[200-repo-reader] Merging into dag/mydag/stage-L0...   ← IMMEDIATE MERGE
[200-repo-reader] ✓ Merged into staging

[201-cli-setup] ✓ Completed
[201-cli-setup] Verifying commit...
[201-cli-setup] ✓ Commit verified (def5678)
[201-cli-setup] Merging into dag/mydag/stage-L0...
[201-cli-setup] ✓ Merged into staging

✓ Layer L0 complete

=== Layer L1 ===
Creating staging branch: dag/mydag/stage-L1 from stage-L0
[202-author-stats] Creating worktree from stage-L0...  ← NOW HAS L0 CODE!
...
```

**Output with automerge=false (batch merge at layer end):**
```bash
$ autospec dag run workflow.yaml --no-automerge

=== Layer L0 ===
[200-repo-reader] ✓ Completed
[201-cli-setup] ✓ Completed

=== Merging Layer L0 into staging ===
Merging 200-repo-reader into dag/mydag/stage-L0...
Merging 201-cli-setup into dag/mydag/stage-L0...
✓ Layer L0 merged into dag/mydag/stage-L0

=== Layer L1 ===
...
```

## NOT Included

- Squash merges into staging (preserve full history)
- Cherry-picking individual commits (merge entire branches)
- Rebase workflow (merge-based for simplicity)
- Cross-layer dependencies within same run (use layer ordering)

## Success Criteria

1. **Layer 1 worktrees contain Layer 0 code** - verify with `ls` after creation
2. **No merge conflicts from missing base code** - staging accumulates changes
3. **Single final merge to main** - clean history
4. **Parallel within layer still works** - same base, no conflicts
5. **Resume works correctly** - staging branches persist across runs
6. **Automerge integrates with autocommit** - validates dependency, merges immediately after commit
7. **Config validation** - `automerge: true` fails if `autocommit: false`

## Run

```bash
autospec run -spti .dev/tasks/dag-orchestration/spec-11-layer-merge-propagation.md
```
