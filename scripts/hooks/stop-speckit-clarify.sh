#!/bin/bash
# Stop Hook for /speckit.clarify validation
# Validates that spec.md was updated with clarifications

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

SPEC_FILE="$SPEC_DIR/spec.md"

# Check if spec.md exists
if [ ! -f "$SPEC_FILE" ]; then
    log_debug "spec.md not found, allowing stop"
    allow_stop "No spec.md found"
    exit "$EXIT_SUCCESS"
fi

# Check if spec.md was modified recently (within last 5 minutes)
if [ -n "$(find "$SPEC_FILE" -mmin -5 2>/dev/null)" ]; then
    reset_retry_count "$CURRENT_SPEC" "clarify"
    allow_stop "Specification updated with clarifications"
    exit "$EXIT_SUCCESS"
fi

# Get retry count
RETRY_COUNT=$(get_retry_count "$CURRENT_SPEC" "clarify")

if [ "$RETRY_COUNT" -ge "$SPECKIT_RETRY_LIMIT" ]; then
    log_error "Retry limit exceeded for spec clarification"
    allow_stop "Retry limit exceeded, manual clarification required"
    exit "$EXIT_SUCCESS"
fi

# Block stop - spec.md wasn't updated
increment_retry_count "$CURRENT_SPEC" "clarify" > /dev/null

block_stop_with_reason \
    "Clarification not complete: spec.md was not updated. Please continue with /speckit.clarify." \
    "{\"spec_name\": \"$CURRENT_SPEC\", \"retry_count\": $RETRY_COUNT}"

exit "$EXIT_RETRY_EXHAUSTED"
