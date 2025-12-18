# Description Propagation Fix

## Problem Statement

When running `autospec run -a "feature description"`, the description argument is passed to **all four core stages** (specify, plan, tasks, implement), not just the specify stage.

**Important distinction**:
- `-a` flag only runs **core stages**: specify, plan, tasks, implement
- Optional stages (clarify, checklist, analyze, constitution) require their own flags and are NOT included in `-a`

This is problematic because:
1. When running a **multi-stage workflow** with `-a`, the feature description is meant for specify only - later stages should work from refined artifacts
2. Raw descriptions can conflict with refined specifications created by earlier stages
3. It can cause Claude to ignore structured workflow artifacts in favor of the original description

**This is NOT a problem for individual stage runs**:
- `autospec run -p "focus on security"` - CORRECT, the hint goes to plan stage only
- `autospec run -i "skip tests"` - CORRECT, the hint goes to implement stage only
- The argument is intended as stage-specific guidance when running individual stages

## Current Behavior Analysis

### Code Path 1: `autospec all "description"` (all.go)
- Calls `orchestrator.RunFullWorkflow(featureDescription, resume)`
- In `RunFullWorkflow` → `executeSpecifyPlanTasks`:
  - `executeSpecify(featureDescription)` → `/autospec.specify "description"` **gets description**
  - `executePlan(specName, "")` → `/autospec.plan` **NO description (empty string)**
  - `executeTasks(specName, "")` → `/autospec.tasks` **NO description**
- Then `executeImplementStage` → `/autospec.implement` **NO description**

**Result**: Description ONLY goes to specify stage (CORRECT behavior)

### Code Path 2: `autospec run -a "description"` (run.go)
- `-a` sets core stages only: specify, plan, tasks, implement (see run.go line 81: `stageConfig.SetAll()`)
- Creates `stageExecutionContext` with `featureDescription` stored
- Each core stage execution passes `ctx.featureDescription`:
  - Line 337: `ctx.orchestrator.ExecutePlan(ctx.specName, ctx.featureDescription)` **gets description**
  - Line 344: `ctx.orchestrator.ExecuteTasks(ctx.specName, ctx.featureDescription)` **gets description**
  - Line 361: `ctx.orchestrator.ExecuteImplement(ctx.specName, ctx.featureDescription, ...)` **gets description**

**Result**: Description goes to all four core stages (INCORRECT when using `-a`)

### Code Path 3: `autospec run -p "hint"` (run.go) - Individual Stage
- Only plan stage is selected
- `ctx.featureDescription` = "hint"
- Passes hint to plan stage

**Result**: Hint goes to plan stage only (CORRECT - this is the intended use case)

### BUG: Inconsistency between `-a` and `all`
The docs state `autospec run -a` is "equivalent to" `autospec all`, but they have different behavior regarding description propagation when running the full workflow.

## Intended Behavior

| Use Case | Command | Expected Behavior |
|----------|---------|-------------------|
| Full workflow from description | `autospec run -a "add auth"` | Description to specify only |
| Full workflow from description | `autospec all "add auth"` | Description to specify only |
| Plan with hint | `autospec run -p "focus on security"` | Hint to plan stage |
| Tasks with hint | `autospec run -t "prioritize P1"` | Hint to tasks stage |
| Implement with hint | `autospec run -i "skip docs"` | Hint to implement stage |
| Multiple stages with hint | `autospec run -ti "focus on tests"` | Hint to both stages |

The key distinction:
- **Feature description** (for `-a`/`all`): Initial input that creates the spec - should ONLY go to specify
- **Stage hint** (for individual stages): Guidance for that specific stage - should go to selected stages

## How Templates Use $ARGUMENTS

| Template | What $ARGUMENTS is | Intended Use | Problem When Feature Description Passed via `-a` |
|----------|-------------------|--------------|------------------------------------------------|
| autospec.specify.md | "The feature description" | Primary input | None - this is correct |
| autospec.plan.md | "User input to consider" | Stage-specific hint | Claude may prioritize raw description over spec.yaml |
| autospec.tasks.md | "Context for task generation" | Stage-specific hint | Claude may ignore plan.yaml structure |
| autospec.implement.md | "Context for implementation" | Stage-specific hint | Claude may implement wrong scope |

## Problematic Scenarios (when `-a` propagates description)

