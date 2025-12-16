package cli

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

func TestArtifactCommand_InvalidType(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := runArtifactCommand([]string{"unknown"}, &stdout, &stderr)

	if err == nil {
		t.Error("expected error for invalid artifact type")
	}

	if exitErr, ok := err.(*exitError); ok {
		if exitErr.code != ExitInvalidArguments {
			t.Errorf("exit code = %d, want %d", exitErr.code, ExitInvalidArguments)
		}
	} else {
		t.Error("expected exitError type")
	}

	if !strings.Contains(stderr.String(), "invalid artifact type") {
		t.Errorf("stderr should contain 'invalid artifact type', got: %s", stderr.String())
	}
}

func TestArtifactCommand_MissingFile(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := runArtifactCommand([]string{"spec", "nonexistent.yaml"}, &stdout, &stderr)

	if err == nil {
		t.Error("expected error for missing file")
	}

	if exitErr, ok := err.(*exitError); ok {
		if exitErr.code != ExitInvalidArguments {
			t.Errorf("exit code = %d, want %d", exitErr.code, ExitInvalidArguments)
		}
	}

	if !strings.Contains(stderr.String(), "not found") {
		t.Errorf("stderr should contain 'not found', got: %s", stderr.String())
	}
}

func TestArtifactCommand_ValidSpec(t *testing.T) {
	var stdout, stderr bytes.Buffer
	testFile := filepath.Join("..", "validation", "testdata", "spec", "valid.yaml")
	err := runArtifactCommand([]string{"spec", testFile}, &stdout, &stderr)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
		t.Logf("stderr: %s", stderr.String())
	}

	if !strings.Contains(stdout.String(), "is valid") {
		t.Errorf("stdout should contain 'is valid', got: %s", stdout.String())
	}

	if !strings.Contains(stdout.String(), "user stories") {
		t.Errorf("stdout should contain summary with 'user stories', got: %s", stdout.String())
	}
}

func TestArtifactCommand_InvalidSpec(t *testing.T) {
	var stdout, stderr bytes.Buffer
	testFile := filepath.Join("..", "validation", "testdata", "spec", "missing_feature.yaml")
	err := runArtifactCommand([]string{"spec", testFile}, &stdout, &stderr)

	if err == nil {
		t.Error("expected error for invalid spec")
	}

	if exitErr, ok := err.(*exitError); ok {
		if exitErr.code != ExitValidationFailed {
			t.Errorf("exit code = %d, want %d", exitErr.code, ExitValidationFailed)
		}
	}

	if !strings.Contains(stderr.String(), "has") && !strings.Contains(stderr.String(), "error") {
		t.Errorf("stderr should indicate errors, got: %s", stderr.String())
	}
}

func TestArtifactCommand_ValidPlan(t *testing.T) {
	var stdout, stderr bytes.Buffer
	testFile := filepath.Join("..", "validation", "testdata", "plan", "valid.yaml")
	err := runArtifactCommand([]string{"plan", testFile}, &stdout, &stderr)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
		t.Logf("stderr: %s", stderr.String())
	}

	if !strings.Contains(stdout.String(), "is valid") {
		t.Errorf("stdout should contain 'is valid', got: %s", stdout.String())
	}
}

func TestArtifactCommand_ValidTasks(t *testing.T) {
	var stdout, stderr bytes.Buffer
	testFile := filepath.Join("..", "validation", "testdata", "tasks", "valid.yaml")
	err := runArtifactCommand([]string{"tasks", testFile}, &stdout, &stderr)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
		t.Logf("stderr: %s", stderr.String())
	}

	if !strings.Contains(stdout.String(), "is valid") {
		t.Errorf("stdout should contain 'is valid', got: %s", stdout.String())
	}
}

func TestArtifactCommand_SchemaSpec(t *testing.T) {
	// Set schema flag
	oldSchemaFlag := artifactSchemaFlag
	artifactSchemaFlag = true
	defer func() { artifactSchemaFlag = oldSchemaFlag }()

	var stdout, stderr bytes.Buffer
	err := runArtifactCommand([]string{"spec"}, &stdout, &stderr)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "Schema for spec") {
		t.Errorf("output should contain 'Schema for spec', got: %s", output)
	}

	if !strings.Contains(output, "feature") {
		t.Errorf("output should contain 'feature' field, got: %s", output)
	}

	if !strings.Contains(output, "user_stories") {
		t.Errorf("output should contain 'user_stories' field, got: %s", output)
	}
}

func TestArtifactCommand_SchemaPlan(t *testing.T) {
	oldSchemaFlag := artifactSchemaFlag
	artifactSchemaFlag = true
	defer func() { artifactSchemaFlag = oldSchemaFlag }()

	var stdout, stderr bytes.Buffer
	err := runArtifactCommand([]string{"plan"}, &stdout, &stderr)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "Schema for plan") {
		t.Errorf("output should contain 'Schema for plan', got: %s", output)
	}

	if !strings.Contains(output, "technical_context") {
		t.Errorf("output should contain 'technical_context' field, got: %s", output)
	}
}

func TestArtifactCommand_SchemaTasks(t *testing.T) {
	oldSchemaFlag := artifactSchemaFlag
	artifactSchemaFlag = true
	defer func() { artifactSchemaFlag = oldSchemaFlag }()

	var stdout, stderr bytes.Buffer
	err := runArtifactCommand([]string{"tasks"}, &stdout, &stderr)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "Schema for tasks") {
		t.Errorf("output should contain 'Schema for tasks', got: %s", output)
	}

	if !strings.Contains(output, "phases") {
		t.Errorf("output should contain 'phases' field, got: %s", output)
	}
}

func TestArtifactCommand_CircularDependency(t *testing.T) {
	var stdout, stderr bytes.Buffer
	testFile := filepath.Join("..", "validation", "testdata", "tasks", "invalid_dep_circular.yaml")
	err := runArtifactCommand([]string{"tasks", testFile}, &stdout, &stderr)

	if err == nil {
		t.Error("expected error for circular dependency")
	}

	if !strings.Contains(stderr.String(), "circular dependency") {
		t.Errorf("stderr should contain 'circular dependency', got: %s", stderr.String())
	}
}

func TestArtifactCommand_NoPathWithoutSchema(t *testing.T) {
	// Ensure schema flag is false
	oldSchemaFlag := artifactSchemaFlag
	artifactSchemaFlag = false
	defer func() { artifactSchemaFlag = oldSchemaFlag }()

	var stdout, stderr bytes.Buffer
	err := runArtifactCommand([]string{"spec"}, &stdout, &stderr)

	if err == nil {
		t.Error("expected error when no path provided and --schema not set")
	}

	if exitErr, ok := err.(*exitError); ok {
		if exitErr.code != ExitInvalidArguments {
			t.Errorf("exit code = %d, want %d", exitErr.code, ExitInvalidArguments)
		}
	}
}

func TestExitCode(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected int
	}{
		{"nil error", nil, ExitSuccess},
		{"exit error 1", &exitError{code: 1}, 1},
		{"exit error 3", &exitError{code: 3}, 3},
		{"generic error", fmt.Errorf("some error"), ExitValidationFailed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ExitCode(tt.err); got != tt.expected {
				t.Errorf("ExitCode() = %d, want %d", got, tt.expected)
			}
		})
	}
}
