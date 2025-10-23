#!/usr/bin/env bats
# Tests for speckit-implement-validate.sh

setup() {
    load "../test_helper"

    TEST_TEMP_DIR="${BATS_TMPDIR}/implement-test-$$"
    mkdir -p "$TEST_TEMP_DIR"

    export SPECKIT_SPECS_DIR="$TEST_TEMP_DIR/specs"
    export SPECKIT_DEBUG="false"

    mkdir -p "$SPECKIT_SPECS_DIR"
}

teardown() {
    rm -rf "$TEST_TEMP_DIR"
    cleanup_retry_state
}

# ------------------------------------------------------------------------------
# T043: Test for all phases complete detection
# ------------------------------------------------------------------------------

@test "T043: detects all phases complete" {
    # Create spec directory
    SPEC_DIR="$SPECKIT_SPECS_DIR/test-feature"
    mkdir -p "$SPEC_DIR"

    # Create tasks.md with all tasks complete
    cat > "$SPEC_DIR/tasks.md" << 'EOF'
## Phase 1: Setup
- [x] Task 1
- [x] Task 2

## Phase 2: Implementation
- [x] Task 3
- [x] Task 4
EOF

    run "${PROJECT_ROOT}/scripts/speckit-implement-validate.sh" test-feature

    [ "$status" -eq 0 ]
    [[ "$output" =~ "All phases complete" ]]
}

@test "T043: returns exit code 0 for complete phases" {
    SPEC_DIR="$SPECKIT_SPECS_DIR/complete-spec"
    mkdir -p "$SPEC_DIR"

    cat > "$SPEC_DIR/tasks.md" << 'EOF'
## Phase 1: Setup
- [x] Task 1
EOF

    run "${PROJECT_ROOT}/scripts/speckit-implement-validate.sh" complete-spec

    [ "$status" -eq 0 ]
}

# ------------------------------------------------------------------------------
# T044: Test for incomplete phases detection
# ------------------------------------------------------------------------------

@test "T044: detects incomplete phases" {
    SPEC_DIR="$SPECKIT_SPECS_DIR/incomplete-spec"
    mkdir -p "$SPEC_DIR"

    cat > "$SPEC_DIR/tasks.md" << 'EOF'
## Phase 1: Setup
- [x] Task 1
- [x] Task 2

## Phase 2: Implementation
- [x] Task 3
- [ ] Task 4
- [ ] Task 5
EOF

    run "${PROJECT_ROOT}/scripts/speckit-implement-validate.sh" incomplete-spec

    [ "$status" -eq 1 ]
    [[ "$output" =~ "INCOMPLETE" ]]
}

@test "T044: lists unchecked tasks for incomplete phases" {
    SPEC_DIR="$SPECKIT_SPECS_DIR/incomplete-spec2"
    mkdir -p "$SPEC_DIR"

    cat > "$SPEC_DIR/tasks.md" << 'EOF'
## Phase 1: Setup
- [x] Task 1

## Phase 2: Implementation
- [ ] Add feature X
- [ ] Add feature Y
EOF

    run "${PROJECT_ROOT}/scripts/speckit-implement-validate.sh" incomplete-spec2

    [[ "$output" =~ "Add feature X" ]]
    [[ "$output" =~ "Add feature Y" ]]
}

@test "T044: shows completion status for each phase" {
    SPEC_DIR="$SPECKIT_SPECS_DIR/mixed-spec"
    mkdir -p "$SPEC_DIR"

    cat > "$SPEC_DIR/tasks.md" << 'EOF'
## Phase 1: Setup
- [x] Task 1
- [x] Task 2

## Phase 2: Implementation
- [x] Task 3
- [ ] Task 4
EOF

    run "${PROJECT_ROOT}/scripts/speckit-implement-validate.sh" mixed-spec

    [[ "$output" =~ "✓ Phase 1" ]]
    [[ "$output" =~ "✗ Phase 2" ]]
}

# ------------------------------------------------------------------------------
# T045: Test for continuation prompt generation
# ------------------------------------------------------------------------------

@test "T045: generates continuation prompt with --continuation flag" {
    SPEC_DIR="$SPECKIT_SPECS_DIR/cont-spec"
    mkdir -p "$SPEC_DIR"

    cat > "$SPEC_DIR/tasks.md" << 'EOF'
## Phase 1: Setup
- [x] Task 1

## Phase 2: Implementation
- [ ] Task 2
- [ ] Task 3
EOF

    run "${PROJECT_ROOT}/scripts/speckit-implement-validate.sh" \
        cont-spec \
        --continuation

    [ "$status" -eq 1 ]
    [[ "$output" =~ "incomplete phases" ]]
    [[ "$output" =~ "Phase 2" ]]
    [[ "$output" =~ "continue" ]]
}

