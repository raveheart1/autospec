#!/usr/bin/env bash
# parse-claude-conversation.sh - Helper for analyzing Claude conversations
#
# Usage:
#   ./scripts/parse-claude-conversation.sh list [pattern]     # List conversations matching pattern
#   ./scripts/parse-claude-conversation.sh info <file>        # Show conversation metadata
#   ./scripts/parse-claude-conversation.sh summary <file>     # Show tool usage summary
#   ./scripts/parse-claude-conversation.sh issues <file>      # Detect common inefficiency patterns
#   ./scripts/parse-claude-conversation.sh parse <file> [lines] # Parse full conversation
#   ./scripts/parse-claude-conversation.sh unreviewed         # List unreviewed autospec conversations
#   ./scripts/parse-claude-conversation.sh mark <id> <cmd>    # Mark conversation as reviewed

set -euo pipefail

CLAUDE_PROJECTS_DIR="${HOME}/.claude/projects"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(dirname "$SCRIPT_DIR")"
REVIEWED_FILE="${REPO_ROOT}/.dev/feedback/reviewed.txt"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

usage() {
    cat <<EOF
Usage: $(basename "$0") <command> [args]

Commands:
  list [pattern]      List autospec project directories (filter by pattern)
  files <project>     List conversation files in a project directory
  info <file>         Show conversation metadata (date, command type, size)
  summary <file>      Show tool usage summary for a conversation
  issues <file>       Detect common inefficiency patterns
  parse <file> [n]    Parse conversation (optionally first n lines)
  unreviewed [proj]   List unreviewed autospec conversations
  status              Show review progress summary
  mark <id> <cmd>     Mark conversation as reviewed (e.g., mark 548be630 implement)

Examples:
  $(basename "$0") list autospec
  $(basename "$0") files -home-ari-repos-autospec
  $(basename "$0") info ~/.claude/projects/-home-ari-repos-autospec/548be630.jsonl
  $(basename "$0") summary /path/to/conversation.jsonl
  $(basename "$0") issues /path/to/conversation.jsonl
  $(basename "$0") parse /path/to/conversation.jsonl 500
  $(basename "$0") unreviewed
  $(basename "$0") mark 548be630 implement

Environment:
  CLAUDE_PROJECTS_DIR  Override default ~/.claude/projects
EOF
    exit 1
}

