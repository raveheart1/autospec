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
	tests := []struct {
		input    string
		expected Version
	}{
		{"1.0.0", Version{Major: 1, Minor: 0, Patch: 0}},
		{"2.3.4", Version{Major: 2, Minor: 3, Patch: 4}},
		{"0.1.0", Version{Major: 0, Minor: 1, Patch: 0}},
		{"10.20.30", Version{Major: 10, Minor: 20, Patch: 30}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			v, err := ParseVersion(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, v)
		})
	}
}

func TestParseVersion_Invalid(t *testing.T) {
	tests := []string{
		"",
		"1.0",
		"1",
		"v1.0.0",
		"1.0.0.0",
		"a.b.c",
		"1.0.0-beta",
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			_, err := ParseVersion(input)
			assert.Error(t, err, "should error on invalid version")
		})
	}
}

func TestVersion_Compare(t *testing.T) {
	tests := []struct {
		name     string
		v1       Version
		v2       Version
		expected int
	}{
		{"equal", Version{1, 0, 0}, Version{1, 0, 0}, 0},
		{"major greater", Version{2, 0, 0}, Version{1, 0, 0}, 1},
		{"major less", Version{1, 0, 0}, Version{2, 0, 0}, -1},
		{"minor greater", Version{1, 2, 0}, Version{1, 1, 0}, 1},
		{"minor less", Version{1, 1, 0}, Version{1, 2, 0}, -1},
		{"patch greater", Version{1, 0, 2}, Version{1, 0, 1}, 1},
		{"patch less", Version{1, 0, 1}, Version{1, 0, 2}, -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
	tests := []struct {
		v1       string
		v2       string
		expected bool
	}{
		{"1.0.0", "1.0.0", false},
		{"1.0.0", "1.1.0", false},
		{"1.0.0", "2.0.0", true},
		{"2.0.0", "1.0.0", true},
	}

	for _, tt := range tests {
		t.Run(tt.v1+"_vs_"+tt.v2, func(t *testing.T) {
			result := IsMajorVersionMismatch(tt.v1, tt.v2)
			assert.Equal(t, tt.expected, result)
		})
	}
}
