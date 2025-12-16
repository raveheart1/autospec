package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ariel-frischer/autospec/internal/agent"
)

func TestUpdateAgentContextCmd_Help(t *testing.T) {
	// Verify the command is registered and has correct help info
	cmd := updateAgentContextCmd

	if cmd.Use != "update-agent-context" {
		t.Errorf("Command Use = %q, want %q", cmd.Use, "update-agent-context")
	}

	if cmd.Short == "" {
		t.Error("Command Short description should not be empty")
	}

	if cmd.Long == "" {
		t.Error("Command Long description should not be empty")
	}

	if cmd.Example == "" {
		t.Error("Command Example should not be empty")
	}
}

func TestUpdateAgentContextCmd_Flags(t *testing.T) {
	cmd := updateAgentContextCmd

	// Check --agent flag exists
	agentFlag := cmd.Flags().Lookup("agent")
	if agentFlag == nil {
		t.Error("--agent flag not found")
	}

	// Check --json flag exists
	jsonFlag := cmd.Flags().Lookup("json")
	if jsonFlag == nil {
		t.Error("--json flag not found")
	}
}

func TestBuildCommandOutput(t *testing.T) {
	planData := &agent.PlanData{
		Language:    "Go 1.25.1",
		Framework:   "Cobra CLI",
		Database:    "None",
		ProjectType: "cli",
		Branch:      "017-feature",
	}

	results := []agent.UpdateResult{
		{
			FilePath:          "CLAUDE.md",
			Created:           false,
			TechnologiesAdded: []string{"Go 1.25.1"},
			Error:             nil,
		},
	}

	output := buildCommandOutput("017-feature", "specs/017-feature/plan.yaml", planData, results, nil)

	if !output.Success {
		t.Error("buildCommandOutput() Success should be true")
	}

	if output.SpecName != "017-feature" {
		t.Errorf("buildCommandOutput() SpecName = %q, want %q", output.SpecName, "017-feature")
	}

	if output.PlanPath != "specs/017-feature/plan.yaml" {
		t.Errorf("buildCommandOutput() PlanPath = %q", output.PlanPath)
	}

	if output.Technologies == nil {
		t.Error("buildCommandOutput() Technologies should not be nil")
	}

	if len(output.UpdatedFiles) != 1 {
		t.Errorf("buildCommandOutput() UpdatedFiles has %d items, want 1", len(output.UpdatedFiles))
	}

	if len(output.Errors) != 0 {
		t.Errorf("buildCommandOutput() Errors has %d items, want 0", len(output.Errors))
	}
}

func TestBuildCommandOutput_WithErrors(t *testing.T) {
	planData := &agent.PlanData{
		Language: "Go 1.25.1",
		Branch:   "017-feature",
	}

	results := []agent.UpdateResult{
		{
			FilePath: "CLAUDE.md",
			Error:    os.ErrPermission,
		},
	}

	output := buildCommandOutput("017-feature", "plan.yaml", planData, results, nil)

	if output.Success {
		t.Error("buildCommandOutput() Success should be false when there are errors")
	}

	if len(output.Errors) != 1 {
		t.Errorf("buildCommandOutput() Errors has %d items, want 1", len(output.Errors))
	}
}

func TestOutputJSON(t *testing.T) {
	output := agent.CommandOutput{
		Success:  true,
		SpecName: "017-feature",
		PlanPath: "specs/017-feature/plan.yaml",
		Technologies: &agent.PlanData{
			Language:    "Go 1.25.1",
			Framework:   "Cobra CLI",
			Database:    "None",
			ProjectType: "cli",
			Branch:      "017-feature",
		},
		UpdatedFiles: []agent.UpdateResult{
			{FilePath: "CLAUDE.md", Created: false, TechnologiesAdded: []string{"Go 1.25.1"}},
		},
		Errors: []string{},
	}

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := outputJSON(output)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	jsonOutput := buf.String()

	if err != nil {
		t.Errorf("outputJSON() returned error: %v", err)
	}

	// Verify it's valid JSON
	var parsed agent.CommandOutput
	if err := json.Unmarshal([]byte(jsonOutput), &parsed); err != nil {
		t.Errorf("outputJSON() produced invalid JSON: %v", err)
	}

	if !parsed.Success {
		t.Error("outputJSON() parsed Success should be true")
	}

	if parsed.SpecName != "017-feature" {
		t.Errorf("outputJSON() parsed SpecName = %q, want %q", parsed.SpecName, "017-feature")
	}
}

