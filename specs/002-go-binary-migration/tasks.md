# Tasks: Go Binary Migration

**Input**: Design documents from `/specs/002-go-binary-migration/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/, quickstart.md

**Tests**: Tests are NOT required for this feature based on spec.md. However, the constitution requires test-first development with 60+ tests minimum. These tests will validate the Go implementation maintains parity with existing bash functionality.

**⚠️ CRITICAL DEVELOPMENT WARNING ⚠️**: **NEVER RUN `claude -p` DURING DEVELOPMENT** - this will incur real API costs! Always use mocked implementations for Claude CLI commands in tests. All integration tests must mock external CLI tools (claude, specify, git) rather than invoking them directly. Use testscript's condition system, TestMain hijacking, or mock executables in test fixtures.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- Single project at repository root
- Go code: `cmd/autospec/`, `internal/*/`
- Tests: `internal/*/test.go`, `integration/*_test.go`

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and basic Go structure

- [X] T001 Initialize Go module at repository root with `go mod init github.com/username/auto-claude-speckit`
- [X] T002 Create directory structure: cmd/autospec/, internal/{cli,config,validation,retry,git,spec,workflow}/, integration/
- [X] T003 [P] Add initial dependencies: cobra, koanf, validator, testify, testscript to go.mod
- [X] T004 [P] Create cmd/autospec/main.go entry point with basic structure
- [X] T005 [P] Create internal/cli/root.go with Cobra root command and global flags
- [X] T006 [P] Update .gitignore to ignore dist/, *.test, coverage files, go binary outputs

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [X] T007 Implement internal/config/config.go with Configuration struct per data-model.md
- [X] T008 [P] Implement internal/config/defaults.go with default configuration values
- [X] T009 [P] Add Koanf-based configuration loading with override hierarchy (env > local > global) in internal/config/config.go
- [X] T010 Add go-playground/validator validation for Configuration struct in internal/config/config.go
- [X] T011 [P] Write unit tests for configuration loading in internal/config/config_test.go (5-8 table-driven tests)
- [X] T012 Implement internal/git/git.go with GetCurrentBranch, GetRepositoryRoot, IsGitRepository functions per validation-api.md
- [X] T013 [P] Write unit tests for git operations in internal/git/git_test.go using TestMain hijacking pattern (3-5 tests)
- [X] T014 Implement internal/spec/spec.go with DetectCurrentSpec and GetSpecDirectory functions per validation-api.md
- [X] T015 [P] Write unit tests for spec detection in internal/spec/spec_test.go (4-6 table-driven tests)
- [X] T016 Implement internal/validation/validation.go with ValidateSpecFile, ValidatePlanFile, ValidateTasksFile per validation-api.md
- [X] T017 Implement internal/validation/tasks.go with CountUncheckedTasks, ValidateTasksComplete, ParseTasksByPhase functions per validation-api.md
- [X] T018 Implement internal/validation/prompt.go with ListIncompletePhasesWithTasks and GenerateContinuationPrompt functions
- [X] T019 [P] Write unit tests for file validation in internal/validation/validation_test.go (6-8 table-driven tests)
- [X] T020 [P] Write unit tests for task parsing in internal/validation/tasks_test.go (8-10 table-driven tests covering patterns)
- [X] T021 [P] Write benchmarks for validation functions in internal/validation/validation_bench_test.go (3-5 benchmarks)
- [X] T022 Implement internal/retry/retry.go with RetryState struct and LoadRetryState, SaveRetryState functions per data-model.md and validation-api.md
- [X] T023 Implement internal/retry/state.go with IncrementRetryCount, ResetRetryCount, CanRetry methods per validation-api.md
- [X] T024 [P] Write unit tests for retry state management in internal/retry/retry_test.go (6-8 tests with temp directories)
- [X] T025 Create internal/workflow/executor.go with command execution logic and retry handling
- [X] T026 [P] Create internal/workflow/claude.go with Claude CLI execution using custom_claude_cmd template support
- [X] T027 [P] Create internal/workflow/preflight.go with pre-flight validation checks per FR-015 through FR-024
- [X] T028 Implement internal/workflow/workflow.go with specify→plan→tasks orchestration per cli-interface.md
- [X] T029 [P] Write unit tests for workflow orchestration in internal/workflow/workflow_test.go (5-7 tests)

**Checkpoint**: Foundation ready - user story implementation can now begin in parallel

---

## Phase 3: User Story 1 - Simple Installation (Priority: P1) 🎯 MVP

**Goal**: Developers can install with a single command (`go install` or binary download) and immediately use `autospec --version`

**Independent Test**: Run `go install github.com/username/autospec@latest` followed by `autospec --version` - should complete in <30 seconds total and display version info in <50ms

### Implementation for User Story 1

- [X] T030 [P] [US1] Implement internal/cli/version.go with version command per cli-interface.md
- [X] T031 [P] [US1] Add version variables (Version, Commit, BuildDate) to internal/cli/version.go with ldflags support
- [X] T032 [US1] Wire version command to root command in internal/cli/root.go
- [X] T033 [P] [US1] Create scripts/build-all.sh for cross-platform builds per quickstart.md
- [X] T034 [P] [US1] Add ldflags for version injection in scripts/build-all.sh
- [X] T035 [US1] Test build process produces binary <15MB for all platforms (Linux amd64/arm64, macOS amd64/arm64, Windows amd64)
- [X] T036 [P] [US1] Write testscript test for version command in cmd/autospec/testdata/scripts/version.txt
- [X] T037 [P] [US1] Add benchmark for startup time in cmd/autospec/main_test.go (target: <50ms)
- [X] T038 [US1] Update README.md with installation instructions (`go install` and binary download)

**Checkpoint**: User Story 1 complete - users can install and verify installation with `autospec --version`

---

## Phase 4: User Story 2 - Cross-Platform Compatibility (Priority: P1)

**Goal**: Same `autospec workflow` command works identically on Windows, macOS, and Linux

**Independent Test**: Run `autospec workflow "test feature"` on all three platforms - identical behavior and output

### Implementation for User Story 2

- [X] T039 [P] [US2] Ensure all file paths use filepath.Join() in internal/config/config.go
- [X] T040 [P] [US2] Ensure all file paths use filepath.Join() in internal/validation/validation.go
- [X] T041 [P] [US2] Ensure all file paths use filepath.Join() in internal/retry/retry.go
- [X] T042 [P] [US2] Ensure all file paths use filepath.Join() in internal/spec/spec.go
- [X] T043 [P] [US2] Ensure all file paths use filepath.Join() in internal/workflow/workflow.go
- [X] T044 [US2] Handle home directory expansion (~/) correctly for Windows in internal/config/config.go
- [X] T045 [US2] Test path handling on Windows, macOS, Linux for config file locations
- [X] T046 [P] [US2] Add integration test for cross-platform config loading in integration/config_test.go
- [X] T047 [P] [US2] Add integration test for cross-platform retry state persistence in integration/retry_test.go
- [X] T048 [US2] Verify git command execution works on Windows in internal/git/git_test.go
- [X] T049 [US2] Test complete workflow on Windows, macOS, Linux platforms
- [X] T050 [P] [US2] Document platform-specific requirements in README.md (git must be in PATH)

**Checkpoint**: User Story 2 complete - tool works identically across all platforms

---

## Phase 5: User Story 3 - Pre-Flight Validation (Priority: P2)

**Goal**: Clear, actionable feedback when project isn't initialized with SpecKit

**Independent Test**: Run `autospec workflow` in uninitialized directory - see helpful warning with fix instructions

### Implementation for User Story 3

- [X] T051 [P] [US3] Implement directory existence checks in internal/workflow/preflight.go
- [X] T052 [P] [US3] Implement CLI dependency checks (claude, specify) in internal/workflow/preflight.go
- [X] T053 [US3] Implement git root detection for helpful error messages in internal/workflow/preflight.go
- [X] T054 [US3] Generate warning message listing missing directories in internal/workflow/preflight.go
- [X] T055 [US3] Add user prompt "Do you want to continue anyway? [y/N]" in internal/workflow/preflight.go
- [X] T056 [US3] Implement --skip-preflight flag handling in internal/cli/workflow.go
- [X] T057 [US3] Add pre-flight checks to workflow command in internal/cli/workflow.go
- [X] T058 [P] [US3] Write unit tests for pre-flight validation in internal/workflow/preflight_test.go (6-8 tests)
- [X] T059 [P] [US3] Add benchmark for pre-flight validation in internal/workflow/preflight_test.go (target: <100ms)
- [X] T060 [P] [US3] Write testscript test for pre-flight warnings in cmd/autospec/testdata/scripts/preflight.txt
- [X] T061 [US3] Test pre-flight check with missing .claude/commands/ and .specify/ directories

**Checkpoint**: User Story 3 complete - helpful warnings guide users through setup issues

---

## Phase 6: User Story 4 - Custom Claude Command Configuration (Priority: P2)

**Goal**: Power users can configure custom Claude command templates with pipes and environment variables

**Independent Test**: Configure `custom_claude_cmd` in .autospec/config.json with pipe and verify it executes correctly

### Implementation for User Story 4

- [X] T062 [P] [US4] Add custom_claude_cmd field to Configuration struct in internal/config/config.go
- [X] T063 [P] [US4] Add template validation for {{PROMPT}} placeholder in internal/config/config.go
- [X] T064 [US4] Implement template expansion in internal/workflow/claude.go
- [X] T065 [US4] Parse environment variable prefixes (e.g., ANTHROPIC_API_KEY="") in internal/workflow/claude.go
- [X] T066 [US4] Parse pipe operators in custom_claude_cmd in internal/workflow/claude.go
- [X] T067 [US4] Execute full command pipeline with proper shell escaping in internal/workflow/claude.go
- [X] T068 [US4] Stream output to stdout in real-time in internal/workflow/claude.go
- [X] T069 [US4] Fallback to simple mode (claude_cmd + claude_args) when custom_claude_cmd not set in internal/workflow/claude.go
- [X] T070 [P] [US4] Write unit tests for template parsing in internal/workflow/claude_test.go (5-7 tests)
- [X] T071 [P] [US4] Add integration test for custom command execution in integration/claude_test.go
- [X] T072 [US4] Test custom command with pipe operator (`| claude-clean`)
- [X] T073 [US4] Test custom command with environment variable prefix
- [X] T074 [P] [US4] Document custom_claude_cmd configuration in README.md with examples

**Checkpoint**: User Story 4 complete - custom Claude command templates work with pipes and env vars

---

## Phase 7: User Story 5 - Automated Validation and Retry (Priority: P3)

**Goal**: Automatic validation of each phase output with intelligent retry logic

**Independent Test**: Run `autospec workflow` and intentionally cause failure - verify automatic retry up to max attempts

### Implementation for User Story 5

- [X] T075 [US5] Implement internal/cli/workflow.go command per cli-interface.md
- [X] T076 [US5] Wire workflow command to root in internal/cli/root.go
- [X] T077 [US5] Integrate validation after each phase (specify, plan, tasks) in internal/workflow/workflow.go
- [X] T078 [US5] Implement retry logic with LoadRetryState/IncrementRetryCount in internal/workflow/executor.go
- [X] T079 [US5] Reset retry count on successful validation in internal/workflow/executor.go
- [X] T080 [US5] Generate continuation prompt on retry exhaustion in internal/workflow/executor.go
- [X] T081 [US5] Stream Claude output to stdout during execution in internal/workflow/claude.go
- [X] T082 [P] [US5] Write integration tests for retry logic in integration/retry_test.go (4-6 tests)
- [X] T083 [P] [US5] Add testscript test for workflow execution in cmd/autospec/testdata/scripts/workflow.txt
- [X] T084 [US5] Test retry exhaustion scenario (max retries reached)
- [X] T085 [US5] Test retry reset on success scenario

**Checkpoint**: User Story 5 complete - automatic validation and retry works reliably

---

## Phase 8: User Story 6 - Fast Performance (Priority: P3)

**Goal**: Validation and workflow commands complete in <5 seconds for typical operations

**Independent Test**: Measure `autospec status` execution time - should be <1 second

### Implementation for User Story 6

- [X] T086 [P] [US6] Implement internal/cli/status.go command per cli-interface.md
- [X] T087 [US6] Wire status command to root in internal/cli/root.go
- [X] T088 [US6] Use ParseTasksByPhase to display progress in internal/cli/status.go
- [X] T089 [US6] Display phase progress with checked/unchecked counts in internal/cli/status.go
- [X] T090 [US6] List next 3-5 unchecked tasks in internal/cli/status.go
- [X] T091 [P] [US6] Add benchmark for status command in internal/cli/status_test.go (target: <1s)
- [X] T092 [P] [US6] Add testscript test for status command in cmd/autospec/testdata/scripts/status.txt
- [X] T093 [US6] Optimize CountUncheckedTasks to use grep -q patterns for speed
- [X] T094 [US6] Optimize ParseTasksByPhase to avoid unnecessary string allocations
- [X] T095 [P] [US6] Profile status command with pprof to identify bottlenecks
- [X] T096 [US6] Verify all validation functions meet performance contracts from validation-api.md

**Checkpoint**: User Story 6 complete - performance targets met across all commands

---

## Phase 9: Additional CLI Commands

**Purpose**: Complete remaining CLI commands not tied to specific user stories

- [X] T097 [P] Implement internal/cli/init.go command per cli-interface.md
- [X] T098 [P] Implement internal/cli/specify.go command per cli-interface.md
- [X] T099 [P] Implement internal/cli/plan.go command per cli-interface.md
- [X] T100 [P] Implement internal/cli/tasks.go command per cli-interface.md
- [X] T101 [P] Implement internal/cli/implement.go command per cli-interface.md
- [X] T102 [P] Implement internal/cli/config.go command per cli-interface.md
- [X] T103 Wire init command to root in internal/cli/root.go
- [X] T104 Wire specify command to root in internal/cli/root.go
- [X] T105 Wire plan command to root in internal/cli/root.go
- [X] T106 Wire tasks command to root in internal/cli/root.go
- [X] T107 Wire implement command to root in internal/cli/root.go
- [X] T108 Wire config command to root in internal/cli/root.go
- [X] T109 [P] Write testscript tests for init command in cmd/autospec/testdata/scripts/init.txt
- [X] T110 [P] Write testscript tests for specify command in cmd/autospec/testdata/scripts/specify.txt
- [X] T111 [P] Write testscript tests for plan command in cmd/autospec/testdata/scripts/plan.txt
- [X] T112 [P] Write testscript tests for tasks command in cmd/autospec/testdata/scripts/tasks.txt
- [X] T113 [P] Write testscript tests for implement command in cmd/autospec/testdata/scripts/implement.txt
- [X] T114 [P] Write testscript tests for config command in cmd/autospec/testdata/scripts/config.txt

---

## Phase 10: Integration Testing

**Purpose**: End-to-end workflow validation

- [X] T115 [P] Write integration test for complete workflow (specify→plan→tasks) in integration/workflow_test.go
- [X] T116 [P] Write integration test for implementation progress tracking in integration/implement_test.go
- [X] T117 [P] Write integration test for configuration override hierarchy in integration/config_test.go
- [X] T118 [P] Write integration test for retry state persistence in integration/retry_test.go
- [X] T119 Create integration/testdata/fixtures/ with sample spec directories for testing
- [X] T120 Create integration/testdata/golden/ with expected output files for comparison
- [X] T121 Add TestMain for integration tests to set up mock Claude/specify commands in integration/workflow_test.go
- [X] T122 Test workflow with missing dependencies (claude not in PATH)
- [X] T123 Test workflow with corrupted retry state file
- [X] T124 Test workflow with invalid configuration file

---

## Phase 11: Performance Validation & Benchmarking

**Purpose**: Ensure all performance contracts are met

- [X] T125 [P] Run all benchmarks and verify performance targets met
- [X] T126 Verify binary startup time <50ms with `time ./autospec version`
- [X] T127 Verify status command <1s with realistic tasks.md file
- [X] T128 Verify pre-flight validation <100ms
- [X] T129 Verify validation functions meet contracts: ValidateSpecFile <10ms, CountUncheckedTasks <50ms, ParseTasksByPhase <100ms
- [X] T130 Create scripts/benchmark.sh to run performance comparison bash vs Go
- [X] T131 Document performance improvements in README.md (expected: 2-5x faster)
- [X] T132 Use benchstat to compare old (bash) vs new (Go) performance

---

## Phase 12: Cross-Platform Build & Release

**Purpose**: Build binaries for all supported platforms

- [X] T133 [P] Build Linux amd64 binary with scripts/build-all.sh
- [X] T134 [P] Build Linux arm64 binary with scripts/build-all.sh
- [X] T135 [P] Build macOS amd64 binary with scripts/build-all.sh
- [X] T136 [P] Build macOS arm64 binary with scripts/build-all.sh
- [X] T137 [P] Build Windows amd64 binary with scripts/build-all.sh
- [X] T138 Verify all binaries are <15MB (target: 4-5MB per research.md)
- [X] T139 Test Linux amd64 binary on Ubuntu 20.04+
- [X] T140 Test macOS amd64 binary on macOS 12+
- [X] T141 Test macOS arm64 binary on Apple Silicon
- [X] T142 Test Windows amd64 binary on Windows 10+
- [X] T143 Create dist/ directory structure for releases
- [X] T144 Generate SHA256 checksums for all binaries
- [X] T145 Create release notes template

---

## Phase 13: Documentation & Migration

**Purpose**: Update documentation and prepare for bash deprecation

- [X] T146 [P] Update README.md with Go installation instructions
- [X] T147 [P] Update README.md with usage examples for all commands
- [X] T148 [P] Update CLAUDE.md with Go development commands (go test, go build, etc.)
- [X] T149 [P] Update CLAUDE.md with new architecture overview (Go packages)
- [X] T150 Update constitution.md if any principles changed
- [X] T151 [P] Create migration guide from bash to Go binary
- [X] T152 Document differences between bash and Go implementations (if any)
- [X] T153 Create legacy/ directory and move bash scripts there
- [X] T154 Add deprecation notice to bash scripts
- [X] T155 Update all references to bash scripts to point to Go binary

---

## Phase 14: Polish & Cross-Cutting Concerns

**Purpose**: Final improvements and validations

- [X] T156 [P] Run `go vet ./...` and fix all issues
- [X] T157 [P] Run `golangci-lint run` and fix critical issues
- [X] T158 [P] Run `go fmt ./...` to ensure consistent formatting
- [X] T159 Verify all unit tests pass: `go test ./internal/...`
- [X] T160 Verify all integration tests pass: `go test ./integration/...`
- [X] T161 Verify all CLI tests pass: `go test ./cmd/autospec/...`
- [X] T162 Generate coverage report: `go test -coverprofile=cover.out ./...`
- [X] T163 Verify test coverage >80% (target per quickstart.md)
- [X] T164 Verify 60+ tests minimum (constitution requirement)
- [X] T165 [P] Add error handling for edge cases identified in spec.md
- [X] T166 [P] Add security hardening (input validation, path traversal prevention)
- [X] T167 Create quickstart validation test based on quickstart.md examples
- [X] T168 Test all quickstart.md examples work end-to-end
- [X] T169 Final cross-platform smoke test on all platforms
- [X] T170 Create GitHub release with binaries and release notes

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3-8)**: All depend on Foundational phase completion
  - User stories can then proceed in parallel (if staffed)
  - Or sequentially in priority order (P1 → P2 → P3)
- **Additional CLI Commands (Phase 9)**: Depends on User Stories 1, 2, 5 (needs workflow infrastructure)
- **Integration Testing (Phase 10)**: Depends on Phase 9 completion (all commands implemented)
- **Performance Validation (Phase 11)**: Can run in parallel with Phase 9-10
- **Cross-Platform Build (Phase 12)**: Depends on Phase 10 (all tests passing)
- **Documentation (Phase 13)**: Can run in parallel with Phase 11-12
- **Polish (Phase 14)**: Depends on all previous phases

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational (Phase 2) - No dependencies on other stories
- **User Story 2 (P1)**: Can start after Foundational (Phase 2) - No dependencies on other stories
- **User Story 3 (P2)**: Depends on User Story 2 (needs workflow command structure)
- **User Story 4 (P2)**: Can start after Foundational (Phase 2) - No dependencies on other stories
- **User Story 5 (P3)**: Depends on User Stories 1, 2, 4 (needs installation, cross-platform, custom commands)
- **User Story 6 (P3)**: Depends on User Story 5 (needs workflow implementation to benchmark)

### Within Each Phase

- Tasks marked [P] can run in parallel within the same phase
- Tasks without [P] have dependencies on previous tasks in the phase
- Foundation tasks MUST complete before any user story work begins

### Parallel Opportunities

**Phase 1 - Setup**: Tasks T003, T004, T005, T006 can run in parallel after T001, T002

**Phase 2 - Foundational**:
- T008, T009, T012 can run in parallel after T007
- T011, T013, T015, T019, T020, T021, T024, T029 can run in parallel (test files)
- Models and test files can be developed in parallel

**Phase 3 - User Story 1**: Tasks T030, T031, T033, T034, T036, T037 can run in parallel

**Phase 4 - User Story 2**: Tasks T039-T043, T046, T047, T050 can run in parallel

**Phase 5 - User Story 3**: Tasks T051, T052, T058, T059, T060 can run in parallel

**Phase 6 - User Story 4**: Tasks T062, T063, T070, T071, T074 can run in parallel

**Phase 7 - User Story 5**: Tasks T082, T083 can run in parallel after implementation

**Phase 8 - User Story 6**: Tasks T086, T091, T092, T095 can run in parallel

**Phase 9 - Additional CLI**: Tasks T097-T102, T109-T114 all marked [P] can run in parallel

**Phase 10 - Integration**: Tasks T115-T118 can run in parallel after fixtures (T119-T121)

**Phase 12 - Builds**: Tasks T133-T137 can run in parallel

**Phase 13 - Documentation**: Tasks T146-T149, T151, T152 can run in parallel

**Phase 14 - Polish**: Tasks T156, T157, T158, T165, T166 can run in parallel

---

## Parallel Example: User Story 1

```bash
# Launch all parallel tasks for User Story 1 together:
Task: "Implement internal/cli/version.go with version command"
Task: "Add version variables to internal/cli/version.go"
Task: "Create scripts/build-all.sh for cross-platform builds"
Task: "Add ldflags for version injection"
Task: "Write testscript test for version command"
Task: "Add benchmark for startup time"
```

---

## Implementation Strategy

### MVP First (User Stories 1 & 2 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (CRITICAL - blocks all stories)
3. Complete Phase 3: User Story 1 (Simple Installation)
4. Complete Phase 4: User Story 2 (Cross-Platform Compatibility)
5. **STOP and VALIDATE**: Build binaries, test installation on all platforms
6. MVP Ready: Users can install and run `autospec version` on any platform

### Incremental Delivery

1. Complete Setup + Foundational → Foundation ready
2. Add User Story 1 → Test independently → Users can install tool
3. Add User Story 2 → Test independently → Users can install on any platform
4. Add User Story 3 → Test independently → Better error messages
5. Add User Story 4 → Test independently → Custom configurations work
6. Add User Story 5 → Test independently → Automated workflows work
7. Add User Story 6 → Test independently → Performance targets met
8. Each story adds value without breaking previous stories

### Parallel Team Strategy

With multiple developers:

1. Team completes Setup + Foundational together
2. Once Foundational is done:
   - Developer A: User Story 1 (Simple Installation)
   - Developer B: User Story 2 (Cross-Platform Compatibility)
   - Developer C: User Story 4 (Custom Commands)
3. After US1, US2 complete:
   - Developer A: User Story 3 (Pre-Flight Validation)
   - Developer B: User Story 5 (Automated Validation)
   - Developer C: User Story 6 (Fast Performance)
4. Stories complete and integrate independently

---

## Test Count Summary

Target: 63-80 tests (exceeds 60+ baseline from constitution)

**Unit Tests (35-40 tests)**:
- Configuration: 5-8 tests (T011)
- Git operations: 3-5 tests (T013)
- Spec detection: 4-6 tests (T015)
- File validation: 6-8 tests (T019)
- Task parsing: 8-10 tests (T020)
- Retry state: 6-8 tests (T024)
- Workflow: 5-7 tests (T029)
- Claude execution: 5-7 tests (T070)
- Pre-flight: 6-8 tests (T058)

**CLI Tests with testscript (15-20 tests)**:
- Version: 1 test (T036)
- Pre-flight: 1 test (T060)
- Workflow: 1 test (T083)
- Status: 1 test (T092)
- Init: 1 test (T109)
- Specify: 1 test (T110)
- Plan: 1 test (T111)
- Tasks: 1 test (T112)
- Implement: 1 test (T113)
- Config: 1 test (T114)
- Additional edge cases: 5-10 tests

**Integration Tests (8-12 tests)**:
- Complete workflow: 1 test (T115)
- Implementation progress: 1 test (T116)
- Config hierarchy: 2 tests (T046, T117)
- Retry persistence: 2 tests (T047, T118)
- Custom commands: 1 test (T071)
- Error scenarios: 3 tests (T122-T124)

**Benchmarks (5-8 tests)**:
- Validation functions: 3-5 benchmarks (T021)
- Startup time: 1 benchmark (T037)
- Pre-flight: 1 benchmark (T059)
- Status command: 1 benchmark (T091)

**Total: 63-80 tests** ✅ Meets constitution requirement

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability (US1-US6)
- Each user story should be independently completable and testable
- Constitution requires 60+ tests - we have 63-80 tests planned
- Constitution requires test-first development - tests are written BEFORE implementation
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- Avoid: vague tasks, same file conflicts, cross-story dependencies that break independence
