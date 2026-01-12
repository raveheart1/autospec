package dag

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

func TestSaveDAGWithState(t *testing.T) {
	t.Parallel()
	now := time.Now().Truncate(time.Second)
	exitCode := 0

	tests := map[string]struct {
		config      *DAGConfig
		wantErr     bool
		wantMarker  bool
		wantContain []string
	}{
		"config without state": {
			config: &DAGConfig{
				SchemaVersion: "1.0",
				DAG:           DAGMetadata{Name: "Test DAG", ID: "test-dag"},
				Layers: []Layer{{
					ID: "L0",
					Features: []Feature{{
						ID:          "spec-a",
						Description: "Test spec",
					}},
				}},
			},
			wantErr:     false,
			wantMarker:  false,
			wantContain: []string{"schema_version:", "dag:", "layers:", "spec-a"},
		},
		"config with state sections": {
			config: &DAGConfig{
				SchemaVersion: "1.0",
				DAG:           DAGMetadata{Name: "Test DAG", ID: "test-dag"},
				Layers: []Layer{{
					ID: "L0",
					Features: []Feature{{
						ID:          "spec-a",
						Description: "Test spec",
					}},
				}},
				Run: &InlineRunState{
					Status:    InlineRunStatusRunning,
					StartedAt: &now,
				},
				Specs: map[string]*InlineSpecState{
					"spec-a": {
						Status:   InlineSpecStatusCompleted,
						Worktree: "/tmp/worktree",
						ExitCode: &exitCode,
					},
				},
				Staging: map[string]*InlineLayerStaging{
					"L0": {
						Branch:      "dag/test-dag/stage-L0",
						SpecsMerged: []string{"spec-a"},
					},
				},
			},
			wantErr:     false,
			wantMarker:  true,
			wantContain: []string{"schema_version:", "run:", "specs:", "staging:", StateCommentSeparator},
		},
		"nil config returns error": {
			config:  nil,
			wantErr: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			tmpDir := t.TempDir()
			path := filepath.Join(tmpDir, "dag.yaml")

			err := SaveDAGWithState(path, tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("SaveDAGWithState() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			// Read and verify content
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("Failed to read output file: %v", err)
			}

			content := string(data)

			// Check marker presence
			hasMarker := strings.Contains(content, StateCommentSeparator)
			if hasMarker != tt.wantMarker {
				t.Errorf("Marker presence: got %v, want %v", hasMarker, tt.wantMarker)
			}

			// Check required strings
			for _, s := range tt.wantContain {
				if !strings.Contains(content, s) {
					t.Errorf("Output missing %q:\n%s", s, content)
				}
			}
		})
	}
}

func TestSaveDAGWithStatePreservesDefinition(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "dag.yaml")
	now := time.Now().Truncate(time.Second)

	config := &DAGConfig{
		SchemaVersion: "1.0",
		DAG:           DAGMetadata{Name: "My Complex DAG", ID: "complex-dag"},
		Layers: []Layer{
			{
				ID:   "L0",
				Name: "Foundation Layer",
				Features: []Feature{
					{ID: "feat-a", Description: "First feature", Timeout: "30m"},
					{ID: "feat-b", Description: "Second feature", DependsOn: []string{"feat-a"}},
				},
			},
			{
				ID:        "L1",
				Name:      "Core Layer",
				DependsOn: []string{"L0"},
				Features: []Feature{
					{ID: "feat-c", Description: "Third feature"},
				},
			},
		},
		Run: &InlineRunState{Status: InlineRunStatusRunning, StartedAt: &now},
	}

	if err := SaveDAGWithState(path, config); err != nil {
		t.Fatalf("SaveDAGWithState() error = %v", err)
	}

	// Read back and verify definition is intact
	loadedConfig, err := LoadDAGConfigFull(path)
	if err != nil {
		t.Fatalf("LoadDAGConfigFull() error = %v", err)
	}

	// Verify definition fields
	if loadedConfig.SchemaVersion != config.SchemaVersion {
		t.Errorf("SchemaVersion: got %q, want %q", loadedConfig.SchemaVersion, config.SchemaVersion)
	}
	if loadedConfig.DAG.Name != config.DAG.Name {
		t.Errorf("DAG.Name: got %q, want %q", loadedConfig.DAG.Name, config.DAG.Name)
	}
	if loadedConfig.DAG.ID != config.DAG.ID {
		t.Errorf("DAG.ID: got %q, want %q", loadedConfig.DAG.ID, config.DAG.ID)
	}
	if len(loadedConfig.Layers) != len(config.Layers) {
		t.Errorf("Layers count: got %d, want %d", len(loadedConfig.Layers), len(config.Layers))
	}

	// Verify state fields
	if loadedConfig.Run == nil {
		t.Error("Run should not be nil")
	} else if loadedConfig.Run.Status != InlineRunStatusRunning {
		t.Errorf("Run.Status: got %v, want %v", loadedConfig.Run.Status, InlineRunStatusRunning)
	}
}

