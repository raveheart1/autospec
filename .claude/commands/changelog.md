---
description: Draft changelog entries from commits, grouped and user-friendly.
---

## User Input

```text
$ARGUMENTS
```

## Instructions

1. Read the changelog guidelines: `.release/CLAUDE.md`
2. Get commits since last tag:
   ```bash
   git log $(git describe --tags --abbrev=0 2>/dev/null || echo "HEAD~50")..HEAD --oneline
   ```
3. Review current `unreleased` section in `internal/changelog/changelog.yaml`

## YAML-First Workflow

**IMPORTANT**: All changelog edits go to `internal/changelog/changelog.yaml`. Never edit `CHANGELOG.md` directly.

The YAML structure:
```yaml
project: autospec
versions:
  - version: unreleased
    changes:
      added:
        - "New feature description"
      changed:
        - "What changed"
      fixed:
        - "Bug fix description"
      removed:
        - "Removed feature"
      deprecated:
        - "Deprecated feature"
      security:
        - "Security fix"
```

## Task

Based on commits and guidelines:

1. **Group commits** into meaningful user-facing changes (many commits → single entry)
2. **Draft entries** that are user-friendly, benefit-focused, concise
3. **Categorize** into: added, changed, fixed, removed, deprecated, security
4. **Show proposed entries** before making changes
5. **Edit `internal/changelog/changelog.yaml`** to add entries under `unreleased.changes`
6. **Run sync** to update CHANGELOG.md:
   ```bash
   make changelog-sync
   ```

Do NOT include internal refactors, test fixes, or CI changes unless they affect users.

## Argument Handling

Interpret `$ARGUMENTS` as:
- "draft" → only show proposed entries, don't edit
- "apply" → update YAML and sync to markdown directly
- Other text → additional context or instructions
- Empty → interactive mode, ask what to do

## Validation

After editing, verify the YAML is valid:
```bash
autospec artifact internal/changelog/changelog.yaml
```
