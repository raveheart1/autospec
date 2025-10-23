#!/bin/bash
# SpecKit Validation Library
# Shared functions for validation scripts and hooks

# Exit codes
readonly EXIT_SUCCESS=0
readonly EXIT_VALIDATION_FAILED=1
readonly EXIT_RETRY_EXHAUSTED=2
readonly EXIT_INVALID_ARGS=3
readonly EXIT_MISSING_DEPS=4

# Default configuration
SPECKIT_RETRY_LIMIT="${SPECKIT_RETRY_LIMIT:-2}"
SPECKIT_SPECS_DIR="${SPECKIT_SPECS_DIR:-./specs}"
SPECKIT_DEBUG="${SPECKIT_DEBUG:-false}"
SPECKIT_DRY_RUN="${SPECKIT_DRY_RUN:-false}"
SPECKIT_VALIDATION_TIMEOUT="${SPECKIT_VALIDATION_TIMEOUT:-5}"

# Retry state directory
readonly RETRY_STATE_DIR="/tmp"

# ------------------------------------------------------------------------------
# Logging and Debugging
# ------------------------------------------------------------------------------

# Log debug message to stderr if debug mode enabled
# Usage: log_debug "message"
log_debug() {
    if [ "$SPECKIT_DEBUG" = "true" ]; then
        echo "[DEBUG] $*" >&2
    fi
}

# Log error message to stderr
# Usage: log_error "message"
log_error() {
    echo "[ERROR] $*" >&2
}

# Log info message to stdout
# Usage: log_info "message"
log_info() {
    echo "$*"
}

# ------------------------------------------------------------------------------
# Dependency Checking
# ------------------------------------------------------------------------------

