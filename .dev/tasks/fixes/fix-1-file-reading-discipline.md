# Fix 1: File Reading Discipline in implement.md (CRITICAL)

**Problem:** Claude reads the same file 5-23 times per session. Example: `preflight_test.go` read 18 times, `parse-claude-conversation.sh` read 14 times.

**Root Cause:** No explicit guidance telling Claude that file contents remain in context after reading.

**Token Savings:** 30-50K/session
**Effort:** Low

## Command

```bash
autospec specify <<'EOF'
Add file reading discipline section to internal/commands/autospec.implement.md template. This addresses a CRITICAL issue where Claude reads the same file 5-23 times per session, wasting 30-50K tokens.

## Problem Statement
Analysis of 29 autospec sessions revealed extreme file re-reading:
- preflight_test.go read 18 times in one session
- workflow.go read 16 times in one session
- parse-claude-conversation.sh read 14 times in one session
- schema_validation.go read 6-7 times in sessions

Claude does not recognize that file contents remain in its context window after the initial Read tool call.

## Required Changes

Add a new section to internal/commands/autospec.implement.md after the context loading section:

### Section Title: 'CRITICAL: File Reading Discipline'

### Content to Add:

1. 'Read Once, Remember Forever' rule block explaining:
   - When you read a file with the Read tool, the content IS NOW in your context window
   - You can reference file contents without re-reading
   - DO NOT read the same file again unless you made changes and need to verify (max 1 re-read)

2. Maximum File Read Counts table:
   | Scenario | Max Reads |
   |----------|-----------|
   | Understanding a file | 1 |
   | Editing a file | 2 (before + after) |
   | Referencing while editing another | 0 (already have it) |
   | Debugging test failures | 2 |

3. 'Pre-Task File Discovery' protocol:
   - Before starting implementation, identify ALL files you will need
   - Read each file ONCE at the start
   - Note line numbers of relevant sections
   - Proceed with implementation WITHOUT re-reading

4. Explicit prohibitions:
   - DO NOT re-read files to 'make sure' you have the content
   - DO NOT re-read files when switching between tasks
   - DO NOT grep a file then read it fully then grep again

## Acceptance Criteria
- [ ] New 'CRITICAL: File Reading Discipline' section exists in implement.md
- [ ] Section appears prominently (near top, after context loading)
- [ ] Includes the Maximum File Read Counts table
- [ ] Includes Pre-Task File Discovery protocol
- [ ] Uses clear formatting with checkmarks and X marks for dos/donts

## Non-Functional Requirements
- Template change only, no Go code changes
- Section must be scannable (headers, bullets, tables)
- Use warning/critical styling to emphasize importance
EOF
```
