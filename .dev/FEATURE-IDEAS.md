# Autospec Feature Ideas

> **Consolidated Index** - Last updated: 2026-01-05
>
> This file summarizes all planned features, improvements, and ideas. Detailed specs are linked where available.

---

## Quick Reference

| Priority | Feature | Size | Status | Details |
|----------|---------|------|--------|---------|
| **P0** | [DAG Parallel Execution](#1-dag-parallel-execution-command) | Large | Planned | [Full spec](tasks/dag-parallel-execution-command.md) |
| **P0** | [Multi-Agent Tool Support](#2-multi-agent-tool-support) | Large | Planned | [Research](tasks/cli-agent-integrations.md) |
| **P1** | [Self-Update Command](#4-self-update-command) | Medium | **Done** | |
| **P1** | [Event-Driven Architecture](#5-event-driven-notification-architecture) | Medium | Planned | [Full spec](tasks/031-event-driven-notifications.md) |
| **P1** | [Workflow Hooks System](#6-workflow-hooks-system) | Medium-Large | Planned | [Full spec](tasks/workflow-hooks-system.md) |
| **P1** | [Interactive Dashboard](#7-interactive-dashboard-progress-bars) | Low | Planned | |
| **P1** | [CLI Test Coverage](#8-cli-test-coverage-improvement) | Medium | Planned | [Full spec](tasks/cli-test-coverage-improvement.md) |
| **P1** | [Regression-Free Integration](#23-regression-free-integration-epic) | Large | Planned | [Full spec](../tasks/regression-free/00-overview.md) |
| **P1** | [Phase Task Verification](#24-phase-task-verification) | Medium | Planned | |
| **P1** | [Implement Command Guardrails](#25-implement-command-guardrails) | Medium | Planned | |
| **P2** | [Config Set Command](#9-config-setget-commands) | Low | Planned | [Full spec](tasks/config-set-command.md) |
| **P2** | [Requirement Traceability](#10-requirement-traceability-in-tasks) | Medium | Planned | |
| **P2** | [Watch Mode](#11-watch-mode) | Medium | Planned | |
| **P2** | [Config Profiles](#12-config-profiles) | Medium | Planned | |
| **P2** | [Spec Templates](#13-spec-templates) | Medium | Planned | |
| **P2** | [Smart Archive Command](#14-smart-archive-command) | Medium | Planned | |
| **P2** | [Phase Context Injection](#15-phase-context-injection) | Medium | Planned | [Full spec](tasks/phase-execution-improvements.md) |
| **P2** | [cclean Library Integration](#16-cclean-library-integration) | Low | **Done** | [Full spec](tasks/integrate-cclean-executor.md) |
| **P3** | [Plugin System](#17-plugin-system) | Large | Planned | |
| **P3** | [Fuzzy Matching UX](#18-fuzzy-matching-for-ux) | Low | Planned | |
| **P3** | [JSON Output Mode](#19-json-output-mode-for-cicd) | Low | Planned | |
| **P3** | [Init Path Argument](#20-init-path-argument) | Low | Planned | [Full spec](tasks/init-path-argument.md) |
| **P3** | [Man Page Generation](#21-man-page-generation) | Low | Planned | |
| **P3** | [Spec List Command](#22-spec-list-command) | Low | Planned | |

### Already Implemented (for reference)

| Feature | Spec |
|---------|------|
| Status Command Enhancement | 063 |
| Command Aliases | 052 |
| Shell Completion Installer | 060 |
| History/Audit Log | 066 |
| Artifact Validation Command | 064 |
| Phase-Based Task Execution (`--phases`) | 019 |
| Task-Level Execution (`--tasks`) | 021 |
| Task Notes Field | 055 |
| Plan Risk Assessment | 054 |
| Orchestrator Schema Validation | 065 |
| Self-Update Command (`autospec update`) | — |
| Worktree Management (`autospec worktree`) | — |
| cclean Library Integration | — |

---

## P0 - Critical Path Features

### 1. DAG Parallel Execution Command

**Size:** Large | **Effort:** 2-3 weeks | **[Full Spec](tasks/dag-parallel-execution-command.md)**

Orchestrate multiple features in parallel across worktrees with dependency-aware scheduling, automatic merging, and cycle detection.

```bash
autospec dag run .autospec/dags/phase-1.yaml
autospec dag status
autospec dag resume <run-id>
autospec dag validate .autospec/dags/phase-1.yaml
autospec dag visualize .autospec/dags/phase-1.yaml --format mermaid
```

**Key capabilities:**
- DAG YAML schema with layers, features, dependencies
- Kahn's algorithm for cycle detection and topological sort
- Parallel process management with configurable `max_parallel`
- Merge strategies: sequential, octopus, manual
- Run state persistence for resume after failure
- Integration with worktree command

**Prerequisite:** Worktree Management Command (#3)

---

### 2. Multi-Agent Tool Support

**Size:** Large | **Effort:** 2-3 weeks | **[Research Doc](tasks/cli-agent-integrations.md)**

Extend beyond Claude Code to support multiple AI coding agents.

**Tier 1 Agents (Automatable):**
| Agent | Prompt Flag | Autonomous Mode | Stars |
|-------|-------------|-----------------|-------|
| Claude Code | `-p` | `--dangerously-skip-permissions` | ~40k |
| Cline | positional | `-Y` (YOLO) | ~48k |
| Gemini CLI | `-p` | `--yolo` | ~87k |
| Codex CLI | `exec` subcommand | inherent | N/A |
| OpenCode | `run` subcommand | inherent | Rising |
| Goose | `run -t` | `GOOSE_MODE=auto` | Growing |

**Implementation:**
```yaml
# New config approach
agent_preset: gemini  # or: claude, cline, codex, opencode, goose

# Or custom
custom_agent:
  command: "gemini"
  args: ["-p", "--yolo", "{{PROMPT}}"]
```

```bash
autospec init --agent gemini
autospec implement --agent cline
```

---

## P1 - High Value Features

### ~~3. Worktree Management Command~~ (DONE)

Implemented. See `autospec worktree --help`.

---

### ~~4. Self-Update Command~~ (DONE)

Implemented. See `autospec update --help`.

---

### 5. Event-Driven Notification Architecture

**Size:** Medium | **Effort:** 1 week | **[Full Spec](tasks/031-event-driven-notifications.md)**

Refactor notification handling from duplicated boilerplate to centralized event-driven architecture.

**Current problem:** ~8-10 lines of notification boilerplate duplicated across 11 CLI commands.

**Target state:**
```go
// BEFORE: 8-10 lines of boilerplate
notifHandler := notify.NewHandler(cfg.Notifications)
startTime := time.Now()
// ... more setup ...
notifHandler.OnCommandComplete("specify", success, duration)

// AFTER: 3 lines
return lifecycle.Run("specify", cfg, func() error {
    return orch.ExecuteSpecify(...)
})
```

**Components:**
- `internal/events/` - Event types and thread-safe event bus
- `internal/lifecycle/` - Command lifecycle wrapper
- Notification handler becomes an event subscriber

---

### 6. Workflow Hooks System

**Size:** Medium-Large | **Effort:** 1-2 weeks | **[Full Spec](tasks/workflow-hooks-system.md)**

Comprehensive hook system with 20+ hook points, 4 execution modes, and conditional execution.

```yaml
hooks:
  stages:
    implement:
      pre:
        - command: "git stash"
          mode: checkpoint
          when:
            - has_uncommitted_changes: true
      post:
        - name: "test"
          command: "make test"
          mode: gate              # Blocks workflow on failure
          timeout: 15m
        - name: "lint"
          command: "make lint"
          mode: gate
          depends_on: [test]
        - command: "notify-send 'Done!'"
          mode: fire-and-forget   # Async, ignore errors

  on_error:
    - command: "echo '{{ERROR_MESSAGE}}' >> errors.log"
```

**4 Execution Modes:**
| Mode | Blocking | On Failure | Use Case |
|------|----------|------------|----------|
| `gate` | Yes | Fail workflow | Validation, quality gates |
| `checkpoint` | Yes | Warn, continue | Nice-to-have checks |
| `best-effort` | No (sync) | Log error | Formatters |
| `fire-and-forget` | No (async) | Ignore | Notifications |

**Key Features:**
- 20+ hook points (pre/post for every stage, phase, task)
- Dependency ordering (`depends_on: [format, lint]`)
- Conditional execution (`when: file_exists: go.mod`)
- Built-in actions (`validate-artifact`, `quality-gates`, `notify-desktop`)
- Rich template variables (`{{SPEC_NAME}}`, `{{PHASE_NUMBER}}`)

---

### 7. Interactive Dashboard / Progress Bars

**Size:** Low | **Effort:** 1-2 days

Enhance `autospec st` with visual progress indicators.

**Current:**
```
Tasks: 12 total | 8 completed | 3 in-progress | 1 pending
```

**Target:**
```
Tasks: ████████░░░░ 67% (8/12)
  → 8 completed | 3 in-progress | 1 pending
```

**Implementation:**
- Use Unicode progress bars (`█░`)
- Color-coded status (green complete, yellow in-progress)
- Summary metrics section

---

### 8. CLI Test Coverage Improvement

**Size:** Medium | **Effort:** 2 weeks | **[Full Spec](tasks/cli-test-coverage-improvement.md)**

Current `internal/cli` coverage: 43.2%. Target: 85-90%.

**Root causes of low coverage:**
1. Hardcoded dependency construction in `RunE` functions
2. Global state via Cobra commands
3. External process dependencies (claude CLI, git)

**Solution:** Interface extraction + dependency injection via `CommandContext` pattern.

**Phases:**
1. Quick wins: Test pure functions (+10-15%)
2. Interface extraction: ConfigLoader, WorkflowExecutor, SpecDetector (+20-25%)
3. Full mock coverage (+20-25%)
4. Integration tests (+5-10%)

---

### 23. Regression-Free Integration (Epic)

**Size:** Large | **Effort:** Multi-phase | **[Full Spec](../tasks/regression-free/00-overview.md)**

Enhance autospec with machine-verifiable specifications and automated validation, enabling agentic coding at scale without human review as a bottleneck.

> Reference: `.dev/tasks/SPECKIT_REGRESSION_FREE_INTEGRATION.md`

**Core Insight:** Traditional prose specifications are ambiguous. AI agents can misinterpret vague requirements. The solution: structured specification languages that bridge natural language and formal verification.

**Integration Philosophy:**
- Progressive enhancement (opt-in, existing workflows unchanged)
- Configuration tiers: `basic` | `enhanced` | `full`
- Non-breaking schema extensions
- Separate verification commands

**Phase 1: Enhanced Schemas (Non-Breaking)**

| Task | Description | Spec |
|------|-------------|------|
| Verification Config | `verification.level` in config | [01-verification-config.md](../tasks/regression-free/01-verification-config.md) |
| EARS Spec Schema | Machine-parseable requirements | [02-ears-spec-schema.md](../tasks/regression-free/02-ears-spec-schema.md) |
| Verification Criteria | Machine-verifiable task completion | [03-verification-criteria-tasks.md](../tasks/regression-free/03-verification-criteria-tasks.md) |

**Phase 2: Verification Command (Opt-In)**

| Task | Description | Spec |
|------|-------------|------|
| Verify Command | `autospec verify` implementation | [04-verify-command.md](../tasks/regression-free/04-verify-command.md) |
| Structured Feedback | AI-optimized error output | [05-structured-feedback.md](../tasks/regression-free/05-structured-feedback.md) |

**Phase 3: Constitution Codegen (Opt-In)**

| Task | Description | Spec |
|------|-------------|------|
| Config Generation | Generate lint/CI from constitution | [06-constitution-codegen.md](../tasks/regression-free/06-constitution-codegen.md) |

**Future Work (Research Phase):**
- Formal-LLM grammar validation for plans
- AgentGuard runtime monitoring
- Full mutation testing integration
- Property-based test generation from specs

**EARS Requirement Patterns:**

| Pattern | Template | Maps To |
|---------|----------|---------|
| Ubiquitous | The [system] shall [action] | Invariant test |
| Event-Driven | When [trigger], the [system] shall [action] | Function test |
| State-Driven | While [state], the [system] shall [action] | State machine test |
| Unwanted Behavior | If [condition], then the [system] shall [action] | Exception test |
| Optional | Where [feature], the [system] shall [action] | Feature flag test |

**Key Benefits:**

| Aspect | Current | Regression-Free |
|--------|---------|-----------------|
| Requirements | Prose descriptions | EARS-formatted, machine-parseable |
| Acceptance | Human interpretation | Machine-verifiable properties |
| Code Review | Human required | Automated verification stack |
| Failure Handling | Human debugging | Auto-retry with structured feedback |

---

## P2 - Medium Priority Features

### 9. Config Set/Get Commands

**Size:** Low | **Effort:** 1-2 days | **[Full Spec](tasks/config-set-command.md)**

```bash
autospec config get timeout
autospec config set timeout 10m
autospec config set notifications.enabled true --project
autospec config toggle skip_preflight
```

**Features:**
- Dotted key path support (`notifications.enabled`)
- Type inference (bool, int, duration, string)
- `--user` vs `--project` scope flags
- Validation against config schema

---

### 10. Requirement Traceability in Tasks

**Size:** Medium | **Effort:** 2-3 days

Link tasks to spec requirements with coverage analysis.

```yaml
# In tasks.yaml
- id: "T015"
  title: "Implement login endpoint"
  status: "Pending"
  requirement_ids: ["FR-001", "FR-003"]  # Links to spec.yaml
```

**Validation:**
- Cross-reference: emit error if requirement_id doesn't exist in spec.yaml
- Coverage analysis in `autospec analyze`:
  ```
  Requirement Coverage:
    FR-001: T015, T016 (covered)
    FR-002: (no tasks!)  ← WARNING
  ```

---

### 11. Watch Mode

**Size:** Medium | **Effort:** 2 days

```bash
autospec watch plan      # Re-run plan when spec.yaml changes
autospec watch tasks     # Re-run tasks when plan.yaml changes
autospec watch --interval 2s
```

**Implementation:**
- Use `github.com/fsnotify/fsnotify`
- Debounce rapid changes
- Show last run timestamp

---

### 12. Config Profiles

**Size:** Medium | **Effort:** 2-3 days

```bash
autospec config profiles           # List profiles
autospec config use fast           # Switch to "fast" profile
autospec config create thorough    # Create new profile
autospec --profile thorough run -a "feature"
```

**Storage:** `~/.config/autospec/profiles/<name>.yml`

**Use cases:**
- Fast dev vs thorough CI
- Different agent configurations
- Per-client project settings

---

### 13. Spec Templates

**Size:** Medium | **Effort:** 2-3 days

```bash
autospec template list
autospec template use api-endpoint
autospec template save my-template
autospec template import ./template.yaml
```

**Storage:** `~/.config/autospec/templates/`

**Default templates:** `api-endpoint`, `bug-fix`, `refactor`, `cli-command`

---

### 14. Smart Archive Command

**Size:** Medium | **Effort:** 2 days

```bash
autospec archive 003-auth           # Archive completed spec
autospec archive --list             # List archived specs
autospec unarchive 003-auth         # Restore from archive
```

**Features:**
- Validates all tasks completed before archiving
- Moves to `specs/archive/YYYY-MM-DD-<name>/`
- Date-stamps archived changes
- Creates audit trail

---

### 15. Phase Context Injection

**Size:** Medium | **Effort:** 3-4 hours | **[Full Spec](tasks/phase-execution-improvements.md)**

Pre-extract full phase context so Claude doesn't read files redundantly.

**Current:** Each phase session, Claude reads spec.yaml + plan.yaml + tasks.yaml (~1200+ lines, 7 times for 7 phases)

**Target:** Inject single context file with spec + plan + phase-specific tasks

**Time savings:** ~15-30 seconds per phase from eliminated file reads

---

### ~~16. cclean Library Integration~~ (DONE)

Implemented. Use `output_style` config or `--output-style` flag.

---

## P3 - Nice to Have Features

### 17. Plugin System

**Size:** Large | **Effort:** 1-2 weeks

```bash
autospec plugin install my-plugin
autospec plugin list
autospec plugin remove my-plugin
autospec my-custom-command    # From plugin
```

**Design:**
- Plugins are Go binaries in `~/.config/autospec/plugins/`
- Naming convention: `autospec-<pluginname>`
- Auto-discover and register as subcommands

---

### 18. Fuzzy Matching for UX

**Size:** Low | **Effort:** 2 hours

When spec not found, suggest nearest matches:
```
Error: spec 'add-usr-auth' not found
Did you mean 'add-user-auth'?
```

**Implementation:**
- Levenshtein distance calculation
- Integrate into spec resolution in `internal/spec/`

---

### 19. JSON Output Mode for CI/CD

**Size:** Low | **Effort:** 3 hours

```bash
autospec st --json
autospec list --json
autospec doctor --json
```

Add `--json` flag to status/view commands for machine-readable output.

---

### 20. Init Path Argument

**Size:** Low | **Effort:** 1 day | **[Full Spec](tasks/init-path-argument.md)**

```bash
autospec init                    # Current directory
autospec init my-project         # Create and init new directory
autospec init ~/repos/my-project # Absolute path
```

---

### 21. Man Page Generation

**Size:** Low | **Effort:** 1 day

```bash
autospec docs man           # Generate man pages to ./man/
autospec docs install       # Install man pages to system
```

Uses Cobra's built-in `doc.GenManTree()`.

---

### 22. Spec List Command

**Size:** Low | **Effort:** 1-2 days

```bash
autospec list
autospec list --status
autospec list --sort=date
autospec list --filter="auth"
```

**Output:**
```
NUM  NAME                  STATUS      TASKS  LAST MODIFIED
001  initial-setup         complete    5/5    2024-01-15
002  go-binary-migration   in-progress 8/12   2024-01-20
003  auth-feature          planned     0/15   2024-01-22
```

---

## Constitution Improvements

**[Full Spec](tasks/constitution-improvements.md)**

17 new principles proposed based on analysis of 35+ conversations:

| ID | Name | Priority |
|----|------|----------|
| PRIN-014 | Explicit Assumptions | SHOULD |
| PRIN-015 | Explicit Scope Boundaries | MUST |
| PRIN-016 | Quality Gate Final Phase | NON-NEGOTIABLE |
| PRIN-017 | Task Dependency Structure | MUST |
| PRIN-018 | Design Decision Documentation | SHOULD |
| PRIN-019 | Infrastructure Reuse | SHOULD |
| PRIN-020 | Backward Compatibility | MUST |
| PRIN-021 | Context Efficiency | SHOULD |
| PRIN-022 | Fail-Fast Validation | MUST |
| PRIN-023 | Test Coverage Targets | MUST |
| PRIN-024 | Data-Driven Specification | SHOULD |
| PRIN-025 | Pre-Spec Research Documents | SHOULD |
| PRIN-026 | Proportional Specification | SHOULD |
| PRIN-027 | Exploration Before Planning | SHOULD |
| PRIN-028 | Graceful Tool Degradation | SHOULD |
| PRIN-029 | Immediate Artifact Validation | MUST |
| PRIN-030 | Structured Ambiguity Detection | SHOULD |
| PRIN-031 | Recommended Options with Rationale | SHOULD |

---

## Core Feature Improvements

**[Full Spec](tasks/core-feature-improvements.md)**

### Task Priority Field
```yaml
- id: "T015"
  priority: "P0"  # P0, P1, P2, P3
```

### Task Complexity Estimate
```yaml
- id: "T015"
  complexity: "L"  # XS, S, M, L, XL
```

### Spec Dependencies Field
```yaml
feature:
  depends_on_specs: ["007-user-auth"]
```

### Phase Prerequisites Validation
```yaml
phases:
  - number: 2
    prerequisites:
      - description: "PostgreSQL running"
        check_command: "pg_isready -h localhost"
```

### Validation Severity Configuration
```yaml
validation:
  severity:
    blocked_reason_missing: warn
    orphan_requirement: error
    high_risk_no_mitigation: error
```

---

## Quick Start Commands

Copy-paste to start specifying any feature:

```bash
# P0 Features
autospec specify "$(cat .dev/tasks/dag-parallel-execution-command.md)"

# P1 Features
autospec specify "$(cat .dev/tasks/031-event-driven-notifications.md)"

autospec specify "$(cat .dev/tasks/workflow-hooks-system.md)"

# P2 Features
autospec specify "$(cat .dev/tasks/config-set-command.md)"

autospec specify "Add 'autospec watch PHASE' command that monitors relevant files and re-runs phase on changes. Use fsnotify library. watch plan monitors spec.yaml, watch tasks monitors plan.yaml."

autospec specify "Add config profile system: 'autospec config profiles' (list), 'autospec config use NAME' (switch), 'autospec config create NAME'. Store in ~/.config/autospec/profiles/."

# P3 Features
autospec specify "Add fuzzy matching for spec names. When spec not found, suggest nearest matches using Levenshtein distance. Integrate into spec resolution in internal/spec/."

autospec specify "Add --json flag to autospec st, autospec list, autospec doctor for machine-readable CI/CD output."
```

---

## Related Files

| File | Purpose |
|------|---------|
| [tasks/dag-parallel-execution-command.md](tasks/dag-parallel-execution-command.md) | DAG execution full spec |
| [tasks/worktree-management-command.md](tasks/worktree-management-command.md) | Worktree management full spec |
| [tasks/cli-agent-integrations.md](tasks/cli-agent-integrations.md) | Multi-agent research |
| [tasks/workflow-hooks-system.md](tasks/workflow-hooks-system.md) | Workflow hooks full spec |
| [tasks/030-hook-based-workflow-automation.md](tasks/030-hook-based-workflow-automation.md) | Hook automation (superseded) |
| [tasks/031-event-driven-notifications.md](tasks/031-event-driven-notifications.md) | Event architecture full spec |
| [tasks/cli-test-coverage-improvement.md](tasks/cli-test-coverage-improvement.md) | Test coverage improvement plan |
| [tasks/constitution-improvements.md](tasks/constitution-improvements.md) | Constitution principle proposals |
| [tasks/core-feature-improvements.md](tasks/core-feature-improvements.md) | YAML schema improvements |
| [tasks/phase-execution-improvements.md](tasks/phase-execution-improvements.md) | Phase context optimization |
| [tasks/config-set-command.md](tasks/config-set-command.md) | Config CLI commands |
| [tasks/init-path-argument.md](tasks/init-path-argument.md) | Init path argument |
| [tasks/integrate-cclean-executor.md](tasks/integrate-cclean-executor.md) | cclean integration |
| [tasks/open-spec-features.md](tasks/open-spec-features.md) | OpenSpec feature comparison |
| [tasks/feature-ideas.md](tasks/feature-ideas.md) | Original feature brainstorm |
| [tasks/regression-free/00-overview.md](../tasks/regression-free/00-overview.md) | Regression-free integration overview |
| [tasks/regression-free/01-verification-config.md](../tasks/regression-free/01-verification-config.md) | Verification level config |
| [tasks/regression-free/02-ears-spec-schema.md](../tasks/regression-free/02-ears-spec-schema.md) | EARS requirement format |
| [tasks/regression-free/03-verification-criteria-tasks.md](../tasks/regression-free/03-verification-criteria-tasks.md) | Task verification criteria |
| [tasks/regression-free/04-verify-command.md](../tasks/regression-free/04-verify-command.md) | Verify command |
| [tasks/regression-free/05-structured-feedback.md](../tasks/regression-free/05-structured-feedback.md) | AI feedback format |
| [tasks/regression-free/06-constitution-codegen.md](../tasks/regression-free/06-constitution-codegen.md) | Constitution config generation |
| [tasks/SPECKIT_REGRESSION_FREE_INTEGRATION.md](tasks/SPECKIT_REGRESSION_FREE_INTEGRATION.md) | Original research document |

---

## Priority Matrix

```
                    High Value
                        │
    [Self-Update]       │    [DAG Parallel]
    [Config Set]        │    [Multi-Agent]
    [Fuzzy Match]       │    [Regression-Free] ←── NEW EPIC
                        │    [Worktree Mgmt]
Low Effort ─────────────┼───────────────── High Effort
                        │
    [Man Pages]         │    [Plugin System]
    [JSON Output]       │    [Event Architecture]
    [Spec List]         │
                        │
                    Low Value
```

---

## Rejected Ideas

| Idea | Reason |
|------|--------|
| NeedsReview Task Status | Blocked + blocked_reason + notes already captures this |
| Task Output Artifacts Field | Multiple tasks touch same files; rely on test suite |
| Cross-Artifact Traceability | Artifacts organized by folder; paths are inferred |
| Two-folder (specs/ vs changes/) | Adds complexity; single-folder works well |
| Markdown-based specs | YAML is more structured |
