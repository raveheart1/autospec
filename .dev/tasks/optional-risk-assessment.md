# Optional Risk Assessment in Plan Stage

Make risk assessment in `plan.yaml` generation **disabled by default** with a config option to enable it.

---

## Problem Statement

Currently, the `autospec.plan` command template (`internal/commands/autospec.plan.md`) always includes the `risks:` section (lines 170-179). This:

1. **Adds cognitive overhead** for simple features that don't need formal risk analysis
2. **Increases Claude token usage** - risk assessment prompts add ~100 tokens to every plan generation
3. **May produce low-value output** - for small features, risk sections often contain boilerplate

Users should opt-in to risk assessment when needed, not have it forced.

---

## Goals

1. **Disable by default** - `enable_risk_assessment: false` in config defaults
2. **Config-driven** - Users can enable in project or user config
3. **Conditional injection** - Risk section only appears in plan prompt when enabled
4. **No template modification** - Keep embedded templates project-agnostic (per constitution)

---

## Design Decision: Injection vs Template Editing

### Option A: Template Editing at Runtime
Copy template to temp file, edit out `risks:` section, use modified file.

**Pros:**
- Can handle complex conditional sections
- Full control over template content

**Cons:**
- Violates constitution principle: "Command templates must be project-agnostic"
- Adds file I/O complexity
- Requires maintaining temp file cleanup
- Template editing is fragile (regex/string replacement on markdown)

### Option B: Sectioned Injection (Recommended)
Use the existing `InjectableInstruction` pattern. The base template is minimal (no risks), and when enabled, risk assessment instructions are injected.

**Approach:**
1. Remove `risks:` section from embedded `autospec.plan.md` template
2. Add `enable_risk_assessment` config field (default: `false`)
3. Create `BuildRiskAssessmentInstructions()` following the `BuildAutoCommitInstructions()` pattern
4. In `(*StageExecutor).buildPlanCommand()`, inject risk instructions when config enabled

**Pros:**
- Follows existing patterns (`InjectAutoCommitInstructions`)
- Templates remain agnostic
- Clean separation of concerns
- Compact display (`[+RiskAssessment]`) in verbose mode

**Cons:**
- Requires restructuring the template (moving risk section out)
- Injection is append-only; must ensure YAML schema section comes before the injected block

### Recommendation: Option B with Schema Ordering

The `autospec.plan.md` template has the YAML schema example at lines 69-187. The `risks:` section is at lines 170-174, embedded within the schema.

**Refined approach:**
1. Remove `risks:` from the embedded schema in `autospec.plan.md`
2. Create a separate injectable instruction that:
   - Adds the `risks:` schema fragment
   - Adds instructions to include risk assessment
3. Inject AFTER the main template when enabled

This requires the plan template to be restructured so the risk section can be appended rather than embedded.

---

## Implementation Plan

### Phase 1: Config Field

**Files:**
- `internal/config/config.go` - Add `EnableRiskAssessment bool`
- `internal/config/defaults.go` - Add default `"enable_risk_assessment": false`

**Changes:**
```go
// config.go
type Config struct {
    // ... existing fields ...
    EnableRiskAssessment bool `mapstructure:"enable_risk_assessment" yaml:"enable_risk_assessment"`
}

// defaults.go
"enable_risk_assessment": false,  // Risk assessment in plan.yaml disabled by default
```

**Config template comment:**
```yaml
# Risk assessment in plan.yaml
enable_risk_assessment: false     # Set true to include risks section in plan.yaml
```

### Phase 2: Risk Assessment Injection

**Files:**
- `internal/workflow/autocommit.go` → rename to `internal/workflow/injection.go` (or create new file `internal/workflow/risk_assessment.go`)

**New functions:**
```go
// risk_assessment.go

const riskAssessmentInstructions = `## Risk Assessment

Include a risks section in your plan.yaml output:

` + "```yaml" + `
risks:
  - id: "RISK-001"           # Optional, format: RISK-NNN
    risk: "Description of the risk"
    likelihood: "low"        # low | medium | high
    impact: "medium"         # low | medium | high
    mitigation: "Strategy to address the risk"
` + "```" + `

Consider:
- Technical risks (dependencies, complexity, performance)
- Integration risks (breaking changes, migrations)
- Resource risks (time, external dependencies)
- Security risks if applicable

Only include meaningful risks; skip for trivial features.
`

func BuildRiskAssessmentInstructions() InjectableInstruction {
    return InjectableInstruction{
        Name:        "RiskAssessment",
        DisplayHint: "include risks section in plan.yaml",
        Content:     riskAssessmentInstructions,
    }
}

func InjectRiskAssessment(command string, enabled bool) string {
    if !enabled {
        return command
    }
    return InjectInstructions(command, []InjectableInstruction{
        BuildRiskAssessmentInstructions(),
    })
}
```

### Phase 3: Template Modification

