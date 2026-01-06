# Task: Constitution Config Generation

> Part of: [Regression-Free Integration](./00-overview.md)
> Reference: `.dev/tasks/SPECKIT_REGRESSION_FREE_INTEGRATION.md` (Layer 1: Constitution → CI)

## Summary

Generate tool configurations (linter rules, CI workflows, complexity gates) from constitution.yaml. This makes constitution principles enforceable rather than advisory—violations fail builds, not just reviews.

## Motivation

Today's constitution.yaml contains principles like "keep functions under 40 lines" that rely on AI compliance and human review. By generating linter configs from these principles, violations become compile-time errors. The constitution becomes executable.

## Design

### Enhanced Constitution Schema

Extend constitution.yaml with machine-enforceable constraints:

```yaml
# .autospec/memory/constitution.yaml
name: "My Project"
version: "1.0"

principles:
  - id: "P-001"
    text: "Functions must be under 40 lines"
    enforceable: true
    enforcement:
      tool: golangci-lint
      rule: funlen
      config:
        lines: 40

  - id: "P-002"
    text: "Cyclomatic complexity under 10"
    enforceable: true
    enforcement:
      tool: golangci-lint
      rule: gocyclo
      config:
        max-complexity: 10

  - id: "P-003"
    text: "No circular dependencies"
    enforceable: true
    enforcement:
      tool: go-arch-lint  # or custom
      rule: no-cycles

  - id: "P-004"
    text: "Test coverage above 80%"
    enforceable: true
    enforcement:
      tool: go-test
      config:
        coverage-threshold: 80
```

### Generation Command

```bash
# Generate all configs from constitution
autospec generate-configs

# Generate specific tool config
autospec generate-configs --tool golangci-lint

# Preview without writing
autospec generate-configs --dry-run

# Output to custom location
autospec generate-configs --output ./ci/
```

### Generated Artifacts

| Constitution Rule | Generated Config |
|-------------------|------------------|
| Function length | `.golangci.yml` linters.funlen |
| Cyclomatic complexity | `.golangci.yml` linters.gocyclo |
| Architecture rules | `.go-arch-lint.yml` or similar |
| Coverage threshold | CI workflow step |
| Test requirements | CI workflow step |

### Example Generated Config

From constitution with complexity and length rules:

```yaml
# Generated: .golangci.yml
# Source: .autospec/memory/constitution.yaml
# DO NOT EDIT - regenerate with: autospec generate-configs

linters:
  enable:
    - funlen
    - gocyclo

linters-settings:
  funlen:
    lines: 40        # From P-001
  gocyclo:
    max-complexity: 10  # From P-002
```

### Conflict Handling

If config file already exists:

1. Parse existing config
2. Merge constitution rules (constitution takes precedence)
3. Preserve non-conflicting user settings
4. Add header comment indicating generated sections

## Implementation Notes

### Generator Package

New package `internal/codegen/`:

- Constitution parser with enforcement extraction
- Per-tool config generators (golangci, eslint, etc.)
- Merge logic for existing configs
- Template system for CI workflows

### Tool Support Matrix

Start with Go tooling, expand based on usage:

| Tool | Config File | Supported Rules |
|------|-------------|-----------------|
| golangci-lint | .golangci.yml | funlen, gocyclo, gocognit |
| go test | Makefile/CI | coverage threshold |
| go-arch-lint | .go-arch-lint.yml | layer dependencies |

### CI Workflow Generation

Optional: Generate GitHub Actions workflow:

```yaml
# Generated: .github/workflows/constitution.yml
name: Constitution Checks
on: [push, pull_request]
jobs:
  verify:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Lint (Constitution Rules)
        run: golangci-lint run
      - name: Coverage Check
        run: |
          go test -coverprofile=coverage.out ./...
          # Check threshold from constitution
```

## Acceptance Criteria

1. `autospec generate-configs` produces valid tool configs
2. Generated configs trace back to constitution principles (comments)
3. Existing configs are merged, not overwritten
4. `--dry-run` shows what would be generated
5. Re-running is idempotent (same input → same output)
6. Invalid enforcement rules produce helpful errors

## Non-Goals (Future Work)

- Runtime monitoring (AgentGuard)
- CI provider abstraction (GitLab, CircleCI)
- Non-Go language support (TypeScript, Python)

## Dependencies

- `01-verification-config.md` (respects verification level)

## Estimated Scope

Medium-Large. Constitution schema extension, per-tool generators, merge logic.
