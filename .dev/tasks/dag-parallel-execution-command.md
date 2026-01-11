# DAG Parallel Execution Command

A new `autospec dag` command for orchestrating parallel feature implementation workflows using DAG-defined dependencies and automatic merge handling.

---

## Current State (What's Already Implemented)

> **Spec 065 implemented task-level parallelization, not spec-level orchestration.**

### Rename In Progress: `dag` → `waves`

> **Note:** The current `dag` command is being renamed to `waves` (Option A) by a separate agent.
> This frees up the `dag` command name for multi-spec orchestration.
> The `internal/dag/` package will also be renamed, making `dag` available as a package name.

### Implemented Commands (Task-Level) — Being Renamed to `waves`

| Old Command | New Command | Description |
|-------------|-------------|-------------|
| `autospec dag [spec-name]` | `autospec waves [spec-name]` | Visualize **task** dependency waves within a single spec |
| `autospec dag --stats` | `autospec waves --stats` | Show wave statistics |
| `autospec dag --compact` | `autospec waves --compact` | Compact single-line output |
| `autospec implement --parallel` | *(unchanged)* | Execute tasks concurrently by wave |
| `autospec implement --parallel --max-parallel N` | *(unchanged)* | Limit concurrent sessions |
| `autospec implement --parallel --worktrees` | *(unchanged)* | Worktree isolation per task |
| `autospec implement --parallel --dry-run` | *(unchanged)* | Preview execution plan |

### What 065 Does

- Computes **waves** from task `dependencies` field in `tasks.yaml`
- Runs independent tasks within each wave concurrently
- Supports resume after interruption
- Optional per-task worktree isolation

### What 065 Does NOT Do

- No multi-spec orchestration (running multiple `autospec run` in parallel)
- No spec-level dependency tracking (spec A depends on spec B)
- No unified status across multiple running specs
- No auto-merge of completed specs to main
- No DAG YAML file format for defining spec workflows

---

## Remaining Work: Spec-Level Orchestration

The rest of this document describes **spec-level** orchestration - running multiple complete `autospec run -spti` workflows in parallel across worktrees.

---

## Problem Statement

Complex projects often have multiple features that can be developed in parallel across different git worktrees. The current workflow requires:

1. Manually creating worktrees for each feature
2. Opening multiple terminals to run `autospec all` concurrently
3. Tracking which features depend on which
4. Manually merging completed features back to main
5. No visibility into overall progress

Example of current manual workflow:
```bash
# Terminal 1                          # Terminal 2
autospec all "1.1 Makefile..."        autospec all "1.2 Debug Overlays..."

# Wait for both, then...
# Terminal 1                          # Terminal 2                    # Terminal 3
autospec all "1.3 Zoom Engine..."     autospec all "1.6 Portal..."    autospec all "1.7 Content..."
```

This is error-prone, hard to track, and doesn't handle dependencies or merging.

---

## Proposed Solution

### Command Structure

```bash
autospec dag <subcommand> [options]
```

| Subcommand | Description |
|------------|-------------|
| `run <file>` | Execute a DAG workflow |
| `status [run-id]` | Show execution status |
| `resume <run-id>` | Resume paused/failed run |
| `validate <file>` | Validate DAG structure |
| `visualize <file>` | Generate DOT/mermaid diagram |
| `init` | Create starter DAG from existing specs |

### Usage Examples

```bash
# Run a DAG
autospec dag run .autospec/dags/phase-1.yaml
# [DAG] Starting 'Phase 1 - Core Features' with 8 features across 3 layers
# [DAG] Creating worktrees...
# [DAG] Layer 'Foundation': Running 2 features in parallel
# [1.1] Starting 'Makefile improvements'...
# [1.2] Starting 'Debug overlays'...

# Check status
autospec dag status
# Run: dag-20250115-143022 (Phase 1 - Core Features)
# Status: running
# Progress: 3/8 features complete, 2 running, 3 pending
# Current: [1.3] Zoom Engine, [1.6] Portal (parallel)

# Resume after failure
autospec dag resume dag-20250115-143022

# Validate DAG structure
autospec dag validate .autospec/dags/phase-1.yaml
# ✓ DAG is valid: 8 features, 3 layers, no cycles
# Dependencies: 1.4 → 1.3, L1 → L0, L2 → L1

# Generate visualization
autospec dag visualize .autospec/dags/phase-1.yaml --format mermaid
```

---

## DAG YAML Schema

