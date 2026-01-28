# Issue: Template Variables Not Rendered in Workflow Execution

## Problem Statement

Command templates (e.g., `autospec.plan.md`) contain Go template variables like `{{.FeatureDir}}` and `{{.FeatureSpec}}` that are meant to be pre-populated before Claude sees them. However, **the workflow execution path passes raw slash commands to Claude without rendering these templates**.

Claude sees literal `{{.FeatureDir}}` text, recognizes these as "pre-computed paths" that should have been populated, and must guess where spec files are located. This results in Claude searching wrong paths like `.autospec/features/` instead of the correct `specs/` directory.

## Affected Commands

**Commands with path template variables (BROKEN):**

| Command | Template Variables Used |
|---------|------------------------|
| `autospec.plan.md` | `{{.FeatureDir}}`, `{{.FeatureSpec}}` |
| `autospec.tasks.md` | `{{.FeatureDir}}`, `{{.FeatureSpec}}`, `{{.ImplPlan}}` |
| `autospec.implement.md` | `{{.FeatureDir}}`, `{{.TasksFile}}` |
| `autospec.clarify.md` | `{{.FeatureDir}}`, `{{.FeatureSpec}}` |
| `autospec.analyze.md` | `{{.FeatureDir}}`, `{{.FeatureSpec}}` |
| `autospec.checklist.md` | `{{.FeatureDir}}`, `{{.FeatureSpec}}` |

**Commands without path variables (working):**

| Command | Template Variables Used |
|---------|------------------------|
| `autospec.specify.md` | None |
| `autospec.constitution.md` | `{{.AutospecVersion}}`, `{{.CreatedDate}}` (no paths) |

**CLI commands affected:**

- `autospec run` - uses plan, tasks, implement
- `autospec prep` - uses plan, tasks
- `autospec plan` - uses plan
- `autospec tasks` - uses tasks
- `autospec implement` - uses implement
- `autospec clarify` - uses clarify
- `autospec analyze` - uses analyze
- `autospec checklist` - uses checklist

**Essentially ALL workflow commands except `specify` and `constitution` are broken.**

## Evidence

When running `autospec run -pti`, Claude's output shows:

```
The pre-computed paths (`{{.FeatureDir}}` and `{{.FeatureSpec}}`) weren't populated
```

Claude then searches incorrect locations:
- `**/.autospec/**/spec.yaml` (wrong)
- `.autospec/features/**/*` (wrong)

Instead of the correct location:
- `specs/*/spec.yaml`

## Root Cause Analysis

### The Rendering Infrastructure Exists

All the pieces are built and working:

1. **`prereqs.ComputeContext()`** - Detects current spec from git branch, computes paths
2. **`commands.RenderTemplate()`** - Renders Go templates with context
3. **`commands.RenderAndValidate()`** - Validates requirements then renders
4. **`RequiredVars` map** - Defines which commands need which context fields

### The `render-command` CLI Works Correctly

```go
// internal/cli/render_command.go
ctx, err := prereqs.ComputeContext(opts)
rendered, err := commands.RenderAndValidate(commandName, content, ctx)
```

This chains the components correctly and produces properly rendered templates.

### The Workflow Execution Path Skips Rendering

```go
// internal/workflow/stage_executor.go
func (s *StageExecutor) buildPlanCommand(prompt string) string {
    if prompt != "" {
        command = fmt.Sprintf("/autospec.plan \"%s\"", prompt)
    } else {
        command = "/autospec.plan"
    }
    return InjectRiskAssessment(command, s.enableRiskAssessment)
}
```

This just builds the string `/autospec.plan` and passes it to Claude. **No call to `prereqs.ComputeContext()` or template rendering.**

### Claude Code Doesn't Understand Go Templates

When Claude Code executes `/autospec.plan`:
1. It reads `.claude/commands/autospec.plan.md` verbatim
2. Template variables remain as literal `{{.FeatureDir}}` text
3. Claude must infer paths from context clues and often guesses wrong

## Impact

- **Workflow failures**: Plan/tasks stages fail because Claude can't find spec files
- **Wasted tokens**: Claude spends turns searching wrong directories
- **User frustration**: Users must manually specify paths in prompts as workarounds
- **Unreliable automation**: The core value prop of autospec (automated workflows) is broken

## Why This Needs Fixing

1. **Core functionality is broken**: The specify → plan → tasks → implement pipeline is the main use case
2. **Infrastructure exists but unused**: The fix is connecting existing components, not building new ones
3. **User experience**: Every failed run wastes time and money (API costs)
4. **Design intent is clear**: Templates were designed with `{{.FeatureDir}}` variables specifically to be rendered - this was always the intended behavior

## Files Involved

- `internal/workflow/stage_executor.go` - Where rendering should be called
- `internal/prereqs/context.go` - Computes feature paths from git/env
- `internal/commands/render.go` - Template rendering logic
- `internal/cli/render_command.go` - Working example of correct usage
- `internal/commands/autospec.plan.md` - Template with unrendered variables

## Workaround

Users can manually specify paths in the prompt:

```bash
autospec run -pti "spec is in specs/002-feature-name/, plan goes in specs/002-feature-name/plan.yaml"
```

This defeats the purpose of auto-detection but works.
