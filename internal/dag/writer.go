package dag

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// StateCommentSeparator is the visual separator between definition and state sections.
const StateCommentSeparator = "# ====== RUNTIME STATE (auto-managed, do not edit) ======"

// SaveDAGWithState writes a DAGConfig to a YAML file, preserving definition sections
// and appending state sections with a visual separator.
// Uses atomic write (temp file + rename) to prevent corruption on crash.
func SaveDAGWithState(path string, config *DAGConfig) error {
	if config == nil {
		return fmt.Errorf("config is nil")
	}

	data, err := marshalDAGWithState(config)
	if err != nil {
		return fmt.Errorf("marshaling DAG config: %w", err)
	}

	if err := atomicWriteToFile(path, data); err != nil {
		return fmt.Errorf("writing DAG file: %w", err)
	}

	return nil
}

// marshalDAGWithState creates YAML bytes with definition sections followed by
// state sections (if present) separated by a comment marker.
func marshalDAGWithState(config *DAGConfig) ([]byte, error) {
	defBytes, err := marshalDefinitionSections(config)
	if err != nil {
		return nil, fmt.Errorf("marshaling definition: %w", err)
	}

	if !hasStateData(config) {
		return defBytes, nil
	}

	stateBytes, err := marshalStateSections(config)
	if err != nil {
		return nil, fmt.Errorf("marshaling state: %w", err)
	}

	return combineWithMarker(defBytes, stateBytes), nil
}

// marshalDefinitionSections creates YAML for only the definition parts of DAGConfig.
func marshalDefinitionSections(config *DAGConfig) ([]byte, error) {
	// Create a definition-only struct to avoid including state fields
	defOnly := struct {
		SchemaVersion string      `yaml:"schema_version"`
		DAG           DAGMetadata `yaml:"dag"`
		Layers        []Layer     `yaml:"layers"`
	}{
		SchemaVersion: config.SchemaVersion,
		DAG:           config.DAG,
		Layers:        config.Layers,
	}

	return yaml.Marshal(defOnly)
}

// marshalStateSections creates YAML for only the state parts of DAGConfig.
func marshalStateSections(config *DAGConfig) ([]byte, error) {
	stateOnly := struct {
		Run     *InlineRunState                `yaml:"run,omitempty"`
		Specs   map[string]*InlineSpecState    `yaml:"specs,omitempty"`
		Staging map[string]*InlineLayerStaging `yaml:"staging,omitempty"`
	}{
		Run:     config.Run,
		Specs:   config.Specs,
		Staging: config.Staging,
	}

	return yaml.Marshal(stateOnly)
}

// hasStateData returns true if the config has any runtime state data.
func hasStateData(config *DAGConfig) bool {
	return config.Run != nil || len(config.Specs) > 0 || len(config.Staging) > 0
}

// HasInlineState returns true if the DAGConfig has any inline runtime state.
// This is the exported version of hasStateData for use by other packages.
func HasInlineState(config *DAGConfig) bool {
	return hasStateData(config)
}

// combineWithMarker joins definition and state bytes with the marker comment.
func combineWithMarker(defBytes, stateBytes []byte) []byte {
	var buf bytes.Buffer
	buf.Write(defBytes)
	buf.WriteString("\n")
	buf.WriteString(StateCommentSeparator)
	buf.WriteString("\n")
	buf.Write(stateBytes)
	return buf.Bytes()
}

// atomicWriteToFile writes data to path using temp file + rename pattern.
// Ensures no partial writes occur on crash.
func atomicWriteToFile(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return fmt.Errorf("writing temp file: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath) // Best effort cleanup
		return fmt.Errorf("renaming temp file: %w", err)
	}

	return nil
}

// ClearDAGState removes state sections from a DAGConfig in-place.
// After calling this, SaveDAGWithState will write only definition sections.
func ClearDAGState(config *DAGConfig) {
	config.Run = nil
	config.Specs = nil
	config.Staging = nil
}

// LoadDAGConfigFull loads a DAGConfig from path including any state sections.
// Unlike ParseDAGFile which focuses on definition parsing, this function
// uses standard YAML unmarshal to get all fields including state.
func LoadDAGConfigFull(path string) (*DAGConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading DAG file: %w", err)
	}

	var config DAGConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parsing DAG file: %w", err)
	}

	return &config, nil
}
