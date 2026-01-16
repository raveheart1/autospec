# Implementation Plan: Workflow Progress Indicators

**Branch**: `004-workflow-progress-indicators` | **Date**: 2025-10-23 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/004-workflow-progress-indicators/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

Add visual progress indicators to autospec CLI workflows including phase counters ([1/3], [2/3], etc.), animated spinners for long-running operations (>2s), and completion checkmarks (✓) when phases succeed. This improves user experience by providing real-time feedback during multi-phase workflows (specify → plan → tasks → implement), reducing user anxiety during silent operations, and creating a clear visual record of completed work.

## Technical Context

**Language/Version**: Go 1.25.1
**Primary Dependencies**: Cobra CLI (v1.10.1), briandowns/spinner (v1.23.0+), golang.org/x/term (v0.25.0+)
**Storage**: N/A (progress state is ephemeral, displayed only during execution)
**Testing**: Go test framework (existing), table-driven tests for progress rendering, benchmark tests for performance validation
**Target Platform**: Linux/macOS/Windows cross-platform CLI (terminal emulators: xterm, iTerm2, GNOME Terminal, Windows Terminal, VS Code integrated terminal)
**Project Type**: Single CLI binary
**Performance Goals**: <100ms overhead per progress update, <5% total workflow execution time increase, spinner animation 10 fps (100ms interval)
**Constraints**: Must detect TTY vs pipe/redirect (using term.IsTerminal), must work in non-interactive CI/CD environments, must support Unicode with ASCII fallback (AUTOSPEC_ASCII=1), must respect NO_COLOR environment variable
**Scale/Scope**: 4-5 workflow phases per execution, typical workflow duration 1-5 minutes per phase

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### I. Validation-First ✓
- **Status**: PASS
- **Analysis**: Progress indicators are purely UI enhancements that don't affect workflow validation logic. The existing validation-first approach in `internal/validation/` remains unchanged. Progress indicators will be added as wrapper around existing workflow execution, not modifying the validation pipeline.

### II. Hook-Based Enforcement ✓
- **Status**: PASS
- **Analysis**: No changes to hook-based enforcement. Progress indicators are display-only and don't interfere with validation hooks in `scripts/hooks/`. The feature is purely additive to the user experience layer.

### III. Test-First Development (NON-NEGOTIABLE) ✓
- **Status**: PASS - Commitment Required
- **Plan**: Will write tests BEFORE implementation:
  - Unit tests for progress state tracking (current phase, total phases, status)
  - Unit tests for spinner animation logic (start, stop, update)
  - Unit tests for TTY detection and capability checking
  - Unit tests for progress formatter (render phase counter, checkmarks, spinners)
  - Integration tests for full workflow progress display
  - Benchmark tests to verify <100ms overhead constraint

### IV. Performance Standards ✓
- **Status**: PASS
- **Target**: Progress indicator overhead <100ms per update, <5% total workflow time
- **Strategy**:
  - Spinner animation will use efficient goroutine with ticker (4-10 fps = 100-250ms intervals)
  - Phase transitions are infrequent (1-5 minutes apart) so overhead is negligible
  - TTY detection cached at startup (single syscall)
  - No file I/O or network calls in progress display path
  - Will add benchmark tests to validate performance contract

### V. Idempotency & Retry Logic ✓
- **Status**: PASS
- **Analysis**: Progress indicators are stateless and ephemeral (display-only during execution). They don't affect retry state in `internal/retry/` or workflow orchestration. The feature will respect existing retry behavior and display retry counts in progress output (e.g., "[1/3] Running specify phase (retry 2/3)").

### Quality Standards ✓
- **Go Code Quality**: All new Go code will follow existing patterns (cobra commands, internal packages)
- **Testing**: Will maintain/increase test coverage with new progress indicator tests
- **Documentation**: Will update README with progress indicator behavior and configuration options

### Conclusion
**GATE STATUS: PASS** - All constitution principles satisfied. No violations to justify in Complexity Tracking section. The feature is purely additive, doesn't modify core validation/retry logic, and aligns with all quality standards.

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
internal/
├── progress/              # NEW: Progress indicator package
│   ├── progress.go        # Progress state tracking and display orchestration
│   ├── progress_test.go   # Unit tests for progress tracking
│   ├── spinner.go         # Spinner animation logic
│   ├── spinner_test.go    # Unit tests for spinner
│   ├── terminal.go        # TTY detection and terminal capabilities
│   ├── terminal_test.go   # Unit tests for terminal detection
│   ├── formatter.go       # Progress output formatting
│   └── formatter_test.go  # Unit tests for formatter
├── workflow/              # EXISTING: Modified to integrate progress
│   ├── workflow.go        # Add progress tracking to orchestrator
│   ├── executor.go        # Add progress callbacks to phase execution
│   └── claude.go          # Add progress updates during Claude execution
└── cli/                   # EXISTING: Commands already exist
    ├── full.go            # Add progress display
    ├── workflow.go        # Add progress display
    ├── specify.go         # Add progress display for standalone
    ├── plan.go            # Add progress display for standalone
    ├── tasks.go           # Add progress display for standalone
    └── implement.go       # Add progress display for standalone