```yaml
# .autospec/dags/v1-features.yaml
schema_version: "1.0"

dag:
  name: "V1 Feature Set"
  description: "Parallel implementation of all v1 features"

git:
  base_branch: "main"
  worktree_prefix: "wt-"
  worktree_base_dir: "../"

execution:
  max_parallel: 4              # Limit concurrent autospec processes
  timeout_per_feature: "2h"    # Timeout for single feature
  retry_failed: true           # Auto-retry failed features
  max_retries: 2
  on_feature_failure: "continue"  # continue | pause | abort

merge:
  strategy: "sequential"       # sequential | octopus | manual
  run_tests_before_merge: true
  test_command: "make test"
  on_conflict: "pause"         # pause | skip | abort
  cleanup_after_merge: false   # Remove worktrees after merge

notifications:
  on_layer_complete: true
  on_dag_complete: true
  on_failure: true

layers:
  - id: "L0"
    name: "Foundation"
    features:
      - id: "1.1"
        name: "Makefile Improvements"
        description: |
          Add parallel build targets, improve clean target,
          add development convenience targets.
        command: "autospec all"  # Default command
        args: []                 # Additional arguments
        timeout: "30m"           # Override default timeout

      - id: "1.2"
        name: "Debug Overlays"
        description: "Add visual debug overlay system"

  - id: "L1"
    name: "Core Engine"
    depends_on: ["L0"]  # Layer depends on Foundation completing
    features:
      - id: "1.3"
        name: "Zoom Engine"
        description: "Implement smooth zoom with gestures"

      - id: "1.6"
        name: "Portal System"
        description: "Add portal rendering"

      - id: "1.7"
        name: "Content Loader"
        description: "Lazy content loading system"

  - id: "L1.5"
    name: "Dependent Features"
    depends_on: ["L1"]
    features:
      - id: "1.4"
        name: "Touch Gestures"
        depends_on: ["1.3"]  # Also depends on specific feature
        description: "Touch input for zoom"

      - id: "1.5"
        name: "Image Layers"
        depends_on: ["1.6", "1.7"]  # Multiple feature deps
        description: "Layered image rendering"

  - id: "L2"
    name: "Integration"
    depends_on: ["L1.5"]
    features:
      - id: "1.8"
        name: "Basic UX"
        description: "User experience polish"
```

---

## Cycle Detection & Topological Sort

The DAG must be validated before execution to ensure:
1. No circular dependencies exist
2. All referenced dependencies are valid
3. Execution order can be determined

### Algorithm: Kahn's Algorithm (BFS-based)

Kahn's algorithm detects cycles and produces a topological ordering in O(V + E) time:

```go
// internal/dag/graph.go
package dag

import "fmt"

type Graph struct {
    nodes    map[string]*Node
    edges    map[string][]string  // node -> dependencies
    inDegree map[string]int       // incoming edge count
}

type Node struct {
    ID       string
    Type     string  // "layer" or "feature"
    LayerID  string  // parent layer for features
}

// ValidateAndSort validates the DAG has no cycles and returns topological order.
// Returns error if cycle detected, listing the nodes involved in the cycle.
func (g *Graph) ValidateAndSort() ([]string, error) {
    // Initialize in-degree for all nodes
    inDegree := make(map[string]int)
    for id := range g.nodes {
        inDegree[id] = 0
    }
    for _, deps := range g.edges {
        for _, dep := range deps {
            inDegree[dep]++
        }
    }

    // Queue all nodes with no incoming edges
    queue := make([]string, 0)
    for id, degree := range inDegree {
        if degree == 0 {
            queue = append(queue, id)
        }
    }

    // Process nodes in topological order
    order := make([]string, 0, len(g.nodes))
    for len(queue) > 0 {
        // Dequeue
        node := queue[0]
        queue = queue[1:]
        order = append(order, node)

        // Reduce in-degree for all dependents
        for _, dep := range g.edges[node] {
            inDegree[dep]--
            if inDegree[dep] == 0 {
                queue = append(queue, dep)
            }
        }
    }

    // If not all nodes processed, cycle exists
    if len(order) != len(g.nodes) {
        cycleNodes := g.findCycleNodes(inDegree)
        return nil, fmt.Errorf("cycle detected involving: %v", cycleNodes)
    }

    return order, nil
}

// findCycleNodes identifies nodes that are part of the cycle
func (g *Graph) findCycleNodes(inDegree map[string]int) []string {
    cycle := make([]string, 0)
    for id, degree := range inDegree {
        if degree > 0 {
            cycle = append(cycle, id)
        }
    }
    return cycle
}
```

### Building the Dependency Graph

Dependencies come from two sources:
1. **Layer dependencies** (`depends_on` at layer level)
2. **Feature dependencies** (`depends_on` at feature level)

