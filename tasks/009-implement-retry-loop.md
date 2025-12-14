# 009 - Implement Retry Loop for /speckit.implement

**Priority:** High
**Status:** Todo
**Created:** 2024-12-13

## Problem

Claude stops after completing only some tasks during `/speckit.implement`. The retry tracking system exists but there's no actual loop that re-invokes Claude with continuation context.

## Root Causes

1. No retry loop: `ExecutePhase()` runs Claude once, validates, increments retry count, returns error
2. Unused continuation prompt: `GenerateContinuationPrompt()` is implemented but never called
3. Low default retries: `max_retries: 3` is insufficient for large task lists
4. Prompt lacks persistence: `speckit.implement.md` doesn't mandate completing ALL tasks

## Tasks

- [ ] Add retry loop in `ExecuteImplement()` that re-invokes Claude on validation failure
- [ ] Wire up `GenerateContinuationPrompt()` to provide remaining task context
- [ ] Add `max_retries_implement` config option (default: 10)
- [ ] Add `implement_prompt_suffix` config option for custom completion instructions
- [ ] Enhance `speckit.implement.md` with completion mandate
- [ ] Handle checklist blocker (Step 2 interactive prompt) - add `--force` flag
- [ ] Add `--until-complete` flag for infinite retries
- [ ] Update docs

## Implementation

**Retry loop in `internal/workflow/workflow.go`:**

```go
for {
    result, err := w.Executor.ExecutePhase(...)
    if err == nil {
        break // all tasks complete
    }
    if result.Exhausted {
        return err
    }
    phases, _ := validation.ParseTasksByPhase(tasksPath)
    prompt := validation.GenerateContinuationPrompt(specDir, "implement", phases)
    command = "/speckit.implement \"" + prompt + "\""
}
```

**Prompt enhancement for `.claude/commands/speckit.implement.md`:**

```markdown
## Critical Requirements

**COMPLETION MANDATE**: You MUST complete ALL tasks in tasks.md before stopping.
Do NOT stop after completing just some tasks. Continue until every task shows `[X]`.
```

## Related Files

- `internal/workflow/workflow.go` - `ExecuteImplement()`
- `internal/workflow/executor.go` - `ExecutePhase()`
- `internal/validation/prompt.go` - `GenerateContinuationPrompt()`
- `internal/validation/tasks.go` - `ParseTasksByPhase()`
- `internal/config/defaults.go`
- `.claude/commands/speckit.implement.md`

## References

- `docs/CLAUDE-AGENT-SDK-EVALUATION.md`
