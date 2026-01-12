# Spec 12: Inline State in DAG File

## Context

Part of **DAG Multi-Spec Orchestration** - a meta-orchestrator that runs multiple `autospec run` workflows in parallel across worktrees with dependency management. See [00-summary.md](00-summary.md) for full vision.

## Problem

**State is stored separately from the DAG definition, creating inconsistency and complexity.**

Current behavior:
```
.autospec/dags/gitstats.yaml          # DAG definition
.autospec/state/dag-runs/
  └── 20260111_112034_ba7166b1.yaml   # State file (run_id generated)
```

**Issues with this design:**

1. **Philosophy mismatch**: Docs say "workflow file is the run identifier" but implementation generates a separate `run_id`

2. **Data redundancy**: State file duplicates data already in dag.yaml:
   - `spec_id`, `layer_id`, `blocked_by` - all derivable from DAG definition
   - `workflow_path`, `dag_file`, `dag_id` - just metadata about source file

3. **Discovery problem**: Must map workflow file → run_id → state file

4. **Unnecessary complexity**: Run IDs solve problems that don't exist:
   - Concurrent runs of same DAG? Not a use case
   - Historical run tracking? Rarely needed, could archive separately

5. **Two files to understand**: Users must look at both files to see current state

## Solution: State Section in dag.yaml

Embed runtime state directly in the DAG file with clear separation:

```yaml
schema_version: "1.0"

dag:
  name: "GitStats CLI v1"

execution:
  max_parallel: 4
  timeout: "4h"
  base_branch: "main"

layers:
  - id: "L0"
    features:
      - id: "200-repo-reader"
        description: "Implement repository reader..."
      - id: "201-cli-setup"
        description: "Set up CLI with Cobra..."

  - id: "L1"
    depends_on: ["L0"]
    features:
      - id: "202-author-stats"
        description: "Collect author statistics..."

# ====== RUNTIME STATE (auto-managed, do not edit) ======
run:
  status: running
  started_at: 2026-01-11T11:20:34-08:00
  completed_at: null

specs:
  200-repo-reader:
    status: completed
    worktree: /home/ari/repos/dag-gitstats-200-repo-reader
    started_at: 2026-01-11T11:20:34-08:00
    completed_at: 2026-01-11T12:00:57-08:00
    commit_sha: 9b4bc192
    commit_status: committed
  201-cli-setup:
    status: completed
    worktree: /home/ari/repos/dag-gitstats-201-cli-setup
    started_at: 2026-01-11T12:00:57-08:00
    completed_at: 2026-01-11T12:40:02-08:00
    commit_sha: 60440ac3
    commit_status: committed
  202-author-stats:
    status: running
    worktree: /home/ari/repos/dag-gitstats-202-author-stats
    started_at: 2026-01-11T12:40:02-08:00
    current_stage: implement

staging:
  L0:
    branch: dag/gitstats/stage-L0
    specs_merged: [200-repo-reader, 201-cli-setup]
```

## Key Design Decisions

### 1. Eliminate run_id Entirely

The workflow file path IS the identity. No generated identifiers needed.

**Current**: `.autospec/dags/foo.yaml` → generates `run_id: 20260111_112034_ba7166b1` → creates state at `.autospec/state/dag-runs/20260111_112034_ba7166b1.yaml`

**New**: `.autospec/dags/foo.yaml` contains its own state. Period.

### 2. State Section at Bottom

Clear visual separation with a comment marker. Definition above, state below.

Users can:
- Read the top half to understand the DAG
- Read the bottom half to see current progress
- Delete the bottom sections to reset

### 3. Minimal State Fields

Only store what can't be derived from the definition:

| Field | Stored | Reason |
|-------|--------|--------|
| `spec_id` | No | Already in `layers[].features[].id` |
| `layer_id` | No | Already associated via layer structure |
| `blocked_by` | No | Derivable from `depends_on` |
| `status` | Yes | Runtime state |
| `worktree` | Yes | Runtime path |
| `started_at`, `completed_at` | Yes | Runtime timestamps |
| `commit_sha`, `commit_status` | Yes | Runtime git state |
| `merge` info | Yes | Runtime merge state |
| `failure_reason` | Yes | Runtime error |

### 4. File Modification During Execution

The dag.yaml gets modified as specs progress. This is intentional:

- Single source of truth
- Progress visible by reading the file
- No sync issues between definition and state

**Git considerations:**
- If dag.yaml is tracked, state changes appear in `git status`
- Options: gitignore the dags directory, or accept state is tracked
- `dag cleanup` can strip the state section

## File Changes

### Modified Files

| File | Change |
|------|--------|
| `internal/dag/state.go` | Remove `DAGRun` struct, add inline state types |
| `internal/dag/executor.go` | Write state to dag.yaml instead of separate file |
| `internal/dag/loader.go` | Parse state section when loading |
| `internal/dag/schema.go` | Add `Run`, `Specs`, `Staging` fields to `DAGConfig` |
| `internal/dag/merge.go` | Read state from dag.yaml |
| `internal/cli/dag/*.go` | Update all commands to use inline state |

### Removed Files/Concepts

| Removed | Reason |
|---------|--------|
| `.autospec/state/dag-runs/` directory | No longer needed |
| `run_id` generation | Workflow file is the ID |
| `LoadStateByWorkflow()` lookup | Direct dag.yaml read |
| `SaveStateByWorkflow()` mapping | Direct dag.yaml write |

