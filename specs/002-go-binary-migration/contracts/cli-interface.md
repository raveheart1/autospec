# CLI Interface Contract

**Feature**: Go Binary Migration (002-go-binary-migration)
**Date**: 2025-10-22

This document defines the command-line interface contract for the `autospec` binary.

---

## Binary Name

`autospec` - Cross-platform binary for autospec workflow automation

---

## Global Flags

Available on all commands:

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--config` | `-c` | string | `.autospec/config.json` | Path to config file |
| `--specs-dir` | | string | `./specs` | Directory containing feature specs |
| `--skip-preflight` | | bool | `false` | Skip pre-flight validation checks |
| `--debug` | `-d` | bool | `false` | Enable debug logging |
| `--help` | `-h` | bool | `false` | Display help information |
| `--version` | `-v` | bool | `false` | Display version information |

---

## Commands

### 1. `autospec init`

Initialize autospec configuration in the current project.

**Usage:**
```bash
autospec init [flags]
```

**Flags:**
| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--global` | `-g` | bool | `false` | Create global config at ~/.autospec/config.json |
| `--force` | `-f` | bool | `false` | Overwrite existing config |

**Behavior:**
1. Detect git repository root (if in git repo)
2. Create `.autospec/config.json` (or `~/.autospec/config.json` if --global)
3. Populate with default configuration values
4. Create state directory at `~/.autospec/state/`
5. Display success message with config location

**Exit Codes:**
- `0`: Configuration created successfully
- `1`: Failed to create configuration
- `3`: Invalid flags or arguments

**Example Output:**
```
Created configuration at /home/user/project/.autospec/config.json
Created state directory at /home/user/.autospec/state/

Default configuration:
  claude_cmd: claude
  max_retries: 3
  specs_dir: ./specs

To customize, edit .autospec/config.json
```

**Contract:**
- MUST create config file at specified location
- MUST create state directory if it doesn't exist
- MUST NOT overwrite existing config unless --force is specified
- MUST validate config schema after creation

---

### 2. `autospec workflow <feature-description>`

Run complete specify → plan → tasks workflow.

**Usage:**
```bash
autospec workflow <feature-description> [flags]
```

**Arguments:**
| Argument | Required | Description |
|----------|----------|-------------|
| `feature-description` | Yes | Natural language description of feature |

**Flags:**
| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--skip-preflight` | | bool | `false` | Skip pre-flight validation checks |
| `--max-retries` | `-r` | int | 3 | Override max retry attempts |

**Behavior:**
1. Run pre-flight checks (unless --skip-preflight)
2. Execute `/speckit.specify "<feature-description>"` via `claude -p`
3. Validate spec.md exists in specs directory
4. If validation fails, retry up to max_retries times
5. Execute `/speckit.plan` via `claude -p`
6. Validate plan.md exists
7. If validation fails, retry up to max_retries times
8. Execute `/speckit.tasks` via `claude -p`
9. Validate tasks.md exists
10. If validation fails, retry up to max_retries times
11. Report completion status

**Exit Codes:**
- `0`: Complete workflow succeeded
- `1`: Phase failed (retryable)
- `2`: Retry limit exhausted
- `3`: Invalid arguments
- `4`: Missing dependencies (claude, specify, git)

**Example Output:**
```
Running pre-flight checks...
✓ claude CLI found
✓ specify CLI found
✓ .claude/commands/ directory exists
✓ .specify/ directory exists

[Phase 1/3] Specify...
Executing: /speckit.specify "Add user authentication"
✓ Created specs/003-add-user-authentication/spec.md

[Phase 2/3] Plan...
Executing: /speckit.plan
✓ Created specs/003-add-user-authentication/plan.md
✓ Created specs/003-add-user-authentication/research.md

[Phase 3/3] Tasks...
Executing: /speckit.tasks
✓ Created specs/003-add-user-authentication/tasks.md

Workflow completed successfully!
Spec: specs/003-add-user-authentication/
Next: autospec implement
```

**Contract:**
- MUST execute phases sequentially (specify → plan → tasks)
- MUST validate each phase output before proceeding
- MUST retry failed validations up to max_retries times
- MUST stream Claude output to stdout in real-time
- MUST persist retry state across command invocations
- MUST reset retry count when phase succeeds

---

### 3. `autospec specify <feature-description>`

Create feature specification.

**Usage:**
```bash
autospec specify <feature-description> [flags]
```

**Arguments:**
| Argument | Required | Description |
|----------|----------|-------------|
| `feature-description` | Yes | Natural language description of feature |

**Behavior:**
1. Execute `/speckit.specify "<feature-description>"` via `claude -p`
2. Validate spec.md exists in specs directory
3. Report spec location

**Exit Codes:**
- `0`: Spec created successfully
- `1`: Creation failed
- `4`: Missing dependencies

**Example Output:**
```
Executing: /speckit.specify "Add dark mode toggle"
✓ Created specs/004-add-dark-mode-toggle/spec.md

