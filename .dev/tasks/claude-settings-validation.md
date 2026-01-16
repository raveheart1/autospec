# Claude Code Settings Validation

**Status:** Proposed
**Priority:** High
**Related:** `autospec init`, `autospec doctor`

---

## Problem Statement

When autospec spawns Claude Code in `-p` (print/non-interactive) mode, Claude needs explicit permissions to run bash commands like `autospec`. Without `Bash(autospec:*)` in the allowlist, the workflow fails silently or Claude refuses to execute the command.

**Current behavior:**
- User runs `autospec init` - no Claude settings validation
- User runs `autospec run -a "feature"` - spawns Claude with `-p` flag
- Claude tries to run `autospec` commands but lacks permission
- Workflow fails because `--dangerously-skip-permissions` is NOT enabled by default (and shouldn't be)

**Expected behavior:**
- `autospec init` automatically configures Claude Code permissions
- `autospec doctor` validates Claude settings and warns if misconfigured
- Users don't need to manually edit JSON files

---

## Required Claude Code Permissions

For autospec to work properly, Claude needs this permission:

```json
{
  "permissions": {
    "allow": [
      "Bash(autospec:*)"
    ]
  }
}
```

**Why this permission:**
| Permission | Reason |
|------------|--------|
| `Bash(autospec:*)` | Allows Claude to run any autospec command |

**Note:** `./bin/autospec` is only used during local development. End users install to PATH, so only `Bash(autospec:*)` is needed.

**Optional but recommended:**
```json
{
  "sandbox": {
    "enabled": true,
    "autoAllowBashIfSandboxed": true
  }
}
```

---

## Configuration Locations

Claude Code settings can exist at multiple levels:

| Level | Path | Scope |
|-------|------|-------|
| Project | `.claude/settings.local.json` | This project only |
| Project (shared) | `.claude/settings.json` | This project, committed to repo |
| User | `~/.claude/settings.json` | All projects for this user |

**Precedence:** Project settings.local.json > Project settings.json > User settings.json

---

## Edge Cases to Handle

### 1. No `.claude/` directory exists
- **Scenario:** Fresh project, never used Claude Code
- **Action:** Create `.claude/` directory and `settings.local.json`
- **Prompt:** No (we're setting up autospec, this is expected)

### 2. `.claude/settings.local.json` doesn't exist
- **Scenario:** `.claude/` exists but no local settings
- **Action:** Create `settings.local.json` with required permissions
- **Prompt:** No (project-level, doesn't affect other projects)

### 3. `.claude/settings.local.json` exists but missing permissions
- **Scenario:** Has other settings, needs our permissions added
- **Action:** Merge our permissions into existing `allow` array
- **Prompt:** No (additive change, doesn't remove anything)

### 4. User-level `~/.claude/settings.json` missing permissions
- **Scenario:** User has global settings but missing autospec permissions
- **Action:** Prompt Y/n before modifying (affects all projects)
- **Alternative:** Just warn and show manual instructions

### 5. Not in a git project directory
- **Scenario:** User runs `autospec init` outside of git repo
- **Action:** Already handled - autospec requires git repo
- **Note:** Doctor should still check Claude CLI availability

### 6. Permissions already present
- **Scenario:** User already has `Bash(autospec:*)` configured
- **Action:** Skip modification, show "already configured" message
- **Detection:** Check if permission string exists in `allow` array

### 7. Conflicting permissions in `deny` array
- **Scenario:** User has `Bash(autospec:*)` in deny list
- **Action:** Warn user, do not auto-modify
- **Note:** This is unusual but possible

---

## Implementation Plan

### Phase 1: Add Claude Settings Package

Create `internal/claude/settings.go`:

```go
package claude

// Settings represents Claude Code settings.json structure
type Settings struct {
    Permissions *Permissions `json:"permissions,omitempty"`
    Sandbox     *Sandbox     `json:"sandbox,omitempty"`
}

type Permissions struct {
    Allow []string `json:"allow,omitempty"`
    Ask   []string `json:"ask,omitempty"`
    Deny  []string `json:"deny,omitempty"`
}

type Sandbox struct {
    Enabled                bool `json:"enabled,omitempty"`
    AutoAllowBashIfSandboxed bool `json:"autoAllowBashIfSandboxed,omitempty"`
}

// RequiredPermissions returns permissions needed for autospec
func RequiredPermissions() []string {
    return []string{
        "Bash(autospec:*)",
    }
}

// ProjectSettingsPath returns path to project-level settings
func ProjectSettingsPath() string {
    return ".claude/settings.local.json"
}

// UserSettingsPath returns path to user-level settings
func UserSettingsPath() (string, error) {
    home, err := os.UserHomeDir()
    if err != nil {
        return "", err
    }
    return filepath.Join(home, ".claude", "settings.json"), nil
}

// Load reads and parses Claude settings from path
func Load(path string) (*Settings, error)

// Save writes settings to path (pretty-printed JSON)
func Save(path string, settings *Settings) error

// HasPermission checks if permission exists in allow list
func (s *Settings) HasPermission(perm string) bool

// AddPermission adds permission to allow list if not present
func (s *Settings) AddPermission(perm string) bool

// IsDenied checks if permission is in deny list
func (s *Settings) IsDenied(perm string) bool

// MissingPermissions returns required permissions not in allow list
func (s *Settings) MissingPermissions() []string
```

### Phase 2: Integrate with `autospec init`

Modify `internal/cli/init.go`:

```go
func runInit(cmd *cobra.Command, args []string) error {
    // ... existing code ...

    // NEW: Configure Claude Code settings
    if err := configureClaudeSettings(out, project); err != nil {
        return fmt.Errorf("configuring Claude settings: %w", err)
    }

    // ... rest of existing code ...
}

func configureClaudeSettings(out io.Writer, projectLevel bool) error {
    settingsPath := claude.ProjectSettingsPath()

    // Load existing settings or create new
    settings, err := claude.Load(settingsPath)
    if os.IsNotExist(err) {
        settings = &claude.Settings{}
    } else if err != nil {
        return err
    }

    // Check for denied permissions (warn, don't modify)
    for _, perm := range claude.RequiredPermissions() {
        if settings.IsDenied(perm) {
            fmt.Fprintf(out, "⚠ Claude settings: %s is in deny list\n", perm)
            fmt.Fprintf(out, "  Remove it from %s to allow autospec to work\n", settingsPath)
            return nil
        }
    }

    // Add missing permissions
    missing := settings.MissingPermissions()
    if len(missing) == 0 {
        fmt.Fprintf(out, "✓ Claude settings: permissions configured\n")
        return nil
    }

    // Add permissions
    for _, perm := range missing {
        settings.AddPermission(perm)
    }

    // Ensure directory exists
    if err := os.MkdirAll(filepath.Dir(settingsPath), 0755); err != nil {
        return err
    }

    // Save settings
    if err := claude.Save(settingsPath, settings); err != nil {
        return err
    }

    if len(missing) == 1 {
        fmt.Fprintf(out, "✓ Claude settings: added permission to %s\n", settingsPath)
    } else {
        fmt.Fprintf(out, "✓ Claude settings: added %d permissions to %s\n",
            len(missing), settingsPath)
    }
    return nil
}
```

### Phase 3: Add Health Check to `autospec doctor`

Modify `internal/health/health.go`:

```go
func RunHealthChecks() *HealthReport {
    // ... existing checks ...

    // NEW: Check Claude settings
    claudeSettingsCheck := CheckClaudeSettings()
    report.Checks = append(report.Checks, claudeSettingsCheck)
    if !claudeSettingsCheck.Passed {
        report.Passed = false
    }

    return report
}

func CheckClaudeSettings() CheckResult {
    settings, err := claude.Load(claude.ProjectSettingsPath())
    if os.IsNotExist(err) {
        return CheckResult{
            Name:    "Claude Settings",
            Passed:  false,
            Message: "Claude settings not configured - run 'autospec init'",
        }
    }
    if err != nil {
        return CheckResult{
            Name:    "Claude Settings",
            Passed:  false,
            Message: fmt.Sprintf("Failed to read Claude settings: %v", err),
        }
    }

    missing := settings.MissingPermissions()
    if len(missing) > 0 {
        return CheckResult{
            Name:    "Claude Settings",
            Passed:  false,
            Message: fmt.Sprintf("Missing permissions: %s - run 'autospec init'",
                strings.Join(missing, ", ")),
        }
    }

    return CheckResult{
        Name:    "Claude Settings",
        Passed:  true,
        Message: "Claude settings configured with required permissions",
    }
}
```

### Phase 4: User-Level Settings (Optional Enhancement)

For user-level settings, add optional flag to `autospec init`:

```bash
autospec init --user-settings    # Also configure ~/.claude/settings.json
```

Behavior:
- Check if user settings exist
- If missing autospec permissions, prompt: "Add autospec permissions to ~/.claude/settings.json? [y/N]"
- Only modify if user confirms
- This is optional because project-level settings are sufficient

---

## UX Flow

### `autospec init` (default)

```
$ autospec init

✓ Commands: 8 installed, 0 updated → .claude/commands/
✓ Config: created at ~/.config/autospec/config.yml
✓ Claude settings: added permission to .claude/settings.local.json
✓ Constitution: found at .autospec/memory/constitution.yaml

Quick start:
  1. autospec specify "Add user authentication"
  ...
```

### `autospec init` (already configured)

```
$ autospec init

✓ Commands: up to date
✓ Config: exists at ~/.config/autospec/config.yml
✓ Claude settings: permissions configured
✓ Constitution: found at .autospec/memory/constitution.yaml
```

### `autospec doctor` (misconfigured)

```
$ autospec doctor

✓ Claude CLI found
✓ Git found
✗ Error: Claude settings missing permission: Bash(autospec:*)
  → Run 'autospec init' to fix
```

### `autospec doctor` (all good)

```
$ autospec doctor

✓ Claude CLI found
✓ Git found
✓ Claude settings configured
```

---

## Testing Strategy

### Unit Tests

1. **Settings parsing:**
   - Load valid settings.json
   - Load empty settings.json
   - Handle missing file
   - Handle malformed JSON

2. **Permission checking:**
   - HasPermission with exact match
   - HasPermission with wildcard patterns
   - MissingPermissions calculation
   - IsDenied detection

3. **Settings modification:**
   - AddPermission to empty allow list
   - AddPermission to existing list (no duplicates)
   - Preserve existing permissions
   - Preserve other settings (sandbox, ask, deny)

### Integration Tests

1. **Init command:**
   - Creates `.claude/settings.local.json` if missing
   - Merges permissions into existing file
   - Doesn't duplicate existing permissions
   - Preserves existing settings structure

2. **Doctor command:**
   - Detects missing settings file
   - Detects missing permissions
   - Passes when properly configured

---

## Security Considerations

1. **Principle of least privilege:**
   - Only add the minimum required permissions
   - Don't enable sandbox bypass
   - Don't add overly broad permissions

2. **No user-level modifications without consent:**
   - Project-level changes are safe (only affect this project)
   - User-level changes require explicit confirmation

3. **Respect existing deny rules:**
   - If user explicitly denied a permission, don't override
   - Warn and provide instructions instead

---

## Open Questions

1. **Should we add sandbox configuration?**
   - `sandbox.enabled: true` is recommended but not required
   - `autoAllowBashIfSandboxed: true` is convenience, not required
   - Recommendation: Don't touch sandbox settings, let user decide

2. **Should we add to `.gitignore`?**
   - `.claude/settings.local.json` contains local permissions
   - Some teams might want to share settings
   - Recommendation: Don't auto-add, but check and suggest like we do for `.autospec/`

3. **What about `Bash(make:*)` and other build commands?**
   - autospec might need to run tests, builds, etc.
   - These vary by project
   - Recommendation: Start minimal, expand based on feedback

---

## Files to Create/Modify

| File | Action |
|------|--------|
| `internal/claude/settings.go` | Create - Claude settings management |
| `internal/claude/settings_test.go` | Create - Unit tests |
| `internal/cli/init.go` | Modify - Add Claude settings configuration |
| `internal/health/health.go` | Modify - Add Claude settings check |
| `internal/health/health_test.go` | Modify - Add tests for new check |

---

## References

- [Claude Code Settings](https://docs.anthropic.com/en/docs/claude-code/settings)
- [docs/claude-settings.md](../../docs/claude-settings.md) - Project documentation on settings
