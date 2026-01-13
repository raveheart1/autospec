# Bug: doctor command doesn't check global Claude settings

## Issue

`autospec doctor` reports missing `Bash(autospec:*)` permission even when it's configured in the user-level global settings.

## Root Cause

- `autospec init` was changed to update **user-level** settings at `~/.claude/settings.json`
- `autospec doctor` still only checks **project-level** settings at `.claude/settings.local.json`
- This mismatch causes false negatives

## Reproduction

```bash
# Permissions ARE set in global settings
grep -E "autospec|specs" ~/.claude/settings.json
#       "Bash(autospec:*)",
#       "Write(.autospec/**)",
#       "Edit(.autospec/**)",
#       "Write(./specs/**)",
#       "Edit(./specs/**)"

# But doctor still fails
autospec doctor
# Output: ✗ Claude settings: missing Bash(autospec:*) permission (run 'autospec init' to fix)
```

## Fix Required

Update `CheckClaudeSettingsInDir` in `internal/health/health.go` to:

1. Check project-level settings (`.claude/settings.local.json`) first
2. If permission found, pass
3. If not found or missing permission, also check global settings (`~/.claude/settings.json`)
4. Only fail if NEITHER location has the permission
5. Update test cases for the new behavior

## Files to Modify

- `internal/health/health.go` - `CheckClaudeSettingsInDir` function
- `internal/health/health_test.go` - Add test cases for global settings

## Notes

- The `claude` package already has `LoadGlobal()` which loads from `~/.claude/settings.json`
- Consider whether to skip this check entirely for non-claude agents (e.g., opencode)
- User workaround: `autospec init --project` writes to project-level settings which doctor currently checks
