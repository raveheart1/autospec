#!/usr/bin/env bats
# T087: Integration tests that run full workflow with hooks enabled

setup() {
    load "test_helper"

    TEST_TEMP_DIR="${BATS_TMPDIR}/integration-test-$$"
    mkdir -p "$TEST_TEMP_DIR"

    export SPECKIT_SPECS_DIR="$TEST_TEMP_DIR/specs"
    export SPECKIT_DEBUG="false"

    mkdir -p "$SPECKIT_SPECS_DIR"

    # Create mock git root
    export GIT_ROOT="$TEST_TEMP_DIR"
    cd "$TEST_TEMP_DIR"
}

teardown() {
    rm -rf "$TEST_TEMP_DIR"
    cleanup_retry_state
}

# ------------------------------------------------------------------------------
# End-to-End Workflow Tests
# ------------------------------------------------------------------------------

@test "integration: full workflow with validation in dry-run" {
    export SPECKIT_DRY_RUN="true"

    run "${PROJECT_ROOT}/scripts/speckit-workflow-validate.sh" \
        "Integration Test Feature" \
        --skip-constitution \
        --dry-run

    [ "$status" -eq 0 ]

    # Verify all steps were executed
    [[ "$output" =~ "Step 1/3" ]]
    [[ "$output" =~ "Step 2/3" ]]
    [[ "$output" =~ "Step 3/3" ]]
    [[ "$output" =~ "completed successfully" ]]
}

@test "integration: workflow with hooks configured" {
    skip "Requires actual Claude Code execution with hooks"

    # This test would:
    # 1. Configure hooks in isolated settings
    # 2. Run workflow
    # 3. Verify hooks execute at appropriate times
    # 4. Verify hooks can block/allow stop correctly
}

@test "integration: retry mechanism works across workflow" {
    skip "Requires mocking Claude to fail/succeed on demand"

    # This test would:
    # 1. Mock Claude to fail on first attempt
    # 2. Run workflow
    # 3. Verify retry counter increments
    # 4. Mock Claude to succeed on retry
    # 5. Verify retry counter resets
}

# ------------------------------------------------------------------------------
# Hook Integration Tests
# ------------------------------------------------------------------------------

@test "integration: stop hook blocks incomplete implementation" {
    # Create incomplete spec
    SPEC_DIR="$SPECKIT_SPECS_DIR/002-hook-test"
    mkdir -p "$SPEC_DIR"

    cat > "$SPEC_DIR/tasks.md" << 'EOF'
## Phase 1: Setup
- [x] Task 1

## Phase 2: Implementation
- [ ] Task 2
- [ ] Task 3
EOF

    detect_current_spec() { echo "002-hook-test"; }
    export -f detect_current_spec

    run "${PROJECT_ROOT}/scripts/hooks/stop-speckit-implement.sh" < /dev/null

    [ "$status" -eq 2 ]
}

@test "integration: stop hook allows complete implementation" {
    SPEC_DIR="$SPECKIT_SPECS_DIR/003-complete-spec"
    mkdir -p "$SPEC_DIR"

    cat > "$SPEC_DIR/tasks.md" << 'EOF'
## Phase 1: Setup
- [x] Task 1
- [x] Task 2
EOF

    detect_current_spec() { echo "003-complete-spec"; }
    export -f detect_current_spec

    run "${PROJECT_ROOT}/scripts/hooks/stop-speckit-implement.sh" < /dev/null

    [ "$status" -eq 0 ]
}

# ------------------------------------------------------------------------------
# Validation Library Integration
# ------------------------------------------------------------------------------

@test "integration: validation library functions work together" {
    SPEC_DIR="$SPECKIT_SPECS_DIR/lib-test"
    mkdir -p "$SPEC_DIR"

    cat > "$SPEC_DIR/tasks.md" << 'EOF'
## Phase 1: Setup
- [x] Task 1
- [ ] Task 2

## Phase 2: Implementation
- [ ] Task 3
- [ ] Task 4
EOF

    source "${PROJECT_ROOT}/scripts/lib/speckit-validation-lib.sh"

    # Test task counting
    unchecked=$(count_unchecked_tasks "$SPEC_DIR/tasks.md")
    [ "$unchecked" -eq 3 ]

    # Test phase extraction
    phase_1_status=$(extract_phase_status "$SPEC_DIR/tasks.md" 1)
    echo "$phase_1_status" | jq -e '.phase_number' > /dev/null

    # Test incomplete phases
    incomplete=$(list_incomplete_phases "$SPEC_DIR/tasks.md")
    [[ "$incomplete" =~ "1" ]]
    [[ "$incomplete" =~ "2" ]]

    # Test continuation prompt
    prompt=$(generate_continuation_prompt "lib-test" "$SPEC_DIR/tasks.md")
    [[ "$prompt" =~ "incomplete" ]]
}