@test "T045: continuation prompt includes phase details" {
    SPEC_DIR="$SPECKIT_SPECS_DIR/detail-spec"
    mkdir -p "$SPEC_DIR"

    cat > "$SPEC_DIR/tasks.md" << 'EOF'
## Phase 1: Testing
- [ ] Write tests
- [ ] Run tests
EOF

    run "${PROJECT_ROOT}/scripts/speckit-implement-validate.sh" \
        detail-spec \
        --continuation

    [[ "$output" =~ "Testing" ]]
    [[ "$output" =~ "Write tests" ]]
    [[ "$output" =~ "Run tests" ]]
}

@test "T045: continuation prompt shows task counts" {
    SPEC_DIR="$SPECKIT_SPECS_DIR/count-spec"
    mkdir -p "$SPEC_DIR"

    cat > "$SPEC_DIR/tasks.md" << 'EOF'
## Phase 1: Implementation
- [x] Task 1
- [ ] Task 2
- [ ] Task 3
EOF

    run "${PROJECT_ROOT}/scripts/speckit-implement-validate.sh" \
        count-spec \
        --continuation

    [[ "$output" =~ "2/3" ]] || [[ "$output" =~ "remaining" ]]
}

# ------------------------------------------------------------------------------
# T046: Test for retry limit enforcement
# ------------------------------------------------------------------------------

@test "T046: increments retry counter on incomplete phases" {
    SPEC_DIR="$SPECKIT_SPECS_DIR/retry-spec"
    mkdir -p "$SPEC_DIR"

    cat > "$SPEC_DIR/tasks.md" << 'EOF'
## Phase 1: Setup
- [ ] Task 1
EOF

    # First run
    "${PROJECT_ROOT}/scripts/speckit-implement-validate.sh" retry-spec > /dev/null 2>&1 || true

    # Check retry file exists
    [ -f "/tmp/.speckit-retry-retry-spec-implement" ]
}

@test "T046: returns exit code 2 when retry limit exceeded" {
    SPEC_DIR="$SPECKIT_SPECS_DIR/limit-spec"
    mkdir -p "$SPEC_DIR"

    cat > "$SPEC_DIR/tasks.md" << 'EOF'
## Phase 1: Setup
- [ ] Task 1
EOF

    # Manually set retry counter to limit
    echo "2" > "/tmp/.speckit-retry-limit-spec-implement"

    export SPECKIT_RETRY_LIMIT="2"

    run "${PROJECT_ROOT}/scripts/speckit-implement-validate.sh" \
        limit-spec \
        --retry-limit 2

    [ "$status" -eq 2 ]
}

@test "T046: retry counter can be reset" {
    SPEC_DIR="$SPECKIT_SPECS_DIR/reset-spec"
    mkdir -p "$SPEC_DIR"

    cat > "$SPEC_DIR/tasks.md" << 'EOF'
## Phase 1: Setup
- [ ] Task 1
EOF

    # Create retry file
    echo "1" > "/tmp/.speckit-retry-reset-spec-implement"

    run "${PROJECT_ROOT}/scripts/speckit-implement-validate.sh" \
        reset-spec \
        --reset-retry

    [[ "$output" =~ "reset" ]]

    # Verify file is deleted
    [ ! -f "/tmp/.speckit-retry-reset-spec-implement" ]
}

# ------------------------------------------------------------------------------
# T047: Test for JSON output format
# ------------------------------------------------------------------------------

@test "T047: --json flag produces valid JSON" {
    SPEC_DIR="$SPECKIT_SPECS_DIR/json-spec"
    mkdir -p "$SPEC_DIR"

    cat > "$SPEC_DIR/tasks.md" << 'EOF'
## Phase 1: Setup
- [x] Task 1
- [ ] Task 2
EOF

    run "${PROJECT_ROOT}/scripts/speckit-implement-validate.sh" \
        json-spec \
        --json

    # Validate JSON with jq
    echo "$output" | jq -e '.spec_name' > /dev/null
    echo "$output" | jq -e '.status' > /dev/null
    echo "$output" | jq -e '.phases' > /dev/null
}

