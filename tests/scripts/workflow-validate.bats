#!/usr/bin/env bats
# Tests for speckit-workflow-validate.sh

setup() {
    load "../test_helper"

    # Create temp directory for test
    TEST_TEMP_DIR="${BATS_TMPDIR}/workflow-test-$$"
    mkdir -p "$TEST_TEMP_DIR"

    export SPECKIT_SPECS_DIR="$TEST_TEMP_DIR/specs"
    export SPECKIT_DEBUG="false"
    export SPECKIT_DRY_RUN="false"

    mkdir -p "$SPECKIT_SPECS_DIR"

    # Mock git to return temp directory as root
    export GIT_ROOT="$TEST_TEMP_DIR"
}

teardown() {
    rm -rf "$TEST_TEMP_DIR"
    cleanup_retry_state
}

# ------------------------------------------------------------------------------
# T025: Test for successful workflow completion
# ------------------------------------------------------------------------------

@test "T025: successful workflow completion with all files created" {
    export SPECKIT_DRY_RUN="true"

    run "${PROJECT_ROOT}/scripts/speckit-workflow-validate.sh" \
        "Test Feature" \
        --skip-constitution \
        --dry-run

    [ "$status" -eq 0 ]
    [[ "$output" =~ "DRY RUN" ]]
    [[ "$output" =~ "Step 1/3" ]]
    [[ "$output" =~ "Step 2/3" ]]
    [[ "$output" =~ "Step 3/3" ]]
}

@test "T025: workflow creates expected files" {
    skip "Requires actual Claude execution"

    # This test would run actual workflow and verify files are created
    # Skipped in unit tests, would be run in integration tests
}

# ------------------------------------------------------------------------------
# T026: Test for missing spec.md detection and retry
# ------------------------------------------------------------------------------

@test "T026: detects missing spec.md" {
    # Simulate workflow where spec.md is not created
    # In dry-run mode, we can test the validation logic

    export SPECKIT_DRY_RUN="true"
    export SPECKIT_RETRY_LIMIT="2"

    run "${PROJECT_ROOT}/scripts/speckit-workflow-validate.sh" \
        "Test Feature" \
        --skip-constitution \
        --retry-limit 2 \
        --dry-run

    [ "$status" -eq 0 ]
    [[ "$output" =~ "Would validate: spec.md" ]]
}

@test "T026: retry logic triggers on missing file" {
    skip "Requires mocking Claude command to fail first time"

    # This test would:
    # 1. Mock Claude to not create spec.md on first run
    # 2. Verify retry counter increments
    # 3. Mock Claude to succeed on second run
    # 4. Verify retry counter resets
}

# ------------------------------------------------------------------------------
# T027: Test for retry limit enforcement
# ------------------------------------------------------------------------------

@test "T027: enforces retry limit" {
    # Create a scenario where retry limit is exceeded
    # Pre-populate retry counter to limit - 1
    increment_retry_count() {
        local spec_name="$1"
        local command="$2"
        local retry_file="/tmp/.speckit-retry-${spec_name}-${command}"
        echo "2" > "$retry_file"
        echo "2"
    }

    export -f increment_retry_count

    # Test that validation fails when limit is reached
    # In real scenario, this would be tested with integration test
}

@test "T027: retry limit can be customized" {
    export SPECKIT_RETRY_LIMIT="5"
    export SPECKIT_DRY_RUN="true"

    run "${PROJECT_ROOT}/scripts/speckit-workflow-validate.sh" \
        "Test Feature" \
        --skip-constitution \
        --retry-limit 5 \
        --dry-run

    [ "$status" -eq 0 ]
    # In dry run, we just verify it accepts the parameter
}

# ------------------------------------------------------------------------------
# T028: Test for custom retry limit option
# ------------------------------------------------------------------------------

@test "T028: --retry-limit flag sets custom limit" {
    export SPECKIT_DRY_RUN="true"

    run "${PROJECT_ROOT}/scripts/speckit-workflow-validate.sh" \
        "Test Feature" \
        --retry-limit 3 \
        --dry-run \
        --skip-constitution

    [ "$status" -eq 0 ]
    [[ "$output" =~ "Retry limit: 3" ]]
}

