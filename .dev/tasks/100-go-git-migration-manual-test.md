# Manual Test Plan: go-git Migration (Branch 100-go-git-migration)

## Overview

This feature migrates core git operations from `git` CLI to the `go-git` library. The key change for implement phase is that `IS_GIT_REPO` is now provided via prereqs output instead of the agent running `git rev-parse --git-dir`.

**Goal**: Verify the implement slash command works correctly with the new IS_GIT_REPO context variable.

---

## Setup: Create Test Repository in /tmp

```bash
# Store autospec binary path
AUTOSPEC_BIN="/home/ari/repos/autospec/bin/autospec"

# Create test directory
rm -rf /tmp/go-git-test && mkdir -p /tmp/go-git-test && cd /tmp/go-git-test

# Initialize git repo
git init
git config user.email "test@test.com"
git config user.name "Test User"

# Create minimal Go project
mkdir -p internal/hello
cat > go.mod << 'EOF'
module testproject

go 1.25
EOF

cat > internal/hello/hello.go << 'EOF'
package hello

// Greet returns a greeting message
func Greet(name string) string {
    return "Hello, " + name
}
EOF

cat > internal/hello/hello_test.go << 'EOF'
package hello

import "testing"

func TestGreet(t *testing.T) {
    got := Greet("World")
    want := "Hello, World"
    if got != want {
        t.Errorf("Greet() = %q, want %q", got, want)
    }
}
EOF

# Initial commit
git add -A
git commit -m "Initial commit"

# Create feature branch
git checkout -b 001-add-farewell
```

---

## Test 1: Initialize autospec in test repo

```bash
cd /tmp/go-git-test

# Run init (use --skip-constitution for quick setup, or create minimal constitution)
$AUTOSPEC_BIN init --agent claude --no-gitignore --no-constitution
```

**Expected**:
- Creates `.autospec/config.yml`
- Installs commands to `.claude/commands/`
- No errors about git CLI

---

## Test 2: Verify prereqs shows IS_GIT_REPO

```bash
cd /tmp/go-git-test

# Create minimal spec for testing
mkdir -p specs/001-add-farewell
cat > specs/001-add-farewell/spec.yaml << 'EOF'
feature:
  branch: "001-add-farewell"
  created: "2026-01-16"
  status: "Draft"
  input: "Add a Farewell function to the hello package"

requirements:
  functional:
    - id: "FR-001"
      description: "Add Farewell function that returns 'Goodbye, <name>'"
      testable: true
      acceptance_criteria: "Farewell('World') returns 'Goodbye, World'"

_meta:
  version: "1.0.0"
  artifact_type: "spec"
EOF

# Check prereqs JSON includes IS_GIT_REPO
$AUTOSPEC_BIN prereqs --json --require-spec | jq '.IS_GIT_REPO'
```

**Expected**: `true`

---

## Test 3: Create minimal tasks.yaml for implement test

```bash
cd /tmp/go-git-test

# Create minimal plan.yaml
cat > specs/001-add-farewell/plan.yaml << 'EOF'
feature:
  branch: "001-add-farewell"

phases:
  - id: "phase-1"
    name: "Implementation"
    tasks:
      - id: "task-1"
        title: "Add Farewell function"
        scope:
          - internal/hello/hello.go
          - internal/hello/hello_test.go

_meta:
  version: "1.0.0"
  artifact_type: "plan"
EOF

# Create minimal tasks.yaml
cat > specs/001-add-farewell/tasks.yaml << 'EOF'
feature:
  branch: "001-add-farewell"

phases:
  - id: "phase-1"
    name: "Implementation"
    tasks:
      - id: "task-1"
        title: "Add Farewell function with test"
        status: "pending"
        description: "Add a Farewell(name string) string function that returns 'Goodbye, <name>'"
        files:
          - internal/hello/hello.go
          - internal/hello/hello_test.go
        verification:
          - "go test ./..."

_meta:
  version: "1.0.0"
  artifact_type: "tasks"
EOF
```

---

## Test 4: Run implement phase with Claude (Critical Test)

This tests that the implement slash command:
1. Receives IS_GIT_REPO from prereqs
2. Does NOT run `git rev-parse --git-dir`
3. Correctly handles .gitignore based on IS_GIT_REPO

