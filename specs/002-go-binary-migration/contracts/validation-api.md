# Validation API Contract

**Feature**: Go Binary Migration (002-go-binary-migration)
**Date**: 2025-10-22

This document defines the internal validation API that replicates the bash validation library functionality.

---

## Package: `internal/validation`

Core validation functions for SpecKit workflow artifacts.

---

## Functions

### 1. `ValidateSpecFile`

Validate that spec.md exists for a given spec.

**Signature:**
```go
func ValidateSpecFile(specsDir, specName string) error
```

**Parameters:**
- `specsDir`: Base directory containing specs (e.g., `./specs`)
- `specName`: Spec identifier (e.g., `"001"`, `"go-binary-migration"`)

**Returns:**
- `nil` if spec.md exists and is readable
- `error` if validation fails

**Behavior:**
1. Construct spec directory path: `{specsDir}/{specName}-*/`
2. Check if `spec.md` exists in spec directory
3. Verify file is readable
4. Return appropriate error if not found

**Error Messages:**
```go
// Example errors
fmt.Errorf("spec directory not found: %s", specDir)
fmt.Errorf("spec.md not found in %s - run 'autospec specify <description>' to create it", specDir)
fmt.Errorf("spec.md exists but is not readable: %w", err)
```

**Contract:**
- MUST return nil only if spec.md exists and is readable
- MUST provide actionable error message with next steps
- MUST complete in <10ms for typical specs

---

### 2. `ValidatePlanFile`

Validate that plan.md exists for a given spec.

**Signature:**
```go
func ValidatePlanFile(specsDir, specName string) error
```

**Parameters:**
- `specsDir`: Base directory containing specs
- `specName`: Spec identifier

**Returns:**
- `nil` if plan.md exists and is readable
- `error` if validation fails

**Behavior:**
Similar to ValidateSpecFile but checks for plan.md

**Contract:**
- MUST return nil only if plan.md exists and is readable
- MUST provide actionable error message: "run 'autospec plan' to create it"

---

### 3. `ValidateTasksFile`

Validate that tasks.md exists for a given spec.

**Signature:**
```go
func ValidateTasksFile(specsDir, specName string) error
```

**Parameters:**
- `specsDir`: Base directory containing specs
- `specName`: Spec identifier

**Returns:**
- `nil` if tasks.md exists and is readable
- `error` if validation fails

**Contract:**
- MUST return nil only if tasks.md exists and is readable
- MUST provide actionable error message: "run 'autospec tasks' to create it"

---

### 4. `CountUncheckedTasks`

Count unchecked tasks in tasks.md.

**Signature:**
```go
func CountUncheckedTasks(tasksPath string) (int, error)
```

**Parameters:**
- `tasksPath`: Full path to tasks.md file

**Returns:**
- `count`: Number of unchecked tasks
- `error`: Error if file can't be read or parsed

**Behavior:**
1. Read tasks.md file
2. Match lines with pattern: `- [ ]` or `* [ ]` (case-insensitive for checkbox)
3. Count matching lines
4. Return count

**Pattern Matching:**
```go
// Matches:
// - [ ] Task description
// * [ ] Task description
// - [x] Completed task (should NOT count)
// * [X] Completed task (should NOT count)

uncheckedPattern := regexp.MustCompile(`(?m)^[-*]\s+\[\s\]`)
checkedPattern := regexp.MustCompile(`(?m)^[-*]\s+\[[xX]\]`)
```

**Contract:**
- MUST match both `- [ ]` and `* [ ]` patterns
- MUST be case-insensitive for checkbox marker ([x] or [X])
- MUST NOT count checked tasks
- MUST handle empty files (return 0, nil)
- MUST complete in <50ms for files up to 1000 lines

---

### 5. `ValidateTasksComplete`

Validate that all tasks in tasks.md are checked.

**Signature:**
```go
func ValidateTasksComplete(tasksPath string) error
```

**Parameters:**
- `tasksPath`: Full path to tasks.md file

**Returns:**
- `nil` if all tasks are checked
- `error` if unchecked tasks remain

**Behavior:**
1. Call CountUncheckedTasks(tasksPath)
2. If count > 0, return error with unchecked count
3. If count == 0, return nil

**Error Message:**
```go
fmt.Errorf("implementation incomplete: %d unchecked tasks remain", count)
```

**Contract:**
- MUST return nil only if all tasks checked
- MUST report unchecked task count in error

---

### 6. `ParseTasksByPhase`

Parse tasks.md and group tasks by phase.

