# Spec 1: DAG Schema & Validation

## Context

Part of **DAG Multi-Spec Orchestration** - a meta-orchestrator that runs multiple `autospec run` workflows in parallel across worktrees with dependency management. See [00-summary.md](00-summary.md) for full vision.

## Scope

Define YAML schema for multi-spec workflows. Add validation and visualization commands.

## Commands

- `autospec dag validate <file>` - Validate DAG structure, detect cycles, check descriptions
- `autospec dag visualize <file>` - ASCII diagram of spec dependencies

## Key Deliverables

- YAML schema for `.autospec/dags/*.yaml` files
- Parser in `internal/dag/` package
- Layers containing features, each with id, description, optional depends_on
- Cycle detection for spec-level dependencies
- Validation: each spec has description (used to create spec if folder doesn't exist)
- ASCII-only visualization

## Schema

```yaml
schema_version: "1.0"

dag:
  name: "V1 Features"

execution:
  max_parallel: 4         # Default concurrency
  timeout: "2h"           # Default per-spec timeout

layers:
  - id: "L0"
    name: "Foundation"    # Optional human-readable name
    features:
      - id: "050-error-handling"
        description: "Improve error handling with context wrapping"

      - id: "051-retry"
        description: "Add retry with exponential backoff"
        depends_on: ["050-error-handling"]
        timeout: "30m"    # Optional override

  - id: "L1"
    depends_on: ["L0"]    # Layer-level dependency
    features:
      - id: "052-caching"
        description: "Add caching layer for API responses"
```

## Field Reference

**Execution fields:**

| Field | Required | Default | Purpose |
|-------|----------|---------|---------|
| `base_branch` | No | current branch | Worktrees created from here, merges target here |
| `max_parallel` | No | 4 | Max concurrent specs |
| `timeout` | No | 2h | Default per-spec timeout |
| `on_conflict` | No | manual | Merge conflict handling: `manual` or `agent` |

**Feature fields:**

| Field | Required | Purpose |
|-------|----------|---------|
| `id` | Yes | Maps to `specs/<id>/` folder |
| `description` | Yes | Used by `autospec run -spti` to create spec if folder doesn't exist |
| `depends_on` | No | Spec-level dependencies (list of spec IDs) |
| `timeout` | No | Override default timeout for this spec |

**Note:** Agent/model come from autospec config, not dag.yaml.

## Validation Rules

- Each spec must have `description` field
- All `depends_on` references must be valid spec IDs in the DAG
- No circular dependencies
- Layer `depends_on` must reference valid layer IDs

## Source of Truth

- `specs/` folders are source of truth for spec content and completion status
- dag.yaml only defines: which specs, dependencies, execution config
- Run state tracked separately in `.autospec/state/dag-runs/`

## NOT Included

- No execution (Spec 2)
- No worktree creation (Spec 2)
- No state persistence (Spec 2)
- No mermaid output
- No per-spec agent/model (use autospec config)

## Run

```bash
autospec run -spti .dev/tasks/dag-orchestration/spec-1-schema-validation.md
```
