# Implementation Plan: YAML Structured Output

**Branch**: `007-yaml-structured-output` | **Date**: 2025-12-13 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/007-yaml-structured-output/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

Add YAML structured output format for SpecKit workflow artifacts (spec.yaml, plan.yaml, tasks.yaml, checklist.yaml, analysis.yaml, constitution.yaml) with embedded command templates in the Go binary and CLI commands for installation, validation, and version management. The implementation uses Go's `go:embed` directive to bundle command templates, `gopkg.in/yaml.v3` for YAML parsing and validation, and extends the existing Cobra CLI with new subcommands.

## Technical Context

**Language/Version**: Go 1.25.1
**Primary Dependencies**: Cobra CLI (v1.10.1), gopkg.in/yaml.v3 (v3.0.1 - already indirect dep), koanf (v2.3.0)
**Storage**: File system (YAML artifacts in `specs/*/`, command templates embedded in binary)
**Testing**: go test with testify (v1.11.1)
**Target Platform**: Linux, macOS, Windows (cross-platform binary)
**Project Type**: Single project (existing Go binary)
**Performance Goals**: YAML syntax check <100ms for 10MB files, command installation <5s (SC-003)
**Constraints**: Max YAML file size 10MB for syntax check (per spec), human-readable YAML output
**Scale/Scope**: Supports existing autospec workflow with 6 artifact types

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

The project constitution template uses placeholders, indicating the project-specific constitution has not been customized. The following principles are derived from CLAUDE.md and apply to this implementation:

| Gate | Status | Notes |
|------|--------|-------|
| Test-First Development (NON-NEGOTIABLE) | ✅ REQUIRED | All new code requires tests written before implementation |
| Performance Standards | ✅ REQUIRED | Sub-second validation; validation functions <10ms |
| Idempotency & Retry Logic | ✅ REQUIRED | All operations must be idempotent |
| Validation-First | ✅ REQUIRED | All workflow transitions validated before proceeding |

**Pre-Design Gate Status**: PASS - No violations. Implementation must follow test-first development.

## Project Structure

### Documentation (this feature)

```text
specs/007-yaml-structured-output/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md        # Phase 1 output (/speckit.plan command)
├── quickstart.md        # Phase 1 output (/speckit.plan command)
├── contracts/           # Phase 1 output (/speckit.plan command)
│   └── yaml-schemas.yaml  # YAML artifact schema definitions
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root)

```text
internal/
├── cli/
│   ├── commands.go         # NEW: `autospec commands` subcommand group
│   ├── commands_install.go # NEW: `autospec commands install` implementation
│   ├── commands_check.go   # NEW: `autospec commands check` implementation
│   ├── commands_info.go    # NEW: `autospec commands info` implementation
│   ├── yaml.go             # NEW: `autospec yaml` subcommand group
│   └── yaml_check.go       # NEW: `autospec yaml check` implementation
├── commands/
│   ├── embed.go            # NEW: go:embed directives for command templates
│   ├── templates.go        # NEW: template management (list, get, version)
│   └── templates_test.go   # NEW: unit tests
├── yaml/
│   ├── validator.go        # NEW: YAML syntax validation logic
│   ├── validator_test.go   # NEW: unit tests
│   ├── meta.go             # NEW: _meta section handling
│   └── meta_test.go        # NEW: unit tests
└── validation/
    └── validation.go       # EXTEND: Add ValidateYamlFile functions

commands/                    # NEW: Embedded command template directory
├── autospec.specify.md     # Template for YAML spec generation
├── autospec.plan.md        # Template for YAML plan generation
├── autospec.tasks.md       # Template for YAML tasks generation
├── autospec.checklist.md   # Template for YAML checklist generation
├── autospec.analyze.md     # Template for YAML analysis generation
└── autospec.constitution.md # Template for YAML constitution generation

tests/
├── integration/
│   └── yaml_workflow_test.go  # NEW: End-to-end YAML workflow tests
└── fixtures/
    ├── valid.yaml          # NEW: Test fixture for valid YAML
    └── invalid.yaml        # NEW: Test fixture for invalid YAML
```

**Structure Decision**: Extends existing single project structure. New packages (`internal/commands/`, `internal/yaml/`) follow existing patterns. Embedded templates stored in top-level `commands/` directory for clean separation.

## Command Template Structure

The `autospec.*.md` command templates follow the same format as existing `speckit.*.md` commands (in `.claude/commands/`), but are **rephrased** to output YAML files instead of Markdown and include validation.

### Template Format

Each template follows the Claude Code command format:

```markdown
---
description: <Short description of what this command does>
handoffs:
  - label: <Next step label>
    agent: <next.command>
    prompt: <Handoff prompt>
---

## User Input

```text
$ARGUMENTS
```

## Outline

