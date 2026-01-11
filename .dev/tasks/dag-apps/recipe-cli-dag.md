# DAG App: Recipe Management CLI

A CLI tool for managing recipes, meal planning, and generating grocery lists. Each layer builds on previous work, demonstrating DAG dependencies.

## Why This is a Good DAG Example

- **Clear layer dependencies**: Each feature genuinely needs previous ones
- **Parallel opportunities**: L0 and L1 have specs that can run simultaneously
- **Optimal spec sizing**: Each spec is 5-12 tasks, completable in 2-6 hours
- **Real-world utility**: Useful tool you can actually use

## DAG File

Create `.autospec/dags/recipe-cli.yaml`:

```yaml
schema_version: "1.0"

dag:
  name: "Recipe CLI v1"

execution:
  max_parallel: 3
  timeout: "4h"
  base_branch: "main"
  on_conflict: "manual"

layers:
  - id: "L0"
    name: "Foundation"
    features:
      - id: "100-recipe-models"
        description: "Define core data models for Recipe and Ingredient with YAML serialization, validation, and a local file-based storage system using YAML files in ~/.recipe-cli/data/"

      - id: "101-cli-framework"
        description: "Set up CLI framework with Cobra, global flags (--config, --format), config file support (~/.recipe-cli/config.yaml), and help system"

  - id: "L1"
    name: "Core Operations"
    depends_on: ["L0"]
    features:
      - id: "102-recipe-crud"
        description: "Implement recipe CRUD commands: 'recipe add' (interactive or from file), 'recipe list' (with table output), 'recipe show <id>', 'recipe edit <id>', 'recipe delete <id>'"
        depends_on: ["100-recipe-models", "101-cli-framework"]

      - id: "103-ingredient-crud"
        description: "Implement ingredient CRUD commands: 'ingredient add' (name, unit, category), 'ingredient list' (grouped by category), 'ingredient show', 'ingredient delete' with usage check"
        depends_on: ["100-recipe-models", "101-cli-framework"]

      - id: "104-search-filter"
        description: "Add search and filter capabilities: 'recipe search <term>' (searches name, ingredients, tags), 'recipe list --tag=<tag>' filtering, 'recipe list --ingredient=<name>' filtering"
        depends_on: ["100-recipe-models"]

  - id: "L2"
    name: "Enhanced Features"
    depends_on: ["L1"]
    features:
      - id: "105-import-export"
        description: "Implement recipe import/export: 'recipe export <id> --format=json|yaml|markdown', 'recipe import <file>', bulk export 'recipe export-all', import from URL"
        depends_on: ["102-recipe-crud"]

      - id: "106-meal-planning"
        description: "Add meal planning: 'plan create <name>' (weekly plan), 'plan add <day> <recipe-id>', 'plan show [name]', 'plan list', store plans as separate YAML files"
        depends_on: ["102-recipe-crud"]

  - id: "L3"
    name: "Smart Features"
    depends_on: ["L2"]
    features:
      - id: "107-grocery-list"
        description: "Generate grocery lists from meal plans: 'grocery generate <plan-name>' aggregates ingredients across all recipes, groups by category, scales by servings, outputs as checklist"
        depends_on: ["106-meal-planning", "103-ingredient-crud"]

      - id: "108-nutrition"
        description: "Add nutrition calculation: store nutrition data per ingredient, 'recipe nutrition <id>' calculates totals, 'plan nutrition <name>' shows weekly totals, warn on missing data"
        depends_on: ["102-recipe-crud", "103-ingredient-crud"]

  - id: "L4"
    name: "Polish"
    depends_on: ["L3"]
    features:
      - id: "109-shell-completion"
        description: "Add shell completion for bash, zsh, fish: complete recipe names, ingredient names, plan names, tag names; install via 'recipe completion install'"
```

## Execution Order

```
L0: [100-recipe-models] ─────────┬─────────> [101-cli-framework]
                                 │
L1:              [102-recipe-crud] ←─────────┤
                 [103-ingredient-crud] ←─────┤
                 [104-search-filter] ←───────┘
                         │
L2:              [105-import-export] ←───────┤
                 [106-meal-planning] ←───────┘
                         │
L3:              [107-grocery-list] ←────────┤
                 [108-nutrition] ←───────────┘
                         │
L4:              [109-shell-completion]
```

## Validation

```bash
# Initialize project (Go module)
mkdir recipe-cli && cd recipe-cli
go mod init github.com/yourname/recipe-cli
autospec init

# Create the DAG file
mkdir -p .autospec/dags
# ... paste the YAML above into .autospec/dags/recipe-cli.yaml

# Validate the DAG
autospec dag validate .autospec/dags/recipe-cli.yaml

# Visualize dependencies
autospec dag visualize .autospec/dags/recipe-cli.yaml

# Dry run to see execution plan
autospec dag run .autospec/dags/recipe-cli.yaml --dry-run

# Execute (sequential for safety, or add --parallel)
autospec dag run .autospec/dags/recipe-cli.yaml
```

## Expected Spec Sizes

| Spec | Est. Tasks | Files | Hours |
|------|-----------|-------|-------|
| 100-recipe-models | 6-8 | 4-5 | 2-3 |
| 101-cli-framework | 5-7 | 3-4 | 2-3 |
| 102-recipe-crud | 8-12 | 4-6 | 3-5 |
| 103-ingredient-crud | 6-8 | 3-4 | 2-3 |
| 104-search-filter | 4-6 | 2-3 | 2-3 |
| 105-import-export | 6-10 | 3-5 | 3-4 |
| 106-meal-planning | 8-10 | 4-5 | 3-4 |
| 107-grocery-list | 5-8 | 2-3 | 2-3 |
| 108-nutrition | 6-8 | 3-4 | 2-3 |
| 109-shell-completion | 4-6 | 2-3 | 1-2 |

## Notes

- Each spec description is detailed enough for autospec to generate a complete spec
- Descriptions include specific commands, flags, and behaviors
- Dependencies are genuine (not artificial) - grocery list NEEDS meal planning
- Can be adapted for web app (replace CLI with REST API)
