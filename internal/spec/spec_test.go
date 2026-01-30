// Package spec_test tests spec detection and directory resolution logic.
// Related: /home/ari/repos/autospec/internal/spec/spec.go
// Tags: spec, detection, directory, resolution

package spec

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectCurrentSpec_FromBranch(t *testing.T) {
	// This test runs against the real git repository
	// It verifies that DetectCurrentSpec returns valid metadata
	// without hardcoding a specific branch (which changes during development)
	specsDir := "./specs" // Use relative path to current repo's specs

	// Get absolute path if we're in repo root
	if cwd, err := os.Getwd(); err == nil {
		// Navigate up from internal/spec to repo root
		repoRoot := filepath.Dir(filepath.Dir(cwd))
		specsDir = filepath.Join(repoRoot, "specs")
	}

	meta, err := DetectCurrentSpec(specsDir)
	if err != nil {
		// If no specs found or detection fails, that's OK for this test
		// (the repo may not have matching specs for current branch)
		t.Skipf("Skipping test: %v", err)
		return
	}

	// Verify we got valid metadata structure
	assert.NotEmpty(t, meta.Number, "spec number should not be empty")
	assert.NotEmpty(t, meta.Name, "spec name should not be empty")
	// Branch may be empty if git detection finds most recent directory instead
	assert.NotEmpty(t, meta.Directory, "directory should not be empty")
	// Verify the directory exists
	_, err = os.Stat(meta.Directory)
	assert.NoError(t, err, "spec directory should exist")
}

// TestDetectCurrentSpec_FromDirectory tests the fallback detection strategy.
// Creates two spec directories with 10ms sleep between to ensure different mod times.
// Verifies that the most recently modified directory is selected (002-new-feature).
func TestDetectCurrentSpec_FromDirectory(t *testing.T) {
	t.Parallel()

	// Create test specs directory
	tmpDir := t.TempDir()
	specsDir := filepath.Join(tmpDir, "specs")
	require.NoError(t, os.MkdirAll(specsDir, 0o755))

	// Create spec directories with different modification times
	oldSpec := filepath.Join(specsDir, "001-old-feature")
	newSpec := filepath.Join(specsDir, "002-new-feature")
	require.NoError(t, os.MkdirAll(oldSpec, 0o755))
	time.Sleep(10 * time.Millisecond) // Ensure different mod times
	require.NoError(t, os.MkdirAll(newSpec, 0o755))

	// Should detect the most recent (002-new-feature)
	meta, err := DetectCurrentSpec(specsDir)
	require.NoError(t, err)
	assert.Equal(t, "002", meta.Number)
	assert.Equal(t, "new-feature", meta.Name)
	assert.Equal(t, newSpec, meta.Directory)
}