func TestSaveDAGWithStateAtomicWrite(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "dag.yaml")

	// Write initial content
	initialConfig := &DAGConfig{
		SchemaVersion: "1.0",
		DAG:           DAGMetadata{Name: "Initial"},
		Layers:        []Layer{{ID: "L0", Features: []Feature{{ID: "f1", Description: "test"}}}},
	}
	if err := SaveDAGWithState(path, initialConfig); err != nil {
		t.Fatalf("Initial write failed: %v", err)
	}

	// Verify no temp file exists after successful write
	tmpPath := path + ".tmp"
	if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
		t.Error("Temp file should not exist after successful write")
	}

	// Verify file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("Target file should exist after write")
	}
}

func TestSaveDAGWithStateErrorOnPermissionDenied(t *testing.T) {
	t.Parallel()

	// Skip on systems where we can't test permissions reliably
	if os.Getuid() == 0 {
		t.Skip("Skipping permission test when running as root")
	}

	tmpDir := t.TempDir()

	// Make directory read-only
	readOnlyDir := filepath.Join(tmpDir, "readonly")
	if err := os.MkdirAll(readOnlyDir, 0o755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}
	if err := os.Chmod(readOnlyDir, 0o444); err != nil {
		t.Fatalf("Failed to set permissions: %v", err)
	}
	defer os.Chmod(readOnlyDir, 0o755)

	path := filepath.Join(readOnlyDir, "dag.yaml")
	config := &DAGConfig{
		SchemaVersion: "1.0",
		DAG:           DAGMetadata{Name: "Test"},
		Layers:        []Layer{{ID: "L0", Features: []Feature{{ID: "f1", Description: "test"}}}},
	}

	err := SaveDAGWithState(path, config)
	if err == nil {
		t.Error("Expected error when writing to read-only directory")
	}
}

func TestMarshalDAGWithStateCommentSeparator(t *testing.T) {
	t.Parallel()
	now := time.Now().Truncate(time.Second)

	config := &DAGConfig{
		SchemaVersion: "1.0",
		DAG:           DAGMetadata{Name: "Test"},
		Layers:        []Layer{{ID: "L0", Features: []Feature{{ID: "f1", Description: "test"}}}},
		Run:           &InlineRunState{Status: InlineRunStatusRunning, StartedAt: &now},
	}

	data, err := marshalDAGWithState(config)
	if err != nil {
		t.Fatalf("marshalDAGWithState() error = %v", err)
	}

	content := string(data)

	// Verify marker is present
	if !strings.Contains(content, StateCommentSeparator) {
		t.Errorf("Output missing state marker:\n%s", content)
	}

	// Verify definition comes before state (marker should be between them)
	schemaIdx := strings.Index(content, "schema_version:")
	markerIdx := strings.Index(content, StateCommentSeparator)
	runIdx := strings.Index(content, "run:")

	if schemaIdx >= markerIdx {
		t.Error("Definition should come before marker")
	}
	if markerIdx >= runIdx {
		t.Error("Marker should come before state sections")
	}
}

func TestClearDAGState(t *testing.T) {
	t.Parallel()
	now := time.Now().Truncate(time.Second)
	exitCode := 0

	config := &DAGConfig{
		SchemaVersion: "1.0",
		DAG:           DAGMetadata{Name: "Test"},
		Layers:        []Layer{{ID: "L0", Features: []Feature{{ID: "f1", Description: "test"}}}},
		Run:           &InlineRunState{Status: InlineRunStatusRunning, StartedAt: &now},
		Specs: map[string]*InlineSpecState{
			"f1": {Status: InlineSpecStatusCompleted, ExitCode: &exitCode},
		},
		Staging: map[string]*InlineLayerStaging{
			"L0": {Branch: "test-branch"},
		},
	}

	ClearDAGState(config)

	if config.Run != nil {
		t.Error("Run should be nil after ClearDAGState")
	}
	if config.Specs != nil {
		t.Error("Specs should be nil after ClearDAGState")
	}
	if config.Staging != nil {
		t.Error("Staging should be nil after ClearDAGState")
	}

	// Definition should be preserved
	if config.SchemaVersion != "1.0" {
		t.Error("Definition should be preserved")
	}
}

