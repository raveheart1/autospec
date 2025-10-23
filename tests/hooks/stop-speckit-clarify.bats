#!/usr/bin/env bats
# Tests for stop-speckit-clarify.sh hook (T076-T077)

setup() {
    load "../test_helper"
    source "${BATS_TEST_DIRNAME}/../../scripts/lib/speckit-validation-lib.sh"

    TEST_TEMP_DIR="${BATS_TMPDIR}/hook-clarify-test-$$"
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

# T076: Test with no spec update (should block)
@test "T076: blocks stop when spec.md was not updated" {
    SPEC_DIR="$SPECKIT_SPECS_DIR/$MOCK_BRANCH"
    mkdir -p "$SPEC_DIR"

    # Create spec.md with old timestamp (more than 5 minutes ago)
    touch -t 202301010000 "$SPEC_DIR/spec.md"

    detect_current_spec() { echo "$MOCK_BRANCH"; }
    export -f detect_current_spec

    run "${PROJECT_ROOT}/scripts/hooks/stop-speckit-clarify.sh" < /dev/null

    [ "$status" -eq 2 ]
}

# T077: Test with spec updated (should allow)
@test "T077: allows stop when spec.md was recently updated" {
    SPEC_DIR="$SPECKIT_SPECS_DIR/$MOCK_BRANCH"
    mkdir -p "$SPEC_DIR"

    # Create spec.md with current timestamp
    touch "$SPEC_DIR/spec.md"

    detect_current_spec() { echo "$MOCK_BRANCH"; }
    export -f detect_current_spec

    run "${PROJECT_ROOT}/scripts/hooks/stop-speckit-clarify.sh" < /dev/null

    [ "$status" -eq 0 ]
}

@test "allows stop when no spec.md exists" {
    SPEC_DIR="$SPECKIT_SPECS_DIR/$MOCK_BRANCH"
    mkdir -p "$SPEC_DIR"
    # No spec.md

    detect_current_spec() { echo "$MOCK_BRANCH"; }
    export -f detect_current_spec

    run "${PROJECT_ROOT}/scripts/hooks/stop-speckit-clarify.sh" < /dev/null

    [ "$status" -eq 0 ]
}
