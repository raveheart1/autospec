# Spec 8: DAG Name in Branch Names

## Context

Part of **DAG Multi-Spec Orchestration** - a meta-orchestrator that runs multiple `autospec run` workflows in parallel across worktrees with dependency management. See [00-summary.md](00-summary.md) for full vision.

## Problem

Current branch names are opaque and don't identify which DAG they belong to:

```
dag/20260111_112034_ba7166b1/200-repo-reader
```

Given this DAG definition:
```yaml
dag:
  name: "GitStats CLI v1"

execution:
  max_parallel: 4
  timeout: "4h"
  base_branch: "main"
```

Users cannot tell which DAG a branch belongs to without looking up the run ID. This makes:
- `git branch -a` output harder to interpret
- Multiple DAG runs confusing to distinguish
- Worktree directories less self-documenting

## Solution

Generate a slug from `dag.name` at runtime for use in branch names. Optionally allow explicit `id` override for power users.

**Before:**
```
dag/20260111_112034_ba7166b1/200-repo-reader
```

**After:**
```
dag/gitstats-cli-v1/200-repo-reader
```

### Design Decision: Runtime vs YAML-Defined Slug

**Runtime generation from `name`** (recommended):
- Single source of truth (DRY)
- Simpler YAML schema
- No sync issues between name and slug
- Slug is technical detail, not semantic

**Optional `id` field for override:**
```yaml
dag:
  name: "GitStats CLI v1"    # Required, displayed in UI
  id: "gitstats"             # Optional, overrides slugified name
```

The AI is NOT prompted to generate `id` - it's a power-user escape hatch for cases where the auto-generated slug is undesirable.

## Key Deliverables

### 1. DAG ID Resolution

Implement `ResolveDAGID(dag *DAG) string`:
1. If `dag.ID` is set → use it directly (user override)
2. If `dag.ID` is empty → slugify `dag.Name`

### 2. Slug Generation

Create a `Slugify(name string) string` function:
- Lowercase
- Replace whitespace with hyphens
- Remove non-alphanumeric characters (except hyphens)
- Collapse multiple hyphens
- Trim leading/trailing hyphens
- Truncate to reasonable length (e.g., 50 chars)

Examples:
| Input | Output |
|-------|--------|
| `"GitStats CLI v1"` | `gitstats-cli-v1` |
| `"Feature: Auth & Sessions"` | `feature-auth-sessions` |
| `"  My  DAG  "` | `my-dag` |
| `"123-numbers-first"` | `123-numbers-first` |

### 3. Branch Name Format

New format: `dag/<dag-name-slug>/<spec-id>`

```
dag/gitstats-cli-v1/200-repo-reader
dag/gitstats-cli-v1/201-commit-analyzer
dag/gitstats-cli-v1/202-author-stats
```

### 4. Worktree Directory Names

Update worktree directory naming to match:

**Before:**
```
../dag-20260111_112034_ba7166b1-200-repo-reader/
```

**After:**
```
../dag-gitstats-cli-v1-200-repo-reader/
```

### 5. Collision Handling

If a branch already exists (from previous run or naming collision):
1. For idempotent resume: reuse existing branch/worktree
2. For `--fresh`: delete and recreate
3. For collision with different DAG: append short hash suffix as fallback

```
dag/gitstats-cli-v1/200-repo-reader           # First run
dag/gitstats-cli-v1-a7f3/200-repo-reader      # Collision fallback
```

### 6. State File Reference

State files should store the resolved ID and original name for display:

```yaml
# .autospec/state/dag-runs/gitstats.yaml.state
dag_name: "GitStats CLI v1"
dag_id: "gitstats-cli-v1"    # Resolved: from dag.id or slugified name
specs:
  200-repo-reader:
    branch: "dag/gitstats-cli-v1/200-repo-reader"
    worktree: "../dag-gitstats-cli-v1-200-repo-reader"
    status: running
```

## Behavior Details

### `dag run workflow.yaml`

1. Parse DAG, extract `dag.name` and optional `dag.id`
2. Resolve ID: use `dag.id` if set, else slugify `dag.name`
3. Create branches as `dag/<id>/<spec-id>`
4. Create worktrees as `../dag-<id>-<spec-id>/`

### Migration

Existing runs with old-format branches continue to work:
- State file records actual branch/worktree names
- Resume uses stored names, not regenerated ones
- Only new runs use the new format

### Fallback for Missing Name

If both `dag.id` and `dag.name` are not specified:
- Use workflow filename as ID source
- `workflow.yaml` → `dag/workflow/200-repo-reader`
- `features/v1.yaml` → `dag/v1/200-repo-reader`

Resolution priority:
1. `dag.id` (explicit override)
2. Slugified `dag.name`
3. Workflow filename (basename without extension)

## Updated Output

```
❯ autospec dag run .autospec/dags/gitstats.yaml

=== Layer L0: Core Infrastructure ===

--- Spec: 200-repo-reader ---
[200-repo-reader] Creating worktree: branch dag/gitstats-cli-v1/200-repo-reader
[200-repo-reader] Running: autospec run -spti ...
```

### 7. Validation: Unique Names and IDs

When running `dag validate`, check for uniqueness across all DAG files in the project:

**Validation rules:**
1. **Resolved IDs must be unique** - No two DAGs can have the same resolved ID (whether from explicit `id` or slugified `name`)
2. **Names should be unique** - Warn if two DAGs have the same `name` (even if `id` differs)

**Implementation:**
```go
// In dag validate command
func ValidateDAGUniqueness(dagFiles []string) error {
    seenIDs := make(map[string]string)   // id -> filepath
    seenNames := make(map[string]string) // name -> filepath

    for _, file := range dagFiles {
        dag := ParseDAG(file)
        resolvedID := ResolveDAGID(dag)

        if existing, ok := seenIDs[resolvedID]; ok {
            return fmt.Errorf("duplicate DAG ID %q: %s and %s", resolvedID, existing, file)
        }
        seenIDs[resolvedID] = file

        if existing, ok := seenNames[dag.Name]; ok {
            warn("duplicate DAG name %q: %s and %s", dag.Name, existing, file)
        }
        seenNames[dag.Name] = file
    }
    return nil
}
```

**Discovery scope:**
- Scan `.autospec/dags/*.yaml` by default
- Also check any path passed to `dag validate`

**Example output:**
```
❯ autospec dag validate .autospec/dags/

Validating DAG files...
  ✓ gitstats.yaml (id: gitstats-cli-v1)
  ✓ auth-features.yaml (id: auth-features)
  ✗ payments.yaml (id: gitstats-cli-v1)
    ERROR: Duplicate ID "gitstats-cli-v1" - conflicts with gitstats.yaml

1 error, 0 warnings
```

## NOT Included

- Changing state file naming (covered in spec-7)
- Custom branch prefix configuration (use simple `dag/` prefix)

## Run

```bash
autospec run -spti .dev/tasks/dag-orchestration/spec-8-dag-name-branches.md
```
