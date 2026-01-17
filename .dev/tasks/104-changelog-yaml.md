# Manual Testing Plan: YAML-First Changelog Management (104)

This document outlines the manual testing steps for verifying the YAML-first changelog management feature.

## Prerequisites

- autospec installed and built (`make build`)
- Access to a terminal
- Git repository initialized

## Test Scenarios

### 1. View Changelog (Default)

**Objective**: Verify `autospec changelog` shows recent entries from embedded changelog.

**Steps**:
1. Run: `autospec changelog`
2. Verify:
   - Shows 5 most recent entries (default)
   - Entries have color-coded categories (Added=green, Fixed=yellow, Changed=blue)
   - Shows "(X of Y entries shown)" message
   - No errors occur

**Expected Result**: Color-coded changelog entries displayed, count shown at bottom.

### 2. View Specific Version

**Objective**: Verify changelog for a specific version can be viewed.

**Steps**:
1. Run: `autospec changelog v0.9.0`
2. Run: `autospec changelog 0.9.0` (without v prefix)
3. Verify:
   - Both commands show the same output
   - All entries for version 0.9.0 are displayed
   - Version header is shown

**Expected Result**: Version-specific changelog displayed, v prefix optional.

### 3. View Unreleased Changes

**Objective**: Verify unreleased changes can be viewed.

**Steps**:
1. Run: `autospec changelog unreleased`
2. Verify:
   - Unreleased entries are displayed
   - Categories are shown (Added, Changed, Fixed, etc.)

**Expected Result**: Unreleased changelog entries displayed.

### 4. Control Entry Count with --last

**Objective**: Verify --last flag controls entry count.

**Steps**:
1. Run: `autospec changelog --last 3`
2. Run: `autospec changelog --last 10`
3. Verify:
   - Entry count matches the --last value (up to total available)
   - "(X of Y entries shown)" message is accurate

**Expected Result**: Entry count matches --last parameter.

### 5. Plain Output Mode

**Objective**: Verify --plain flag disables colors and icons.

**Steps**:
1. Run: `autospec changelog --plain`
2. Compare with: `autospec changelog`
3. Verify:
   - No ANSI color codes in --plain output
   - Category names still appear
   - Icons are not present in --plain mode

**Expected Result**: Plain text output without colors or icons.

### 6. Extract Release Notes

**Objective**: Verify `autospec changelog extract` outputs markdown suitable for GitHub releases.

**Steps**:
1. Run: `autospec changelog extract v0.9.0`
2. Verify:
   - Output is valid markdown
   - Uses `### Category` headers
   - Uses `- entry` list format
   - No ANSI colors

**Expected Result**: Clean markdown output for GitHub release notes.

### 7. Error Handling - Non-existent Version

**Objective**: Verify appropriate error when version doesn't exist.

**Steps**:
1. Run: `autospec changelog v99.99.99`
2. Verify:
   - Error message mentions "not found"
   - Available versions are listed
   - Exit code is non-zero

**Expected Result**: Clear error with available versions listed.

### 8. Changelog Sync

**Objective**: Verify `make changelog-sync` regenerates CHANGELOG.md from YAML.

**Steps**:
1. Make a backup: `cp CHANGELOG.md CHANGELOG.md.bak`
2. Modify CHANGELOG.md manually (add a test line)
3. Run: `make changelog-sync`
4. Verify:
   - CHANGELOG.md is regenerated
   - Manual change is removed
   - Output shows "Synced internal/changelog/changelog.yaml → CHANGELOG.md"

**Expected Result**: CHANGELOG.md regenerated from YAML source.

### 9. Changelog Check

**Objective**: Verify `make changelog-check` detects out-of-sync state.

**Steps**:
1. Ensure in-sync state: `make changelog-sync`
2. Run: `make changelog-check`
3. Verify: Output shows "✓ CHANGELOG.md is in sync"
4. Modify CHANGELOG.md manually
5. Run: `make changelog-check`
6. Verify: Output shows "✗ CHANGELOG.md is out of sync" and suggests fix

**Expected Result**: Check detects sync/out-of-sync state correctly.

### 10. Help Text Verification

**Objective**: Verify help text for changelog commands.

**Steps**:
1. Run: `autospec changelog --help`
2. Run: `autospec changelog extract --help`
3. Run: `autospec changelog sync --help`
4. Run: `autospec changelog check --help`
5. Verify:
   - Each command has clear description
   - Examples are provided
   - Flags are documented

**Expected Result**: Clear and helpful documentation for all subcommands.

## Report Summaries

_Completed 2026-01-16._

| Test | Status | Notes |
|------|--------|-------|
| 1. View Changelog (Default) | ✓ Pass | Shows 5 entries, color-coded categories, count message |
| 2. View Specific Version | ✓ Pass | Both v0.9.0 and 0.9.0 show identical output with header |
| 3. View Unreleased Changes | ✓ Pass | All categories displayed (Added, Changed, Removed, Fixed) |
| 4. Control Entry Count | ✓ Pass | --last 3 and --last 10 return correct counts |
| 5. Plain Output Mode | ✓ Pass | Uses `### Category` headers instead of icons |
| 6. Extract Release Notes | ✓ Pass | Clean markdown with `### Category` and `- entry` format |
| 7. Error Handling | ✓ Pass | Clear error, lists available versions, exit code 1 |
| 8. Changelog Sync | ✓ Pass | Regenerates CHANGELOG.md, removes manual changes |
| 9. Changelog Check | ✓ Pass | Detects sync/out-of-sync correctly, suggests fix |
| 10. Help Text | ✓ Pass | All commands have descriptions, examples, documented flags |

**Overall Assessment**: All 10 tests passed. YAML-first changelog management is working as expected.

## User Story Coverage

| User Story | Test Coverage |
|------------|---------------|
| US-001: Edit changelog in structured YAML format | Tests 8, 9 |
| US-002: Auto-generate CHANGELOG.md from YAML | Tests 8, 9 |
| US-003: View changelog via CLI command | Tests 1-5, 7, 10 |
| US-004: See what changed after update | Integration with update command |
| US-005: Preview changes when update available | Integration with ck command |
| US-006: Extract release notes for GitHub releases | Test 6 |
| US-007: Draft changelog entries from commits | /changelog command |
| US-008: Release workflow uses YAML source | /release command |

## Additional Notes

- The embedded changelog shows entries up to when the binary was built
- Remote changelog fetching is used for `autospec ck` to show unreleased changes
- The /changelog and /release slash commands have been updated for YAML-first workflow
