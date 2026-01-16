# Quickstart: Building the autospec Go Binary

**Feature**: Go Binary Migration (002-go-binary-migration)
**Date**: 2025-10-22

This guide walks through the development process for migrating the bash-based autospec tool to a Go binary.

---

## Prerequisites

### Required Tools
- **Go 1.21+**: `go version` should show 1.21 or higher
- **Git**: For version control and spec detection
- **Claude CLI**: Required for workflow orchestration
- **specify CLI**: Required for SpecKit template operations

### Optional Tools
- **shellcheck**: For validating existing bash scripts during migration
- **bats-core**: For running existing tests before migration
- **benchstat**: For performance comparison (`go install golang.org/x/perf/cmd/benchstat@latest`)

### Development Environment
```bash
# Verify Go installation
go version  # Should be 1.21+

# Verify required CLIs
claude --version
specify --version
git --version
```

---

## Phase 0: Project Setup (15 minutes)

### 1. Initialize Go Module

```bash
# Create project directory structure
cd /home/ari/repos/autospec
mkdir -p cmd/autospec
mkdir -p internal/{validation,retry,config,git,spec,workflow,cli}
mkdir -p integration/testdata

# Initialize Go module
go mod init github.com/username/auto-claude-speckit

# Initial dependencies
go get github.com/spf13/cobra@latest
go get github.com/knadh/koanf/v2@latest
go get github.com/knadh/koanf/parsers/json@latest
go get github.com/knadh/koanf/providers/file@latest
go get github.com/knadh/koanf/providers/env@latest
go get github.com/go-playground/validator/v10@latest
go get github.com/stretchr/testify@latest
go get github.com/rogpeppe/go-internal/testscript@latest
```

### 2. Set Up Project Structure

```bash
# Create main entry point
cat > cmd/autospec/main.go <<'EOF'
package main

import (
    "os"
    "github.com/username/auto-claude-speckit/internal/cli"
)

func main() {
    if err := cli.Execute(); err != nil {
        os.Exit(1)
    }
}
EOF

# Create basic CLI structure
cat > internal/cli/root.go <<'EOF'
package cli

import "github.com/spf13/cobra"

var rootCmd = &cobra.Command{
    Use:   "autospec",
    Short: "autospec workflow automation",
    Long:  "Cross-platform CLI tool for SpecKit workflow validation and orchestration",
}

func Execute() error {
    return rootCmd.Execute()
}

func init() {
    // Global flags
    rootCmd.PersistentFlags().StringP("config", "c", ".autospec/config.json", "Path to config file")
    rootCmd.PersistentFlags().String("specs-dir", "./specs", "Directory containing feature specs")
    rootCmd.PersistentFlags().Bool("skip-preflight", false, "Skip pre-flight validation checks")
    rootCmd.PersistentFlags().BoolP("debug", "d", false, "Enable debug logging")
}
EOF
```

### 3. Verify Initial Setup

```bash
# Build initial binary
go build -o autospec ./cmd/autospec

# Test basic execution
./autospec --help

# Expected output:
# autospec workflow automation
#
# Usage:
#   autospec [command]
#
# Available Commands:
#   ...
```

**Success Criteria**: `autospec --help` displays help text in <50ms

---

## Phase 1: Foundation (1-2 days)

### 1. Implement Configuration Management

**File**: `internal/config/config.go`

