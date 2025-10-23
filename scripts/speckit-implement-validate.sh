#!/bin/bash
# SpecKit Implementation Validation Script
# Executes /speckit.implement with validation and retry logic

set -euo pipefail

# Source validation library
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib/speckit-validation-lib.sh"

# Configuration
RETRY_LIMIT="${SPECKIT_RETRY_LIMIT:-2}"
DRY_RUN="${SPECKIT_DRY_RUN:-false}"
OUTPUT_JSON=false
OUTPUT_CONTINUATION=false
RESET_RETRY=false
PROMPT_SUFFIX=""

# Find git root directory
GIT_ROOT=$(git rev-parse --show-toplevel 2>/dev/null)
if [ -z "$GIT_ROOT" ]; then
    log_error "Not in a git repository"
    exit "$EXIT_MISSING_DEPS"
fi

# Change to git root
cd "$GIT_ROOT" || exit "$EXIT_VALIDATION_FAILED"

# ------------------------------------------------------------------------------
# Argument Parsing
# ------------------------------------------------------------------------------

show_help() {
    cat <<EOF
Usage: $0 [spec-name] [options]

Arguments:
  spec-name             Name of the spec (optional, auto-detected from git branch)
                        (e.g., "my-feature" or "002-my-feature")

Options:
  --retry-limit N       Maximum retry attempts (default: 2)
  --prompt "text"       Additional instructions to append to /speckit.implement
  --dry-run            Show what would be executed without running
  --verbose            Enable detailed logging
  --json               Output results as JSON (validation-only mode)
  --continuation       Generate continuation prompt only (validation-only mode)
  --reset-retry        Reset retry counter
  --help               Show this help message

Environment Variables:
  SPECKIT_RETRY_LIMIT   Override default retry limit
  SPECKIT_SPECS_DIR     Override specs directory location
  SPECKIT_DEBUG         Enable verbose logging
  SPECKIT_DRY_RUN       Set to "true" for dry-run mode
  ANTHROPIC_API_KEY     API key for Claude (can be empty for local auth)

Examples:
  # Execute implementation with validation and retry (auto-detect spec)
  $0

  # Execute for specific spec
  $0 my-feature

  # Add custom instructions to the implementation
  $0 --prompt "focus on error handling in phase 2"

  # Execute specific spec with custom instructions
  $0 my-feature --prompt "implement API tests thoroughly"

  # Dry run to see what would be executed
  $0 --dry-run

  # Validation-only mode (no execution)
  $0 --json

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
        --prompt)
            PROMPT_SUFFIX="$2"
            shift 2
            ;;
        --dry-run)
            DRY_RUN=true
            shift
            ;;
        --verbose)
            SPECKIT_DEBUG=true
            export SPECKIT_DEBUG
            shift
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

# Auto-detect spec if not provided
if [ -z "$SPEC_NAME" ]; then
    log_debug "No spec provided, attempting auto-detection..."
    DETECTED_SPEC=$(detect_current_spec)
    if [ -n "$DETECTED_SPEC" ]; then
        log_info "Detected active spec: $DETECTED_SPEC"
        SPEC_NAME="$DETECTED_SPEC"
    else
        log_error "Could not auto-detect spec. Please provide spec name or ensure you're on a feature branch."
        show_help
        exit "$EXIT_INVALID_ARGS"
    fi
fi

# Check dependencies (add claude if not in validation-only mode)
if [ "$OUTPUT_JSON" = "false" ] && [ "$OUTPUT_CONTINUATION" = "false" ]; then
    check_dependencies git claude jq grep sed awk || exit "$EXIT_MISSING_DEPS"
else
    check_dependencies jq grep sed awk || exit "$EXIT_MISSING_DEPS"
fi

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

log_debug "Using spec: $SPEC_NAME"
log_debug "Spec directory: $SPEC_DIR"

TASKS_FILE="$SPEC_DIR/tasks.md"

if [ ! -f "$TASKS_FILE" ]; then
    log_error "tasks.md not found in $SPEC_DIR"
    exit "$EXIT_MISSING_DEPS"
fi

log_debug "Found spec directory: $SPEC_DIR"
log_debug "Tasks file: $TASKS_FILE"

# Get spec title from tasks.md if available
SPEC_TITLE=$(basename "$SPEC_DIR")
if [ -f "$TASKS_FILE" ]; then
    # Try to extract title from first heading
    EXTRACTED_TITLE=$(head -5 "$TASKS_FILE" | grep '^# ' | sed 's/^# //' | head -1)
    if [ -n "$EXTRACTED_TITLE" ]; then
        SPEC_TITLE="$EXTRACTED_TITLE"
    fi
fi

# ------------------------------------------------------------------------------
# Helper Functions
# ------------------------------------------------------------------------------

