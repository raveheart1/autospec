#!/usr/bin/env bats
# T084: Quickstart examples validation
# Tests that all examples from quickstart.md work correctly

setup() {
    load "test_helper"

    TEST_TEMP_DIR="${BATS_TMPDIR}/quickstart-test-$$"
    mkdir -p "$TEST_TEMP_DIR"

    export SPECKIT_SPECS_DIR="$TEST_TEMP_DIR/specs"
    export SPECKIT_DEBUG="false"
    export SPECKIT_DRY_RUN="true"  # Use dry-run for quickstart tests

    mkdir -p "$SPECKIT_SPECS_DIR"
}

teardown() {
    rm -rf "$TEST_TEMP_DIR"
    cleanup_retry_state
}

# ------------------------------------------------------------------------------
# Use Case 1: Full Workflow with Validation
# ------------------------------------------------------------------------------

@test "quickstart example 1: full validated workflow executes" {
    run "${PROJECT_ROOT}/scripts/speckit-workflow-validate.sh" \
        "Add Mastodon crossposting support" \
        --skip-constitution \
        --dry-run

    [ "$status" -eq 0 ]
    [[ "$output" =~ "Step 1/3" ]]
    [[ "$output" =~ "Step 2/3" ]]
    [[ "$output" =~ "Step 3/3" ]]
}

# ------------------------------------------------------------------------------
# Use Case 2: Retry on Failure
# ------------------------------------------------------------------------------

@test "quickstart example 2: retry logic demonstration" {
    # This would test the retry scenario described in quickstart
    # In dry-run mode, we verify the flags work
    run "${PROJECT_ROOT}/scripts/speckit-workflow-validate.sh" \
        "Add feature" \
        --retry-limit 3 \
        --skip-constitution \
        --dry-run

    [ "$status" -eq 0 ]
    [[ "$output" =~ "Retry limit: 3" ]]
}

# ------------------------------------------------------------------------------
# Use Case 3: Validate Incomplete Implementation
# ------------------------------------------------------------------------------

@test "quickstart example 3: implementation validation" {
    SPEC_DIR="$SPECKIT_SPECS_DIR/my-feature-name"
    mkdir -p "$SPEC_DIR"

    cat > "$SPEC_DIR/tasks.md" << 'EOF'
## Phase 1: Setup
- [x] Task 1

## Phase 2: Implementation
- [ ] Task 2
EOF

    run "${PROJECT_ROOT}/scripts/speckit-implement-validate.sh" my-feature

    [ "$status" -eq 1 ]
    [[ "$output" =~ "INCOMPLETE" ]]
}

# ------------------------------------------------------------------------------
# Use Case 4: Generate Continuation Prompt
# ------------------------------------------------------------------------------

@test "quickstart example 4: continuation prompt generation" {
    SPEC_DIR="$SPECKIT_SPECS_DIR/my-feature"
    mkdir -p "$SPEC_DIR"

    cat > "$SPEC_DIR/tasks.md" << 'EOF'
## Phase 1: Testing
- [ ] Add integration tests
- [ ] Test hook payloads
- [ ] Document test coverage
EOF

    run "${PROJECT_ROOT}/scripts/speckit-implement-validate.sh" \
        my-feature \
        --continuation

    [ "$status" -eq 1 ]
    [[ "$output" =~ "incomplete phases" ]]
    [[ "$output" =~ "Testing" ]]
}

# ------------------------------------------------------------------------------
# Use Case 5: JSON Output for Scripting
# ------------------------------------------------------------------------------

@test "quickstart example 5: JSON output" {
    SPEC_DIR="$SPECKIT_SPECS_DIR/json-feature"
    mkdir -p "$SPEC_DIR"

    cat > "$SPEC_DIR/tasks.md" << 'EOF'
## Phase 1: Setup
- [x] Task 1
- [ ] Task 2
EOF

    run "${PROJECT_ROOT}/scripts/speckit-implement-validate.sh" \
        json-feature \
        --json

    # Verify valid JSON
    echo "$output" | jq -e '.spec_name' > /dev/null
    echo "$output" | jq -e '.status' > /dev/null
    echo "$output" | jq -e '.phases' > /dev/null
}