func TestDetectCurrentSpec_NoSpecsFound(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	specsDir := filepath.Join(tmpDir, "empty-specs")
	require.NoError(t, os.MkdirAll(specsDir, 0o755))

	_, err := DetectCurrentSpec(specsDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no spec directories found")
}

func TestGetSpecDirectory_ExactMatch(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	specsDir := filepath.Join(tmpDir, "specs")
	specDir := filepath.Join(specsDir, "002-go-binary-migration")
	require.NoError(t, os.MkdirAll(specDir, 0o755))

	result, err := GetSpecDirectory(specsDir, "002-go-binary-migration")
	require.NoError(t, err)
	assert.Equal(t, specDir, result)
}

func TestGetSpecDirectory_NumberMatch(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	specsDir := filepath.Join(tmpDir, "specs")
	specDir := filepath.Join(specsDir, "002-go-binary-migration")
	require.NoError(t, os.MkdirAll(specDir, 0o755))

	result, err := GetSpecDirectory(specsDir, "002")
	require.NoError(t, err)
	assert.Equal(t, specDir, result)
}

func TestGetSpecDirectory_NameMatch(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	specsDir := filepath.Join(tmpDir, "specs")
	specDir := filepath.Join(specsDir, "002-go-binary-migration")
	require.NoError(t, os.MkdirAll(specDir, 0o755))

	result, err := GetSpecDirectory(specsDir, "go-binary-migration")
	require.NoError(t, err)
	assert.Equal(t, specDir, result)
}

func TestGetSpecDirectory_NotFound(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	specsDir := filepath.Join(tmpDir, "specs")
	require.NoError(t, os.MkdirAll(specsDir, 0o755))

	_, err := GetSpecDirectory(specsDir, "999")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestGetSpecDirectory_MultipleMatches(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	specsDir := filepath.Join(tmpDir, "specs")
	spec1 := filepath.Join(specsDir, "001-test-feature")
	spec2 := filepath.Join(specsDir, "002-test-feature")
	require.NoError(t, os.MkdirAll(spec1, 0o755))
	require.NoError(t, os.MkdirAll(spec2, 0o755))

	_, err := GetSpecDirectory(specsDir, "test-feature")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "multiple specs found")
}

func TestUpdateSpecStatus(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		initialYAML   string
		newStatus     string
		withTimestamp bool
		wantUpdated   bool
		wantStatus    string
		wantHasTime   bool
	}{
		"update Draft to Completed": {
			initialYAML: `feature:
  branch: "001-test"
  created: "2025-01-01"
  status: "Draft"
`,
			newStatus:     "Completed",
			withTimestamp: true,
			wantUpdated:   true,
			wantStatus:    "Completed",
			wantHasTime:   true,
		},
		"update InProgress to Completed": {
			initialYAML: `feature:
  branch: "001-test"
  created: "2025-01-01"
  status: "InProgress"
`,
			newStatus:     "Completed",
			withTimestamp: true,
			wantUpdated:   true,
			wantStatus:    "Completed",
			wantHasTime:   true,
		},
		"already Completed is idempotent": {
			initialYAML: `feature:
  branch: "001-test"
  created: "2025-01-01"
  status: "Completed"
  completed_at: "2025-01-01T00:00:00Z"
`,
			newStatus:     "Completed",
			withTimestamp: true,
			wantUpdated:   false,
			wantStatus:    "Completed",
			wantHasTime:   false,
		},
		"update without timestamp": {
			initialYAML: `feature:
  branch: "001-test"
  created: "2025-01-01"
  status: "Draft"
`,
			newStatus:     "InProgress",
			withTimestamp: false,
			wantUpdated:   true,
			wantStatus:    "InProgress",
			wantHasTime:   false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			tmpDir := t.TempDir()
			specDir := filepath.Join(tmpDir, "001-test-feature")
			require.NoError(t, os.MkdirAll(specDir, 0o755))

			specPath := filepath.Join(specDir, "spec.yaml")
			require.NoError(t, os.WriteFile(specPath, []byte(tt.initialYAML), 0o644))

			var completedAt time.Time
			if tt.withTimestamp {
				completedAt = time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC)
			}

			result, err := UpdateSpecStatus(specDir, tt.newStatus, completedAt)
			require.NoError(t, err)

			assert.Equal(t, tt.wantUpdated, result.Updated)
			assert.Equal(t, tt.wantStatus, result.NewStatus)

			if tt.wantHasTime {
				assert.NotEmpty(t, result.CompletedAt)
			}

			// Read back and verify
			data, err := os.ReadFile(specPath)
			require.NoError(t, err)
			content := string(data)
			// Status may be quoted or unquoted depending on YAML serialization
			assert.True(t, containsStatus(content, tt.wantStatus),
				"expected status %s in content:\n%s", tt.wantStatus, content)
		})
	}
}

// containsStatus checks if the YAML content contains the expected status value
func containsStatus(content, status string) bool {
	// Check both quoted and unquoted forms
	return strings.Contains(content, "status: "+status) ||
		strings.Contains(content, "status: \""+status+"\"")
}

