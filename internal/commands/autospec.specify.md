---
description: Generate YAML feature specification from natural language description.
version: "1.0.0"
handoffs:
  - label: Create Plan
    agent: autospec.plan
    prompt: Generate implementation plan from the specification
---

## User Input

```text
$ARGUMENTS
```

## Outline

1. **Setup**: Run `.specify/scripts/bash/check-prerequisites.sh --json` to get FEATURE_DIR.

2. **Load context**:
   - Read `.specify/templates/commands/specify.md` for workflow guidance
   - Read `contracts/yaml-schemas.yaml` for `spec_artifact` schema

3. **Generate spec.yaml**: Create YAML output following `spec_artifact` schema:

   ```yaml
   _meta:
     version: "1.0.0"
     generator: "autospec"
     generator_version: "<from autospec version>"
     created: "<ISO 8601 timestamp>"
     artifact_type: "spec"

   feature:
     branch: "<current branch>"
     created: "<today's date>"
     status: "Draft"
     input: "<original user description>"

   user_stories:
     - id: "US-001"
       title: "<story title>"
       priority: "P1"
       as_a: "<role>"
       i_want: "<action>"
       so_that: "<benefit>"
       acceptance_scenarios:
         - given: "<context>"
           when: "<action>"
           then: "<outcome>"

   requirements:
     functional:
       - id: "FR-001"
         description: "<requirement>"

   # ... additional sections per spec_artifact schema
   ```

4. **Validate output**:
   ```bash
   autospec yaml check specs/<feature>/spec.yaml
   ```
   - If validation fails: fix YAML syntax errors and retry
   - If validation passes: proceed to report

5. **Report**: Output feature name, spec.yaml path, and readiness for `/autospec.plan`.

## Key Rules

- Output MUST be valid YAML (use `autospec yaml check` to verify)
- All fields from schema marked `required` MUST be present
- Use `_meta.artifact_type: "spec"` exactly
- Timestamps in ISO 8601 format
- Arrays use YAML list syntax (not JSON inline)
- Include at least one user story
- Include at least one functional requirement
