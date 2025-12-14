---
description: Generate YAML implementation plan from feature specification.
version: "1.0.0"
handoffs:
  - label: Create Tasks
    agent: autospec.tasks
    prompt: Generate tasks from the plan
---

## User Input

```text
$ARGUMENTS
```

## Outline

1. **Setup**: Run `.specify/scripts/bash/check-prerequisites.sh --json --require-spec` to get FEATURE_DIR and SPEC_PATH.

2. **Load context**:
   - Read SPEC_PATH (`spec.yaml` or `spec.md`)
   - Read `.specify/memory/constitution.md`
   - Read `contracts/yaml-schemas.yaml` for `plan_artifact` schema

3. **Generate plan.yaml**: Create YAML output following `plan_artifact` schema:

   ```yaml
   _meta:
     version: "1.0.0"
     generator: "autospec"
     generator_version: "<from autospec version>"
     created: "<ISO 8601 timestamp>"
     artifact_type: "plan"

   plan:
     branch: "<current branch>"
     date: "<today>"
     spec_path: "<path to spec file>"

   summary: "<1-2 paragraph summary>"

   technical_context:
     language: "<detected or specified>"
     primary_dependencies: [...]
     storage: "<storage technology>"
     testing: "<test framework>"
     target_platform: "<platforms>"
     project_type: "<single|web|mobile>"
     performance_goals: "<goals>"
     constraints: "<constraints>"
     scale_scope: "<scope>"

   constitution_check:
     gates:
       - name: "<gate name>"
         status: "PASS"
         notes: "<notes>"

   project_structure:
     documentation:
       - path: "<path>"
         description: "<what this file is>"
     source_code:
       - path: "<path>"
         description: "<what this creates>"
   ```

4. **Validate output**:
   ```bash
   autospec yaml check specs/<feature>/plan.yaml
   ```
   - If validation fails: fix YAML syntax errors and retry
   - If validation passes: proceed to report

5. **Report**: Output branch name, plan.yaml path, and readiness for `/autospec.tasks`.

## Key Rules

- Output MUST be valid YAML (use `autospec yaml check` to verify)
- All required fields per schema MUST be present
- Use `_meta.artifact_type: "plan"` exactly
- Technical context should reflect the actual project setup
- Constitution check gates should be validated against project constitution