func TestOutputJSON_WithErrors(t *testing.T) {
	output := agent.CommandOutput{
		Success:      false,
		SpecName:     "017-feature",
		PlanPath:     "plan.yaml",
		Technologies: nil,
		UpdatedFiles: nil,
		Errors:       []string{"some error"},
	}

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := outputJSON(output)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)

	// Should return error when Success is false
	if err == nil {
		t.Error("outputJSON() should return error when Success is false")
	}
}

func TestOutputText(t *testing.T) {
	output := agent.CommandOutput{
		Success:  true,
		SpecName: "017-feature",
		PlanPath: "specs/017-feature/plan.yaml",
		Technologies: &agent.PlanData{
			Language:    "Go 1.25.1",
			Framework:   "Cobra CLI",
			Database:    "None",
			ProjectType: "cli",
			Branch:      "017-feature",
		},
		UpdatedFiles: []agent.UpdateResult{
			{FilePath: "CLAUDE.md", Created: false, TechnologiesAdded: []string{"Go 1.25.1"}},
		},
		Errors: []string{},
	}

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := outputText(output)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	textOutput := buf.String()

	if err != nil {
		t.Errorf("outputText() returned error: %v", err)
	}

	// Verify text contains expected information
	if !strings.Contains(textOutput, "017-feature") {
		t.Error("outputText() should contain spec name")
	}

	if !strings.Contains(textOutput, "CLAUDE.md") {
		t.Error("outputText() should contain updated file name")
	}

	if !strings.Contains(textOutput, "Go 1.25.1") {
		t.Error("outputText() should contain technology")
	}

	if !strings.Contains(textOutput, "successfully") {
		t.Error("outputText() should contain success message")
	}
}

func TestOutputText_WithErrors(t *testing.T) {
	output := agent.CommandOutput{
		Success:      false,
		SpecName:     "017-feature",
		PlanPath:     "plan.yaml",
		Technologies: nil,
		UpdatedFiles: nil,
		Errors:       []string{"some error"},
	}

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := outputText(output)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	textOutput := buf.String()

	// Should return error
	if err == nil {
		t.Error("outputText() should return error when there are errors")
	}

	// Should contain error message
	if !strings.Contains(textOutput, "some error") {
		t.Error("outputText() should display error messages")
	}
}

func TestOutputError(t *testing.T) {
	testErr := os.ErrNotExist

	// Test text output
	err := outputError(testErr, false)
	if err != testErr {
		t.Errorf("outputError() should return original error, got %v", err)
	}

	// Test JSON output
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err = outputError(testErr, true)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	jsonOutput := buf.String()

	if err != testErr {
		t.Errorf("outputError() should return original error")
	}

	// Verify JSON contains error
	var parsed agent.CommandOutput
	if jsonErr := json.Unmarshal([]byte(jsonOutput), &parsed); jsonErr != nil {
		t.Errorf("outputError() produced invalid JSON: %v", jsonErr)
	}

	if parsed.Success {
		t.Error("outputError() parsed Success should be false")
	}

	if len(parsed.Errors) != 1 {
		t.Errorf("outputError() parsed Errors has %d items, want 1", len(parsed.Errors))
	}
}

// TestIntegration_UpdateAgentContext tests the command in a controlled environment
func TestIntegration_UpdateAgentContext(t *testing.T) {
	// This test requires setting up a temporary git repository
	// and spec directory with plan.yaml, which is complex.
	// We'll test the core functions are exported and working.

	// Test that supported agents are accessible
	ids := agent.GetAllAgentIDs()
	if len(ids) != 17 {
		t.Errorf("Expected 17 supported agents, got %d", len(ids))
	}

	// Test agent lookup
	claude, err := agent.GetAgentByID("claude")
	if err != nil {
		t.Errorf("GetAgentByID(claude) error: %v", err)
	}
	if claude.FilePath != "CLAUDE.md" {
		t.Errorf("Claude file path = %q, want CLAUDE.md", claude.FilePath)
	}
}

