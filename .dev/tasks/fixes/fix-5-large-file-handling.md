# Fix 5: Large File Handling Strategy (MEDIUM)

**Problem:** workflow_test.go (45K+ tokens) exceeds read limits, causing multiple grep→partial-read cycles.

**Token Savings:** 5-10K/session
**Effort:** Medium

## Command

```bash
autospec specify <<'EOF'
Add large file handling hints to tasks.yaml schema and implement.md template. This provides reading strategies for files that exceed Claude's token limits.

## Problem Statement
Several files consistently exceed Claude's read limits:
- workflow_test.go: 45K+ tokens, read limit exceeded in 8/20 sessions
- troubleshooting.md: 960 lines exceeds 950 line limit
- Large test files require multiple offset/limit reads

Without guidance, Claude attempts to read these files fully, fails, then tries grep, then partial reads - wasting tokens on the discovery process.

## Required Changes

### 1. Update tasks.yaml schema (internal/validation/tasks_yaml.go)

Add optional _implementation_hints field to task schema:

```yaml
tasks:
  - id: T001
    # ... existing fields ...

_implementation_hints:
  large_files:
    - path: 'internal/workflow/workflow_test.go'
      size_estimate: '45K tokens'
      strategy: 'grep for function names, read sections with offset/limit'
      key_functions:
        - name: 'newTestOrchestratorWithSpecName'
          line: 3296
        - name: 'writeTestTasks'
          line: 3448
    - path: 'docs/troubleshooting.md'
      size_estimate: '960 lines'
      strategy: 'read sections by topic, use offset/limit'
```

### 2. Update tasks.yaml validation

Modify ValidateTasksYAML to accept but not require _implementation_hints field.

### 3. Update internal/commands/autospec.tasks.md

Add guidance to generate _implementation_hints when large files are identified during task generation:

```markdown
## Large File Detection

When generating tasks that involve files over 500 lines:

1. Note the file in _implementation_hints.large_files
2. Include size_estimate (lines or token estimate)
3. Suggest reading strategy
4. If known, include key function names and line numbers

This helps the implement phase read efficiently without discovery overhead.
```

### 4. Update internal/commands/autospec.implement.md

Add section on using _implementation_hints:

```markdown
## Large File Handling

Check _implementation_hints.large_files in tasks.yaml before reading:

1. If a file is listed with strategy, follow that strategy
2. For files marked 'grep for function names':
   - Use Grep to find function locations first
   - Read only the sections you need with offset/limit
3. For files with key_functions listed:
   - Go directly to those line numbers

### Default Strategy for Unlisted Large Files

If you encounter a file that exceeds read limits:
1. Use Grep to find relevant patterns
2. Read sections with offset/limit (500 lines at a time)
3. Note the file for future _implementation_hints
```

## Acceptance Criteria
- [ ] tasks.yaml schema accepts _implementation_hints field
- [ ] Validation passes with or without _implementation_hints
- [ ] autospec.tasks.md documents large file detection
- [ ] autospec.implement.md documents large file handling
- [ ] Example _implementation_hints structure is documented

## Non-Functional Requirements
- Schema change is backward compatible
- _implementation_hints is optional, not required
- No validation errors for existing tasks.yaml files
- Documentation includes concrete examples
EOF
```
