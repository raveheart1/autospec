// Package yaml_test tests YAML metadata extraction, version parsing, and artifact type validation.
// Related: internal/yaml/meta.go
// Tags: yaml, metadata, version, artifact-type, parsing
package yaml

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractMeta_ValidMeta(t *testing.T) {
	input := `_meta:
  version: "1.0.0"
  generator: "autospec"
  generator_version: "0.1.0"
  created: "2025-12-13T10:30:00Z"
  artifact_type: "spec"
feature:
  branch: "test-branch"`

	meta, err := ExtractMeta(strings.NewReader(input))
	require.NoError(t, err)

	assert.Equal(t, "1.0.0", meta.Version)
	assert.Equal(t, "autospec", meta.Generator)
	assert.Equal(t, "0.1.0", meta.GeneratorVersion)
	assert.Equal(t, "2025-12-13T10:30:00Z", meta.Created)
	assert.Equal(t, "spec", meta.ArtifactType)
}

func TestExtractMeta_MissingMeta(t *testing.T) {
	input := `feature:
  branch: "test-branch"`

	meta, err := ExtractMeta(strings.NewReader(input))
	require.NoError(t, err, "should not error on missing _meta")
	assert.Empty(t, meta.Version, "version should be empty")
	assert.Empty(t, meta.ArtifactType, "artifact_type should be empty")
}

func TestExtractMeta_PartialMeta(t *testing.T) {
	input := `_meta:
  version: "1.0.0"
  artifact_type: "plan"
plan:
  branch: "test"`

	meta, err := ExtractMeta(strings.NewReader(input))
	require.NoError(t, err)

	assert.Equal(t, "1.0.0", meta.Version)
	assert.Equal(t, "plan", meta.ArtifactType)
	assert.Empty(t, meta.Generator, "generator should be empty")
}

func TestExtractMeta_InvalidYAML(t *testing.T) {
	input := `_meta:
  version: "1.0.0"
  bad_indent: error`

	_, err := ExtractMeta(strings.NewReader(input))
	// May or may not error depending on how strict parsing is
	// The important thing is it doesn't panic
	_ = err
}

func TestParseVersion_Valid(t *testing.T) {
	tests := map[string]struct {
		input    string
		expected Version
	}{
		"1.0.0":    {input: "1.0.0", expected: Version{Major: 1, Minor: 0, Patch: 0}},
		"2.3.4":    {input: "2.3.4", expected: Version{Major: 2, Minor: 3, Patch: 4}},
		"0.1.0":    {input: "0.1.0", expected: Version{Major: 0, Minor: 1, Patch: 0}},
		"10.20.30": {input: "10.20.30", expected: Version{Major: 10, Minor: 20, Patch: 30}},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			v, err := ParseVersion(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, v)
		})
	}
}

func TestParseVersion_Invalid(t *testing.T) {
	tests := map[string]struct {
		input string
	}{
		"empty string":     {input: ""},
		"two parts only":   {input: "1.0"},
		"single part":      {input: "1"},
		"with v prefix":    {input: "v1.0.0"},
		"four parts":       {input: "1.0.0.0"},
		"non-numeric":      {input: "a.b.c"},
		"with beta suffix": {input: "1.0.0-beta"},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := ParseVersion(tt.input)
			assert.Error(t, err, "should error on invalid version")
		})
	}
}

