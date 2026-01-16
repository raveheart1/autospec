# E2E Test Coverage Analysis

Analysis of current E2E test coverage for autospec CLI commands, flags, and mock infrastructure.

## Summary

| Metric | Value |
|--------|-------|
| Total CLI Commands | 35+ |
| Commands with E2E Tests | 8 (23%) |
| Total E2E Tests | 12 |
| Command Flags Tested | 1 (`run -a`) |
| Global Flags Tested | 0 |

## Key Sections

| Section | Description |
|---------|-------------|
| **CRITICAL** | **E2E Test Requirements (FR-001 to FR-008)** - Non-negotiable safety requirements |
| §1-7 | Command coverage matrix and current test analysis |
| §8 | **Test Implementation Checklist** (5 phases) |
| §9 | OpenCode mock specification |
| §10 | **Phase Ordering Test Strategy** (core workflows, property-based testing) |
| §11 | **Interactive Mode / Slash Command Testing** (clarify, checklist, analyze) |
| §12 | File locations reference |

## Safety Requirements Summary

| Requirement | Description |
|-------------|-------------|
| **FR-001** | No Real API Calls - never invoke real claude/opencode |
| **FR-002** | Environment Isolation - identical on local/CI |
| **FR-003** | Mock Response Completeness - all stages supported |
| **FR-004** | CI Compatibility - no TTY, parallel-safe |
| **FR-005** | State File Isolation - temp dirs for retry/config/history |
| **FR-006** | Git Isolation - temp repos, no ~/.gitconfig |
| **FR-007** | Health/Preflight Isolation - mock or skip |
| **FR-008** | Exit Code Verification - test all 6 exit codes |

---

## CRITICAL: E2E Test Requirements

### FR-001: No Real API Calls (NON-NEGOTIABLE)

E2E tests **MUST NEVER**:
- Invoke the real `claude` CLI binary
- Invoke the real `opencode` CLI binary
- Make any network requests to Anthropic, OpenAI, or any LLM API
- Incur any API costs whatsoever

**Enforcement:**
- All agent binaries must be mocked (`mock-claude.sh`, `mock-opencode.sh`, etc.)
- PATH must be isolated to only include mock binaries
- `ANTHROPIC_API_KEY` and similar env vars must be unset/sanitized
- Tests must fail fast if real binaries are detected in PATH

### FR-002: Environment Isolation (NON-NEGOTIABLE)

E2E tests **MUST** run identically on:
- Developer local machines (Linux, macOS)
- CI environments (GitHub Actions, etc.)
- Any system with Go installed

**Requirements:**
- No dependency on user's home directory config (`~/.config/autospec/`)
- No dependency on global git config
- No dependency on installed CLI tools (except Go, git, make)
- Temp directories for all test artifacts (cleaned up after)
- Deterministic outputs (no timestamps, random values in assertions)

**Enforcement:**
- Use `t.TempDir()` for all file operations
- Sanitize environment variables at test start
- Mock all external binaries via isolated PATH
- Use fixed test data, not generated content

### FR-003: Mock Response Completeness

All mocked binaries must support:
- Configurable exit codes (`MOCK_EXIT_CODE`)
- Configurable delays (`MOCK_DELAY`)
- Call logging for verification (`MOCK_CALL_LOG`)
- Artifact generation for all stages (`MOCK_ARTIFACT_DIR`)
- Response file injection (`MOCK_RESPONSE_FILE`)

**Current mock-claude.sh supports:**
- `/autospec.specify` → spec.yaml
- `/autospec.plan` → plan.yaml
- `/autospec.tasks` → tasks.yaml
- `/autospec.implement` → marks tasks completed
- `/autospec.constitution` → constitution.yaml

**Must add support for:**
- `/autospec.clarify` → updates spec.yaml
- `/autospec.checklist` → checklist.yaml
- `/autospec.analyze` → analysis.yaml

### FR-004: CI Compatibility

Tests must pass in CI without special configuration:
- No interactive prompts (use `-y` flags or mock responses)
- No TTY requirements
- Timeout limits respected (no hanging tests)
- Parallel test execution safe (isolated temp dirs)

### FR-005: State File Isolation (from architecture.md)

E2E tests **MUST NOT** read or write to real state files:

