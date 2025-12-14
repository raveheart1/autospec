# Tasks: YAML Structured Output

**Input**: Design documents from `/specs/007-yaml-structured-output/`
**Prerequisites**: plan.md (required), spec.md (required), research.md, data-model.md, contracts/yaml-schemas.yaml

**Tests**: Tests are included per constitution requirement (Test-First Development - NON-NEGOTIABLE from CLAUDE.md).

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- **Single project** (Go binary): `internal/`, `cmd/`, `commands/` at repository root
- Test files: `*_test.go` alongside implementation files
- Embedded templates: Top-level `commands/` directory

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and new package structure

- [X] T001 Create `commands/` directory at repository root for embedded templates
- [X] T002 [P] Create `internal/commands/` package directory for template management logic
- [X] T003 [P] Create `internal/yaml/` package directory for YAML validation logic
- [X] T004 Promote gopkg.in/yaml.v3 from indirect to direct dependency in go.mod

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

**CRITICAL**: No user story work can begin until this phase is complete

### Core Types (from data-model.md)

- [X] T005 [P] Create Meta struct and artifact types in internal/yaml/types.go
- [X] T006 [P] Create CommandTemplate struct in internal/commands/types.go

### Embed Infrastructure (from research.md)

- [X] T007 Write unit tests for template embedding in internal/commands/embed_test.go
- [X] T008 Implement go:embed directive and TemplateFS in internal/commands/embed.go

### YAML Validation Core (from research.md)

- [X] T009 Write unit tests for YAML syntax validation in internal/yaml/validator_test.go
- [X] T010 Implement ValidateSyntax function using yaml.Decoder streaming in internal/yaml/validator.go

### Meta Section Handling

- [X] T011 Write unit tests for _meta section parsing in internal/yaml/meta_test.go
- [X] T012 Implement _meta extraction and version parsing in internal/yaml/meta.go

**Checkpoint**: Foundation ready - YAML validation and embed infrastructure operational

---

## Phase 3: User Story 4 - Validate YAML Syntax (Priority: P3)

**Goal**: Enable syntax validation of YAML artifacts via CLI command

**Independent Test**: Run `autospec yaml check specs/007-yaml-structured-output/spec.yaml` on valid and invalid files

**Why Phase 3**: Although P3 priority, YAML validation is a prerequisite for all other user stories (FR-013 requires validation in all command templates)

### Tests for User Story 4

- [X] T013 [P] [US4] Write integration tests for yaml check command in internal/cli/yaml_check_test.go
- [X] T014 [P] [US4] Create test fixtures in tests/fixtures/valid.yaml and tests/fixtures/invalid.yaml

### Implementation for User Story 4

- [X] T015 [US4] Create yaml subcommand group in internal/cli/yaml.go
- [X] T016 [US4] Implement `autospec yaml check` command in internal/cli/yaml_check.go
- [X] T017 [US4] Add exit code handling (0=valid, non-zero=error with line number)
- [X] T018 [US4] Register yaml commands with root command in internal/cli/root.go

**Checkpoint**: `autospec yaml check <file>` operational with correct exit codes

---

## Phase 4: User Story 3 - Install and Manage AutoSpec Commands (Priority: P2)

**Goal**: Enable installation and management of autospec command templates to .claude/commands/

**Independent Test**: Run `autospec commands install`, verify files created, run `autospec commands check` and `autospec commands info`

### Tests for User Story 3

- [X] T019 [P] [US3] Write unit tests for template listing in internal/commands/templates_test.go
- [X] T020 [P] [US3] Write integration tests for commands install in internal/cli/commands_install_test.go
- [X] T021 [P] [US3] Write integration tests for commands check in internal/cli/commands_check_test.go
- [X] T022 [P] [US3] Write integration tests for commands info in internal/cli/commands_info_test.go

### Implementation for User Story 3

- [X] T023 [US3] Implement template listing and retrieval in internal/commands/templates.go
- [X] T024 [US3] Implement version comparison logic (VersionMismatch) in internal/commands/templates.go
- [X] T025 [US3] Create commands subcommand group in internal/cli/commands.go
- [X] T026 [US3] Implement `autospec commands install` in internal/cli/commands_install.go
- [X] T027 [US3] Implement `autospec commands check` in internal/cli/commands_check.go
- [X] T028 [US3] Implement `autospec commands info` in internal/cli/commands_info.go
- [X] T029 [US3] Register commands subgroup with root command in internal/cli/root.go

**Checkpoint**: Command installation, checking, and info display operational

---

## Phase 5: User Story 1 - Create YAML-Based Feature Specifications (Priority: P1) MVP

**Goal**: Generate spec.yaml files using /autospec.specify command template