func TestVersion_Compare(t *testing.T) {
	tests := map[string]struct {
		v1       Version
		v2       Version
		expected int
	}{
		"equal":         {v1: Version{1, 0, 0}, v2: Version{1, 0, 0}, expected: 0},
		"major greater": {v1: Version{2, 0, 0}, v2: Version{1, 0, 0}, expected: 1},
		"major less":    {v1: Version{1, 0, 0}, v2: Version{2, 0, 0}, expected: -1},
		"minor greater": {v1: Version{1, 2, 0}, v2: Version{1, 1, 0}, expected: 1},
		"minor less":    {v1: Version{1, 1, 0}, v2: Version{1, 2, 0}, expected: -1},
		"patch greater": {v1: Version{1, 0, 2}, v2: Version{1, 0, 1}, expected: 1},
		"patch less":    {v1: Version{1, 0, 1}, v2: Version{1, 0, 2}, expected: -1},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := tt.v1.Compare(tt.v2)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestVersion_String(t *testing.T) {
	v := Version{Major: 1, Minor: 2, Patch: 3}
	assert.Equal(t, "1.2.3", v.String())
}

func TestIsMajorVersionMismatch(t *testing.T) {
	tests := map[string]struct {
		v1       string
		v2       string
		expected bool
	}{
		"same version":          {v1: "1.0.0", v2: "1.0.0", expected: false},
		"same major diff minor": {v1: "1.0.0", v2: "1.1.0", expected: false},
		"v1 major greater":      {v1: "2.0.0", v2: "1.0.0", expected: true},
		"v2 major greater":      {v1: "1.0.0", v2: "2.0.0", expected: true},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := IsMajorVersionMismatch(tt.v1, tt.v2)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestIsMajorVersionMismatch_InvalidVersions tests IsMajorVersionMismatch with invalid input
func TestIsMajorVersionMismatch_InvalidVersions(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		v1       string
		v2       string
		expected bool
	}{
		"v1 invalid":   {v1: "invalid", v2: "1.0.0", expected: false},
		"v2 invalid":   {v1: "1.0.0", v2: "invalid", expected: false},
		"both invalid": {v1: "invalid", v2: "also-invalid", expected: false},
		"empty v1":     {v1: "", v2: "1.0.0", expected: false},
		"empty v2":     {v1: "1.0.0", v2: "", expected: false},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			result := IsMajorVersionMismatch(tt.v1, tt.v2)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestExtractMetaFromBytes tests the ExtractMetaFromBytes function
func TestExtractMetaFromBytes(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		input       string
		wantVersion string
		wantType    string
		wantGen     string
		wantGenVer  string
		wantCreated string
		wantErr     bool
	}{
		"complete meta": {
			input: `_meta:
  version: "1.0.0"
  generator: "autospec"
  generator_version: "0.2.0"
  created: "2025-12-17T10:00:00Z"
  artifact_type: "spec"
feature:
  branch: "test"`,
			wantVersion: "1.0.0",
			wantType:    "spec",
			wantGen:     "autospec",
			wantGenVer:  "0.2.0",
			wantCreated: "2025-12-17T10:00:00Z",
			wantErr:     false,
		},
		"partial meta": {
			input: `_meta:
  version: "2.0.0"
  artifact_type: "plan"`,
			wantVersion: "2.0.0",
			wantType:    "plan",
			wantErr:     false,
		},
		"no meta section": {
			input: `feature:
  branch: "test"`,
			wantVersion: "",
			wantType:    "",
			wantErr:     false,
		},
		"empty input": {
			input:   "",
			wantErr: false,
		},
		"only whitespace": {
			input:   "   \n\n   ",
			wantErr: false,
		},
		"invalid yaml": {
			input: `_meta:
  version: "1.0.0"
    bad_indent: this is wrong`,
			wantErr: true,
		},
		"meta at bottom": {
			input: `feature:
  branch: "test"
_meta:
  version: "1.0.0"
  artifact_type: "tasks"`,
			wantVersion: "1.0.0",
			wantType:    "tasks",
			wantErr:     false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			meta, err := ExtractMetaFromBytes([]byte(tt.input))

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantVersion, meta.Version)
			assert.Equal(t, tt.wantType, meta.ArtifactType)
			assert.Equal(t, tt.wantGen, meta.Generator)
			assert.Equal(t, tt.wantGenVer, meta.GeneratorVersion)
			assert.Equal(t, tt.wantCreated, meta.Created)
		})
	}
}

// TestGetArtifactType tests the GetArtifactType function
func TestGetArtifactType(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		meta     Meta
		expected string
	}{
		"spec type": {
			meta:     Meta{ArtifactType: "spec"},
			expected: "spec",
		},
		"plan type": {
			meta:     Meta{ArtifactType: "plan"},
			expected: "plan",
		},
		"tasks type": {
			meta:     Meta{ArtifactType: "tasks"},
			expected: "tasks",
		},
		"empty type": {
			meta:     Meta{},
			expected: "",
		},
		"with other fields": {
			meta:     Meta{Version: "1.0.0", Generator: "autospec", ArtifactType: "checklist"},
			expected: "checklist",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			result := GetArtifactType(tt.meta)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestIsValidArtifactType tests the IsValidArtifactType function
func TestIsValidArtifactType(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		artifactType string
		expected     bool
	}{
		"valid spec":         {artifactType: "spec", expected: true},
		"valid plan":         {artifactType: "plan", expected: true},
		"valid tasks":        {artifactType: "tasks", expected: true},
		"valid checklist":    {artifactType: "checklist", expected: true},
		"valid analysis":     {artifactType: "analysis", expected: true},
		"valid constitution": {artifactType: "constitution", expected: true},
		"invalid type":       {artifactType: "invalid", expected: false},
		"empty type":         {artifactType: "", expected: false},
		"uppercase SPEC":     {artifactType: "SPEC", expected: false},
		"mixed case Spec":    {artifactType: "Spec", expected: false},
		"similar but wrong":  {artifactType: "specification", expected: false},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			result := IsValidArtifactType(tt.artifactType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestValidArtifactTypes_Coverage tests that ValidArtifactTypes slice contains expected values
func TestValidArtifactTypes_Coverage(t *testing.T) {
	t.Parallel()
	expectedTypes := []string{
		"spec",
		"plan",
		"tasks",
		"checklist",
		"analysis",
		"constitution",
	}

	assert.Len(t, ValidArtifactTypes, len(expectedTypes), "ValidArtifactTypes length mismatch")

	for _, expected := range expectedTypes {
		found := false
		for _, actual := range ValidArtifactTypes {
			if actual == expected {
				found = true
				break
			}
		}
		assert.True(t, found, "expected artifact type %q not found in ValidArtifactTypes", expected)
	}
}

// TestExtractMeta_EmptyInput tests ExtractMeta with empty reader
func TestExtractMeta_EmptyInput(t *testing.T) {
	t.Parallel()
	meta, err := ExtractMeta(strings.NewReader(""))
	require.NoError(t, err)
	assert.Empty(t, meta.Version)
	assert.Empty(t, meta.ArtifactType)
}
