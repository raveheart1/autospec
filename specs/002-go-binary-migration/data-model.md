# Data Model: Go Binary Migration

**Feature**: Go Binary Migration (002-go-binary-migration)
**Date**: 2025-10-22

This document defines the core entities, their fields, relationships, validation rules, and state transitions for the autospec CLI tool.

---

## Entity Overview

```
┌─────────────────┐
│  Configuration  │──┐
└─────────────────┘  │
                     │ references
┌─────────────────┐  │
│  RetryState     │  │
└─────────────────┘  │
                     │
┌─────────────────┐  │
│  SpecMetadata   │◄─┘
└─────────────────┘
         │
         │ contains
         ▼
┌─────────────────┐
│     Phase       │
└─────────────────┘
         │
         │ contains
         ▼
┌─────────────────┐
│      Task       │
└─────────────────┘
```

---

## 1. Configuration

Represents user settings for the autospec CLI tool.

### Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `claude_cmd` | string | No | `"claude"` | Path to Claude CLI binary |
| `claude_args` | []string | No | `["-p", "--dangerously-skip-permissions", "--verbose", "--output-format", "stream-json"]` | Default arguments for Claude CLI |
| `use_api_key` | bool | No | `false` | Whether to use API key (if false, sets `ANTHROPIC_API_KEY=""`) |
| `custom_claude_cmd` | string | No | `""` | Custom Claude command template with placeholders |
| `specify_cmd` | string | No | `"specify"` | Path to specify CLI binary |
| `max_retries` | int | No | `3` | Maximum retry attempts for validation |
| `specs_dir` | string | No | `"./specs"` | Directory containing feature specs |
| `state_dir` | string | No | `"~/.autospec/state"` | Directory for retry state persistence |
| `skip_preflight` | bool | No | `false` | Skip pre-flight validation checks |
| `timeout` | int | No | `300` | Command timeout in seconds |

### Relationships
- Configuration is loaded from global (`~/.autospec/config.json`) and local (`.autospec/config.json`) locations
- Environment variables override file-based configuration
- Referenced by all workflow commands

### Validation Rules

**Field-Level Validation:**
- `max_retries`: Must be between 1 and 10
- `timeout`: Must be between 1 and 3600 seconds
- `specs_dir`: Must be a valid directory path
- `claude_cmd`, `specify_cmd`: Must be valid executable paths or in PATH

**Template Validation:**
- `custom_claude_cmd`: Must contain `{{PROMPT}}` placeholder if specified
- `custom_claude_cmd`: Environment variable prefixes must be valid (e.g., `ANTHROPIC_API_KEY=""`)

**Override Hierarchy Validation:**
```go
// Priority: Env Vars > Local Config > Global Config > Defaults
func (c *Config) Validate() error {
    if c.max_retries < 1 || c.max_retries > 10 {
        return fmt.Errorf("max_retries must be 1-10, got %d", c.max_retries)
    }
    if c.timeout < 1 || c.timeout > 3600 {
        return fmt.Errorf("timeout must be 1-3600 seconds, got %d", c.timeout)
    }
    if c.custom_claude_cmd != "" && !strings.Contains(c.custom_claude_cmd, "{{PROMPT}}") {
        return errors.New("custom_claude_cmd must contain {{PROMPT}} placeholder")
    }
    return nil
}
```

### State Transitions
Configuration is immutable after load. Changes require:
1. Edit config file
2. Restart autospec command

### JSON Schema Example
```json
{
  "claude_cmd": "claude",
  "claude_args": ["-p", "--dangerously-skip-permissions", "--verbose", "--output-format", "stream-json"],
  "use_api_key": false,
  "custom_claude_cmd": "",
  "specify_cmd": "specify",
  "max_retries": 3,
  "specs_dir": "./specs",
  "state_dir": "~/.autospec/state",
  "skip_preflight": false,
  "timeout": 300
}
```

---

## 2. RetryState

Represents retry tracking for a specific spec and phase combination.

### Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `spec_name` | string | Yes | - | Feature spec identifier (e.g., "001", "go-binary-migration") |
| `phase` | string | Yes | - | Workflow phase (specify, plan, tasks, implement) |
| `count` | int | Yes | `0` | Current retry count |
| `last_attempt` | time.Time | Yes | - | Timestamp of last retry attempt |
| `max_retries` | int | Yes | - | Maximum retries allowed (from Configuration) |

### Relationships
- One RetryState per (spec_name, phase) tuple
- References Configuration for max_retries
- Persisted to `~/.autospec/state/retry.json`

### Validation Rules

**Field-Level Validation:**
- `spec_name`: Must not be empty
- `phase`: Must be one of: "specify", "plan", "tasks", "implement"
- `count`: Must be >= 0 and <= max_retries
- `last_attempt`: Must not be in future