// TestJSONOutputSchema verifies JSON output matches expected schema
func TestJSONOutputSchema(t *testing.T) {
	// Create a full output with all fields
	output := agent.CommandOutput{
		Success:  true,
		SpecName: "017-update-agent-context-go",
		PlanPath: "specs/017-update-agent-context-go/plan.yaml",
		Technologies: &agent.PlanData{
			Language:    "Go 1.25.1",
			Framework:   "Cobra CLI v1.10.1",
			Database:    "PostgreSQL",
			ProjectType: "cli",
			Branch:      "017-update-agent-context-go",
		},
		UpdatedFiles: []agent.UpdateResult{
			{
				FilePath:          "CLAUDE.md",
				Created:           false,
				TechnologiesAdded: []string{"Go 1.25.1", "Cobra CLI v1.10.1"},
			},
			{
				FilePath:          "GEMINI.md",
				Created:           true,
				TechnologiesAdded: []string{"Go 1.25.1", "Cobra CLI v1.10.1", "PostgreSQL"},
			},
		},
		Errors: []string{},
	}

	// Serialize and deserialize to ensure all fields work
	jsonBytes, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("Failed to marshal output: %v", err)
	}

	var parsed agent.CommandOutput
	if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal output: %v", err)
	}

	// Verify key fields
	if parsed.Success != output.Success {
		t.Errorf("Success mismatch")
	}
	if parsed.SpecName != output.SpecName {
		t.Errorf("SpecName mismatch")
	}
	if parsed.PlanPath != output.PlanPath {
		t.Errorf("PlanPath mismatch")
	}
	if parsed.Technologies.Language != output.Technologies.Language {
		t.Errorf("Technologies.Language mismatch")
	}
	if len(parsed.UpdatedFiles) != len(output.UpdatedFiles) {
		t.Errorf("UpdatedFiles count mismatch")
	}
}

// TestCommandWithMockedEnvironment tests command behavior with controlled inputs
func TestCommandWithMockedEnvironment(t *testing.T) {
	// Create temporary directory structure
	tmpDir := t.TempDir()

	// Create specs directory with plan.yaml
	specDir := filepath.Join(tmpDir, "specs", "017-test-feature")
	if err := os.MkdirAll(specDir, 0755); err != nil {
		t.Fatalf("Failed to create spec dir: %v", err)
	}

	planContent := `plan:
  branch: "017-test-feature"
  created: "2025-01-01"
  spec_path: "specs/017-test-feature/spec.yaml"

technical_context:
  language: "Go 1.25.1"
  storage: "None"
  project_type: "cli"
  primary_dependencies:
    - name: "github.com/spf13/cobra"
      version: "v1.10.1"
`
	planPath := filepath.Join(specDir, "plan.yaml")
	if err := os.WriteFile(planPath, []byte(planContent), 0644); err != nil {
		t.Fatalf("Failed to write plan.yaml: %v", err)
	}

	// Test parsing the plan
	planData, err := agent.ParsePlanData(planPath)
	if err != nil {
		t.Fatalf("Failed to parse plan.yaml: %v", err)
	}

	if planData.Language != "Go 1.25.1" {
		t.Errorf("PlanData.Language = %q, want %q", planData.Language, "Go 1.25.1")
	}

	if planData.Framework != "github.com/spf13/cobra v1.10.1" {
		t.Errorf("PlanData.Framework = %q, want %q", planData.Framework, "github.com/spf13/cobra v1.10.1")
	}

	if planData.Branch != "017-test-feature" {
		t.Errorf("PlanData.Branch = %q, want %q", planData.Branch, "017-test-feature")
	}

	// Test updating agent files in the temp directory
	results, err := agent.UpdateAllAgents(tmpDir, planData)
	if err != nil {
		t.Fatalf("UpdateAllAgents() error: %v", err)
	}

	// Should create CLAUDE.md since no agent files exist
	if len(results) != 1 {
		t.Errorf("UpdateAllAgents() returned %d results, want 1", len(results))
	}

	if !results[0].Created {
		t.Error("CLAUDE.md should be marked as created")
	}

	// Verify file was created
	claudePath := filepath.Join(tmpDir, "CLAUDE.md")
	content, err := os.ReadFile(claudePath)
	if err != nil {
		t.Fatalf("Failed to read CLAUDE.md: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "Go 1.25.1") {
		t.Error("CLAUDE.md should contain Go 1.25.1")
	}
	if !strings.Contains(contentStr, "github.com/spf13/cobra v1.10.1") {
		t.Error("CLAUDE.md should contain framework")
	}
}
