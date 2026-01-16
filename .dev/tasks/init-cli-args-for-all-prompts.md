# Plan: CLI Arguments for All `autospec init` Prompts

Add CLI flags to `autospec init` so every interactive prompt can be bypassed for fully non-interactive/automated usage.

## Current State

### Existing Flags
| Flag | Purpose |
|------|---------|
| `-p, --project` | Create project-level config (`.autospec/config.yml`) |
| `-f, --force` | Overwrite existing config |
| `--ai <agents>` | Specify agents (e.g., `--ai claude,opencode`) |
| `--no-agents` | Skip agent configuration entirely |
| `--here` | Initialize in current directory |

### Missing Flag Coverage

| # | Prompt | Default | Missing Flag |
|---|--------|---------|--------------|
| 1 | "Configure Claude sandbox for autospec?" | Yes | `--sandbox` / `--no-sandbox` |
| 2 | "Use API key for billing?" | No | `--use-subscription` / `--no-use-subscription` |
| 3 | "Enable skip_permissions (recommended)?" | Yes | `--skip-permissions` / `--no-skip-permissions` |
| 4 | "Add to .gitignore?" | No | `--gitignore` / `--no-gitignore` |
| 5 | "Create constitution?" | Yes | `--constitution` / `--no-constitution` |

## Proposed Flags

### 1. `--sandbox` / `--no-sandbox`
- **Purpose**: Auto-configure or skip Claude sandbox setup
- **Default behavior** (no flag): Prompt user
- **`--sandbox`**: Auto-configure sandbox (enable + add paths)
- **`--no-sandbox`**: Skip sandbox configuration entirely
- **Affects**: `handleSandboxConfiguration()` in `init_cmd.go:559-624`

### 2. `--use-subscription` / `--no-use-subscription`
- **Purpose**: Control OAuth vs API key billing preference
- **Default behavior** (no flag): Prompt user (if conditions met)
- **`--use-subscription`**: Use OAuth/subscription billing
- **`--no-use-subscription`**: Use API key billing
- **Affects**: `handleClaudeAuthDetection()` in `init_cmd.go:667-674`
- **Note**: Only relevant when OAuth is not detected but API key is set

### 3. `--skip-permissions` / `--no-skip-permissions`
- **Purpose**: Enable/disable autonomous mode (skip permission prompts in Claude)
- **Default behavior** (no flag): Prompt user
- **`--skip-permissions`**: Enable autonomous mode
- **`--no-skip-permissions`**: Disable autonomous mode (more interactive)
- **Affects**: `handleSkipPermissionsPrompt()` in `init_cmd.go:798-867`

### 4. `--gitignore` / `--no-gitignore`
- **Purpose**: Auto-add `.autospec/` to `.gitignore`
- **Default behavior** (no flag): Prompt user
- **`--gitignore`**: Add `.autospec/` to `.gitignore`
- **`--no-gitignore`**: Skip gitignore modification
- **Affects**: `collectPendingActions()` in `init_cmd.go:1374`

### 5. `--constitution` / `--no-constitution`
- **Purpose**: Auto-create or skip constitution generation
- **Default behavior** (no flag): Prompt user
- **`--constitution`**: Create constitution (runs Claude session)
- **`--no-constitution`**: Skip constitution creation
- **Affects**: `collectPendingActions()` in `init_cmd.go:1386`

## Implementation Tasks

### Phase 1: Add Flag Definitions

1. Add flag variables to `initCmd` in `init_cmd.go`:
   ```go
   var (
       sandboxFlag           *bool  // nil = prompt, true = auto-configure, false = skip
       useSubscriptionFlag   *bool  // nil = prompt, true = use subscription, false = use API key
       skipPermissionsFlag   *bool  // nil = prompt, true = enable, false = disable
       gitignoreFlag         *bool  // nil = prompt, true = add, false = skip
       constitutionFlag      *bool  // nil = prompt, true = create, false = skip
   )
   ```

2. Register flags in command setup (~line 94-104):
   ```go
   // Use NoOptDefVal for boolean flags that can be nil/unset
   cmd.Flags().Bool("sandbox", false, "Auto-configure Claude sandbox")
   cmd.Flags().Bool("no-sandbox", false, "Skip sandbox configuration")
   cmd.Flags().Bool("use-subscription", false, "Use OAuth/subscription billing")
   cmd.Flags().Bool("no-use-subscription", false, "Use API key billing")
   cmd.Flags().Bool("skip-permissions", false, "Enable skip_permissions (autonomous mode)")
   cmd.Flags().Bool("no-skip-permissions", false, "Disable skip_permissions")
   cmd.Flags().Bool("gitignore", false, "Add .autospec/ to .gitignore")
   cmd.Flags().Bool("no-gitignore", false, "Skip .gitignore modification")
   cmd.Flags().Bool("constitution", false, "Create constitution automatically")
   cmd.Flags().Bool("no-constitution", false, "Skip constitution creation")
   ```