# Check if required dependencies are available
# Usage: check_dependencies git jq grep
# Returns: 0 if all found, 4 if any missing
check_dependencies() {
    local missing=()

    for dep in "$@"; do
        if ! command -v "$dep" &> /dev/null; then
            missing+=("$dep")
        fi
    done

    if [ ${#missing[@]} -gt 0 ]; then
        log_error "Missing dependencies: ${missing[*]}"
        log_error "Please install: ${missing[*]}"
        return "$EXIT_MISSING_DEPS"
    fi

    log_debug "All dependencies available: $*"
    return "$EXIT_SUCCESS"
}

# ------------------------------------------------------------------------------
# File Validation
# ------------------------------------------------------------------------------

# Validate that a file exists in a directory
# Usage: validate_file_exists "spec.md" "specs/002-speckit-validation-hooks"
# Returns: 0 if exists, 1 if not
validate_file_exists() {
    local filename="$1"
    local directory="$2"

    log_debug "Checking for file: $filename in $directory"

    if [ -z "$filename" ] || [ -z "$directory" ]; then
        log_error "validate_file_exists: filename and directory required"
        return "$EXIT_VALIDATION_FAILED"
    fi

    # Handle glob patterns in directory
    local expanded_dir="$directory"

    if [ -f "$expanded_dir/$filename" ]; then
        log_debug "File found: $expanded_dir/$filename"
        return "$EXIT_SUCCESS"
    fi

    log_debug "File not found: $expanded_dir/$filename"
    return "$EXIT_VALIDATION_FAILED"
}

# ------------------------------------------------------------------------------
# Retry State Management
# ------------------------------------------------------------------------------

# Get current retry count for a spec+command
# Usage: get_retry_count "002-speckit-validation-hooks" "plan"
# Returns: Count (0 if file missing) via stdout
get_retry_count() {
    local spec_name="$1"
    local command="$2"
    local retry_file="${RETRY_STATE_DIR}/.speckit-retry-${spec_name}-${command}"

    if [ -f "$retry_file" ]; then
        cat "$retry_file"
    else
        echo "0"
    fi
}

# Increment retry count for a spec+command
# Usage: increment_retry_count "002-speckit-validation-hooks" "plan"
# Returns: New count via stdout
increment_retry_count() {
    local spec_name="$1"
    local command="$2"
    local retry_file="${RETRY_STATE_DIR}/.speckit-retry-${spec_name}-${command}"

    local current_count
    current_count=$(get_retry_count "$spec_name" "$command")

    local new_count=$((current_count + 1))
    echo "$new_count" > "$retry_file"

    log_debug "Retry count for $spec_name/$command: $new_count"
    echo "$new_count"
}

# Reset retry count for a spec+command
# Usage: reset_retry_count "002-speckit-validation-hooks" "plan"
# Returns: 0
reset_retry_count() {
    local spec_name="$1"
    local command="$2"
    local retry_file="${RETRY_STATE_DIR}/.speckit-retry-${spec_name}-${command}"

    if [ -f "$retry_file" ]; then
        rm -f "$retry_file"
        log_debug "Reset retry count for $spec_name/$command"
    fi

    return "$EXIT_SUCCESS"
}

# ------------------------------------------------------------------------------
# Spec Name Parsing
# ------------------------------------------------------------------------------

# Parse spec name from command string
# Usage: parse_spec_from_command "/speckit.plan (for spec: my-feature)"
# Returns: Spec name via stdout (e.g., "my-feature")
parse_spec_from_command() {
    local command_string="$1"

    # Extract spec name from "(for spec: NAME)" pattern
    if echo "$command_string" | grep -q "for spec:"; then
        echo "$command_string" | sed -n 's/.*for spec: *\([^)]*\).*/\1/p' | xargs
    else
        echo ""
    fi
}

# Detect current active spec from git branch or recent activity
# Usage: detect_current_spec
# Returns: Spec name via stdout, or empty string if none detected
detect_current_spec() {
    # Try to detect from git branch name (e.g., "002-speckit-validation-hooks")
    if git rev-parse --git-dir &> /dev/null; then
        local branch
        branch=$(git rev-parse --abbrev-ref HEAD 2>/dev/null)

        # Check if branch name matches spec pattern (NNN-name)
        if echo "$branch" | grep -qE '^[0-9]{3}-'; then
            log_debug "Detected spec from branch: $branch"
            echo "$branch"
            return "$EXIT_SUCCESS"
        fi
    fi

    # Try to find most recently modified tasks.md in specs/
    if [ -d "$SPECKIT_SPECS_DIR" ]; then
        local recent_spec
        recent_spec=$(find "$SPECKIT_SPECS_DIR" -name "tasks.md" -type f -printf '%T@ %p\n' 2>/dev/null | \
                      sort -rn | head -1 | awk '{print $2}' | xargs dirname | xargs basename)

        if [ -n "$recent_spec" ]; then
            log_debug "Detected spec from recent activity: $recent_spec"
            echo "$recent_spec"
            return "$EXIT_SUCCESS"
        fi
    fi

    log_debug "No active spec detected"
    echo ""
    return "$EXIT_VALIDATION_FAILED"
}

# ------------------------------------------------------------------------------
# Task Parsing
# ------------------------------------------------------------------------------

# Count unchecked tasks in tasks.md
# Usage: count_unchecked_tasks "specs/002-*/tasks.md" [phase_number]
# Returns: Count via stdout
count_unchecked_tasks() {
    local tasks_file="$1"
    local phase_number="${2:-}"

    if [ ! -f "$tasks_file" ]; then
        log_error "Tasks file not found: $tasks_file"
        echo "0"
        return "$EXIT_VALIDATION_FAILED"
    fi

    if [ -z "$phase_number" ]; then
        # Count all unchecked tasks
        grep -cE '^\s*- \[ \]' "$tasks_file" 2>/dev/null || echo "0"
    else
        # Count unchecked tasks in specific phase
        # Extract phase section and count unchecked tasks
        awk -v phase="$phase_number" '
            /^## Phase [0-9]+/ {
                current_phase = substr($3, 1, length($3)-1)  # Remove trailing colon
                in_phase = (current_phase == phase)
            }
            /^##[^#]/ && !/^## Phase/ {
                in_phase = 0
            }
            in_phase && /^\s*- \[ \]/ {
                count++
            }
            END {
                print count+0
            }
        ' "$tasks_file"
    fi
}

