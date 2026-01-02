// Package validation_test benchmarks artifact file validation performance.
// Related: internal/validation/validation.go
// Tags: validation, benchmark, performance, spec, plan, tasks, artifact
package validation

import (
	"os"
	"path/filepath"
	"testing"
)

func BenchmarkValidateSpecFile(b *testing.B) {
	// Setup: Create a temp directory with spec.md
	tmpDir := b.TempDir()
	specPath := filepath.Join(tmpDir, "spec.md")
	if err := os.WriteFile(specPath, []byte("# Spec"), 0o644); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ValidateSpecFile(tmpDir)
	}
}

func BenchmarkValidatePlanFile(b *testing.B) {
	// Setup: Create a temp directory with plan.md
	tmpDir := b.TempDir()
	planPath := filepath.Join(tmpDir, "plan.md")
	if err := os.WriteFile(planPath, []byte("# Plan"), 0o644); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ValidatePlanFile(tmpDir)
	}
}

func BenchmarkValidateTasksFile(b *testing.B) {
	// Setup: Create a temp directory with tasks.md
	tmpDir := b.TempDir()
	tasksPath := filepath.Join(tmpDir, "tasks.md")
	if err := os.WriteFile(tasksPath, []byte("# Tasks"), 0o644); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ValidateTasksFile(tmpDir)
	}
}

func BenchmarkCountUncheckedTasks(b *testing.B) {
	// Setup: Create a realistic tasks.md file
	content := `# Tasks

## Phase 0: Research
- [x] Research Go libraries
- [x] Research testing frameworks
- [x] Research build tools

## Phase 1: Foundation
- [x] Setup Go module
- [ ] Implement config loading
- [ ] Implement validation
- [ ] Write tests

## Phase 2: Implementation
- [ ] Task 1
- [ ] Task 2
- [ ] Task 3
- [ ] Task 4
- [ ] Task 5
- [ ] Task 6
- [ ] Task 7
- [ ] Task 8
- [ ] Task 9
- [ ] Task 10
`

	tmpDir := b.TempDir()
	tasksPath := filepath.Join(tmpDir, "tasks.md")
	if err := os.WriteFile(tasksPath, []byte(content), 0o644); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = CountUncheckedTasks(tasksPath)
	}
}

func BenchmarkParseTasksByPhase(b *testing.B) {
	// Setup: Create a realistic tasks.md file with multiple phases
	content := `# Tasks

## Phase 0: Research
- [x] Research Go libraries
- [x] Research testing frameworks
- [x] Research build tools

## Phase 1: Foundation
- [x] Setup Go module
- [ ] Implement config loading
- [ ] Implement validation
- [ ] Write tests

## Phase 2: Implementation
- [ ] Task 1
- [ ] Task 2
- [ ] Task 3
- [ ] Task 4
- [ ] Task 5
- [ ] Task 6
- [ ] Task 7
- [ ] Task 8
- [ ] Task 9
- [ ] Task 10

## Phase 3: Polish
- [ ] Run tests
- [ ] Build binaries
- [ ] Update documentation
`

	tmpDir := b.TempDir()
	tasksPath := filepath.Join(tmpDir, "tasks.md")
	if err := os.WriteFile(tasksPath, []byte(content), 0o644); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ParseTasksByPhase(tasksPath)
	}
}