func TestMarkSpecCompleted(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	specDir := filepath.Join(tmpDir, "001-test-feature")
	require.NoError(t, os.MkdirAll(specDir, 0o755))

	specYAML := `feature:
  branch: "001-test"
  created: "2025-01-01"
  status: "Draft"
  input: "test input"
`
	specPath := filepath.Join(specDir, "spec.yaml")
	require.NoError(t, os.WriteFile(specPath, []byte(specYAML), 0o644))

	result, err := MarkSpecCompleted(specDir)
	require.NoError(t, err)

	assert.True(t, result.Updated)
	assert.Equal(t, "Draft", result.PreviousStatus)
	assert.Equal(t, "Completed", result.NewStatus)
	assert.NotEmpty(t, result.CompletedAt)

	// Read back and verify
	data, err := os.ReadFile(specPath)
	require.NoError(t, err)
	content := string(data)
	assert.True(t, containsStatus(content, "Completed"),
		"expected status Completed in content:\n%s", content)
	assert.Contains(t, content, "completed_at:")
	// Verify input field preserved (may be quoted)
	assert.True(t, strings.Contains(content, "input: test input") ||
		strings.Contains(content, "input: \"test input\""),
		"expected input field preserved in content:\n%s", content)
}

func TestUpdateSpecStatus_FileNotFound(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	specDir := filepath.Join(tmpDir, "nonexistent")

	_, err := UpdateSpecStatus(specDir, "Completed", time.Now())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read spec.yaml")
}

func TestUpdateSpecStatus_MissingFeatureSection(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	specDir := filepath.Join(tmpDir, "001-test-feature")
	require.NoError(t, os.MkdirAll(specDir, 0o755))

	// YAML without feature section
	specYAML := `metadata:
  version: "1.0"
`
	specPath := filepath.Join(specDir, "spec.yaml")
	require.NoError(t, os.WriteFile(specPath, []byte(specYAML), 0o644))

	_, err := UpdateSpecStatus(specDir, "Completed", time.Now())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "feature section not found")
}

func TestUpdateSpecStatus_MissingStatusField(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	specDir := filepath.Join(tmpDir, "001-test-feature")
	require.NoError(t, os.MkdirAll(specDir, 0o755))

	// YAML with feature section but no status field
	specYAML := `feature:
  branch: "001-test"
  created: "2025-01-01"
`
	specPath := filepath.Join(specDir, "spec.yaml")
	require.NoError(t, os.WriteFile(specPath, []byte(specYAML), 0o644))

	_, err := UpdateSpecStatus(specDir, "Completed", time.Now())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "status field not found")
}

// TestGetSpecMetadata tests the GetSpecMetadata function.
// This includes regression tests for the leading dash bug (issue #111) where
// spec names like "110-dag-spec-validation" must be parsed correctly:
// - Number = "110" (not empty)
// - Name = "dag-spec-validation" (not "110-dag-spec-validation")
func TestGetSpecMetadata(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		specDirName    string
		identifier     string
		wantNumber     string
		wantName       string
		wantErr        bool
		wantErrContain string
	}{
		"exact match": {
			specDirName: "002-go-binary-migration",
			identifier:  "002-go-binary-migration",
			wantNumber:  "002",
			wantName:    "go-binary-migration",
			wantErr:     false,
		},
		"number only match": {
			specDirName: "003-test-feature",
			identifier:  "003",
			wantNumber:  "003",
			wantName:    "test-feature",
			wantErr:     false,
		},
		"name only match": {
			specDirName: "004-feature-name",
			identifier:  "feature-name",
			wantNumber:  "004",
			wantName:    "feature-name",
			wantErr:     false,
		},
		"not found": {
			specDirName:    "001-existing",
			identifier:     "999-nonexistent",
			wantErr:        true,
			wantErrContain: "not found",
		},
		// Regression tests for leading dash bug (issue #111)
		// Verifies that Number and Name are correctly parsed, not left empty
		"regression: 110-dag-spec-validation parses correctly": {
			specDirName: "110-dag-spec-validation",
			identifier:  "110-dag-spec-validation",
			wantNumber:  "110",
			wantName:    "dag-spec-validation",
			wantErr:     false,
		},
		// Edge case: feature name starts with a number (e.g., 2fa)
		"edge case: feature name starting with number": {
			specDirName: "110-2fa-implementation",
			identifier:  "110-2fa-implementation",
			wantNumber:  "110",
			wantName:    "2fa-implementation",
			wantErr:     false,
		},
		// Boundary test: lowest valid spec number
		"boundary: spec number 001": {
			specDirName: "001-first-feature",
			identifier:  "001-first-feature",
			wantNumber:  "001",
			wantName:    "first-feature",
			wantErr:     false,
		},
		// Boundary test: highest valid spec number
		"boundary: spec number 999": {
			specDirName: "999-last-feature",
			identifier:  "999-last-feature",
			wantNumber:  "999",
			wantName:    "last-feature",
			wantErr:     false,
		},
		// Edge case: spec name with multiple dashes
		"edge case: multiple dashes in name": {
			specDirName: "111-fix-spec-path-prefix",
			identifier:  "111-fix-spec-path-prefix",
			wantNumber:  "111",
			wantName:    "fix-spec-path-prefix",
			wantErr:     false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			tmpDir := t.TempDir()
			specsDir := filepath.Join(tmpDir, "specs")
			specDir := filepath.Join(specsDir, tt.specDirName)
			require.NoError(t, os.MkdirAll(specDir, 0o755))

			meta, err := GetSpecMetadata(specsDir, tt.identifier)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.wantErrContain != "" {
					assert.Contains(t, err.Error(), tt.wantErrContain)
				}
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantNumber, meta.Number)
			assert.Equal(t, tt.wantName, meta.Name)
			assert.Equal(t, specDir, meta.Directory)
		})
	}
}

