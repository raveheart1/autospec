# Issues Found During Manual Testing

Date: 2025-12-19

## Issues

### 1. Deprecation Warnings in User Config
**Status**: ✅ Fixed
**Severity**: Medium
**Location**: `~/.config/autospec/config.yml`

The user config contained deprecated fields that triggered warnings on every command:
- `custom_claude_cmd` → `custom_agent_cmd`
- `claude_cmd` / `claude_args` → `agent_preset: claude`

**Fix Applied**: Updated `~/.config/autospec/config.yml` to use new field names.

---

### 2. State Directory Permissions (Sandbox Issue)
**Status**: ⏭️ Skipped (sandbox limitation)
**Severity**: N/A (not a code bug)
**Location**: `~/.autospec/state/`

In sandboxed environment, `~/.autospec/state/` is read-only, preventing:
- History logging (`history.yaml.tmp`)
- Worktree state persistence (`worktrees.yaml.tmp`)
- Retry state (`retry.json.tmp`)

**Note**: This is a sandbox/environment issue, not a code bug. Works fine outside sandbox.

---

### 3. Worktree Default Path Uses Parent Directory
**Status**: ℹ️ Documentation
**Severity**: Low
**Location**: `internal/cli/worktree/create.go`

Default worktree path is `../<name>` which may be unexpected. Users should use `--path` for custom locations.

**Note**: This is by design but could be clearer in help text.

---

### 4. gen-script Requires Interactive Claude Session
**Status**: ℹ️ Expected
**Severity**: N/A
**Location**: `internal/cli/worktree/gen_script.go`

The `autospec worktree gen-script` command requires Claude to execute, which needs an interactive session.

**Note**: Expected behavior - requires manual testing.

---

## Summary

| # | Issue | Status | Action |
|---|-------|--------|--------|
| 1 | Deprecation warnings | 🔧 Fixing | Update user config |
| 2 | State dir read-only | ⏭️ Skip | Sandbox limitation |
| 3 | Default worktree path | ℹ️ Doc | By design |
| 4 | gen-script interactive | ℹ️ Doc | Expected |
