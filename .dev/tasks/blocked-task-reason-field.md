# Blocked Task Reason Field - Research & Recommendations

## Problem Statement

When tasks in `tasks.yaml` are marked as `Blocked`, there is no field to capture **why** they are blocked. This makes it difficult to:
1. Understand at a glance what's preventing task completion
2. Prioritize unblocking efforts
3. Track historical reasons for blocks
4. Enable Claude to document blocking reasons during implementation

**Evidence from `specs/038-test-coverage-85/tasks.yaml`:**
```yaml
- id: "T033"
  title: "Run full test suite and verify 85% threshold for all packages"
  status: "Blocked"   # <-- No reason field!
  type: "test"
  # ... coverage thresholds not met, but this isn't captured anywhere
```

## Current State Analysis

### Schema (from `internal/validation/artifact_tasks.go`)

**Task required fields:**
- `id` - Task identifier (T001, T002, etc.)
- `title` - Task description
- `status` - Enum: `Pending`, `InProgress`, `Completed`, `Blocked`
- `type` - Enum: `setup`, `implementation`, `test`, `documentation`, `refactor`
- `parallel` - Boolean

**Optional fields:**
- `story_id` - User story reference
- `file_path` - Target file
- `dependencies` - Task ID array
- `acceptance_criteria` - String array

**Missing:** No `blocked_reason` field

### CLI Commands (`internal/cli/update_task.go`)

Current command:
```bash
autospec update-task T001 Blocked
```

**Limitations:**
- Cannot capture reason for block
- No way to add reason without manually editing YAML
- Display commands don't show blocking reasons (because they don't exist)

### Go Types (`internal/validation/tasks_yaml.go`)

```go
type TaskItem struct {
    ID                 string   `yaml:"id"`
    Title              string   `yaml:"title"`
    Status             string   `yaml:"status"`
    Type               string   `yaml:"type"`
    Parallel           bool     `yaml:"parallel"`
    StoryID            string   `yaml:"story_id,omitempty"`
    FilePath           string   `yaml:"file_path,omitempty"`
    Dependencies       []string `yaml:"dependencies"`
    AcceptanceCriteria []string `yaml:"acceptance_criteria"`
    // Missing: BlockedReason
}
```

## Recommendation 1: Add `blocked_reason` Field

### Schema Change

Add optional field to task schema:
```yaml
- id: "T033"
  title: "Run full test suite and verify 85% threshold"
  status: "Blocked"
  blocked_reason: "Coverage thresholds not met: workflow (48.8% vs 90%), completion (51.8% vs 85%)"
  type: "test"
```

### Validation Rules

1. **Optional by default** - Only shown/required when `status: "Blocked"`
2. **Warning if missing** - If `status: "Blocked"` but no `blocked_reason`, emit validation warning (not error)
3. **Ignored if status not Blocked** - If `status: "Completed"` and `blocked_reason` present, ignore or warn

### Files to Modify

| File | Change |
|------|--------|
| `internal/validation/artifact_tasks.go` | Add `blocked_reason` validation in `validateTask()` |
| `internal/validation/tasks_yaml.go` | Add `BlockedReason string` to `TaskItem` struct |
| `docs/YAML-STRUCTURED-OUTPUT.md` | Document the new field |
| `.claude/commands/autospec.implement.md` | Document using blocked_reason |

### Example Implementation in `artifact_tasks.go`

```go
// validateTask validates a single task.
func (v *TasksValidator) validateTask(...) {
    // ... existing validation ...

    // If blocked, recommend blocked_reason
    statusNode := findNode(node, "status")
    if statusNode != nil && statusNode.Value == "Blocked" {
        blockedReasonNode := findNode(node, "blocked_reason")
        if blockedReasonNode == nil || blockedReasonNode.Value == "" {
            result.AddWarning(&ValidationError{
                Path:    path + ".blocked_reason",
                Line:    getNodeLine(node),
                Message: "blocked task missing blocked_reason",
                Hint:    "Add blocked_reason to document why this task is blocked",
            })
        }
    }
}
```

## Recommendation 2: New `autospec task` Command

### Overview

Create a new parent command `autospec task` with subcommands for task management:

```bash
autospec task block T001 --reason "Waiting for API spec finalization"
autospec task unblock T001
autospec task status T001
autospec task list --blocked
```

### Command Structure

```
autospec task
  ├── block <task-id> --reason "..." [--deps T002,T003]
  ├── unblock <task-id>              # Sets to Pending, clears reason
  ├── start <task-id>                # Alias for update-task InProgress
  ├── complete <task-id>             # Alias for update-task Completed
  ├── status <task-id>               # Show task details
  └── list [--blocked|--pending|--in-progress|--completed]
```

### `autospec task block` Implementation

```bash
# Usage
autospec task block T001 --reason "Coverage gap: workflow at 48.8%, need 90%"

# Output
✓ Task T001: InProgress -> Blocked
  Reason: Coverage gap: workflow at 48.8%, need 90%
```

**Behavior:**
1. Find task by ID in tasks.yaml
2. Set `status: "Blocked"`
3. Set `blocked_reason: "<reason>"`
4. Validate and save

