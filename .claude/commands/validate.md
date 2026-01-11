---
description: Run all Go quality gates (fmt, lint, test, build).
---

## Instructions

Run the full validation suite for this Go project. All checks must pass before committing.

## Checks to Run

Execute these commands **sequentially** (stop on first failure):

1. **Format** - Check for formatting issues:
   ```bash
   go fmt ./...
   ```
   If any files are modified, report which ones need formatting.

2. **Vet** - Run static analysis:
   ```bash
   go vet ./...
   ```

3. **Lint Bash** - Check shell scripts:
   ```bash
   find . -name '*.sh' -type f -not -path './.specify/*' -not -path '*/.autospec/*' -not -name 'quickstart-demo.sh' | xargs shellcheck -x --severity=warning
   ```

4. **Test** - Run all tests with race detection:
   ```bash
   go test -race -cover ./...
   ```

5. **Build** - Verify compilation:
   ```bash
   go build -o /tmp/autospec-validate-check ./cmd/autospec && rm /tmp/autospec-validate-check
   ```

## Argument Handling

Interpret `$ARGUMENTS` as:
- Empty → run all checks
- "fmt" → only format check
- "lint" → only vet + shellcheck
- "test" → only tests
- "build" → only build
- "quick" → fmt + vet + build (skip tests)

## Output

Report results clearly:
- Use checkmarks for passed checks
- Stop immediately on failure and report which check failed
- At the end, summarize: "All checks passed" or "Failed at: <check>"
