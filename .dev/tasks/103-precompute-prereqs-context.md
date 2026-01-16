# Manual Testing Plan: Pre-compute Prereqs Context (103)

This document outlines the manual testing steps for verifying the pre-computed prereqs context feature.

## Prerequisites

- autospec installed and built (`make build`)
- A feature branch with spec.yaml, plan.yaml, and tasks.yaml files
- Access to Claude Code or another agent that uses slash commands

## Test Scenarios

### 1. Basic Template Rendering

**Objective**: Verify that `render-command` outputs templates with pre-computed context.

**Steps**:
1. Navigate to a feature branch (e.g., `git checkout 103-precompute-prereqs-context`)
2. Run: `autospec render-command autospec.plan | head -30`
3. Verify:
   - `{{.FeatureDir}}` is replaced with actual path (e.g., `specs/103-precompute-prereqs-context`)
   - `{{.FeatureSpec}}` is replaced with spec path
   - `{{.AutospecVersion}}` shows the version string
   - `{{.CreatedDate}}` shows an ISO 8601 timestamp

**Expected Result**: Template variables are replaced with actual values.

### 2. All Commands Render Successfully

**Objective**: Verify all autospec commands can be rendered without error.

**Steps**:
1. Run each command and verify it renders:
   ```bash
   autospec render-command autospec.specify
   autospec render-command autospec.plan
   autospec render-command autospec.tasks
   autospec render-command autospec.implement
   autospec render-command autospec.checklist
   autospec render-command autospec.clarify
   autospec render-command autospec.analyze
   autospec render-command autospec.constitution
   ```
2. Verify no `{{.` template markers remain in output

**Expected Result**: All commands render without error, no unrendered placeholders.

### 3. Error Handling for Missing Prerequisites

**Objective**: Verify appropriate errors when prerequisites are missing.

**Steps**:
1. Create a new branch without specs: `git checkout -b 999-test-missing`
2. Run: `autospec render-command autospec.plan`
3. Verify error message mentions missing spec or feature detection failure

**Expected Result**: Clear error message guiding user to run prerequisite commands.

### 4. Output File Option

**Objective**: Verify the `--output` flag works correctly.

**Steps**:
1. Run: `autospec render-command autospec.plan --output /tmp/rendered-plan.md`
2. Verify file exists: `cat /tmp/rendered-plan.md | head -20`
3. Verify content has rendered template variables

**Expected Result**: File created with rendered content.

### 5. Prereqs Command Still Works

**Objective**: Verify the existing `prereqs` command still functions.

**Steps**:
1. Run: `autospec prereqs --json --require-spec`
2. Verify JSON output contains all expected fields:
   - FEATURE_DIR
   - FEATURE_SPEC
   - AUTOSPEC_VERSION
   - CREATED_DATE
   - IS_GIT_REPO

**Expected Result**: JSON output unchanged from previous behavior.

### 6. Installed Commands Work

**Objective**: Verify installed command templates still work correctly.

**Steps**:
1. Run: `autospec init` (if needed)
2. Check installed commands have template placeholders:
   ```bash
   head -30 .claude/commands/autospec.plan.md
   ```
3. Verify `{{.FeatureDir}}` and other placeholders are present
4. Test with Claude Code: invoke `/autospec.plan` and verify it works

**Expected Result**: Installed commands contain template variables that can be rendered.

### 7. Help Text Verification

**Objective**: Verify help text for new render-command.

**Steps**:
1. Run: `autospec render-command --help`
2. Verify:
   - Usage description is clear
   - Example commands are present
   - Flag descriptions are accurate

**Expected Result**: Clear and helpful documentation.

## Report Summaries

_Completed: 2026-01-16_

| Test | Status | Notes |
|------|--------|-------|
| 1. Basic Template Rendering | PASS | All template variables rendered correctly (FEATURE_DIR, FEATURE_SPEC, AUTOSPEC_VERSION, CREATED_DATE) |
| 2. All Commands Render | PASS | All 8 commands render without errors or unrendered placeholders |
| 3. Error Handling | PASS | Clear error message when spec not found: "could not detect current feature: no spec directories found" |
| 4. Output File Option | PASS | File created at specified path with rendered content |
| 5. Prereqs Command | PASS | JSON output contains all expected fields (FEATURE_DIR, FEATURE_SPEC, IMPL_PLAN, TASKS, AVAILABLE_DOCS, AUTOSPEC_VERSION, CREATED_DATE, IS_GIT_REPO) |
| 6. Installed Commands | PASS | After `commands install`, templates contain `{{.FeatureDir}}` placeholders (4 occurrences in autospec.plan.md) |
| 7. Help Text | PASS | Clear usage description, examples, and flag documentation |

**Overall Assessment**: All 7 tests passed. The pre-compute prereqs context feature is working correctly.

## Additional Notes

- The feature adds a new `internal/prereqs` package for computing context
- The feature adds a new `internal/version` package to avoid import cycles
- Command templates now use Go text/template syntax with `{{.FieldName}}` placeholders
- The `render-command` CLI command is for debugging/testing purposes