// TestGetSpecMetadata_InvalidDirName tests GetSpecMetadata with invalid directory name
func TestGetSpecMetadata_InvalidDirName(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	specsDir := filepath.Join(tmpDir, "specs")
	// Create a directory that doesn't match the spec pattern
	invalidDir := filepath.Join(specsDir, "invalid-no-number")
	require.NoError(t, os.MkdirAll(invalidDir, 0o755))

	_, err := GetSpecMetadata(specsDir, "invalid-no-number")
	// This will fail because the directory doesn't match the pattern *-name
	// It won't find it with the glob patterns
	assert.Error(t, err)
}

// TestMetadata_FormatInfo tests the FormatInfo method
func TestMetadata_FormatInfo(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		metadata    Metadata
		wantContain string
	}{
		"git branch detection": {
			metadata: Metadata{
				Directory: "specs/001-test",
				Detection: DetectionGitBranch,
			},
			wantContain: "via git branch",
		},
		"fallback detection": {
			metadata: Metadata{
				Directory: "specs/002-feature",
				Detection: DetectionFallbackRecent,
			},
			wantContain: "fallback - most recent",
		},
		"env var detection": {
			metadata: Metadata{
				Directory: "specs/003-feature",
				Detection: DetectionEnvVar,
			},
			wantContain: "via SPECIFY_FEATURE env",
		},
		"explicit detection": {
			metadata: Metadata{
				Directory: "specs/004-feature",
				Detection: DetectionExplicit,
			},
			wantContain: "explicitly specified",
		},
		"unknown detection": {
			metadata: Metadata{
				Directory: "specs/005-feature",
				Detection: DetectionMethod("unknown"),
			},
			wantContain: "auto-detected",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			info := tt.metadata.FormatInfo()
			assert.Contains(t, info, tt.wantContain)
			assert.Contains(t, info, tt.metadata.Directory)
			assert.Contains(t, info, "✓ Using spec:")
		})
	}
}

// TestDetectCurrentSpec_InvalidDirName tests handling of directory names without valid spec pattern
func TestDetectCurrentSpec_InvalidDirName(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	specsDir := filepath.Join(tmpDir, "specs")

	// Create only invalid directory (matches *-* pattern but not 3-digit prefix)
	invalidDir := filepath.Join(specsDir, "invalid-no-number-prefix")
	require.NoError(t, os.MkdirAll(invalidDir, 0o755))

	// Should error since no valid spec pattern matches
	_, err := DetectCurrentSpec(specsDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "could not parse spec directory name")
}

