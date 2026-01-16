# Manual Test Plan: 099-init-cli-args

Testing CLI flags for non-interactive `autospec init` execution.

## Test Environment
- Date: 2026-01-15
- Branch: 099-init-cli-args
- Test directory: /tmp/autospec-test-*

## New Flags Being Tested

| Flag Pair | Purpose |
|-----------|---------|
| `--sandbox` / `--no-sandbox` | Claude sandbox configuration |
| `--use-subscription` / `--no-use-subscription` | Billing preference |
| `--skip-permissions` / `--no-skip-permissions` | Autonomous mode |
| `--gitignore` / `--no-gitignore` | .gitignore modification |
| `--constitution` / `--no-constitution` | Constitution creation |

## Test Results Summary

| Test | Status |
|------|--------|
| T1: Help output | PASS |
| T2: Mutually exclusive errors | PASS |
| T3: Non-interactive init | PASS |
| T4: --no-sandbox | PASS |
| T5: --sandbox | PASS |
| T6: --gitignore | PASS |
| T7: --no-gitignore | PASS |
| T8: --skip-permissions | PASS |
| T9: --no-skip-permissions | PASS |
| T10: Basic init | PASS |
| T11: --project flag | PASS |

**Overall: ALL TESTS PASSED**

---

## Test Cases

### T1: Help output shows all new flags
```bash
autospec init --help | grep -E "(sandbox|subscription|skip-permissions|gitignore|constitution)"
```
**Expected**: All 10 new flags visible in help output
**Result**: PASS
```
      --constitution          Create project constitution (skips prompt)
      --gitignore             Add .autospec/ to .gitignore (skips prompt)
      --no-constitution       Skip constitution creation (skips prompt)
      --no-gitignore          Skip adding .autospec/ to .gitignore (skips prompt)
      --no-sandbox            Skip Claude sandbox configuration (skips prompt)
      --no-skip-permissions   Disable autonomous mode (more interactive) (skips prompt)
      --no-use-subscription   Use API key billing instead of subscription (skips prompt)
      --sandbox               Enable Claude sandbox configuration (skips prompt)
      --skip-permissions      Enable autonomous mode (skip permission prompts) (skips prompt)
      --use-subscription      Use subscription billing (OAuth/Pro/Max) instead of API key (skips prompt)
```

### T2: Mutually exclusive flag errors
**Expected**: Each returns error about mutually exclusive flags
**Result**: PASS (all 5 pairs)
```
Error: invalid flags: flags --sandbox and --no-sandbox are mutually exclusive
Error: invalid flags: flags --gitignore and --no-gitignore are mutually exclusive
Error: invalid flags: flags --constitution and --no-constitution are mutually exclusive
Error: invalid flags: flags --skip-permissions and --no-skip-permissions are mutually exclusive
Error: invalid flags: flags --use-subscription and --no-use-subscription are mutually exclusive
```

### T3: Non-interactive init with all flags (skip constitution)
```bash
autospec init --project --ai claude --no-sandbox --no-constitution --no-gitignore --skip-permissions
```
**Expected**: Completes without prompts, creates config
**Result**: PASS
- Config created at .autospec/config.yml
- No interactive prompts displayed
- Output shows skipped items: `⏭ Sandbox configuration: skipped (--no-sandbox)`

### T4: --no-sandbox skips sandbox prompt
**Result**: PASS - Output confirmed: `⏭ Sandbox configuration: skipped (--no-sandbox)`

### T5: --sandbox enables sandbox configuration
```bash
autospec init --project --ai claude --sandbox --no-constitution --no-gitignore --skip-permissions
```
**Expected**: Sandbox auto-configured
**Result**: PASS
```
✓ Claude Code sandbox: enabled
✓ Claude Code sandbox: configured with paths:
```

### T6: --gitignore adds .autospec/ to .gitignore
**Result**: PASS - `.gitignore` contents: `.autospec/`

### T7: --no-gitignore skips .gitignore modification
**Result**: PASS - Output confirmed: `⏭ Gitignore: skipped (--no-gitignore)`

### T8: --skip-permissions enables autonomous mode
**Result**: PASS - Config contains: `skip_permissions: true`

### T9: --no-skip-permissions disables autonomous mode
**Result**: PASS - Config contains: `skip_permissions: false`

### T10: Basic init still works
**Result**: PASS - Init completes with agent configured when using subset of flags

### T11: --project flag creates project-level config
**Result**: PASS - `.autospec/config.yml` and `.autospec/init.yml` created

---

## Cleanup

All test directories in `/tmp/autospec-test-*` removed after testing.