| File/Directory | Real Location | Test Requirement |
|----------------|---------------|------------------|
| Retry state | `~/.autospec/state/retry.json` | Use temp dir |
| Global config | `~/.config/autospec/config.yml` | Use temp dir or skip |
| Local config | `.autospec/config.yml` | Create in temp dir |
| Spec artifacts | `specs/NNN-feature/` | Create in temp dir |
| Command history | `~/.autospec/state/history.json` | Use temp dir |

**Enforcement:**
- Override `XDG_CONFIG_HOME` to temp directory
- Override `HOME` if needed for state isolation
- Set `AUTOSPEC_STATE_DIR` to temp directory
- Verify no writes to real home directory

### FR-006: Git Isolation

E2E tests must use isolated git repositories:
- Create fresh git repo in temp directory with `git init`
- Configure minimal git user: `git config user.email "test@test.com"`
- Do NOT interact with real project's `.git/`
- Do NOT rely on global git config (`~/.gitconfig`)

**Enforcement:**
```bash
# In test setup
export GIT_CONFIG_GLOBAL=/dev/null
export GIT_CONFIG_SYSTEM=/dev/null
git config --local user.email "test@test.com"
git config --local user.name "Test User"
```

### FR-007: Health/Preflight Check Isolation

The `autospec doctor` and preflight checks verify external dependencies.
In E2E tests:
- Health checks must find mock binaries, not real ones
- Network connectivity checks must be skipped or mocked
- File permission checks must use temp directories

**Enforcement:**
- Use `--skip-preflight` flag where appropriate
- Mock health check responses if testing health command itself

### FR-008: Exit Code Verification

E2E tests must verify correct exit codes per architecture spec:

| Code | Meaning | Test Scenario |
|------|---------|---------------|
| 0 | Success | Normal workflow completion |
| 1 | Validation failed | Invalid artifact format |
| 2 | Retry limit exhausted | `MOCK_EXIT_CODE=1` with max retries |
| 3 | Invalid arguments | Bad flag combinations |
| 4 | Missing dependencies | Remove mock binary from PATH |
| 5 | Command timeout | `MOCK_DELAY` > timeout setting |

### Implementation: E2EEnv Guarantees

The `internal/testutil/e2e.go` `E2EEnv` struct enforces these requirements:

```go
// NewE2EEnv creates an isolated test environment that:
// 1. Creates isolated temp directory (all file ops here)
// 2. Sets PATH to only include: mock binaries + autospec binary
// 3. Removes ANTHROPIC_API_KEY from environment
// 4. Removes OPENAI_API_KEY from environment
// 5. Removes any agent-specific API keys
// 6. Sets XDG_CONFIG_HOME to temp directory (FR-005)
// 7. Sets HOME to temp directory if needed (FR-005)
// 8. Sets AUTOSPEC_STATE_DIR to temp directory (FR-005)
// 9. Sets GIT_CONFIG_GLOBAL=/dev/null (FR-006)
// 10. Sets GIT_CONFIG_SYSTEM=/dev/null (FR-006)
// 11. Provides deterministic test data
// 12. Cleans up on test completion
```

**Verification tests (MUST exist):**
```go
func TestE2E_NoAPIKeyInEnvironment(t *testing.T) {
    env := testutil.NewE2EEnv(t)
    if env.HasAPIKeyInEnv() {
        t.Fatal("API key found in test environment - tests must not have access to real APIs")
    }
}

func TestE2E_PathIsolation(t *testing.T) {
    env := testutil.NewE2EEnv(t)
    // Verify only mock binaries are in PATH
    // Real claude/opencode must NOT be accessible
}

func TestE2E_StateFileIsolation(t *testing.T) {
    env := testutil.NewE2EEnv(t)
    // Run a command that writes state (e.g., implement with retry)
    // Verify NO writes to real ~/.autospec/ or ~/.config/autospec/
    // All state files must be in temp directory
}

func TestE2E_GitIsolation(t *testing.T) {
    env := testutil.NewE2EEnv(t)
    // Verify git operations use temp repo, not real .git/
    // Verify no dependency on ~/.gitconfig
}

func TestE2E_ExitCodes(t *testing.T) {
    // Verify exit code 0 for success
    // Verify exit code 1 for validation failure
    // Verify exit code 2 for retry exhaustion
    // Verify exit code 3 for invalid args
    // Verify exit code 4 for missing dependencies
    // Verify exit code 5 for timeout
}
```