```bash
# Create config package
mkdir -p internal/config

# Implement configuration loading
cat > internal/config/config.go <<'EOF'
package config

import (
    "path/filepath"
    "os"
    "github.com/knadh/koanf/v2"
    "github.com/knadh/koanf/parsers/json"
    "github.com/knadh/koanf/providers/file"
    "github.com/knadh/koanf/providers/env"
    "github.com/go-playground/validator/v10"
)

type Config struct {
    ClaudeCmd       string   `koanf:"claude_cmd" validate:"required"`
    ClaudeArgs      []string `koanf:"claude_args"`
    UseAPIKey       bool     `koanf:"use_api_key"`
    CustomClaudeCmd string   `koanf:"custom_claude_cmd"`
    SpecifyCmd      string   `koanf:"specify_cmd" validate:"required"`
    MaxRetries      int      `koanf:"max_retries" validate:"min=1,max=10"`
    SpecsDir        string   `koanf:"specs_dir" validate:"required"`
    StateDir        string   `koanf:"state_dir" validate:"required"`
    SkipPreflight   bool     `koanf:"skip_preflight"`
    Timeout         int      `koanf:"timeout" validate:"min=1,max=3600"`
}

func Load() (*Config, error) {
    k := koanf.New(".")

    // Load global config
    homeDir, _ := os.UserHomeDir()
    globalPath := filepath.Join(homeDir, ".autospec", "config.json")
    k.Load(file.Provider(globalPath), json.Parser())

    // Override with local config
    k.Load(file.Provider(".autospec/config.json"), json.Parser())

    // Override with environment variables
    k.Load(env.Provider("AUTOSPEC_", ".", envTransform), nil)

    // Unmarshal
    var cfg Config
    if err := k.Unmarshal("", &cfg); err != nil {
        return nil, err
    }

    // Validate
    validate := validator.New()
    if err := validate.Struct(cfg); err != nil {
        return nil, err
    }

    return &cfg, nil
}
EOF
```

**Test**:
```bash
# Write test
cat > internal/config/config_test.go <<'EOF'
package config

import (
    "testing"
    "testing/fstest"
    "github.com/stretchr/testify/assert"
)

func TestLoad(t *testing.T) {
    tests := map[string]struct {
        globalConfig string
        localConfig  string
        envVars      map[string]string
        wantRetries  int
    }{
        "defaults": {
            globalConfig: `{"max_retries": 3}`,
            wantRetries:  3,
        },
        "local override": {
            globalConfig: `{"max_retries": 3}`,
            localConfig:  `{"max_retries": 5}`,
            wantRetries:  5,
        },
        "env override": {
            globalConfig: `{"max_retries": 3}`,
            envVars:      map[string]string{"AUTOSPEC_MAX_RETRIES": "7"},
            wantRetries:  7,
        },
    }

    for name, tc := range tests {
        t.Run(name, func(t *testing.T) {
            // Test implementation
        })
    }
}
EOF

# Run test
go test ./internal/config/...
```

### 2. Implement Git Operations

**File**: `internal/git/git.go`

```bash
cat > internal/git/git.go <<'EOF'
package git

import (
    "os/exec"
    "strings"
)

func GetCurrentBranch() (string, error) {
    cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
    output, err := cmd.Output()
    if err != nil {
        return "", err
    }
    return strings.TrimSpace(string(output)), nil
}

func GetRepositoryRoot() (string, error) {
    cmd := exec.Command("git", "rev-parse", "--show-toplevel")
    output, err := cmd.Output()
    if err != nil {
        return "", err
    }
    return strings.TrimSpace(string(output)), nil
}

func IsGitRepository() bool {
    cmd := exec.Command("git", "rev-parse", "--git-dir")
    return cmd.Run() == nil
}
EOF
```

**Test with Mocking**:
```bash
cat > internal/git/git_test.go <<'EOF'
package git

import (
    "os"
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
    behavior := os.Getenv("TEST_MOCK_BEHAVIOR")
    switch behavior {
    case "":
        os.Exit(m.Run())
    case "gitBranch":
        println("002-go-binary-migration")
        os.Exit(0)
    case "gitRoot":
        println("/home/user/project")
        os.Exit(0)
    }
}

func TestGetCurrentBranch(t *testing.T) {
    // Mock git command
    testExe, _ := os.Executable()
    origPath := os.Getenv("PATH")
    defer os.Setenv("PATH", origPath)

    t.Setenv("TEST_MOCK_BEHAVIOR", "gitBranch")
    t.Setenv("PATH", filepath.Dir(testExe))

    branch, err := GetCurrentBranch()
    assert.NoError(t, err)
    assert.Equal(t, "002-go-binary-migration", branch)
}
EOF

go test ./internal/git/...
```

### 3. Implement Validation Package

**File**: `internal/validation/validation.go`