**Independent Test**: Install commands, run `/autospec.specify "test feature"`, verify spec.yaml created and passes syntax validation

### Implementation for User Story 1

- [X] T030 [US1] Create autospec.specify.md command template in commands/autospec.specify.md
- [X] T031 [US1] Include _meta section generation instructions in template
- [X] T032 [US1] Include schema reference from contracts/yaml-schemas.yaml in template
- [X] T033 [US1] Add `autospec yaml check` validation step to template (FR-013)
- [X] T034 [US1] Add version field to template frontmatter for version tracking

**Checkpoint**: `/autospec.specify` generates valid spec.yaml files

---

## Phase 6: User Story 5 - Generate Implementation Plans in YAML (Priority: P3)

**Goal**: Generate plan.yaml files using /autospec.plan command template

**Independent Test**: Run `/autospec.plan` after spec.yaml exists, verify plan.yaml created with technical context

### Implementation for User Story 5

- [X] T035 [P] [US5] Create autospec.plan.md command template in commands/autospec.plan.md
- [X] T036 [US5] Include technical_context extraction instructions in template
- [X] T037 [US5] Include constitution_check section generation in template
- [X] T038 [US5] Add `autospec yaml check` validation step to template (FR-013)
- [X] T039 [US5] Add handoff to autospec.tasks in template frontmatter

**Checkpoint**: `/autospec.plan` generates valid plan.yaml files

---

## Phase 7: User Story 2 - Generate Structured Task Breakdowns (Priority: P2)

**Goal**: Generate tasks.yaml files using /autospec.tasks command template

**Independent Test**: Run `/autospec.tasks` after plan.yaml exists, verify tasks.yaml created with phases and task structure

### Implementation for User Story 2

- [X] T040 [P] [US2] Create autospec.tasks.md command template in commands/autospec.tasks.md
- [X] T041 [US2] Include phase structure generation instructions in template
- [X] T042 [US2] Include task status tracking format in template
- [X] T043 [US2] Add `autospec yaml check` validation step to template (FR-013)

**Checkpoint**: `/autospec.tasks` generates valid tasks.yaml files

---

## Phase 8: Additional Command Templates (P3+ Stories)

**Goal**: Complete remaining command templates for checklist, analysis, and constitution

### Checklist Template (FR-004)

- [X] T044 [P] Create autospec.checklist.md command template in commands/autospec.checklist.md
- [X] T045 Add category and item structure to checklist template

### Analysis Template (FR-005)

- [X] T046 [P] Create autospec.analyze.md command template in commands/autospec.analyze.md
- [X] T047 Add findings and summary structure to analysis template

### Constitution Template (FR-006)

- [X] T048 [P] Create autospec.constitution.md command template in commands/autospec.constitution.md
- [X] T049 Add principles and governance structure to constitution template

**Checkpoint**: All 6 command templates created and embedded

---

## Phase 9: User Story 6 - Migrate Existing Markdown to YAML (Priority: P5)

**Goal**: Convert existing markdown artifacts to YAML format

**Independent Test**: Run `autospec migrate md-to-yaml specs/existing-feature/`, verify YAML files created

### Tests for User Story 6

- [X] T050 [P] [US6] Write unit tests for markdown parsing in internal/yaml/migrate_test.go
- [X] T051 [P] [US6] Write integration tests for migrate command in internal/cli/migrate_test.go

### Implementation for User Story 6

- [X] T052 [US6] Implement markdown-to-YAML conversion logic in internal/yaml/migrate.go
- [X] T053 [US6] Create migrate subcommand group in internal/cli/migrate.go
- [X] T054 [US6] Implement `autospec migrate md-to-yaml` command in internal/cli/migrate_mdtoyaml.go
- [X] T055 [US6] Handle mixed format directories (preserve existing YAML)
- [X] T056 [US6] Register migrate commands with root command in internal/cli/root.go

**Checkpoint**: Migration from markdown to YAML operational

---

## Phase 10: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple user stories

- [X] T057 [P] Add benchmark tests for YAML validation (<100ms for 10MB) in internal/yaml/validator_bench_test.go
- [X] T058 [P] Update existing validation.go to call new YAML validation for .yaml files in internal/validation/validation.go
- [X] T059 Write end-to-end workflow test in tests/integration/yaml_workflow_test.go
- [X] T060 Verify all command templates pass `autospec yaml check` on generated output
- [X] T061 Run quickstart.md validation scenarios

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **US4 - YAML Validation (Phase 3)**: Depends on Foundational - Required for FR-013 (validation in templates)
- **US3 - Command Management (Phase 4)**: Depends on Foundational + US4 (needs yaml check)
- **US1 - Specify (Phase 5)**: Depends on US3 (needs commands install) + US4 (needs yaml check)
- **US5 - Plan (Phase 6)**: Depends on US1 (needs spec.yaml input)
- **US2 - Tasks (Phase 7)**: Depends on US5 (needs plan.yaml input)
- **Additional Templates (Phase 8)**: Can proceed in parallel with US1/US5/US2
- **US6 - Migration (Phase 9)**: Independent after Foundational - optional feature
- **Polish (Phase 10)**: Depends on all prior phases