Next: autospec plan
```

---

### 4. `autospec plan`

Create implementation plan for current feature.

**Usage:**
```bash
autospec plan [spec-name] [flags]
```

**Arguments:**
| Argument | Required | Description |
|----------|----------|-------------|
| `spec-name` | No | Spec to plan (auto-detected if omitted) |

**Behavior:**
1. Detect current spec (from git branch or spec-name argument)
2. Execute `/speckit.plan` via `claude -p`
3. Validate plan.md, research.md exist
4. Report plan location

**Exit Codes:**
- `0`: Plan created successfully
- `1`: Creation failed
- `3`: Spec not found or invalid

**Example Output:**
```
Detected spec: 004-add-dark-mode-toggle
Executing: /speckit.plan

✓ Created specs/004-add-dark-mode-toggle/plan.md
✓ Created specs/004-add-dark-mode-toggle/research.md
✓ Created specs/004-add-dark-mode-toggle/data-model.md

Next: autospec tasks
```

---

### 5. `autospec tasks`

Generate task breakdown for current feature.

**Usage:**
```bash
autospec tasks [spec-name] [flags]
```

**Arguments:**
| Argument | Required | Description |
|----------|----------|-------------|
| `spec-name` | No | Spec to generate tasks for (auto-detected if omitted) |

**Behavior:**
1. Detect current spec
2. Execute `/speckit.tasks` via `claude -p`
3. Validate tasks.md exists
4. Count total tasks and display summary

**Exit Codes:**
- `0`: Tasks created successfully
- `1`: Creation failed
- `3`: Spec not found or invalid

**Example Output:**
```
Detected spec: 004-add-dark-mode-toggle
Executing: /speckit.tasks

✓ Created specs/004-add-dark-mode-toggle/tasks.md

Task summary:
  Phase 0: Research - 5 tasks
  Phase 1: Foundation - 8 tasks
  Phase 2: Implementation - 15 tasks
  Total: 28 tasks

Next: autospec implement
```

---

### 6. `autospec implement`

Execute implementation for current feature.

**Usage:**
```bash
autospec implement [spec-name] [flags]
```

**Arguments:**
| Argument | Required | Description |
|----------|----------|-------------|
| `spec-name` | No | Spec to implement (auto-detected if omitted) |

**Flags:**
| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--resume` | | bool | `false` | Resume from last unchecked task |

**Behavior:**
1. Detect current spec
2. Load tasks.md and find first unchecked task (or resume point)
3. Execute `/speckit.implement` via `claude -p`
4. Monitor task completion (validate all tasks checked)
5. Report progress periodically

**Exit Codes:**
- `0`: All tasks completed
- `1`: Implementation incomplete
- `3`: Spec not found or tasks.md missing

**Example Output:**
```
Detected spec: 004-add-dark-mode-toggle
Progress: 8/28 tasks completed (29%)

Executing: /speckit.implement

[Implementation in progress - streaming Claude output]

Implementation paused. Progress: 15/28 tasks (54%)
Remaining tasks: 13

To resume: autospec implement --resume
```

---

### 7. `autospec status`

Show implementation progress for current feature.

**Usage:**
```bash
autospec status [spec-name] [flags]
```

**Arguments:**
| Argument | Required | Description |
|----------|----------|-------------|
| `spec-name` | No | Spec to check status (auto-detected if omitted) |

**Flags:**
| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--verbose` | `-v` | bool | `false` | Show all tasks, not just unchecked |

**Behavior:**
1. Detect current spec
2. Parse tasks.md
3. Count checked/unchecked tasks per phase
4. Display progress summary
5. List next unchecked tasks

**Exit Codes:**
- `0`: Status displayed successfully
- `3`: Spec not found or tasks.md missing

**Example Output:**
```
Feature: 004-add-dark-mode-toggle
Status: In Progress

Phase Progress:
  Phase 0: Research        [✓] 5/5 tasks (100%)
  Phase 1: Foundation      [~] 3/8 tasks (38%)
  Phase 2: Implementation  [ ] 0/15 tasks (0%)

Overall: 8/28 tasks completed (29%)

Next unchecked tasks:
  - Phase 1: Set up Go module structure (line 42)
  - Phase 1: Implement configuration loading (line 43)
  - Phase 1: Add git operations wrapper (line 44)

To continue: autospec implement --resume
```

**Contract:**
- MUST complete in under 1 second (performance requirement)
- MUST parse tasks.md and count checked/unchecked tasks
- MUST identify phases by ## markdown headings
- MUST list next 3-5 unchecked tasks by default

---

### 8. `autospec config`

Display current configuration.

**Usage:**
```bash
autospec config [flags]
```

**Flags:**
| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--show-sources` | | bool | `false` | Show config source (global/local/env) |

**Behavior:**
1. Load configuration from all sources
2. Display merged configuration
3. If --show-sources, indicate override hierarchy

**Exit Codes:**
- `0`: Config displayed successfully
- `1`: Failed to load config

