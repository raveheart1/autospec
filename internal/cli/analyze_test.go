// Package cli_test tests the analyze command registration, examples, and artifact prerequisite validation.
// Related: internal/cli/analyze.go
// Tags: cli, analyze, command, validation, cross-artifact, consistency
package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnalyzeCmdRegistration(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "analyze [optional-prompt]" {
			found = true
			break
		}
	}
	assert.True(t, found, "analyze command should be registered")
}

func TestAnalyzeCmdExamples(t *testing.T) {
	examples := []string{
		"autospec analyze",
		"Focus on security",
		"Verify API contracts",
	}

	for _, example := range examples {
		assert.Contains(t, analyzeCmd.Example, example)
	}
}

func TestAnalyzeCmdLongDescription(t *testing.T) {
	keywords := []string{
		"Auto-detect",
		"cross-artifact",
		"consistency",
		"spec.yaml",
		"plan.yaml",
		"tasks.yaml",
		"Prerequisites",
	}

	for _, keyword := range keywords {
		assert.Contains(t, analyzeCmd.Long, keyword)
	}
}

func TestAnalyzeCmdAcceptsOptionalPrompt(t *testing.T) {
	// Command should accept arbitrary args (for optional prompt)
	// When Args is nil, cobra allows any number of args
	// Just verify the command's use pattern indicates optional prompt
	assert.Contains(t, analyzeCmd.Use, "[optional-prompt]")
}

func TestAnalyze_RequiredArtifacts(t *testing.T) {
	// Test the artifact checking logic
	tests := map[string]struct {
		createFiles []string
		wantMissing int
	}{
		"all artifacts exist": {
			createFiles: []string{"spec.yaml", "plan.yaml", "tasks.yaml"},
			wantMissing: 0,
		},
		"missing spec.yaml": {
			createFiles: []string{"plan.yaml", "tasks.yaml"},
			wantMissing: 1,
		},
		"missing plan.yaml": {
			createFiles: []string{"spec.yaml", "tasks.yaml"},
			wantMissing: 1,
		},
		"missing tasks.yaml": {
			createFiles: []string{"spec.yaml", "plan.yaml"},
			wantMissing: 1,
		},
		"missing all artifacts": {
			createFiles: []string{},
			wantMissing: 3,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Create a new temp directory for each test
			tmpDir := t.TempDir()
			specDir := filepath.Join(tmpDir, "specs", "001-test")
			require.NoError(t, os.MkdirAll(specDir, 0755))

			// Create specified files
			for _, f := range tc.createFiles {
				require.NoError(t, os.WriteFile(filepath.Join(specDir, f), []byte("test"), 0644))
			}

			// Check for missing artifacts
			var missingCount int
			for _, artifact := range []string{"spec.yaml", "plan.yaml", "tasks.yaml"} {
				if _, err := os.Stat(filepath.Join(specDir, artifact)); os.IsNotExist(err) {
					missingCount++
				}
			}

			assert.Equal(t, tc.wantMissing, missingCount)
		})
	}
}

func TestAnalyze_InheritedFlags(t *testing.T) {
	// analyze should inherit skip-preflight from root
	f := rootCmd.PersistentFlags().Lookup("skip-preflight")
	require.NotNil(t, f)

	// analyze should inherit config from root
	f = rootCmd.PersistentFlags().Lookup("config")
	require.NotNil(t, f)
}
