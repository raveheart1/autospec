# 009 - Optional Phase Commands

Add standalone CLI commands and `run` command flags for the optional SpecKit phases: analyze, constitution, clarify, and checklist.

## Problem

Currently, the optional SpecKit commands (`/autospec.analyze`, `/autospec.constitution`, `/autospec.clarify`, `/autospec.checklist`) are only accessible via Claude Code slash commands. Users cannot:
1. Run them directly via `autospec <command>`
2. Include them in flexible phase combinations via `autospec run -c`

The 008-flexible-phase-workflow spec only covered the 4 core phases (specify, plan, tasks, implement), leaving these auxiliary commands without CLI equivalents.

## User Stories

### US-001: Standalone Optional Commands
As a developer, I want to run optional phases directly from the command line (`autospec checklist`, `autospec analyze`, etc.) so that I can use them without invoking Claude Code slash commands manually.

**Acceptance:**
- `autospec checklist` runs `/autospec.checklist`
- `autospec analyze` runs `/autospec.analyze`
- `autospec clarify` runs `/autospec.clarify`
- `autospec constitution` runs `/autospec.constitution`
- Each command accepts optional prompt arguments like the core commands

### US-002: Run Command Integration
As a developer using flexible phase workflows, I want to include optional phases in my `autospec run` combinations so that I can run complete workflows with validation in a single command.

**Acceptance:**
- `autospec run -c` or `autospec run --checklist` runs checklist phase
- `autospec run -z` or `autospec run --analyze` runs analyze phase
- `autospec run -r` or `autospec run --clarify` runs clarify phase
- `autospec run -n` or `autospec run --constitution` runs constitution phase
- Flags can be combined: `autospec run -sptic` (specify + plan + tasks + implement + checklist)

### US-003: Execution Order
As a developer, I want optional phases to execute at sensible points in the workflow so that validations happen at the right time.

**Acceptance:**
- Constitution runs first (before specify) if included
- Clarify runs after specify (before plan) if included
- Checklist runs after tasks (before implement) if included
- Analyze runs after tasks (before implement) if included
- Order: constitution → specify → clarify → plan → tasks → checklist → analyze → implement

## Requirements

### Functional

| ID | Requirement |
|----|-------------|
| FR-001 | System MUST provide `autospec checklist [prompt]` command that executes `/autospec.checklist` |
| FR-002 | System MUST provide `autospec analyze [prompt]` command that executes `/autospec.analyze` |
| FR-003 | System MUST provide `autospec clarify [prompt]` command that executes `/autospec.clarify` |
| FR-004 | System MUST provide `autospec constitution [prompt]` command that executes `/autospec.constitution` |
| FR-005 | System MUST support `-c/--checklist` flag in `run` command |
| FR-006 | System MUST support `-z/--analyze` flag in `run` command |
| FR-007 | System MUST support `-r/--clarify` flag in `run` command |
| FR-008 | System MUST support `-n/--constitution` flag in `run` command |
| FR-009 | System MUST execute phases in canonical order when combined |
| FR-010 | System MUST validate prerequisites for optional phases (e.g., spec.yaml exists for clarify) |

### Flag Mapping

| Phase | Short | Long | Slash Command |
|-------|-------|------|---------------|
| constitution | `-n` | `--constitution` | `/autospec.constitution` |
| specify | `-s` | `--specify` | `/autospec.specify` |
| clarify | `-r` | `--clarify` | `/autospec.clarify` |
| plan | `-p` | `--plan` | `/autospec.plan` |
| tasks | `-t` | `--tasks` | `/autospec.tasks` |
| checklist | `-c` | `--checklist` | `/autospec.checklist` |
| analyze | `-z` | `--analyze` | `/autospec.analyze` |
| implement | `-i` | `--implement` | `/autospec.implement` |

Short flag rationale:
- `-c` for **c**hecklist
- `-z` for analy**z**e (since `-a` is taken by `--all`)
- `-r` for cla**r**ify
- `-n` for co**n**stitution

### Canonical Execution Order

```
constitution → specify → clarify → plan → tasks → checklist → analyze → implement
```

This order ensures:
1. Constitution sets project principles before any work
2. Clarify refines spec before planning
3. Checklist/analyze validate before implementation

## Implementation Notes

### New CLI Files
- `internal/cli/checklist.go` - standalone checklist command
- `internal/cli/analyze.go` - standalone analyze command
- `internal/cli/clarify.go` - standalone clarify command
- `internal/cli/constitution.go` - standalone constitution command

### Modified Files
- `internal/cli/run.go` - add new flags
- `internal/workflow/phase_config.go` - add new phases to config
- `internal/workflow/preflight.go` - add prerequisite checks for optional phases

### Artifact Dependencies

| Phase | Requires | Produces |
|-------|----------|----------|
| constitution | (none) | `.specify/memory/constitution.md` or updates CLAUDE.md |
| clarify | spec.yaml | updates spec.yaml |
| checklist | spec.yaml | checklists/*.yaml |
| analyze | spec.yaml, plan.yaml, tasks.yaml | (validation output only) |

## Examples

```bash
# Standalone commands
autospec checklist                     # Run checklist on current spec
autospec checklist "Focus on security" # With prompt guidance
autospec analyze                       # Run cross-artifact analysis
autospec clarify                       # Identify spec gaps
autospec constitution                  # Create/update constitution

# Combined with run command
autospec run -a                        # All 4 core phases (existing)
autospec run -sptic                    # Core phases + checklist
autospec run -sptiz                    # Core phases + analyze
autospec run -nspti                    # Constitution + all core phases
autospec run -sr                       # Specify + clarify
autospec run -tcz                      # Tasks + checklist + analyze
autospec run -czi                      # Checklist + analyze + implement
```

## Out of Scope

- Adding optional phases to `autospec all` command (keeps existing behavior)
- Creating new slash commands (already exist as `/autospec.*`)
- Modifying the YAML artifact schemas