---

## 1. Commands Coverage Matrix

### Core Workflow Commands

| Command | Aliases | E2E Tested | Flags Tested | Flags NOT Tested |
|---------|---------|------------|--------------|------------------|
| `specify` | `spec`, `s` | YES | None | `--max-retries`, `--agent`, `--auto-commit`, `--no-auto-commit` |
| `plan` | `p` | YES | None | `--max-retries`, `--agent`, `--auto-commit`, `--no-auto-commit`, `[prompt]` |
| `tasks` | `t` | YES | None | `--max-retries`, `--agent`, `--auto-commit`, `--no-auto-commit`, `[prompt]` |
| `implement` | `impl`, `i` | YES | None | `--resume`, `--phases`, `--phase N`, `--from-phase N`, `--tasks`, `--from-task`, `--single-session`, `--max-retries`, `--agent`, `--auto-commit` |

### Multi-Stage Workflow Commands

| Command | E2E Tested | Flags Tested | Flags NOT Tested |
|---------|------------|--------------|------------------|
| `prep` | YES | None | `--max-retries`, `--agent`, `--auto-commit` |
| `all` | YES | None | `--max-retries`, `--resume`, `--debug`, `--agent`, `--auto-commit` |
| `run` | **NO** | N/A | ALL: `-s`, `-p`, `-t`, `-i`, `-a`, `-n`, `-r`, `-l`, `-z`, `--spec`, `-y`, `--dry-run`, `--agent`, `--auto-commit`, `--resume` |

### Optional Stage Commands

| Command | E2E Tested | Notes |
|---------|------------|-------|
| `constitution` | **NO** | All flags untested |
| `clarify` | **NO** | All flags untested |
| `checklist` | **NO** | All flags untested |
| `analyze` | **NO** | All flags untested |

### Configuration Commands

| Command | Subcommands | E2E Tested | Notes |
|---------|-------------|------------|-------|
| `init` | — | **NO** | Project setup wizard |
| `config` | `show`, `set`, `get`, `toggle`, `keys`, `sync` | **NO** | Integration tests only |
| `migrate` | `mdtoyaml` | **NO** | Legacy migration |
| `doctor` | — | **NO** | Dependency checker |

### Utility Commands

| Command | E2E Tested | Notes |
|---------|------------|-------|
| `status` (`st`) | **NO** | Feature status display |
| `history` | **NO** | Command history |
| `version` | YES | Basic check only |
| `update` | **NO** | Self-update |
| `clean` | **NO** | Cleanup utility |
| `view` | **NO** | Artifact viewer |
| `ck` | **NO** | YAML validation |

### Worktree Commands

| Command | Subcommands | E2E Tested |
|---------|-------------|------------|
| `worktree` | `create`, `list`, `remove`, `prune`, `setup` | **NO** |

### DAG Commands

| Command | Subcommands | E2E Tested |
|---------|-------------|------------|
| `dag` | `run`, `status`, `resume`, `logs`, `reset`, `validate`, `watch` | **NO** |

### Admin Commands

| Command | Subcommands | E2E Tested |
|---------|-------------|------------|
| `commands` | `check`, `info`, `install` | **NO** |
| `completion` | `bash`, `zsh`, `fish`, `powershell`, `install` | **NO** |
| `uninstall` | — | **NO** |

---

## 2. Global Flags NOT Tested

These flags apply to all commands but have zero E2E coverage:

| Flag | Short | Description |
|------|-------|-------------|
| `--config` | `-c` | Custom config path |
| `--specs-dir` | — | Custom specs directory |
| `--skip-preflight` | — | Skip pre-flight validation |
| `--debug` | `-d` | Debug logging |
| `--verbose` | `-v` | Verbose output |
| `--output-style` | — | Output formatting (default, compact, minimal, plain, raw) |

---

## 3. Current E2E Test Files

```
tests/e2e/
├── e2e_test.go       # Basic CLI behavior (3 tests)
├── stage_test.go     # Individual stages (4 tests)
├── workflow_test.go  # Multi-stage workflows (2 tests)
└── error_test.go     # Error handling (3 tests)
```

