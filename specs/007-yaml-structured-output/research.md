# Research: YAML Structured Output

**Feature**: 007-yaml-structured-output
**Date**: 2025-12-13

## Research Summary

This document consolidates research findings for implementing YAML structured output in autospec.

---

## 1. Go Embedding with go:embed

### Decision
Use `go:embed` with `embed.FS` for embedding command templates in the binary.

### Rationale
- Single self-contained binary aligns with autospec's deployment model
- Command templates are static, small files versioned with the app
- `embed.FS` provides standard `fs.FS` APIs for reading templates

### Implementation Pattern

```go
package commands

import "embed"

//go:embed templates/*.md
var TemplateFS embed.FS

// Access via: TemplateFS.ReadFile("templates/autospec.specify.md")
```

### Key Decisions

| Aspect | Decision | Rationale |
|--------|----------|-----------|
| Embed type | `embed.FS` | Standard fs.FS APIs, supports multiple files |
| Directory | Top-level `commands/` | go:embed requires package-relative paths |
| File pattern | `*.md` glob | Narrow pattern, explicit asset selection |
| Subdirectories | None | Flat structure for 6 command templates |

### Alternatives Considered

1. **`[]byte` per file** - Rejected: Requires separate variable per template, less maintainable
2. **External files at runtime** - Rejected: Breaks single-binary deployment model
3. **Build-time code generation** - Rejected: Unnecessary complexity for static templates

---

## 2. YAML Validation with yaml.v3

### Decision
Use `gopkg.in/yaml.v3` with `yaml.Decoder` streaming for syntax validation.

### Rationale
- Already an indirect dependency (via testify)
- Streaming with `Decoder` handles large files efficiently
- Decode into `yaml.Node` for syntax-only validation (no schema)

### Implementation Pattern

```go
package yaml

import (
    "io"
    "gopkg.in/yaml.v3"
)

func ValidateSyntax(r io.Reader) error {
    dec := yaml.NewDecoder(r)
    for {
        var n yaml.Node
        if err := dec.Decode(&n); err != nil {
            if err == io.EOF {
                return nil // valid
            }
            return err // syntax error with line info
        }
    }
}
```

### Performance Characteristics

| Metric | Target | Approach |
|--------|--------|----------|
| 10MB file validation | <100ms | Streaming Decoder, yaml.Node target |
| Memory usage | O(document) not O(file) | Stream processing, no full file read |
| Error reporting | Line numbers | yaml.v3 provides line/column in errors |

### Alternatives Considered

1. **goccy/go-yaml** - Rejected: Would add new dependency; yaml.v3 sufficient for validation
2. **Full struct unmarshal** - Rejected: More expensive for syntax-only check
3. **Read full file + Unmarshal** - Rejected: Memory-intensive for large files

---

## 3. Claude Code Command Template Format

### Decision
Use Markdown files with YAML frontmatter in `.claude/commands/` directory.

### Rationale
- Standard Claude Code command format
- Supports `$ARGUMENTS` for user input injection
- Integrates with Claude Code's `/help` system

### Template Structure

```markdown
---
description: Brief description shown in /help
---

# Command Title

Instructions for Claude to execute.

Use $ARGUMENTS for user-provided input.

## Steps
1. Step one
2. Step two

## Expected Output
Description of what gets generated.
```

### Key Decisions

| Aspect | Decision | Rationale |
|--------|----------|-----------|
| Naming | `autospec.*.md` | Distinguishes from existing `speckit.*` commands |
| Variables | `$ARGUMENTS` | Standard Claude Code variable for user input |
| Frontmatter | Required `description` | Enables /help integration |
| Location | `.claude/commands/` | Standard Claude Code directory |

### Template Content Requirements

Each template must instruct Claude to:
1. Generate the appropriate YAML artifact (spec.yaml, plan.yaml, etc.)
2. Include `_meta` section with version, generator, timestamp
3. Run `autospec yaml check <file>` as final validation step
4. Handle errors gracefully if validation fails

---

## 4. _meta Section Schema

### Decision
All YAML artifacts include a `_meta` section with version, generator, and timestamp.

### Rationale
- Enables version compatibility checking (FR-014)
- Provides traceability for generated artifacts
- Follows common metadata patterns in structured formats

### Schema

```yaml
_meta:
  version: "1.0.0"           # Artifact schema version
  generator: "autospec"      # Generator tool name
  generator_version: "0.1.0" # Generator tool version
  created: "2025-12-13T10:30:00Z" # ISO 8601 timestamp
```

### Version Compatibility Strategy

| Scenario | Behavior |
|----------|----------|
| Major version mismatch | Warn but proceed (FR-014) |
| Minor version mismatch | Silent processing |
| Missing _meta | Add default metadata, warn |

---

## 5. Command Installation Strategy

### Decision
Copy embedded templates to `.claude/commands/` with idempotent overwrite behavior.

### Rationale
- Simple file copy operation
- Idempotent (can run multiple times safely)
- Preserves user-added commands (only overwrites autospec.* files)

### Implementation

```go
func InstallCommands(targetDir string) error {
    entries, _ := TemplateFS.ReadDir("templates")
    for _, entry := range entries {
        if !strings.HasPrefix(entry.Name(), "autospec.") {
            continue // Only install our commands
        }
        content, _ := TemplateFS.ReadFile("templates/" + entry.Name())
        targetPath := filepath.Join(targetDir, entry.Name())
        os.WriteFile(targetPath, content, 0644)
    }
    return nil
}
```

### Key Behaviors

| Behavior | Implementation |
|----------|----------------|
| Create directory if missing | `os.MkdirAll(targetDir, 0755)` |
| Overwrite existing | Always overwrite `autospec.*` files |
| Preserve user commands | Skip files not matching `autospec.*` |
| Atomic writes | Write to temp, rename |

---

## 6. Version Comparison Strategy

### Decision
Embed version in binary and in each template's frontmatter for comparison.

### Rationale
- Enables `autospec commands check` to detect outdated commands
- Version stored in template allows per-command version tracking

### Implementation

```yaml
# In embedded template:
---
description: Generate YAML specification
version: "1.0.0"
---
```

```go
// Version comparison
func CheckCommandVersions(installedDir string) ([]VersionMismatch, error) {
    // Compare embedded template versions vs installed file versions
}
```

---

## Dependencies Summary

| Dependency | Version | Purpose | Status |
|------------|---------|---------|--------|
| gopkg.in/yaml.v3 | v3.0.1 | YAML parsing/validation | Already indirect dep |
| embed (stdlib) | Go 1.25 | File embedding | Standard library |
| Cobra | v1.10.1 | CLI subcommands | Existing dependency |

**No new dependencies required.** The yaml.v3 package is already an indirect dependency via testify, and will be promoted to direct dependency for YAML validation.
