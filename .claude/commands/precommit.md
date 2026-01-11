---
description: Quick pre-commit checks (fmt, vet, build) - no tests.
---

## Instructions

Run fast pre-commit validation. Use this before committing when you're confident tests will pass.

## Checks to Run

Execute sequentially (stop on first failure):

1. **Format check**:
   ```bash
   gofmt -l .
   ```
   If any files listed, they need formatting. Report them.

2. **Vet**:
   ```bash
   go vet ./...
   ```

3. **Build**:
   ```bash
   go build -o /tmp/autospec-precommit-check ./cmd/autospec && rm /tmp/autospec-precommit-check
   ```

## Output

Quick summary:
- Report any formatting issues found
- Report any vet errors
- Confirm build succeeds
- Final: "Ready to commit" or "Issues found"
