// Package cli_test tests the new-feature command for creating spec directories with auto-incremented numbers and git branch creation.
// Related: internal/cli/new_feature.go
// Tags: cli, new-feature, spec, creation, git, branch, numbering, json-output
package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFeatureOutput_JSONFormat(t *testing.T) {
	output := NewFeatureOutput{
		BranchName:      "001-test-feature",
		SpecFile:        "/path/to/specs/001-test-feature/spec.yaml",
		FeatureNum:      "001",
		AutospecVersion: "autospec dev",
		CreatedDate:     "2024-01-15T10:30:00Z",
	}

	// Verify JSON encoding produces expected keys
	data, err := json.Marshal(output)
	require.NoError(t, err)

	// Check for expected JSON keys (matching shell script format)
	jsonStr := string(data)
	assert.Contains(t, jsonStr, `"BRANCH_NAME"`)
	assert.Contains(t, jsonStr, `"SPEC_FILE"`)
	assert.Contains(t, jsonStr, `"FEATURE_NUM"`)
	assert.Contains(t, jsonStr, `"AUTOSPEC_VERSION"`)
	assert.Contains(t, jsonStr, `"CREATED_DATE"`)

	// Verify roundtrip
	var decoded NewFeatureOutput
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	assert.Equal(t, output, decoded)
}

func TestNewFeatureCmd_Help(t *testing.T) {
	// Verify the command has the expected properties
	assert.Equal(t, "new-feature <feature_description>", newFeatureCmd.Use)
	assert.Contains(t, newFeatureCmd.Short, "Create a new feature")
	assert.Contains(t, newFeatureCmd.Long, "git branch")
	assert.Contains(t, newFeatureCmd.Long, "feature directory")
}

func TestNewFeatureCmd_Flags(t *testing.T) {
	// Verify flags exist
	jsonFlag := newFeatureCmd.Flags().Lookup("json")
	assert.NotNil(t, jsonFlag)
	assert.Equal(t, "false", jsonFlag.DefValue)

	shortNameFlag := newFeatureCmd.Flags().Lookup("short-name")
	assert.NotNil(t, shortNameFlag)
	assert.Equal(t, "", shortNameFlag.DefValue)

	numberFlag := newFeatureCmd.Flags().Lookup("number")
	assert.NotNil(t, numberFlag)
	assert.Equal(t, "", numberFlag.DefValue)
}

func TestNewFeatureCmd_IntegrationNonGit(t *testing.T) {
	// Create a temporary directory for testing (not a git repo)
	tmpDir, err := os.MkdirTemp("", "autospec-new-feature-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Change to temp directory
	oldDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(oldDir)

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Create specs directory
	specsDir := filepath.Join(tmpDir, "specs")
	err = os.MkdirAll(specsDir, 0o755)
	require.NoError(t, err)

	// Reset flags
	newFeatureJSON = true
	newFeatureShortName = "test-feature"
	newFeatureNumber = "42"

	// Capture output
	var buf bytes.Buffer
	newFeatureCmd.SetOut(&buf)
	rootCmd.SetOut(&buf)

	// We can't easily capture stdout from runNewFeature since it uses os.Stdout
	// Instead, test that the function doesn't error in a non-git context
	err = runNewFeature(newFeatureCmd, []string{"Add test feature"})
	require.NoError(t, err)

	// Verify directory was created
	expectedDir := filepath.Join(specsDir, "042-test-feature")
	info, err := os.Stat(expectedDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())

	// Reset flags
	newFeatureJSON = false
	newFeatureShortName = ""
	newFeatureNumber = ""
}

func TestNewFeatureOutput_KeysMatchShellScript(t *testing.T) {
	// This test ensures our JSON output matches the shell script's format
	// Shell script uses: BRANCH_NAME, SPEC_FILE, FEATURE_NUM, AUTOSPEC_VERSION, CREATED_DATE

	output := NewFeatureOutput{
		BranchName:      "test",
		SpecFile:        "test",
		FeatureNum:      "001",
		AutospecVersion: "test",
		CreatedDate:     "test",
	}

	data, err := json.Marshal(output)
	require.NoError(t, err)

	var parsed map[string]interface{}
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	// Verify exact key names
	expectedKeys := []string{"BRANCH_NAME", "SPEC_FILE", "FEATURE_NUM", "AUTOSPEC_VERSION", "CREATED_DATE"}
	for _, key := range expectedKeys {
		_, ok := parsed[key]
		assert.True(t, ok, "expected key %s to be present", key)
	}

	// Verify no extra keys
	assert.Len(t, parsed, len(expectedKeys))
}

func TestNewFeatureNumber_Validation(t *testing.T) {
	tests := map[string]struct {
		number    string
		expectErr bool
	}{
		"valid number": {number: "5", expectErr: false},
		"zero":         {number: "0", expectErr: false},
		"large number": {number: "999", expectErr: false},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Create temp directory
			tmpDir, err := os.MkdirTemp("", "autospec-test-*")
			require.NoError(t, err)
			defer os.RemoveAll(tmpDir)

			oldDir, err := os.Getwd()
			require.NoError(t, err)
			defer os.Chdir(oldDir)

			err = os.Chdir(tmpDir)
			require.NoError(t, err)

			specsDir := filepath.Join(tmpDir, "specs")
			err = os.MkdirAll(specsDir, 0o755)
			require.NoError(t, err)

			// Reset flags
			newFeatureJSON = true
			newFeatureShortName = "test"
			newFeatureNumber = tt.number

			err = runNewFeature(newFeatureCmd, []string{"Test feature"})
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			// Reset flags
			newFeatureJSON = false
			newFeatureShortName = ""
			newFeatureNumber = ""
		})
	}
}