**Business Logic:**
```go
func (r *RetryState) CanRetry() bool {
    return r.count < r.max_retries
}

func (r *RetryState) Increment() error {
    if !r.CanRetry() {
        return fmt.Errorf("retry limit exhausted: %d/%d", r.count, r.max_retries)
    }
    r.count++
    r.last_attempt = time.Now()
    return nil
}

func (r *RetryState) Reset() {
    r.count = 0
    r.last_attempt = time.Time{}
}
```

### State Transitions

```
┌─────────┐  Increment()  ┌──────────┐  Increment()  ┌──────────┐
│ count=0 │──────────────►│ count=1  │──────────────►│ count=2  │
└─────────┘               └──────────┘               └──────────┘
     ▲                         │                           │
     │                         │                           │
     │                         ▼                           ▼
     │                    ┌──────────┐               ┌──────────┐
     └────────────────────│ Success  │               │Exhausted │
           Reset()        │ Reset    │               │count>=max│
                          └──────────┘               └──────────┘
```

### Persistence Format
```json
{
  "retries": {
    "001:specify": {
      "spec_name": "001",
      "phase": "specify",
      "count": 2,
      "last_attempt": "2025-10-22T10:30:00Z",
      "max_retries": 3
    },
    "002:plan": {
      "spec_name": "002",
      "phase": "plan",
      "count": 0,
      "last_attempt": "2025-10-22T09:15:00Z",
      "max_retries": 3
    }
  }
}
```

---

## 3. ValidationResult

Represents the outcome of a validation check.

### Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `success` | bool | Yes | - | Whether validation passed |
| `error` | string | No | `""` | Error message if validation failed |
| `continuation_prompt` | string | No | `""` | Prompt for continuing after failure |
| `artifact_path` | string | Yes | - | Path to validated artifact |
| `checked_at` | time.Time | Yes | - | Timestamp of validation check |

### Relationships
- Produced by validation functions
- Used to decide retry or proceed logic
- Not persisted (ephemeral)

### Validation Rules

**Business Logic:**
```go
func (v *ValidationResult) ShouldRetry(retryState *RetryState) bool {
    return !v.success && retryState.CanRetry()
}

func (v *ValidationResult) ExitCode() int {
    if v.success {
        return 0 // Success
    }
    if v.error == "missing dependencies" {
        return 4 // Missing deps
    }
    if v.error == "invalid arguments" {
        return 3 // Invalid
    }
    return 1 // Failed (retryable)
}
```

### State Transitions
ValidationResult is immutable after creation.

---

## 4. SpecMetadata

Represents information about a feature specification.

### Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `name` | string | Yes | - | Feature name (e.g., "go-binary-migration") |
| `number` | string | Yes | - | Spec number (e.g., "002") |
| `directory` | string | Yes | - | Full path to spec directory |
| `branch` | string | No | `""` | Git branch name (if in git repo) |
| `spec_file` | string | Yes | - | Path to spec.md file |
| `plan_file` | string | Yes | - | Path to plan.md file |
| `tasks_file` | string | Yes | - | Path to tasks.md file |

### Relationships
- One SpecMetadata per feature
- Contains references to Phases and Tasks (loaded from tasks.md)
- Used by all validation functions

### Validation Rules

**Field-Level Validation:**
- `name`: Must not be empty, must match directory name pattern
- `number`: Must match pattern `\d{3}` (3 digits)
- `directory`: Must exist and be readable
- File paths: Must be within spec directory

**Detection Logic:**
```go
func DetectSpec(specsDir string) (*SpecMetadata, error) {
    // 1. Try git branch name (e.g., "002-go-binary-migration")
    branch, err := git.GetCurrentBranch()
    if err == nil {
        if match := specBranchPattern.FindStringSubmatch(branch); match != nil {
            number := match[1]
            name := match[2]
            return &SpecMetadata{
                number: number,
                name: name,
                directory: filepath.Join(specsDir, fmt.Sprintf("%s-%s", number, name)),
                branch: branch,
            }, nil
        }
    }

    // 2. Try most recently modified spec directory
    dirs, _ := filepath.Glob(filepath.Join(specsDir, "*-*"))
    // Find most recent...

    return nil, errors.New("no spec detected")
}
```

### State Transitions
SpecMetadata is immutable after detection.

---

## 5. Task

Represents an individual task in tasks.md.

### Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `description` | string | Yes | - | Task description text |
| `checked` | bool | Yes | - | Whether task is checked off |
| `line_number` | int | Yes | - | Line number in tasks.md |
| `phase_name` | string | Yes | - | Parent phase name |
| `indent_level` | int | No | `0` | Indentation level (for nested tasks) |