### Test Breakdown

**e2e_test.go** (3 tests)
- `TestE2E_MockClaudeInvoked` - version/help commands
- `TestE2E_NoAPIKeyInEnvironment` - security check
- `TestE2E_PathIsolation` - mock PATH isolation

**stage_test.go** (4 tests)
- `TestE2E_Specify` - basic specify execution
- `TestE2E_Plan` - basic plan execution
- `TestE2E_Tasks` - basic tasks execution
- `TestE2E_Implement` - basic implement execution

**workflow_test.go** (2 tests)
- `TestE2E_PrepWorkflow` - specify → plan → tasks
- `TestE2E_FullWorkflow` - run -a full workflow

**error_test.go** (3 tests)
- `TestE2E_MissingConstitution` - missing prereq errors
- `TestE2E_MissingPrerequisite` - missing artifact errors
- `TestE2E_MockFailure` - exit code propagation

---

## 4. Mock Infrastructure

### Existing Mock Binary

**`mocks/scripts/mock-claude.sh`** - Claude CLI mock

Supported environment variables:
```bash
MOCK_EXIT_CODE=<int>          # Exit code (default: 0)
MOCK_DELAY=<int>              # Seconds delay (default: 0)
MOCK_ARTIFACT_DIR=<path>      # Artifact output directory
MOCK_SPEC_NAME=<name>         # Spec name (default: 001-test-feature)
MOCK_RESPONSE_FILE=<path>     # Response text file
MOCK_CALL_LOG=<path>          # Call logging file
```

### Test Data Files

```
tests/e2e/testdata/responses/
├── spec.yaml         # Valid spec artifact
├── plan.yaml         # Valid plan artifact
├── tasks.yaml        # Valid tasks artifact
├── constitution.yaml # Valid constitution artifact
└── implement.txt     # Implementation response
```

### E2E Test Helper

`internal/testutil/e2e.go` - `E2EEnv` struct

Key methods:
- `NewE2EEnv(t)` - Create isolated environment
- `Run(args...)` - Execute autospec command
- `SetupConstitution()` - Create constitution
- `SetupSpec(name)` / `SetupPlan(name)` / `SetupTasks(name)` - Create artifacts
- `SetMockExitCode(code)` - Configure mock behavior
- `InitGitRepo()` / `CreateBranch(name)` - Git setup

---

## 5. Mock Binaries Needed

### Required: mock-opencode.sh

OpenCode is a supported agent preset. Need equivalent mock for testing OpenCode workflows.

**Minimum requirements:**
- Accept same command patterns as OpenCode CLI
- Support artifact generation for all stages
- Support configurable exit codes
- Support call logging for verification

**Implementation outline:**
```bash
#!/bin/bash
# mocks/scripts/mock-opencode.sh

# Environment configuration
MOCK_EXIT_CODE="${MOCK_EXIT_CODE:-0}"
MOCK_DELAY="${MOCK_DELAY:-0}"
MOCK_ARTIFACT_DIR="${MOCK_ARTIFACT_DIR:-}"
MOCK_CALL_LOG="${MOCK_CALL_LOG:-}"

# Log the call
if [ -n "$MOCK_CALL_LOG" ]; then
    echo "opencode $*" >> "$MOCK_CALL_LOG"
fi

# Simulate delay
if [ "$MOCK_DELAY" -gt 0 ]; then
    sleep "$MOCK_DELAY"
fi

# OpenCode uses different command patterns than Claude CLI
# Parse -m flag for message/prompt
# Generate appropriate artifacts

exit "$MOCK_EXIT_CODE"
```

### Recommended: Additional Mock Infrastructure

| Mock | Purpose | Priority |
|------|---------|----------|
| `mock-opencode.sh` | OpenCode agent testing | HIGH |
| `mock-gemini.sh` | Gemini agent testing (if supported) | MEDIUM |
| `mock-cline.sh` | Cline agent testing (if supported) | LOW |
| `mock-slow-claude.sh` | Timeout testing (long delays) | LOW |
| `mock-flaky-claude.sh` | Retry/resilience testing (random failures) | LOW |

---

## 6. Priority Test Gaps

