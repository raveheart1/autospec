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

## Documentation Reference

| Document | Purpose |
|----------|---------|
| `docs/internal/architecture.md` | System design and component diagrams |
| `docs/internal/go-best-practices.md` | Full Go conventions |
| `docs/public/reference.md` | CLI command reference |
| `docs/internal/internals.md` | Spec detection, validation, retry system |