```go
// BuildGraph constructs the dependency graph from a DAG definition
func BuildGraph(dag *DAG) (*Graph, error) {
    g := &Graph{
        nodes:    make(map[string]*Node),
        edges:    make(map[string][]string),
        inDegree: make(map[string]int),
    }

    // Add all layers and features as nodes
    for _, layer := range dag.Layers {
        g.nodes[layer.ID] = &Node{ID: layer.ID, Type: "layer"}

        for _, feature := range layer.Features {
            g.nodes[feature.ID] = &Node{
                ID:      feature.ID,
                Type:    "feature",
                LayerID: layer.ID,
            }
        }
    }

    // Add edges from dependencies
    for _, layer := range dag.Layers {
        // Layer-level dependencies: all features in layer depend on all features in dep layers
        for _, depLayerID := range layer.DependsOn {
            depLayer := dag.findLayer(depLayerID)
            if depLayer == nil {
                return nil, fmt.Errorf("layer %s depends on unknown layer %s", layer.ID, depLayerID)
            }
            for _, feature := range layer.Features {
                for _, depFeature := range depLayer.Features {
                    g.edges[depFeature.ID] = append(g.edges[depFeature.ID], feature.ID)
                }
            }
        }

        // Feature-level dependencies
        for _, feature := range layer.Features {
            for _, depID := range feature.DependsOn {
                if _, exists := g.nodes[depID]; !exists {
                    return nil, fmt.Errorf("feature %s depends on unknown feature %s", feature.ID, depID)
                }
                g.edges[depID] = append(g.edges[depID], feature.ID)
            }
        }
    }

    return g, nil
}
```

### Validation Output

```bash
$ autospec dag validate .autospec/dags/phase-1.yaml
✓ DAG 'Phase 1 - Core Features' is valid
  - 8 features across 4 layers
  - No cycles detected
  - Execution order: 1.1, 1.2 → 1.3, 1.6, 1.7 → 1.4, 1.5 → 1.8

$ autospec dag validate .autospec/dags/bad.yaml
✗ DAG validation failed: cycle detected involving: [1.3, 1.4, 1.5]
  - 1.3 depends on 1.5
  - 1.4 depends on 1.3
  - 1.5 depends on 1.4
```

---

## Execution Flow

### 1. Initialization Phase

```
┌─────────────────────────────────────────────────────────────┐
│ 1. Parse DAG YAML                                           │
│ 2. Build dependency graph                                   │
│ 3. Run cycle detection (Kahn's algorithm)                   │
│ 4. Validate all dependencies reference valid nodes          │
│ 5. Create worktrees for all features (via autospec worktree)│
│ 6. Initialize run state                                     │
└─────────────────────────────────────────────────────────────┘
```

### 2. Execution Phase

```
┌─────────────────────────────────────────────────────────────┐
│ For each layer (in dependency order):                       │
│   1. Wait for layer dependencies to complete                │
│   2. Identify runnable features (deps satisfied)            │
│   3. Spawn processes up to max_parallel limit               │
│   4. Stream output with feature prefixes                    │
│   5. Track completion, update state                         │
│   6. On failure: retry or handle per on_feature_failure     │
└─────────────────────────────────────────────────────────────┘
```

### 3. Merge Phase

```
┌─────────────────────────────────────────────────────────────┐
│ After each feature completes (if merge.strategy=sequential):│
│   1. Run tests in worktree (if configured)                  │
│   2. Checkout base branch, pull latest                      │
│   3. Merge feature branch                                   │
│   4. Push to origin                                         │
│   5. On conflict: handle per on_conflict setting            │
└─────────────────────────────────────────────────────────────┘
```

### 4. Cleanup Phase

```
┌─────────────────────────────────────────────────────────────┐
│ 1. Generate summary report                                  │
│ 2. If cleanup_after_merge=true, remove merged worktrees     │
│ 3. Send completion notification                             │
│ 4. Update final run state                                   │
└─────────────────────────────────────────────────────────────┘
```

---

## Terminal Output

Simple streaming output with feature prefixes:

```
$ autospec dag run phase-1.yaml
[DAG] Starting 'Phase 1 - Core Features' with 8 features across 3 layers
[DAG] Creating worktrees...
[DAG] Layer 'Foundation': Running 2 features in parallel

[1.1] Starting 'Makefile improvements'...
[1.2] Starting 'Debug overlays'...
[1.1] Specifying feature...
[1.2] Specifying feature...
[1.1] Planning implementation...
[1.2] Planning implementation...
[1.1] Generating tasks...
[1.1] Implementing (8 tasks)...
[1.2] Generating tasks...
[1.1] ✓ Completed in 5m32s
[DAG] Merging 1.1 to main...
[1.2] Implementing (6 tasks)...
[1.2] ✓ Completed in 4m18s
[DAG] Merging 1.2 to main...

[DAG] Layer 'Core Engine': Running 3 features in parallel
[1.3] Starting 'Zoom Engine'...
[1.6] Starting 'Portal System'...
[1.7] Starting 'Content Loader'...
...

[DAG] ✓ All 8 features completed successfully
[DAG] Total time: 47m23s
```