```bash
cd /tmp/go-git-test

# Option A: Run implement via autospec
$AUTOSPEC_BIN implement --tasks

# Option B: Run Claude directly to test slash command
# claude "/autospec.implement"
```

**Expected**:
- Claude should NOT attempt to run `git rev-parse --git-dir`
- Claude should see IS_GIT_REPO=true from prereqs output
- Implementation should add Farewell function
- Test should pass

---

## Test 5: Verify implement template references IS_GIT_REPO

```bash
# Check the implement template uses IS_GIT_REPO variable
grep -A2 "IS_GIT_REPO" /home/ari/repos/autospec/internal/commands/autospec.implement.md
```

**Expected**: Template mentions using IS_GIT_REPO from prereqs, not git CLI

---

## Test 6: Test in non-git directory

```bash
# Create non-git directory
rm -rf /tmp/no-git-test && mkdir -p /tmp/no-git-test && cd /tmp/no-git-test

# Check IS_GIT_REPO is false
$AUTOSPEC_BIN prereqs --json --paths-only | jq '.IS_GIT_REPO'
```

**Expected**: `false`

---

## Test 7: Doctor command no longer checks git

```bash
cd /tmp/go-git-test
$AUTOSPEC_BIN doctor
```

**Expected**:
- No "Git CLI" line in output
- Shows Claude CLI and settings checks only

---

## Test 8: Debug logging shows go-git operations

```bash
cd /tmp/go-git-test
$AUTOSPEC_BIN --debug prereqs --json 2>&1 | grep -i "\[git\]"
```

**Expected**: Debug output showing go-git operations:
- `[git] opening repository at ...`
- `[git] IsGitRepository: true`

---

## Cleanup

```bash
rm -rf /tmp/go-git-test /tmp/no-git-test
```

---

## Test Results

| Test | Status | Notes |
|------|--------|-------|
| 1. Init in test repo | PASS | Created config, installed commands, no git CLI errors |
| 2. prereqs IS_GIT_REPO | PASS | `IS_GIT_REPO: true` in JSON output |
| 3. Create tasks.yaml | PASS | Valid schema with Pending/InProgress/Completed/Blocked status |
| 4. Implement phase | **PASS** | Claude used IS_GIT_REPO from prereqs, created .gitignore, completed task |
| 5. Template check | PASS | Template uses IS_GIT_REPO, no git CLI commands |
| 6. Non-git directory | PASS | `IS_GIT_REPO: false` in non-git /tmp directory |
| 7. Doctor no git check | PASS | No "Git CLI" line in doctor output |
| 8. Debug logging | SKIP | SetDebugLogger not wired up (minor, not blocking) |

### Implement Phase Verification (2026-01-16)

After installing built binary globally (`make ip`), the full end-to-end test passed:

1. **prereqs output included IS_GIT_REPO**:
   ```json
   {"IS_GIT_REPO":true, ...}
   ```

2. **Claude did NOT run `git rev-parse --git-dir`** - used IS_GIT_REPO from prereqs instead

3. **Claude correctly created .gitignore** because IS_GIT_REPO was true

4. **Implementation completed successfully**:
   - Added `Farewell` function to `internal/hello/hello.go`
   - Added `TestFarewell` to `internal/hello/hello_test.go`
   - Tests passed
   - Task T001 marked as Completed

### go-git IS_GIT_REPO CLI Tests (2026-01-16)

| Test | Expected | Result |
|------|----------|--------|
| Inside git repo | true | ✓ PASS |
| Outside git repo | false | ✓ PASS |
| Nested subdirectory (DetectDotGit) | true | ✓ PASS |
| Empty git repo (no commits) | true | ✓ PASS |
| Deeply nested (5 levels) | true | ✓ PASS |
| Bare repository (no worktree) | false | ✓ PASS |

All go-git PlainOpenWithOptions with DetectDotGit tests pass.

---

## Key Validation Points

1. **IS_GIT_REPO in prereqs**: JSON output must include `"IS_GIT_REPO": true/false`
2. **Template updated**: `autospec.implement.md` must reference IS_GIT_REPO variable
3. **No git CLI in template**: Implement template should NOT tell agent to run git commands for repo detection
4. **Doctor simplified**: No git CLI health check
