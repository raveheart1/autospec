# Fix 4: Sandbox Pre-Approval Documentation (MEDIUM)

**Problem:** 90 sandbox workarounds across 6 sessions. Go build/test commands consistently require `dangerouslyDisableSandbox: true`.

**Token Savings:** 2-5K/session
**Effort:** Low

## Command

```bash
autospec specify <<'EOF'
Document sandbox pre-approval configuration for Go commands in CLAUDE.md and docs/claude-settings.md. This eliminates 90+ sandbox workaround prompts per 6 sessions.

## Problem Statement
Go build and test commands consistently fail in Claude Code sandbox:
- 'go build' requires GOCACHE workaround or sandbox disable
- 'go test' requires sandbox disable
- 'make build' and 'make test' require sandbox disable
- Average 15 sandbox prompts per implement session

Users must repeatedly approve sandbox overrides, adding friction and wasting tokens on error/retry cycles.

## Required Changes

### 1. Update CLAUDE.md

Add a 'Sandbox Configuration' section recommending these allowlist entries:

```markdown
## Sandbox Configuration

To avoid repeated sandbox prompts during Go development, add these to your Claude Code settings:

```json
{
  "permissions": {
    "allow": [
      "Bash(go build:*)",
      "Bash(go test:*)",
      "Bash(make build:*)",
      "Bash(make test:*)",
      "Bash(make fmt:*)",
      "Bash(make lint:*)",
      "Bash(GOCACHE=/tmp/claude/go-cache go build:*)",
      "Bash(GOCACHE=/tmp/claude/go-cache go test:*)",
      "Bash(GOCACHE=/tmp/claude/go-cache make:*)"
    ]
  }
}
```

These commands are safe for auto-approval as they only read source files and write to designated output directories.
```

### 2. Update docs/claude-settings.md

Add detailed explanation of each permission and why it is safe:

- go build: Compiles Go code, writes to ./bin or current directory
- go test: Runs tests, writes coverage files to designated locations
- make targets: Wrapper commands that invoke go build/test
- GOCACHE variant: Explicit cache directory for sandbox compatibility

### 3. Add to internal/commands/autospec.implement.md

Add note about expected sandbox behavior:

```markdown
## Sandbox Notes

Go build and test commands may trigger sandbox prompts. Recommended pre-approvals:
- 'Bash(go build:*)'
- 'Bash(go test:*)'
- 'Bash(make build:*)'
- 'Bash(make test:*)'

If sandbox errors occur, the GOCACHE workaround usually resolves them:
'GOCACHE=/tmp/claude/go-cache go build ./...'
```

## Acceptance Criteria
- [ ] CLAUDE.md contains Sandbox Configuration section
- [ ] docs/claude-settings.md documents each permission
- [ ] implement.md includes Sandbox Notes
- [ ] Permission list covers all common Go development commands
- [ ] Explanation of why each permission is safe

## Non-Functional Requirements
- Documentation only, no code changes
- JSON examples must be valid and copy-pasteable
- Include both standard and GOCACHE variants
EOF
```
