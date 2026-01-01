---
title: Checklists
parent: Guides
nav_order: 5
---

# Checklists

Checklists in autospec are "unit tests for requirements" — they validate the quality, clarity, and completeness of your specifications, not the implementation.

## Table of Contents

- [Purpose](#purpose)
- [Workflow Position](#workflow-position)
- [Generating Checklists](#generating-checklists)
- [Validation](#validation)
- [Implementation Gating](#implementation-gating)
- [YAML Schema](#yaml-schema)

---

## Purpose

Checklists validate **requirements quality**, not implementation correctness.

**Wrong approach** (testing implementation):
- "Verify the button clicks correctly"
- "Test error handling works"
- "Confirm the API returns 200"

**Correct approach** (testing requirements):
- "Are visual hierarchy requirements defined for all card types?" (completeness)
- "Is 'prominent display' quantified with specific sizing/positioning?" (clarity)
- "Are hover state requirements consistent across all interactive elements?" (consistency)
- "Does the spec define what happens when logo image fails to load?" (edge cases)

### Quality Dimensions

| Dimension | Question |
|-----------|----------|
| completeness | Are all necessary requirements present? |
| clarity | Are requirements unambiguous and specific? |
| consistency | Do requirements align with each other? |
| measurability | Can requirements be objectively verified? |
| coverage | Are all scenarios/edge cases addressed? |
| edge_cases | Are boundary conditions defined? |

---

## Workflow Position

Checklists are **optional** and can be generated at any point after `spec.yaml` exists:

```
specify → plan → tasks → implement
              ↓
          checklist (optional, can run anytime after spec)
```

The checklist command is offered as a next step from both `/autospec.plan` and `/autospec.implement`.

---

## Generating Checklists

Run the checklist slash command:

```bash
claude /autospec.checklist "security review"
```

The command:

1. Reads `spec.yaml`, `plan.yaml`, and `tasks.yaml` (if available)
2. Asks clarifying questions about scope, depth, and audience
3. Generates domain-specific checklist items
4. Writes to `FEATURE_DIR/checklists/<domain>.yaml`
5. Validates against schema with `autospec artifact checklist`

### Checklist Domains

Checklists are organized by domain:
- `ux.yaml` — UI/UX requirements quality
- `api.yaml` — API contract requirements quality
- `security.yaml` — Security requirements quality
- `performance.yaml` — Performance requirements quality

Multiple checklists can coexist in the `checklists/` directory.

---

## Validation

### Schema Validation

When a checklist is generated, autospec validates against the checklist schema:

```bash
autospec artifact checklist specs/NNN-feature/checklists/security.yaml
```

This validates:
- Valid YAML syntax
- Required fields present (checklist metadata, categories, items)
- Enum values correct (status: pending/pass/fail, quality_dimension, etc.)

Returns:
- Exit 0: Valid checklist
- Exit 1: Validation error with details

### Syntax-Only Validation

For quick syntax checks without schema validation:

```bash
autospec yaml check path/to/checklist.yaml
```

---

## Implementation Gating

The `/autospec.implement` command checks all checklists before starting work:

1. Scans `FEATURE_DIR/checklists/*.yaml`
2. Parses each file, counting items by status
3. Displays summary table:

```
Checklist    | Total | Passed | Pending | Pass Rate
-------------|-------|--------|---------|----------
security     |    12 |     10 |       2 |      83%
ux           |     8 |      8 |       0 |     100%
```

4. Gates implementation:
   - **All passed**: Proceeds automatically
   - **Pending/failed items**: Prompts "Do you want to proceed anyway?"

---

## YAML Schema

Checklists use this structure (defined in `internal/yaml/types.go`):

```yaml
checklist:
  feature: "Feature Name"
  branch: "NNN-feature-name"
  spec_path: "specs/NNN-feature/spec.yaml"
  domain: "security"           # ux, api, security, performance, etc.
  audience: "reviewer"         # author, reviewer, qa, release
  depth: "standard"            # lightweight, standard, comprehensive

categories:
  - name: "Requirement Completeness"
    description: "Are all necessary requirements documented?"
    items:
      - id: "CHK001"
        description: "Are authentication requirements specified for all API endpoints?"
        quality_dimension: "completeness"
        spec_reference: "FR-001"    # or null if checking for gap
        status: "pending"           # pending, pass, fail
        notes: ""

      - id: "CHK002"
        description: "Are rate limiting thresholds defined?"
        quality_dimension: "completeness"
        spec_reference: null
        status: "pending"
        notes: ""

  - name: "Requirement Clarity"
    description: "Are requirements specific and unambiguous?"
    items:
      - id: "CHK003"
        description: "Is 'secure communication' quantified with specific protocols?"
        quality_dimension: "clarity"
        spec_reference: "NFR-002"
        status: "pending"
        notes: ""

summary:
  total_items: 15
  passed: 0
  failed: 0
  pending: 15
  pass_rate: "0%"

_meta:
  version: "1.0.0"
  generator: "autospec"
  generator_version: "0.1.0"
  created: "2025-01-15T10:30:00Z"
  artifact_type: "checklist"
```

### Status Values

| Status | Meaning |
|--------|---------|
| `pending` | Not yet evaluated |
| `pass` | Requirement quality is adequate |
| `fail` | Requirement needs improvement |

### Updating Status

Checklist items are updated manually by editing the YAML file or through the review process. When reviewing a spec, change `status` from `pending` to `pass` or `fail`, and add notes explaining any issues found.