### `autospec task unblock` Implementation

**REQUIRED** - This command is essential for completing the block/unblock workflow.

```bash
# Usage
autospec task unblock T001

# Output
✓ Task T001: Blocked -> Pending
  Previous reason: Coverage gap: workflow at 48.8%, need 90%

# With optional new status
autospec task unblock T001 --status InProgress

# Output
✓ Task T001: Blocked -> InProgress
  Previous reason: Coverage gap: workflow at 48.8%, need 90%
```

**Behavior:**
1. Find task by ID in tasks.yaml
2. Verify current status is `Blocked` (warn if not)
3. Set `status: "Pending"` (or `--status` value if provided)
4. Preserve `blocked_reason` in output for visibility
5. Remove `blocked_reason` field from YAML (or set to empty)
6. Validate and save

**Flags:**
- `--status <status>` - Set status to something other than Pending (e.g., InProgress)
- `--keep-reason` - Preserve the blocked_reason in YAML (for historical tracking)

**Use cases:**
1. **Dependency resolved** - Blocking task completed, can now proceed
2. **Workaround found** - Issue bypassed, task can continue
3. **Requirements changed** - Block no longer applies
4. **Resuming work** - Directly to InProgress after unblock

### Files to Create/Modify

| File | Action |
|------|--------|
| `internal/cli/task.go` | New parent command |
| `internal/cli/task_block.go` | New subcommand |
| `internal/cli/task_unblock.go` | New subcommand |
| `internal/cli/task_list.go` | New subcommand |
| `internal/cli/update_task.go` | Deprecate or refactor as alias |

## Recommendation 3: Display Improvements

### `autospec st` (Status) Output

Currently shows:
```
32/35 tasks completed (91%)
5/7 task phases completed
(3 blocked)
```

Should show:
```
32/35 tasks completed (91%)
5/7 task phases completed

Blocked Tasks (3):
  T033: Run full test suite (Coverage thresholds not met)
  T034: Verify 90% critical packages (Depends on T033)
  T035: Pass quality gates (Depends on T033, T034)
```

### `autospec artifact tasks.yaml` Output

Currently shows:
```
✓ specs/038-test-coverage-85/tasks.yaml is valid

Summary:
  phases: 7
  total tasks: 35
  pending: 0
  in progress: 0
  completed: 32
  blocked: 3
```

Could show (when `--verbose` or when blocked > 0):
```
...
  blocked: 3
    - T033: Coverage thresholds not met
    - T034: (no reason specified)
    - T035: (no reason specified)
```

## Implementation Priority

| Priority | Item | Rationale |
|----------|------|-----------|
| **P1** | Add `blocked_reason` field to schema | Foundation for all other changes |
| **P1** | Add `BlockedReason` to Go structs | Required for CLI and validation |
| **P2** | `autospec task block --reason` | Enable Claude to document blocks |
| **P2** | `autospec task unblock` | **Required** - Complete block/unblock workflow |
| **P2** | Update `autospec st` to show reasons | Immediate visibility benefit |
| **P3** | `autospec task list --blocked` | Convenient filtering |

## Alternative Approaches Considered

### 1. Use `notes` Field Instead
**Rejected**: Notes are general-purpose; `blocked_reason` is semantic and enables tooling.

### 2. Embed Reason in Status
```yaml
status: "Blocked: Coverage not met"
```
**Rejected**: Breaks enum validation, harder to parse.

### 3. Separate `blocks.yaml` File
**Rejected**: Adds complexity, separates related data.

## Migration Path

1. **No migration needed** - `blocked_reason` is optional
2. **Existing blocked tasks** - Will get validation warning (not error)
3. **Backward compatible** - Old tasks.yaml files work unchanged

## Example Final State

```yaml
phases:
  - number: 6
    title: "Final Verification"
    tasks:
      - id: "T033"
        title: "Run full test suite and verify 85% threshold"
        status: "Blocked"
        blocked_reason: "5 packages below 85%: workflow (48.8%), completion (51.8%), notify (56.9%), health (83.3%), retry (83.4%)"
        type: "test"
        parallel: false
        dependencies: ["T005", "T006", "T007", ...]

      - id: "T034"
        title: "Verify 90% threshold for critical packages"
        status: "Blocked"
        blocked_reason: "Depends on T033; workflow package at 48.8% (needs 90%)"
        type: "test"
        dependencies: ["T023", "T024", ...]

      - id: "T035"
        title: "Pass all quality gates"
        status: "Blocked"
        blocked_reason: "Blocked by T033, T034"
        type: "test"
        dependencies: ["T033", "T034"]
```

## Summary

Adding a `blocked_reason` field is a small schema change with significant usability benefits:

1. **Documentation** - Captures why tasks are blocked at the source
2. **Automation** - Enables Claude to document blocks during implementation
3. **Visibility** - Status commands can show actionable information
4. **History** - Reason persists even after task is unblocked (if preserved)

The recommended approach is:
1. Add `blocked_reason` as optional field (P1)
2. Add `autospec task block --reason` command (P2)
3. Add `autospec task unblock` command (P2) - completes block/unblock workflow
4. Enhance display in `st` and `artifact` commands (P2/P3)
