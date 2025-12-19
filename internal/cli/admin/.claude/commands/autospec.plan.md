---
description: Generate YAML implementation plan from feature specification.
version: "1.0.0"
handoffs:
  - label: Create Tasks
    agent: autospec.tasks
    prompt: Generate tasks from the plan
  - label: Create Checklist
    agent: autospec.checklist
    prompt: Create a checklist for the following domain...
---

## User Input

```text
$ARGUMENTS
```

You **MUST** consider the user input before proceeding (if not empty).

## Outline

1. **Setup**: Run the prerequisites command to get feature paths:

   ```bash
   autospec prereqs --json --require-spec
   ```

   Parse the JSON output for:
   - `FEATURE_DIR`: The feature directory path
   - `FEATURE_SPEC`: Path to the spec file (spec.yaml)
   - `AUTOSPEC_VERSION`: The autospec version (for _meta section)
   - `CREATED_DATE`: ISO 8601 timestamp (for _meta section)

   If the script fails, it will output an error message instructing the user to run `/autospec.specify` first.

2. **Load context**:
   - Read the spec file at `FEATURE_SPEC`
   - Read project constitution if exists (`.autospec/memory/constitution.yaml` or `CLAUDE.md`)
   - Extract: feature description, user stories, requirements, constraints

3. **Execute plan workflow**:

   **Phase 0: Outline & Research**

   a. Identify technical unknowns from the spec:
      - For each unclear technology choice → research task
      - For each dependency → best practices research
      - For each integration → patterns research

   b. Resolve unknowns through exploration:
      - Examine existing codebase patterns
      - Consider project constraints
      - Make informed technology decisions

   c. Document research findings for inclusion in plan

   **Phase 1: Design & Architecture**

   a. Define technical context based on spec and research:
      - Language/framework (detect from existing code or choose)
      - Primary dependencies
      - Storage requirements
      - Testing approach
      - Target platform

   b. Design project structure:
      - Documentation files to create
      - Source code organization
      - Test file locations

   c. Identify data model entities from spec requirements

   d. Design API contracts if applicable

4. **Generate plan.yaml**: Create the YAML plan file with this structure:

   ```yaml
   plan:
     branch: "<current git branch>"
     created: "<today's date YYYY-MM-DD>"
     spec_path: "<relative path to spec file>"

   summary: |
     <1-2 paragraph summary of the implementation approach.
     Explain key technical decisions and how they address the spec requirements.>

   technical_context:
     language: "<primary language>"
     framework: "<framework if applicable, or 'None'>"
     primary_dependencies:
       - name: "<dependency name>"
         version: "<version constraint>"
         purpose: "<why needed>"
     storage: "<storage technology or 'None'>"
     testing:
       framework: "<test framework>"
       approach: "<unit/integration/e2e strategy>"
     target_platform: "<platform(s)>"
     project_type: "<cli|web|mobile|library|service>"
     performance_goals: "<specific targets from spec>"
     constraints:
       - "<constraint from spec or technical>"
     scale_scope: "<expected scale/scope>"

   constitution_check:
     constitution_path: "<path to constitution file or 'Not found'>"
     gates:
       - name: "<principle name from constitution>"
         status: "PASS"  # or "FAIL" or "N/A"
         notes: "<how this plan addresses the principle>"

   research_findings:
     decisions:
       - topic: "<what was researched>"
         decision: "<what was chosen>"
         rationale: "<why chosen>"
         alternatives_considered:
           - "<alternative 1>"
           - "<alternative 2>"

   data_model:
     entities:
       - name: "<entity name>"
         description: "<what it represents>"
         fields:
           - name: "<field name>"
             type: "<data type>"
             description: "<purpose>"
             constraints: "<validation rules>"
         relationships:
           - target: "<related entity>"
             type: "<one-to-many|many-to-many|etc>"
             description: "<relationship meaning>"

   api_contracts:
     endpoints:
       - method: "<HTTP method>"
         path: "<endpoint path>"
         description: "<what it does>"
         request:
           content_type: "<content type>"
           body_schema: "<inline schema or reference>"
         response:
           success_code: 200
           body_schema: "<inline schema or reference>"
         errors:
           - code: 400
             description: "<when this occurs>"

   project_structure:
     documentation:
       - path: "<relative path>"
         description: "<purpose of this file>"
     source_code:
       - path: "<relative path or pattern>"
         description: "<what this contains>"
     tests:
       - path: "<relative path or pattern>"
         description: "<what tests live here>"

   implementation_phases:
     - phase: 1
       name: "<phase name>"
       goal: "<what this phase accomplishes>"
       deliverables:
         - "<deliverable 1>"
         - "<deliverable 2>"
     - phase: 2
       name: "<phase name>"
       goal: "<what this phase accomplishes>"
       dependencies:
         - "Phase 1"
       deliverables:
         - "<deliverable>"

   risks:
     - risk: "<potential risk>"
       likelihood: "<low|medium|high>"
       impact: "<low|medium|high>"
       mitigation: "<how to address>"

   open_questions:
     - question: "<unresolved question>"
       context: "<why it matters>"
       proposed_resolution: "<suggested approach>"

   _meta:
     version: "1.0.0"
     generator: "autospec"
     generator_version: "<AUTOSPEC_VERSION from step 1>"
     created: "<CREATED_DATE from step 1>"
     artifact_type: "plan"
   ```