### Log Files

Each feature gets its own log file:
```
.autospec/logs/
└── dag-runs/
    └── dag-20250115-143022/
        ├── 1.1-makefile.log
        ├── 1.2-debug.log
        ├── 1.3-zoom.log
        └── ...
```

---

## State Management

### Run State File

State stored in `.autospec/state/dag-runs/<run-id>.yaml`:

```yaml
run_id: "dag-20250115-143022"
dag_file: ".autospec/dags/phase-1.yaml"
dag_name: "Phase 1 - Core Features"
started_at: "2025-01-15T14:30:22Z"
status: "running"
current_layer: "L1"

features:
  "1.1":
    status: "completed"
    worktree: "../wt-1.1"
    started_at: "2025-01-15T14:30:25Z"
    completed_at: "2025-01-15T14:35:57Z"
    duration: "5m32s"
    merged: true
    merge_commit: "abc123"

  "1.3":
    status: "running"
    worktree: "../wt-1.3"
    started_at: "2025-01-15T14:36:00Z"
    pid: 12345
    current_stage: "implement"
    current_task: 3

  "1.4":
    status: "pending"
    blocked_by:
      - "1.3"

layers_completed:
  - "L0"

errors: []
```

### Status Values

| Status | Description |
|--------|-------------|
| `pending` | Not yet started, may be blocked |
| `running` | Currently executing |
| `completed` | Successfully finished |
| `failed` | Failed after retries exhausted |
| `skipped` | Skipped due to dependency failure |
| `paused` | Waiting for user intervention |

---

## Package Architecture

```
internal/
└── dag/
    ├── parser.go       # YAML parsing and validation
    ├── graph.go        # DAG construction, cycle detection, topo sort
    ├── executor.go     # Parallel execution engine
    ├── merger.go       # Git merge orchestration
    ├── state.go        # Run state persistence
    ├── output.go       # Prefixed log streaming
    └── types.go        # Core type definitions

internal/cli/orchestration/
└── dag.go              # CLI command implementation
```

### Key Types

```go
// internal/dag/types.go
package dag

type DAG struct {
    SchemaVersion string      `yaml:"schema_version"`
    Config        DAGConfig   `yaml:"dag"`
    Git           GitConfig   `yaml:"git"`
    Execution     ExecConfig  `yaml:"execution"`
    Merge         MergeConfig `yaml:"merge"`
    Notifications NotifyConfig `yaml:"notifications"`
    Layers        []Layer     `yaml:"layers"`
}

type Layer struct {
    ID        string    `yaml:"id"`
    Name      string    `yaml:"name"`
    DependsOn []string  `yaml:"depends_on,omitempty"`
    Features  []Feature `yaml:"features"`
}

type Feature struct {
    ID          string   `yaml:"id"`
    Name        string   `yaml:"name"`
    Description string   `yaml:"description"`
    DependsOn   []string `yaml:"depends_on,omitempty"`
    Command     string   `yaml:"command,omitempty"`
    Args        []string `yaml:"args,omitempty"`
    Timeout     string   `yaml:"timeout,omitempty"`
}

type RunState struct {
    RunID           string                  `yaml:"run_id"`
    DAGFile         string                  `yaml:"dag_file"`
    DAGName         string                  `yaml:"dag_name"`
    StartedAt       time.Time               `yaml:"started_at"`
    Status          RunStatus               `yaml:"status"`
    CurrentLayer    string                  `yaml:"current_layer"`
    Features        map[string]FeatureState `yaml:"features"`
    LayersCompleted []string                `yaml:"layers_completed"`
    Errors          []RunError              `yaml:"errors"`
}

type FeatureState struct {
    Status      FeatureStatus `yaml:"status"`
    Worktree    string        `yaml:"worktree"`
    StartedAt   time.Time     `yaml:"started_at,omitempty"`
    CompletedAt time.Time     `yaml:"completed_at,omitempty"`
    Duration    string        `yaml:"duration,omitempty"`
    Merged      bool          `yaml:"merged"`
    MergeCommit string        `yaml:"merge_commit,omitempty"`
    PID         int           `yaml:"pid,omitempty"`
    BlockedBy   []string      `yaml:"blocked_by,omitempty"`
    Error       string        `yaml:"error,omitempty"`
    Retries     int           `yaml:"retries"`
}
```

### Executor Interface