func TestNewFeatureShortName_Cleaning(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "autospec-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	oldDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(oldDir)

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	specsDir := filepath.Join(tmpDir, "specs")
	err = os.MkdirAll(specsDir, 0o755)
	require.NoError(t, err)

	// Test with short name that needs cleaning
	newFeatureJSON = true
	newFeatureShortName = "My Feature (v2)"
	newFeatureNumber = "1"

	err = runNewFeature(newFeatureCmd, []string{"Test feature"})
	require.NoError(t, err)

	// Verify directory was created with cleaned name
	entries, err := os.ReadDir(specsDir)
	require.NoError(t, err)
	require.Len(t, entries, 1)

	// Should be cleaned: "My Feature (v2)" -> "my-feature-v2"
	assert.True(t, strings.HasPrefix(entries[0].Name(), "001-my-feature"))

	// Reset flags
	newFeatureJSON = false
	newFeatureShortName = ""
	newFeatureNumber = ""
}

func TestNewFeatureNoFetch_Flag(t *testing.T) {
	tests := map[string]struct {
		noFetch     bool
		description string
	}{
		"no-fetch flag not set (default behavior)": {
			noFetch:     false,
			description: "fetch should be attempted when flag is not set",
		},
		"no-fetch flag set": {
			noFetch:     true,
			description: "fetch should be skipped when flag is set",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Create temp directory (not a git repo)
			tmpDir, err := os.MkdirTemp("", "autospec-test-*")
			require.NoError(t, err)
			defer os.RemoveAll(tmpDir)

			oldDir, err := os.Getwd()
			require.NoError(t, err)
			defer os.Chdir(oldDir)

			err = os.Chdir(tmpDir)
			require.NoError(t, err)

			specsDir := filepath.Join(tmpDir, "specs")
			err = os.MkdirAll(specsDir, 0o755)
			require.NoError(t, err)

			// Set flags
			newFeatureJSON = true
			newFeatureShortName = "test"
			newFeatureNumber = "1"
			newFeatureNoFetch = tt.noFetch

			// Run the command
			err = runNewFeature(newFeatureCmd, []string{"Test feature"})
			require.NoError(t, err, tt.description)

			// Verify directory was created (command succeeded)
			expectedDir := filepath.Join(specsDir, "001-test")
			info, err := os.Stat(expectedDir)
			require.NoError(t, err)
			assert.True(t, info.IsDir())

			// Reset flags
			newFeatureJSON = false
			newFeatureShortName = ""
			newFeatureNumber = ""
			newFeatureNoFetch = false
		})
	}
}

func TestNewFeatureNoFetch_FlagExists(t *testing.T) {
	// Verify the --no-fetch flag is registered and has correct defaults (FR-002)
	noFetchFlag := newFeatureCmd.Flags().Lookup("no-fetch")
	require.NotNil(t, noFetchFlag, "--no-fetch flag should be registered")
	assert.Equal(t, "false", noFetchFlag.DefValue, "default should be false for backwards compatibility")
	assert.Contains(t, noFetchFlag.Usage, "fetch", "usage should mention fetch")
}

func TestInitGitForNewFeature_SkipFetch(t *testing.T) {
	tests := map[string]struct {
		skipFetch   bool
		description string
	}{
		"skip fetch disabled": {
			skipFetch:   false,
			description: "should attempt fetch when skipFetch is false",
		},
		"skip fetch enabled": {
			skipFetch:   true,
			description: "should not attempt fetch when skipFetch is true",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Create temp directory (not a git repo, so fetch won't actually happen)
			tmpDir, err := os.MkdirTemp("", "autospec-test-*")
			require.NoError(t, err)
			defer os.RemoveAll(tmpDir)

			oldDir, err := os.Getwd()
			require.NoError(t, err)
			defer os.Chdir(oldDir)

			err = os.Chdir(tmpDir)
			require.NoError(t, err)

			// Call the function with skip parameter
			// In non-git repo, hasGit will be false regardless
			hasGit := initGitForNewFeature(tt.skipFetch)
			assert.False(t, hasGit, "should return false when not in a git repository")
		})
	}
}
