#!/bin/bash
# mock-claude.sh - Simulates Claude CLI behavior for testing
#
# This script provides a test double for the Claude CLI that:
# - NEVER makes network calls or runs actual claude
# - Returns configurable responses
# - Logs all calls for verification
# - Simulates delays and failures
#
# Environment Variables:
#   MOCK_RESPONSE_FILE - Path to file containing response to return (optional)
#   MOCK_CALL_LOG      - Path to log file for recording calls (optional)
#   MOCK_EXIT_CODE     - Exit code to return (default: 0)
#   MOCK_DELAY         - Seconds to delay before responding (default: 0)
#
# Usage:
#   export MOCK_RESPONSE_FILE=/tmp/response.yaml
#   export MOCK_CALL_LOG=/tmp/calls.log
#   ./mock-claude.sh --print "Generate a spec"

set -euo pipefail

# Defaults
EXIT_CODE="${MOCK_EXIT_CODE:-0}"
DELAY="${MOCK_DELAY:-0}"

# Log the call if MOCK_CALL_LOG is set
log_call() {
    if [[ -n "${MOCK_CALL_LOG:-}" ]]; then
        local timestamp
        timestamp=$(date -Iseconds 2>/dev/null || date +%Y-%m-%dT%H:%M:%S)
        {
            echo "---"
            echo "timestamp: \"${timestamp}\""
            echo "args:"
            for arg in "$@"; do
                # Escape special characters for YAML
                local escaped_arg
                escaped_arg=$(echo "$arg" | sed 's/"/\\"/g')
                echo "  - \"${escaped_arg}\""
            done
            echo "pid: $$"
            echo "response_file: \"${MOCK_RESPONSE_FILE:-}\""
            echo "exit_code: ${EXIT_CODE}"
            echo "delay: ${DELAY}"
        } >> "${MOCK_CALL_LOG}"
    fi
}

# Output response
output_response() {
    if [[ -n "${MOCK_RESPONSE_FILE:-}" && -f "${MOCK_RESPONSE_FILE}" ]]; then
        cat "${MOCK_RESPONSE_FILE}"
    fi
}

# Main execution
main() {
    # Log the call with all arguments
    log_call "$@"

    # Apply delay if configured (for timeout testing)
    if [[ "${DELAY}" -gt 0 ]]; then
        sleep "${DELAY}"
    fi

    # Output configured response
    output_response

    # Exit with configured code
    exit "${EXIT_CODE}"
}

# Run main with all arguments
main "$@"
