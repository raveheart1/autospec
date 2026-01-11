# DAG Multi-Spec Orchestration

Run multiple autospec workflows in parallel across git worktrees with dependency management.

## Overview

DAG orchestration is a meta-orchestrator that:
- Executes multiple specs simultaneously
- Respects dependencies between features
- Isolates each spec in its own git worktree
- Auto-merges completed specs back to your base branch

**Problem it solves**: Running 5+ features manually requires multiple terminals, tracking dependencies by hand, and manually merging each one. DAG orchestration handles all of this with a single command.

## Prerequisites

1. **Constitution required**: Run `autospec constitution` first — all workflow stages fail without `.autospec/memory/constitution.yaml`
2. **Worktree setup script** (recommended): Run `autospec worktree gen-script` to generate a project-specific setup script that installs dependencies in each worktree

## Quick Start

```bash
# 1. Create a DAG file defining your features
cat > .autospec/dags/my-features.yaml << 'EOF'
schema_version: "1.0"

dag:
  name: "Q1 Features"

execution:
  max_parallel: 4
  timeout: "4h"
  base_branch: "main"

layers:
  - id: "L0"
    features:
      - id: "050-auth-core"
        description: "Implement JWT authentication with login/logout endpoints, token refresh, and middleware"

      - id: "051-user-model"
        description: "Create user data model with CRUD operations, password hashing, and email validation"

  - id: "L1"
    depends_on: ["L0"]
    features:
      - id: "052-user-profile"
        description: "Add user profile endpoints: view, update, avatar upload, and account deletion"
        depends_on: ["050-auth-core", "051-user-model"]
EOF

# 2. Validate the DAG
autospec dag validate .autospec/dags/my-features.yaml

# 3. Run with parallelism
autospec dag run .autospec/dags/my-features.yaml --parallel
```

## DAG File Reference

```yaml
schema_version: "1.0"  # Required, currently "1.0"

dag:
  name: "Feature Set Name"  # Human-readable name

execution:
  max_parallel: 4      # Max specs to run simultaneously
  timeout: "2h"        # Per-spec timeout
  base_branch: "main"  # Source for worktrees, merge target
  on_conflict: "manual"  # "manual" or "agent"

layers:
  - id: "L0"           # Layer identifier
    name: "Foundation" # Optional human-readable name
    features:
      - id: "001-feature-name"  # Maps to specs/001-feature-name/
        description: "Detailed description used by autospec run -spti"
        timeout: "30m"  # Optional per-spec override
        depends_on: ["other-spec-id"]  # Optional spec-level deps

  - id: "L1"
    depends_on: ["L0"]  # All L1 specs wait for ALL L0 specs
    features:
      - id: "002-next-feature"
        description: "..."
```

## Commands

| Command | Purpose |
|---------|---------|
| `dag validate <file>` | Check DAG structure and dependencies |
| `dag visualize <file>` | ASCII diagram of spec dependencies |
| `dag run <file>` | Execute specs (resumes automatically if interrupted) |
| `dag run <file> --parallel` | Execute specs in parallel |
| `dag run <file> --fresh` | Discard existing state and start fresh |
| `dag run <file> --only spec1,spec2` | Run only specified specs |
| `dag run <file> --only spec1 --clean` | Clean and restart specific specs |
| `dag run <file> --dry-run` | Preview execution plan |
| `dag status <file>` | Show progress for workflow |
| `dag watch <file>` | Live status table (auto-refresh) |
| `dag logs <file> <spec>` | Tail a spec's output |
| `dag list` | List all DAG runs |
| `dag merge <file>` | Merge completed specs to base |
| `dag cleanup <file>` | Remove worktrees for a workflow |

## How It Works

```
dag.yaml
    │
    ▼
┌─────────────────┐
│ Parse & Validate│ ← Check deps, detect cycles
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Create Worktrees│ ← One per spec (../wt-001-feature/)
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Execute Specs   │ ← Run autospec run -spti in each
│ (parallel/seq)  │   Respect dependencies
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Merge & Cleanup │ ← Auto-merge to base branch
└─────────────────┘
```

## Dependency Types

### Layer Dependencies

All specs in a layer wait for ALL specs in dependent layers:

```yaml
layers:
  - id: "L0"
    features: [A, B]
  - id: "L1"
    depends_on: ["L0"]  # C and D wait for BOTH A and B
    features: [C, D]
```

### Spec Dependencies (Additive)

Fine-grained control within or across layers:

```yaml
layers:
  - id: "L0"
    features:
      - id: "A"
      - id: "B"
        depends_on: ["A"]  # B waits for A specifically
```

## Dynamic Spec Creation

Specs are created on-the-fly if they don't exist:

- If `specs/<id>/` exists → Resume from current stage
- If `specs/<id>/` doesn't exist → Create using `description` field

This means you can define 10 features in a DAG and run them all without manually creating specs first.

## Configuration

DAG settings in `.autospec/config.yml`:

```yaml
dag:
  on_conflict: "manual"     # Default conflict handling
  max_spec_retries: 0       # Auto-retry failed specs
  max_log_size: "50MB"      # Max log file size per spec

worktree:
  base_dir: ""              # Parent directory for worktrees
  prefix: ""                # Worktree directory prefix
  setup_script: ""          # Custom setup script path
  setup_timeout: "5m"       # Setup script timeout
  copy_dirs: .autospec,.claude,.opencode  # Dirs to copy
```

## Conflict Handling

When merging completed specs:

**`on_conflict: manual`** (default)
- Pause merge, output context for manual resolution
- Run `dag merge --continue` after fixing

**`on_conflict: agent`**
- Spawn agent to resolve conflicts automatically
- Merge continues after resolution

## Best Practices

1. **Right-size your specs**: Each should be 3-15 tasks, 2-8 hours (see [task-sizing.md](./task-sizing.md))
2. **Genuine dependencies**: Only add deps where truly needed
3. **Detailed descriptions**: Include specific commands, flags, behaviors
4. **Start sequential**: Use `--dry-run` first, then sequential, then parallel
5. **Monitor with watch**: Use `dag watch` in a separate terminal

## Example Use Cases

- **Feature release**: Ship 5 features for a version, some dependent
- **Refactoring**: Break large refactor into independent chunks
- **Onboarding project**: Build a new app feature-by-feature
- **Tech debt**: Parallelize multiple cleanup tasks

## See Also

- [Task Sizing Guide](./task-sizing.md) — Right-size specs for optimal results
- [Parallel Execution](./parallel-execution.md) — Parallel task execution within a spec
- [Worktree Management](./worktree.md) — Git worktree configuration
