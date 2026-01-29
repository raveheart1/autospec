# CLI Argument Parsing Edge Cases

**Related to**: 107-frontmatter-cli-parsing-bug.md
**Status**: Research complete, optional hardening identified
**Priority**: Low (main issue already fixed)

## Context

After fixing the YAML frontmatter `---` bug, we investigated other string patterns that could break CLI argument parsing when passing rendered templates to `claude -p "<prompt>"`.

Since we use `exec.Command()` (not shell invocation), shell metacharacters are NOT a concern. The risks are in:
1. The Claude CLI's argument parser (likely Node.js Commander.js)
2. OS-level argument handling (POSIX, ARG_MAX)

## Already Fixed

| Issue | Fix | Location |
|-------|-----|----------|
| Leading dashes (`-`, `--`, `---`) | Prepend newline to prevent flag interpretation | `sanitizePromptForCLI()` in `internal/cliagent/base.go` |
| YAML frontmatter | Strip before passing to CLI | `StripFrontmatter()` in `internal/commands/templates.go` |

## Remaining Edge Cases

### 1. Null Bytes (`\0`) - HIGH PRIORITY

**Risk**: POSIX and Node.js treat null bytes as argument terminators. Content after `\0` is silently truncated.

**Source**: Node.js issue [#44768](https://github.com/nodejs/node/issues/44768) - "null bytes and subsequent data are silently ignored without error"

**Impact**: Template content could be silently cut off mid-prompt.

**Likelihood**: Low (null bytes unlikely in YAML/Markdown templates), but silent failure makes this dangerous.

**Fix**:
```go
prompt = strings.ReplaceAll(prompt, "\x00", "")
```

### 2. UTF-8 BOM (`\xEF\xBB\xBF`) - LOW PRIORITY

**Risk**: If template files are saved with BOM, the rendered output starts with invisible bytes that could confuse parsing or display.

**Impact**: Cosmetic issues, possible parsing confusion.

**Likelihood**: Low (modern editors don't add BOM by default).

**Fix**:
```go
prompt = strings.TrimPrefix(prompt, "\xef\xbb\xbf")
```

### 3. ARG_MAX Limits - LOW PRIORITY

**Risk**: OS limits on total argument length (~128KB Linux, ~256KB macOS).

**Impact**: `exec.Command` fails with "argument list too long" error.

**Likelihood**: Very low. Templates are small. Even with retry context injection, prompts stay under 50KB.

**Fix**: Log warning if prompt exceeds 100KB threshold.

```go
if len(prompt) > 100*1024 {
    log.Warn("prompt exceeds 100KB, may hit OS ARG_MAX limits")
}
```

### 4. Carriage Returns (`\r`) - LOW PRIORITY

**Risk**: Mixed line endings (`\r\n` or bare `\r`) could cause display issues or unexpected behavior.

**Impact**: Cosmetic, possible parsing quirks on Windows-originated content.

**Likelihood**: Low.

**Fix**:
```go
prompt = strings.ReplaceAll(prompt, "\r\n", "\n")
prompt = strings.ReplaceAll(prompt, "\r", "\n")
```

### 5. Control Characters (`\x00-\x1F`, `\x7F`) - LOW PRIORITY

**Risk**: Control characters (except `\t`, `\n`) could cause terminal display issues.

**Impact**: Cosmetic only.

**Likelihood**: Very low in template content.

**Fix**: Not recommended (would strip legitimate tabs/newlines). Only strip if issues arise.

### 6. Unrendered Template Variables - MEDIUM PRIORITY

**Risk**: If template rendering fails silently, literal `{{.FeatureDir}}` could appear in output.

**Impact**: Agent receives malformed prompt, unexpected behavior.

**Likelihood**: Low (Go templates error on missing keys by default).

**Status**: Already tested in `tests/e2e/template_rendering_test.go`.

## Recommended Hardening

Update `sanitizePromptForCLI()` in `internal/cliagent/base.go`:

```go
// sanitizePromptForCLI ensures prompt content won't break CLI argument parsing.
// Handles:
// - Null bytes (silently truncate args in POSIX/Node.js)
// - UTF-8 BOM (invisible prefix bytes)
// - Mixed line endings (normalize to \n)
// - Leading dashes (prevent flag interpretation)
func sanitizePromptForCLI(prompt string) string {
    // CRITICAL: Null bytes truncate arguments silently
    prompt = strings.ReplaceAll(prompt, "\x00", "")

    // Strip UTF-8 BOM if present
    prompt = strings.TrimPrefix(prompt, "\xef\xbb\xbf")

    // Normalize line endings
    prompt = strings.ReplaceAll(prompt, "\r\n", "\n")
    prompt = strings.ReplaceAll(prompt, "\r", "\n")

    // Prevent leading dash interpretation
    if strings.HasPrefix(prompt, "-") {
        return "\n" + prompt
    }
    return prompt
}
```

## Alternative: `--` Option Terminator

A more robust approach is using `--` to signal end of options:

```go
args = append(args, "-p", "--", prompt)
```

Commander.js respects `--` as an option terminator. However, this changes the argument structure and requires testing with Claude CLI.

## Testing Recommendations

Add test cases to `TestSanitizePromptForCLI`:

```go
"null byte in prompt gets stripped": {
    input: "before\x00after",
    want:  "beforeafter",
},
"utf8 bom gets stripped": {
    input: "\xef\xbb\xbfcontent",
    want:  "content",
},
"crlf normalized to lf": {
    input: "line1\r\nline2",
    want:  "line1\nline2",
},
"bare cr normalized to lf": {
    input: "line1\rline2",
    want:  "line1\nline2",
},
```

## Decision

**Implement**: Null byte stripping (silent truncation is dangerous)
**Defer**: BOM, CRLF, ARG_MAX (low likelihood, easy to add if issues arise)

## Sources

- Perplexity research on Commander.js edge cases
- Node.js issue [#44768](https://github.com/nodejs/node/issues/44768) - null byte handling
- [oneuptime blog](https://oneuptime.com/blog/post/2026-01-22-nodejs-parse-command-line-arguments/view) - CLI parsing best practices
