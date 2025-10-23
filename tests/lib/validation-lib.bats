#!/usr/bin/env bats
# Unit tests for speckit-validation-lib.sh

# Setup - runs before each test
setup() {
    # Load the validation library
    load "../test_helper"
    source "${BATS_TEST_DIRNAME}/../../scripts/lib/speckit-validation-lib.sh"

    # Create temp directory for test files
    TEST_TEMP_DIR="${BATS_TMPDIR}/speckit-test-$$"
    mkdir -p "$TEST_TEMP_DIR"

    # Set test environment variables
    export SPECKIT_SPECS_DIR="$TEST_TEMP_DIR/specs"
    export SPECKIT_DEBUG="false"
    mkdir -p "$SPECKIT_SPECS_DIR"
}

# Teardown - runs after each test
teardown() {
    # Clean up temp directory
    rm -rf "$TEST_TEMP_DIR"

    # Clean up any retry state files
    rm -f /tmp/.speckit-retry-*
}

# ------------------------------------------------------------------------------
# Dependency Checking Tests
# ------------------------------------------------------------------------------

@test "check_dependencies: all dependencies found" {
    run check_dependencies echo grep sed
    [ "$status" -eq 0 ]
}

@test "check_dependencies: missing dependency detected" {
    run check_dependencies nonexistent_command_xyz
    [ "$status" -eq 4 ]
    [[ "$output" =~ "Missing dependencies" ]]
}

# ------------------------------------------------------------------------------
# File Validation Tests
# ------------------------------------------------------------------------------

@test "validate_file_exists: returns 0 when file exists" {
    touch "$TEST_TEMP_DIR/test-file.md"

    run validate_file_exists "test-file.md" "$TEST_TEMP_DIR"
    [ "$status" -eq 0 ]
}

@test "validate_file_exists: returns 1 when file missing" {
    run validate_file_exists "missing-file.md" "$TEST_TEMP_DIR"
    [ "$status" -eq 1 ]
}

@test "validate_file_exists: handles missing directory" {
    run validate_file_exists "file.md" "$TEST_TEMP_DIR/nonexistent"
    [ "$status" -eq 1 ]
}

@test "validate_file_exists: requires both arguments" {
    run validate_file_exists "file.md"
    [ "$status" -eq 1 ]
}

# ------------------------------------------------------------------------------
# Retry State Management Tests
# ------------------------------------------------------------------------------

@test "get_retry_count: returns 0 for new spec" {
    result=$(get_retry_count "test-spec" "plan")
    [ "$result" -eq 0 ]
}

@test "increment_retry_count: increments from 0 to 1" {
    result=$(increment_retry_count "test-spec" "plan")
    [ "$result" -eq 1 ]
}

@test "increment_retry_count: increments from 1 to 2" {
    increment_retry_count "test-spec" "plan" > /dev/null
    result=$(increment_retry_count "test-spec" "plan")
    [ "$result" -eq 2 ]
}

@test "get_retry_count: returns correct count after increment" {
    increment_retry_count "test-spec" "plan" > /dev/null
    increment_retry_count "test-spec" "plan" > /dev/null
    result=$(get_retry_count "test-spec" "plan")
    [ "$result" -eq 2 ]
}

@test "reset_retry_count: removes retry file" {
    increment_retry_count "test-spec" "plan" > /dev/null
    reset_retry_count "test-spec" "plan"
    result=$(get_retry_count "test-spec" "plan")
    [ "$result" -eq 0 ]
}

@test "reset_retry_count: handles non-existent retry file" {
    run reset_retry_count "nonexistent-spec" "plan"
    [ "$status" -eq 0 ]
}

# ------------------------------------------------------------------------------
# Spec Name Parsing Tests
# ------------------------------------------------------------------------------

@test "parse_spec_from_command: extracts spec name from standard format" {
    result=$(parse_spec_from_command "/speckit.plan (for spec: my-feature)")
    [ "$result" = "my-feature" ]
}

@test "parse_spec_from_command: handles spec name with hyphens" {
    result=$(parse_spec_from_command "/speckit.specify (for spec: add-mastodon-integration)")
    [ "$result" = "add-mastodon-integration" ]
}

@test "parse_spec_from_command: handles spec name with spaces" {
    result=$(parse_spec_from_command "/speckit.tasks (for spec: Mastodon Integration)")
    [ "$result" = "Mastodon Integration" ]
}

@test "parse_spec_from_command: returns empty for invalid format" {
    result=$(parse_spec_from_command "/speckit.plan without spec name")
    [ "$result" = "" ]
}

# ------------------------------------------------------------------------------
# Task Counting Tests
# ------------------------------------------------------------------------------