### New State Schema

```go
// In schema.go - add to DAGConfig
type DAGConfig struct {
    SchemaVersion string           `yaml:"schema_version"`
    DAG           DAGMetadata      `yaml:"dag"`
    Execution     ExecutionConfig  `yaml:"execution"`
    Layers        []Layer          `yaml:"layers"`

    // Runtime state (auto-managed)
    Run     *RunState            `yaml:"run,omitempty"`
    Specs   map[string]SpecState `yaml:"specs,omitempty"`
    Staging map[string]LayerStaging `yaml:"staging,omitempty"`
}

type RunState struct {
    Status      string     `yaml:"status"`
    StartedAt   time.Time  `yaml:"started_at"`
    CompletedAt *time.Time `yaml:"completed_at,omitempty"`
}

type SpecState struct {
    Status        string     `yaml:"status"`
    Worktree      string     `yaml:"worktree,omitempty"`
    StartedAt     *time.Time `yaml:"started_at,omitempty"`
    CompletedAt   *time.Time `yaml:"completed_at,omitempty"`
    CurrentStage  string     `yaml:"current_stage,omitempty"`
    CommitSHA     string     `yaml:"commit_sha,omitempty"`
    CommitStatus  string     `yaml:"commit_status,omitempty"`
    FailureReason string     `yaml:"failure_reason,omitempty"`
    ExitCode      int        `yaml:"exit_code,omitempty"`
    Merge         *MergeState `yaml:"merge,omitempty"`
}

type LayerStaging struct {
    Branch      string   `yaml:"branch"`
    SpecsMerged []string `yaml:"specs_merged,omitempty"`
}
```

## Command Changes

### `dag run`

```bash
# Same usage, state written to dag.yaml
autospec dag run .autospec/dags/gitstats.yaml

# --fresh clears run:, specs:, staging: sections before starting
autospec dag run .autospec/dags/gitstats.yaml --fresh
```

### `dag status`

```bash
# Reads state directly from dag.yaml
autospec dag status .autospec/dags/gitstats.yaml
```

### `dag list`

Without run_ids, `dag list` shows DAG files and their embedded status:

```bash
$ autospec dag list
DAG FILE                          STATUS     PROGRESS  LAST ACTIVITY
.autospec/dags/gitstats.yaml      running    8/10      2026-01-11 16:54
.autospec/dags/refactor.yaml      completed  5/5       2026-01-10 14:22
.autospec/dags/features.yaml      (no state) -         -
```

### `dag cleanup`

```bash
# Removes worktrees AND clears state section from dag.yaml
autospec dag cleanup .autospec/dags/gitstats.yaml

# --keep-state removes worktrees but preserves state for reference
autospec dag cleanup .autospec/dags/gitstats.yaml --keep-state
```

## State Persistence Flow

```
dag run workflow.yaml
    │
    ▼
┌─────────────────────────────────────┐
│ Load dag.yaml (definition + state)  │
└─────────────────┬───────────────────┘
                  │
    ▼ (if --fresh or no state)
┌─────────────────────────────────────┐
│ Initialize run: section             │
│ Initialize specs: map (empty)       │
└─────────────────┬───────────────────┘
                  │
    ▼ (for each spec)
┌─────────────────────────────────────┐
│ Update specs[id].status = running   │
│ Write dag.yaml                      │
└─────────────────┬───────────────────┘
                  │
    ▼ (spec completes)
┌─────────────────────────────────────┐
│ Update specs[id].status = completed │
│ Update specs[id].commit_sha         │
│ Write dag.yaml                      │
└─────────────────────────────────────┘
```

## Alternative Considered: Sibling State File

Instead of embedding state, use a sibling file:
- `gitstats.yaml` (definition, tracked)
- `gitstats.yaml.state` (runtime, gitignored)

**Pros**: Clean separation, easy gitignore (`*.yaml.state`)
**Cons**: Still two files, still need linking logic

The inline approach is simpler and matches the documented philosophy better.

## Migration

For existing DAG runs with state files:

1. **Detection**: Check for `.autospec/state/dag-runs/` with workflow references
2. **Migration**: Copy state from state file into dag.yaml
3. **Cleanup**: Remove old state file after successful migration
4. **Fallback**: If old state file exists but dag.yaml has no state, auto-migrate on next run

```bash
# Explicit migration command (optional)
autospec dag migrate-state .autospec/dags/gitstats.yaml
```

## NOT Included

- Multiple concurrent runs of same DAG (not a use case)
- Historical run archiving (can add later if needed)
- State file backup before overwrite (git provides history)
- YAML anchors/aliases for state (keep it simple)

## Success Criteria

1. **Single file contains everything** - definition + current state in dag.yaml
2. **No run_id generation** - workflow file path is the identifier
3. **No state directory** - `.autospec/state/dag-runs/` eliminated
4. **All commands work** - run, status, list, merge, cleanup read/write dag.yaml
5. **--fresh works** - clears state sections cleanly
6. **Migration works** - existing state files migrate seamlessly
7. **Resume works** - interrupted runs pick up where they left off

## Run

```bash
autospec run -spti .dev/tasks/dag-orchestration/spec-12-inline-state.md
```