### Scenario 1: Vague description with detailed spec
```
Description: "add auth"
Spec: 5 user stories, 12 requirements, OAuth2 specification
Plan: Detailed JWT implementation, database schemas
Tasks: 30 tasks across 5 phases
```
**Risk**: Claude sees "add auth" in implement stage and thinks it's simple, skips steps

### Scenario 2: Description with conflicting instructions
```
Description: "add user auth using Firebase"
Spec: Refined to use custom JWT (after clarification)
Plan: Custom JWT implementation planned
```
**Risk**: Claude tries to integrate Firebase in implement despite plan specifying JWT

### Scenario 3: Description with implementation hints
```
Description: "add user auth - just use a simple password file for now"
Spec: Proper bcrypt hashing, database storage
Plan: PostgreSQL user table with password hashes
```
**Risk**: Claude implements password file in implement stage, ignoring proper plan

### Scenario 4: Multi-feature description
```
Description: "add auth AND user profiles AND settings page"
Spec: Scoped to just auth (others for later)
```
**Risk**: Claude tries to implement all three features in implement stage

## Root Cause

Semantic confusion between:
1. **Feature description**: Initial user input for creating a new spec (only for specify stage)
2. **Stage hint**: Optional guidance for any individual stage

The current `autospec run -a` conflates these - using the feature description as if it were a "hint" for all stages, when it should only go to specify.

---

## Tasks

### Task 1: Fix `autospec run -a` behavior
**File**: `internal/cli/run.go`
**Priority**: High

Modify the stage execution to differentiate between:
- Running with `-a` (full workflow): description only to specify
- Running individual stages: argument goes to selected stages

**Implementation approach**:
```go
// Add field to track if we're in full workflow mode
type stageExecutionContext struct {
    // ... existing fields ...
    isFullWorkflow bool  // true when -a flag is used
}

// In executeStages, set the flag
ctx := &stageExecutionContext{
    // ... existing ...
    isFullWorkflow: stageConfig.IsAllCoreStages(), // new method or check
}

// In executePlan, check the flag
func (ctx *stageExecutionContext) executePlan() error {
    prompt := ""
    if !ctx.isFullWorkflow {
        // Only pass description as hint when NOT running full workflow
        prompt = ctx.featureDescription
    }
    if err := ctx.orchestrator.ExecutePlan(ctx.specName, prompt); err != nil {
        return fmt.Errorf("plan stage failed: %w", err)
    }
    return nil
}

// Same pattern for executeTasks, executeImplement
```

**Acceptance criteria**:
- [ ] `autospec run -a "desc"` only passes description to specify stage
- [ ] `autospec run -p "hint"` passes hint to plan stage (individual stage use case)
- [ ] `autospec run -ti "hint"` passes hint to both tasks and implement
- [ ] `autospec run -spt "desc"` passes description to specify, then to plan, then to tasks (borderline case - user explicitly chose stages)
- [ ] Behavior of `autospec run -a` matches `autospec all`

### Task 2: Add backward compatibility flag (optional)
**File**: `internal/cli/run.go`
**Priority**: Low (only if breaking change is unacceptable)

Add a `--propagate-description` flag that preserves current behavior for users who want it.

```go
runCmd.Flags().Bool("propagate-description", false, "Pass feature description to all selected stages (legacy behavior)")
```

**Recommendation**: Skip this unless users complain. The current behavior is likely unintentional.

### Task 3: Update command templates for clarity
**Files**: `internal/commands/*.md`
**Priority**: Medium

Update templates to clarify that $ARGUMENTS is for stage-specific hints, not feature descriptions:

#### autospec.plan.md
Add after "You **MUST** consider the user input before proceeding":
```markdown
**Note**: $ARGUMENTS is for stage-specific guidance (e.g., "focus on security", "prioritize performance").
If $ARGUMENTS contains what looks like a feature description, use `spec.yaml` as your primary source instead -
the spec contains the refined, validated requirements.
```

#### autospec.tasks.md
Add after "You **MUST** consider the user input before proceeding":
```markdown
**Note**: $ARGUMENTS is for stage-specific hints (e.g., "focus on test coverage", "prioritize P1 stories").
Your primary sources are `spec.yaml` and `plan.yaml` which contain the refined requirements and technical decisions.
```