# Count completed tasks in tasks.md
# Usage: count_completed_tasks "specs/002-*/tasks.md" [phase_number]
# Returns: Count via stdout
count_completed_tasks() {
    local tasks_file="$1"
    local phase_number="${2:-}"

    if [ ! -f "$tasks_file" ]; then
        log_error "Tasks file not found: $tasks_file"
        echo "0"
        return "$EXIT_VALIDATION_FAILED"
    fi

    if [ -z "$phase_number" ]; then
        # Count all completed tasks
        grep -ciE '^\s*- \[[xX]\]' "$tasks_file" 2>/dev/null || echo "0"
    else
        # Count completed tasks in specific phase
        awk -v phase="$phase_number" '
            /^## Phase [0-9]+/ {
                current_phase = substr($3, 1, length($3)-1)  # Remove trailing colon
                in_phase = (current_phase == phase)
            }
            /^##[^#]/ && !/^## Phase/ {
                in_phase = 0
            }
            in_phase && /^\s*- \[[xX]\]/ {
                count++
            }
            END {
                print count+0
            }
        ' "$tasks_file"
    fi
}

# Extract phase status information
# Usage: extract_phase_status "tasks.md" 3
# Returns: JSON object with phase status via stdout
extract_phase_status() {
    local tasks_file="$1"
    local phase_number="$2"

    if [ ! -f "$tasks_file" ]; then
        log_error "Tasks file not found: $tasks_file"
        echo "{}"
        return "$EXIT_VALIDATION_FAILED"
    fi

    # Extract phase name
    local phase_header
    phase_header=$(grep -E "^## Phase ${phase_number}:" "$tasks_file" | head -1)
    local phase_name
    phase_name=$(echo "$phase_header" | sed -n "s/^## Phase ${phase_number}: *//p")

    # Count tasks
    local unchecked
    unchecked=$(count_unchecked_tasks "$tasks_file" "$phase_number")
    local completed
    completed=$(count_completed_tasks "$tasks_file" "$phase_number")
    local total=$((unchecked + completed))

    # Determine completion status
    local is_complete="false"
    if [ "$unchecked" -eq 0 ] && [ "$total" -gt 0 ]; then
        is_complete="true"
    fi

    # Build JSON
    cat <<EOF
{
  "phase_number": $phase_number,
  "phase_name": "$phase_name",
  "total_tasks": $total,
  "completed_tasks": $completed,
  "unchecked_tasks": $unchecked,
  "completion_percentage": $((total > 0 ? (completed * 100) / total : 100)),
  "is_complete": $is_complete
}
EOF
}

# List all incomplete phases
# Usage: list_incomplete_phases "tasks.md"
# Returns: Space-separated list of phase numbers via stdout
list_incomplete_phases() {
    local tasks_file="$1"

    if [ ! -f "$tasks_file" ]; then
        echo ""
        return "$EXIT_VALIDATION_FAILED"
    fi

    # Extract phase numbers
    local phases
    phases=$(grep -E '^## Phase [0-9]+:' "$tasks_file" | sed -n 's/^## Phase \([0-9]*\):.*/\1/p')

    local incomplete_phases=()

    for phase in $phases; do
        local unchecked
        unchecked=$(count_unchecked_tasks "$tasks_file" "$phase")
        if [ "$unchecked" -gt 0 ]; then
            incomplete_phases+=("$phase")
        fi
    done

    echo "${incomplete_phases[*]}"
}

# ------------------------------------------------------------------------------
# Continuation Prompt Generation
# ------------------------------------------------------------------------------