```go
// internal/dag/executor.go
package dag

type Executor interface {
    // Run executes a DAG from a file
    Run(dagFile string, opts RunOptions) (*RunResult, error)

    // Resume continues a paused/failed run
    Resume(runID string) (*RunResult, error)

    // Status returns the current status of a run
    Status(runID string) (*RunState, error)

    // Cancel stops a running DAG
    Cancel(runID string) error

    // List returns all run states
    List() ([]RunState, error)
}

type RunOptions struct {
    DryRun         bool   // Validate and show plan, don't execute
    MaxParallel    int    // Override max_parallel from DAG
    SkipMerge      bool   // Don't merge completed features
    ContinueOnFail bool   // Continue even if features fail
}

type RunResult struct {
    RunID       string
    Status      RunStatus
    Completed   int
    Failed      int
    Skipped     int
    TotalTime   time.Duration
    MergeErrors []string
}
```

---

## Merge Strategies

### Sequential (Default)

Merge each feature immediately after completion:

```go
func (m *Merger) mergeSequential(feature *Feature, state *FeatureState) error {
    // 1. Run tests in worktree
    if m.cfg.RunTestsBeforeMerge {
        if err := m.runTests(state.Worktree); err != nil {
            return fmt.Errorf("tests failed: %w", err)
        }
    }

    // 2. Checkout base and pull
    if err := m.git.Checkout(m.cfg.BaseBranch); err != nil {
        return err
    }
    if err := m.git.Pull(); err != nil {
        return err
    }

    // 3. Merge feature branch
    branch := fmt.Sprintf("feat/%s", feature.ID)
    if err := m.git.Merge(branch); err != nil {
        if isConflict(err) {
            return &MergeConflictError{Feature: feature.ID, Files: conflictFiles(err)}
        }
        return err
    }

    // 4. Push
    return m.git.Push()
}
```

### Octopus (Layer-based)

Merge all features in a layer at once:

```go
func (m *Merger) mergeOctopus(layer *Layer, states map[string]*FeatureState) error {
    branches := make([]string, 0, len(layer.Features))
    for _, f := range layer.Features {
        branches = append(branches, fmt.Sprintf("feat/%s", f.ID))
    }

    // git merge -s octopus branch1 branch2 branch3...
    return m.git.MergeOctopus(branches...)
}
```

### Manual (No Auto-merge)

Leave features on branches for manual review/merge:

```go
func (m *Merger) mergeManual(feature *Feature) error {
    // Just log - no automatic merge
    m.logger.Info("Feature ready for manual merge",
        "feature", feature.ID,
        "branch", fmt.Sprintf("feat/%s", feature.ID))
    return nil
}
```

---

## Error Handling & Recovery

### Failure Scenarios

| Scenario | Behavior |
|----------|----------|
| Feature execution fails | Retry up to max_retries, then apply on_feature_failure |
| Merge conflict | Apply on_conflict (pause/skip/abort) |
| Worktree creation fails | Fail fast with error message |
| Process killed (Ctrl-C) | Save state, can resume later |
| System crash | State persisted, `dag resume` restarts from checkpoint |

### Resume Command

```go
func (e *executor) Resume(runID string) (*RunResult, error) {
    // 1. Load run state
    state, err := e.state.Load(runID)
    if err != nil {
        return nil, err
    }

    // 2. Validate state is resumable
    if state.Status == RunStatusCompleted {
        return nil, fmt.Errorf("run %s already completed", runID)
    }

    // 3. Check for stale processes
    for id, fs := range state.Features {
        if fs.Status == FeatureStatusRunning {
            if !processExists(fs.PID) {
                // Process died, mark as failed for retry
                fs.Status = FeatureStatusFailed
                fs.Error = "process terminated unexpectedly"
            }
        }
    }

    // 4. Continue execution from current state
    return e.continueExecution(state)
}
```

### Conflict Resolution Hook

```yaml
merge:
  on_conflict: "pause"
  conflict_hook: ".autospec/scripts/on-merge-conflict.sh"
```

Hook receives environment variables:
```bash
AUTOSPEC_DAG_RUN_ID="dag-20250115-143022"
AUTOSPEC_FEATURE_ID="1.3"
AUTOSPEC_FEATURE_NAME="Zoom Engine"
AUTOSPEC_BRANCH="feat/1.3"
AUTOSPEC_CONFLICT_FILES="/path/to/file1.go,/path/to/file2.go"
```

---

## Dependency on Worktree Command

The DAG executor uses `autospec worktree` for worktree management:

```go
func (e *executor) initializeWorktrees(dag *DAG) error {
    wtMgr := worktree.NewManager(e.cfg.WorktreeConfig)

    for _, layer := range dag.Layers {
        for _, feature := range layer.Features {
            name := fmt.Sprintf("%s%s", dag.Git.WorktreePrefix, feature.ID)
            branch := fmt.Sprintf("feat/%s", feature.ID)

            wt, err := wtMgr.Create(name, branch, worktree.CreateOptions{})
            if err != nil {
                return fmt.Errorf("creating worktree for %s: %w", feature.ID, err)
            }

            e.worktrees[feature.ID] = wt
        }
    }

    return nil
}
```