### HIGH Priority - Implement Command Flags

The `implement` command has the most flags and zero flag coverage:

```bash
# Execution mode tests needed
autospec implement --phases           # Phase-based execution
autospec implement --phase 2          # Specific phase only
autospec implement --from-phase 2     # Resume from phase
autospec implement --tasks            # Task-based execution
autospec implement --from-task T003   # Resume from task
autospec implement --single-session   # Legacy single-session mode

# Resume tests needed
autospec implement --resume           # Resume interrupted workflow
```

### HIGH Priority - Run Command

The `run` command has **zero E2E coverage**:

```bash
# Stage selection tests needed
autospec run -s "feature"    # specify only
autospec run -p              # plan only
autospec run -t              # tasks only
autospec run -i              # implement only
autospec run -pt             # plan + tasks
autospec run -ti             # tasks + implement
autospec run -spt "feature"  # specify + plan + tasks
autospec run -a "feature"    # all stages

# Optional stage tests needed
autospec run -n              # include constitution
autospec run -r              # include clarify
autospec run -l              # include checklist
autospec run -z              # include analyze

# Control flag tests needed
autospec run --spec 001-feat # explicit spec selection
autospec run -y -a "feat"    # skip confirmations
autospec run --dry-run -a    # preview mode
autospec run --resume -i     # resume implement
```

### MEDIUM Priority - Optional Stages

```bash
# Optional stage workflow tests
autospec constitution        # Create/update constitution
autospec clarify            # Clarify requirements
autospec checklist          # Generate checklist
autospec analyze            # Cross-artifact analysis
```

### MEDIUM Priority - Global Flags

Test across all commands:
```bash
autospec specify --max-retries 5 "feat"
autospec plan --agent opencode
autospec tasks --skip-preflight
autospec --config ./custom.yml specify "feat"
autospec --output-style minimal specify "feat"
```

### LOW Priority - Utility Commands

```bash
autospec status             # Current spec status
autospec history            # Command history
autospec clean              # Cleanup artifacts
autospec view spec          # View artifacts
autospec ck spec.yaml       # Validate YAML
```

---

## 7. Error Scenario Gaps

### Missing Error Tests

| Scenario | Expected Behavior |
|----------|-------------------|
| `implement` without tasks.yaml | Exit 1, error mentions "tasks" |
| `run -i` without tasks.yaml | Exit 1, validation error |
| Invalid `--phase` number | Exit 1, validation error |
| Invalid `--from-task` ID | Exit 1, task not found |
| Timeout during execution | Exit with timeout error |
| Agent binary not found | Exit 1, clear error message |
| Invalid config file | Exit 1, parse error |
| Conflicting flags | Exit 1, usage error |

### Missing Edge Case Tests

| Scenario | What to Test |
|----------|--------------|
| Feature description with quotes | `specify "feature with 'quotes'"` |
| Feature description with special chars | `specify "feature & more"` |
| Very long feature description | 1000+ character description |
| Empty feature description | `specify ""` should fail |
| Spec name mismatch | `--spec` different from branch name |

---

## 8. Test Implementation Checklist

### Phase 0: Safety Verification Tests (MUST EXIST FIRST)

- [ ] `TestE2E_NoAPIKeyInEnvironment` - Verify no API keys accessible
- [ ] `TestE2E_PathIsolation` - Verify only mock binaries in PATH
- [ ] `TestE2E_StateFileIsolation` - Verify no writes to real ~/.autospec/
- [ ] `TestE2E_GitIsolation` - Verify git uses temp repo only
- [ ] `TestE2E_ExitCodes` - Verify all 6 exit codes (0-5)
- [ ] Update `E2EEnv` to set `XDG_CONFIG_HOME`, `AUTOSPEC_STATE_DIR` to temp
- [ ] Update `E2EEnv` to set `GIT_CONFIG_GLOBAL=/dev/null`

### Phase 1: Critical Gaps

- [ ] Create `mock-opencode.sh` with artifact generation
- [ ] Add `implement` execution mode tests (--phases, --tasks, --phase N)
- [ ] Add `implement` resume tests (--resume, --from-phase, --from-task)
- [ ] Add `run` command stage selection tests
- [ ] Add `run` command optional stage tests

