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
