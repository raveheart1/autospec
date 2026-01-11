package dag

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRenderASCII(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		cfg      *DAGConfig
		contains []string
		excludes []string
	}{
		"empty layers": {
			cfg: &DAGConfig{
				SchemaVersion: "1.0",
				DAG:           DAGMetadata{Name: "Empty DAG"},
				Layers:        []Layer{},
			},
			contains: []string{"no layers"},
		},
		"single layer single feature": {
			cfg: &DAGConfig{
				SchemaVersion: "1.0",
				DAG:           DAGMetadata{Name: "Simple DAG"},
				Layers: []Layer{
					{
						ID: "L0",
						Features: []Feature{
							{ID: "feat-a", Description: "Feature A"},
						},
					},
				},
			},
			contains: []string{"Simple DAG", "[L0]", "feat-a", "Layers: 1", "Features: 1"},
			excludes: []string{"feat-a *", "feat-a -->"}, // no dep marker on feature
		},
		"layer with name": {
			cfg: &DAGConfig{
				SchemaVersion: "1.0",
				DAG:           DAGMetadata{Name: "Named Layers"},
				Layers: []Layer{
					{
						ID:   "L0",
						Name: "Foundation",
						Features: []Feature{
							{ID: "feat-a", Description: "Feature A"},
						},
					},
				},
			},
			contains: []string{"[L0 (Foundation)]"},
		},
		"multiple layers with connector": {
			cfg: &DAGConfig{
				SchemaVersion: "1.0",
				DAG:           DAGMetadata{Name: "Multi Layer"},
				Layers: []Layer{
					{
						ID: "L0",
						Features: []Feature{
							{ID: "feat-a", Description: "Feature A"},
						},
					},
					{
						ID: "L1",
						Features: []Feature{
							{ID: "feat-b", Description: "Feature B"},
						},
					},
				},
			},
			contains: []string{"[L0]", "[L1]", "|\n    v"},
		},
		"features with dependencies": {
			cfg: &DAGConfig{
				SchemaVersion: "1.0",
				DAG:           DAGMetadata{Name: "With Deps"},
				Layers: []Layer{
					{
						ID: "L0",
						Features: []Feature{
							{ID: "feat-a", Description: "Feature A"},
						},
					},
					{
						ID: "L1",
						Features: []Feature{
							{ID: "feat-b", Description: "Feature B", DependsOn: []string{"feat-a"}},
						},
					},
				},
			},
			contains: []string{"feat-b *", "feat-b --> feat-a", "Feature Dependencies:", "-->"},
		},
		"multiple dependencies": {
			cfg: &DAGConfig{
				SchemaVersion: "1.0",
				DAG:           DAGMetadata{Name: "Multi Deps"},
				Layers: []Layer{
					{
						ID: "L0",
						Features: []Feature{
							{ID: "feat-a", Description: "Feature A"},
							{ID: "feat-b", Description: "Feature B"},
						},
					},
					{
						ID: "L1",
						Features: []Feature{
							{ID: "feat-c", Description: "Feature C", DependsOn: []string{"feat-a", "feat-b"}},
						},
					},
				},
			},
			contains: []string{"feat-c --> feat-a, feat-b"},
		},
		"features sorted alphabetically": {
			cfg: &DAGConfig{
				SchemaVersion: "1.0",
				DAG:           DAGMetadata{Name: "Sorted"},
				Layers: []Layer{
					{
						ID: "L0",
						Features: []Feature{
							{ID: "feat-c", Description: "Feature C"},
							{ID: "feat-a", Description: "Feature A"},
							{ID: "feat-b", Description: "Feature B"},
						},
					},
				},
			},
			contains: []string{"feat-a", "feat-b", "feat-c"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result := RenderASCII(tt.cfg)

			for _, s := range tt.contains {
				assert.Contains(t, result, s, "output should contain %q", s)
			}

			for _, s := range tt.excludes {
				assert.NotContains(t, result, s, "output should not contain %q", s)
			}
		})
	}
}

func TestRenderASCII_OutputWidth(t *testing.T) {
	t.Parallel()

	cfg := &DAGConfig{
		SchemaVersion: "1.0",
		DAG:           DAGMetadata{Name: "Width Test"},
		Layers: []Layer{
			{
				ID: "L0",
				Features: []Feature{
					{ID: "feature-with-reasonably-long-name", Description: "Desc"},
				},
			},
		},
	}

	result := RenderASCII(cfg)
	lines := strings.Split(result, "\n")

	for _, line := range lines {
		assert.LessOrEqual(t, len(line), 120, "line should be within terminal width: %q", line)
	}
}

func TestRenderASCII_ASCIIOnly(t *testing.T) {
	t.Parallel()

	cfg := &DAGConfig{
		SchemaVersion: "1.0",
		DAG:           DAGMetadata{Name: "ASCII Test"},
		Layers: []Layer{
			{
				ID: "L0",
				Features: []Feature{
					{ID: "feat-a", Description: "Feature A"},
				},
			},
			{
				ID: "L1",
				Features: []Feature{
					{ID: "feat-b", Description: "Feature B", DependsOn: []string{"feat-a"}},
				},
			},
		},
	}

	result := RenderASCII(cfg)

	for _, r := range result {
		assert.True(t, r < 128, "character should be ASCII: %q (code %d)", string(r), r)
	}
}

func TestRenderCompact(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		cfg      *DAGConfig
		expected string
	}{
		"empty dag": {
			cfg: &DAGConfig{
				Layers: []Layer{},
			},
			expected: "Empty DAG",
		},
		"single layer": {
			cfg: &DAGConfig{
				Layers: []Layer{
					{
						ID: "L0",
						Features: []Feature{
							{ID: "feat-a", Description: "A"},
						},
					},
				},
			},
			expected: "L0: [feat-a]",
		},
		"multiple layers": {
			cfg: &DAGConfig{
				Layers: []Layer{
					{
						ID: "L0",
						Features: []Feature{
							{ID: "feat-a", Description: "A"},
							{ID: "feat-b", Description: "B"},
						},
					},
					{
						ID: "L1",
						Features: []Feature{
							{ID: "feat-c", Description: "C"},
						},
					},
				},
			},
			expected: "L0: [feat-a, feat-b] -> L1: [feat-c]",
		},
		"features sorted": {
			cfg: &DAGConfig{
				Layers: []Layer{
					{
						ID: "L0",
						Features: []Feature{
							{ID: "feat-z", Description: "Z"},
							{ID: "feat-a", Description: "A"},
						},
					},
				},
			},
			expected: "L0: [feat-a, feat-z]",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result := RenderCompact(tt.cfg)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRenderLegend(t *testing.T) {
	t.Parallel()

	result := renderLegend()

	assert.Contains(t, result, "Legend:")
	assert.Contains(t, result, "*")
	assert.Contains(t, result, "-->")
}

func TestCountAllFeatures(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		cfg      *DAGConfig
		expected int
	}{
		"empty": {
			cfg:      &DAGConfig{},
			expected: 0,
		},
		"one layer one feature": {
			cfg: &DAGConfig{
				Layers: []Layer{
					{Features: []Feature{{ID: "a"}}},
				},
			},
			expected: 1,
		},
		"multiple layers": {
			cfg: &DAGConfig{
				Layers: []Layer{
					{Features: []Feature{{ID: "a"}, {ID: "b"}}},
					{Features: []Feature{{ID: "c"}}},
				},
			},
			expected: 3,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result := countAllFeatures(tt.cfg)
			assert.Equal(t, tt.expected, result)
		})
	}
}
