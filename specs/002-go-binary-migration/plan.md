# Implementation Plan: Go Binary Migration

**Branch**: `002-go-binary-migration` | **Date**: 2025-10-22 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/002-go-binary-migration/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

Transform the current bash-based autospec validation tool into a single, cross-platform Go binary that provides the same validation and workflow orchestration capabilities without requiring bash, jq, git, or other shell utilities. The Go binary will support all major platforms (Linux, macOS, Windows) and provide a simple installation experience via `go install` or pre-built binaries.

## Technical Context

**Language/Version**: Go 1.21+
**Primary Dependencies**: NEEDS CLARIFICATION (cobra for CLI, go-git for git operations, spf13/viper for config - need to research best practices)
**Storage**: File-based (JSON for config at ~/.autospec/config.json and .autospec/config.json, retry state at ~/.autospec/state/retry.json)
**Testing**: Go testing package (testing), table-driven tests, need to determine test coverage tool
**Target Platform**: Cross-platform (Linux amd64/arm64, macOS amd64/arm64, Windows amd64)
**Project Type**: Single CLI binary
**Performance Goals**: Startup <50ms, validation <100ms, status command <1s, workflow orchestration <5s (excluding Claude execution)
**Constraints**: Binary size <15MB, zero runtime dependencies beyond claude and specify CLIs, must support custom command templates with pipes and env vars
**Scale/Scope**: Single-user CLI tool, orchestrates 3-5 SpecKit workflow phases, manages ~10 configuration parameters, validates markdown files up to several MB

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### I. Validation-First
**Status**: ✅ PASS
- Go binary will maintain all existing validation logic from bash scripts
- Automatic retry mechanisms will be preserved (max 3 attempts)
- All workflow transitions validated before proceeding

### II. Hook-Based Enforcement
**Status**: ⚠️ NOT APPLICABLE (with note)
- This migration focuses on the CLI tool itself, not the Claude Code hooks
- Existing hook scripts in `scripts/hooks/` will continue to work as-is
- Hooks call the validation logic which will be migrated to Go
- Future work may migrate hooks themselves, but not in this feature scope

### III. Test-First Development
**Status**: ✅ PASS
- All 60+ existing bash tests will be ported to Go tests before implementation
- Go testing framework supports table-driven tests for comprehensive coverage
- Will maintain test-first approach: write tests, then implementation
- Test coverage must not decrease below 60+ baseline

### IV. Performance Standards
**Status**: ✅ PASS
- Performance requirements explicitly defined in spec (startup <50ms, validation <100ms, status <1s)
- Go compiled binaries are typically faster than bash scripts
- Performance targets align with constitution (<1s for validation operations)

### V. Idempotency & Retry Logic
**Status**: ✅ PASS
- All retry mechanisms will be ported from bash to Go
- Persistent retry state at ~/.autospec/state/retry.json
- Standardized exit codes preserved (0=success, 1=failed, 2=exhausted, 3=invalid, 4=missing deps)
- Idempotent operations maintained

