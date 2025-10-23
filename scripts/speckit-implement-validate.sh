#!/bin/bash
# SpecKit Implementation Validation Script
# Validates /speckit.implement completion by checking tasks.md for incomplete phases

set -euo pipefail

# Source validation library
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib/speckit-validation-lib.sh"

# Configuration
RETRY_LIMIT="${SPECKIT_RETRY_LIMIT:-2}"
OUTPUT_JSON=false
OUTPUT_CONTINUATION=false
RESET_RETRY=false

# ------------------------------------------------------------------------------
# Argument Parsing
# ------------------------------------------------------------------------------

show_help() {
    cat <<EOF
Usage: $0 <spec-name> [options]

Arguments:
  spec-name             Name of the spec (e.g., "my-feature" or "002-my-feature")

Options:
  --retry-limit N       Maximum retry attempts (default: 2)
  --json               Output results as JSON
  --continuation       Generate continuation prompt for Claude
  --reset-retry        Reset retry counter
  --help               Show this help message

Environment Variables:
  SPECKIT_RETRY_LIMIT   Override default retry limit
  SPECKIT_SPECS_DIR     Override specs directory location
  SPECKIT_DEBUG         Enable verbose logging

Examples:
  # Validate implementation completion
  $0 my-feature

  # Output as JSON for programmatic use
  $0 002-my-feature --json

  # Generate continuation prompt
  $0 my-feature --continuation

Exit Codes:
  0 - All phases complete
  1 - Incomplete phases, can retry
  2 - Incomplete phases, retry limit exceeded
  3 - Invalid arguments
  4 - Missing dependencies or spec not found
EOF
}

# Parse arguments
SPEC_NAME=""

while [[ $# -gt 0 ]]; do
    case "$1" in
        --help|-h)
            show_help
            exit 0
            ;;
        --retry-limit)
            RETRY_LIMIT="$2"
            shift 2
            ;;
        --json)
            OUTPUT_JSON=true
            shift
            ;;
        --continuation)
            OUTPUT_CONTINUATION=true
            shift
            ;;
        --reset-retry)
            RESET_RETRY=true
            shift
            ;;
        --*)
            log_error "Unknown option: $1"
            show_help
            exit "$EXIT_INVALID_ARGS"
            ;;
        *)
            if [ -z "$SPEC_NAME" ]; then
                SPEC_NAME="$1"
            else
                log_error "Too many arguments"
                show_help
                exit "$EXIT_INVALID_ARGS"
            fi
            shift
            ;;
    esac
done

# Validate required arguments
if [ -z "$SPEC_NAME" ]; then
    log_error "Spec name required"
    show_help
    exit "$EXIT_INVALID_ARGS"
fi

# Check dependencies
check_dependencies jq grep sed awk || exit "$EXIT_MISSING_DEPS"

# ------------------------------------------------------------------------------
# Find Spec Directory
# ------------------------------------------------------------------------------

# Find spec directory (may have number prefix like 002-)
SPEC_DIR=$(find "$SPECKIT_SPECS_DIR" -maxdepth 1 -type d -name "*${SPEC_NAME}*" | head -1)

if [ -z "$SPEC_DIR" ]; then
    log_error "Spec not found: $SPEC_NAME"
    log_error "Looked in: $SPECKIT_SPECS_DIR"
    exit "$EXIT_MISSING_DEPS"
fi

TASKS_FILE="$SPEC_DIR/tasks.md"

if [ ! -f "$TASKS_FILE" ]; then
    log_error "tasks.md not found in $SPEC_DIR"
    exit "$EXIT_MISSING_DEPS"
fi

log_debug "Found spec directory: $SPEC_DIR"
log_debug "Tasks file: $TASKS_FILE"

# ------------------------------------------------------------------------------
# Retry State Management
# ------------------------------------------------------------------------------

# Reset retry if requested
if [ "$RESET_RETRY" = "true" ]; then
    reset_retry_count "$SPEC_NAME" "implement"
    log_info "Retry counter reset for $SPEC_NAME"
fi

# Get current retry count
RETRY_COUNT=$(get_retry_count "$SPEC_NAME" "implement")

log_debug "Current retry count: $RETRY_COUNT"
log_debug "Retry limit: $RETRY_LIMIT"

# ------------------------------------------------------------------------------
# Phase Analysis
# ------------------------------------------------------------------------------

# Get all incomplete phases
INCOMPLETE_PHASES=$(list_incomplete_phases "$TASKS_FILE")

log_debug "Incomplete phases: $INCOMPLETE_PHASES"

# Count total unchecked tasks
TOTAL_UNCHECKED=$(count_unchecked_tasks "$TASKS_FILE")

log_debug "Total unchecked tasks: $TOTAL_UNCHECKED"

# ------------------------------------------------------------------------------
# Output Generation
# ------------------------------------------------------------------------------

