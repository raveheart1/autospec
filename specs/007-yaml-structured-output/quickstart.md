# Quickstart: YAML Structured Output

**Feature**: 007-yaml-structured-output
**Date**: 2025-12-13

This guide provides quick implementation patterns for the YAML structured output feature.

---

## 1. Installation

### Install Commands to Project

```bash
# Install autospec commands to .claude/commands/
autospec commands install

# Verify installation
autospec commands info
```

### Check for Updates

```bash
# Compare installed vs embedded versions
autospec commands check
```

---

## 2. YAML Workflow

### Generate YAML Specification

```bash
# In Claude Code, use the slash command
/autospec.specify "Add user authentication feature"
```

This creates `specs/<feature>/spec.yaml` with structured data.

### Generate YAML Plan

```bash
# After spec.yaml exists
/autospec.plan
```

Creates `specs/<feature>/plan.yaml`.

### Generate YAML Tasks

```bash
# After plan.yaml exists
/autospec.tasks
```

Creates `specs/<feature>/tasks.yaml`.

---

## 3. YAML Validation

### Syntax Check

```bash
# Validate a single YAML file
autospec yaml check specs/007-yaml-structured-output/spec.yaml

# Exit code 0 = valid, non-zero = invalid with line numbers
```

### Example Output (Valid)

```
✓ specs/007-yaml-structured-output/spec.yaml is valid YAML
```

### Example Output (Invalid)

```
✗ specs/007-yaml-structured-output/spec.yaml has errors:
  Line 15: yaml: mapping values are not allowed in this context
```

---

## 4. Working with YAML Artifacts

### Extract Specific Fields with yq

```bash
# Get all user story IDs
yq '.user_stories[].id' specs/007-yaml-structured-output/spec.yaml

# Get pending tasks
yq '.phases[].tasks[] | select(.status == "Pending")' specs/007-yaml-structured-output/tasks.yaml

# Get technical context
yq '.technical_context' specs/007-yaml-structured-output/plan.yaml
```

### Parse with Python

```python
import yaml

with open('specs/007-yaml-structured-output/spec.yaml') as f:
    spec = yaml.safe_load(f)

# Access structured data
for story in spec['user_stories']:
    print(f"{story['id']}: {story['title']} ({story['priority']})")
```

---

## 5. Artifact Structure

### spec.yaml

```yaml
_meta:
  version: "1.0.0"
  generator: "autospec"
  generator_version: "0.1.0"
  created: "2025-12-13T10:30:00Z"
  artifact_type: "spec"

feature:
  branch: "007-yaml-structured-output"
  status: "Draft"

user_stories:
  - id: "US-001"
    title: "Create YAML-Based Feature Specifications"
    priority: "P1"
    # ...

requirements:
  functional:
    - id: "FR-001"
      description: "System MUST generate spec.yaml files..."
```

### tasks.yaml

```yaml
_meta:
  artifact_type: "tasks"
  # ...

phases:
  - number: 1
    title: "Core Infrastructure"
    tasks:
      - id: "1.1"
        title: "Implement YAML validation"
        status: "Pending"
        type: "implementation"
        acceptance_criteria:
          - "Validates YAML syntax"
          - "Reports line numbers on error"
```

---

## 6. CI/CD Integration

### GitHub Actions Example

```yaml
name: Validate YAML Artifacts
on: [push, pull_request]

jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Install autospec
        run: make install

      - name: Validate YAML artifacts
        run: |
          for yaml in specs/*/spec.yaml; do
            autospec yaml check "$yaml" || exit 1
          done
```

### Pre-commit Hook

```bash
#!/bin/bash
# .git/hooks/pre-commit

for yaml in specs/*/spec.yaml specs/*/plan.yaml specs/*/tasks.yaml; do
  if [ -f "$yaml" ]; then
    autospec yaml check "$yaml" || exit 1
  fi
done
```

---

## 7. Common Operations

| Task | Command |
|------|---------|
| Install commands | `autospec commands install` |
| Check command versions | `autospec commands check` |
| Show command info | `autospec commands info` |
| Validate YAML syntax | `autospec yaml check <file>` |
| Generate spec | `/autospec.specify "<description>"` |
| Generate plan | `/autospec.plan` |
| Generate tasks | `/autospec.tasks` |

---

## 8. Troubleshooting

### "Invalid YAML syntax"

Check for:
- Incorrect indentation (use spaces, not tabs)
- Missing colons after keys
- Unquoted special characters

### "Command not found"

Ensure commands are installed:
```bash
autospec commands install
```

### "Version mismatch warning"

Update installed commands:
```bash
autospec commands install  # Overwrites with latest
```
