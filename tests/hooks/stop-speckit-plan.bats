#!/usr/bin/env bats
# Tests for stop-speckit-plan.sh hook (T069-T070)

setup() {
    load "../test_helper"
    source "${BATS_TEST_DIRNAME}/../../scripts/lib/speckit-validation-lib.sh"

    TEST_TEMP_DIR="${BATS_TMPDIR}/hook-plan-test-$$"
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

# T069: Test with missing plan.md (should block)
@test "T069: blocks stop when plan.md is missing" {
    SPEC_DIR="$SPECKIT_SPECS_DIR/$MOCK_BRANCH"
    mkdir -p "$SPEC_DIR"

    detect_current_spec() { echo "$MOCK_BRANCH"; }
    export -f detect_current_spec

    run "${PROJECT_ROOT}/scripts/hooks/stop-speckit-plan.sh" < /dev/null

    [ "$status" -eq 2 ]
}

# T070: Test with existing plan.md (should allow)
@test "T070: allows stop when plan.md exists" {
    SPEC_DIR="$SPECKIT_SPECS_DIR/$MOCK_BRANCH"
    mkdir -p "$SPEC_DIR"
    touch "$SPEC_DIR/plan.md"

    detect_current_spec() { echo "$MOCK_BRANCH"; }
    export -f detect_current_spec

    run "${PROJECT_ROOT}/scripts/hooks/stop-speckit-plan.sh" < /dev/null

    [ "$status" -eq 0 ]
}
