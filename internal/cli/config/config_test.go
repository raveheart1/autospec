// Package config tests CLI configuration commands for autospec.
// Related: internal/cli/config/config_cmd.go
// Tags: config, cli, show

package config

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestRunConfigShow_YAMLOutput(t *testing.T) {
	// Create isolated command
	cmd := &cobra.Command{
		Use:  "show",
		RunE: runConfigShow,
	}
	cmd.Flags().Bool("json", false, "Output in JSON format")
	cmd.Flags().Bool("yaml", true, "Output in YAML format")

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	err := cmd.Execute()
	assert.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Configuration Sources")
	// YAML output should have key: value format
	assert.Contains(t, output, "agent_preset:")
}

func TestRunConfigShow_JSONOutput(t *testing.T) {
	// Create isolated command
	cmd := &cobra.Command{
		Use:  "show",
		RunE: runConfigShow,
	}
	cmd.Flags().Bool("json", false, "Output in JSON format")
	cmd.Flags().Bool("yaml", true, "Output in YAML format")
	_ = cmd.Flags().Set("json", "true")

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	err := cmd.Execute()
	assert.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Configuration Sources")
	// JSON output should have braces
	assert.Contains(t, output, "{")
	assert.Contains(t, output, "}")
}

func TestConfigShowCmd_OutputFormats(t *testing.T) {
	tests := map[string]struct {
		jsonFlag bool
		wantYAML bool
	}{
		"yaml output by default": {
			jsonFlag: false,
			wantYAML: true,
		},
		"json output when flag set": {
			jsonFlag: true,
			wantYAML: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Create a fresh command for each test
			cmd := &cobra.Command{
				Use:  "show",
				RunE: runConfigShow,
			}
			cmd.Flags().Bool("json", false, "Output in JSON format")
			cmd.Flags().Bool("yaml", true, "Output in YAML format")

			var buf bytes.Buffer
			cmd.SetOut(&buf)
			cmd.SetErr(&buf)

			if tt.jsonFlag {
				_ = cmd.Flags().Set("json", "true")
			}

			err := cmd.Execute()
			assert.NoError(t, err)

			output := buf.String()
			if tt.wantYAML {
				assert.Contains(t, output, "agent_preset:")
			} else {
				assert.Contains(t, output, "{")
			}
		})
	}
}

func TestConfigCmd_SubcommandExecution(t *testing.T) {
	// Verify that config command has subcommands properly set up
	subcommands := configCmd.Commands()

	// Should have show subcommand
	found := make(map[string]bool)
	for _, cmd := range subcommands {
		found[cmd.Name()] = true
	}

	assert.True(t, found["show"], "Should have show subcommand")
}

func TestConfigShowCmd_HasRunE(t *testing.T) {
	assert.NotNil(t, configShowCmd.RunE)
}

func TestConfigShow_VerificationBlock(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		jsonFlag     bool
		wantContains []string
	}{
		"yaml output includes verification block": {
			jsonFlag: false,
			wantContains: []string{
				"verification:",
				"level:",
			},
		},
		"json output includes verification settings": {
			jsonFlag: true,
			wantContains: []string{
				`"verification"`,
				`"Level"`,
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			cmd := &cobra.Command{
				Use:  "show",
				RunE: runConfigShow,
			}
			cmd.Flags().Bool("json", false, "Output in JSON format")
			cmd.Flags().Bool("yaml", true, "Output in YAML format")

			var buf bytes.Buffer
			cmd.SetOut(&buf)
			cmd.SetErr(&buf)

			if tt.jsonFlag {
				_ = cmd.Flags().Set("json", "true")
			}

			err := cmd.Execute()
			assert.NoError(t, err)

			output := buf.String()
			for _, want := range tt.wantContains {
				assert.Contains(t, output, want, "Output should contain %q", want)
			}
		})
	}
}

func TestConfigShow_VerificationDefaultValues(t *testing.T) {
	t.Parallel()

	cmd := &cobra.Command{
		Use:  "show",
		RunE: runConfigShow,
	}
	cmd.Flags().Bool("json", false, "Output in JSON format")
	cmd.Flags().Bool("yaml", true, "Output in YAML format")

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	err := cmd.Execute()
	assert.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "level: basic", "Default verification level should be 'basic'")
	assert.Contains(t, output, "mutation_threshold:", "Mutation threshold should be visible")
	assert.Contains(t, output, "coverage_threshold:", "Coverage threshold should be visible")
	assert.Contains(t, output, "complexity_max:", "Complexity max should be visible")
}

func TestConfigShow_VerificationJSONStructure(t *testing.T) {
	t.Parallel()

	cmd := &cobra.Command{
		Use:  "show",
		RunE: runConfigShow,
	}
	cmd.Flags().Bool("json", false, "Output in JSON format")
	cmd.Flags().Bool("yaml", true, "Output in YAML format")
	_ = cmd.Flags().Set("json", "true")

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	err := cmd.Execute()
	assert.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, `"verification"`, "JSON output should contain verification key")
	assert.Contains(t, output, `"Level"`, "JSON output should contain Level key")
	assert.Contains(t, output, `"MutationThreshold"`, "JSON output should contain MutationThreshold")
	assert.Contains(t, output, `"CoverageThreshold"`, "JSON output should contain CoverageThreshold")
	assert.Contains(t, output, `"ComplexityMax"`, "JSON output should contain ComplexityMax")
}
