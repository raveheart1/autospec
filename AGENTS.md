# AGENTS.md

Guidelines for AI coding agents working in the autospec repository.

## Build/Test/Lint Commands

```bash
make build              # Build binary for current platform
make install            # Build and install to ~/.local/bin
make test               # Run all tests (quiet, failures only)
make test-v             # Run all tests (verbose)
make fmt                # Format Go code (required before commit)
make lint               # Run all linters (fmt + vet + shellcheck)

# Run single test
go test -run TestName ./internal/package/

# Full validation before committing
make fmt && make lint && make test && make build
```

## Project Structure

```
cmd/autospec/main.go     # Entry point (minimal, no business logic)
internal/
  cli/                   # Cobra commands (stages/, config/, util/, admin/)
  workflow/              # Workflow orchestration and Claude execution
  config/                # Hierarchical config (koanf-based)
  validation/            # Artifact validation (<10ms performance contract)
  errors/                # Structured error types with remediation
  agent/                 # Agent abstraction layer
specs/                   # Feature specifications (spec.yaml, plan.yaml, tasks.yaml)
```

## Code Style

### Imports
Group in order with blank lines: 1) Standard library, 2) External packages, 3) Internal packages

```go
import (
    "fmt"
    "os"

    "github.com/spf13/cobra"

    "github.com/ariel-frischer/autospec/internal/config"
)
```

### Naming Conventions

```go
package validation     // Packages: short, lowercase, no underscores
func ValidateSpecFile  // Exported: CamelCase
func parseTaskLine     // Unexported: camelCase
type Config struct{}   // Avoid stutter (not config.ConfigStruct)
type Validator interface { Validate() error }  // Interfaces: -er suffix
```

### Error Handling (CRITICAL)

**Always wrap errors with context** - never use bare `return err`:

```go
// BAD
return err

// GOOD
return fmt.Errorf("loading config file: %w", err)
```

Use structured errors from `internal/errors/` for user-facing CLI errors:
```go
return errors.NewValidationError("spec.yaml", "missing required field: feature")
```

### Function Design

- **Max 40 lines** per function. Extract helpers for complex logic.
- **Accept interfaces, return concrete types**
- **Context as first parameter** when needed for cancellation

### Testing (Map-Based Table Tests Required)

```go
func TestValidateSpecFile(t *testing.T) {
    tests := map[string]struct {
        input   string
        wantErr bool
    }{
        "valid input": {input: "foo"},
        "empty input": {input: "", wantErr: true},
    }
    for name, tt := range tests {
        t.Run(name, func(t *testing.T) {
            t.Parallel()
            // test logic
        })
    }
}
```

### CLI Command Lifecycle Wrapper

All workflow commands MUST use the lifecycle wrapper for notifications/history:

```go
notifHandler := notify.NewHandler(cfg.Notifications)
historyLogger := history.NewWriter(cfg.StateDir, cfg.MaxHistoryEntries)

return lifecycle.RunWithHistory(notifHandler, historyLogger, "command-name", specName, func() error {
    return orch.ExecuteXxx(...)
})
```

## Key Patterns

### Configuration (koanf)
Priority: `AUTOSPEC_*` env vars > `.autospec/config.yml` > `~/.config/autospec/config.yml` > defaults

Adding a config field requires updates to:
1. `internal/config/config.go` - struct field with `koanf:"field_name"` tag
2. `internal/config/schema.go` - entry in `KnownKeys` map
3. `internal/config/defaults.go` - default value

### Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Validation failed (retryable) |
| 2 | Retry limit exhausted |
| 3 | Invalid arguments |
| 4 | Missing dependencies |
| 5 | Timeout |

### Performance Contracts
- Validation functions: <10ms
- Retry state load/save: <10ms
- Config loading: <100ms

## Non-Negotiable Rules

1. **Test-First Development**: Write tests before implementation
2. **Error Context**: Never use bare `return err`
3. **Function Length**: Max 40 lines
4. **Map-Based Tests**: Use `map[string]struct{}` pattern
5. **Lifecycle Wrapper**: All workflow commands use `lifecycle.RunWithHistory`
6. **Command Template Independence**: `internal/commands/*.md` must be project-agnostic

## Git Commits

```bash
# BAD - heredocs fail in sandbox
git commit -m "$(cat <<'EOF'
message
EOF
)"

# GOOD - use quoted string with newlines
git commit -m "feat(scope): description

Body text here."
```

## Documentation Workflow

**Before starting any task**: Review relevant docs from the list below.
**After completing work**: Use a subagent (Task tool) to update affected documentation to preserve main agent context.

### Documentation Reference

#### Internal (Developer-Focused)
| Document | Purpose |
|----------|---------|
| `docs/internal/architecture.md` | System design, component structure, execution patterns with diagrams |
| `docs/internal/go-best-practices.md` | Go coding conventions and patterns for autospec |
| `docs/internal/internals.md` | Spec detection, validation, retry handling, phase context injection |
| `docs/internal/agents.md` | CLI agent configuration and multi-agent abstraction layer |
| `docs/internal/YAML-STRUCTURED-OUTPUT.md` | YAML artifact schemas and command templates |
| `docs/internal/testing-mocks.md` | Mock testing infrastructure for workflows without real Claude calls |
| `docs/internal/events.md` | Event-driven architecture using kelindar/event |
| `docs/internal/risks.md` | Documenting implementation risks in plan.yaml |
| `docs/internal/cclean.md` | claude-clean tool for transforming streaming JSON output |

#### Public (User-Focused)
| Document | Purpose |
|----------|---------|
| `docs/public/reference.md` | Complete CLI command reference with flags and examples |
| `docs/public/quickstart.md` | Getting started guide for first workflow |
| `docs/public/overview.md` | High-level introduction to autospec |
| `docs/public/claude-settings.md` | Claude Code settings, sandboxing, permissions |
| `docs/public/troubleshooting.md` | Common issues and solutions |
| `docs/public/TIMEOUT.md` | Timeout configuration for Claude commands |
| `docs/public/SHELL-COMPLETION.md` | Shell completion setup (bash, zsh, fish, powershell) |
| `docs/public/checklists.md` | Checklists as "unit tests for requirements" |
| `docs/public/worktree.md` | Git worktree management for parallel agent execution |
| `docs/public/parallel-execution.md` | Sequential and parallel execution with DAG scheduling |
| `docs/public/self-update.md` | Version checking and self-update functionality |
| `docs/public/faq.md` | Frequently asked questions |

#### Research
| Document | Purpose |
|----------|---------|
| `docs/research/claude-opus-4.5-context-performance.md` | Claude Opus 4.5 performance in extended sessions |

### Subagent Documentation Update Pattern

After completing implementation work, spawn a subagent to update docs:

```
Task tool with prompt:
"Review and update documentation affected by [describe changes].
Files to check: [list relevant doc files from above].
Only update if changes are needed. Do not create new files."
```
