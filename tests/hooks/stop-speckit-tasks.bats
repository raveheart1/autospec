#!/usr/bin/env bats
# Tests for stop-speckit-tasks.sh hook (T071-T072)

setup() {
    load "../test_helper"
    source "${BATS_TEST_DIRNAME}/../../scripts/lib/speckit-validation-lib.sh"

    TEST_TEMP_DIR="${BATS_TMPDIR}/hook-tasks-test-$$"
    mkdir -p "$TEST_TEMP_DIR"

    export SPECKIT_SPECS_DIR="$TEST_TEMP_DIR/specs"
    export SPECKIT_DEBUG="false"
    export SPECKIT_RETRY_LIMIT="2"

    mkdir -p "$SPECKIT_SPECS_DIR"
    MOCK_BRANCH="002-test-feature"
}

teardown() {
    rm -rf "$TEST_TEMP_DIR"
    cleanup_retry_state
}

# T071: Test with missing tasks.md (should block)
@test "T071: blocks stop when tasks.md is missing" {
    SPEC_DIR="$SPECKIT_SPECS_DIR/$MOCK_BRANCH"
    mkdir -p "$SPEC_DIR"

    detect_current_spec() { echo "$MOCK_BRANCH"; }
    export -f detect_current_spec

    run "${PROJECT_ROOT}/scripts/hooks/stop-speckit-tasks.sh" < /dev/null

    [ "$status" -eq 2 ]
}

# T072: Test with existing tasks.md (should allow)
@test "T072: allows stop when tasks.md exists" {
    SPEC_DIR="$SPECKIT_SPECS_DIR/$MOCK_BRANCH"
    mkdir -p "$SPEC_DIR"
    touch "$SPEC_DIR/tasks.md"

    detect_current_spec() { echo "$MOCK_BRANCH"; }
    export -f detect_current_spec

    run "${PROJECT_ROOT}/scripts/hooks/stop-speckit-tasks.sh" < /dev/null

    [ "$status" -eq 0 ]
}
