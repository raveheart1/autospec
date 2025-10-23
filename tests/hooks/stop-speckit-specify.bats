#!/usr/bin/env bats
# Tests for stop-speckit-specify.sh hook

setup() {
    load "../test_helper"
    source "${BATS_TEST_DIRNAME}/../../scripts/lib/speckit-validation-lib.sh"

    TEST_TEMP_DIR="${BATS_TMPDIR}/hook-specify-test-$$"
    mkdir -p "$TEST_TEMP_DIR"

    export SPECKIT_SPECS_DIR="$TEST_TEMP_DIR/specs"
    export SPECKIT_DEBUG="false"
    export SPECKIT_RETRY_LIMIT="2"

    mkdir -p "$SPECKIT_SPECS_DIR"

    # Mock git branch to return a spec name
    MOCK_BRANCH="002-test-feature"
}

teardown() {
    rm -rf "$TEST_TEMP_DIR"
    cleanup_retry_state
}

# ------------------------------------------------------------------------------
# T067: Test stop hook with missing spec.md (should block)
# ------------------------------------------------------------------------------

@test "T067: blocks stop when spec.md is missing" {
    # Create spec directory without spec.md
    SPEC_DIR="$SPECKIT_SPECS_DIR/$MOCK_BRANCH"
    mkdir -p "$SPEC_DIR"

    # Mock detect_current_spec to return our test spec
    detect_current_spec() {
        echo "$MOCK_BRANCH"
    }
    export -f detect_current_spec

    # Run hook
    run "${PROJECT_ROOT}/scripts/hooks/stop-speckit-specify.sh" < /dev/null

    [ "$status" -eq 2 ]
    [[ "$output" =~ "blocked" ]] || [[ "$output" =~ "missing" ]]
}

@test "T067: outputs JSON when blocking" {
    SPEC_DIR="$SPECKIT_SPECS_DIR/$MOCK_BRANCH"
    mkdir -p "$SPEC_DIR"

    detect_current_spec() {
        echo "$MOCK_BRANCH"
    }
    export -f detect_current_spec

    result=$("${PROJECT_ROOT}/scripts/hooks/stop-speckit-specify.sh" < /dev/null || true)

    # Validate JSON structure
    echo "$result" | jq -e '.status' > /dev/null
    echo "$result" | jq -e '.reason' > /dev/null
}

@test "T067: increments retry counter when blocking" {
    SPEC_DIR="$SPECKIT_SPECS_DIR/$MOCK_BRANCH"
    mkdir -p "$SPEC_DIR"

    detect_current_spec() {
        echo "$MOCK_BRANCH"
    }
    export -f detect_current_spec

    # Run hook (will fail and increment)
    "${PROJECT_ROOT}/scripts/hooks/stop-speckit-specify.sh" < /dev/null || true

    # Check retry file exists
    retry_count=$(get_retry_count "$MOCK_BRANCH" "specify")
    [ "$retry_count" -eq 1 ]
}

@test "T067: allows stop when retry limit exceeded" {
    SPEC_DIR="$SPECKIT_SPECS_DIR/$MOCK_BRANCH"
    mkdir -p "$SPEC_DIR"

    detect_current_spec() {
        echo "$MOCK_BRANCH"
    }
    export -f detect_current_spec

    # Set retry counter to limit
    echo "2" > "/tmp/.speckit-retry-$MOCK_BRANCH-specify"

    run "${PROJECT_ROOT}/scripts/hooks/stop-speckit-specify.sh" < /dev/null

    [ "$status" -eq 0 ]
    [[ "$output" =~ "allowed" ]] || [[ "$output" =~ "exceeded" ]]
}

# ------------------------------------------------------------------------------
# T068: Test stop hook with existing spec.md (should allow)
# ------------------------------------------------------------------------------

@test "T068: allows stop when spec.md exists" {
    # Create spec directory with spec.md
    SPEC_DIR="$SPECKIT_SPECS_DIR/$MOCK_BRANCH"
    mkdir -p "$SPEC_DIR"
    touch "$SPEC_DIR/spec.md"

    detect_current_spec() {
        echo "$MOCK_BRANCH"
    }
    export -f detect_current_spec

    run "${PROJECT_ROOT}/scripts/hooks/stop-speckit-specify.sh" < /dev/null

    [ "$status" -eq 0 ]
    [[ "$output" =~ "allowed" ]] || [[ "$output" =~ "complete" ]]
}

@test "T068: resets retry counter when spec.md exists" {
    SPEC_DIR="$SPECKIT_SPECS_DIR/$MOCK_BRANCH"
    mkdir -p "$SPEC_DIR"
    touch "$SPEC_DIR/spec.md"

    # Pre-populate retry counter
    echo "1" > "/tmp/.speckit-retry-$MOCK_BRANCH-specify"

    detect_current_spec() {
        echo "$MOCK_BRANCH"
    }
    export -f detect_current_spec

    "${PROJECT_ROOT}/scripts/hooks/stop-speckit-specify.sh" < /dev/null

    # Verify retry counter was reset
    retry_count=$(get_retry_count "$MOCK_BRANCH" "specify")
    [ "$retry_count" -eq 0 ]
}

@test "T068: outputs success JSON when allowing" {
    SPEC_DIR="$SPECKIT_SPECS_DIR/$MOCK_BRANCH"
    mkdir -p "$SPEC_DIR"
    touch "$SPEC_DIR/spec.md"

    detect_current_spec() {
        echo "$MOCK_BRANCH"
    }
    export -f detect_current_spec

    result=$("${PROJECT_ROOT}/scripts/hooks/stop-speckit-specify.sh" < /dev/null)

    echo "$result" | jq -e '.status' > /dev/null
    status_value=$(echo "$result" | jq -r '.status')
    [ "$status_value" = "allowed" ]
}

# ------------------------------------------------------------------------------
# Edge Cases
# ------------------------------------------------------------------------------

@test "allows stop when no active spec detected" {
    detect_current_spec() {
        echo ""
    }
    export -f detect_current_spec

    run "${PROJECT_ROOT}/scripts/hooks/stop-speckit-specify.sh" < /dev/null

    [ "$status" -eq 0 ]
}

@test "allows stop when spec directory not found" {
    detect_current_spec() {
        echo "nonexistent-spec"
    }
    export -f detect_current_spec

    run "${PROJECT_ROOT}/scripts/hooks/stop-speckit-specify.sh" < /dev/null

    [ "$status" -eq 0 ]
}