@test "count_unchecked_tasks: counts all unchecked tasks" {
    cat > "$TEST_TEMP_DIR/tasks.md" << 'EOF'
## Phase 1: Setup
- [x] Task 1
- [ ] Task 2
- [ ] Task 3

## Phase 2: Implementation
- [x] Task 4
- [ ] Task 5
EOF

    result=$(count_unchecked_tasks "$TEST_TEMP_DIR/tasks.md")
    [ "$result" -eq 3 ]
}

@test "count_unchecked_tasks: returns 0 for all complete" {
    cat > "$TEST_TEMP_DIR/tasks.md" << 'EOF'
## Phase 1: Setup
- [x] Task 1
- [x] Task 2
EOF

    result=$(count_unchecked_tasks "$TEST_TEMP_DIR/tasks.md")
    [ "$result" -eq 0 ]
}

@test "count_unchecked_tasks: counts specific phase" {
    cat > "$TEST_TEMP_DIR/tasks.md" << 'EOF'
## Phase 1: Setup
- [x] Task 1
- [ ] Task 2

## Phase 2: Implementation
- [ ] Task 3
- [ ] Task 4
EOF

    result=$(count_unchecked_tasks "$TEST_TEMP_DIR/tasks.md" 2)
    [ "$result" -eq 2 ]
}

@test "count_completed_tasks: counts all completed tasks" {
    cat > "$TEST_TEMP_DIR/tasks.md" << 'EOF'
## Phase 1: Setup
- [x] Task 1
- [X] Task 2
- [ ] Task 3
EOF

    result=$(count_completed_tasks "$TEST_TEMP_DIR/tasks.md")
    [ "$result" -eq 2 ]
}

# ------------------------------------------------------------------------------
# Phase Status Extraction Tests
# ------------------------------------------------------------------------------

@test "extract_phase_status: returns valid JSON" {
    cat > "$TEST_TEMP_DIR/tasks.md" << 'EOF'
## Phase 1: Setup
- [x] Task 1
- [ ] Task 2
- [ ] Task 3
EOF

    result=$(extract_phase_status "$TEST_TEMP_DIR/tasks.md" 1)

    # Validate JSON structure
    echo "$result" | jq -e '.phase_number' > /dev/null
    echo "$result" | jq -e '.phase_name' > /dev/null
    echo "$result" | jq -e '.total_tasks' > /dev/null
}

@test "extract_phase_status: calculates completion correctly" {
    cat > "$TEST_TEMP_DIR/tasks.md" << 'EOF'
## Phase 1: Setup
- [x] Task 1
- [x] Task 2
- [ ] Task 3
EOF

    result=$(extract_phase_status "$TEST_TEMP_DIR/tasks.md" 1)

    completed=$(echo "$result" | jq -r '.completed_tasks')
    unchecked=$(echo "$result" | jq -r '.unchecked_tasks')

    [ "$completed" -eq 2 ]
    [ "$unchecked" -eq 1 ]
}

@test "extract_phase_status: marks phase complete when all tasks done" {
    cat > "$TEST_TEMP_DIR/tasks.md" << 'EOF'
## Phase 1: Setup
- [x] Task 1
- [x] Task 2
EOF

    result=$(extract_phase_status "$TEST_TEMP_DIR/tasks.md" 1)
    is_complete=$(echo "$result" | jq -r '.is_complete')

    [ "$is_complete" = "true" ]
}

# ------------------------------------------------------------------------------
# Incomplete Phases Tests
# ------------------------------------------------------------------------------

@test "list_incomplete_phases: finds all incomplete phases" {
    cat > "$TEST_TEMP_DIR/tasks.md" << 'EOF'
## Phase 1: Setup
- [x] Task 1

## Phase 2: Implementation
- [ ] Task 2

## Phase 3: Testing
- [x] Task 3
- [ ] Task 4
EOF

    result=$(list_incomplete_phases "$TEST_TEMP_DIR/tasks.md")

    # Should return "2 3"
    [[ "$result" =~ "2" ]]
    [[ "$result" =~ "3" ]]
}

@test "list_incomplete_phases: returns empty for all complete" {
    cat > "$TEST_TEMP_DIR/tasks.md" << 'EOF'
## Phase 1: Setup
- [x] Task 1

## Phase 2: Implementation
- [x] Task 2
EOF

    result=$(list_incomplete_phases "$TEST_TEMP_DIR/tasks.md")
    [ "$result" = "" ]
}

# ------------------------------------------------------------------------------
# Continuation Prompt Tests
# ------------------------------------------------------------------------------

@test "generate_continuation_prompt: generates prompt for incomplete phases" {
    cat > "$TEST_TEMP_DIR/tasks.md" << 'EOF'
## Phase 1: Setup
- [x] Task 1

## Phase 2: Implementation
- [ ] Task 2
- [ ] Task 3
EOF

    result=$(generate_continuation_prompt "test-spec" "$TEST_TEMP_DIR/tasks.md")

    [[ "$result" =~ "incomplete phases" ]]
    [[ "$result" =~ "Phase 2" ]]
}

