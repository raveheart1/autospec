#!/bin/bash
# Stop Hook for /speckit.tasks validation
# Validates that tasks.md was created

set -euo pipefail

# Source validation library
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../lib/speckit-validation-lib.sh"

# Read hook payload from stdin (reserved for future use)
# PAYLOAD=$(cat)

log_debug "Stop hook activated"

# Detect current spec
CURRENT_SPEC=$(detect_current_spec)

if [ -z "$CURRENT_SPEC" ]; then
    log_debug "No active spec detected, allowing stop"
    allow_stop "No active spec work detected"
    exit "$EXIT_SUCCESS"
fi

log_debug "Current spec: $CURRENT_SPEC"

# Find spec directory
SPEC_DIR=$(find "$SPECKIT_SPECS_DIR" -maxdepth 1 -type d -name "*${CURRENT_SPEC}*" | head -1)

if [ -z "$SPEC_DIR" ]; then
    log_debug "Spec directory not found, allowing stop"
    allow_stop "Spec directory not found"
    exit "$EXIT_SUCCESS"
fi

# Check if tasks.md exists
if ! validate_file_exists "tasks.md" "$SPEC_DIR"; then
    # Get retry count
    RETRY_COUNT=$(get_retry_count "$CURRENT_SPEC" "tasks")

    if [ "$RETRY_COUNT" -ge "$SPECKIT_RETRY_LIMIT" ]; then
        log_error "Retry limit exceeded for tasks.md creation"
        allow_stop "Retry limit exceeded, manual intervention required"
        exit "$EXIT_SUCCESS"
    fi

    # Block stop and request retry
    increment_retry_count "$CURRENT_SPEC" "tasks" > /dev/null

    block_stop_with_reason \
        "Task breakdown not complete: tasks.md missing. Please continue with /speckit.tasks." \
        "{\"spec_name\": \"$CURRENT_SPEC\", \"retry_count\": $RETRY_COUNT}"

    exit "$EXIT_RETRY_EXHAUSTED"
fi

# Success - tasks.md exists
reset_retry_count "$CURRENT_SPEC" "tasks"
allow_stop "Task breakdown complete: tasks.md exists"
exit "$EXIT_SUCCESS"
