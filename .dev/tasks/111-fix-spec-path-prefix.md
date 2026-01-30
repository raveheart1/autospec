# Manual Testing Plan: Fix Spec Path Prefix Bug (#111)

## Overview

This document contains manual test scenarios for verifying the fix for the spec path prefix bug where paths were incorrectly constructed with a leading dash (e.g., `specs/-110-dag-spec-validation/` instead of `specs/110-dag-spec-validation/`).

## Pre-requisites

- Clean build: `make build`
- Ensure you're on the `111-fix-spec-path-prefix` branch
- Have an existing spec directory for testing (e.g., `specs/110-dag-spec-validation/`)

---

## Test Scenarios

### Scenario 1: --spec flag with full spec name

**Purpose**: Verify --spec flag correctly parses spec name into Number and Name components.

**Steps**:
1. Navigate to a project with autospec configured
2. Ensure a spec directory exists (e.g., `specs/110-dag-spec-validation/` with at least `spec.yaml`)
3. Run: `autospec run --spec 110-dag-spec-validation -pti --dry-run` (or similar)
4. Observe debug output or validation paths

**Expected Result**:
- Path should be `specs/110-dag-spec-validation/`, NOT `specs/-110-dag-spec-validation/`
- Command should find the spec correctly
- No "spec not found" errors due to incorrect path

**Actual Result**: _[To be filled during testing]_

**Status**: [ ] PASS / [ ] FAIL

---

### Scenario 2: SPECIFY_FEATURE environment variable

**Purpose**: Verify SPECIFY_FEATURE env var correctly parses spec name.

**Steps**:
1. Set environment variable: `export SPECIFY_FEATURE=110-dag-spec-validation`
2. Run: `autospec run -ti` (without --spec flag)
3. Observe which spec directory is used

**Expected Result**:
- Should correctly detect `specs/110-dag-spec-validation/`
- Should NOT look for `specs/-110-dag-spec-validation/`

**Actual Result**: _[To be filled during testing]_

**Status**: [ ] PASS / [ ] FAIL

---

### Scenario 3: Edge case - Spec name starting with number

**Purpose**: Verify spec names like `110-2fa-implementation` are handled correctly.

**Steps**:
1. Create a spec directory: `mkdir -p specs/110-2fa-implementation`
2. Create minimal spec.yaml in that directory
3. Run: `autospec run --spec 110-2fa-implementation --dry-run`

**Expected Result**:
- Number: `110`
- Name: `2fa-implementation`
- Path: `specs/110-2fa-implementation/` (no leading dash)

**Actual Result**: _[To be filled during testing]_

**Status**: [ ] PASS / [ ] FAIL

---

### Scenario 4: Multi-stage workflow consistency

**Purpose**: Verify spec name is preserved correctly through all workflow stages.

**Steps**:
1. Run a full workflow: `autospec run -spti "test feature"` (this will create a new spec)
2. Observe the spec directory name created
3. Verify each stage (specify, plan, tasks, implement) uses the same path

**Expected Result**:
- All stages should reference the same `specs/NNN-test-feature/` directory
- No stage should have a leading dash in the path

**Actual Result**: _[To be filled during testing]_

**Status**: [ ] PASS / [ ] FAIL

---

### Scenario 5: Boundary spec numbers

**Purpose**: Test spec numbers at boundaries (001 and 999).

**Steps**:
1. Create test directories:
   - `mkdir -p specs/001-first-feature`
   - `mkdir -p specs/999-last-feature`
2. Create minimal spec.yaml in each
3. Run: `autospec status --spec 001-first-feature`
4. Run: `autospec status --spec 999-last-feature`

**Expected Result**:
- Both should resolve to correct paths without leading dash
- `001-first-feature` -> Number: `001`, Name: `first-feature`
- `999-last-feature` -> Number: `999`, Name: `last-feature`

**Actual Result**: _[To be filled during testing]_

**Status**: [ ] PASS / [ ] FAIL

---

### Scenario 6: Git branch detection

**Purpose**: Verify git branch-based spec detection still works.

**Steps**:
1. Checkout a branch matching spec pattern: `git checkout -b 112-test-branch`
2. Create spec directory: `mkdir -p specs/112-test-branch`
3. Create minimal spec.yaml
4. Run: `autospec status` (without --spec flag)

**Expected Result**:
- Should detect spec from git branch name
- Should use path `specs/112-test-branch/` (no leading dash)

**Actual Result**: _[To be filled during testing]_

**Status**: [ ] PASS / [ ] FAIL

---

## Report Summaries

### Test Date: _[YYYY-MM-DD]_
### Tester: _[Name]_

| Scenario | Status | Notes |
|----------|--------|-------|
| 1. --spec flag | | |
| 2. SPECIFY_FEATURE env | | |
| 3. Numeric feature name | | |
| 4. Multi-stage workflow | | |
| 5. Boundary numbers | | |
| 6. Git branch detection | | |

### Overall Status: [ ] ALL PASS / [ ] SOME FAILURES

### Failure Details (if any):
_[Document any failures with reproduction steps and error messages]_

### Additional Notes:
_[Any observations or recommendations from testing]_
