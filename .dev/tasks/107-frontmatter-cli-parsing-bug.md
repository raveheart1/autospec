# Bug: YAML Frontmatter Causes Claude CLI Parsing Error

**Issue**: 107-simplify-yaml-meta-init
**Severity**: Critical (breaks plan, tasks, and other rendered stages)
**Introduced in**: v0.10.2 (spec 106-wire-template-rendering)
**Fixed in**: v0.10.3

## Summary

After implementing template rendering (spec 106), the `plan`, `tasks`, and other stages that use rendered templates started failing with:

```
error: unknown option '---
description: Generate YAML implementation plan from feature specification.
version: "1.0.0"
---
...
```

## Root Cause

The template rendering feature (spec 106) changed how commands are sent to the Claude CLI:

**Before (v0.10.1 and earlier)**:
- Commands were passed as slash commands (e.g., `/autospec.plan`)
- Claude Code natively expanded these using `.claude/commands/` templates
- Frontmatter was handled internally by Claude Code

**After (v0.10.2)**:
- Templates are rendered by autospec with pre-computed context values
- Full rendered content is passed to `claude -p "<content>"`
- Template content includes YAML frontmatter (`---...---`)

The Claude CLI's argument parser misinterprets the leading `---` as a flag delimiter, causing the entire prompt to be treated as an unknown option.

## Why Testing Didn't Catch This

1. **Mock script detection**: The mock-claude.sh script detected templates by patterns in the YAML frontmatter (e.g., "Generate YAML implementation plan"), which was always present during testing
2. **Unit tests**: Template rendering tests verified variable substitution worked but didn't test CLI parsing behavior
3. **E2E tests**: Existing tests used mock scripts that didn't replicate Claude CLI's argument parsing

## Fix

### 1. Added `StripFrontmatter()` function (`internal/commands/templates.go`)

```go
// StripFrontmatter removes YAML frontmatter from template content.
func StripFrontmatter(content []byte) []byte {
    if !bytes.HasPrefix(content, []byte("---")) {
        return content
    }
    rest := content[3:]
    endIdx := bytes.Index(rest, []byte("\n---"))
    if endIdx == -1 {
        return content
    }
    afterFrontmatter := rest[endIdx+4:]
    return bytes.TrimLeft(afterFrontmatter, "\n")
}
```

### 2. Added `sanitizePromptForCLI()` with null byte stripping (`internal/cliagent/base.go`)

```go
// sanitizePromptForCLI ensures prompt content won't be misinterpreted as CLI flags
// and removes dangerous characters.
func sanitizePromptForCLI(prompt string) string {
    // Strip null bytes (string terminators that could truncate arguments)
    prompt = strings.ReplaceAll(prompt, "\x00", "")

    // Prevent flag interpretation for prompts starting with "-"
    if strings.HasPrefix(prompt, "-") {
        return "\n" + prompt
    }
    return prompt
}
```

### 3. Integrated into render pipeline (`internal/commands/render.go`)

```go
func RenderTemplate(content []byte, ctx *prereqs.Context) ([]byte, error) {
    // ... template execution ...

    // Strip YAML frontmatter after rendering
    return StripFrontmatter(buf.Bytes()), nil
}
```

### 4. Updated mock-claude.sh detection patterns

Changed from frontmatter-based detection:
```bash
# OLD: Looked for frontmatter descriptions
if [[ "$command" == *"Generate YAML implementation plan"* ]]; then
```

To body-based detection:
```bash
# NEW: Looks for patterns in template body
if [[ "$command" == *"**Write the plan** to"* ]]; then
```

### 5. Added regression tests

- Unit test: `TestStripFrontmatter` in `internal/commands/templates_test.go`
- Unit test: `TestSanitizePromptForCLI` in `internal/cliagent/base_test.go` (includes null byte tests)
- Unit test: `TestRenderPreservesMarkdownStructure` updated to verify frontmatter is stripped
- E2E test: `TestE2E_TemplateRendering_FrontmatterStripped` in `tests/e2e/template_rendering_test.go`

### 6. Added display truncation for long prompts

To avoid showing the entire rendered template in terminal output:

```go
// TruncatePromptForDisplay shortens long prompts for terminal display.
// Returns original prompt if short enough, otherwise first line truncated + "...".
func TruncatePromptForDisplay(prompt string) string {
    // Get first line only (most prompts are multiline rendered templates)
    firstNewline := strings.Index(prompt, "\n")
    firstLine := prompt
    hasMoreLines := false
    if firstNewline > 0 {
        firstLine = prompt[:firstNewline]
        hasMoreLines = true
    }
    // ... truncation logic
}
```

In debug mode (`--debug`), the full command is still shown.

## Files Changed

- `internal/commands/templates.go` - Added `StripFrontmatter()`
- `internal/commands/templates_test.go` - Added unit tests
- `internal/commands/render.go` - Integrated frontmatter stripping
- `internal/cli/render_command_test.go` - Updated test expectations
- `internal/cliagent/base.go` - Added `sanitizePromptForCLI()`
- `internal/cliagent/base_test.go` - Added unit tests
- `internal/workflow/executor.go` - Added `TruncatePromptForDisplay()` and display truncation
- `internal/workflow/executor_test.go` - Added unit tests
- `tests/mocks/scripts/mock-claude.sh` - Updated detection patterns
- `tests/e2e/template_rendering_test.go` - Added regression test

## Prevention

To prevent similar issues:

1. **CLI argument testing**: Add tests that verify rendered prompts don't contain patterns that could be misinterpreted as CLI flags
2. **Integration testing**: Test actual `claude` CLI parsing behavior, not just mock scripts
3. **Frontmatter awareness**: Any code that passes content to CLI tools should strip or escape YAML frontmatter

## Related

- Spec 106: Wire template rendering (introduced the regression)
- Spec 107: Simplify YAML meta init (branch where bug was discovered)