**Overall Gate Status**: ✅ PASS - Proceed to Phase 0

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
auto-claude-speckit/
├── cmd/
│   └── autospec/
│       ├── main.go                    # Entry point
│       ├── main_test.go               # CLI integration tests
│       └── testdata/
│           └── scripts/               # testscript CLI tests
│
├── internal/                          # Private application code
│   ├── cli/                          # Cobra CLI commands
│   │   ├── root.go                   # Root command and global flags
│   │   ├── init.go                   # autospec init command
│   │   ├── workflow.go               # autospec workflow command
│   │   ├── specify.go                # autospec specify command
│   │   ├── plan.go                   # autospec plan command
│   │   ├── tasks.go                  # autospec tasks command
│   │   ├── implement.go              # autospec implement command
│   │   ├── status.go                 # autospec status command
│   │   ├── config.go                 # autospec config command
│   │   └── version.go                # autospec version command
│   │
│   ├── config/                       # Configuration management
│   │   ├── config.go                 # Koanf-based config loading
│   │   ├── config_test.go            # Unit tests
│   │   └── defaults.go               # Default configuration values
│   │
│   ├── validation/                   # Validation library (ported from bash)
│   │   ├── validation.go             # File validation functions
│   │   ├── validation_test.go        # Unit tests
│   │   ├── validation_bench_test.go  # Benchmarks
│   │   ├── tasks.go                  # Task parsing and counting
│   │   ├── tasks_test.go             # Task parsing tests
│   │   └── prompt.go                 # Continuation prompt generation
│   │
│   ├── retry/                        # Retry state management
│   │   ├── retry.go                  # Load, save, increment, reset
│   │   ├── retry_test.go             # Unit tests
│   │   └── state.go                  # RetryState struct and methods
│   │
│   ├── git/                          # Git operations
│   │   ├── git.go                    # Branch, root, repo check
│   │   └── git_test.go               # Unit tests with mocking
│   │
│   ├── spec/                         # Spec detection and metadata
│   │   ├── spec.go                   # DetectCurrentSpec, GetSpecDirectory
│   │   └── spec_test.go              # Unit tests
│   │
│   └── workflow/                     # Workflow orchestration
│       ├── workflow.go               # Specify→plan→tasks orchestration
│       ├── workflow_test.go          # Unit tests
│       ├── claude.go                 # Claude CLI execution
│       ├── preflight.go              # Pre-flight validation checks
│       └── executor.go               # Command execution with retry
│
├── integration/                       # Integration tests
│   ├── workflow_test.go              # End-to-end workflow tests
│   ├── retry_test.go                 # Retry logic integration tests
│   └── testdata/
│       ├── fixtures/                 # Test spec directories
│       └── golden/                   # Golden file outputs
│
├── scripts/                          # Build and maintenance scripts
│   ├── build-all.sh                  # Cross-platform build script
│   ├── test-all.sh                   # Run all tests (unit + integration)
│   └── benchmark.sh                  # Performance benchmarking
│
├── specs/                            # Feature specifications (existing)
│   └── 002-go-binary-migration/
│       ├── spec.md
│       ├── plan.md                   # This file
│       ├── research.md
│       ├── data-model.md
│       ├── quickstart.md
│       └── contracts/
│           ├── cli-interface.md
│           └── validation-api.md
│
├── go.mod                            # Go module definition
├── go.sum                            # Dependency checksums
├── README.md                         # Updated with Go installation
├── CLAUDE.md                         # Updated with Go commands
└── .gitignore                        # Ignore dist/, *.test, coverage files
```

**Structure Decision**:

This project uses **Option 1: Single Project** structure, as it's a single CLI binary with no separate frontend/backend components.

**Key Design Decisions:**
1. **cmd/autospec/**: Single entry point for the binary, following Go conventions
2. **internal/**: All application code is internal (not importable by other projects)
3. **internal/cli/**: One file per Cobra command for clarity and maintainability
4. **internal/validation/**: Direct port of bash validation library functionality
5. **internal/retry/**: Persistent retry state management (replaces bash /tmp files)
6. **integration/**: Separate from unit tests for clear test organization
7. **testscript tests**: CLI-specific testing using Go's standard approach

**Why internal/ over pkg/**:
- This is a single binary, not a library
- No code needs to be importable by external projects
- `internal/` enforces encapsulation at compile time

## Complexity Tracking

No constitution violations identified. This design maintains simplicity while achieving all functional requirements.

**Complexity Assessment:**
- Single binary architecture (no microservices)
- Standard Go project layout (cmd/, internal/)
- Minimal dependencies (4 primary: Cobra, Koanf, validator, testify)
- Direct file system operations (no abstraction layers)
- Simple JSON persistence (no database)
- Standard library for most functionality

---

## Post-Design Constitution Re-evaluation

After completing Phase 0 (Research) and Phase 1 (Design), re-evaluating constitution compliance:

### I. Validation-First ✅ PASS
- Design includes complete validation package at internal/validation/
- All workflow transitions will validate artifacts before proceeding
- Retry logic preserved and enhanced in internal/retry/ package
- Exit codes standardized (0=success, 1=failed, 2=exhausted, 3=invalid, 4=missing deps)

### II. Hook-Based Enforcement ✅ PASS (No Change)
- Hooks remain in bash (out of scope for this migration)
- Hooks will continue to call validation logic
- Future migration of hooks is possible but not required

### III. Test-First Development ✅ PASS
- Test structure defined in quickstart.md
- Target: 63-80 tests (exceeds 60+ baseline)
- Unit tests: 35-40 tests across packages
- CLI tests with testscript: 15-20 tests
- Integration tests: 8-12 tests
- Benchmarks: 5-8 tests for performance validation
- All tests must pass before implementation

### IV. Performance Standards ✅ PASS
- Explicit performance contracts defined in validation-api.md
- Targets: startup <50ms, validation <100ms, status <1s
- Benchmarks planned for all performance-critical functions
- Go compiled binaries typically faster than bash scripts

### V. Idempotency & Retry Logic ✅ PASS
- Retry state persists to ~/.autospec/state/retry.json (not /tmp)
- All validation functions are idempotent (multiple calls safe)
- Atomic file writes for state persistence (temp + rename)
- Exit codes support programmatic composition

**Final Gate Status**: ✅ PASS - All constitution principles maintained

**Design Changes from Initial Assessment:**
- Resolved "NEEDS CLARIFICATION" items through research
- Selected battle-tested libraries (Cobra, Koanf) over lighter alternatives for reliability
- Binary size impact: 4-5 MB total (well under 15 MB limit)
- All performance targets achievable with selected architecture

**No Complexity Justifications Required** - Design is simple and aligns with constitution principles.
