#!/usr/bin/env bash
# Common test helper functions for bats tests

# Get the project root directory
export PROJECT_ROOT="$(cd "${BATS_TEST_DIRNAME}/.." && pwd)"

# Helper to create a mock spec directory structure
create_mock_spec() {
    local spec_name="$1"
    local spec_dir="${BATS_TMPDIR}/specs/${spec_name}"

    mkdir -p "$spec_dir"
    echo "$spec_dir"
}

# Helper to create a mock tasks.md file
create_mock_tasks() {
    local tasks_file="$1"
    local content="${2:-complete}"

    if [ "$content" = "complete" ]; then
        cat > "$tasks_file" << 'EOF'
## Phase 1: Setup
- [x] Task 1
- [x] Task 2

## Phase 2: Implementation
- [x] Task 3
- [x] Task 4
EOF
    else
        cat > "$tasks_file" << 'EOF'
## Phase 1: Setup
- [x] Task 1
- [x] Task 2

## Phase 2: Implementation
- [x] Task 3
- [ ] Task 4
- [ ] Task 5
EOF
    fi
}

# Helper to clean up retry state
cleanup_retry_state() {
    rm -f /tmp/.speckit-retry-* 2>/dev/null || true
}
