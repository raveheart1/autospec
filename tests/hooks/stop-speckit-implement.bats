#!/usr/bin/env bats
# Tests for stop-speckit-implement.sh hook (T073-T075)

setup() {
    load "../test_helper"
    source "${BATS_TEST_DIRNAME}/../../scripts/lib/speckit-validation-lib.sh"

    TEST_TEMP_DIR="${BATS_TMPDIR}/hook-implement-test-$$"
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

# T073: Test with incomplete phases (should block)
@test "T073: blocks stop when phases are incomplete" {
    SPEC_DIR="$SPECKIT_SPECS_DIR/$MOCK_BRANCH"
    mkdir -p "$SPEC_DIR"

    cat > "$SPEC_DIR/tasks.md" << 'EOF'
## Phase 1: Setup
- [x] Task 1

## Phase 2: Implementation
- [ ] Task 2
- [ ] Task 3
EOF

    detect_current_spec() { echo "$MOCK_BRANCH"; }
    export -f detect_current_spec

    run "${PROJECT_ROOT}/scripts/hooks/stop-speckit-implement.sh" < /dev/null

    [ "$status" -eq 2 ]
    [[ "$output" =~ "blocked" ]] || [[ "$output" =~ "incomplete" ]]
}

@test "T073: includes phase details when blocking" {
    SPEC_DIR="$SPECKIT_SPECS_DIR/$MOCK_BRANCH"
    mkdir -p "$SPEC_DIR"

    cat > "$SPEC_DIR/tasks.md" << 'EOF'
## Phase 1: Testing
- [ ] Test 1
EOF

    detect_current_spec() { echo "$MOCK_BRANCH"; }
    export -f detect_current_spec

    result=$("${PROJECT_ROOT}/scripts/hooks/stop-speckit-implement.sh" < /dev/null || true)

    # Check for phase information in JSON
    echo "$result" | jq -e '.details.incomplete_phases' > /dev/null
}

# T074: Test with complete phases (should allow)
@test "T074: allows stop when all phases complete" {
    SPEC_DIR="$SPECKIT_SPECS_DIR/$MOCK_BRANCH"
    mkdir -p "$SPEC_DIR"

    cat > "$SPEC_DIR/tasks.md" << 'EOF'
## Phase 1: Setup
- [x] Task 1
- [x] Task 2

## Phase 2: Implementation
- [x] Task 3
- [x] Task 4
EOF

    detect_current_spec() { echo "$MOCK_BRANCH"; }
    export -f detect_current_spec

    run "${PROJECT_ROOT}/scripts/hooks/stop-speckit-implement.sh" < /dev/null

    [ "$status" -eq 0 ]
    [[ "$output" =~ "allowed" ]] || [[ "$output" =~ "complete" ]]
}

@test "T074: resets retry counter when complete" {
    SPEC_DIR="$SPECKIT_SPECS_DIR/$MOCK_BRANCH"
    mkdir -p "$SPEC_DIR"

    cat > "$SPEC_DIR/tasks.md" << 'EOF'
## Phase 1: Setup
- [x] Task 1
EOF

    # Set retry counter
    echo "1" > "/tmp/.speckit-retry-$MOCK_BRANCH-implement"

    detect_current_spec() { echo "$MOCK_BRANCH"; }
    export -f detect_current_spec

    "${PROJECT_ROOT}/scripts/hooks/stop-speckit-implement.sh" < /dev/null

    retry_count=$(get_retry_count "$MOCK_BRANCH" "implement")
    [ "$retry_count" -eq 0 ]
}

# T075: Test retry limit enforcement
@test "T075: allows stop when retry limit exceeded" {
    SPEC_DIR="$SPECKIT_SPECS_DIR/$MOCK_BRANCH"
    mkdir -p "$SPEC_DIR"

    cat > "$SPEC_DIR/tasks.md" << 'EOF'
## Phase 1: Setup
- [ ] Task 1
EOF

    # Set retry to limit
    echo "2" > "/tmp/.speckit-retry-$MOCK_BRANCH-implement"

    detect_current_spec() { echo "$MOCK_BRANCH"; }
    export -f detect_current_spec

    run "${PROJECT_ROOT}/scripts/hooks/stop-speckit-implement.sh" < /dev/null

    [ "$status" -eq 0 ]
}

@test "T075: increments retry counter when blocking" {
    SPEC_DIR="$SPECKIT_SPECS_DIR/$MOCK_BRANCH"
    mkdir -p "$SPEC_DIR"

    cat > "$SPEC_DIR/tasks.md" << 'EOF'
## Phase 1: Setup
- [ ] Task 1
EOF

    detect_current_spec() { echo "$MOCK_BRANCH"; }
    export -f detect_current_spec

    "${PROJECT_ROOT}/scripts/hooks/stop-speckit-implement.sh" < /dev/null || true

    retry_count=$(get_retry_count "$MOCK_BRANCH" "implement")
    [ "$retry_count" -eq 1 ]
}

# Edge cases
@test "allows stop when no tasks.md found" {
    SPEC_DIR="$SPECKIT_SPECS_DIR/$MOCK_BRANCH"
    mkdir -p "$SPEC_DIR"
    # No tasks.md

    detect_current_spec() { echo "$MOCK_BRANCH"; }
    export -f detect_current_spec

    run "${PROJECT_ROOT}/scripts/hooks/stop-speckit-implement.sh" < /dev/null

    [ "$status" -eq 0 ]
}