Follow validation-api.md contract:
```bash
cat > internal/validation/validation.go <<'EOF'
package validation

import (
    "fmt"
    "os"
    "path/filepath"
    "regexp"
    "strings"
)

func ValidateSpecFile(specsDir, specName string) error {
    specDir, err := findSpecDirectory(specsDir, specName)
    if err != nil {
        return err
    }

    specPath := filepath.Join(specDir, "spec.md")
    if _, err := os.Stat(specPath); os.IsNotExist(err) {
        return fmt.Errorf("spec.md not found in %s - run 'autospec specify <description>' to create it", specDir)
    }

    return nil
}

func CountUncheckedTasks(tasksPath string) (int, error) {
    content, err := os.ReadFile(tasksPath)
    if err != nil {
        return 0, err
    }

    uncheckedPattern := regexp.MustCompile(`(?m)^[-*]\s+\[\s\]`)
    matches := uncheckedPattern.FindAllString(string(content), -1)

    return len(matches), nil
}
EOF

# Test with table-driven tests
cat > internal/validation/validation_test.go <<'EOF'
package validation

import (
    "testing"
    "testing/fstest"
    "github.com/stretchr/testify/assert"
)

func TestCountUncheckedTasks(t *testing.T) {
    tests := map[string]struct {
        content   string
        wantCount int
    }{
        "no tasks": {
            content:   "# Tasks\n\n## Phase 1\n\nNo tasks here.",
            wantCount: 0,
        },
        "all unchecked": {
            content:   "- [ ] Task 1\n- [ ] Task 2\n* [ ] Task 3",
            wantCount: 3,
        },
        "mixed checked/unchecked": {
            content:   "- [x] Done\n- [ ] Todo\n* [X] Done\n* [ ] Todo",
            wantCount: 2,
        },
    }

    for name, tc := range tests {
        t.Run(name, func(t *testing.T) {
            t.Parallel()

            // Write to temp file
            tmpDir := t.TempDir()
            tasksPath := filepath.Join(tmpDir, "tasks.md")
            os.WriteFile(tasksPath, []byte(tc.content), 0644)

            count, err := CountUncheckedTasks(tasksPath)
            assert.NoError(t, err)
            assert.Equal(t, tc.wantCount, count)
        })
    }
}
EOF

go test ./internal/validation/...
```

### 4. Implement Retry State Management

**File**: `internal/retry/retry.go`

Follow validation-api.md contract for LoadRetryState, SaveRetryState, IncrementRetryCount, ResetRetryCount.

**Test**: Use table-driven tests with t.TempDir() for state file operations.

---

## Phase 2: CLI Commands (2-3 days)

### 1. Implement `version` Command

```bash
cat > internal/cli/version.go <<'EOF'
package cli

import (
    "fmt"
    "github.com/spf13/cobra"
)

var (
    Version   = "dev"
    Commit    = "unknown"
    BuildDate = "unknown"
)

var versionCmd = &cobra.Command{
    Use:   "version",
    Short: "Display version information",
    Run: func(cmd *cobra.Command, args []string) {
        fmt.Printf("autospec version %s\n", Version)
        fmt.Printf("Built from commit: %s\n", Commit)
        fmt.Printf("Build date: %s\n", BuildDate)
    },
}

func init() {
    rootCmd.AddCommand(versionCmd)
}
EOF
```

**Build with Version Info**:
```bash
go build -ldflags "-X github.com/username/auto-claude-speckit/internal/cli.Version=1.0.0 \
                   -X github.com/username/auto-claude-speckit/internal/cli.Commit=$(git rev-parse --short HEAD) \
                   -X github.com/username/auto-claude-speckit/internal/cli.BuildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
         -o autospec ./cmd/autospec

./autospec version
```

### 2. Implement `init` Command