# Generate continuation prompt for Claude
# Usage: generate_continuation_prompt "002-speckit-validation-hooks" "specs/002-*/tasks.md"
# Returns: Formatted prompt via stdout
generate_continuation_prompt() {
    local spec_name="$1"
    local tasks_file="$2"

    if [ ! -f "$tasks_file" ]; then
        log_error "Tasks file not found: $tasks_file"
        echo ""
        return "$EXIT_VALIDATION_FAILED"
    fi

    local incomplete_phases
    incomplete_phases=$(list_incomplete_phases "$tasks_file")

    if [ -z "$incomplete_phases" ]; then
        echo "All phases complete for $spec_name"
        return "$EXIT_SUCCESS"
    fi

    echo "The implementation for $spec_name has incomplete phases:"
    echo ""

    for phase in $incomplete_phases; do
        local phase_status
        phase_status=$(extract_phase_status "$tasks_file" "$phase")
        local phase_name
        phase_name=$(echo "$phase_status" | jq -r '.phase_name')
        local unchecked
        unchecked=$(echo "$phase_status" | jq -r '.unchecked_tasks')
        local total
        total=$(echo "$phase_status" | jq -r '.total_tasks')

        echo "Phase $phase: $phase_name ($unchecked/$total tasks remaining)"

        # Extract unchecked task descriptions
        awk -v phase="$phase" '
            /^## Phase [0-9]+/ {
                current_phase = substr($3, 1, length($3)-1)
                in_phase = (current_phase == phase)
            }
            /^##[^#]/ && !/^## Phase/ {
                in_phase = 0
            }
            in_phase && /^\s*- \[ \]/ {
                print
            }
        ' "$tasks_file"
        echo ""
    done

    echo "Please continue with /speckit.implement to complete these phases."
}

# ------------------------------------------------------------------------------
# Settings Isolation (User Story 3)
# ------------------------------------------------------------------------------

# Create isolated settings file for workflow run
# Usage: create_isolated_settings "stop-speckit-implement.sh"
# Returns: Path to temp settings file via stdout
create_isolated_settings() {
    local hook_script="$1"

    if [ -z "$hook_script" ]; then
        log_error "Hook script name required"
        return "$EXIT_INVALID_ARGS"
    fi

    # Generate unique temp file
    local temp_settings
    temp_settings="${RETRY_STATE_DIR}/.speckit-settings-$$-$(date +%s).json"

    # Check if template exists
    local template=".claude/spec-workflow-settings.json"
    if [ ! -f "$template" ]; then
        log_error "Settings template not found: $template"
        return "$EXIT_VALIDATION_FAILED"
    fi

    # Copy template and inject hook
    cp "$template" "$temp_settings"

    # Replace hook placeholder with actual script path
    # Assuming template has: "Stop": ["{{HOOK_SCRIPT}}"]
    sed -i "s|{{HOOK_SCRIPT}}|./scripts/hooks/$hook_script|g" "$temp_settings"

    log_debug "Created isolated settings: $temp_settings"
    echo "$temp_settings"
}

# Cleanup isolated settings file
# Usage: cleanup_isolated_settings "/tmp/.speckit-settings-123.json"
# Returns: 0
cleanup_isolated_settings() {
    local settings_file="$1"

    if [ -f "$settings_file" ]; then
        rm -f "$settings_file"
        log_debug "Cleaned up isolated settings: $settings_file"
    fi

    return "$EXIT_SUCCESS"
}

# ------------------------------------------------------------------------------
# Hook Response Helpers
# ------------------------------------------------------------------------------

# Block Claude from stopping with a reason
# Usage: block_stop_with_reason "Implementation not complete" "incomplete_phases"
# Returns: Exit code 2 (block), outputs JSON to stdout
block_stop_with_reason() {
    local reason="$1"
    local details="${2:-}"

    cat <<EOF
{
  "status": "blocked",
  "reason": "$reason",
  "details": $details
}
EOF

    return "$EXIT_RETRY_EXHAUSTED"  # Exit code 2 blocks stop
}

# Allow Claude to stop
# Usage: allow_stop "All work complete"
# Returns: Exit code 0 (allow), outputs JSON to stdout
allow_stop() {
    local message="$1"

    cat <<EOF
{
  "status": "allowed",
  "message": "$message"
}
EOF

    return "$EXIT_SUCCESS"
}

# ------------------------------------------------------------------------------
# Library initialized
# ------------------------------------------------------------------------------

log_debug "SpecKit validation library loaded"