### User Story Dependencies

```
Foundational
    ├── US4 (yaml check) ─────────┬── US3 (commands) ── US1 (specify) ── US5 (plan) ── US2 (tasks)
    │                             │
    │                             └── Additional Templates (Phase 8)
    │
    └── US6 (migration) - Independent, can run in parallel after Foundational
```

### Within Each User Story

- Tests MUST be written and FAIL before implementation
- Types/structs before business logic
- Core implementation before CLI integration
- Story complete before moving to dependent stories

### Parallel Opportunities

**Phase 2 (Foundational)**:
- T005, T006 can run in parallel (different packages)
- T007-T008, T009-T010, T011-T012 are test-first pairs (sequential within pair, parallel between pairs)

**Phase 3 (US4)**:
- T013, T014 can run in parallel (tests + fixtures)

**Phase 4 (US3)**:
- T019, T020, T021, T022 can run in parallel (all tests)

**Phase 8 (Additional Templates)**:
- T044, T046, T048 can run in parallel (different template files)

---

## Parallel Example: Phase 2 (Foundational)

```bash
# Launch type definitions together:
Task: "Create Meta struct and artifact types in internal/yaml/types.go"
Task: "Create CommandTemplate struct in internal/commands/types.go"

# After types, launch test-first pairs in parallel:
# Pair 1: Embed
Task: "Write unit tests for template embedding in internal/commands/embed_test.go"
Task: "Implement go:embed directive and TemplateFS in internal/commands/embed.go"

# Pair 2: Validator (can run in parallel with Pair 1)
Task: "Write unit tests for YAML syntax validation in internal/yaml/validator_test.go"
Task: "Implement ValidateSyntax function in internal/yaml/validator.go"
```

---

## Implementation Strategy

### MVP First (User Stories 4, 3, 1)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (CRITICAL)
3. Complete Phase 3: US4 (yaml check) - Required for all templates
4. Complete Phase 4: US3 (commands install/check/info)
5. Complete Phase 5: US1 (specify template)
6. **STOP and VALIDATE**: Test `/autospec.specify` end-to-end

### Incremental Delivery

1. Setup + Foundational + US4 → `autospec yaml check` operational
2. Add US3 → Command management operational
3. Add US1 → `/autospec.specify` generates spec.yaml (MVP!)
4. Add US5 → `/autospec.plan` generates plan.yaml
5. Add US2 → `/autospec.tasks` generates tasks.yaml
6. Add Phase 8 → All 6 command templates available
7. Add US6 → Migration available (optional)

### Test-First Enforcement

Per CLAUDE.md constitution (NON-NEGOTIABLE):
- Every implementation file has corresponding `*_test.go`
- Tests written BEFORE implementation
- Tests MUST fail before implementation proceeds

---

## Summary

| Metric | Count |
|--------|-------|
| Total Tasks | 61 |
| Phase 1 (Setup) | 4 |
| Phase 2 (Foundational) | 8 |
| Phase 3 (US4 - yaml check) | 6 |
| Phase 4 (US3 - commands) | 11 |
| Phase 5 (US1 - specify) | 5 |
| Phase 6 (US5 - plan) | 5 |
| Phase 7 (US2 - tasks) | 4 |
| Phase 8 (Additional) | 6 |
| Phase 9 (US6 - migrate) | 7 |
| Phase 10 (Polish) | 5 |

| User Story | Task Count | Priority |
|------------|------------|----------|
| US1 (specify) | 5 | P1 |
| US2 (tasks) | 4 | P2 |
| US3 (commands) | 11 | P2 |
| US4 (yaml check) | 6 | P3 |
| US5 (plan) | 5 | P3 |
| US6 (migrate) | 7 | P5 |

**Parallel Opportunities**: 18 tasks marked [P]

**Suggested MVP Scope**: Phases 1-5 (Setup → Foundational → US4 → US3 → US1) = 34 tasks

**Format Validation**: All 61 tasks follow checklist format (checkbox, ID, labels, file paths)

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Test-first development is NON-NEGOTIABLE per project constitution
- Performance target: YAML validation <100ms for 10MB files
- All command templates must include `autospec yaml check` step per FR-013
