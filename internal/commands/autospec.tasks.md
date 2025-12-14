---
description: Generate YAML task breakdown from implementation plan.
version: "1.0.0"
handoffs:
  - label: Implement Tasks
    agent: autospec.implement
    prompt: Execute the implementation tasks
---

## User Input

```text
$ARGUMENTS
```

## Outline

1. **Setup**: Run `.specify/scripts/bash/check-prerequisites.sh --json --require-plan` to get FEATURE_DIR, SPEC_PATH, and PLAN_PATH.

2. **Load context**:
   - Read PLAN_PATH (`plan.yaml` or `plan.md`)
   - Read SPEC_PATH for user stories and requirements
   - Read `contracts/yaml-schemas.yaml` for `tasks_artifact` schema

3. **Generate tasks.yaml**: Create YAML output following `tasks_artifact` schema:

   ```yaml
   _meta:
     version: "1.0.0"
     generator: "autospec"
     generator_version: "<from autospec version>"
     created: "<ISO 8601 timestamp>"
     artifact_type: "tasks"

   tasks:
     branch: "<current branch>"
     spec_path: "<path to spec>"
     plan_path: "<path to plan>"

   phases:
     - number: 1
       title: "Core Infrastructure"
       description: "<phase goal>"
       tasks:
         - id: "1.1"
           title: "<task title>"
           status: "Pending"
           type: "implementation"
           dependencies: []
           acceptance_criteria:
             - "<criterion 1>"
             - "<criterion 2>"
           implementation_notes: "<notes>"

     - number: 2
       title: "Feature Implementation"
       tasks:
         - id: "2.1"
           title: "<task title>"
           status: "Pending"
           type: "test"
           dependencies: ["1.1"]
           acceptance_criteria:
             - "<criterion>"
   ```

4. **Validate output**:
   ```bash
   autospec yaml check specs/<feature>/tasks.yaml
   ```
   - If validation fails: fix YAML syntax errors and retry
   - If validation passes: proceed to report

5. **Report**: Output tasks.yaml path, phase summary, and task count.

## Key Rules

- Output MUST be valid YAML (use `autospec yaml check` to verify)
- Use `_meta.artifact_type: "tasks"` exactly
- Task types: "test", "implementation", "documentation", "refactor"
- Task status: "Pending", "InProgress", "Completed"
- Test tasks should precede their implementation tasks (test-first)
- Dependencies must reference valid task IDs
- Each phase should have a clear purpose
