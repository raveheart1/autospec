# Core Feature Improvements

Ideas for improving core autospec features: YAML schemas, validation, task management, and workflow orchestration. These are incremental enhancements to existing functionality, similar to the recently added `blocked_reason` field.

---

## Quick Start Commands

Copy-paste any of these to start specifying a feature:

```bash
# 0. Orchestrator Schema Validation (HIGH PRIORITY) - See separate file
# Full spec: .dev/tasks/orchestrator-schema-validation.md
autospec specify "$(cat .dev/tasks/orchestrator-schema-validation.md)"

# 1. Task Priority Field
autospec specify "Add optional 'priority' field to tasks in tasks.yaml. PROBLEM: All tasks appear equal, making it hard to identify critical-path work. SCHEMA CHANGE: Add priority field (P0/P1/P2/P3) to TaskItem struct with validation. DEFAULT: P2 (normal). DISPLAY: Show priority in 'autospec st' output, sort blocked tasks by priority. VALIDATION: Emit warning if phase contains all P0 tasks (suggests poor prioritization). CLI: Add --priority flag to 'autospec task' commands. BENEFIT: Enables smarter retry ordering and helps Claude prioritize when context-limited."

# 2. Task Complexity Estimate
autospec specify "Add optional 'complexity' field to tasks in tasks.yaml. PROBLEM: No way to estimate effort or identify tasks likely to fail. SCHEMA CHANGE: Add complexity field (XS/S/M/L/XL) to TaskItem struct. VALIDATION: Emit warning if complexity missing on implementation tasks. DISPLAY: Show total complexity in 'autospec st' summary. ANALYTICS: Track actual completion time vs estimated complexity to calibrate future estimates. BENEFIT: Helps identify tasks that need session isolation (--tasks flag) vs batching."

# 3. Task Notes/Comments Field - DONE!
autospec specify "Add optional 'notes' field to tasks in tasks.yaml for implementation hints. PROBLEM: Task titles are brief; complex tasks need context that doesn't fit elsewhere. SCHEMA CHANGE: Add notes string field to TaskItem struct. SLASH COMMAND: Update /autospec.implement to read and display notes before starting each task. USE CASES: Edge cases to handle, gotchas from planning, links to relevant docs. VALIDATION: No validation needed (purely informational). BENEFIT: Preserves planning context through to implementation."

# 4. Requirement Traceability in Tasks
autospec specify "Add optional 'requirement_ids' field to tasks in tasks.yaml linking to spec.yaml requirements. PROBLEM: No explicit connection between tasks and the requirements they fulfill. SCHEMA CHANGE: Add requirement_ids array field to TaskItem (e.g., ['FR-001', 'NFR-003']). VALIDATION: Cross-reference validation ensuring referenced IDs exist in spec.yaml. DISPLAY: Show coverage matrix in 'autospec analyze' output. BENEFIT: Enables requirement coverage analysis - which requirements have implementing tasks, which are orphaned."

# 5. Spec Dependencies Field
autospec specify "Add optional 'depends_on_specs' field to spec.yaml for multi-feature dependencies. PROBLEM: Features often depend on other features but this isn't captured. SCHEMA CHANGE: Add depends_on_specs array field to feature section (e.g., ['007-auth-base']). VALIDATION: Emit warning if dependent spec not found or not Completed. DISPLAY: Show dependency graph in 'autospec list' with --deps flag. CLI: Block 'autospec implement' if dependencies not complete. BENEFIT: Prevents implementing features that depend on unfinished prerequisites."

# 6. Plan Risk Assessment Section - DONE
autospec specify "Add optional 'risks' section to plan.yaml for documenting implementation risks. PROBLEM: Risks identified during planning aren't captured in artifacts. SCHEMA CHANGE: Add risks array to plan.yaml with fields: id, description, likelihood (low/medium/high), impact (low/medium/high), mitigation. VALIDATION: Emit warning if high-impact risks have no mitigation. DISPLAY: Show risks summary in 'autospec st' for specs in planning phase. BENEFIT: Forces explicit risk acknowledgment before implementation."

# 7. NeedsReview Task Status - REJECTED
# Reason: Overcomplicates status enum. Blocked + blocked_reason + notes already captures this.
# "blocked_reason: Need human review - uncertain about edge cases" is clear enough.
# KISS - don't add enum values when text fields suffice.

# 8. Phase Prerequisites Validation
autospec specify "Add optional 'prerequisites' section to phases in tasks.yaml for pre-implementation checks. PROBLEM: Phases may require external setup (database running, env vars set) not captured in YAML. SCHEMA CHANGE: Add prerequisites array to TaskPhase with fields: description, check_command (optional bash command). VALIDATION: Run check_commands before starting phase in --phases mode, block if any fail. DISPLAY: Show prerequisites in phase summary. EXAMPLE: 'Database migrations applied' with check 'psql -c \"SELECT 1\"'. BENEFIT: Catches environment issues before wasting implementation tokens."

# 9. Task Output Artifacts Field - REJECTED
# Reason: Multiple tasks touch same files; tasks modify existing code, not just create new files.
# Git diff verification might work but is weak (just checks "did anything change?").
# Better to rely on test suite and existing validation.

# 10. Validation Severity Levels
autospec specify "Add severity configuration for validation warnings. PROBLEM: All validation warnings treated equally; some are more important than others. CONFIG CHANGE: Add validation_severity section to config.yml mapping warning types to levels (ignore/warn/error). DEFAULTS: blocked_reason_missing: warn, orphan_requirement: warn, high_risk_no_mitigation: error. BEHAVIOR: 'error' level warnings cause non-zero exit code. CLI: Add --strict flag to treat all warnings as errors. BENEFIT: Teams can enforce project-specific quality gates."
```