func TestLoadDAGConfigFull(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	tests := map[string]struct {
		content       string
		wantErr       bool
		wantRunStatus InlineRunStatus
		wantSpecCount int
	}{
		"full config with state": {
			content: `schema_version: "1.0"
dag:
  name: Test DAG
layers:
  - id: L0
    features:
      - id: spec-a
        description: Test
run:
  status: running
specs:
  spec-a:
    status: completed
`,
			wantErr:       false,
			wantRunStatus: InlineRunStatusRunning,
			wantSpecCount: 1,
		},
		"config without state": {
			content: `schema_version: "1.0"
dag:
  name: Test DAG
layers:
  - id: L0
    features:
      - id: spec-a
        description: Test
`,
			wantErr:       false,
			wantRunStatus: "",
			wantSpecCount: 0,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(tmpDir, name+".yaml")
			if err := os.WriteFile(path, []byte(tt.content), 0o644); err != nil {
				t.Fatalf("Failed to write test file: %v", err)
			}

			config, err := LoadDAGConfigFull(path)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadDAGConfigFull() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			if tt.wantRunStatus == "" {
				if config.Run != nil {
					t.Error("Run should be nil")
				}
			} else {
				if config.Run == nil || config.Run.Status != tt.wantRunStatus {
					t.Errorf("Run.Status: got %v, want %v", config.Run, tt.wantRunStatus)
				}
			}

			if len(config.Specs) != tt.wantSpecCount {
				t.Errorf("Specs count: got %d, want %d", len(config.Specs), tt.wantSpecCount)
			}
		})
	}
}

func TestLoadDAGConfigFullNotFound(t *testing.T) {
	t.Parallel()
	_, err := LoadDAGConfigFull("/nonexistent/path/dag.yaml")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestHasStateData(t *testing.T) {
	t.Parallel()
	now := time.Now()

	tests := map[string]struct {
		config *DAGConfig
		want   bool
	}{
		"no state": {
			config: &DAGConfig{},
			want:   false,
		},
		"run only": {
			config: &DAGConfig{Run: &InlineRunState{Status: InlineRunStatusPending}},
			want:   true,
		},
		"specs only": {
			config: &DAGConfig{Specs: map[string]*InlineSpecState{"a": {Status: InlineSpecStatusPending}}},
			want:   true,
		},
		"staging only": {
			config: &DAGConfig{Staging: map[string]*InlineLayerStaging{"L0": {Branch: "test"}}},
			want:   true,
		},
		"all state": {
			config: &DAGConfig{
				Run:     &InlineRunState{Status: InlineRunStatusRunning, StartedAt: &now},
				Specs:   map[string]*InlineSpecState{"a": {}},
				Staging: map[string]*InlineLayerStaging{"L0": {}},
			},
			want: true,
		},
		"empty maps": {
			config: &DAGConfig{
				Specs:   map[string]*InlineSpecState{},
				Staging: map[string]*InlineLayerStaging{},
			},
			want: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := hasStateData(tt.config)
			if got != tt.want {
				t.Errorf("hasStateData() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSaveDAGWithStateYAMLValidity(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "dag.yaml")
	now := time.Now().Truncate(time.Second)
	exitCode := 0

	config := &DAGConfig{
		SchemaVersion: "1.0",
		DAG:           DAGMetadata{Name: "Test DAG", ID: "test-dag"},
		Layers: []Layer{{
			ID:   "L0",
			Name: "Layer Zero",
			Features: []Feature{
				{ID: "spec-a", Description: "Test spec A"},
				{ID: "spec-b", Description: "Test spec B", DependsOn: []string{"spec-a"}},
			},
		}},
		Run: &InlineRunState{
			Status:    InlineRunStatusRunning,
			StartedAt: &now,
		},
		Specs: map[string]*InlineSpecState{
			"spec-a": {Status: InlineSpecStatusCompleted, ExitCode: &exitCode},
			"spec-b": {Status: InlineSpecStatusRunning},
		},
	}

	if err := SaveDAGWithState(path, config); err != nil {
		t.Fatalf("SaveDAGWithState() error = %v", err)
	}

	// Verify the output is valid YAML that can be parsed
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	var parsed DAGConfig
	if err := yaml.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Output is not valid YAML: %v\nContent:\n%s", err, string(data))
	}

	// Verify key fields survived the round-trip
	if parsed.SchemaVersion != "1.0" {
		t.Errorf("SchemaVersion: got %q, want %q", parsed.SchemaVersion, "1.0")
	}
	if parsed.Run == nil || parsed.Run.Status != InlineRunStatusRunning {
		t.Error("Run state not preserved")
	}
	if len(parsed.Specs) != 2 {
		t.Errorf("Specs count: got %d, want 2", len(parsed.Specs))
	}
}