### Relationships
- Belongs to one Phase
- Contained in tasks.md file
- Multiple tasks per phase

### Validation Rules

**Parsing Rules:**
- Checked pattern: `- [x]` or `* [x]` (case-insensitive)
- Unchecked pattern: `- [ ]` or `* [ ]`
- Must be under a phase (## heading)

**Parsing Logic:**
```go
func ParseTask(line string, lineNum int, phaseName string) (*Task, error) {
    // Match: "- [ ] Task description" or "* [x] Task description"
    re := regexp.MustCompile(`^(\s*)[-*]\s+\[([ xX])\]\s+(.+)$`)
    match := re.FindStringSubmatch(line)
    if match == nil {
        return nil, errors.New("not a task line")
    }

    indent := len(match[1])
    checked := strings.ToLower(match[2]) == "x"
    desc := match[3]

    return &Task{
        description: desc,
        checked: checked,
        line_number: lineNum,
        phase_name: phaseName,
        indent_level: indent,
    }, nil
}
```

### State Transitions

```
┌───────────┐  User checks  ┌──────────┐
│ Unchecked │──────────────►│ Checked  │
│  [ ]      │   in file     │  [x]     │
└───────────┘               └──────────┘
```

Tasks are modified externally (in tasks.md file), not by autospec.

---

## 6. Phase

Represents a section in tasks.md (identified by ## heading).

### Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `name` | string | Yes | - | Phase name (from ## heading) |
| `tasks` | []Task | Yes | - | Tasks within this phase |
| `line_number` | int | Yes | - | Line number of phase heading |
| `total_tasks` | int | Yes | - | Total number of tasks in phase |
| `checked_tasks` | int | Yes | - | Number of checked tasks |

### Relationships
- Contains multiple Tasks
- Contained in SpecMetadata
- Identified by markdown ## heading

### Validation Rules

**Parsing Rules:**
- Phase heading pattern: `^##\s+(.+)$`
- All tasks between this heading and next ## heading belong to this phase

**Derived Fields:**
```go
func (p *Phase) UncheckedTasks() int {
    return p.total_tasks - p.checked_tasks
}

func (p *Phase) IsComplete() bool {
    return p.UncheckedTasks() == 0
}

func (p *Phase) Progress() float64 {
    if p.total_tasks == 0 {
        return 1.0
    }
    return float64(p.checked_tasks) / float64(p.total_tasks)
}
```

### State Transitions
Phase state derives from Task states. No direct state mutations.

---

## Implementation Status Tracking

The `autospec status` command uses this model to report progress:

```
Feature: 002-go-binary-migration
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
```

---

## Persistence Layer

### File Locations

| Entity | Storage | Format | Persistence |
|--------|---------|--------|-------------|
| Configuration | `~/.autospec/config.json`, `.autospec/config.json` | JSON | Persistent |
| RetryState | `~/.autospec/state/retry.json` | JSON | Persistent |
| ValidationResult | Memory only | - | Ephemeral |
| SpecMetadata | Derived from file system | - | Ephemeral |
| Task | `specs/<feature>/tasks.md` | Markdown | Persistent (external) |
| Phase | `specs/<feature>/tasks.md` | Markdown | Persistent (external) |

### Concurrency Considerations

**RetryState:**
- Single writer (current autospec process)
- File locking not required (single-user tool)
- Atomic write with temp file + rename pattern

**Configuration:**
- Read-only after load
- No concurrent modification expected

**Tasks.md:**
- Modified externally (by Claude or user)
- Read by autospec for validation
- No concurrent write from autospec

---

## Error Handling

### Exit Codes

| Code | Meaning | Example |
|------|---------|---------|
| 0 | Success | Validation passed, command completed |
| 1 | Failed (retryable) | Spec.md not found, validation failed |
| 2 | Retry exhausted | Max retries reached |
| 3 | Invalid arguments | Unknown command, invalid flag |
| 4 | Missing dependencies | claude or specify not in PATH |

### Error Messages

All errors should be actionable:
```go
// Bad
return errors.New("validation failed")

// Good
return fmt.Errorf("spec.md not found at %s - run 'autospec specify <description>' to create it", specPath)
```

---

## Summary

This data model provides:
1. **Configuration**: User settings with override hierarchy
2. **RetryState**: Persistent retry tracking with exhaustion detection
3. **ValidationResult**: Ephemeral validation outcomes
4. **SpecMetadata**: Feature spec detection and file path management
5. **Task/Phase**: Markdown task parsing and progress tracking

All entities support the functional requirements (FR-001 through FR-065) and maintain idempotency, validation-first approach, and clear state transitions per constitution principles.
