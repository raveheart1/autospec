---
description: Prepare a release (changelog, version bump, tag commands).
---

## User Input

```text
$ARGUMENTS
```

## Instructions

1. Read release guidelines: `.release/CLAUDE.md`
2. Check current state:
   ```bash
   git describe --tags --abbrev=0 2>/dev/null || echo "no tags yet"
   git branch --show-current
   ```
3. Review `unreleased` section in `internal/changelog/changelog.yaml`:
   ```bash
   head -50 internal/changelog/changelog.yaml
   ```

## YAML-First Workflow

**IMPORTANT**: All release updates go to `internal/changelog/changelog.yaml`. Never edit `CHANGELOG.md` directly.

## Task

1. **Verify** changelog is ready (unreleased section has entries)
2. **Confirm** version number with user (unless provided in arguments)
3. **Update `internal/changelog/changelog.yaml`**:
   - Change `version: unreleased` → `version: X.Y.Z`
   - Add `date: "YYYY-MM-DD"` (today's date)
   - Add fresh unreleased section at the top:
     ```yaml
     versions:
       - version: unreleased
         changes:
           added: []
       - version: X.Y.Z
         date: "YYYY-MM-DD"
         changes:
           ...
     ```
4. **Sync to markdown**:
   ```bash
   make changelog-sync
   ```
5. **Test extraction**:
   ```bash
   autospec changelog extract X.Y.Z
   ```
6. **Commit and push** (execute these):
   ```bash
   git add internal/changelog/changelog.yaml CHANGELOG.md
   git commit -m "chore: release vX.Y.Z"
   git checkout main
   git merge dev
   git push origin main
   git push github main
   ```
7. **Wait for CI**: Ask user to confirm CI is passing before proceeding
8. **Tag after CI passes** (only after user confirms):
   ```bash
   git tag -a vX.Y.Z -m "Release vX.Y.Z"
   git push origin main --tags
   git push github main --tags
   ```

Remember: Tag on `main` after merging AND after CI passes.

## Argument Handling

Interpret `$ARGUMENTS` as:
- Version number (e.g., "0.3.0") → use that version, skip confirmation
- "check" → only verify readiness, don't make changes
- "dry-run" → show what would happen without executing
- Other text → additional context or instructions
- Empty → interactive mode, ask what version

## Validation

After editing, verify the YAML is valid:
```bash
autospec artifact internal/changelog/changelog.yaml
```

Then verify extraction works:
```bash
autospec changelog extract X.Y.Z
```