if [ "$TOTAL_UNCHECKED" -eq 0 ]; then
    # All phases complete
    if [ "$OUTPUT_JSON" = "true" ]; then
        cat <<EOF
{
  "spec_name": "$(basename "$SPEC_DIR")",
  "status": "complete",
  "retry_count": $RETRY_COUNT,
  "retry_limit": $RETRY_LIMIT,
  "can_retry": false,
  "phases": [],
  "message": "All implementation phases complete"
}
EOF
    else
        log_info "Validating implementation: $(basename "$SPEC_DIR")"
        log_info ""
        log_info "✓ All phases complete"
        log_info "Status: COMPLETE"
    fi

    # Clean up retry state on success
    reset_retry_count "$SPEC_NAME" "implement"

    exit "$EXIT_SUCCESS"
else
    # Incomplete phases detected
    CAN_RETRY="true"
    EXIT_CODE="$EXIT_VALIDATION_FAILED"

    # Check if retry limit exceeded
    if [ "$RETRY_COUNT" -ge "$RETRY_LIMIT" ]; then
        CAN_RETRY="false"
        EXIT_CODE="$EXIT_RETRY_EXHAUSTED"
    fi

    # Generate continuation prompt if requested or if showing text output
    CONTINUATION_PROMPT=""
    if [ "$OUTPUT_CONTINUATION" = "true" ] || [ "$OUTPUT_JSON" = "false" ]; then
        CONTINUATION_PROMPT=$(generate_continuation_prompt "$SPEC_NAME" "$TASKS_FILE")
    fi

    if [ "$OUTPUT_JSON" = "true" ]; then
        # JSON output
        echo "{"
        echo "  \"spec_name\": \"$(basename "$SPEC_DIR")\","
        echo "  \"status\": \"incomplete\","
        echo "  \"retry_count\": $RETRY_COUNT,"
        echo "  \"retry_limit\": $RETRY_LIMIT,"
        echo "  \"can_retry\": $CAN_RETRY,"
        echo "  \"phases\": ["

        FIRST=true
        for phase in $INCOMPLETE_PHASES; do
            if [ "$FIRST" = "false" ]; then
                echo ","
            fi
            FIRST=false

            PHASE_JSON=$(extract_phase_status "$TASKS_FILE" "$phase")
            # Indent each line with 4 spaces
            echo "$PHASE_JSON" | while IFS= read -r line; do
                echo "    $line"
            done
        done

        echo ""
        echo "  ],"
        echo "  \"continuation_prompt\": $(echo "$CONTINUATION_PROMPT" | jq -Rs .)"
        echo "}"

    elif [ "$OUTPUT_CONTINUATION" = "true" ]; then
        # Output only continuation prompt
        echo "$CONTINUATION_PROMPT"

    else
        # Text output with phase analysis
        log_info "Validating implementation: $(basename "$SPEC_DIR")"
        if [ "$RETRY_COUNT" -gt 0 ]; then
            log_info "Retry attempt: $RETRY_COUNT/$RETRY_LIMIT"
        fi
        echo ""

        log_info "Phase Analysis:"

        # Show all phases (complete and incomplete)
        ALL_PHASES=$(grep -E '^## Phase [0-9]+:' "$TASKS_FILE" | sed -n 's/^## Phase \([0-9]*\):.*/\1/p')

        for phase in $ALL_PHASES; do
            PHASE_STATUS=$(extract_phase_status "$TASKS_FILE" "$phase")

            IS_COMPLETE=$(echo "$PHASE_STATUS" | jq -r '.is_complete')
            PHASE_NAME=$(echo "$PHASE_STATUS" | jq -r '.phase_name')
            COMPLETED=$(echo "$PHASE_STATUS" | jq -r '.completed_tasks')
            TOTAL=$(echo "$PHASE_STATUS" | jq -r '.total_tasks')

            if [ "$IS_COMPLETE" = "true" ]; then
                log_info "✓ Phase $phase: $PHASE_NAME ($COMPLETED/$TOTAL tasks complete)"
            else
                log_info "✗ Phase $phase: $PHASE_NAME ($COMPLETED/$TOTAL tasks complete)"

                # Show unchecked tasks for incomplete phases
                awk -v phase="$phase" '
                    /^## Phase [0-9]+/ {
                        current_phase = substr($3, 1, length($3)-1)
                        in_phase = (current_phase == phase)
                    }
                    /^##[^#]/ && !/^## Phase/ {
                        in_phase = 0
                    }
                    in_phase && /^\s*- \[ \]/ {
                        print "  " $0
                    }
                ' "$TASKS_FILE"
            fi
        done

        echo ""

        if [ "$CAN_RETRY" = "true" ]; then
            log_info "Status: INCOMPLETE (can retry)"
            log_info "Recommendation: Continue with remaining phases"
        else
            log_error "Status: INCOMPLETE (retry limit exceeded)"
            log_error "Please complete the remaining tasks manually"
        fi

        echo ""
        echo "--- Continuation Prompt ---"
        echo "$CONTINUATION_PROMPT"
    fi

    # Increment retry counter for next run
    if [ "$CAN_RETRY" = "true" ] && [ "$OUTPUT_CONTINUATION" = "false" ]; then
        increment_retry_count "$SPEC_NAME" "implement" > /dev/null
    fi

    exit "$EXIT_CODE"
fi
