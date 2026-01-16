# render-command

Preview rendered command templates with pre-computed feature context.

## Overview

The `render-command` command renders autospec slash command templates with the current feature context already filled in. This is primarily a **debugging and preview tool** that lets you see exactly what the agent will receive when a slash command is executed.

This command was introduced as part of the pre-computed prereqs context feature, which eliminates bash calls from slash commands by computing context (feature directory, spec paths, version info) in Go before template execution.

## Syntax

```bash
autospec render-command <command-name> [flags]
```

## Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--output` | `-o` | stdout | Output file path |
| `--specs-dir` | | `./specs` | Specs directory (global flag) |

## Arguments

| Argument | Required | Description |
|----------|----------|-------------|
| `command-name` | Yes | Name of the command template to render (e.g., `autospec.plan`) |

## Available Commands

The following command templates can be rendered:

| Command | Required Context | Description |
|---------|-----------------|-------------|
| `autospec.specify` | (none) | Create feature specification |
| `autospec.plan` | `spec.yaml` | Generate implementation plan |
| `autospec.tasks` | `spec.yaml`, `plan.yaml` | Generate task breakdown |
| `autospec.implement` | `tasks.yaml` | Execute implementation |
| `autospec.checklist` | `spec.yaml` | Generate quality checklist |
| `autospec.clarify` | `spec.yaml` | Clarify spec ambiguities |
| `autospec.analyze` | `spec.yaml` | Cross-artifact analysis |
| `autospec.constitution` | (none) | Create project constitution |
| `autospec.worktree-setup` | (none) | Generate worktree setup script |

## Template Variables

Commands use Go `text/template` syntax with these variables:

| Variable | Description | Example |
|----------|-------------|---------|
| `{{.FeatureDir}}` | Feature spec directory | `specs/103-feature-name` |
| `{{.FeatureSpec}}` | Path to spec.yaml | `specs/103-feature-name/spec.yaml` |
| `{{.ImplPlan}}` | Path to plan.yaml | `specs/103-feature-name/plan.yaml` |
| `{{.TasksFile}}` | Path to tasks.yaml | `specs/103-feature-name/tasks.yaml` |
| `{{.AutospecVersion}}` | Current version | `autospec 0.9.0` |
| `{{.CreatedDate}}` | ISO 8601 timestamp | `2026-01-16T10:30:00Z` |
| `{{.IsGitRepo}}` | Git repo detection | `true` |
| `{{.AvailableDocs}}` | Available artifact files | `[spec.yaml, plan.yaml]` |

## Examples

### Preview a Command Template

```bash
# See what the plan command looks like for current feature
autospec render-command autospec.plan
```

### Save to File for Inspection

```bash
# Write rendered template to a file
autospec render-command autospec.tasks --output /tmp/tasks-prompt.md

# Then inspect it
cat /tmp/tasks-prompt.md
```

### Pipe to Clipboard

```bash
# macOS
autospec render-command autospec.implement | pbcopy

# Linux (xclip)
autospec render-command autospec.implement | xclip -selection clipboard
```

### Debug Feature Detection

If commands aren't detecting the right feature, render the template to see what context is being computed:

```bash
# Check what feature directory is detected
autospec render-command autospec.checklist | head -50
```

### Use with External Tools

```bash
# Pipe to another agent or tool
autospec render-command autospec.plan | my-custom-agent --stdin

# Diff two rendered outputs
autospec render-command autospec.plan > /tmp/before.md
# ... make changes ...
autospec render-command autospec.plan > /tmp/after.md
diff /tmp/before.md /tmp/after.md
```

## Use Cases

### 1. Debugging Template Issues

When a slash command isn't working as expected, render it to see the actual content:

```bash
autospec render-command autospec.implement
```

Check that all `{{.Variable}}` placeholders are properly substituted.

### 2. Verifying Context Detection

Ensure the correct feature is detected from your git branch:

```bash
# On branch 103-precompute-prereqs-context
autospec render-command autospec.plan | grep -A5 "Feature directory"
```

### 3. Testing Template Changes

After modifying a command template in `internal/commands/`, verify the rendering:

```bash
# Before
autospec render-command autospec.plan > /tmp/before.md

# After changes
make build
autospec render-command autospec.plan > /tmp/after.md

# Compare
diff /tmp/before.md /tmp/after.md
```

### 4. CI/CD Integration

Generate rendered prompts for automated workflows:

```bash
# In CI pipeline
autospec render-command autospec.plan --output artifacts/plan-prompt.md
```

## Feature Detection

The command detects the current feature using this priority:

1. **Environment variable**: `SPECIFY_FEATURE` if set
2. **Git branch**: Extracts from branch name matching `^\d{3}-.+$` (e.g., `103-feature-name`)
3. **Error**: Returns helpful message if detection fails

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Template loading or rendering failed |
| 3 | Invalid arguments |

## Related Commands

- [`autospec prereqs`](reference.md#internal-commands) - Show prereqs context without rendering
- [`autospec commands`](reference.md#autospec-commands) - List installed command templates

## See Also

- [YAML Structured Output](../internal/YAML-STRUCTURED-OUTPUT.md) - Command template schemas
- [Architecture](../internal/architecture.md) - System design overview