# Run /speckit.implement with validation and retry logic
run_implement_with_validation() {
    local spec_name="$1"
    local spec_title="$2"
    local tasks_file="$3"

    log_debug "Running /speckit.implement for: $spec_title"
    log_debug "Tasks file: $tasks_file"

    # Dry run mode - check early to avoid retry limit checks
    if [ "$DRY_RUN" = "true" ]; then
        local total_unchecked
        total_unchecked=$(count_unchecked_tasks "$tasks_file")
        local prompt_cmd="/speckit.implement"
        if [ -n "$PROMPT_SUFFIX" ]; then
            prompt_cmd="/speckit.implement $PROMPT_SUFFIX"
        fi
        echo "[DRY RUN] Would execute: $prompt_cmd"
        echo "[DRY RUN] Current tasks remaining: $total_unchecked"
        echo "[DRY RUN] Would validate: all tasks checked in $tasks_file"
        return "$EXIT_SUCCESS"
    fi

    # IDEMPOTENCY CHECK: Skip if all tasks already complete
    local total_unchecked
    total_unchecked=$(count_unchecked_tasks "$tasks_file")

    if [ "$total_unchecked" -eq 0 ]; then
        log_info "✓ All tasks already complete, skipping execution"
        reset_retry_count "$spec_name" "implement"
        return "$EXIT_SUCCESS"
    fi

    # Get current retry count
    local retry_count
    retry_count=$(get_retry_count "$spec_name" "implement")

    # Check if retry limit exceeded
    if [ "$retry_count" -ge "$RETRY_LIMIT" ]; then
        log_error "Retry limit ($RETRY_LIMIT) exceeded for implementation"
        log_error "Cannot proceed with implementation"
        return "$EXIT_RETRY_EXHAUSTED"
    fi

    # Execute Claude command
    log_info "Executing /speckit.implement for '$spec_title'..."
    log_info "Tasks remaining: $total_unchecked"

    # Build the prompt command with optional suffix
    local prompt_cmd="/speckit.implement"
    if [ -n "$PROMPT_SUFFIX" ]; then
        prompt_cmd="/speckit.implement $PROMPT_SUFFIX"
        log_info "Additional instructions: $PROMPT_SUFFIX"
    fi

    if ! ANTHROPIC_API_KEY="" claude -p "$prompt_cmd" \
        --dangerously-skip-permissions \
        --verbose \
        --output-format stream-json | claude-clean; then
        log_error "Claude command failed: /speckit.implement"
        return "$EXIT_VALIDATION_FAILED"
    fi

    # Re-count unchecked tasks after execution
    total_unchecked=$(count_unchecked_tasks "$tasks_file")

    log_debug "Tasks remaining after execution: $total_unchecked"

    # Validate all tasks are complete
    if [ "$total_unchecked" -eq 0 ]; then
        log_info "✓ Implementation complete: all tasks checked"
        reset_retry_count "$spec_name" "implement"
        return "$EXIT_SUCCESS"
    else
        log_error "✗ Implementation incomplete: $total_unchecked tasks remaining (attempt $((retry_count + 1))/$RETRY_LIMIT)"

        # Increment retry counter
        local new_count
        new_count=$(increment_retry_count "$spec_name" "implement")

        if [ "$new_count" -ge "$RETRY_LIMIT" ]; then
            log_error "Retry limit reached. Aborting implementation."
            return "$EXIT_RETRY_EXHAUSTED"
        fi

        log_info "Retrying /speckit.implement..."
        # Recursive retry
        run_implement_with_validation "$spec_name" "$spec_title" "$tasks_file"
        return $?
    fi
}

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
# Main Execution
# ------------------------------------------------------------------------------

# Determine mode: execution or validation-only
VALIDATION_ONLY=false
if [ "$OUTPUT_JSON" = "true" ] || [ "$OUTPUT_CONTINUATION" = "true" ]; then
    VALIDATION_ONLY=true
fi

if [ "$VALIDATION_ONLY" = "false" ]; then
    # EXECUTION MODE: Run /speckit.implement with validation and retry
    log_info "Running /speckit.implement with validation..."
    log_info "Spec: $(basename "$SPEC_DIR")"
    log_info "Retry limit: $RETRY_LIMIT"
    echo ""

    # Reset retry count for fresh execution (manual invocation)
    # Retries are tracked within a single execution session, not across manual runs
    reset_retry_count "$SPEC_NAME" "implement"

    if ! run_implement_with_validation "$SPEC_NAME" "$SPEC_TITLE" "$TASKS_FILE"; then
        EXIT_CODE=$?
        if [ "$EXIT_CODE" -eq "$EXIT_RETRY_EXHAUSTED" ]; then
            log_error "Implementation failed after $RETRY_LIMIT attempts"
            log_error "Remaining tasks must be completed manually"
        fi
        exit "$EXIT_CODE"
    fi

    # Success
    log_info "✓ Implementation completed successfully!"
    exit "$EXIT_SUCCESS"
fi

# ------------------------------------------------------------------------------
# VALIDATION-ONLY MODE
# Phase Analysis and Reporting
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