---

## 0. Orchestrator Schema Validation (HIGH PRIORITY) - DONE

**See full spec:** [.dev/tasks/orchestrator-schema-validation.md](orchestrator-schema-validation.md)

```bash
autospec specify "$(cat .dev/tasks/orchestrator-schema-validation.md)"
```

**Summary:** Replace file-existence checks with full schema validation. On retry, inject validation errors into Claude's prompt so it knows what to fix.

**Effort:** Medium (2-3 days)

---

## 1. Task Priority Field

**Why:** When tasks pile up (especially blocked ones), there's no way to identify critical-path work. All tasks appear equally important.

**Current state:**
```yaml
- id: "T015"
  title: "Implement core authentication logic"
  status: "Pending"
  type: "implementation"
  # No indication this is more critical than T016
```

**Proposed:**
```yaml
- id: "T015"
  title: "Implement core authentication logic"
  status: "Pending"
  type: "implementation"
  priority: "P0"  # Critical path - must complete first
```

**Schema change:**
```go
type TaskItem struct {
    // ... existing fields ...
    Priority string `yaml:"priority,omitempty"` // P0, P1, P2, P3
}
```

**Validation rules:**
- Optional field (default: P2)
- Enum: P0 (critical), P1 (high), P2 (normal), P3 (low)
- Warning if phase has >50% P0 tasks (suggests poor prioritization)

**Benefits:**
- Smarter retry ordering (P0 first)
- Claude can prioritize when context-limited
- Clear triage for blocked tasks

**Effort:** Low (1-2 days)

---

## 2. Task Complexity Estimate

**Why:** Some tasks are trivial (add a field), others are complex (implement auth flow). No way to estimate or track this.

**Current state:**
```yaml
- id: "T001"
  title: "Add config field"
  status: "Pending"
  type: "implementation"
  # Implicitly simple, but not captured

- id: "T015"
  title: "Implement OAuth2 flow with PKCE"
  status: "Pending"
  type: "implementation"
  # Implicitly complex, but not captured
```

**Proposed:**
```yaml
- id: "T001"
  title: "Add config field"
  status: "Pending"
  type: "implementation"
  complexity: "XS"

- id: "T015"
  title: "Implement OAuth2 flow with PKCE"
  status: "Pending"
  type: "implementation"
  complexity: "L"
```

**Validation rules:**
- Optional (warning if missing on implementation tasks)
- Enum: XS, S, M, L, XL
- Warning if phase total exceeds threshold (too much in one phase)

**Benefits:**
- Identify tasks needing session isolation (complex → --tasks flag)
- Better phase balancing during /autospec.tasks
- Historical calibration: compare estimate vs actual duration

**Effort:** Low (1-2 days)

---

## 3. Task Notes/Comments Field - DONE!

**Why:** Task titles must be concise, but complex tasks often need implementation hints that don't fit in the title or acceptance criteria.

**Current state:**
```yaml
- id: "T015"
  title: "Implement retry logic with exponential backoff"
  status: "Pending"
  type: "implementation"
  # Where do implementation hints go?
```

