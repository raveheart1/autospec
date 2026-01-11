# DAG Schema Validation

The `autospec dag` commands enable multi-spec workflow orchestration by defining dependencies between feature specifications in a YAML configuration file. This page covers the DAG schema format, validation, and visualization commands.

## Overview

A DAG (Directed Acyclic Graph) file defines how multiple feature specifications relate to each other and in what order they should be processed. Features are organized into layers, where each layer can depend on previous layers, and individual features can depend on other features.

## Quick Start

```bash
# Validate a DAG configuration file
autospec dag validate .autospec/dags/my-workflow.yaml

# Visualize the DAG structure as ASCII
autospec dag visualize .autospec/dags/my-workflow.yaml
```

## DAG File Location

DAG files are stored in `.autospec/dags/*.yaml`. This location follows the XDG-style project configuration directory pattern and keeps DAG files alongside other autospec artifacts.

## Schema Reference

### Root Structure

```yaml
schema_version: "1.0"      # Required: Schema format version
dag:
  name: "My Workflow"      # Required: Human-readable name
layers:                    # Required: At least one layer
  - ...
```

### Layer Structure

```yaml
layers:
  - id: L0                 # Required: Unique identifier (e.g., L0, L1, foundation)
    name: "Foundation"     # Optional: Human-readable name
    depends_on: []         # Optional: List of layer IDs this layer depends on
    features:              # Required: At least one feature
      - ...
```

### Feature Structure

```yaml
features:
  - id: 001-user-auth      # Required: Spec folder name (must exist in specs/<id>/)
    description: "..."     # Required: Description for spec creation
    depends_on: []         # Optional: List of feature IDs this feature depends on
    timeout: "30m"         # Optional: Override default timeout (e.g., "30m", "1h")
```

## Complete Example

```yaml
schema_version: "1.0"
dag:
  name: "E-commerce Platform"
layers:
  - id: L0
    name: "Foundation"
    features:
      - id: 001-database-schema
        description: "Core database schema and migrations"
      - id: 002-auth-system
        description: "User authentication and authorization"
  - id: L1
    name: "Core Features"
    depends_on: [L0]
    features:
      - id: 003-product-catalog
        description: "Product listing and search"
        depends_on: [001-database-schema]
      - id: 004-shopping-cart
        description: "Shopping cart functionality"
        depends_on: [001-database-schema, 002-auth-system]
  - id: L2
    name: "Checkout"
    depends_on: [L1]
    features:
      - id: 005-payment-processing
        description: "Payment gateway integration"
        depends_on: [004-shopping-cart]
        timeout: "1h"
```

## Commands

### dag validate

Validate a DAG configuration file for structural correctness.

```bash
autospec dag validate <file>
```

**What it validates:**
- YAML syntax and structure
- Required fields are present
- Layer IDs are unique
- Layer dependencies reference existing layers
- Feature IDs are unique across all layers
- Feature dependencies reference existing features
- No cycles exist in the dependency graph
- Referenced spec folders exist in `specs/<id>/`

**Exit codes:**
- `0`: Valid DAG configuration
- `1`: Validation errors found

**Example output (valid):**

```
Validating DAG: .autospec/dags/my-workflow.yaml

DAG is valid.
  Layers: 3
  Features: 5
  Dependencies: 4
```

**Example output (invalid):**

```
Validating DAG: .autospec/dags/my-workflow.yaml

Validation errors:
  line 15: feature "004-shopping-cart" depends on non-existent feature "003-products"
  line 20: missing spec folder for feature "005-payment": expected specs/005-payment/

Found 2 error(s).
```

### dag visualize

Display an ASCII diagram of the DAG structure.

```bash
autospec dag visualize <file>
```

**Example output:**

```
DAG: E-commerce Platform
========================
Layers: 3  |  Features: 5

[L0 (Foundation)]
  |- 001-database-schema
  +- 002-auth-system
    |
    v
[L1 (Core Features)]
  |- 003-product-catalog *
  +- 004-shopping-cart *
    |
    v
[L2 (Checkout)]
  +- 005-payment-processing *

Feature Dependencies:
---------------------
  003-product-catalog --> 001-database-schema
  004-shopping-cart --> 001-database-schema, 002-auth-system
  005-payment-processing --> 004-shopping-cart

Legend:
  * = has dependencies (see list above)
  --> = depends on
```

## Validation Rules

### Required Fields

| Field | Context | Description |
|-------|---------|-------------|
| `schema_version` | Root | Version of the DAG schema format |
| `dag.name` | Root | Human-readable name for the DAG |
| `layers` | Root | At least one layer required |
| `id` | Layer | Unique identifier for the layer |
| `features` | Layer | At least one feature per layer |
| `id` | Feature | Spec folder name |
| `description` | Feature | Human-readable description |

### Uniqueness Constraints

- Layer IDs must be unique within the DAG
- Feature IDs must be unique across all layers

### Dependency Validation

- Layer `depends_on` must reference existing layer IDs
- Feature `depends_on` must reference existing feature IDs
- No cycles are allowed in the dependency graph

### Spec Folder Validation

Each feature ID must correspond to an existing spec folder at `specs/<id>/`. If the folder doesn't exist, validation fails with the expected path.

## Error Messages

Error messages include line numbers for quick location of issues:

```
line 12: missing required field "description" in feature "003-api-gateway"
line 18: layer "L2" depends on non-existent layer "L99"; valid layers: [L0, L1]
line 25: duplicate feature ID "001-auth" (first defined in layer "L0", duplicate in layer "L1")
line 30: cycle detected in feature dependencies: A -> B -> C -> A
```

## Edge Cases

| Scenario | Behavior |
|----------|----------|
| Empty layers list | Validation error: at least one layer required |
| Feature depends on itself | Detected as cycle and reported |
| Cross-layer feature dependency | Valid as long as no cycle exists |
| Empty YAML file | Parse error with line number |
| Missing spec folder | Validation error with expected path |

## Best Practices

1. **Use descriptive layer names**: Layer IDs like `L0`, `L1` are concise, but add a `name` field for clarity.

2. **Keep features focused**: Each feature should represent a single, cohesive piece of functionality.

3. **Minimize cross-layer dependencies**: Prefer dependencies within the same layer or to the immediately preceding layer.

4. **Validate early**: Run `dag validate` before attempting to execute workflows.

5. **Use meaningful feature IDs**: Feature IDs should match your spec folder naming convention (e.g., `NNN-feature-name`).

## Troubleshooting

### "missing spec folder"

The feature ID doesn't correspond to an existing directory in `specs/`. Either:
- Create the spec folder: `mkdir -p specs/<feature-id>`
- Fix the feature ID to match an existing spec

### "cycle detected"

The dependency graph contains a cycle. Check the reported path and remove one of the dependencies to break the cycle.

### "non-existent layer/feature"

A `depends_on` reference points to a layer or feature that doesn't exist. Check for typos or add the missing definition.

### YAML parse errors

YAML syntax issues are reported with line and column numbers. Common issues:
- Incorrect indentation
- Missing quotes around special characters
- Unclosed strings