---

## Testing Strategy

### Unit Tests

```go
// internal/dag/graph_test.go
func TestGraph_DetectCycles(t *testing.T) {
    t.Parallel()

    tests := map[string]struct {
        edges   map[string][]string
        wantErr bool
    }{
        "no cycles": {
            edges: map[string][]string{
                "A": {"B"},
                "B": {"C"},
                "C": {},
            },
            wantErr: false,
        },
        "simple cycle": {
            edges: map[string][]string{
                "A": {"B"},
                "B": {"C"},
                "C": {"A"},
            },
            wantErr: true,
        },
        "self-reference": {
            edges: map[string][]string{
                "A": {"A"},
            },
            wantErr: true,
        },
    }

    for name, tt := range tests {
        t.Run(name, func(t *testing.T) {
            t.Parallel()
            g := dag.NewGraph(tt.edges)
            err := g.DetectCycles()
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}

// internal/dag/executor_test.go
func TestExecutor_Run(t *testing.T) {
    t.Parallel()

    tests := map[string]struct {
        dag     *DAG
        opts    RunOptions
        wantErr bool
    }{
        "simple sequential": {
            dag: &DAG{
                Layers: []Layer{
                    {ID: "L0", Features: []Feature{{ID: "1.1"}}},
                },
            },
        },
        "parallel features": {
            dag: &DAG{
                Execution: ExecConfig{MaxParallel: 2},
                Layers: []Layer{
                    {ID: "L0", Features: []Feature{{ID: "1.1"}, {ID: "1.2"}}},
                },
            },
        },
    }

    for name, tt := range tests {
        t.Run(name, func(t *testing.T) {
            t.Parallel()
            // Mock command execution
            exec := dag.NewExecutor(dag.ExecutorConfig{
                CommandRunner: &mockRunner{},
            })
            result, err := exec.Run(tt.dag, tt.opts)
            // ... assertions
        })
    }
}
```

### Integration Tests

```go
func TestDAGIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    // Create temp repo with test DAG
    repoDir := t.TempDir()
    initGitRepo(t, repoDir)
    writeDagFile(t, repoDir, testDAG)

    // Run DAG
    exec := dag.NewExecutor(dag.ExecutorConfig{
        RepoDir:   repoDir,
        StateDir:  filepath.Join(repoDir, ".autospec/state"),
        LogDir:    filepath.Join(repoDir, ".autospec/logs"),
    })

    result, err := exec.Run(".autospec/dags/test.yaml", dag.RunOptions{})
    require.NoError(t, err)
    assert.Equal(t, dag.RunStatusCompleted, result.Status)
}
```

---

## Proposed Specs (Spec-Level Orchestration)

Split into 4 well-scoped specs. Each uses `autospec run -spti "description"`.

---

### Spec 1: DAG Schema & Validation

**Scope:** YAML schema definition, parsing, validation, and visualization for multi-spec workflows.

**Commands:**
- `autospec dag validate <file>` - Validate DAG structure, detect cycles
- `autospec dag visualize <file>` - Generate ASCII/mermaid diagram of spec dependencies

**Key Deliverables:**
- Define `dag.yaml` schema (layers, features, depends_on)
- Parser in `internal/dag/spec_dag.go` (separate from task-level dag)
- Cycle detection for spec-level dependencies
- Schema validation with clear error messages

**Run:**
```bash
autospec run -spti "Add DAG schema and validation for multi-spec workflows. Define YAML schema for .autospec/dags/*.yaml files with layers containing features (specs). Each feature has id, name, description, depends_on. Add autospec dag validate command to check for cycles and invalid references. Add autospec dag visualize command for ASCII output. Reuse patterns from internal/dag/ but keep spec-level separate from task-level."
```

---

### Spec 2: DAG Run Sequential

**Scope:** Core execution engine - create worktrees, run specs sequentially, track state.

**Commands:**
- `autospec dag run <file>` - Execute a DAG workflow (sequential mode)
- `autospec dag list` - List all DAG runs

**Key Deliverables:**
- Worktree creation per spec (uses existing worktree.Manager)
- Sequential spec execution (one at a time, respecting dependencies)
- Run state persistence in `.autospec/state/dag-runs/<run-id>.yaml`
- Terminal output with `[feature-id]` prefixes
- Log files per feature in `.autospec/logs/dag-runs/`

**Run:**
```bash
autospec run -spti "Add autospec dag run command for sequential multi-spec execution. Parse dag.yaml, create worktrees for each feature using worktree.Manager, execute autospec run -spti in each worktree sequentially respecting layer dependencies. Track run state in .autospec/state/dag-runs/. Prefix output with feature ID. Create per-feature log files. Add autospec dag list to show all runs."
```