```bash
cat > internal/cli/init.go <<'EOF'
package cli

import (
    "encoding/json"
    "os"
    "path/filepath"
    "github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
    Use:   "init",
    Short: "Initialize autospec configuration",
    RunE: func(cmd *cobra.Command, args []string) error {
        global, _ := cmd.Flags().GetBool("global")
        force, _ := cmd.Flags().GetBool("force")

        configPath := ".autospec/config.json"
        if global {
            homeDir, _ := os.UserHomeDir()
            configPath = filepath.Join(homeDir, ".autospec", "config.json")
        }

        // Create default config
        defaultConfig := map[string]interface{}{
            "claude_cmd":    "claude",
            "claude_args":   []string{"-p", "--dangerously-skip-permissions", "--verbose", "--output-format", "stream-json"},
            "use_api_key":   false,
            "specify_cmd":   "specify",
            "max_retries":   3,
            "specs_dir":     "./specs",
            "state_dir":     "~/.autospec/state",
            "skip_preflight": false,
            "timeout":       300,
        }

        // Check if exists
        if _, err := os.Stat(configPath); err == nil && !force {
            return fmt.Errorf("config already exists at %s (use --force to overwrite)", configPath)
        }

        // Create directory
        os.MkdirAll(filepath.Dir(configPath), 0755)

        // Write config
        data, _ := json.MarshalIndent(defaultConfig, "", "  ")
        os.WriteFile(configPath, data, 0644)

        fmt.Printf("Created configuration at %s\n", configPath)
        return nil
    },
}

func init() {
    rootCmd.AddCommand(initCmd)
    initCmd.Flags().BoolP("global", "g", false, "Create global config")
    initCmd.Flags().BoolP("force", "f", false, "Overwrite existing config")
}
EOF
```

### 3. Implement `status` Command

**File**: `internal/cli/status.go`

Use `internal/validation/ParseTasksByPhase` to display progress.

### 4. Implement `workflow` Command

**File**: `internal/cli/workflow.go`

Orchestrate specify → plan → tasks with validation and retry.

---

## Phase 3: Testing (2-3 days)

### 1. Unit Tests

Write tests for all packages:
```bash
# Run all unit tests
go test ./internal/...

# With coverage
go test -cover ./internal/...

# Coverage report
go test -coverprofile=cover.out ./internal/...
go tool cover -html=cover.out
```

### 2. CLI Integration Tests with testscript

```bash
# Create testscript tests
mkdir -p cmd/autospec/testdata/scripts

cat > cmd/autospec/testdata/scripts/version.txt <<'EOF'
exec autospec version
stdout 'autospec version'
! stderr .
EOF

cat > cmd/autospec/testdata/scripts/help.txt <<'EOF'
exec autospec --help
stdout 'autospec'
! stderr .
EOF

# Add TestMain to cmd/autospec/main_test.go
cat > cmd/autospec/main_test.go <<'EOF'
package main

import (
    "testing"
    "github.com/rogpeppe/go-internal/testscript"
)

func TestMain(m *testing.M) {
    testscript.Main(m, map[string]func(){
        "autospec": main,
    })
}

func TestCLI(t *testing.T) {
    testscript.Run(t, testscript.Params{
        Dir: "testdata/scripts",
    })
}
EOF

# Run testscript tests
go test ./cmd/autospec/...
```

### 3. Benchmarks

```bash
cat > internal/validation/validation_bench_test.go <<'EOF'
package validation

import (
    "testing"
    "os"
    "path/filepath"
)

func BenchmarkCountUncheckedTasks(b *testing.B) {
    tmpDir := b.TempDir()
    tasksPath := filepath.Join(tmpDir, "tasks.md")

    // Create realistic tasks.md
    content := generateTasksContent(100) // 100 tasks
    os.WriteFile(tasksPath, []byte(content), 0644)

    for b.Loop() {
        _, _ = CountUncheckedTasks(tasksPath)
    }
}
EOF

# Run benchmarks
go test -bench=. -benchmem ./internal/validation/...
```

---

## Phase 4: Cross-Platform Builds (1 day)

### 1. Build for All Platforms

```bash
# Create build script
cat > scripts/build-all.sh <<'EOF'
#!/bin/bash
set -euo pipefail

VERSION=${VERSION:-"dev"}
COMMIT=$(git rev-parse --short HEAD)
BUILD_DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS="-X github.com/username/auto-claude-speckit/internal/cli.Version=${VERSION} \
         -X github.com/username/auto-claude-speckit/internal/cli.Commit=${COMMIT} \
         -X github.com/username/auto-claude-speckit/internal/cli.BuildDate=${BUILD_DATE} \
         -s -w"

# Linux
GOOS=linux GOARCH=amd64 go build -ldflags="${LDFLAGS}" -o dist/autospec-linux-amd64 ./cmd/autospec
GOOS=linux GOARCH=arm64 go build -ldflags="${LDFLAGS}" -o dist/autospec-linux-arm64 ./cmd/autospec

# macOS
GOOS=darwin GOARCH=amd64 go build -ldflags="${LDFLAGS}" -o dist/autospec-darwin-amd64 ./cmd/autospec
GOOS=darwin GOARCH=arm64 go build -ldflags="${LDFLAGS}" -o dist/autospec-darwin-arm64 ./cmd/autospec

# Windows
GOOS=windows GOARCH=amd64 go build -ldflags="${LDFLAGS}" -o dist/autospec-windows-amd64.exe ./cmd/autospec

echo "Built binaries in dist/"
ls -lh dist/
EOF

chmod +x scripts/build-all.sh
./scripts/build-all.sh
```

