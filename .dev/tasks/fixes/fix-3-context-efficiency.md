# Fix 3: Context Efficiency Guidance in specify.md and plan.md (HIGH)

**Problem:** specify and plan commands also exhibit file re-reading, though less severe than implement. workflow.go read 16 times in one plan session.

**Token Savings:** 10-20K/session
**Effort:** Low

## Command

```bash
autospec specify <<'EOF'
Add context efficiency guidance to internal/commands/autospec.specify.md and internal/commands/autospec.plan.md templates. This reduces file re-reading during specification and planning phases.

## Problem Statement
While less severe than implement sessions, specify and plan sessions also show inefficiency:
- workflow.go read 16 times in one plan session
- executor_test.go read 9 times
- Multiple files read 3-4 times during codebase exploration

## Required Changes

### 1. Add to autospec.specify.md

Add 'Context Efficiency' section after the codebase exploration guidance:

```markdown
## Context Efficiency

When analyzing the codebase for specification:

1. **Grep Before Read**: Use Grep to locate patterns BEFORE reading entire files
2. **Read Sections**: For large files (>500 lines), use offset/limit to read only needed sections
3. **No Re-Reads**: Once you have read a file, DO NOT read it again in this session
4. **Track Mentally**: Keep mental note of files read - they are in your context

### Large File Strategy
For files exceeding 1000 lines:
- Use Grep to find function/class locations
- Read specific sections with offset and limit parameters
- Never attempt to read the entire file
```

### 2. Add to autospec.plan.md

Add similar 'Context Efficiency' section:

```markdown
## Context Efficiency

When exploring code for implementation planning:

1. **Targeted Discovery**: Search for specific patterns rather than reading entire files
2. **Symbol-First**: Use Serena MCP symbol tools when available for structured exploration
3. **Read Once**: Each file should be read at most once during planning
4. **Cache Coverage Data**: If spec.yaml already contains coverage analysis, do not re-run coverage commands

### Coverage Data Reuse
If the spec.yaml non_functional section contains coverage baseline data:
- DO NOT run 'go test -cover' again
- DO NOT run 'go tool cover -func' again
- Use the cached values from spec.yaml
```

## Acceptance Criteria
- [ ] autospec.specify.md contains 'Context Efficiency' section
- [ ] autospec.plan.md contains 'Context Efficiency' section
- [ ] Both include 'Large File Strategy' guidance
- [ ] plan.md includes 'Coverage Data Reuse' guidance
- [ ] Guidance is actionable with clear dos/donts

## Non-Functional Requirements
- Template changes only
- Sections should be concise (<20 lines each)
- Use consistent formatting with implement.md
EOF
```