# ------------------------------------------------------------------------------
# Retry State Management Integration
# ------------------------------------------------------------------------------

@test "integration: retry state persists across script invocations" {
    source "${PROJECT_ROOT}/scripts/lib/speckit-validation-lib.sh"

    # First invocation
    count1=$(increment_retry_count "integration-spec" "plan")
    [ "$count1" -eq 1 ]

    # Second invocation (simulating retry)
    count2=$(increment_retry_count "integration-spec" "plan")
    [ "$count2" -eq 2 ]

    # Verify get_retry_count returns correct value
    current=$(get_retry_count "integration-spec" "plan")
    [ "$current" -eq 2 ]

    # Reset and verify
    reset_retry_count "integration-spec" "plan"
    after_reset=$(get_retry_count "integration-spec" "plan")
    [ "$after_reset" -eq 0 ]
}

# ------------------------------------------------------------------------------
# Multi-Spec Support
# ------------------------------------------------------------------------------

@test "integration: multiple specs can be validated simultaneously" {
    source "${PROJECT_ROOT}/scripts/lib/speckit-validation-lib.sh"

    # Create multiple specs
    for i in 1 2 3; do
        SPEC_DIR="$SPECKIT_SPECS_DIR/spec-$i"
        mkdir -p "$SPEC_DIR"

        cat > "$SPEC_DIR/tasks.md" << EOF
## Phase 1: Setup
- [x] Task 1
- $( [ "$i" -eq 2 ] && echo "[ ] Task 2" || echo "[x] Task 2" )
EOF
    done

    # Validate each
    run "${PROJECT_ROOT}/scripts/speckit-implement-validate.sh" spec-1
    [ "$status" -eq 0 ]

    run "${PROJECT_ROOT}/scripts/speckit-implement-validate.sh" spec-2
    [ "$status" -eq 1 ]

    run "${PROJECT_ROOT}/scripts/speckit-implement-validate.sh" spec-3
    [ "$status" -eq 0 ]
}

# ------------------------------------------------------------------------------
# Performance Tests
# ------------------------------------------------------------------------------

@test "integration: validation completes quickly" {
    SPEC_DIR="$SPECKIT_SPECS_DIR/perf-test"
    mkdir -p "$SPEC_DIR"

    # Create large tasks.md file
    {
        for phase in {1..6}; do
            echo "## Phase $phase: Testing"
            for task in {1..20}; do
                echo "- [x] Task $task"
            done
        done
    } > "$SPEC_DIR/tasks.md"

    # Time the validation
    start=$(date +%s%N)
    "${PROJECT_ROOT}/scripts/speckit-implement-validate.sh" perf-test > /dev/null
    end=$(date +%s%N)

    # Calculate duration in milliseconds
    duration=$(( (end - start) / 1000000 ))

    # Should complete in less than 5000ms (5 seconds)
    [ "$duration" -lt 5000 ]
}

# ------------------------------------------------------------------------------
# Error Handling Integration
# ------------------------------------------------------------------------------

@test "integration: handles corrupted tasks.md gracefully" {
    SPEC_DIR="$SPECKIT_SPECS_DIR/corrupted-spec"
    mkdir -p "$SPEC_DIR"

    # Create malformed tasks.md
    echo "This is not valid task markdown" > "$SPEC_DIR/tasks.md"

    run "${PROJECT_ROOT}/scripts/speckit-implement-validate.sh" corrupted-spec

    # Should not crash, should handle gracefully
    # Exit code might be 0 (no unchecked tasks found) or 1
    [ "$status" -eq 0 ] || [ "$status" -eq 1 ]
}

@test "integration: handles missing dependencies" {
    # Test that check_dependencies works
    source "${PROJECT_ROOT}/scripts/lib/speckit-validation-lib.sh"

    run check_dependencies nonexistent_command_12345

    [ "$status" -eq 4 ]
    [[ "$output" =~ "Missing dependencies" ]]
}

# ------------------------------------------------------------------------------
# Concurrent Workflow Support
# ------------------------------------------------------------------------------

@test "integration: multiple workflows can run independently" {
    source "${PROJECT_ROOT}/scripts/lib/speckit-validation-lib.sh"

    # Simulate two concurrent workflows with different specs
    increment_retry_count "workflow-1" "plan" > /dev/null
    increment_retry_count "workflow-1" "plan" > /dev/null

    increment_retry_count "workflow-2" "plan" > /dev/null

    # Verify they maintain separate state
    count1=$(get_retry_count "workflow-1" "plan")
    count2=$(get_retry_count "workflow-2" "plan")

    [ "$count1" -eq 2 ]
    [ "$count2" -eq 1 ]
}
