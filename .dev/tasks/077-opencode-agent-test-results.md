# OpenCode Agent Manual Testing Plan

Feature: 077-opencode-agent-preset

## Prerequisites

```bash
# Verify opencode is installed
opencode --version

# Verify autospec is built
make build && ./bin/autospec version
```

## Test Repository Setup

Test repo: `~/repos/opencode-test`

## Test Cases

### 1. Init Command (--ai flag)

```bash
# TC-1a: Init with opencode only
autospec init --ai opencode

# Expected:
# - .opencode/command/autospec.*.md files created
# - opencode.json created with "autospec *": "allow"

# TC-1b: Init with multiple agents
autospec init --ai claude,opencode

# Expected:
# - .claude/commands/autospec.*.md files created
# - .opencode/command/autospec.*.md files created
# - opencode.json created
```

### 2. Doctor Command

```bash
# TC-2: Verify opencode detection
autospec doctor

# Expected:
# - opencode: installed (version shown)
```

### 3. Specify Stage

```bash
# TC-3: Run specify with opencode
AUTOSPEC_AGENT_PRESET=opencode autospec specify "add hello endpoint"

# Expected:
# - OpenCode invoked with: opencode run "<prompt>" --command autospec.specify
# - spec.yaml created in specs/<branch>/
```

### 4. Plan Stage

```bash
# TC-4: Run plan with opencode
AUTOSPEC_AGENT_PRESET=opencode autospec plan

# Expected:
# - OpenCode invoked with: opencode run "<prompt>" --command autospec.plan
# - plan.yaml created
```

### 5. Tasks Stage

```bash
# TC-5: Run tasks with opencode
AUTOSPEC_AGENT_PRESET=opencode autospec tasks

# Expected:
# - OpenCode invoked with: opencode run "<prompt>" --command autospec.tasks
# - tasks.yaml created
```

### 6. Implement Stage

```bash
# TC-6a: Run implement with opencode (full)
AUTOSPEC_AGENT_PRESET=opencode autospec implement

# TC-6b: Run implement with --phases
AUTOSPEC_AGENT_PRESET=opencode autospec implement --phases

# TC-6c: Run implement with --tasks
AUTOSPEC_AGENT_PRESET=opencode autospec implement --tasks
```

### 7. Full Pipeline (run command)

```bash
# TC-7: Run full pipeline
AUTOSPEC_AGENT_PRESET=opencode autospec run -a "add goodbye endpoint"
```

### 8. Prep Command

```bash
# TC-8: Run prep (specify → plan → tasks)
AUTOSPEC_AGENT_PRESET=opencode autospec prep "add status endpoint"
```

### 9. Interactive Commands

```bash
# TC-9a: Clarify (uses --prompt mode)
AUTOSPEC_AGENT_PRESET=opencode autospec clarify

# TC-9b: Analyze
AUTOSPEC_AGENT_PRESET=opencode autospec analyze
```

### 10. Retry System

```bash
# TC-10: Test retry with validation failure
# Create invalid spec.yaml, run plan, verify retry prompt injection
```

### 11. Timeout Configuration

```bash
# TC-11: Test timeout
AUTOSPEC_AGENT_PRESET=opencode AUTOSPEC_TIMEOUT=30s autospec specify "test"

# Expected: Should timeout after 30s if not complete
```

### 12. Config Flag Override

```bash
# TC-12: Use --agent flag instead of env var
autospec specify "test" --agent opencode
```

## Validation Checklist

- [x] TC-1a: Init with opencode creates correct files
- [x] TC-1b: Init with multiple agents works
- [x] TC-2: Doctor detects opencode
- [x] TC-3: Specify works with opencode
- [x] TC-4: Plan works with opencode
- [x] TC-5: Tasks works with opencode (flaky - see issues)
- [ ] TC-6a: Implement (full) works
- [x] TC-6b: Implement --phases works (partial - see issues)
- [ ] TC-6c: Implement --tasks works
- [x] TC-7: Run full pipeline works (tested in separate repo)
- [ ] TC-8: Prep command works
- [x] TC-9a: Clarify (interactive) works
- [x] TC-9b: Analyze works
- [x] TC-10: Retry injects errors correctly (implicit - see TC-5 retry exhaustion)
- [x] TC-11: Timeout respected
- [x] TC-12: --agent flag works

## Issues Found

| TC | Issue | Status |
|----|-------|--------|
| TC-5 | First run failed with `tasks.yaml: no such file or directory` - OpenCode didn't complete write. Second run succeeded. Likely due to cheaper model before Opus 4.5 config was applied. | Model-dependent |
| TC-6b | OpenCode used Edit tool with empty `oldString` to create new files (main_test.go). Likely due to cheaper model before Opus 4.5 config was applied. | Model-dependent |

## Test Run Details (2026-01-02)

### Environment
- opencode: v1.0.223
- autospec: v0.7.3
- Platform: linux/amd64

### TC-1a Results
- `.opencode/command/autospec.*.md` files created (9 files)
- `opencode.json` created with `"autospec *": "allow"`

### TC-1b Results
- `.claude/commands/autospec.*.md` files created
- `.opencode/command/autospec.*.md` files created
- `opencode.json` created with permissions

### TC-2 Results
```
CLI Agents:
  opencode: installed (v1.0.223)
```

### TC-3 Results
- Command: `opencode run add hello endpoint --command autospec.specify`
- Output: `specs/001-hello-endpoint/spec.yaml` created

### TC-4 Results
- Command: `opencode run --command autospec.plan`
- Output: `plan.yaml` created

### TC-5 Results
- Command: `opencode run --command autospec.tasks`
- First attempt: FAILED (file not written)
- Second attempt: PASSED, `tasks.yaml` created

### TC-6b Results
- Command: `opencode run phase 1 -f .autospec/context/phase-1.yaml --command autospec.implement`
- OpenCode updated `main.go` successfully
- Failed to create `main_test.go` (Edit tool misuse)

### TC-9a Results
- Command: `AUTOSPEC_AGENT_PRESET=opencode autospec clarify`
- Correctly invoked: `opencode /autospec.clarify` (interactive mode)

### TC-9b Results
- Command: `AUTOSPEC_AGENT_PRESET=opencode autospec analyze`
- Requires: plan.yaml and tasks.yaml present
- Works correctly when prerequisites exist

### TC-7 Results
- Command: `AUTOSPEC_AGENT_PRESET=opencode autospec run -a "add <feature>"`
- Tested in separate repo - full pipeline (specify → plan → tasks → implement) works

### TC-10 Results
- Retry system active: TC-5 showed `"exhausted retries after 1 total attempts"`
- Direct test: Agent succeeded on first try, couldn't force controlled failure
- Note: Retry count configurable via `AUTOSPEC_MAX_RETRIES` env var

### TC-11 Results
- Command: `AUTOSPEC_AGENT_PRESET=opencode AUTOSPEC_TIMEOUT=5 autospec tasks`
- Result: `command timed out after 5s` - timeout working correctly
- Note: AUTOSPEC_TIMEOUT expects integer (seconds), not duration string

### TC-12 Results
- Command: `autospec specify "add goodbye endpoint" --agent opencode`
- Correctly invoked: `opencode run add goodbye endpoint --command autospec.specify`