**Signature:**
```go
func ParseTasksByPhase(tasksPath string) ([]Phase, error)
```

**Parameters:**
- `tasksPath`: Full path to tasks.md file

**Returns:**
- `phases`: Slice of Phase structs (see data-model.md)
- `error`: Error if file can't be read or parsed

**Behavior:**
1. Read tasks.md file
2. Identify phase headers (## headings)
3. Parse tasks under each phase
4. Count checked/unchecked tasks per phase
5. Return Phase structs

**Phase Detection:**
```go
// Match: ## Phase 0: Research
phasePattern := regexp.MustCompile(`^##\s+(.+)$`)
```

**Task Parsing:**
```go
// Match: - [ ] Task description
// Capture: indent, checked status, description
taskPattern := regexp.MustCompile(`^(\s*)[-*]\s+\[([ xX])\]\s+(.+)$`)
```

**Contract:**
- MUST identify phases by ## markdown headings
- MUST associate tasks with their parent phase
- MUST count total and checked tasks per phase
- MUST preserve line numbers for tasks
- MUST handle nested tasks (different indent levels)
- MUST complete in <100ms for typical tasks.md files

---

### 7. `ListIncompletePhasesWithTasks`

List phases that have unchecked tasks.

**Signature:**
```go
func ListIncompletePhasesWithTasks(tasksPath string, maxTasks int) ([]PhaseWithTasks, error)
```

**Parameters:**
- `tasksPath`: Full path to tasks.md file
- `maxTasks`: Maximum unchecked tasks to include per phase

**Returns:**
- `phaseWithTasks`: Slice of PhaseWithTasks structs
- `error`: Error if parsing fails

**PhaseWithTasks Structure:**
```go
type PhaseWithTasks struct {
    PhaseName      string
    UncheckedCount int
    UncheckedTasks []Task // Up to maxTasks items
}
```

**Behavior:**
1. Call ParseTasksByPhase(tasksPath)
2. Filter to phases with unchecked tasks
3. For each phase, include first N unchecked tasks (up to maxTasks)
4. Return filtered list

**Contract:**
- MUST only include phases with unchecked tasks
- MUST limit unchecked tasks to maxTasks per phase
- MUST preserve task order from file

---

### 8. `GenerateContinuationPrompt`

Generate a continuation prompt for incomplete work.

**Signature:**
```go
func GenerateContinuationPrompt(tasksPath string) (string, error)
```

**Parameters:**
- `tasksPath`: Full path to tasks.md file

**Returns:**
- `prompt`: Formatted continuation prompt
- `error`: Error if parsing fails

**Behavior:**
1. Call ListIncompletePhasesWithTasks(tasksPath, 5)
2. Format as continuation prompt
3. Include phase name, unchecked count, and first few tasks

**Example Output:**
```
Implementation incomplete. Please continue with the following tasks:

Phase 1: Foundation (5 unchecked tasks)
  - Set up Go module structure (line 42)
  - Implement configuration loading (line 43)
  - Add git operations wrapper (line 44)
  - Create validation package (line 45)
  - Write unit tests for config (line 46)

Phase 2: Implementation (15 unchecked tasks)
  - Implement CLI command structure (line 52)
  - Add workflow orchestration (line 53)
  - Implement retry logic (line 54)
  - Add pre-flight validation (line 55)
  - Write CLI integration tests (line 56)
```

**Contract:**
- MUST list phases with incomplete work
- MUST include unchecked task count per phase
- MUST show first 5 tasks per phase with line numbers
- MUST be suitable for passing to Claude as context

---

## Package: `internal/retry`

Retry state management for workflow validation.

---

## Functions

### 1. `LoadRetryState`

Load retry state from persistent storage.

**Signature:**
```go
func LoadRetryState(stateDir, specName, phase string) (*RetryState, error)
```

**Parameters:**
- `stateDir`: Directory containing retry state (e.g., `~/.autospec/state`)
- `specName`: Spec identifier
- `phase`: Workflow phase (specify, plan, tasks, implement)

**Returns:**
- `retryState`: RetryState struct (see data-model.md)
- `error`: Error if loading fails

**Behavior:**
1. Construct state file path: `{stateDir}/retry.json`
2. Read JSON file
3. Parse retry state for (specName, phase) key
4. If not found, return new RetryState with count=0
5. Return RetryState

**Contract:**
- MUST return new RetryState if file doesn't exist
- MUST return new RetryState if (specName, phase) not in file
- MUST parse JSON safely (handle corrupted files)
- MUST complete in <10ms

---

### 2. `SaveRetryState`

Save retry state to persistent storage.

**Signature:**
```go
func SaveRetryState(stateDir string, state *RetryState) error
```

**Parameters:**
- `stateDir`: Directory containing retry state
- `state`: RetryState to save

**Returns:**
- `error`: Error if save fails

**Behavior:**
1. Load existing retry.json file
2. Update entry for (state.SpecName, state.Phase)
3. Write to temp file
4. Atomic rename to retry.json

**Atomic Write Pattern:**
```go
// 1. Write to temp file
tmpFile := filepath.Join(stateDir, "retry.json.tmp")
ioutil.WriteFile(tmpFile, data, 0644)

// 2. Atomic rename
os.Rename(tmpFile, filepath.Join(stateDir, "retry.json"))
```

**Contract:**
- MUST use atomic write (temp file + rename)
- MUST preserve other retry states in file
- MUST create state directory if it doesn't exist
- MUST handle concurrent access gracefully (though not expected)

---

### 3. `IncrementRetryCount`

Increment retry count for a spec/phase.

**Signature:**
```go
func IncrementRetryCount(stateDir, specName, phase string, maxRetries int) (*RetryState, error)
```

**Parameters:**
- `stateDir`: Directory containing retry state
- `specName`: Spec identifier
- `phase`: Workflow phase
- `maxRetries`: Maximum retry attempts allowed

**Returns:**
- `retryState`: Updated RetryState
- `error`: Error if max retries exceeded or save fails

**Behavior:**
1. Load retry state for (specName, phase)
2. Check if count >= maxRetries
3. If exhausted, return error with exhausted code
4. Increment count
5. Update last_attempt timestamp
6. Save retry state
7. Return updated state

**Error on Exhaustion:**
```go
if state.count >= maxRetries {
    return nil, &RetryExhaustedError{
        SpecName:   specName,
        Phase:      phase,
        Count:      state.count,
        MaxRetries: maxRetries,
    }
}
```

**Contract:**
- MUST return RetryExhaustedError if max retries exceeded
- MUST update timestamp on increment
- MUST persist state before returning
- MUST be idempotent (safe to call multiple times)

---

### 4. `ResetRetryCount`

Reset retry count for a spec/phase after success.

**Signature:**
```go
func ResetRetryCount(stateDir, specName, phase string) error
```

**Parameters:**
- `stateDir`: Directory containing retry state
- `specName`: Spec identifier
- `phase`: Workflow phase

**Returns:**
- `error`: Error if reset fails

**Behavior:**
1. Load retry state for (specName, phase)
2. Set count = 0
3. Clear last_attempt timestamp
4. Save retry state

**Contract:**
- MUST set count to 0
- MUST clear timestamp
- MUST persist state before returning
- MUST be idempotent

---

## Package: `internal/git`

Git operations for spec detection.

---

## Functions

### 1. `GetCurrentBranch`

Get current git branch name.

**Signature:**
```go
func GetCurrentBranch() (string, error)
```

**Returns:**
- `branchName`: Current branch name
- `error`: Error if not in git repo or command fails

**Implementation:**
```go
cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
output, err := cmd.Output()
if err != nil {
    return "", err
}
return strings.TrimSpace(string(output)), nil
```

**Contract:**
- MUST return branch name without whitespace
- MUST return error if not in git repository
- MUST return error if git not in PATH
- MUST complete in <50ms

---

### 2. `GetRepositoryRoot`

Get git repository root directory.

**Signature:**
```go
func GetRepositoryRoot() (string, error)
```

**Returns:**
- `rootPath`: Absolute path to repository root
- `error`: Error if not in git repo or command fails

**Implementation:**
```go
cmd := exec.Command("git", "rev-parse", "--show-toplevel")
output, err := cmd.Output()
if err != nil {
    return "", err
}
return strings.TrimSpace(string(output)), nil
```

**Contract:**
- MUST return absolute path to repo root
- MUST return error if not in git repository
- MUST complete in <50ms

---

### 3. `IsGitRepository`

Check if current directory is in a git repository.

**Signature:**
```go
func IsGitRepository() bool
```

**Returns:**
- `true` if in git repository
- `false` otherwise

**Implementation:**
```go
cmd := exec.Command("git", "rev-parse", "--git-dir")
return cmd.Run() == nil
```

**Contract:**
- MUST return true only if in git repository
- MUST NOT panic or return error
- MUST complete in <50ms

---

## Package: `internal/spec`

Spec detection and metadata.

---

## Functions

### 1. `DetectCurrentSpec`

Detect current spec from git branch or directory.

**Signature:**
```go
func DetectCurrentSpec(specsDir string) (*SpecMetadata, error)
```

**Parameters:**
- `specsDir`: Base directory containing specs

**Returns:**
- `metadata`: SpecMetadata struct (see data-model.md)
- `error`: Error if spec can't be detected

**Behavior:**
1. Try git branch name (pattern: `\d{3}-[a-z-]+`)
2. If match, extract spec number and name
3. Verify spec directory exists in specsDir
4. If git detection fails, try most recently modified spec directory
5. Return SpecMetadata

**Branch Pattern:**
```go
branchPattern := regexp.MustCompile(`^(\d{3})-([a-z0-9-]+)$`)
// Matches: 002-go-binary-migration
// Groups: [1]="002", [2]="go-binary-migration"
```

**Contract:**
- MUST try git branch detection first
- MUST fall back to directory modification time
- MUST verify spec directory exists
- MUST return error if no spec found
- MUST populate all SpecMetadata fields

---

### 2. `GetSpecDirectory`

Get spec directory path for a given spec name.

**Signature:**
```go
func GetSpecDirectory(specsDir, specName string) (string, error)
```

**Parameters:**
- `specsDir`: Base directory containing specs
- `specName`: Spec identifier (e.g., "001", "go-binary-migration")

**Returns:**
- `specDir`: Full path to spec directory
- `error`: Error if spec directory not found

**Behavior:**
1. Try exact match: `{specsDir}/{specName}/`
2. Try pattern match: `{specsDir}/{specName}-*/` or `{specsDir}/*-{specName}/`
3. Return first match

**Contract:**
- MUST support numeric spec names ("001")
- MUST support feature name ("go-binary-migration")
- MUST support full spec directory name ("002-go-binary-migration")
- MUST return error if no match found

---

## Error Types

### RetryExhaustedError

Indicates retry limit has been reached.

```go
type RetryExhaustedError struct {
    SpecName   string
    Phase      string
    Count      int
    MaxRetries int
}

func (e *RetryExhaustedError) Error() string {
    return fmt.Sprintf("retry limit exhausted for %s:%s (%d/%d attempts)",
        e.SpecName, e.Phase, e.Count, e.MaxRetries)
}

func (e *RetryExhaustedError) ExitCode() int {
    return 2 // Exhausted exit code
}
```

---

## Performance Contracts

All validation functions must meet performance targets:

| Function | Target Time | Requirement |
|----------|-------------|-------------|
| ValidateSpecFile | <10ms | File existence check |
| CountUncheckedTasks | <50ms | Parse tasks.md (up to 1000 lines) |
| ParseTasksByPhase | <100ms | Full tasks.md parsing |
| LoadRetryState | <10ms | Read JSON file |
| GetCurrentBranch | <50ms | Git command execution |
| DetectCurrentSpec | <100ms | Git + file system operations |

---

## Testing Requirements

All validation functions MUST have:
1. **Unit tests** with table-driven test pattern
2. **Edge case tests** (empty files, corrupted data, missing files)
3. **Performance benchmarks** to verify contracts
4. **Mock file system tests** using testing/fstest.MapFS
5. **Mock external command tests** using TestMain hijacking pattern

Example test structure:
```go
func TestValidateSpecFile(t *testing.T) {
    tests := map[string]struct {
        fs       fstest.MapFS
        specName string
        wantErr  bool
    }{
        "spec exists": {
            fs: fstest.MapFS{
                "specs/001-feature/spec.md": {Data: []byte("# Spec")},
            },
            specName: "001",
            wantErr:  false,
        },
        "spec missing": {
            fs:       fstest.MapFS{},
            specName: "999",
            wantErr:  true,
        },
    }

    for name, tc := range tests {
        t.Run(name, func(t *testing.T) {
            t.Parallel()
            err := ValidateSpecFile(tc.fs, "specs", tc.specName)
            if (err != nil) != tc.wantErr {
                t.Errorf("error = %v, wantErr %v", err, tc.wantErr)
            }
        })
    }
}
```

---

## Summary

This validation API provides:
1. **File validation**: Verify spec.md, plan.md, tasks.md exist
2. **Task parsing**: Count unchecked tasks, parse by phase, generate continuation prompts
3. **Retry management**: Load, save, increment, reset retry state
4. **Git operations**: Branch detection, repo root, repository check
5. **Spec detection**: Auto-detect current spec from branch or directory

All functions meet performance contracts, provide actionable error messages, and support the test-first development approach required by the constitution.
