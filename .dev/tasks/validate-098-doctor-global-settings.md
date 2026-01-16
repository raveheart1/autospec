# Validation Plan: 098-doctor-global-settings

Validate the new `init.yml` tracking and doctor global settings features.

## Setup

```bash
# Build latest autospec
cd ~/repos/autospec
make build

# Create test repo
mkdir -p ~/repos/test-doctor-validation
cd ~/repos/test-doctor-validation
git init
```

## Test Cases

### 1. Validate `autospec init` creates `init.yml`

```bash
cd ~/repos/test-doctor-validation

# Run init (accept defaults - global scope)
autospec init

# Verify init.yml was created
cat .autospec/init.yml
```

**Expected**: File exists with:
- `version: 1`
- `settings_scope: global`
- `autospec_version` matching current version
- `agents` section with claude configured

### 2. Validate `autospec doctor` reads `init.yml` and checks global settings

```bash
cd ~/repos/test-doctor-validation

# Run doctor
autospec doctor
```

**Expected**:
- `✓ Init settings: configured (scope: global, ...)`
- `✓ Claude settings: ... (global)` - should check ~/.claude/settings.json

### 3. Validate legacy fallback (no init.yml)

```bash
cd ~/repos/test-doctor-validation

# Remove init.yml to simulate legacy project
rm .autospec/init.yml

# Run doctor
autospec doctor
```

**Expected**:
- `⚠ Init settings: .autospec/init.yml not found`
- Doctor should still find permissions via fallback (checks both global and project)

### 4. Validate `--project` scope creates correct init.yml

```bash
# Create another test repo
mkdir -p ~/repos/test-doctor-project-scope
cd ~/repos/test-doctor-project-scope
git init

# Run init with --project flag
autospec init --project

# Check init.yml
cat .autospec/init.yml
```

**Expected**: `settings_scope: project`

### 5. Validate doctor checks project settings when scope is project

```bash
cd ~/repos/test-doctor-project-scope
autospec doctor
```

**Expected**: Permission check should indicate `(project)` source

## Cleanup

```bash
rm -rf ~/repos/test-doctor-validation
rm -rf ~/repos/test-doctor-project-scope
```

## Pass Criteria

- [x] `autospec init` creates `.autospec/init.yml` with correct schema
- [x] `init.yml` records `settings_scope: global` by default
- [x] `init.yml` records `settings_scope: project` with `--project` flag
- [x] `autospec doctor` shows init settings status (via source indicator)
- [x] `autospec doctor` checks global settings when scope is global
- [x] `autospec doctor` checks project settings when scope is project
- [x] Legacy projects (no init.yml) still work with fallback behavior

## Validation Results (2026-01-15)

All tests passed:

| Test | Result | Notes |
|------|--------|-------|
| Test 1: init.yml creation | PASS | Created with version, scope, agent info |
| Test 2: doctor global scope | PASS | Shows `(global)` source |
| Test 3: legacy fallback | PASS | Shows `(legacy)` + warning message |
| Test 4: --project scope | PASS | `settings_scope: project` in init.yml |
| Test 5: doctor project scope | PASS | Shows `(project)` source |
