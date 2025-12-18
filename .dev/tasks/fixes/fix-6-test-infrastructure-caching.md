# Fix 6: Test Infrastructure Caching (MEDIUM)

**Problem:** Mock infrastructure (mock-claude.sh, MockClaudeExecutor, test helpers) rediscovered in every implement phase.

**Token Savings:** Variable
**Effort:** Medium

## Command

```bash
autospec specify <<'EOF'
Add test infrastructure caching via spec-level notes.yaml artifact. This eliminates repeated discovery of mock scripts, test helpers, and test patterns across phases.

## Problem Statement
Each implement phase rediscovers the same test infrastructure:
- Location of mock-claude.sh (mocks/scripts/ vs tests/mocks/)
- MockClaudeExecutor in mocks_test.go
- Test helper functions (newTestOrchestratorWithSpecName, writeTestSpec)
- Coverage baseline and target functions

This discovery is repeated in EVERY phase of EVERY implement session.

## Required Changes

### 1. Create new artifact type: notes.yaml

Add validation for optional notes.yaml in specs/<feature>/:

```yaml
# specs/043-workflow-mock-coverage/notes.yaml
_discovered:
  test_infrastructure:
    mock_claude_script: 'mocks/scripts/mock-claude.sh'
    mock_executor: 'internal/workflow/mocks_test.go:MockClaudeExecutor'
    test_helpers:
      - name: 'newTestOrchestratorWithSpecName'
        file: 'internal/workflow/workflow_test.go'
        line: 3296
      - name: 'writeTestTasks'
        file: 'internal/workflow/workflow_test.go'
        line: 3448
  large_files:
    - path: 'internal/workflow/workflow_test.go'
      strategy: 'grep for function names, use offset/limit reads'
  coverage:
    baseline: '79.4%'
    target: '85%'
    zero_coverage_functions:
      - 'PromptUserToContinue:preflight.go:117'
      - 'runPreflightChecks:workflow.go:217'
```

### 2. Add notes.yaml validation (internal/validation/notes_yaml.go)

Create minimal validation that accepts the structure above. Notes.yaml is informational, not prescriptive, so validation should be permissive.

### 3. Update implement.md template

Add section on using notes.yaml:

```markdown
## Spec Notes (notes.yaml)

If specs/<feature>/notes.yaml exists, read it BEFORE exploring the codebase.

It may contain:
- _discovered.test_infrastructure: Paths to mock scripts and test helpers
- _discovered.large_files: Reading strategies for oversized files
- _discovered.coverage: Baseline metrics and target functions

This cache saves rediscovery time across phases.
```

### 4. Update Phase 1 guidance

In implement.md, add Phase 1 responsibility:

```markdown
## Phase 1 Responsibilities

After completing Phase 1 setup tasks:
1. Create or update specs/<feature>/notes.yaml
2. Document discovered test infrastructure paths
3. Note any large files and reading strategies
4. Record coverage baseline if measured

This benefits subsequent phases.
```

### 5. Bundle notes.yaml in phase context

Update phase context generation to include notes.yaml content if it exists.

## Acceptance Criteria
- [ ] notes.yaml validation exists and is permissive
- [ ] implement.md documents notes.yaml usage
- [ ] Phase 1 guidance includes notes.yaml creation
- [ ] Phase context generation bundles notes.yaml
- [ ] Example notes.yaml structure documented

## Non-Functional Requirements
- notes.yaml is OPTIONAL - not required for workflow
- Validation does not enforce specific fields
- Backward compatible with specs lacking notes.yaml
- notes.yaml can be created manually or by Claude during Phase 1
EOF
```