# ------------------------------------------------------------------------------
# Configuration Options
# ------------------------------------------------------------------------------

@test "quickstart: environment variable SPECKIT_RETRY_LIMIT" {
    export SPECKIT_RETRY_LIMIT="3"

    run "${PROJECT_ROOT}/scripts/speckit-workflow-validate.sh" \
        "Test" \
        --skip-constitution \
        --dry-run

    [ "$status" -eq 0 ]
}

@test "quickstart: SPECKIT_DEBUG enables debug logging" {
    export SPECKIT_DEBUG="true"

    SPEC_DIR="$SPECKIT_SPECS_DIR/debug-test"
    mkdir -p "$SPEC_DIR"

    cat > "$SPEC_DIR/tasks.md" << 'EOF'
## Phase 1: Setup
- [x] Task 1
EOF

    run "${PROJECT_ROOT}/scripts/speckit-implement-validate.sh" debug-test

    # Debug output should appear in stderr
    [ "$status" -eq 0 ]
}

@test "quickstart: dry-run mode" {
    run "${PROJECT_ROOT}/scripts/speckit-workflow-validate.sh" \
        "Test Feature" \
        --dry-run \
        --skip-constitution

    [ "$status" -eq 0 ]
    [[ "$output" =~ "DRY RUN" ]]
}

# ------------------------------------------------------------------------------
# Command-Line Options
# ------------------------------------------------------------------------------

@test "quickstart: --help shows usage" {
    run "${PROJECT_ROOT}/scripts/speckit-workflow-validate.sh" --help

    [ "$status" -eq 0 ]
    [[ "$output" =~ "Usage:" ]]
}

@test "quickstart: --retry-limit flag" {
    run "${PROJECT_ROOT}/scripts/speckit-workflow-validate.sh" \
        "Test" \
        --retry-limit 5 \
        --skip-constitution \
        --dry-run

    [ "$status" -eq 0 ]
    [[ "$output" =~ "Retry limit: 5" ]]
}

@test "quickstart: --verbose flag" {
    run "${PROJECT_ROOT}/scripts/speckit-workflow-validate.sh" \
        "Test" \
        --verbose \
        --skip-constitution \
        --dry-run

    [ "$status" -eq 0 ]
}

@test "quickstart: --reset-retry flag" {
    SPEC_DIR="$SPECKIT_SPECS_DIR/reset-test"
    mkdir -p "$SPEC_DIR"

    cat > "$SPEC_DIR/tasks.md" << 'EOF'
## Phase 1: Setup
- [ ] Task 1
EOF

    run "${PROJECT_ROOT}/scripts/speckit-implement-validate.sh" \
        reset-test \
        --reset-retry

    [[ "$output" =~ "reset" ]]
}

# ------------------------------------------------------------------------------
# Integration Scenarios
# ------------------------------------------------------------------------------

@test "quickstart: spec name resolution by number" {
    SPEC_DIR="$SPECKIT_SPECS_DIR/002-test-feature"
    mkdir -p "$SPEC_DIR"

    cat > "$SPEC_DIR/tasks.md" << 'EOF'
## Phase 1: Setup
- [x] Task 1
EOF

    run "${PROJECT_ROOT}/scripts/speckit-implement-validate.sh" 002

    [ "$status" -eq 0 ]
}

@test "quickstart: spec name resolution by partial match" {
    SPEC_DIR="$SPECKIT_SPECS_DIR/003-mastodon-integration"
    mkdir -p "$SPEC_DIR"

    cat > "$SPEC_DIR/tasks.md" << 'EOF'
## Phase 1: Setup
- [x] Task 1
EOF

    run "${PROJECT_ROOT}/scripts/speckit-implement-validate.sh" mastodon

    [ "$status" -eq 0 ]
}