**Proposed:**
```yaml
- id: "T015"
  title: "Implement retry logic with exponential backoff"
  status: "Pending"
  type: "implementation"
  notes: |
    - Use 2^attempt * 100ms base delay
    - Cap at 30 seconds max
    - See internal/http/client.go for existing pattern
    - Watch for goroutine leaks in tests
```

**Slash command update:**
Update `/autospec.implement` to display notes before starting each task:
```
Starting T015: Implement retry logic with exponential backoff

Notes:
  - Use 2^attempt * 100ms base delay
  - Cap at 30 seconds max
  - See internal/http/client.go for existing pattern
```

**Benefits:**
- Preserves planning context through implementation
- Reduces re-discovery of gotchas
- Enables cross-session knowledge transfer

**Effort:** Low (1 day)

---

## 4. Requirement Traceability in Tasks

**Why:** No explicit link between tasks and the requirements they implement. Can't verify all requirements have tasks.

**Current state:**
- spec.yaml has requirements (FR-001, FR-002, NFR-001...)
- tasks.yaml has tasks (T001, T002...)
- No explicit connection between them

**Proposed:**
```yaml
# In tasks.yaml
- id: "T015"
  title: "Implement user login endpoint"
  status: "Pending"
  type: "implementation"
  requirement_ids: ["FR-001", "FR-003"]  # Links to spec.yaml
```

**Validation:**
- Cross-reference: emit error if requirement_id doesn't exist in spec.yaml
- Coverage analysis: emit warning for requirements with no implementing tasks

**Display in `autospec analyze`:**
```
Requirement Coverage:
  FR-001: T015, T016 (covered)
  FR-002: (no tasks!)  ← WARNING
  FR-003: T015 (covered)
  NFR-001: T020 (covered)
```

**Benefits:**
- Verify all requirements have implementing tasks
- Identify orphan requirements before implementation
- Support compliance/audit requirements

**Effort:** Medium (2-3 days)

---

## 5. Spec Dependencies Field

**Why:** Features often depend on other features (e.g., "user profile" depends on "user auth"), but this isn't captured.

**Current state:**
```yaml
# 008-user-profile/spec.yaml
feature:
  branch: "008-user-profile"
  # No indication this depends on 007-user-auth
```

**Proposed:**
```yaml
# 008-user-profile/spec.yaml
feature:
  branch: "008-user-profile"
  depends_on_specs: ["007-user-auth"]  # Must be complete first
```

**Validation:**
- Warning if dependent spec not found
- Warning if dependent spec not Completed
- Error on `autospec implement` if dependencies incomplete

**Display in `autospec list --deps`:**
```
NUM  NAME               STATUS      DEPENDS ON
007  user-auth          complete    -
008  user-profile       pending     007-user-auth ✓
009  admin-dashboard    pending     007-user-auth ✓, 008-user-profile ✗
```

**Benefits:**
- Prevents implementing features with unmet prerequisites
- Documents feature relationships
- Enables dependency-aware scheduling

**Effort:** Medium (2 days)

---

## 6. Plan Risk Assessment Section

**Why:** Risks identified during planning aren't captured. They exist in the planner's head but not in artifacts.

**Current state:**
- plan.yaml has technical_context, project_structure, data_model
- No place for "this might be tricky because..."

**Proposed:**
```yaml
# In plan.yaml
risks:
  - id: "RISK-001"
    description: "OAuth provider rate limits may cause test flakiness"
    likelihood: "medium"
    impact: "medium"
    mitigation: "Use mock OAuth server for tests; real provider only in E2E"

  - id: "RISK-002"
    description: "Database migration may lock tables during deploy"
    likelihood: "low"
    impact: "high"
    mitigation: "Use online DDL; schedule deploy during low-traffic window"
```

**Validation:**
- Warning if high-impact risks have no mitigation
- Warning if >5 high-likelihood risks (plan may need revision)

**Benefits:**
- Forces explicit risk acknowledgment
- Creates audit trail of known risks
- Helps implementation anticipate issues

**Effort:** Low-Medium (1-2 days)

---

## 7. ~~NeedsReview Task Status~~ - REJECTED

**Rejected because:**
- Overcomplicates the status enum
- `Blocked` + `blocked_reason` + `notes` already captures this use case
- The semantic difference is clear from reading the text:
  - `blocked_reason: "Waiting for API spec"` → external
  - `blocked_reason: "Uncertain about edge cases, need review"` → needs help