---

### Spec 3: DAG Parallel & Status

**Scope:** Parallel execution with concurrency control and unified status view.

**Commands:**
- `autospec dag run <file> --parallel` - Execute with parallelization
- `autospec dag run --max-parallel N` - Limit concurrent specs
- `autospec dag status [run-id]` - Show unified status across specs

**Key Deliverables:**
- Parallel process management with semaphore (max_parallel from config)
- Output multiplexing with feature prefixes
- Progress tracking (X/Y features complete)
- Graceful shutdown on Ctrl-C (save state)
- `autospec dag status` showing all running/completed/failed specs

**Run:**
```bash
autospec run -spti "Add parallel execution to autospec dag run. Add --parallel and --max-parallel flags. Use semaphore to limit concurrent autospec processes. Multiplex output with feature ID prefixes. Track progress and update state. Handle Ctrl-C gracefully by saving state. Add autospec dag status command showing unified view: completed, running, pending, failed specs with progress percentages."
```

---

### Spec 4: DAG Resume & Merge

**Scope:** Resume failed runs and auto-merge completed specs.

**Commands:**
- `autospec dag resume <run-id>` - Resume paused/failed run
- `autospec dag merge <run-id>` - Merge all completed specs

**Key Deliverables:**
- Resume from checkpoint (skip completed features)
- Detect stale processes (PID no longer exists)
- Sequential merge strategy (merge each completed feature to base branch)
- Pre-merge test execution (optional, configurable)
- Conflict detection with pause for user resolution
- Cleanup worktrees after successful merge (optional)

**Run:**
```bash
autospec run -spti "Add autospec dag resume to continue failed/interrupted DAG runs. Load run state, skip completed features, retry failed ones. Add autospec dag merge to merge completed features to base branch. Run tests before merge if configured. Detect conflicts and pause for user resolution. Add merge.strategy config (sequential/manual). Optionally cleanup worktrees after merge."
```

---

### Spec 5: Watch & Monitor Commands

**Scope:** Real-time monitoring of single spec and multi-worktree dashboard.

**Commands:**
- `autospec watch [spec-name]` - Real-time monitoring of current spec execution
- `autospec watch --all` - Monitor all active worktrees in table view

**Key Deliverables:**
- Stream task progress with timestamps
- Show tool calls (Edit, Bash, etc.) as they happen
- Table view for multi-worktree monitoring
- Auto-refresh with configurable interval

**Run:**
```bash
autospec run -spti "Add autospec watch command for real-time spec monitoring. Show timestamped progress: phase/task numbers, tool calls (Edit, Bash, Read), test output. Add --all flag for multi-worktree table view showing: worktree name, status (specify/plan/impl), progress (X/Y tasks), last update time. Use terminal table formatting. Auto-refresh every 2s by default, configurable with --interval."
```

---

### Spec 6: Worktree Copy Files Config

**Scope:** Auto-copy essential files to new worktrees.

**Commands:**
- Enhancement to `autospec worktree create`

**Key Deliverables:**
- `worktree.copy_files` config option
- Copy untracked but essential files (.autospec/, .claude/, .opencode/, etc.)
- Support glob patterns
- Run before setup script

**Run:**
```bash
autospec run -spti "Add worktree.copy_files config to auto-copy essential files when creating worktrees. Config is list of paths/globs: ['.autospec/', '.claude/', '.opencode/', 'opencode.json']. Copy these from source repo to worktree before running setup script. Handle missing files gracefully. Add --skip-copy flag to bypass."
```

---

### Spec 7: DAG Retry Command

**Scope:** Smart retry for failed specs with cleanup options.

**Commands:**
- `autospec dag retry <spec-id> [--clean]` - Retry failed spec

**Key Deliverables:**
- Identify failure stage (specify/plan/tasks/implement)
- `--clean` removes old artifacts and restarts from scratch
- Without `--clean`, resume from failure point
- Update DAG run state

**Run:**
```bash
autospec run -spti "Add autospec dag retry command to retry failed specs in a DAG run. Detect failure stage from run state. With --clean flag: remove spec artifacts, delete worktree, recreate and restart from specify. Without --clean: resume from failure point using existing --resume logic. Update DAG run state after retry. Support retrying multiple specs: autospec dag retry spec1 spec2."
```

---

### Spec 8: Fix Tests Command (Future)

**Scope:** AI-assisted test failure fixing.

**Commands:**
- `autospec fix-tests` - Analyze and fix failing tests

**Key Deliverables:**
- Run test command and capture failures
- Analyze failure output with AI
- Generate fixes for common patterns
- Interactive mode for complex failures

