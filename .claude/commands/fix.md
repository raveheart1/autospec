---
description: Auto-fix common issues (format, mod tidy).
---

## Instructions

Automatically fix common code quality issues in this Go project.

## Fixes to Apply

1. **Format Go code**:
   ```bash
   go fmt ./...
   ```
   Report which files were formatted.

2. **Tidy modules**:
   ```bash
   go mod tidy
   ```

3. **Verify after fixes**:
   ```bash
   go vet ./...
   ```

## Argument Handling

Interpret `$ARGUMENTS` as:
- Empty → apply all fixes
- "fmt" → only format
- "tidy" → only go mod tidy
- "imports" → run goimports if available

## Output

Report:
- Which files were modified
- Any remaining issues that need manual attention
- "All fixed" or "Manual fixes needed for: <issues>"