#### autospec.implement.md
Add after "You **MUST** consider the user input before proceeding":
```markdown
**Note**: $ARGUMENTS is for implementation hints (e.g., "skip documentation tasks", "focus on phase 1 only").
Your implementation guidance comes from `tasks.yaml`, `plan.yaml`, and `spec.yaml` in that priority order.
```

**Acceptance criteria**:
- [ ] Templates distinguish between stage hints and feature descriptions
- [ ] Priority hierarchy documented: artifacts > $ARGUMENTS for implementation guidance

### Task 4: Add tests for description propagation
**File**: `internal/cli/run_test.go` (new tests)
**Priority**: High

```go
func TestRunAllFlagOnlyPassesDescriptionToSpecify(t *testing.T) {
    // Test that -a flag only passes description to specify stage
    // Plan, tasks, implement should receive empty string
}

func TestRunIndividualStageReceivesPrompt(t *testing.T) {
    // Test that -p "hint" passes hint to plan stage
    // Test that -t "hint" passes hint to tasks stage
    // Test that -i "hint" passes hint to implement stage
}

func TestRunMultipleStagesReceivePrompt(t *testing.T) {
    // Test that -ti "hint" passes hint to both tasks and implement
}

func TestRunAllMatchesAllCommand(t *testing.T) {
    // Test that `run -a` behavior matches `all` command
}
```

**Acceptance criteria**:
- [ ] Test confirms description NOT passed to plan/tasks/implement when using `-a`
- [ ] Test confirms individual stage flags work with prompts
- [ ] Test confirms multi-stage selections receive prompts
- [ ] Tests document expected behavior for regression prevention

### Task 5: Update documentation
**File**: `docs/reference.md` or relevant docs
**Priority**: Low

Update documentation to clarify:
1. `autospec run -a "desc"` and `autospec all "desc"` are equivalent
2. Feature description only goes to specify stage when using `-a`
3. Stage-specific hints are for individual stage runs

**Example documentation**:
```markdown
## Stage Arguments

When running multiple stages with `-a`, the argument is the feature description and only goes to the specify stage:
  autospec run -a "Add user authentication"  # Description to specify only

When running individual stages, the argument is a stage-specific hint:
  autospec run -p "focus on security aspects"  # Hint to plan
  autospec run -i "skip documentation tasks"   # Hint to implement
  autospec run -ti "prioritize P1 stories"     # Hint to tasks and implement
```

### Task 6: Handle edge case - explicit stage selection with specify
**Priority**: Medium

Consider the edge case: `autospec run -spt "add auth"`

User explicitly selected specify, plan, and tasks (not using `-a`). Should the description propagate?

**Options**:
1. Propagate to all selected stages (current behavior when not using `-a`)
2. Only propagate to specify even with explicit selection
3. Warn user about the ambiguity

**Recommendation**: Option 1 - if user explicitly selects stages, honor that choice. The `-a` flag is the "smart" option that knows description should only go to specify.

---

## Decision Points

### Q1: Is this a breaking change?
**Analysis**: Users who run `autospec run -a "desc"` and rely on description being passed to all stages would be affected.
**Recommendation**: Yes, but current behavior is inconsistent with `autospec all` and likely unintentional. The fix aligns behavior.

### Q2: Should description EVER be passed beyond specify in `-a` mode?
**Recommendation**: No. The spec.yaml created by specify contains the refined feature description. Later stages should use artifacts, not raw input.

### Q3: What about explicit stage selection like `-spt`?
**Recommendation**: Honor user's explicit choice - pass argument to all selected stages. Only `-a` gets the "smart" behavior.

---

## Summary

| Scenario | Command | specify | plan | tasks | implement |
|----------|---------|---------|------|-------|-----------|
| Full workflow | `run -a "desc"` | "desc" | "" | "" | "" |
| Full workflow | `all "desc"` | "desc" | "" | "" | "" |
| Individual stage | `run -p "hint"` | - | "hint" | - | - |
| Individual stage | `run -i "hint"` | - | - | - | "hint" |
| Multiple stages | `run -ti "hint"` | - | - | "hint" | "hint" |
| Explicit selection | `run -spt "desc"` | "desc" | "desc" | "desc" | - |

The fix ensures `-a` behaves like `all` - description only to specify. Individual stage runs continue to work as intended with stage-specific hints.