**Example Output:**
```
Configuration:

  claude_cmd: claude
  claude_args: ["-p", "--dangerously-skip-permissions", "--verbose", "--output-format", "stream-json"]
  use_api_key: false
  max_retries: 3
  specs_dir: ./specs
  state_dir: ~/.autospec/state
  skip_preflight: false

Sources:
  - Global: ~/.autospec/config.json
  - Local: .autospec/config.json (overrides global)
  - Environment: AUTOSPEC_MAX_RETRIES=3 (highest priority)

To edit: vim .autospec/config.json
```

---

### 9. `autospec version` / `autospec --version`

Display version information.

**Usage:**
```bash
autospec version
autospec --version
```

**Behavior:**
1. Display version number
2. Display build information (commit, date)
3. Display Go version

**Exit Codes:**
- `0`: Always succeeds

**Example Output:**
```
autospec version 1.0.0
Built from commit: a1b2c3d
Build date: 2025-10-22T10:30:00Z
Go version: go1.21.5
```

**Contract:**
- MUST complete in under 50ms (startup performance requirement)

---

## Pre-Flight Validation

Run before workflow commands unless `--skip-preflight` is specified.

**Checks:**
1. Verify `claude` CLI is in PATH
2. Verify `specify` CLI is in PATH
3. Verify `.claude/commands/` directory exists
4. Verify `.specify/` directory exists

**Behavior on Missing Directories:**
```
WARNING: Project not initialized with SpecKit

Missing directories:
  - .claude/commands/ (required for SpecKit slash commands)
  - .specify/ (required for SpecKit templates)

Git repository root: /home/user/project

Recommended setup:
  cd /home/user/project
  specify init . --ai claude --force

Do you want to continue anyway? [y/N]:
```

**Contract:**
- MUST complete in under 100ms
- MUST detect git root directory for helpful error messages
- MUST prompt user if directories missing (unless CI/CD environment detected)
- MUST proceed if user confirms despite warnings
- MUST be skippable with `--skip-preflight` flag

---

## Environment Variables

Override configuration values:

| Variable | Type | Overrides Config Field |
|----------|------|------------------------|
| `AUTOSPEC_CLAUDE_CMD` | string | `claude_cmd` |
| `AUTOSPEC_MAX_RETRIES` | int | `max_retries` |
| `AUTOSPEC_SPECS_DIR` | string | `specs_dir` |
| `AUTOSPEC_STATE_DIR` | string | `state_dir` |
| `AUTOSPEC_SKIP_PREFLIGHT` | bool | `skip_preflight` |
| `AUTOSPEC_DEBUG` | bool | (debug logging) |
| `ANTHROPIC_API_KEY` | string | (passed to claude if use_api_key=true) |

**Precedence:** Environment Variables > Local Config > Global Config > Defaults

---

## Exit Codes

Standardized across all commands:

| Code | Meaning | Example |
|------|---------|---------|
| 0 | Success | Command completed successfully |
| 1 | Failed (retryable) | Validation failed, file not found |
| 2 | Retry exhausted | Max retries reached |
| 3 | Invalid arguments | Unknown command, missing required arg |
| 4 | Missing dependencies | claude or specify not in PATH |

---

## Output Format

### Standard Output (stdout)
- Command results
- Status information
- Progress indicators
- Claude streaming output

### Standard Error (stderr)
- Error messages
- Warning messages
- Debug logs (when --debug enabled)

### Color Support
- Auto-detect TTY for color output
- Disable colors in non-TTY environments
- Support `NO_COLOR` environment variable

---

## Performance Contracts

From functional requirements:

| Command | Target Time | Requirement |
|---------|-------------|-------------|
| `autospec version` | <50ms | FR-005: Startup time |
| `autospec status` | <1s | FR-012, SC-006 |
| Pre-flight validation | <100ms | FR-024, SC-005 |
| Complete workflow | <5s | SC-007 (excluding Claude execution) |

---

## Example: Custom Claude Command

Configuration with custom command template:

```json
{
  "custom_claude_cmd": "ANTHROPIC_API_KEY=\"\" claude -p \"{{PROMPT}}\" | claude-clean",
  "use_api_key": false
}
```

**Behavior:**
1. Replace `{{PROMPT}}` with actual command (e.g., `/speckit.specify "feature"`)
2. Parse environment variable prefix (`ANTHROPIC_API_KEY=""`)
3. Parse pipe operator (`| claude-clean`)
4. Execute full pipeline
5. Stream output to stdout

**Contract:**
- MUST support `{{PROMPT}}` placeholder
- MUST support environment variable prefixes
- MUST support pipe operators
- MUST stream output in real-time
- MUST validate template syntax on config load

---

## Contract Validation

All commands MUST:
1. Accept global flags (`--config`, `--debug`, `--help`, `--version`)
2. Use standardized exit codes
3. Provide actionable error messages
4. Stream long-running output to stdout
5. Respect configuration override hierarchy (env > local > global > default)
6. Meet performance targets
7. Work identically on Linux, macOS, Windows

Testing strategy:
- Unit tests for flag parsing
- Integration tests for command execution
- testscript tests for CLI interface
- Benchmark tests for performance targets