**Files:**
- `internal/commands/autospec.plan.md`

**Changes:**
Remove the `risks:` section from the embedded YAML schema (lines 170-174):

```yaml
# BEFORE (in schema example):
   risks:
     - risk: "<potential risk>"
       likelihood: "<low|medium|high>"
       impact: "<low|medium|high>"
       mitigation: "<how to address>"

# AFTER:
# (section removed - injected when enable_risk_assessment=true)
```

Also update the template description or add a note:
```markdown
> Note: Risk assessment is optional. Enable with `enable_risk_assessment: true` in config.
```

### Phase 4: Executor Integration

**Files:**
- `internal/workflow/stage_executor.go`

**Changes to `(*StageExecutor).buildPlanCommand()`:**

```go
func (se *StageExecutor) buildPlanCommand(prompt string, cfg *config.Config) string {
    var command string
    if prompt != "" {
        command = fmt.Sprintf("/autospec.plan \"%s\"", prompt)
    } else {
        command = "/autospec.plan"
    }

    // Inject risk assessment if enabled
    command = InjectRiskAssessment(command, cfg.EnableRiskAssessment)

    return command
}
```

**Note:** Need to pass config to `buildPlanCommand()`. Currently signature is:
```go
func (se *StageExecutor) buildPlanCommand(prompt string) string
```

Will need to update to:
```go
func (se *StageExecutor) buildPlanCommand(prompt string, cfg *config.Config) string
```

And update callers in `ExecutePlan()`.

### Phase 5: Documentation

**Files:**
- `docs/risks.md` - Add section on enabling risk assessment
- `docs/reference.md` - Document config option

**Changes:**
```markdown
# docs/risks.md

## Enabling Risk Assessment

Risk assessment in `plan.yaml` is **disabled by default**. To enable:

```bash
autospec config set enable_risk_assessment true
```

Or add to `.autospec/config.yml`:

```yaml
enable_risk_assessment: true
```

When enabled, the `/autospec.plan` command will include instructions to generate a `risks:` section.
```

**README.md addition** (in Configuration section):
```markdown
Risk assessment in plans is opt-in: `autospec config set enable_risk_assessment true`
```

---

## Files to Modify

| File | Changes |
|------|---------|
| `internal/config/config.go` | Add `EnableRiskAssessment` field |
| `internal/config/defaults.go` | Add default `false` |
| `internal/workflow/risk_assessment.go` | New file with injection logic |
| `internal/commands/autospec.plan.md` | Remove embedded `risks:` section |
| `internal/workflow/stage_executor.go` | Inject risk instructions when enabled |
| `docs/risks.md` | Document opt-in behavior |
| `docs/reference.md` | Document config option |

---

## Testing

### Unit Tests

1. **Config parsing:**
   - `TestConfigEnableRiskAssessment` - Verify field parsing from YAML
   - `TestConfigDefaultRiskAssessment` - Verify default is `false`

2. **Injection:**
   - `TestBuildRiskAssessmentInstructions` - Verify instruction struct
   - `TestInjectRiskAssessment_Enabled` - Verify injection when enabled
   - `TestInjectRiskAssessment_Disabled` - Verify no injection when disabled

3. **Command building:**
   - `TestBuildPlanCommand_WithRiskAssessment` - Verify injection in command
   - `TestBuildPlanCommand_WithoutRiskAssessment` - Verify clean command

### Integration Tests

1. Run `autospec plan` with default config → verify no `[+RiskAssessment]` in output
2. Run `autospec plan` with `enable_risk_assessment: true` → verify `[+RiskAssessment]` appears
3. Verify generated `plan.yaml` includes `risks:` section when enabled

---

## Validation Schema

The existing `internal/validation/plan.go` schema should already handle optional `risks:` section. Verify:
- `risks` field is optional (not in required fields)
- Validation passes with or without `risks:` section

---

## Migration

**Backward compatibility:**
- Existing configs without `enable_risk_assessment` get default `false`
- No breaking changes - just stops generating risk section by default
- Users wanting risks can add `enable_risk_assessment: true`

---

## Success Criteria

1. Default config produces `plan.yaml` without `risks:` section
2. `enable_risk_assessment: true` produces `plan.yaml` with `risks:` section
3. No changes to validation (risks remain optional in schema)
4. Output shows `[+RiskAssessment]` when enabled (verbose mode)
5. Documentation updated
6. CHANGELOG updated

---

## Post-Implementation

**CHANGELOG.md** (under next version):
```markdown
- Risk assessment in `plan.yaml` now opt-in; enable with `autospec config set enable_risk_assessment true`
```

---

## Related

- `internal/workflow/autocommit.go` - Injection pattern reference
- `internal/commands/autospec.plan.md` - Template to modify
- `docs/risks.md` - Current risk documentation
- `.dev/tasks/auto-commit-injection-refactor.md` - Related injection pattern work