1. **Setup**: <Initialize paths, detect feature>
2. **Load context**: <Read inputs AND yaml-schemas.yaml>
3. **Generate YAML**: <Output structured YAML per schema>
4. **Validate**: Run `autospec yaml check <file>`
5. **Report**: <Summary of what was created>
```

### Key Differences from `speckit.*.md`

| Aspect | `speckit.*.md` (existing) | `autospec.*.md` (new) |
|--------|---------------------------|------------------------|
| Output file | `spec.md`, `plan.md`, `tasks.md` | `spec.yaml`, `plan.yaml`, `tasks.yaml` |
| Output format | Markdown with headings/sections | YAML per `contracts/yaml-schemas.yaml` |
| Schema reference | Implicit (template structure) | Explicit (load schema, follow structure) |
| Validation step | None | **Required**: `autospec yaml check <file>` |
| `_meta` section | N/A | Required in all output files |

### Example: `autospec.plan.md` Structure

```markdown
---
description: Generate YAML implementation plan from feature specification.
handoffs:
  - label: Create Tasks
    agent: autospec.tasks
    prompt: Generate tasks from the plan
---

## User Input

```text
$ARGUMENTS
```

## Outline

1. **Setup**: Run setup script, parse JSON for FEATURE_SPEC, SPECS_DIR, BRANCH.

2. **Load context**:
   - Read FEATURE_SPEC (`spec.yaml` or `spec.md`)
   - Read `.specify/memory/constitution.md`
   - Read `contracts/yaml-schemas.yaml` for `plan_artifact` schema

3. **Generate plan.yaml**: Create YAML output following `plan_artifact` schema:

   ```yaml
   _meta:
     version: "1.0.0"
     generator: "autospec"
     generator_version: "<from autospec version>"
     created: "<ISO 8601 timestamp>"
     artifact_type: "plan"

   plan:
     branch: "<current branch>"
     date: "<today>"
     spec_path: "<path to spec file>"

   summary: "<1-2 paragraph summary>"

   technical_context:
     language: "<detected or specified>"
     primary_dependencies: [...]
     # ... per schema

   # ... remaining sections per plan_artifact schema
   ```

4. **Validate output**:
   ```bash
   autospec yaml check specs/<feature>/plan.yaml
   ```
   - If validation fails: fix YAML syntax errors and retry
   - If validation passes: proceed to report

5. **Report**: Output branch name, plan.yaml path, and readiness for `/autospec.tasks`.

## Key Rules

- Output MUST be valid YAML (use `autospec yaml check` to verify)
- All fields from schema marked `required` MUST be present
- Use `_meta.artifact_type: "plan"` exactly
- Timestamps in ISO 8601 format
- Arrays use YAML list syntax (not JSON inline)
```

### Validation Integration (FR-013)

Every `autospec.*.md` template MUST include this validation step before completion:

```markdown
## Validate Output

Run syntax validation on the generated YAML file:

```bash
autospec yaml check specs/<feature>/<artifact>.yaml
```

**On failure**:
- Parse error message for line number
- Fix the YAML syntax issue
- Re-run validation (max 3 retries)

**On success**: Proceed to report completion.
```

This ensures all generated YAML artifacts are syntactically valid before the workflow continues.

## Complexity Tracking

> No Constitution Check violations. No complexity justification required.

| Aspect | Decision | Rationale |
|--------|----------|-----------|
| Separate `internal/yaml/` package | New package | Clean separation of YAML validation from existing validation logic; allows independent testing |
| Separate `internal/commands/` package | New package | Isolates embed logic and template management from CLI layer |
| Top-level `commands/` directory | New directory | Required by go:embed - must be separate from `internal/` for embedding |

---

## Post-Design Constitution Check

*Re-evaluation after Phase 1 design completion.*

| Gate | Status | Verification |
|------|--------|--------------|
| Test-First Development (NON-NEGOTIABLE) | ✅ PASS | Design includes test files for each new package (`*_test.go`) |
| Performance Standards | ✅ PASS | YAML validation uses streaming Decoder (<100ms for 10MB); validation functions target <10ms |
| Idempotency & Retry Logic | ✅ PASS | `autospec commands install` is idempotent (overwrites existing); `autospec yaml check` is read-only |
| Validation-First | ✅ PASS | All YAML artifacts validated via `autospec yaml check` before workflow continues |

**Post-Design Gate Status**: PASS - All gates satisfied by design.

### Implementation Notes

1. **Test-First**: Each new file requires corresponding `*_test.go` written before implementation
2. **Performance**: Use `yaml.Decoder` streaming (not `yaml.Unmarshal` on full file) for large files
3. **Idempotency**: Command installation only overwrites `autospec.*` files, preserving user commands
4. **Validation**: Command templates instruct Claude to run `autospec yaml check` as final step

---

## Related Artifacts

| Artifact | Path | Status |
|----------|------|--------|
| Feature Spec | [spec.md](spec.md) | Complete |
| Research | [research.md](research.md) | Complete |
| Data Model | [data-model.md](data-model.md) | Complete |
| YAML Schemas | [contracts/yaml-schemas.yaml](contracts/yaml-schemas.yaml) | Complete |
| Quickstart | [quickstart.md](quickstart.md) | Complete |
| Tasks | tasks.md | Pending (Phase 2 - `/speckit.tasks`) |
