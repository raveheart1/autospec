# Manual Testing Plan: E2E Mock Testing (102-e2e-mock-testing)

## Overview

This document outlines the manual testing plan for verifying the E2E mock testing infrastructure and command coverage audit functionality implemented in this feature.

## Prerequisites

- Go 1.25+ installed
- Git repository initialized
- `autospec` built locally (`make build`)

## Test Scenarios

### 1. E2E Test Framework

#### 1.1 Mock Binary Setup
**Objective**: Verify the mock OpenCode binary works correctly in test environments.

**Steps**:
1. Build autospec: `make build`
2. Run E2E tests: `make test-e2e`
3. Verify all tests pass without network calls

**Expected Result**: All E2E tests pass, mock binary intercepts agent calls.

---

#### 1.2 E2EEnv Test Helpers
**Objective**: Verify E2EEnv helper methods work correctly.

**Steps**:
1. Review `internal/testutil/e2e.go` for helper methods
2. Verify these methods are used in test files:
   - `SetupAutospecInit()` - Creates .autospec directory structure
   - `SetupConstitution()` - Creates constitution.yaml
   - `InitGitRepo()` - Initializes git repository
   - `CreateBranch()` - Creates and switches to branch
   - `SetupTasks()` - Creates tasks.yaml
   - `SetupSpec()` - Creates spec.yaml
   - `SetupPlan()` - Creates plan.yaml

**Expected Result**: All helper methods function correctly in tests.

---

### 2. Command Coverage Audit

#### 2.1 Coverage Audit Test
**Objective**: Verify the command coverage audit test correctly identifies coverage.

**Steps**:
1. Run: `go test -tags=e2e -v -run TestE2E_CommandCoverageAudit ./tests/e2e/...`
2. Verify output shows 100% coverage
3. Verify all 68 commands are listed

**Expected Result**: Test passes with 100% command coverage.

---

#### 2.2 Missing Command Detection
**Objective**: Verify the audit detects missing test coverage.

**Steps**:
1. Temporarily remove a test for a command (e.g., comment out `TestE2E_VersionCommand`)
2. Run the coverage audit test
3. Verify it reports the missing command
4. Restore the test

**Expected Result**: Audit correctly identifies commands without test coverage.

---

### 3. Internal Commands E2E Tests

#### 3.1 `all` Command
**Objective**: Verify the `all` command E2E test works.

**Steps**:
1. Run: `go test -tags=e2e -v -run TestE2E_AllCommand ./tests/e2e/...`
2. Verify help output includes workflow stages

**Expected Result**: Test passes, help shows specify/plan/tasks/implement workflow.

---

#### 3.2 Task Management Commands
**Objective**: Verify task block/unblock/list E2E tests work.

**Steps**:
1. Run: `go test -tags=e2e -v -run "TestE2E_Task" ./tests/e2e/...`
2. Verify tests for:
   - `task block`
   - `task unblock`
   - `task list`

**Expected Result**: All task management tests pass.

---

#### 3.3 `update-task` Command
**Objective**: Verify the `update-task` command E2E test works.

**Steps**:
1. Run: `go test -tags=e2e -v -run TestE2E_UpdateTaskCommand ./tests/e2e/...`
2. Verify task status can be updated in tasks.yaml

**Expected Result**: Test passes, task status updates correctly.

---

#### 3.4 `yaml check` Command
**Objective**: Verify the `yaml check` subcommand E2E test works.

**Steps**:
1. Run: `go test -tags=e2e -v -run TestE2E_YamlCheck ./tests/e2e/...`
2. Verify both valid and invalid YAML detection

**Expected Result**: Test passes, valid YAML passes, invalid YAML fails.

---

### 4. CI Integration

#### 4.1 CI Workflow Verification
**Objective**: Verify E2E tests are included in CI workflow.

**Steps**:
1. Review `.github/workflows/ci.yml`
2. Verify E2E test step exists (line 67-68)
3. Push a branch and verify CI runs E2E tests

**Expected Result**: CI workflow includes E2E tests, tests run on push.

---

### 5. Test File Organization

#### 5.1 Test File Structure
**Objective**: Verify E2E test files are well-organized.

**Steps**:
1. List test files: `ls tests/e2e/*_test.go`
2. Verify each file tests related commands:
   - `admin_test.go` - commands, completion, uninstall
   - `config_test.go` - config, init, migrate, doctor
   - `dag_test.go` - dag subcommands
   - `worktree_test.go` - worktree subcommands
   - `util_test.go` - status, history, view, clean, artifact, version, help
   - `internal_commands_test.go` - all, new-feature, prereqs, task, update-task, yaml check, sauce
   - `coverage_audit_test.go` - command coverage verification

**Expected Result**: Test files are organized by command groups.

---

## Report Summaries

*(To be filled in after manual testing is complete)*

### Test Results

| Test Scenario | Status | Notes |
|---------------|--------|-------|
| 1.1 Mock Binary Setup | ☐ | |
| 1.2 E2EEnv Test Helpers | ☐ | |
| 2.1 Coverage Audit Test | ☐ | |
| 2.2 Missing Command Detection | ☐ | |
| 3.1 `all` Command | ☐ | |
| 3.2 Task Management Commands | ☐ | |
| 3.3 `update-task` Command | ☐ | |
| 3.4 `yaml check` Command | ☐ | |
| 4.1 CI Workflow Verification | ☐ | |
| 5.1 Test File Structure | ☐ | |

### Issues Found

*(Document any issues discovered during manual testing)*

### Overall Assessment

*(Summary of manual testing results and any recommendations)*
