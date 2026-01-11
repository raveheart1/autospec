# Spec 1: DAG Schema & Validation

## Context

Part of **DAG Multi-Spec Orchestration** - a meta-orchestrator that runs multiple `autospec run` workflows in parallel across worktrees with dependency management. See [00-summary.md](00-summary.md) for full vision.

## Scope

Define YAML schema for multi-spec workflows. Add validation and visualization commands.

## Commands

- `autospec dag validate <file>` - Validate DAG structure, detect cycles, verify specs exist
- `autospec dag visualize <file>` - ASCII diagram of spec dependencies

## Key Deliverables

- YAML schema for `.autospec/dags/*.yaml` files
- Layers containing features, each referencing a spec folder in `specs/*`
- Parser in `internal/dag/` package
- Cycle detection for spec-level dependencies
- Spec existence validation (error if `specs/<id>/` doesn't exist)
- ASCII-only visualization (no mermaid)

## Schema Example

```yaml
schema_version: "1.0"
dag:
  name: "V1 Feature Set"

layers:
  - id: "L0"
    name: "Foundation"
    features:
      - id: "050-error-handling"  # Must exist: specs/050-error-handling/
        depends_on: []
      - id: "051-retry-backoff"
        depends_on: ["050-error-handling"]

  - id: "L1"
    depends_on: ["L0"]
    features:
      - id: "052-caching"
```

## NOT Included

- No execution (Spec 2)
- No worktree creation (Spec 2)
- No state persistence (Spec 2)
- No mermaid output

## Run

```bash
autospec run -spti .dev/tasks/dag-orchestration/spec-1-schema-validation.md
```
