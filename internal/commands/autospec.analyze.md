---
description: Analyze cross-artifact consistency and quality.
version: "1.0.0"
---

## User Input

```text
$ARGUMENTS
```

## Outline

1. **Setup**: Run `.specify/scripts/bash/check-prerequisites.sh --json --include-tasks` to get FEATURE_DIR and available documents.

2. **Load context**:
   - Read `spec.yaml` or `spec.md` for requirements
   - Read `plan.yaml` or `plan.md` for technical design
   - Read `tasks.yaml` or `tasks.md` for implementation breakdown
   - Read `contracts/yaml-schemas.yaml` for `analysis_artifact` schema

3. **Perform analysis**: Check for:
   - **Consistency**: All spec requirements covered in plan and tasks
   - **Quality**: No ambiguous or incomplete sections
   - **Coverage**: All user stories have corresponding tasks
   - **Dependencies**: Task dependencies are valid and acyclic

4. **Generate analysis.yaml**: Create YAML output following `analysis_artifact` schema:

   ```yaml
   _meta:
     version: "1.0.0"
     generator: "autospec"
     generator_version: "<from autospec version>"
     created: "<ISO 8601 timestamp>"
     artifact_type: "analysis"

   analysis:
     spec_path: "<path to spec>"
     plan_path: "<path to plan>"
     tasks_path: "<path to tasks>"
     timestamp: "<ISO 8601 timestamp>"

   findings:
     - category: "consistency"
       severity: "warning"
       description: "FR-003 not referenced in tasks"
       location: "tasks.yaml"
       recommendation: "Add task for FR-003 implementation"

     - category: "quality"
       severity: "info"
       description: "Consider breaking down large task 2.3"
       location: "tasks.yaml:phase.2.task.3"

   summary:
     total_issues: 2
     errors: 0
     warnings: 1
     info: 1
   ```

5. **Validate output**:
   ```bash
   autospec yaml check specs/<feature>/analysis.yaml
   ```

6. **Report**: Output analysis summary with findings grouped by severity.

## Key Rules

- Output MUST be valid YAML (use `autospec yaml check` to verify)
- Use `_meta.artifact_type: "analysis"` exactly
- Categories: "consistency", "quality", "coverage"
- Severities: "error", "warning", "info"
- Summary counts must match findings
- Each finding should have actionable recommendation
