# Spec 8: Fix Tests Command (Future)

## Context

Part of **DAG Multi-Spec Orchestration** - a meta-orchestrator that runs multiple `autospec run` workflows in parallel across worktrees with dependency management. See [00-summary.md](00-summary.md) for full vision.

## Scope

AI-assisted test failure fixing.

## Commands

- `autospec fix-tests` - Analyze and fix failing tests
- `autospec fix-tests --dry-run` - Show proposed fixes without applying

## Key Deliverables

- Run configured test command, capture failures
- Parse failure output to identify failing tests
- Send to AI agent with context (test file, related source files)
- Generate and apply fixes
- Limit fix attempts (default 3)
- Track which tests were fixed

## Behavior

```bash
$ autospec fix-tests
Running: make test
Found 3 failing tests

[1/3] TestUserAuth_InvalidToken
  Analyzing... found issue in src/auth/token.go
  Applying fix...
  Re-running test... PASS

[2/3] TestCache_Expiry
  Analyzing... found issue in src/cache/cache.go
  Applying fix...
  Re-running test... PASS

[3/3] TestAPI_RateLimit
  Analyzing... unable to determine fix
  Skipped (manual intervention needed)

Result: 2/3 tests fixed
```

## NOT Included

- No automatic triggering (manual command only)
- No complex multi-file refactoring

## Run

```bash
autospec run -spti .dev/tasks/dag-orchestration/spec-8-fix-tests.md
```
