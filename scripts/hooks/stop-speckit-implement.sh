#!/bin/bash
# Stop Hook for /speckit.implement validation
# Validates that all implementation phases are complete

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

TASKS_FILE="$SPEC_DIR/tasks.md"

# Check if tasks.md exists
if [ ! -f "$TASKS_FILE" ]; then
    log_debug "tasks.md not found, allowing stop"
    allow_stop "No tasks.md found"
    exit "$EXIT_SUCCESS"
fi

# Count unchecked tasks
TOTAL_UNCHECKED=$(count_unchecked_tasks "$TASKS_FILE")

log_debug "Total unchecked tasks: $TOTAL_UNCHECKED"

if [ "$TOTAL_UNCHECKED" -eq 0 ]; then
    # All phases complete
    reset_retry_count "$CURRENT_SPEC" "implement"
    allow_stop "All implementation phases complete"
    exit "$EXIT_SUCCESS"
fi

# Get retry count
RETRY_COUNT=$(get_retry_count "$CURRENT_SPEC" "implement")

if [ "$RETRY_COUNT" -ge "$SPECKIT_RETRY_LIMIT" ]; then
    log_error "Retry limit exceeded for implementation"
    allow_stop "Retry limit exceeded, please complete remaining tasks manually"
    exit "$EXIT_SUCCESS"
fi

# Generate continuation prompt
CONTINUATION=$(generate_continuation_prompt "$CURRENT_SPEC" "$TASKS_FILE")

# Get incomplete phases for details
INCOMPLETE_PHASES=$(list_incomplete_phases "$TASKS_FILE")

# Build phase details JSON
PHASE_DETAILS="["
FIRST=true
for phase in $INCOMPLETE_PHASES; do
    if [ "$FIRST" = "false" ]; then
        PHASE_DETAILS+=","
    fi
    FIRST=false

    PHASE_STATUS=$(extract_phase_status "$TASKS_FILE" "$phase")
    PHASE_DETAILS+="$PHASE_STATUS"
done
PHASE_DETAILS+="]"

# Increment retry counter
increment_retry_count "$CURRENT_SPEC" "implement" > /dev/null

# Block stop with details
cat <<EOF
{
  "status": "blocked",
  "reason": "Implementation not complete: $TOTAL_UNCHECKED tasks remaining",
  "details": {
    "spec_name": "$CURRENT_SPEC",
    "retry_count": $RETRY_COUNT,
    "incomplete_phases": $PHASE_DETAILS,
    "continuation_prompt": $(echo "$CONTINUATION" | jq -Rs .)
  }
}
EOF

exit "$EXIT_RETRY_EXHAUSTED"