### 2. Verify Binary Sizes

```bash
ls -lh dist/

# Expected output (all should be <15 MB):
# autospec-linux-amd64: ~4-5 MB
# autospec-darwin-amd64: ~4-5 MB
# autospec-windows-amd64.exe: ~4-5 MB
```

---

## Testing Checklist

Before considering migration complete:

### Unit Tests
- [ ] All packages have unit tests
- [ ] Table-driven tests for core functions
- [ ] Mock external dependencies (git, claude, specify)
- [ ] Test coverage >80%

### Integration Tests
- [ ] testscript tests for all CLI commands
- [ ] End-to-end workflow tests
- [ ] Retry logic integration tests

### Performance Tests
- [ ] `autospec version` <50ms
- [ ] `autospec status` <1s
- [ ] Pre-flight validation <100ms
- [ ] Validation functions meet contracts

### Cross-Platform Tests
- [ ] Builds succeed on Linux, macOS, Windows
- [ ] Binary sizes <15 MB
- [ ] All tests pass on each platform

### Constitution Compliance
- [ ] 60+ tests (unit + integration + testscript)
- [ ] Test-first development followed
- [ ] Performance targets met
- [ ] Idempotent operations verified
- [ ] Validation-first approach maintained

---

## Common Issues

### Issue: Binary too large (>15 MB)

**Solution**: Strip debug symbols
```bash
go build -ldflags="-s -w" -o autospec ./cmd/autospec
```

### Issue: Tests fail on Windows (path separators)

**Solution**: Use `filepath.Join()` everywhere
```go
// Bad
path := "specs/001/spec.md"

// Good
path := filepath.Join("specs", "001", "spec.md")
```

### Issue: Git operations fail on Windows

**Solution**: Ensure git is in PATH, use `exec.LookPath("git")` to verify

### Issue: Config not loading

**Solution**: Check file permissions, use absolute paths
```go
// Expand ~ to home directory
if strings.HasPrefix(path, "~/") {
    home, _ := os.UserHomeDir()
    path = filepath.Join(home, path[2:])
}
```

---

## Next Steps

After completing this quickstart:

1. **Port remaining bash functionality**: Any scripts not yet migrated
2. **Add GitHub Actions CI/CD**: Automate builds and releases
3. **Create release pipeline**: Tag versions, generate binaries
4. **Update documentation**: README, CLAUDE.md references
5. **Deprecate bash scripts**: Move to `legacy/` directory

---

## Performance Baseline

Record performance before and after migration:

```bash
# Before (bash)
time ./scripts/speckit-workflow-validate.sh 001

# After (Go)
time ./autospec workflow "feature"

# Compare
# Expected: Go binary 2-5x faster
```

Use `benchstat` to compare scientifically.

---

## Success Criteria Verification

From spec.md success criteria:

- [ ] SC-001: Install in <30 seconds (`go install` or download binary)
- [ ] SC-002: Runs identically on Linux, macOS, Windows
- [ ] SC-003: Binary size <15 MB
- [ ] SC-004: Startup <50ms (`time autospec --version`)
- [ ] SC-005: Pre-flight <100ms
- [ ] SC-006: Status <1s
- [ ] SC-007: Workflow <5s (excluding Claude)
- [ ] SC-008: 60+ tests pass
- [ ] SC-009: Zero runtime dependencies
- [ ] SC-011: Actionable error messages
- [ ] SC-012: Custom command configs work

---

This quickstart provides a clear path from initial setup to production-ready Go binary, following constitution principles (test-first, validation-first, performance standards) throughout the development process.
