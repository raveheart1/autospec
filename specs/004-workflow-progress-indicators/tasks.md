# Tasks: Workflow Progress Indicators

**Input**: Design documents from `/specs/004-workflow-progress-indicators/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/progress_package_api.md

**Tests**: Test-first development is REQUIRED per constitution principle III. Tests must be written BEFORE implementation and must FAIL before implementation begins.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `- [ ] [ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

Single Go project at repository root: `internal/`, `cmd/`, tests in `*_test.go` files

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Package structure and type definitions

- [ ] T001 Create internal/progress package directory structure (progress/, types.go, terminal.go, display.go, formatter.go)
- [ ] T002 Add dependencies to go.mod: github.com/briandowns/spinner v1.23.0+ and golang.org/x/term v0.25.0+

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core types and terminal detection that ALL user stories depend on

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [ ] T003 [P] Write unit tests for PhaseStatus enum in internal/progress/types_test.go
- [ ] T004 [P] Write unit tests for PhaseInfo.Validate() in internal/progress/types_test.go (test all validation rules)
- [ ] T005 [P] Write unit tests for TerminalCapabilities in internal/progress/terminal_test.go (test NO_COLOR, AUTOSPEC_ASCII env vars)
- [ ] T006 Define PhaseStatus enum constants in internal/progress/types.go (PhasePending, PhaseInProgress, PhaseCompleted, PhaseFailed)
- [ ] T007 Implement PhaseStatus.String() method in internal/progress/types.go
- [ ] T008 Define PhaseInfo struct in internal/progress/types.go
- [ ] T009 Implement PhaseInfo.Validate() method in internal/progress/types.go
- [ ] T010 Define TerminalCapabilities struct in internal/progress/types.go
- [ ] T011 Define ProgressSymbols struct in internal/progress/types.go
- [ ] T012 [P] Write unit tests for DetectTerminalCapabilities() in internal/progress/terminal_test.go
- [ ] T013 [P] Write unit tests for selectSymbols() helper in internal/progress/terminal_test.go
- [ ] T014 Implement DetectTerminalCapabilities() in internal/progress/terminal.go using term.IsTerminal()
- [ ] T015 Implement selectSymbols() helper in internal/progress/terminal.go (Unicode vs ASCII based on capabilities)
- [ ] T016 Write benchmark test for terminal detection in internal/progress/terminal_bench_test.go (verify <10ms)

**Checkpoint**: Foundation ready - user story implementation can now begin in parallel

---

## Phase 3: User Story 1 - Phase Progress Counter (Priority: P1) 🎯 MVP

**Goal**: Display phase progress as [current/total] format (e.g., [1/3], [2/3], [3/3]) so users know which phase is running and how many remain

**Independent Test**: Run any multi-phase workflow and verify phase counter increments correctly ([1/3] → [2/3] → [3/3])

### Tests for User Story 1

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation**

- [ ] T017 [P] [US1] Write unit tests for ProgressDisplay.StartPhase() in internal/progress/display_test.go (verify [N/Total] format in output)
- [ ] T018 [P] [US1] Write unit tests for phase counter rendering in TTY mode in internal/progress/display_test.go
- [ ] T019 [P] [US1] Write unit tests for phase counter rendering in non-TTY mode in internal/progress/display_test.go
- [ ] T020 [P] [US1] Write unit tests for retry count display in internal/progress/display_test.go (verify "(retry X/MaxRetries)" suffix)

### Implementation for User Story 1

- [ ] T021 [US1] Define ProgressDisplay struct in internal/progress/display.go
- [ ] T022 [US1] Implement NewProgressDisplay() constructor in internal/progress/display.go
- [ ] T023 [US1] Implement StartPhase() method in internal/progress/display.go (format: [N/Total] Running <name> phase)
- [ ] T024 [US1] Implement UpdateRetry() method in internal/progress/display.go (wrapper around StartPhase with retry count)
- [ ] T025 [US1] Add helper method to format phase counter string in internal/progress/formatter.go
- [ ] T026 [US1] Add helper method to build phase message in internal/progress/formatter.go

**Checkpoint**: At this point, phase progress counters should display correctly in workflows

---

## Phase 4: User Story 2 - Activity Spinners (Priority: P2)

**Goal**: Show animated spinners during long-running operations (>2s) so users know the system hasn't frozen

**Independent Test**: Run any single long-running command (e.g., autospec specify) and verify spinner animates while Claude is processing

### Tests for User Story 2

- [ ] T027 [P] [US2] Write unit tests for spinner initialization in TTY mode in internal/progress/display_test.go
- [ ] T028 [P] [US2] Write unit tests for spinner disabled in non-TTY mode in internal/progress/display_test.go
- [ ] T029 [P] [US2] Write integration test for spinner lifecycle (start → stop) in internal/progress/display_test.go
- [ ] T030 [P] [US2] Write benchmark test for spinner animation CPU usage in internal/progress/display_bench_test.go (verify <0.1% CPU)

### Implementation for User Story 2

- [ ] T031 [US2] Extend StartPhase() to create and start spinner if IsTTY in internal/progress/display.go
- [ ] T032 [US2] Configure spinner with 100ms update interval (10 fps) in internal/progress/display.go
- [ ] T033 [US2] Add spinner cleanup logic to handle goroutine lifecycle in internal/progress/display.go
- [ ] T034 [US2] Implement helper to select spinner character set based on Unicode support in internal/progress/terminal.go
- [ ] T035 [US2] Add spinner field to ProgressDisplay struct in internal/progress/display.go

**Checkpoint**: At this point, spinners should animate during long-running operations in TTY mode

---

## Phase 5: User Story 3 - Completion Checkmarks (Priority: P3)

**Goal**: Display visual checkmarks (✓) when phases complete successfully and failure indicators (✗) when phases fail

**Independent Test**: Run a workflow to completion and verify each completed phase shows a checkmark

### Tests for User Story 3

- [ ] T036 [P] [US3] Write unit tests for CompletePhase() checkmark rendering in internal/progress/display_test.go
- [ ] T037 [P] [US3] Write unit tests for FailPhase() failure indicator rendering in internal/progress/display_test.go
- [ ] T038 [P] [US3] Write unit tests for color support (green checkmark, red X) in internal/progress/display_test.go
- [ ] T039 [P] [US3] Write unit tests for NO_COLOR environment variable in internal/progress/display_test.go
- [ ] T040 [P] [US3] Write unit tests for ASCII fallback ([OK], [FAIL]) in internal/progress/display_test.go

### Implementation for User Story 3

- [ ] T041 [US3] Implement CompletePhase() method in internal/progress/display.go (stop spinner, show checkmark)
- [ ] T042 [US3] Implement FailPhase() method in internal/progress/display.go (stop spinner, show failure indicator)
- [ ] T043 [US3] Add checkmark() helper method for symbol selection in internal/progress/formatter.go
- [ ] T044 [US3] Add failure() helper method for symbol selection in internal/progress/formatter.go
- [ ] T045 [US3] Implement ANSI color code rendering (green for success, red for failure) in internal/progress/formatter.go
- [ ] T046 [US3] Implement color/Unicode fallback logic in internal/progress/formatter.go

**Checkpoint**: All user stories should now be independently functional - phase counters, spinners, and completion indicators all work

---

## Phase 6: Integration with Workflow Executor

**Purpose**: Connect progress display to existing workflow execution logic

- [ ] T047 Add progressDisplay field to Executor struct in internal/workflow/executor.go
- [ ] T048 Update NewExecutor() constructor signature to accept optional ProgressDisplay in internal/workflow/executor.go
- [ ] T049 [P] Write unit tests for Executor with progress display in internal/workflow/executor_test.go
- [ ] T050 [P] Write integration tests for full workflow with progress in internal/workflow/workflow_test.go
- [ ] T051 Add helper method getPhaseNumber() to map Phase enum to sequential numbers in internal/workflow/executor.go
- [ ] T052 Add helper method getTotalPhases() to return total phase count in internal/workflow/executor.go
- [ ] T053 Add helper method buildPhaseInfo() to construct PhaseInfo from Phase enum in internal/workflow/executor.go
- [ ] T054 Modify ExecutePhase() to call display.StartPhase() before execution in internal/workflow/executor.go
- [ ] T055 Modify ExecutePhase() to call display.CompletePhase() on success in internal/workflow/executor.go
- [ ] T056 Modify ExecutePhase() to call display.FailPhase() on error in internal/workflow/executor.go
- [ ] T057 Add nil-safety checks for optional progressDisplay field in internal/workflow/executor.go
- [ ] T058 Update retry logic to call display.UpdateRetry() on retry attempts in internal/workflow/executor.go

**Checkpoint**: Workflow executor now integrates with progress display

---

## Phase 7: Integration with CLI Commands

**Purpose**: Enable progress indicators in CLI commands

- [ ] T059 [P] Modify workflow.go to detect terminal capabilities and create ProgressDisplay in internal/cli/workflow.go
- [ ] T060 [P] Modify full.go to detect terminal capabilities and create ProgressDisplay in internal/cli/full.go
- [ ] T061 [P] Modify specify.go to create ProgressDisplay for standalone phase in internal/cli/specify.go
- [ ] T062 [P] Modify plan.go to create ProgressDisplay for standalone phase in internal/cli/plan.go
- [ ] T063 [P] Modify tasks.go to create ProgressDisplay for standalone phase in internal/cli/tasks.go
- [ ] T064 [P] Modify implement.go to create ProgressDisplay for standalone phase in internal/cli/implement.go
- [ ] T065 Update all CLI commands to pass ProgressDisplay to NewExecutor() in internal/cli/*.go
- [ ] T066 Add logic to disable progress display when not TTY (piped/redirected output) in internal/cli/*.go

**Checkpoint**: All CLI commands now show progress indicators in interactive mode

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: Performance validation, documentation, and edge case handling

- [ ] T067 [P] Write benchmark test for StartPhase() overhead in internal/progress/display_bench_test.go (verify <10ms)
- [ ] T068 [P] Write benchmark test for CompletePhase() overhead in internal/progress/display_bench_test.go (verify <10ms)
- [ ] T069 [P] Add integration test for full workflow performance (verify <5% overhead) in internal/workflow/workflow_test.go
- [ ] T070 [P] Add test for terminal resize handling in internal/progress/display_test.go
- [ ] T071 [P] Add test for phase completion under 1 second (no spinner shown) in internal/progress/display_test.go
- [ ] T072 Manual testing: Verify progress indicators in xterm terminal emulator
- [ ] T073 Manual testing: Verify progress indicators in iTerm2 terminal emulator
- [ ] T074 Manual testing: Verify progress indicators in GNOME Terminal emulator
- [ ] T075 Manual testing: Verify progress indicators in Windows Terminal emulator
- [ ] T076 Manual testing: Verify progress indicators in VS Code integrated terminal
- [ ] T077 Manual testing: Test piped output (autospec workflow "test" | cat) - verify no ANSI codes
- [ ] T078 Manual testing: Test with NO_COLOR=1 environment variable
- [ ] T079 Manual testing: Test with AUTOSPEC_ASCII=1 environment variable
- [ ] T080 Update README.md with progress indicator documentation (behavior, environment variables)
- [ ] T081 Update CLAUDE.md active technologies section with new dependencies
- [ ] T082 Run go fmt ./... to format all code
- [ ] T083 Run go vet ./... to check for issues
- [ ] T084 Run make test to verify all tests pass
- [ ] T085 Run make lint to verify linting passes
- [ ] T086 Run make build to verify binary builds successfully
- [ ] T087 Run ./autospec doctor to verify dependencies are available
- [ ] T088 Run quickstart.md validation steps manually

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Story 1 (Phase 3)**: Depends on Foundational (Phase 2) - provides base progress display
- **User Story 2 (Phase 4)**: Depends on Foundational (Phase 2) AND User Story 1 - extends StartPhase with spinner
- **User Story 3 (Phase 5)**: Depends on Foundational (Phase 2) AND User Story 1 - adds completion methods
- **Integration (Phase 6)**: Depends on User Stories 1, 2, 3 completion
- **CLI Integration (Phase 7)**: Depends on Integration (Phase 6)
- **Polish (Phase 8)**: Depends on CLI Integration (Phase 7)

### User Story Dependencies

- **User Story 1 (P1)**: Foundation only - no dependency on other stories (displays phase counter)
- **User Story 2 (P2)**: Depends on User Story 1 (extends StartPhase with spinner animation)
- **User Story 3 (P3)**: Depends on User Story 1 (adds CompletePhase and FailPhase methods)

### Within Each User Story

- Tests MUST be written and FAIL before implementation
- Types before methods
- Unit tests before implementation
- Benchmark tests after implementation
- Story complete before moving to next priority

### Parallel Opportunities

- **Phase 1**: T001 and T002 can run in parallel
- **Phase 2**:
  - Test tasks T003, T004, T005 can run in parallel
  - Test tasks T012, T013 can run in parallel
  - Type definitions T006-T011 can run in parallel after tests written
  - Implementation T014, T015 can run in parallel after types defined
- **User Story 1 Tests**: T017, T018, T019, T020 can run in parallel
- **User Story 1 Implementation**: T025, T026 can run in parallel
- **User Story 2 Tests**: T027, T028, T029, T030 can run in parallel
- **User Story 3 Tests**: T036, T037, T038, T039, T040 can run in parallel
- **User Story 3 Implementation**: T043, T044 can run in parallel
- **Integration Tests**: T049, T050 can run in parallel
- **CLI Integration**: T059-T064 can run in parallel (different files)
- **Polish**: Benchmark tests T067, T068, T069 can run in parallel; manual tests T072-T079 can run in parallel

---

## Parallel Example: User Story 1

```bash
# Launch all tests for User Story 1 together:
Task: "Write unit tests for ProgressDisplay.StartPhase() in internal/progress/display_test.go"
Task: "Write unit tests for phase counter rendering in TTY mode in internal/progress/display_test.go"
Task: "Write unit tests for phase counter rendering in non-TTY mode in internal/progress/display_test.go"
Task: "Write unit tests for retry count display in internal/progress/display_test.go"

# Launch formatter helpers together:
Task: "Add helper method to format phase counter string in internal/progress/formatter.go"
Task: "Add helper method to build phase message in internal/progress/formatter.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (CRITICAL - blocks all stories)
3. Complete Phase 3: User Story 1
4. **STOP and VALIDATE**: Test phase progress counters independently
5. If working, proceed to User Story 2 (spinners)

### Incremental Delivery

1. Complete Setup + Foundational → Foundation ready
2. Add User Story 1 → Test independently → Phase counters working (basic MVP!)
3. Add User Story 2 → Test independently → Spinners animating
4. Add User Story 3 → Test independently → Completion indicators showing
5. Add Integration → Full workflow with progress
6. Add CLI Integration → All commands have progress
7. Each increment adds value without breaking previous functionality

### Parallel Team Strategy

With multiple developers:

1. Team completes Setup + Foundational together
2. Once Foundational is done:
   - Developer A: User Story 1 (phase counters)
   - Developer B: User Story 2 (spinners) - starts after US1 basics done
   - Developer C: User Story 3 (checkmarks) - starts after US1 basics done
3. Once all user stories complete:
   - Developer A: Integration with Executor
   - Developer B: Integration with CLI commands
4. All developers: Polish and testing

---

## Task Count Summary

- **Phase 1 (Setup)**: 2 tasks
- **Phase 2 (Foundational)**: 14 tasks (6 test tasks, 8 implementation tasks)
- **Phase 3 (User Story 1)**: 10 tasks (4 test tasks, 6 implementation tasks)
- **Phase 4 (User Story 2)**: 9 tasks (4 test tasks, 5 implementation tasks)
- **Phase 5 (User Story 3)**: 11 tasks (5 test tasks, 6 implementation tasks)
- **Phase 6 (Integration)**: 12 tasks (2 test tasks, 10 implementation tasks)
- **Phase 7 (CLI Integration)**: 8 tasks
- **Phase 8 (Polish)**: 22 tasks (5 test tasks, 11 manual tests, 6 validation/doc tasks)

**Total**: 88 tasks
- **Test tasks**: 21 unit/integration test tasks
- **Benchmark tasks**: 4 benchmark test tasks
- **Manual test tasks**: 11 manual test tasks
- **Implementation tasks**: 52 implementation tasks

**Parallel opportunities**: 28 tasks marked [P] can run in parallel with other [P] tasks in their phase

---

## Notes

- [P] tasks = different files, no dependencies within phase
- [Story] label maps task to specific user story for traceability
- Each user story should be independently completable and testable
- Verify tests fail before implementing (test-first development)
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- Run `make test` frequently to ensure no regressions
- Use `go test -v ./internal/progress/` to run package-specific tests
- Use `go test -bench=. ./internal/progress/` to run benchmarks