### Phase 2: Core Workflow & Phase Ordering Tests

- [ ] Add `autospec run -spti "prompt"` full workflow test
- [ ] Add `autospec run -a "prompt"` alias test
- [ ] Add `autospec prep "prompt"` workflow test
- [ ] Add phase ordering property tests (flag order independence)
- [ ] Add partial workflow tests (`-sp`, `-spt`, `-pti`, `-ti`)
- [ ] Add reversed flag test (`-itps` same as `-spti`)
- [ ] Add single-stage run tests (`-s`, `-p`, `-t`, `-i`)
- [ ] Add prerequisite error tests for each stage combination

### Phase 3: Interactive Mode / Slash Command Support

- [ ] Enhance `mock-claude.sh` with `/autospec.clarify` support
- [ ] Enhance `mock-claude.sh` with `/autospec.checklist` support
- [ ] Enhance `mock-claude.sh` with `/autospec.analyze` support
- [ ] Add test data files: `clarification.yaml`, `checklist.yaml`, `analysis.yaml`
- [ ] Add `autospec clarify` E2E test
- [ ] Add `autospec checklist` E2E test
- [ ] Add `autospec analyze` E2E test
- [ ] Add `autospec run -a -rlz` combined optional stages test

### Phase 4: Important Coverage

- [ ] Add global flag tests (--max-retries, --agent, --skip-preflight)
- [ ] Add optional stage command tests (constitution standalone)
- [ ] Add config command tests (init, config show/set/get)
- [ ] Add error scenario tests (invalid flags, missing binaries)

### Phase 5: Comprehensive Coverage

- [ ] Add utility command tests (status, history, clean, view, ck)
- [ ] Add worktree command tests
- [ ] Add dag command tests
- [ ] Add admin command tests (commands, completion)
- [ ] Add edge case tests (special characters, long inputs)
- [ ] Add timeout/resilience tests (MOCK_DELAY)

---

## 9. OpenCode Mock Specification

OpenCode CLI uses different invocation patterns than Claude CLI. The mock needs to handle:

### Command Pattern Differences

**Claude CLI:**
```bash
claude -p "prompt" --allowedTools ... --output-format ...
```

**OpenCode CLI:**
```bash
opencode -m "prompt" [options]
```

### mock-opencode.sh Requirements

1. Parse `-m` flag for prompt content
2. Detect stage from prompt content (specify, plan, tasks, implement)
3. Generate appropriate YAML artifact to output directory
4. Support all MOCK_* environment variables
5. Log calls for test verification
6. Support configurable exit codes and delays

### Test Data Needed

Same test data files work for both agents:
- `tests/e2e/testdata/responses/spec.yaml`
- `tests/e2e/testdata/responses/plan.yaml`
- `tests/e2e/testdata/responses/tasks.yaml`
- `tests/e2e/testdata/responses/constitution.yaml`

---

## 10. Phase Ordering Test Strategy

### Core Workflow Tests (Required)

These workflows must have dedicated E2E tests:

```bash
# Full workflows (most common usage)
autospec run -spti "feature"     # All core stages in order
autospec run -a "feature"        # Alias for all stages
autospec prep "feature"          # specify → plan → tasks (no implement)

# Partial workflows (common patterns)
autospec run -sp "feature"       # specify + plan only
autospec run -spt "feature"      # specify + plan + tasks (same as prep)
autospec run -pti                # plan + tasks + implement (spec exists)
autospec run -ti                 # tasks + implement (plan exists)
```

### Phase Ordering Validation Approach

**Key insight**: We don't need to test all 2^4=16 combinations. Instead, test that:

1. **Flag order is irrelevant** - `-spti` and `-itps` produce same execution order
2. **Prerequisites validated** - Missing artifacts cause proper errors
3. **Artifacts created correctly** - Each stage produces expected output

**Test matrix (minimal coverage)**:

| Test Name | Flags | Prereqs Needed | Validates |
|-----------|-------|----------------|-----------|
| `TestRun_AllStages` | `-spti` | constitution | Full workflow |
| `TestRun_AllAlias` | `-a` | constitution | Alias works |
| `TestRun_ReversedFlags` | `-itps` | constitution | Order independence |
| `TestRun_SpecifyOnly` | `-s` | constitution | Single stage |
| `TestRun_PlanOnly` | `-p` | spec.yaml | Single stage with prereq |
| `TestRun_TasksOnly` | `-t` | plan.yaml | Single stage with prereq |
| `TestRun_ImplementOnly` | `-i` | tasks.yaml | Single stage with prereq |
| `TestRun_PrepEquivalent` | `-spt` | constitution | Matches `prep` behavior |
| `TestRun_LastTwoStages` | `-ti` | plan.yaml | Partial workflow |
| `TestRun_MissingPrereq` | `-p` | NO spec | Error handling |

### Property-Based Testing for Phase Orderings

For comprehensive coverage without exhaustive tests, implement property-based tests:

```go
// TestRun_PhaseOrderingProperties tests phase ordering invariants
func TestRun_PhaseOrderingProperties(t *testing.T) {
    // Property 1: Flag order doesn't affect execution order
    // Generate random permutations of "spti" and verify same artifacts created

    // Property 2: Subset flags create only relevant artifacts
    // e.g., "-sp" creates spec.yaml and plan.yaml, NOT tasks.yaml

    // Property 3: Missing prerequisites cause correct errors
    // e.g., "-t" without plan.yaml fails with "plan" in error message
}
```

### Random Phase Ordering Test Helper

```go
// PhaseOrderingTest defines a phase ordering test case
type PhaseOrderingTest struct {
    Flags           string   // e.g., "-spt"
    PrereqArtifacts []string // artifacts that must exist before run
    ExpectedOutputs []string // artifacts that should exist after run
    ExpectError     bool     // whether command should fail
    ErrorContains   string   // substring expected in error
}

// GeneratePhaseOrderingTests creates test cases for phase combinations
func GeneratePhaseOrderingTests() map[string]PhaseOrderingTest {
    return map[string]PhaseOrderingTest{
        "all_stages": {
            Flags:           "-spti",
            PrereqArtifacts: []string{}, // constitution handled by setup
            ExpectedOutputs: []string{"spec.yaml", "plan.yaml", "tasks.yaml"},
        },
        "reversed_flags": {
            Flags:           "-itps",
            PrereqArtifacts: []string{},
            ExpectedOutputs: []string{"spec.yaml", "plan.yaml", "tasks.yaml"},
        },
        "plan_without_spec": {
            Flags:           "-p",
            PrereqArtifacts: []string{}, // no spec.yaml
            ExpectError:     true,
            ErrorContains:   "spec",
        },
        // ... more cases
    }
}
```

---

## 11. Interactive Mode / Slash Command Testing

### Background

Some autospec stages invoke Claude in "interactive" mode where the agent:
- Reads existing artifacts
- May modify artifacts (clarify updates spec.yaml)
- May create new artifacts (checklist creates checklist.yaml)
- Runs via slash commands like `/autospec.clarify`, `/autospec.checklist`

### Current mock-claude.sh Limitations

The current mock detects these slash commands:
- `/autospec.specify` → generates spec.yaml
- `/autospec.plan` → generates plan.yaml
- `/autospec.tasks` → generates tasks.yaml
- `/autospec.implement` → marks tasks completed
- `/autospec.constitution` → generates constitution.yaml

**Missing interactive stage support:**
- `/autospec.clarify` → should update spec.yaml with clarifications
- `/autospec.checklist` → should generate checklist.yaml
- `/autospec.analyze` → should generate analysis.yaml (or similar)

### Mock Enhancements Needed

```bash
# Add to mock-claude.sh generate_artifact() function:

elif [[ "$command" == *"/autospec.clarify"* ]]; then
    generate_clarification "$spec_dir"
elif [[ "$command" == *"/autospec.checklist"* ]]; then
    generate_checklist "$spec_dir"
elif [[ "$command" == *"/autospec.analyze"* ]]; then
    generate_analysis "$spec_dir"
```

### New Artifact Generator Functions