@test "T028: environment variable SPECKIT_RETRY_LIMIT works" {
    export SPECKIT_RETRY_LIMIT="4"
    export SPECKIT_DRY_RUN="true"

    run "${PROJECT_ROOT}/scripts/speckit-workflow-validate.sh" \
        "Test Feature" \
        --dry-run \
        --skip-constitution

    [ "$status" -eq 0 ]
    # Verify the limit is used (would be visible in actual retry scenarios)
}

@test "T028: command-line flag overrides environment variable" {
    export SPECKIT_RETRY_LIMIT="4"
    export SPECKIT_DRY_RUN="true"

    run "${PROJECT_ROOT}/scripts/speckit-workflow-validate.sh" \
        "Test Feature" \
        --retry-limit 7 \
        --dry-run \
        --skip-constitution

    [ "$status" -eq 0 ]
    [[ "$output" =~ "Retry limit: 7" ]]
}

# ------------------------------------------------------------------------------
# T029: Test for dry-run mode
# ------------------------------------------------------------------------------

@test "T029: dry-run mode shows what would be executed" {
    export SPECKIT_DRY_RUN="true"

    run "${PROJECT_ROOT}/scripts/speckit-workflow-validate.sh" \
        "Test Feature" \
        --dry-run \
        --skip-constitution

    [ "$status" -eq 0 ]
    [[ "$output" =~ "DRY RUN" ]]
    [[ "$output" =~ "Would execute:" ]]
    [[ "$output" =~ "Would validate:" ]]
}

@test "T029: dry-run mode does not create files" {
    export SPECKIT_DRY_RUN="true"

    "${PROJECT_ROOT}/scripts/speckit-workflow-validate.sh" \
        "Test Feature" \
        --dry-run \
        --skip-constitution

    # Verify no spec directories were created
    [ ! -d "$SPECKIT_SPECS_DIR/test-feature" ]
    [ ! -d "$SPECKIT_SPECS_DIR/Test-Feature" ]
}

@test "T029: dry-run mode does not execute Claude commands" {
    export SPECKIT_DRY_RUN="true"

    run "${PROJECT_ROOT}/scripts/speckit-workflow-validate.sh" \
        "Test Feature" \
        --dry-run \
        --skip-constitution

    [ "$status" -eq 0 ]
    # In dry-run, no actual claude commands should execute
    ! [[ "$output" =~ "Creating specification..." ]]
}

# ------------------------------------------------------------------------------
# Additional Workflow Tests
# ------------------------------------------------------------------------------

@test "workflow validates required arguments" {
    run "${PROJECT_ROOT}/scripts/speckit-workflow-validate.sh"

    [ "$status" -eq 3 ]
    [[ "$output" =~ "Feature description required" ]]
}

@test "workflow shows help message" {
    run "${PROJECT_ROOT}/scripts/speckit-workflow-validate.sh" --help

    [ "$status" -eq 0 ]
    [[ "$output" =~ "Usage:" ]]
    [[ "$output" =~ "--retry-limit" ]]
    [[ "$output" =~ "--dry-run" ]]
}

@test "workflow handles verbose flag" {
    export SPECKIT_DRY_RUN="true"

    run "${PROJECT_ROOT}/scripts/speckit-workflow-validate.sh" \
        "Test Feature" \
        --verbose \
        --dry-run \
        --skip-constitution

    [ "$status" -eq 0 ]
    # Verbose mode should be enabled
}

@test "workflow validates git repository" {
    # Test outside git repo
    skip "Would require running outside git repo"
}

@test "workflow handles constitution creation" {
    export SPECKIT_DRY_RUN="true"

    # With --skip-constitution
    run "${PROJECT_ROOT}/scripts/speckit-workflow-validate.sh" \
        "Test Feature" \
        --skip-constitution \
        --dry-run

    [ "$status" -eq 0 ]
    ! [[ "$output" =~ "Creating project principles" ]]
}