### Phase 2: Implement Flag Resolution

1. Create helper to resolve paired flags:
   ```go
   // resolveBoolFlag returns nil if neither flag set, true if positive flag, false if negative flag
   func resolveBoolFlag(cmd *cobra.Command, positive, negative string) (*bool, error) {
       posSet := cmd.Flags().Changed(positive)
       negSet := cmd.Flags().Changed(negative)
       if posSet && negSet {
           return nil, fmt.Errorf("--%s and --%s are mutually exclusive", positive, negative)
       }
       if posSet {
           t := true
           return &t, nil
       }
       if negSet {
           f := false
           return &f, nil
       }
       return nil, nil // Neither set, will prompt
   }
   ```

2. Resolve flags at start of `runInit()`:
   ```go
   sandboxChoice, err := resolveBoolFlag(cmd, "sandbox", "no-sandbox")
   // ... etc for other flags
   ```

### Phase 3: Update Prompt Functions

1. **`handleSandboxConfiguration()`** (~line 559):
   - Accept `sandboxChoice *bool` parameter
   - If `sandboxChoice != nil`, skip prompt and use value
   - If `*sandboxChoice == true`, auto-configure
   - If `*sandboxChoice == false`, skip entirely

2. **`handleClaudeAuthDetection()`** (~line 667):
   - Accept `useSubscriptionChoice *bool` parameter
   - If set, skip prompt and use value directly

3. **`handleSkipPermissionsPrompt()`** (~line 798):
   - Accept `skipPermissionsChoice *bool` parameter
   - If set, skip prompt and configure accordingly

4. **`collectPendingActions()`** (~line 1362):
   - Accept `gitignoreChoice, constitutionChoice *bool` parameters
   - If set, skip respective prompts

### Phase 4: Update Non-Interactive Error Handling

Currently, init fails with error if stdin is not a terminal and `--no-agents` is not provided. With all prompts having flag overrides, update logic:

```go
// If all prompts have flag overrides, allow non-interactive execution
if !isInteractive && !allPromptsHaveOverrides(flags) {
    return fmt.Errorf("non-interactive mode requires flags for all prompts; missing: %s", missingFlags)
}
```

### Phase 5: Add Tests

1. **Unit tests** for flag resolution:
   - Test mutually exclusive flag detection
   - Test nil behavior when no flags set

2. **Integration tests** for non-interactive init:
   ```go
   func TestInitNonInteractive(t *testing.T) {
       // Test: autospec init --no-agents --no-sandbox --no-constitution --no-gitignore
       // Should succeed without any prompts
   }
   ```

3. **Test each flag pair**:
   - `--sandbox` configures sandbox correctly
   - `--no-sandbox` skips sandbox
   - Error when both `--sandbox` and `--no-sandbox` provided

### Phase 6: Update Documentation

1. Update `docs/public/reference.md` with new flags
2. Update help text in command
3. Add examples for CI/CD non-interactive usage:
   ```bash
   # Fully non-interactive init for CI
   autospec init --project --ai claude --sandbox --skip-permissions --no-gitignore --no-constitution
   ```

## Testing Checklist

- [ ] `autospec init --help` shows all new flags
- [ ] `autospec init --sandbox` auto-configures sandbox without prompt
- [ ] `autospec init --no-sandbox` skips sandbox without prompt
- [ ] `autospec init --sandbox --no-sandbox` returns error
- [ ] `autospec init --use-subscription` sets config without prompt
- [ ] `autospec init --skip-permissions` enables autonomous mode
- [ ] `autospec init --no-skip-permissions` disables autonomous mode
- [ ] `autospec init --gitignore` adds to .gitignore without prompt
- [ ] `autospec init --no-gitignore` skips gitignore without prompt
- [ ] `autospec init --constitution` creates constitution without prompt
- [ ] `autospec init --no-constitution` skips constitution without prompt
- [ ] Fully non-interactive init works: `autospec init --ai claude --no-sandbox --no-constitution --no-gitignore --skip-permissions`
- [ ] All existing tests still pass
- [ ] `make lint` passes
- [ ] `make build` passes

## Example Usage After Implementation

```bash
# CI/CD: Minimal non-interactive setup
autospec init --project --ai claude --no-sandbox --no-constitution --no-gitignore

# CI/CD: Full autonomous setup
autospec init --project --ai claude --sandbox --skip-permissions --no-gitignore --constitution

# Local dev: Quick setup with defaults
autospec init --ai claude --sandbox --skip-permissions --gitignore --constitution

# Scripted: Specific billing preference
autospec init --ai claude --use-subscription --skip-permissions
```

## Files to Modify

| File | Changes |
|------|---------|
| `internal/cli/config/init_cmd.go` | Add flags, update prompt functions |
| `internal/cli/config/init_test.go` | Add tests for new flags |
| `docs/public/reference.md` | Document new flags |

## Priority

Medium-High: Enables CI/CD automation and scripted setups.
