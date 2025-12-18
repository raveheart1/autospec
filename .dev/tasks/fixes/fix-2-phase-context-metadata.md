# Fix 2: Phase Context Metadata Flags (HIGH)

**Problem:** Claude reads phase context (which bundles spec/plan/tasks) then immediately reads spec.yaml, plan.yaml, tasks.yaml separately. Also checks for non-existent checklists directory 128 times across 6 sessions.

**Root Cause:** Phase context file doesn't explicitly tell Claude what's bundled and what to skip.

**Token Savings:** 5-15K/session
**Effort:** Medium

## Command

```bash
autospec specify <<'EOF'
Add metadata flags to phase context generation in internal/workflow/phase_context.go. This addresses redundant artifact reads (15K tokens/session) and unnecessary checklists checks (128 checks across 6 sessions for non-existent directory).

## Problem Statement
Analysis shows Claude consistently:
1. Reads phase-X.yaml context file (contains bundled spec/plan/tasks)
2. Immediately reads spec.yaml separately (REDUNDANT)
3. Reads plan.yaml separately (REDUNDANT)
4. Reads tasks.yaml separately (REDUNDANT)
5. Checks for checklists/ directory (NEVER EXISTS in most projects)

The phase context header says 'This file bundles spec, plan, and phase-specific tasks' but Claude ignores this because there is no machine-readable metadata.

## Required Changes

### 1. Update PhaseContext struct in internal/workflow/types.go or phase_context.go

Add a ContextMeta field to the phase context YAML structure:

```yaml
_context_meta:
  phase_artifacts_bundled: true    # Signals DO NOT read individual artifacts
  bundled_artifacts:
    - spec.yaml
    - plan.yaml
    - tasks.yaml (phase-filtered)
  has_checklists: false            # Skip checklists directory check
  skip_reads:
    - 'specs/<feature>/spec.yaml'
    - 'specs/<feature>/plan.yaml'
    - 'specs/<feature>/tasks.yaml'
```

### 2. Update phase context generation

Modify the function that generates phase-X.yaml files (likely in internal/workflow/phase_context.go or similar) to:
- Always include _context_meta section at the top of generated files
- Set phase_artifacts_bundled: true
- List the bundled artifacts explicitly
- Check if checklists/ directory exists and set has_checklists accordingly
- Generate skip_reads list with actual paths

### 3. Update implement.md template

Add section explaining _context_meta:

```markdown
## Phase Context Metadata

The phase context file includes a '_context_meta' section:

- 'phase_artifacts_bundled: true' means DO NOT read spec.yaml, plan.yaml, or tasks.yaml separately
- 'has_checklists: false' means DO NOT check for checklists/ directory
- 'skip_reads' lists files that are already bundled - DO NOT read them

If _context_meta.phase_artifacts_bundled is true, you MUST NOT read individual artifact files.
```

## Acceptance Criteria
- [ ] PhaseContext struct includes ContextMeta field
- [ ] Generated phase-X.yaml files include _context_meta section
- [ ] _context_meta.phase_artifacts_bundled is always true for phase contexts
- [ ] _context_meta.has_checklists reflects actual directory existence
- [ ] _context_meta.skip_reads lists bundled artifact paths
- [ ] implement.md template documents _context_meta usage
- [ ] Existing tests pass after changes

## Non-Functional Requirements
- Backward compatible (old phase contexts without _context_meta still work)
- _context_meta section appears at TOP of generated YAML
- Performance: directory existence check <1ms
EOF
```