5. **Write the plan** to `FEATURE_DIR/plan.yaml`

6. **Validate the artifact**:
   ```bash
   autospec artifact FEATURE_DIR/plan.yaml
   ```
   - If validation fails: fix schema errors (missing required fields, invalid types) and retry
   - If validation passes: proceed to report

7. **Report**: Output:
   - Branch name
   - Full path to plan.yaml
   - Summary of technical context
   - Number of implementation phases
   - Any constitution gate failures (CRITICAL if any FAIL)
   - Readiness for `/autospec.tasks`

## Retry Context

When this command is invoked as a retry (after schema validation failed on a previous attempt), the `$ARGUMENTS` section will contain retry context **before** any user input. You MUST detect and handle this.

### Detecting Retry Context

Check if `$ARGUMENTS` starts with `RETRY X/Y` where X is the current attempt and Y is the maximum attempts:

```text
RETRY 2/3
Schema validation failed:
- missing required field: plan.branch
- invalid type for implementation_phases[0].phase: expected int, got string

<original user input follows>
```

### Handling Retry Context

When retry context is present:

1. **Parse the retry indicator**: `RETRY X/Y` tells you this is attempt X of Y maximum attempts
2. **Read the validation errors**: Each line starting with `- ` after "Schema validation failed:" is a specific error
3. **Fix the specific errors**: Address each listed schema error in your output
4. **Preserve user intent**: The original user input (if any) follows the blank line after errors
5. **Re-validate**: After generating the fixed artifact, run `autospec artifact` to verify

### Common Schema Errors to Fix

| Error | Cause | Fix |
|-------|-------|-----|
| `missing required field: X` | Field X was omitted | Add the missing field |
| `invalid enum value for X: expected one of [...]` | Wrong value for enum field | Use a valid enum value |
| `invalid type for X: expected Y, got Z` | Wrong data type | Convert to correct type |
| `plan.branch does not match pattern` | Branch name format wrong | Use `NNN-name` format |
| `missing required field: technical_context` | Core section missing | Add technical_context section |

### Example Retry Scenario

If you receive:
```text
RETRY 2/3
Schema validation failed:
- missing required field: plan.branch
- missing required field: summary
```

You should:
1. Note this is retry 2 of 3
2. Ensure `plan.branch` field is included with the correct branch name
3. Ensure `summary` field contains a description of the implementation approach

## Key Rules

- Output MUST be valid YAML (use `autospec artifact FEATURE_DIR/plan.yaml` to verify schema compliance)
- Technical context should reflect actual project setup (detect from existing code)
- Constitution gates are mandatory if constitution exists
- Research findings should document all significant technical decisions
- Data model should be derived from spec requirements
- Project structure should follow existing codebase conventions
- All YAML arrays use list syntax (not JSON inline)
- Multi-line strings use `|` or `>` block scalar style