**generate_clarification** - Appends clarifications to spec.yaml:
```bash
generate_clarification() {
    local spec_dir="$1"
    local spec_file="$spec_dir/spec.yaml"

    # Only update if spec.yaml exists
    if [[ ! -f "$spec_file" ]]; then
        return
    fi

    # Append clarifications section if not present
    if ! grep -q "clarifications:" "$spec_file"; then
        cat >> "$spec_file" << 'CLARIFY_EOF'
clarifications:
  - question: "What authentication method?"
    answer: "JWT tokens"
    timestamp: "2025-01-01T00:00:00Z"
    impact: "Updated FR-001 acceptance criteria"
CLARIFY_EOF
    fi
}
```

**generate_checklist** - Creates checklist.yaml:
```bash
generate_checklist() {
    local spec_dir="$1"
    mkdir -p "$spec_dir"
    cat > "$spec_dir/checklist.yaml" << 'CHECKLIST_EOF'
checklist:
  branch: "001-test-feature"
  spec_path: "specs/001-test-feature/spec.yaml"
  generated: "2025-01-01T00:00:00Z"
categories:
  - name: "Functional Requirements"
    items:
      - id: "CL-001"
        requirement_id: "FR-001"
        description: "Feature implemented"
        status: "pending"
        verification_method: "manual"
  - name: "Code Quality"
    items:
      - id: "CL-002"
        requirement_id: "NFR-001"
        description: "Tests pass"
        status: "pending"
        verification_method: "automated"
_meta:
  version: "1.0.0"
  artifact_type: "checklist"
CHECKLIST_EOF
}
```

**generate_analysis** - Creates analysis.yaml:
```bash
generate_analysis() {
    local spec_dir="$1"
    mkdir -p "$spec_dir"
    cat > "$spec_dir/analysis.yaml" << 'ANALYSIS_EOF'
analysis:
  branch: "001-test-feature"
  spec_path: "specs/001-test-feature/spec.yaml"
  generated: "2025-01-01T00:00:00Z"
consistency_checks:
  - check: "Requirements traceable"
    status: "pass"
    details: "All FRs linked to user stories"
  - check: "Test coverage complete"
    status: "pass"
    details: "All acceptance criteria have tests"
gaps_identified: []
recommendations: []
quality_score: 95
_meta:
  version: "1.0.0"
  artifact_type: "analysis"
ANALYSIS_EOF
}
```

### Test Data Files Needed

Add to `tests/e2e/testdata/responses/`:
- `clarification.yaml` - Sample clarification response
- `checklist.yaml` - Sample checklist artifact
- `analysis.yaml` - Sample analysis artifact

### Interactive Mode Test Cases

```bash
# Optional stage tests
autospec clarify                # Updates existing spec with clarifications
autospec checklist              # Generates checklist from spec
autospec analyze                # Analyzes cross-artifact consistency

# Run command with optional stages
autospec run -nspti "feature"   # constitution + core stages
autospec run -spti -r "feature" # core + clarify
autospec run -spti -l "feature" # core + checklist
autospec run -spti -z "feature" # core + analyze
autospec run -a -rlz "feature"  # all core + all optional
```

### Non-Interactive Simulation

For E2E tests, interactive stages should:

1. **Auto-accept defaults** - Use `-y` flag or equivalent
2. **Pre-seed responses** - Use MOCK_RESPONSE_FILE with prepared answers
3. **Skip prompts** - Use environment variable to skip interactive prompts

```bash
# Environment variable for non-interactive mode
export AUTOSPEC_NON_INTERACTIVE=1
export MOCK_CLARIFY_RESPONSES=/path/to/responses.yaml
```

---

## 12. File Locations Reference

### E2E Test Files
- `tests/e2e/e2e_test.go`
- `tests/e2e/stage_test.go`
- `tests/e2e/workflow_test.go`
- `tests/e2e/error_test.go`

### Mock Scripts
- `mocks/scripts/mock-claude.sh` (EXISTS)
- `mocks/scripts/mock-opencode.sh` (NEEDED)

### Test Utilities
- `internal/testutil/e2e.go`

### Test Data
- `tests/e2e/testdata/responses/`

### CLI Command Implementations
- `internal/cli/stages/` - Core stage commands
- `internal/cli/run.go` - Run command
- `internal/cli/prep.go` - Prep command
- `internal/cli/all.go` - All command
- `internal/cli/config/` - Config commands
- `internal/cli/util/` - Utility commands
- `internal/cli/admin/` - Admin commands
- `internal/cli/worktree/` - Worktree commands