**Run:**
```bash
autospec run -spti "Add autospec fix-tests command for AI-assisted test fixing. Run configured test command, capture failures. Send failure output to AI agent with context (failing test file, related source files). Generate and apply fixes. Support --dry-run to show proposed fixes without applying. Limit to N fix attempts (default 3). Track which tests were fixed."
```

---

## Configuration in autospec config

```yaml
# .autospec/config.yml
dag:
  default_max_parallel: 4
  default_timeout: "2h"
  state_dir: ".autospec/state/dag-runs"
  log_dir: ".autospec/logs/dag-runs"
  cleanup_old_runs: true
  max_run_history: 10
  merge:
    strategy: "sequential"  # sequential | manual
    run_tests_before_merge: true
    test_command: "make test"
    cleanup_after_merge: false

worktree:
  copy_files:              # Files to copy to new worktrees
    - ".autospec/"
    - ".claude/"
    - ".opencode/"
    - "opencode.json"
  copy_on_create: true     # Enable auto-copy (default: true)
```

---

## Relationship to Worktree Command

```
┌─────────────────────────────────────────────────────────────────────┐
│                     autospec dag run                                │
│                           │                                         │
│                    ┌──────┴──────┐                                  │
│                    ▼             ▼                                  │
│          ┌─────────────┐  ┌─────────────┐                          │
│          │  dag/parser │  │ dag/graph   │                          │
│          └─────────────┘  └─────────────┘                          │
│                    │                                                │
│                    ▼                                                │
│          ┌─────────────────────────────┐                           │
│          │      dag/executor           │                           │
│          │  (parallel process mgmt)    │                           │
│          └─────────────────────────────┘                           │
│                    │                                                │
│        ┌───────────┼───────────┐                                   │
│        ▼           ▼           ▼                                   │
│   ┌─────────┐ ┌─────────┐ ┌─────────┐                             │
│   │worktree │ │autospec │ │  git    │                             │
│   │ manager │ │   all   │ │ merge   │                             │
│   └─────────┘ └─────────┘ └─────────┘                             │
│        │           │           │                                   │
│        ▼           ▼           ▼                                   │
│   ┌─────────────────────────────────┐                              │
│   │           Git Repository         │                              │
│   │  (main + feature worktrees)     │                              │
│   └─────────────────────────────────┘                              │
└─────────────────────────────────────────────────────────────────────┘
```

This creates a "meta-orchestrator" that orchestrates multiple autospec orchestrators, each running in its own worktree with independent Claude sessions.

---

## Summary: Implemented vs Planned

| Feature | Status | Spec |
|---------|--------|------|
| **Task-Level Parallelization** | | |
| `autospec dag [spec]` - visualize task waves | ✅ Implemented | 065 |
| `autospec implement --parallel` | ✅ Implemented | 065 |
| `--max-parallel`, `--worktrees`, `--dry-run` | ✅ Implemented | 065 |
| Task-level resume after interrupt | ✅ Implemented | 065 |
| **Spec-Level Orchestration** | | |
| DAG YAML schema for multi-spec workflows | ⬜ Planned | Spec 1 |
| `autospec dag validate <file>` | ⬜ Planned | Spec 1 |
| `autospec dag visualize <file>` | ⬜ Planned | Spec 1 |
| `autospec dag run <file>` (sequential) | ⬜ Planned | Spec 2 |
| `autospec dag list` | ⬜ Planned | Spec 2 |
| `autospec dag run --parallel` | ⬜ Planned | Spec 3 |
| `autospec dag status` | ⬜ Planned | Spec 3 |
| `autospec dag resume` | ⬜ Planned | Spec 4 |
| `autospec dag merge` | ⬜ Planned | Spec 4 |
| **Monitoring & UX** | | |
| `autospec watch` | ⬜ Planned | Spec 5 |
| `autospec watch --all` | ⬜ Planned | Spec 5 |
| **Worktree Enhancements** | | |
| `worktree.copy_files` config | ⬜ Planned | Spec 6 |
| **Recovery** | | |
| `autospec dag retry` | ⬜ Planned | Spec 7 |
| `autospec fix-tests` | ⬜ Future | Spec 8 |

### Recommended Order

1. **Spec 6** (worktree.copy_files) - Quick win, improves worktree UX
2. **Spec 1** (DAG Schema) - Foundation for spec-level orchestration
3. **Spec 2** (DAG Run Sequential) - Core execution, no parallelization complexity
4. **Spec 3** (DAG Parallel & Status) - Add parallelization + status
5. **Spec 4** (DAG Resume & Merge) - Recovery and merge automation
6. **Spec 5** (Watch) - Nice-to-have monitoring
7. **Spec 7** (DAG Retry) - Advanced recovery
8. **Spec 8** (Fix Tests) - Future enhancement
