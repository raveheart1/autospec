---
description: Generate or update project constitution in YAML format.
version: "1.0.0"
---

## User Input

```text
$ARGUMENTS
```

## Outline

1. **Setup**: Run `.specify/scripts/bash/check-prerequisites.sh --json` to get project root.

2. **Load context**:
   - Read existing `.specify/memory/constitution.md` if present
   - Read `contracts/yaml-schemas.yaml` for `constitution_artifact` schema
   - Read user input for new principles or updates

3. **Generate constitution.yaml**: Create YAML output following `constitution_artifact` schema:

   ```yaml
   _meta:
     version: "1.0.0"
     generator: "autospec"
     generator_version: "<from autospec version>"
     created: "<ISO 8601 timestamp>"
     artifact_type: "constitution"

   constitution:
     project_name: "<project name>"
     version: "1.0.0"
     ratified: "<date>"
     last_amended: "<date>"

   principles:
     - name: "Test-First Development"
       description: "All new code must have tests written before implementation"
       enforcement: "Pre-commit hooks and CI checks"

     - name: "Performance Standards"
       description: "Validation functions must complete in <10ms"
       enforcement: "Benchmark tests"

   sections:
     - name: "Code Quality"
       content: "All code must pass linting and formatting checks"

   governance:
     rules:
       - "Changes require review by at least one maintainer"
       - "Breaking changes require RFC process"
   ```

4. **Validate output**:
   ```bash
   autospec yaml check .specify/memory/constitution.yaml
   ```

5. **Report**: Output constitution.yaml path and principle summary.

## Key Rules

- Output MUST be valid YAML (use `autospec yaml check` to verify)
- Use `_meta.artifact_type: "constitution"` exactly
- At least one principle is required
- Each principle should have enforcement mechanism
- Preserve existing principles when updating
