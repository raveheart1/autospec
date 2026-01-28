# Manual Testing Plan: Wire Template Rendering (106)

Feature branch: `106-wire-template-rendering`

## Overview

This feature wires the existing template rendering infrastructure into workflow execution. The core issue was that StageExecutor built slash commands (e.g., "/autospec.plan") and passed them directly to Claude without rendering template variables like `{{.FeatureDir}}`.

## Affected Commands

| Command | Template Variables | Expected Behavior |
|---------|-------------------|-------------------|
| `autospec plan` | `{{.FeatureDir}}`, `{{.FeatureSpec}}` | Variables rendered to actual spec paths |
| `autospec tasks` | `{{.FeatureDir}}`, `{{.FeatureSpec}}`, `{{.ImplPlan}}` | Variables rendered to actual spec paths |
| `autospec implement` | `{{.FeatureDir}}`, `{{.TasksFile}}`, `{{.IsGitRepo}}` | Variables rendered to actual paths |
| `autospec clarify` | `{{.FeatureDir}}`, `{{.FeatureSpec}}` | Variables rendered to actual spec paths |
| `autospec analyze` | `{{.FeatureDir}}`, `{{.FeatureSpec}}` | Variables rendered to actual spec paths |
| `autospec checklist` | `{{.FeatureDir}}`, `{{.FeatureSpec}}` | Variables rendered to actual spec paths |

## Test Cases

### 1. render-command CLI Validation

**Purpose**: Verify the render-command CLI correctly renders all template variables.

**Steps**:
1. Checkout a feature branch (e.g., `106-wire-template-rendering`)
2. Run `autospec render-command autospec.plan`
3. Run `autospec render-command autospec.tasks`
4. Run `autospec render-command autospec.implement`
5. Run `autospec render-command autospec.clarify`
6. Run `autospec render-command autospec.analyze`
7. Run `autospec render-command autospec.checklist`

**Expected**:
- No literal `{{.FeatureDir}}` or `{{.Variable}}` patterns in output
- Paths should resolve to `specs/<branch-name>/` format
- `{{.IsGitRepo}}` should be `true` in a git repository

**Status**: [ ] Pass / [ ] Fail

### 2. ExecutePlan Template Rendering

**Purpose**: Verify ExecutePlan renders templates before Claude execution.

**Steps**:
1. Create a new feature branch with spec.yaml
2. Run `autospec plan --debug` (if available) or inspect execution
3. Verify Claude receives rendered paths, not template variables

**Expected**:
- Claude sees `specs/XXX-feature/spec.yaml`, not `{{.FeatureSpec}}`
- Claude finds spec file on first attempt without searching

**Status**: [ ] Pass / [ ] Fail

### 3. ExecuteTasks Template Rendering

**Purpose**: Verify ExecuteTasks renders templates before Claude execution.

**Steps**:
1. Have a feature with spec.yaml and plan.yaml
2. Run `autospec tasks`
3. Verify Claude receives rendered paths for spec, plan, and feature directory

**Expected**:
- Claude sees actual paths for `{{.FeatureSpec}}` and `{{.ImplPlan}}`
- No "searching for spec" behavior from Claude

**Status**: [ ] Pass / [ ] Fail

### 4. ExecuteImplement Template Rendering (Phase Mode)

**Purpose**: Verify phase execution renders implement templates.

**Steps**:
1. Have a feature with complete tasks.yaml
2. Run `autospec implement --phases --from-phase 1`
3. Verify Claude receives rendered `{{.FeatureDir}}` and `{{.TasksFile}}`

**Expected**:
- Claude sees `specs/XXX-feature/` and `specs/XXX-feature/tasks.yaml`
- Implementation proceeds without path-related errors

**Status**: [ ] Pass / [ ] Fail

### 5. ExecuteImplement Template Rendering (Task Mode)

**Purpose**: Verify task execution renders implement templates.

**Steps**:
1. Have a feature with complete tasks.yaml
2. Run `autospec implement --tasks --from-task T001`
3. Verify Claude receives rendered paths

**Expected**:
- Claude sees actual paths, not template variables
- Task execution proceeds without path-related errors

**Status**: [ ] Pass / [ ] Fail

### 6. Auxiliary Commands (clarify, analyze, checklist)

**Purpose**: Verify auxiliary commands render templates.

**Steps**:
1. Have a feature with spec.yaml
2. Run `autospec clarify`
3. Run `autospec analyze` (requires tasks.yaml)
4. Run `autospec checklist`
5. Verify each command receives rendered paths

**Expected**:
- All three commands receive actual paths
- No template variable syntax visible to Claude

**Status**: [ ] Pass / [ ] Fail

### 7. Error Handling - No Spec Found

**Purpose**: Verify clear error when spec doesn't exist.

**Steps**:
1. Checkout a branch that doesn't have a spec (e.g., `main`)
2. Run `autospec plan`

**Expected**:
- Clear error message indicating no spec found
- Suggestion to run `autospec specify` first

**Status**: [ ] Pass / [ ] Fail

### 8. Error Handling - Detached HEAD

**Purpose**: Verify clear error when in detached HEAD state.

**Steps**:
1. `git checkout --detach HEAD`
2. Run `autospec plan`

**Expected**:
- Clear error about branch detection failure
- Suggestion to checkout a feature branch

**Status**: [ ] Pass / [ ] Fail

### 9. Full Pipeline Test

**Purpose**: Verify complete `autospec run` works without manual paths.

**Steps**:
1. Create a new feature branch
2. Run `autospec run -a "test feature description"`
3. Monitor execution through specify → plan → tasks → implement

**Expected**:
- All stages complete without Claude searching wrong directories
- No `{{.Variable}}` patterns in any Claude command

**Status**: [ ] Pass / [ ] Fail

### 10. autospec prep Test

**Purpose**: Verify prep command (specify → plan → tasks) works.

**Steps**:
1. Create a new feature branch
2. Run `autospec prep "test feature"`
3. Verify spec.yaml, plan.yaml, and tasks.yaml are created

**Expected**:
- All three artifacts created in correct directory
- No path-related errors during execution

**Status**: [ ] Pass / [ ] Fail

## Report Summaries

### Execution Date: _____________

### Overall Status: [ ] PASS / [ ] FAIL

### Test Results Summary

| Test Case | Status | Notes |
|-----------|--------|-------|
| 1. render-command CLI | | |
| 2. ExecutePlan | | |
| 3. ExecuteTasks | | |
| 4. ExecuteImplement (Phase) | | |
| 5. ExecuteImplement (Task) | | |
| 6. Auxiliary Commands | | |
| 7. Error - No Spec | | |
| 8. Error - Detached HEAD | | |
| 9. Full Pipeline | | |
| 10. autospec prep | | |

### Issues Found

_List any issues discovered during manual testing:_

1.

### Notes

_Additional observations or recommendations:_