- KISS: don't add enum values when text fields suffice

---

## 8. Phase Prerequisites Validation

**Why:** Phases may require external setup not captured in YAML (database running, env vars set, etc.).

**Current state:**
- Phases just list tasks
- No way to specify "before starting this phase, ensure X"

**Proposed:**
```yaml
phases:
  - number: 2
    title: "Database Integration"
    prerequisites:
      - description: "PostgreSQL database running"
        check_command: "pg_isready -h localhost"
      - description: "Migrations applied"
        check_command: "autospec db status | grep -q 'up to date'"
      - description: "OAUTH_SECRET environment variable set"
        check_command: "test -n \"$OAUTH_SECRET\""
    tasks:
      - id: "T005"
        # ...
```

**Behavior in `--phases` mode:**
- Before starting phase, run all check_commands
- If any fail, block phase and display failed prerequisites
- Skip check_commands if not provided (optional)

**Benefits:**
- Catches environment issues before wasting tokens
- Self-documenting setup requirements
- Enables CI/CD integration checks

**Effort:** Medium (2-3 days)

---

## 9. ~~Task Output Artifacts Field~~ - REJECTED

**Rejected because:**
- Multiple tasks touch the same files (not 1:1 mapping)
- Tasks modify existing code, not just create new files
- Git diff verification is weak ("did anything change?" isn't meaningful)
- Better to rely on test suite passing and existing validation

---

## 10. Validation Severity Configuration

**Why:** All validation warnings are treated equally, but some matter more than others depending on the team.

**Current state:**
- Warnings are warnings, errors are errors
- No way to customize which warnings block workflows
- `--strict` would treat ALL warnings as errors (too coarse)

**Proposed config:**
```yaml
# .autospec/config.yml
validation:
  severity:
    blocked_reason_missing: warn      # Default
    orphan_requirement: error         # Treat as error for this project
    high_risk_no_mitigation: error    # Must have mitigation
    missing_complexity: ignore        # Don't care about this
    high_retry_count: warn            # Default
```

**Behavior:**
- `ignore`: Suppress the warning entirely
- `warn`: Show warning but continue (default)
- `error`: Treat as error (non-zero exit)

**CLI flag:**
```bash
autospec artifact tasks.yaml --strict  # All warnings become errors
```

**Benefits:**
- Teams enforce project-specific quality gates
- Gradual adoption: start with warnings, promote to errors
- CI/CD can enforce stricter rules than local dev

**Effort:** Medium (2-3 days)

---

## Implementation Priority Matrix

| Priority | Improvement | Rationale |
|----------|-------------|-----------|
| **P0** | Orchestrator Schema Validation | Catches Claude's schema errors before next stage |
| **P1** | Task Priority Field | Enables smarter retry/execution ordering |
| **P1** | Task Notes Field | Low effort, high value for complex tasks - **DONE!** |
| **P2** | Requirement Traceability | Important for completeness verification |
| **P2** | Validation Severity Config | Enables team-specific quality gates |
| ~~P1~~ | ~~NeedsReview Status~~ | REJECTED - Blocked + blocked_reason + notes suffices |
| **P3** | Task Complexity | Nice for estimation, lower immediate value |
| **P3** | Spec Dependencies | Multi-feature projects only |
| **P3** | Plan Risk Assessment | Planning enhancement |
| **P3** | Phase Prerequisites | Advanced orchestration |
| ~~P3~~ | ~~Task Output Artifacts~~ | REJECTED - multiple tasks touch same files |

---

## Design Principles

These improvements follow the patterns established by `blocked_reason`:

1. **Optional by default** - New fields don't break existing YAML files
2. **Validation warnings, not errors** - Guide toward best practices without blocking
3. **Incremental value** - Each feature useful independently
4. **Schema-first** - Define the YAML structure, then build tooling around it
5. **Display in existing commands** - Integrate into `autospec st`, `autospec artifact`, etc.

---

## Related Files

When implementing these features, modify:

| Feature | Files to Modify |
|---------|-----------------|
| TaskItem fields | `internal/validation/tasks_yaml.go`, `internal/validation/artifact_tasks.go` |
| Spec fields | `internal/validation/artifact_spec.go` |
| Plan fields | `internal/validation/artifact_plan.go` |
| Display | `internal/cli/status.go`, `internal/validation/phase_display.go` |
| Config | `internal/config/config.go` |
| Docs | `docs/YAML-STRUCTURED-OUTPUT.md` |