tests/                     # EXISTING: Will add new test files
```

**Structure Decision**: Single project (Option 1 from template). New `internal/progress/` package will be created to encapsulate all progress indicator logic. The package follows existing Go project patterns with separate files for different concerns (progress state, spinner animation, terminal detection, formatting). Existing `internal/workflow/` and `internal/cli/` packages will be minimally modified to integrate progress callbacks. This keeps the feature isolated and testable while maintaining separation of concerns.

## Complexity Tracking

No violations - this section is intentionally empty per constitution guidance.

---

## Post-Design Constitution Re-Evaluation

*Required by constitution: Re-check after Phase 1 design*
**Date**: 2025-10-23
**Status**: PASS - No violations introduced during design

### Review Against Principles

#### I. Validation-First ✓
**Re-evaluation**: PASS - Design confirms progress indicators are purely display-layer additions.
- No changes to validation logic in `internal/validation/`
- No changes to workflow validation contracts
- Progress callbacks are optional (nil-safe) and non-blocking
- **Conclusion**: Principle remains satisfied after detailed design.

#### II. Hook-Based Enforcement ✓
**Re-evaluation**: PASS - No impact on hook-based quality gates.
- Progress display doesn't interfere with stop hooks in `scripts/hooks/`
- Validation failures still trigger hooks correctly
- Spinner stops before error output (doesn't interfere with hook messages)
- **Conclusion**: Principle remains satisfied after detailed design.

#### III. Test-First Development (NON-NEGOTIABLE) ✓
**Re-evaluation**: PASS - Comprehensive test plan established.
- Quickstart.md documents test-first approach with specific test cases
- Unit test coverage: types validation, terminal detection, display methods
- Benchmark tests: performance validation (<100ms overhead)
- Integration tests: full workflow with progress
- **Test count estimate**: 20-30 unit tests + 5-10 integration tests + 3 benchmark tests
- **Conclusion**: Principle remains satisfied. Implementation MUST follow test-first approach.

#### IV. Performance Standards ✓
**Re-evaluation**: PASS - Design choices support performance requirements.
- Progress update overhead: <10ms per operation (validated by benchmark tests)
- Spinner animation: 100ms interval = 10 fps (within 4-10 fps target)
- No file I/O in display path (ephemeral state only)
- TTY detection cached at startup (single syscall)
- Total overhead: <5% of workflow time (typical phase is 60-300 seconds, progress adds <1 second)
- **Conclusion**: Principle remains satisfied. Performance contracts defined in API contract.

#### V. Idempotency & Retry Logic ✓
**Re-evaluation**: PASS - Progress display respects existing retry mechanisms.
- Progress state is ephemeral (no persistent state files)
- Display correctly shows retry count via `UpdateRetry()` method
- Spinner lifecycle is stateless (start → stop, no persistent state)
- No interference with `internal/retry/` package
- **Conclusion**: Principle remains satisfied after detailed design.

### Design Artifacts Review

**New Package**: `internal/progress/`
- **Files**: 8 files (4 implementation + 4 test files)
- **External Dependencies**: 2 (briandowns/spinner, golang.org/x/term)
- **Lines of Code Estimate**: ~500 LOC implementation + ~700 LOC tests
- **Complexity**: Low-to-moderate (spinner lifecycle management, terminal detection)

**Modified Packages**:
- `internal/workflow/executor.go`: Added optional progress callbacks (~30 LOC)
- `internal/cli/*.go`: Added progress display instantiation (~15 LOC per command)
- Total modifications: ~150 LOC across 6 files

**Dependency Justification**:
- `briandowns/spinner`: 2.3k stars, actively maintained, battle-tested spinner library
- `golang.org/x/term`: Official Go subrepo, standard for TTY detection
- Both dependencies justified in research.md with alternatives analysis

### Risk Assessment

**Low Risk**:
- Feature is purely additive (doesn't modify existing logic)
- Progress display is optional (nil-safe, backward compatible)
- Isolated in new package (easy to disable/remove if issues)
- No database changes, no config changes, no API changes

**Mitigation Strategies**:
- Comprehensive test coverage (20-30 unit tests)
- Benchmark tests validate performance claims
- Manual testing across multiple terminals documented in quickstart.md
- Can be disabled by not passing progressDisplay to Executor (degradation path)

### Final Verdict

**GATE STATUS: PASS** ✅

All constitution principles remain satisfied after detailed design phase. No complexity violations introduced. Design artifacts (research.md, data-model.md, contracts/, quickstart.md) demonstrate thorough planning consistent with constitution requirements.

**Ready to proceed to Phase 2** (/speckit.tasks command) for task breakdown and implementation planning.
