package dag

import (
	"testing"

	internaldag "github.com/ariel-frischer/autospec/internal/dag"
	"github.com/stretchr/testify/assert"
)

func TestVisualizeCmd_Structure(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "visualize <file>", visualizeCmd.Use)
	assert.NotEmpty(t, visualizeCmd.Short)
	assert.NotEmpty(t, visualizeCmd.Long)
	assert.NotEmpty(t, visualizeCmd.Example)
}

func TestRenderVisualization(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		cfg            *internaldag.DAGConfig
		compact        bool
		wantContains   []string
		wantNotContain []string
	}{
		"full visualization": {
			cfg: &internaldag.DAGConfig{
				SchemaVersion: "1.0",
				DAG:           internaldag.DAGMetadata{Name: "Test"},
				Layers: []internaldag.Layer{
					{
						ID:       "L0",
						Features: []internaldag.Feature{{ID: "feat-a", Description: "A"}},
					},
				},
			},
			compact:      false,
			wantContains: []string{"DAG: Test", "[L0]", "feat-a", "Layers: 1"},
		},
		"compact visualization": {
			cfg: &internaldag.DAGConfig{
				SchemaVersion: "1.0",
				DAG:           internaldag.DAGMetadata{Name: "Test"},
				Layers: []internaldag.Layer{
					{
						ID:       "L0",
						Features: []internaldag.Feature{{ID: "feat-a", Description: "A"}},
					},
				},
			},
			compact:        true,
			wantContains:   []string{"L0: [feat-a]"},
			wantNotContain: []string{"DAG: Test"},
		},
		"multiple layers full": {
			cfg: &internaldag.DAGConfig{
				SchemaVersion: "1.0",
				DAG:           internaldag.DAGMetadata{Name: "Multi"},
				Layers: []internaldag.Layer{
					{
						ID:       "L0",
						Features: []internaldag.Feature{{ID: "feat-a", Description: "A"}},
					},
					{
						ID:       "L1",
						Features: []internaldag.Feature{{ID: "feat-b", Description: "B"}},
					},
				},
			},
			compact:      false,
			wantContains: []string{"[L0]", "[L1]", "feat-a", "feat-b"},
		},
		"multiple layers compact": {
			cfg: &internaldag.DAGConfig{
				SchemaVersion: "1.0",
				DAG:           internaldag.DAGMetadata{Name: "Multi"},
				Layers: []internaldag.Layer{
					{
						ID:       "L0",
						Features: []internaldag.Feature{{ID: "feat-a", Description: "A"}},
					},
					{
						ID:       "L1",
						Features: []internaldag.Feature{{ID: "feat-b", Description: "B"}},
					},
				},
			},
			compact:      true,
			wantContains: []string{"L0: [feat-a]", "->", "L1: [feat-b]"},
		},
		"with dependencies": {
			cfg: &internaldag.DAGConfig{
				SchemaVersion: "1.0",
				DAG:           internaldag.DAGMetadata{Name: "Deps"},
				Layers: []internaldag.Layer{
					{
						ID:       "L0",
						Features: []internaldag.Feature{{ID: "feat-a", Description: "A"}},
					},
					{
						ID: "L1",
						Features: []internaldag.Feature{{
							ID:          "feat-b",
							Description: "B",
							DependsOn:   []string{"feat-a"},
						}},
					},
				},
			},
			compact:      false,
			wantContains: []string{"feat-b *", "feat-b --> feat-a"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result := renderVisualization(tt.cfg, tt.compact)

			for _, s := range tt.wantContains {
				assert.Contains(t, result, s, "should contain %q", s)
			}

			for _, s := range tt.wantNotContain {
				assert.NotContains(t, result, s, "should not contain %q", s)
			}
		})
	}
}

func TestVisualizeCmd_CompactFlagRegistered(t *testing.T) {
	t.Parallel()

	flag := visualizeCmd.Flags().Lookup("compact")
	assert.NotNil(t, flag)
	assert.Empty(t, flag.Shorthand) // No shorthand to avoid conflict with -c (config)
	assert.Equal(t, "false", flag.DefValue)
}

func TestVisualizeCmd_SpecsDirFlagRegistered(t *testing.T) {
	t.Parallel()

	flag := visualizeCmd.Flags().Lookup("specs-dir")
	assert.NotNil(t, flag)
}
