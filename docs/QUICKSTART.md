# Autospec Quickstart

Copy-paste ready commands to get started fast.

## Installation

```bash
curl -fsSL https://raw.githubusercontent.com/ariel-frischer/autospec/main/install.sh | sh
```

## Cost Warning

> **Check your Claude auth method before long runs.** API keys (`ANTHROPIC_API_KEY`) bill per-token and can get expensive. Pro/Max plans ($20+/mo) include usage at no extra cost.
>
> Run `claude` interactively, then `/status` to see your login method.

## First-Time Setup (Once per Project)

```bash
# 1. Check dependencies (Claude Code CLI, git)
autospec doctor

# 2. Initialize autospec in your repo
autospec init

# 3. (Optional) Create project constitution
autospec constitution
```

## Core Workflows

### Full Workflow - All Stages

```bash
# Specify → Plan → Tasks → Implement (all at once)
autospec run -a "Add user authentication with OAuth"

# Or use the shortcut:
autospec all "Add user authentication with OAuth"

# Skip confirmation prompts with -y:
autospec run -a -y "Add health check endpoint at /health"
```

### Recommended: Iterative Approach

```bash
# Step 1: Generate spec only
autospec run -s "Add rate limiting to API endpoints"

# Step 2: Review/edit specs/001-rate-limiting/spec.yaml

# Step 3: Continue with plan → tasks → implement
autospec run -pti
```

### Planning Only (No Implementation)

```bash
# Generate spec, plan, and tasks (review before implementing)
autospec prep "Add caching layer for database queries"

# Then implement when ready:
autospec implement
```

## Example Feature Descriptions

```bash
# API Features
autospec all "Add a health check endpoint at /health that returns JSON status"
autospec all "Add rate limiting middleware with configurable limits per route"
autospec all "Implement pagination for all list endpoints"

# Authentication
autospec all "Add JWT authentication with refresh token support"
autospec all "Add OAuth2 login with Google and GitHub providers"

# Database
autospec all "Add database connection pooling with configurable pool size"
autospec all "Implement soft delete for user records with restore functionality"

# Testing
autospec all "Add integration tests for the payment processing module"
autospec all "Implement load testing suite with k6"

# DevOps
autospec all "Add Dockerfile with multi-stage build for production"
autospec all "Create GitHub Actions CI pipeline with test and lint stages"
```

## Monitoring Progress

```bash
# Quick status check
autospec st

# Verbose status with all details
autospec st -v

# View command history
autospec history
autospec history -n 10
```

## Implementation Modes

```bash
# Default: One session per phase
autospec implement

# Per-task isolation (recommended for complex features)
autospec implement --tasks

# Single session (for small/simple specs)
autospec implement --single-session

# Resume from specific point
autospec implement --from-phase 3
autospec implement --from-task T005
autospec implement --task T003  # Single task only
```

## Stage Flags Cheatsheet

| Flag | Stage | Description |
|------|-------|-------------|
| `-s` | specify | Generate feature specification |
| `-p` | plan | Generate implementation plan |
| `-t` | tasks | Generate task breakdown |
| `-i` | implement | Execute implementation |
| `-a` | all | All core stages (`-spti`) |
| `-r` | clarify | Refine spec with Q&A |
| `-l` | checklist | Generate validation checklist |
| `-z` | analyze | Cross-artifact consistency check |
| `-y` | - | Skip confirmation prompts |

## Common Combinations

```bash
# All core stages
autospec run -a "feature"

# Planning only
autospec run -spt "feature"   # or: autospec prep "feature"

# Specify + clarify (refine spec)
autospec run -sr "feature"

# All + checklist + analyze
autospec run -alz "feature"

# Tasks + implement (already have spec/plan)
autospec run -ti
```

## Output Structure

```
specs/
└── 001-user-auth/
    ├── spec.yaml      # Feature specification
    ├── plan.yaml      # Implementation plan
    └── tasks.yaml     # Task breakdown with status
```

## Troubleshooting

```bash
# Check all dependencies
autospec doctor

# Debug mode
autospec --debug run -a "feature"

# Show current config
autospec config show
```
