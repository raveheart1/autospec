# Pre-compute Prereqs Context for Slash Commands

## Summary

The `internal/commands/*.md` slash command files currently instruct the agent to run bash commands that autospec could pre-compute and inject as context. This would save agent time, guarantee accuracy, and reduce token usage.

## Pattern 1: `autospec prereqs` (Most Impactful)

Almost every command starts with a bash call to get feature paths:

| File | Current Command |
|------|----------------|
| `autospec.plan.md:17-19` | `autospec prereqs --json --require-spec` |
| `autospec.tasks.md:15-17` | `autospec prereqs --json --require-plan` |
| `autospec.implement.md:24-26` | `autospec prereqs --json --require-tasks --include-tasks` |
| `autospec.checklist.md:38-40` | `autospec prereqs --json --require-spec` |
| `autospec.clarify.md:23-25` | `autospec prereqs --json --require-spec` |
| `autospec.analyze.md:29-31` | `autospec prereqs --json --require-tasks --include-tasks` |

### Current Behavior

Agent must run:
```bash
autospec prereqs --json --require-spec
```

Then parse JSON output for `FEATURE_DIR`, `FEATURE_SPEC`, `AUTOSPEC_VERSION`, `CREATED_DATE`, etc.

### Proposed Improvement

Autospec pre-computes and injects as templated context in the prompt:

```yaml
# Auto-injected by autospec (no bash needed)
_prereqs:
  feature_dir: "specs/008-user-auth/"
  feature_spec: "specs/008-user-auth/spec.yaml"
  impl_plan: "specs/008-user-auth/plan.yaml"
  tasks_file: "specs/008-user-auth/tasks.yaml"
  autospec_version: "0.5.2"
  created_date: "2026-01-16T14:30:00Z"
  is_git_repo: true
```

### Benefits

- Eliminates 6 bash calls per workflow
- Guarantees accurate paths (no JSON parsing errors by agent)
- Reduces tokens spent on command output

---

## Pattern 2: Version/Date Info

**autospec.constitution.md:20-22** currently runs:
```bash
echo "AUTOSPEC_VERSION=$(autospec version --plain | head -1)" && echo "CREATED_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
```

### Proposed Improvement

Pre-inject `AUTOSPEC_VERSION` and `CREATED_DATE` into the prompt context. These values are known at invocation time.

---

## Pattern 3: Pre-compute `IS_GIT_REPO`

The `implement` command uses `IS_GIT_REPO` from prereqs output to decide whether to create/verify `.gitignore`.

### Current Behavior

Agent parses `IS_GIT_REPO` from prereqs JSON output.

### Proposed Improvement

Include `is_git_repo: true/false` in the pre-computed `_prereqs` context block (shown above). This is a simple `git rev-parse --is-inside-work-tree` check that autospec can do before invoking the agent.

---

## Implementation Approach

### Option A: Template Variables in Command Files

Update command files to use template variables:

```markdown
## Setup

Feature directory: {{.FeatureDir}}
Spec file: {{.SpecPath}}
Version: {{.AutospecVersion}}
Created: {{.CreatedDate}}
Is Git Repo: {{.IsGitRepo}}
```

Autospec renders these before passing to the agent.

### Option B: Inject YAML Context Block

Prepend a `_prereqs:` YAML block to the prompt that the agent can reference directly without running any bash commands.

### Option C: Environment-Style Header

```markdown
## Pre-computed Context

FEATURE_DIR=specs/008-user-auth/
FEATURE_SPEC=specs/008-user-auth/spec.yaml
AUTOSPEC_VERSION=0.5.2
CREATED_DATE=2026-01-16T14:30:00Z
IS_GIT_REPO=true
```

---

## Summary of Opportunities

| Opportunity | Commands Affected | Estimated Savings |
|-------------|-------------------|-------------------|
| Pre-compute `prereqs` JSON | 6 commands | 1 bash call + JSON parsing each |
| Pre-inject version/date | constitution, all with `_meta` | 1 bash call |
| Pre-compute `IS_GIT_REPO` | implement | Included in prereqs |

---

## Files to Modify

1. **Workflow execution code** - Pre-run prereqs and inject results
2. **Command template rendering** - Support variable substitution
3. **`internal/commands/*.md`** - Update to use injected context instead of bash commands:
   - `autospec.plan.md`
   - `autospec.tasks.md`
   - `autospec.implement.md`
   - `autospec.checklist.md`
   - `autospec.clarify.md`
   - `autospec.analyze.md`
   - `autospec.constitution.md`

## Why This Is Possible Now

GitHub SpecKit uses MD-based artifacts which can't be pre-computed - markdown is prose without predictable structure, so the agent must read and interpret files at runtime.

autospec's YAML-based artifacts have strict schemas, enabling:
- Deterministic parsing via `yaml.Unmarshal`
- Known field paths (`feature.branch`, `_meta.version`)
- CLI-side extraction before agent invocation

**YAML migration unlocked this optimization.**

---

## Priority

Medium - This is an optimization that improves reliability and speed but doesn't add new functionality.
