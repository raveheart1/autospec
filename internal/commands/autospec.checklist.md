---
description: Generate YAML checklist for feature quality validation.
version: "1.0.0"
---

## User Input

```text
$ARGUMENTS
```

## Outline

1. **Setup**: Run `.specify/scripts/bash/check-prerequisites.sh --json --require-spec` to get FEATURE_DIR and SPEC_PATH.

2. **Load context**:
   - Read SPEC_PATH (`spec.yaml` or `spec.md`) for requirements and acceptance criteria
   - Read `contracts/yaml-schemas.yaml` for `checklist_artifact` schema
   - Read user input for specific checklist focus areas

3. **Generate checklist.yaml**: Create YAML output following `checklist_artifact` schema:

   ```yaml
   _meta:
     version: "1.0.0"
     generator: "autospec"
     generator_version: "<from autospec version>"
     created: "<ISO 8601 timestamp>"
     artifact_type: "checklist"

   checklist:
     feature: "<feature name>"
     spec_path: "<path to spec>"

   categories:
     - name: "Requirements Validation"
       items:
         - id: "REQ-001"
           description: "Verify FR-001 implementation"
           checked: false
           notes: ""

     - name: "Testing"
       items:
         - id: "TEST-001"
           description: "Unit tests pass"
           checked: false

     - name: "Documentation"
       items:
         - id: "DOC-001"
           description: "API documentation updated"
           checked: false
   ```

4. **Validate output**:
   ```bash
   autospec yaml check specs/<feature>/checklist.yaml
   ```
   - If validation fails: fix YAML syntax errors and retry
   - If validation passes: proceed to report

5. **Report**: Output checklist.yaml path and category summary.

## Key Rules

- Output MUST be valid YAML (use `autospec yaml check` to verify)
- Use `_meta.artifact_type: "checklist"` exactly
- Each category should have at least one item
- Items should be derived from the spec's requirements
- All `checked` fields start as `false`
