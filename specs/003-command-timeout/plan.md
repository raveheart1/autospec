# Implementation Plan: Command Execution Timeout

**Branch**: `003-command-timeout` | **Date**: 2025-10-22 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/003-command-timeout/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

Implement timeout functionality for Claude CLI command execution to prevent indefinite command hangs in automated workflows. The feature will use Go's context package with deadline enforcement to abort long-running commands after a configurable duration. Timeout configuration will integrate with the existing koanf-based configuration system and support the standard configuration hierarchy (environment variables > local config > global config > defaults).

## Technical Context

**Language/Version**: Go 1.25.1
**Primary Dependencies**:
  - github.com/spf13/cobra v1.10.1 (CLI framework)
  - github.com/knadh/koanf/v2 v2.3.0 (configuration management)
  - github.com/go-playground/validator/v10 v10.28.0 (struct validation)
  - github.com/stretchr/testify v1.11.1 (testing framework)
**Storage**: File system (JSON config files in ~/.autospec/config.json and .autospec/config.json, state in ~/.autospec/state/retry.json)
**Testing**: Go standard testing package with testify assertions, table-driven tests, benchmarks
**Target Platform**: Cross-platform (Linux, macOS, Windows) - single binary distribution
**Project Type**: Single CLI binary (autospec)
**Performance Goals**:
  - Validation operations: <1 second
  - Validation functions: <10ms per call
  - Timeout enforcement: <5 seconds after threshold
  - Overall workflow: Sub-second for validation checks
**Constraints**:
  - Must maintain backward compatibility (default no timeout)
  - Must not add >1% performance overhead when timeout is enabled
  - Must support timeout values from 1 second to 1 hour
  - Must clean up all spawned processes on timeout
  - Binary size should remain <15MB
**Scale/Scope**:
  - Single CLI binary (~10MB)
  - ~15 packages across internal/
  - ~15 test files with Go tests
  - ~10 commands in internal/cli/

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### Principle I: Validation-First
**Status**: ✅ PASS
- Timeout configuration validation is already present in config.go:27 with validator tag `validate:"min=1,max=3600"`
- ClaudeExecutor will validate timeout application before command execution
- Error cases will be tested with table-driven tests

### Principle II: Hook-Based Enforcement
**Status**: ✅ PASS
- No new hooks required
- Feature modifies existing command execution infrastructure
- Existing hooks will continue to enforce workflow gates

### Principle III: Test-First Development
**Status**: ✅ PASS (Commitment Required)
- **Action Required**: Write tests before implementation:
  - Unit tests for timeout enforcement in ClaudeExecutor
  - Unit tests for config validation
  - Integration tests for timeout behavior
  - Benchmark tests to verify <1% overhead
  - Edge case tests (timeout during cleanup, system clock changes)
- Tests must be written first, implementation must pass all tests

### Principle IV: Performance Standards
**Status**: ✅ PASS
- Timeout enforcement adds minimal overhead (Go context is lightweight)
- Config validation already meets <10ms requirement (part of startup)
- Benchmark test required to verify <1% overhead requirement (SC-004)
- Performance regression must be caught in CI

### Principle V: Idempotency & Retry Logic
**Status**: ✅ PASS
- Timeout enforcement is idempotent (command either completes or times out)
- Timeout errors will use existing error code system
- New error code for timeout (proposed: exit code 5 for timeout)
- Retry logic unaffected (timeout occurs at execution layer, not validation layer)

## Project Structure

### Documentation (this feature)

```text
specs/[###-feature]/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md        # Phase 1 output (/speckit.plan command)
├── quickstart.md        # Phase 1 output (/speckit.plan command)
├── contracts/           # Phase 1 output (/speckit.plan command)
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root)

```text
cmd/
└── autospec/
    └── main.go              # Binary entry point (no changes)

internal/
├── config/
│   ├── config.go            # MODIFY: Timeout field already exists (line 27)
│   ├── defaults.go          # MODIFY: Add default timeout value (0 = no timeout)
│   └── config_test.go       # ADD: Tests for timeout validation
│
├── workflow/
│   ├── claude.go            # MODIFY: Add context.WithTimeout to Execute()
│   ├── claude_test.go       # ADD: Tests for timeout enforcement
│   └── executor.go          # MODIFY: Pass timeout to ClaudeExecutor
│
└── cli/
    └── root.go              # MODIFY: Add --timeout flag (optional)

integration/
├── claude_test.go           # ADD: Integration tests for timeout behavior
└── config_test.go           # MODIFY: Add timeout config scenarios

tests/
└── [legacy bats tests]      # NO CHANGES (being phased out)
```

**Structure Decision**: This is a single Go binary project (Option 1). The feature modifies existing files in `internal/config/` and `internal/workflow/` packages. No new packages are required. The timeout configuration field already exists in the Configuration struct (config.go:27), so the primary work is implementing the timeout enforcement in ClaudeExecutor and adding comprehensive tests.

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

**No violations detected.** All constitution principles are satisfied:
- Validation-First: Config validation already present
- Hook-Based Enforcement: No new hooks needed
- Test-First Development: Committed to write tests first
- Performance Standards: Go context adds minimal overhead
- Idempotency: Timeout enforcement is naturally idempotent

No complexity justification required.

---

## Post-Design Constitution Re-evaluation

*Re-check after Phase 1 design complete*

### Principle I: Validation-First
**Status**: ✅ PASS (Confirmed)
- Config validation confirmed in contracts/config-api.md
- Error handling confirmed in contracts/executor-api.md
- Test coverage planned in data-model.md

### Principle II: Hook-Based Enforcement
**Status**: ✅ PASS (Confirmed)
- No new hooks required
- Feature is transparent to existing workflow hooks
- Hooks continue to enforce workflow gates independently of timeout

### Principle III: Test-First Development
**Status**: ✅ PASS (Design Complete)
- Comprehensive test plan documented in contracts/executor-api.md:
  - 7 unit test cases defined
  - 3 benchmark tests defined
  - 2 integration tests defined
- Test cases cover:
  - Normal execution without timeout
  - Execution with timeout (success and failure)
  - Error metadata validation
  - Process cleanup verification
  - Performance overhead validation (<1% requirement)

### Principle IV: Performance Standards
**Status**: ✅ PASS (Design Validated)
- Go context.WithTimeout has <100ns overhead (documented in research.md)
- Benchmark tests defined to verify <1% overhead requirement
- Performance contract documented in contracts/executor-api.md
- No polling or background goroutines (zero ongoing overhead)

### Principle V: Idempotency & Retry Logic
**Status**: ✅ PASS (Design Confirmed)
- Timeout enforcement is idempotent (same timeout = same behavior)
- Exit code 5 defined for timeout errors (distinct from other errors)
- TimeoutError type supports programmatic error detection
- Integration with existing retry logic: timeout errors fail immediately (no retry)

### Summary

All constitution principles remain satisfied after detailed design phase:
- ✅ Validation-First: Comprehensive validation and error handling
- ✅ Hook-Based Enforcement: No new hooks needed, transparent to existing system
- ✅ Test-First Development: Detailed test plan ready for implementation
- ✅ Performance Standards: <1% overhead guaranteed by Go context design
- ✅ Idempotency & Retry Logic: Clear error codes and idempotent behavior

**No design changes required to satisfy constitution.**