// TestDetectCurrentSpec_FileNotDir tests that files are skipped, only directories match
func TestDetectCurrentSpec_FileNotDir(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	specsDir := filepath.Join(tmpDir, "specs")
	require.NoError(t, os.MkdirAll(specsDir, 0o755))

	// Create a file instead of a directory
	filePath := filepath.Join(specsDir, "001-fake-spec")
	require.NoError(t, os.WriteFile(filePath, []byte("not a directory"), 0o644))

	// Create an actual directory
	specDir := filepath.Join(specsDir, "002-real-spec")
	require.NoError(t, os.MkdirAll(specDir, 0o755))

	meta, err := DetectCurrentSpec(specsDir)
	require.NoError(t, err)

	// Should skip the file and find the directory
	assert.Equal(t, "002", meta.Number)
	assert.Equal(t, "real-spec", meta.Name)
}

// TestGetSpecDirectory_MultipleNumberMatches tests multiple specs with same number
func TestGetSpecDirectory_MultipleNumberMatches(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	specsDir := filepath.Join(tmpDir, "specs")

	// Create two directories with same number but different names
	// This should not normally happen, but we test the error case
	spec1 := filepath.Join(specsDir, "001-feature-a")
	spec2 := filepath.Join(specsDir, "001-feature-b")
	require.NoError(t, os.MkdirAll(spec1, 0o755))
	require.NoError(t, os.MkdirAll(spec2, 0o755))

	_, err := GetSpecDirectory(specsDir, "001")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "multiple specs found")
}

// TestUpdateSpecStatus_RemoveCompletedAt tests removing completed_at when changing status away from Completed
func TestUpdateSpecStatus_RemoveCompletedAt(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	specDir := filepath.Join(tmpDir, "001-test-feature")
	require.NoError(t, os.MkdirAll(specDir, 0o755))

	// Start with a completed spec
	specYAML := `feature:
  branch: "001-test"
  created: "2025-01-01"
  status: "Completed"
  completed_at: "2025-01-15T12:00:00Z"
`
	specPath := filepath.Join(specDir, "spec.yaml")
	require.NoError(t, os.WriteFile(specPath, []byte(specYAML), 0o644))

	// Update to Draft (should remove completed_at)
	result, err := UpdateSpecStatus(specDir, "Draft", time.Time{})
	require.NoError(t, err)

	assert.True(t, result.Updated)
	assert.Equal(t, "Completed", result.PreviousStatus)
	assert.Equal(t, "Draft", result.NewStatus)

	// Read back and verify completed_at was removed
	data, err := os.ReadFile(specPath)
	require.NoError(t, err)
	content := string(data)
	assert.True(t, containsStatus(content, "Draft"),
		"expected status Draft in content:\n%s", content)
	assert.NotContains(t, content, "completed_at:")
}

// TestUpdateSpecStatus_InvalidYAML tests handling of invalid YAML
func TestUpdateSpecStatus_InvalidYAML(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	specDir := filepath.Join(tmpDir, "001-test-feature")
	require.NoError(t, os.MkdirAll(specDir, 0o755))

	// Write invalid YAML
	specPath := filepath.Join(specDir, "spec.yaml")
	require.NoError(t, os.WriteFile(specPath, []byte("this is not: valid: yaml:\n  - bad"), 0o644))

	_, err := UpdateSpecStatus(specDir, "Completed", time.Now())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse spec.yaml")
}

// TestUpdateSpecStatus_FeatureNotMapping tests handling feature section that's not a mapping
func TestUpdateSpecStatus_FeatureNotMapping(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	specDir := filepath.Join(tmpDir, "001-test-feature")
	require.NoError(t, os.MkdirAll(specDir, 0o755))

	// Write YAML where feature is a scalar, not a mapping
	specYAML := `feature: "just a string"
`
	specPath := filepath.Join(specDir, "spec.yaml")
	require.NoError(t, os.WriteFile(specPath, []byte(specYAML), 0o644))

	_, err := UpdateSpecStatus(specDir, "Completed", time.Now())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "feature section is not a mapping")
}