@test "generate_continuation_prompt: returns complete message when all done" {
    cat > "$TEST_TEMP_DIR/tasks.md" << 'EOF'
## Phase 1: Setup
- [x] Task 1
EOF

    result=$(generate_continuation_prompt "test-spec" "$TEST_TEMP_DIR/tasks.md")

    [[ "$result" =~ "All phases complete" ]]
}

# ------------------------------------------------------------------------------
# Logging Tests
# ------------------------------------------------------------------------------

@test "log_debug: outputs when debug enabled" {
    export SPECKIT_DEBUG="true"

    run log_debug "Test message"
    [[ "$output" =~ "Test message" ]]
}

@test "log_debug: silent when debug disabled" {
    export SPECKIT_DEBUG="false"

    run log_debug "Test message"
    [ "$output" = "" ]
}

@test "log_error: always outputs to stderr" {
    run log_error "Error message"
    [[ "$output" =~ "Error message" ]]
}

# ------------------------------------------------------------------------------
# Hook Response Tests
# ------------------------------------------------------------------------------

@test "allow_stop: returns exit code 0" {
    run allow_stop "Test message"
    [ "$status" -eq 0 ]
}

@test "allow_stop: outputs valid JSON" {
    result=$(allow_stop "Test message")
    echo "$result" | jq -e '.status' > /dev/null
    echo "$result" | jq -e '.message' > /dev/null
}

@test "block_stop_with_reason: returns exit code 2" {
    run block_stop_with_reason "Test reason" "{}"
    [ "$status" -eq 2 ]
}

@test "block_stop_with_reason: outputs valid JSON" {
    result=$(block_stop_with_reason "Test reason" '{"key":"value"}')
    echo "$result" | jq -e '.status' > /dev/null
    echo "$result" | jq -e '.reason' > /dev/null
}

# ------------------------------------------------------------------------------
# Isolated Settings Tests (T078-T080)
# ------------------------------------------------------------------------------

@test "T078: create_isolated_settings creates temp file" {
    # Create settings template
    mkdir -p "$TEST_TEMP_DIR/.claude"
    cat > "$TEST_TEMP_DIR/.claude/spec-workflow-settings.json" << 'EOF'
{
  "hooks": {
    "Stop": ["{{HOOK_SCRIPT}}"]
  }
}
EOF

    cd "$TEST_TEMP_DIR"

    result=$(create_isolated_settings "stop-speckit-implement.sh")

    # Verify file exists
    [ -f "$result" ]

    # Verify it's in /tmp
    [[ "$result" =~ ^/tmp ]]
}

@test "T078: create_isolated_settings injects hook script" {
    mkdir -p "$TEST_TEMP_DIR/.claude"
    cat > "$TEST_TEMP_DIR/.claude/spec-workflow-settings.json" << 'EOF'
{
  "hooks": {
    "Stop": ["{{HOOK_SCRIPT}}"]
  }
}
EOF

    cd "$TEST_TEMP_DIR"

    result=$(create_isolated_settings "stop-speckit-implement.sh")

    # Verify hook was injected
    grep -q "stop-speckit-implement.sh" "$result"
}

@test "T079: cleanup_isolated_settings removes temp file" {
    temp_file="/tmp/.speckit-settings-test-$$"
    touch "$temp_file"

    cleanup_isolated_settings "$temp_file"

    [ ! -f "$temp_file" ]
}

@test "T079: cleanup_isolated_settings handles missing file" {
    run cleanup_isolated_settings "/tmp/nonexistent-file-$$"

    [ "$status" -eq 0 ]
}

@test "T080: normal settings file never modified during workflow" {
    # This is more of an integration test
    # Verify that .claude/settings.local.json is never touched

    mkdir -p "$TEST_TEMP_DIR/.claude"
    cat > "$TEST_TEMP_DIR/.claude/settings.local.json" << 'EOF'
{
  "original": "settings"
}
EOF

    original_content=$(cat "$TEST_TEMP_DIR/.claude/settings.local.json")

    # Create isolated settings (should not touch original)
    cat > "$TEST_TEMP_DIR/.claude/spec-workflow-settings.json" << 'EOF'
{
  "hooks": {
    "Stop": ["{{HOOK_SCRIPT}}"]
  }
}
EOF

    cd "$TEST_TEMP_DIR"
    temp_settings=$(create_isolated_settings "test-hook.sh")

    # Verify original is unchanged
    current_content=$(cat "$TEST_TEMP_DIR/.claude/settings.local.json")
    [ "$original_content" = "$current_content" ]

    # Cleanup
    cleanup_isolated_settings "$temp_settings"
}