# List autospec-related project directories
cmd_list() {
    local pattern="${1:-autospec}"
    echo -e "${CYAN}Claude project directories matching '$pattern':${NC}"
    echo
    ls -d "${CLAUDE_PROJECTS_DIR}"/*"${pattern}"* 2>/dev/null | while read -r dir; do
        local count
        count=$(find "$dir" -name "*.jsonl" 2>/dev/null | wc -l)
        echo -e "  ${GREEN}$(basename "$dir")${NC} (${count} conversations)"
    done
}

# List conversation files in a project
cmd_files() {
    local project="$1"
    local project_dir="${CLAUDE_PROJECTS_DIR}/${project}"

    if [[ ! -d "$project_dir" ]]; then
        echo -e "${RED}Project directory not found: $project_dir${NC}" >&2
        exit 1
    fi

    echo -e "${CYAN}Conversations in ${project}:${NC}"
    echo -e "${CYAN}(sorted by modification time, most recent first)${NC}"
    echo

    ls -lt "${project_dir}"/*.jsonl 2>/dev/null | head -30 | while read -r line; do
        local file size date
        file=$(echo "$line" | awk '{print $NF}')
        size=$(echo "$line" | awk '{print $5}')
        date=$(echo "$line" | awk '{print $6, $7, $8}')

        # Try to detect autospec command at START of conversation (first 100 lines)
        # This distinguishes true autospec-triggered sessions from manual sessions
        # Note: autospec-triggered sessions have <command-name>/autospec.X</command-name> format
        local cmd_type="unknown"
        local first_lines
        first_lines=$(head -100 "$file" 2>/dev/null)
        if echo "$first_lines" | grep -q '<command-name>/autospec\.implement'; then
            cmd_type="implement"
        elif echo "$first_lines" | grep -q '<command-name>/autospec\.specify'; then
            cmd_type="specify"
        elif echo "$first_lines" | grep -q '<command-name>/autospec\.plan'; then
            cmd_type="plan"
        elif echo "$first_lines" | grep -q '<command-name>/autospec\.tasks'; then
            cmd_type="tasks"
        elif echo "$first_lines" | grep -q '<command-name>/autospec\.'; then
            cmd_type="autospec"
        elif echo "$first_lines" | grep -q 'prereqs --json'; then
            cmd_type="autospec"  # Legacy prereqs-only detection
        fi

        local basename
        basename=$(basename "$file" .jsonl)
        local short_id="${basename:0:8}"

        # Human readable size
        local hr_size
        if (( size > 1048576 )); then
            hr_size="$(( size / 1048576 ))M"
        elif (( size > 1024 )); then
            hr_size="$(( size / 1024 ))K"
        else
            hr_size="${size}B"
        fi

        echo -e "  ${short_id} ${date} ${hr_size}\t${YELLOW}${cmd_type}${NC}"
    done
}

# Show conversation metadata
cmd_info() {
    local file="$1"

    if [[ ! -f "$file" ]]; then
        echo -e "${RED}File not found: $file${NC}" >&2
        exit 1
    fi

    echo -e "${CYAN}Conversation Info:${NC}"
    echo -e "  File: $(basename "$file")"
    echo -e "  Size: $(du -h "$file" | cut -f1)"
    echo -e "  Lines: $(wc -l < "$file")"
    echo -e "  Modified: $(stat -c '%y' "$file" 2>/dev/null || stat -f '%Sm' "$file" 2>/dev/null)"

    # Detect command type - check if at START of conversation
    echo -e "\n${CYAN}Command Detection:${NC}"
    local autospec_cmd is_triggered
    local first_lines
    first_lines="$(head -100 "$file" 2>/dev/null)"
    autospec_cmd="$(echo "$first_lines" | grep -o '/autospec\.[a-z]*' | head -1)" || autospec_cmd="none"

    # Check if this is a true autospec-triggered session
    # Note: autospec-triggered sessions have <command-name>/autospec.X</command-name> format
    # Use head + grep directly on file to avoid variable expansion issues with large content
    if head -100 "$file" 2>/dev/null | grep -qE '(<command-name>/autospec\.|prereqs --json)'; then
        is_triggered="${GREEN}YES${NC} (command at conversation start)"
    else
        # Check if command appears later in file (manual session that mentions autospec)
        if grep -q '/autospec\.' "$file" 2>/dev/null; then
            is_triggered="${YELLOW}MAYBE${NC} (command found but not at start - likely manual session)"
            autospec_cmd=$(grep -o '/autospec\.[a-z]*' "$file" 2>/dev/null | head -1 || echo "none")
        else
            is_triggered="${RED}NO${NC} (no autospec command found)"
        fi
    fi
    echo -e "  Autospec command: ${YELLOW}${autospec_cmd}${NC}"
    echo -e "  Autospec-triggered: ${is_triggered}"

    # Count messages
    echo -e "\n${CYAN}Message Counts:${NC}"
    local user_count assistant_count tool_use_count tool_result_count
    user_count=$(grep -c '"type":"human"' "$file" 2>/dev/null || true)
    assistant_count=$(grep -c '"type":"assistant"' "$file" 2>/dev/null || true)
    tool_use_count=$(grep -c '"type":"tool_use"' "$file" 2>/dev/null || true)
    tool_result_count=$(grep -c '"type":"tool_result"' "$file" 2>/dev/null || true)
    echo -e "  User messages: ${user_count:-0}"
    echo -e "  Assistant messages: ${assistant_count:-0}"
    echo -e "  Tool uses: ${tool_use_count:-0}"
    echo -e "  Tool results: ${tool_result_count:-0}"
}

# Show tool usage summary
cmd_summary() {
    local file="$1"

    if [[ ! -f "$file" ]]; then
        echo -e "${RED}File not found: $file${NC}" >&2
        exit 1
    fi

    echo -e "${CYAN}Tool Usage Summary:${NC}"
    echo

    if command -v cclean &>/dev/null; then
        cclean -s plain "$file" 2>/dev/null | grep "^TOOL:" | sort | uniq -c | sort -rn | head -20
    else
        echo -e "${YELLOW}cclean not found, using basic analysis${NC}"
        echo
        # Fallback: extract tool names from raw JSON
        grep -o '"name":"[^"]*"' "$file" 2>/dev/null | sort | uniq -c | sort -rn | head -20
    fi
}

# Detect common inefficiency patterns
cmd_issues() {
    local file="$1"

    if [[ ! -f "$file" ]]; then
        echo -e "${RED}File not found: $file${NC}" >&2
        exit 1
    fi

    echo -e "${CYAN}Inefficiency Pattern Detection:${NC}"
    echo

    local parsed
    if command -v cclean &>/dev/null; then
        parsed=$(cclean -s plain "$file" 2>/dev/null)
    else
        echo -e "${YELLOW}cclean not found, using basic analysis${NC}"
        parsed=$(cat "$file")
    fi

    # Check for duplicate file reads
    echo -e "${BLUE}1. Duplicate File Reads:${NC}"
    echo "$parsed" | grep -o 'file_path[":]*[^,}]*' 2>/dev/null | sort | uniq -c | sort -rn | awk '$1 > 1 {print "   " $0}' | head -10
    echo

    # Check for checklists directory checks
    echo -e "${BLUE}2. Checklists Directory Checks:${NC}"
    local checklist_count
    checklist_count=$(echo "$parsed" | grep -ci 'checklist' 2>/dev/null | head -1 || true)
    checklist_count=${checklist_count:-0}
    if [[ "$checklist_count" -gt 0 ]]; then
        echo -e "   ${YELLOW}Found $checklist_count references to checklists${NC}"
    else
        echo "   None found"
    fi
    echo

    # Check for large file errors
    echo -e "${BLUE}3. Large File Handling Issues:${NC}"
    local large_file_errors
    large_file_errors=$(echo "$parsed" | grep -ci 'exceeds\|too large\|token.*limit' 2>/dev/null | head -1 || true)
    large_file_errors=${large_file_errors:-0}
    if [[ "$large_file_errors" -gt 0 ]]; then
        echo -e "   ${YELLOW}Found $large_file_errors large file issues${NC}"
        echo "$parsed" | grep -i 'exceeds\|too large\|token.*limit' 2>/dev/null | head -5 | sed 's/^/   /'
    else
        echo "   None found"
    fi
    echo

    # Check for Serena MCP errors
    echo -e "${BLUE}4. Serena MCP Errors:${NC}"
    local serena_errors
    serena_errors=$(echo "$parsed" | grep -ci 'language server\|not initialized\|mcp.*error' 2>/dev/null | head -1 || true)
    serena_errors=${serena_errors:-0}
    if [[ "$serena_errors" -gt 0 ]]; then
        echo -e "   ${YELLOW}Found $serena_errors Serena MCP issues${NC}"
    else
        echo "   None found"
    fi
    echo

    # Check for sandbox failures
    echo -e "${BLUE}5. Sandbox Restriction Issues:${NC}"
    local sandbox_issues
    sandbox_issues=$(echo "$parsed" | grep -ci 'sandbox\|dangerouslyDisable' 2>/dev/null | head -1 || true)
    sandbox_issues=${sandbox_issues:-0}
    if [[ "$sandbox_issues" -gt 0 ]]; then
        echo -e "   ${YELLOW}Found $sandbox_issues sandbox-related issues${NC}"
    else
        echo "   None found"
    fi
    echo

    # Check for redundant artifact reads
    echo -e "${BLUE}6. Redundant Artifact Reads (phase-context then individual files):${NC}"
    local phase_context
    phase_context=$(echo "$parsed" | grep -c 'phase-[0-9]*\.yaml' 2>/dev/null | head -1 || true)
    phase_context=${phase_context:-0}
    local spec_reads
    spec_reads=$(echo "$parsed" | grep -c 'spec\.yaml\|tasks\.yaml\|plan\.yaml' 2>/dev/null | head -1 || true)
    spec_reads=${spec_reads:-0}
    if [[ "$phase_context" -gt 0 ]] && [[ "$spec_reads" -gt 2 ]]; then
        echo -e "   ${YELLOW}Phase context read + $spec_reads individual artifact reads (potential redundancy)${NC}"
    else
        echo "   No obvious redundancy detected"
    fi
}

# Parse full conversation
cmd_parse() {
    local file="$1"
    local lines="${2:-}"

    if [[ ! -f "$file" ]]; then
        echo -e "${RED}File not found: $file${NC}" >&2
        exit 1
    fi

    if command -v cclean &>/dev/null; then
        if [[ -n "$lines" ]]; then
            cclean -s plain "$file" 2>/dev/null | head -"$lines"
        else
            cclean -s plain "$file" 2>/dev/null
        fi
    else
        echo -e "${RED}cclean not found. Install it to parse conversations.${NC}" >&2
        exit 1
    fi
}

# List unreviewed autospec conversations
cmd_unreviewed() {
    local project_filter="${1:-}"

    echo -e "${CYAN}Unreviewed Autospec Conversations:${NC}"
    echo

    # Get list of reviewed IDs (filter comments, empty lines, get first column)
    local reviewed_ids=""
    if [[ -f "$REVIEWED_FILE" ]]; then
        reviewed_ids=$(grep -v '^#' "$REVIEWED_FILE" | grep -v '^[[:space:]]*$' | awk '{print $1}' | tr '\n' '|')
        reviewed_ids="${reviewed_ids%|}"  # Remove trailing |
    fi

    # Find all autospec project directories
    local projects
    if [[ -n "$project_filter" ]]; then
        projects=$(ls -d "${CLAUDE_PROJECTS_DIR}"/*"${project_filter}"* 2>/dev/null || true)
    else
        projects=$(ls -d "${CLAUDE_PROJECTS_DIR}"/*autospec* 2>/dev/null || true)
    fi

    local found=0
    for project_dir in $projects; do
        local project_name
        project_name=$(basename "$project_dir")

        # Find autospec conversations
        for file in "${project_dir}"/*.jsonl; do
            [[ -f "$file" ]] || continue

            # Check if it's an autospec-TRIGGERED conversation (command at START)
            # Only check first 100 lines to distinguish from manual sessions
            # Note: autospec-triggered sessions have <command-name>/autospec.X</command-name> format
            local first_lines
            first_lines=$(head -100 "$file" 2>/dev/null)
            if ! echo "$first_lines" | grep -qE '(<command-name>/autospec\.|prereqs --json)'; then
                continue
            fi

            local basename
            basename=$(basename "$file" .jsonl)
            local short_id="${basename:0:8}"

            # Check if already reviewed
            if [[ -n "$reviewed_ids" ]] && echo "$short_id" | grep -qE "^($reviewed_ids)"; then
                continue
            fi

            # Get modification date and command type
            local mod_date
            mod_date=$(stat -c '%y' "$file" 2>/dev/null | cut -d' ' -f1 || stat -f '%Sm' -t '%Y-%m-%d' "$file" 2>/dev/null)

            local cmd_type="autospec"
            if echo "$first_lines" | grep -q '<command-name>/autospec\.implement'; then
                cmd_type="implement"
            elif echo "$first_lines" | grep -q '<command-name>/autospec\.specify'; then
                cmd_type="specify"
            elif echo "$first_lines" | grep -q '<command-name>/autospec\.plan'; then
                cmd_type="plan"
            elif echo "$first_lines" | grep -q '<command-name>/autospec\.tasks'; then
                cmd_type="tasks"
            elif echo "$first_lines" | grep -q '<command-name>/autospec\.'; then
                cmd_type="autospec"
            fi

            echo -e "  ${GREEN}${short_id}${NC} ${mod_date} ${YELLOW}${cmd_type}${NC} ${BLUE}${project_name}${NC}"
            found=$((found + 1))
        done
    done

    if (( found == 0 )); then
        echo -e "  ${GREEN}All autospec conversations have been reviewed!${NC}"
    else
        echo
        echo -e "${CYAN}Found $found unreviewed conversation(s)${NC}"
        echo -e "Use: $(basename "$0") info <project>/<id>.jsonl"
        echo -e "Then: $(basename "$0") mark <id> <cmd_type>"
    fi
}

# Show review status summary
cmd_status() {
    echo -e "${CYAN}Feedback Review Status:${NC}"
    echo

    # Count reviewed
    local reviewed_count=0
    if [[ -f "$REVIEWED_FILE" ]]; then
        reviewed_count=$(grep -v '^#' "$REVIEWED_FILE" | grep -v '^[[:space:]]*$' | wc -l)
    fi
    echo -e "  Reviewed: ${GREEN}${reviewed_count}${NC}"

    # Count unreviewed (quick scan)
    local unreviewed_count=0
    local projects
    projects=$(ls -d "${CLAUDE_PROJECTS_DIR}"/*autospec* 2>/dev/null || true)

    local reviewed_ids=""
    if [[ -f "$REVIEWED_FILE" ]]; then
        reviewed_ids=$(grep -v '^#' "$REVIEWED_FILE" | grep -v '^[[:space:]]*$' | awk '{print $1}' | tr '\n' '|')
        reviewed_ids="${reviewed_ids%|}"
    fi

    for project_dir in $projects; do
        for file in "${project_dir}"/*.jsonl; do
            [[ -f "$file" ]] || continue
            # Only count autospec-TRIGGERED conversations (command at START)
            # Note: autospec-triggered sessions have <command-name>/autospec.X</command-name> format
            local first_lines
            first_lines=$(head -100 "$file" 2>/dev/null)
            if ! echo "$first_lines" | grep -qE '(<command-name>/autospec\.|prereqs --json)'; then
                continue
            fi
            local basename
            basename=$(basename "$file" .jsonl)
            local short_id="${basename:0:8}"
            if [[ -n "$reviewed_ids" ]] && echo "$short_id" | grep -qE "^($reviewed_ids)"; then
                continue
            fi
            unreviewed_count=$((unreviewed_count + 1))
        done
    done

    echo -e "  Unreviewed: ${YELLOW}${unreviewed_count}${NC}"
    local total=$((reviewed_count + unreviewed_count))
    if [[ "$total" -gt 0 ]]; then
        local percent=$((reviewed_count * 100 / total))
        echo -e "  Progress: ${percent}%"
    fi
    echo
    echo -e "Run '$(basename "$0") unreviewed' to see unreviewed list"
}

# Mark conversation as reviewed
cmd_mark() {
    local id="$1"
    local cmd_type="$2"
    local project="${3:--home-ari-repos-autospec}"
    local date
    date=$(date +%Y-%m-%d)

    # Ensure reviewed file exists
    mkdir -p "$(dirname "$REVIEWED_FILE")"
    touch "$REVIEWED_FILE"

    # Check if already marked
    if grep -q "^${id}" "$REVIEWED_FILE" 2>/dev/null; then
        echo -e "${YELLOW}Conversation $id is already marked as reviewed${NC}"
        return 0
    fi

    # Add entry
    echo "$id $project $date $cmd_type" >> "$REVIEWED_FILE"
    echo -e "${GREEN}Marked $id as reviewed (${cmd_type})${NC}"
}

# Main
main() {
    local cmd="${1:-}"

    case "$cmd" in
        list)
            cmd_list "${2:-autospec}"
            ;;
        files)
            [[ -z "${2:-}" ]] && usage
            cmd_files "$2"
            ;;
        info)
            [[ -z "${2:-}" ]] && usage
            cmd_info "$2"
            ;;
        summary)
            [[ -z "${2:-}" ]] && usage
            cmd_summary "$2"
            ;;
        issues)
            [[ -z "${2:-}" ]] && usage
            cmd_issues "$2"
            ;;
        parse)
            [[ -z "${2:-}" ]] && usage
            cmd_parse "$2" "${3:-}"
            ;;
        unreviewed)
            cmd_unreviewed "${2:-}"
            ;;
        status)
            cmd_status
            ;;
        mark)
            [[ -z "${2:-}" || -z "${3:-}" ]] && { echo "Usage: mark <id> <cmd_type> [project]"; exit 1; }
            cmd_mark "$2" "$3" "${4:-}"
            ;;
        -h|--help|help|"")
            usage
            ;;
        *)
            echo -e "${RED}Unknown command: $cmd${NC}" >&2
            usage
            ;;
    esac
}

main "$@"