@test "T047: JSON includes phase details" {
    SPEC_DIR="$SPECKIT_SPECS_DIR/json-detail-spec"
    mkdir -p "$SPEC_DIR"

    cat > "$SPEC_DIR/tasks.md" << 'EOF'
## Phase 1: Implementation
- [x] Task 1
- [ ] Task 2
- [ ] Task 3
EOF

    run "${PROJECT_ROOT}/scripts/speckit-implement-validate.sh" \
        json-detail-spec \
        --json

    # Check JSON structure
    status_value=$(echo "$output" | jq -r '.status')
    [ "$status_value" = "incomplete" ]

    # Check phase information
    echo "$output" | jq -e '.phases[0].phase_number' > /dev/null
    echo "$output" | jq -e '.phases[0].phase_name' > /dev/null
    echo "$output" | jq -e '.phases[0].unchecked_tasks' > /dev/null
}

@test "T047: JSON output includes retry information" {
    SPEC_DIR="$SPECKIT_SPECS_DIR/json-retry-spec"
    mkdir -p "$SPEC_DIR"

    cat > "$SPEC_DIR/tasks.md" << 'EOF'
## Phase 1: Setup
- [ ] Task 1
EOF

    run "${PROJECT_ROOT}/scripts/speckit-implement-validate.sh" \
        json-retry-spec \
        --json

    echo "$output" | jq -e '.retry_count' > /dev/null
    echo "$output" | jq -e '.retry_limit' > /dev/null
    echo "$output" | jq -e '.can_retry' > /dev/null
}

# ------------------------------------------------------------------------------
# T048: Test for spec name resolution
# ------------------------------------------------------------------------------

@test "T048: resolves spec by full name" {
    SPEC_DIR="$SPECKIT_SPECS_DIR/full-name-spec"
    mkdir -p "$SPEC_DIR"

    cat > "$SPEC_DIR/tasks.md" << 'EOF'
## Phase 1: Setup
- [x] Task 1
EOF

    run "${PROJECT_ROOT}/scripts/speckit-implement-validate.sh" full-name-spec

    [ "$status" -eq 0 ]
}

@test "T048: resolves spec by directory number" {
    SPEC_DIR="$SPECKIT_SPECS_DIR/001-numbered-spec"
    mkdir -p "$SPEC_DIR"

    cat > "$SPEC_DIR/tasks.md" << 'EOF'
## Phase 1: Setup
- [x] Task 1
EOF

    run "${PROJECT_ROOT}/scripts/speckit-implement-validate.sh" 001-numbered-spec

    [ "$status" -eq 0 ]
}

@test "T048: resolves spec by partial name match" {
    SPEC_DIR="$SPECKIT_SPECS_DIR/002-my-feature-spec"
    mkdir -p "$SPEC_DIR"

    cat > "$SPEC_DIR/tasks.md" << 'EOF'
## Phase 1: Setup
- [x] Task 1
EOF

    run "${PROJECT_ROOT}/scripts/speckit-implement-validate.sh" my-feature

    [ "$status" -eq 0 ]
}

@test "T048: returns error for nonexistent spec" {
    run "${PROJECT_ROOT}/scripts/speckit-implement-validate.sh" nonexistent-spec

    [ "$status" -eq 4 ]
    [[ "$output" =~ "not found" ]]
}

@test "T048: returns error when tasks.md missing" {
    SPEC_DIR="$SPECKIT_SPECS_DIR/no-tasks-spec"
    mkdir -p "$SPEC_DIR"
    # Don't create tasks.md

    run "${PROJECT_ROOT}/scripts/speckit-implement-validate.sh" no-tasks-spec

    [ "$status" -eq 4 ]
    [[ "$output" =~ "tasks.md not found" ]]
}

# ------------------------------------------------------------------------------
# Additional Tests
# ------------------------------------------------------------------------------

@test "shows help message" {
    run "${PROJECT_ROOT}/scripts/speckit-implement-validate.sh" --help

    [ "$status" -eq 0 ]
    [[ "$output" =~ "Usage:" ]]
    [[ "$output" =~ "--json" ]]
    [[ "$output" =~ "--continuation" ]]
}

@test "validates required arguments" {
    run "${PROJECT_ROOT}/scripts/speckit-implement-validate.sh"

    [ "$status" -eq 3 ]
    [[ "$output" =~ "Spec name required" ]]
}
