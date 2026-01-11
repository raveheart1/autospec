package dag

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// ParseResult contains the parsed DAG config and source location information.
type ParseResult struct {
	Config    *DAGConfig
	NodeInfos map[string]NodeInfo // Maps path (e.g., "layers[0].features[0].id") to location
}

// NodeInfo stores source location information for a YAML node.
type NodeInfo struct {
	Line   int
	Column int
}

// ParseDAGFile parses a DAG configuration from a YAML file.
// Returns the parsed config with source location tracking for error reporting.
func ParseDAGFile(path string) (*ParseResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading DAG file: %w", err)
	}

	return ParseDAGBytes(data)
}

// ParseDAGBytes parses a DAG configuration from YAML bytes.
// Returns the parsed config with source location tracking for error reporting.
func ParseDAGBytes(data []byte) (*ParseResult, error) {
	var rootNode yaml.Node
	if err := yaml.Unmarshal(data, &rootNode); err != nil {
		return nil, fmt.Errorf("parsing YAML: %w", err)
	}

	if rootNode.Kind != yaml.DocumentNode || len(rootNode.Content) == 0 {
		return nil, fmt.Errorf("parsing YAML: empty document")
	}

	result := &ParseResult{
		Config:    &DAGConfig{},
		NodeInfos: make(map[string]NodeInfo),
	}

	if err := parseRootNode(rootNode.Content[0], result); err != nil {
		return nil, err
	}

	return result, nil
}

// parseRootNode parses the root mapping node into a DAGConfig.
func parseRootNode(node *yaml.Node, result *ParseResult) error {
	if node.Kind != yaml.MappingNode {
		return &ParseError{Line: node.Line, Column: node.Column, Message: "expected mapping node at root"}
	}

	for i := 0; i < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		valueNode := node.Content[i+1]

		switch keyNode.Value {
		case "schema_version":
			if err := parseSchemaVersion(valueNode, result); err != nil {
				return err
			}
		case "dag":
			if err := parseDAGMetadata(valueNode, result); err != nil {
				return err
			}
		case "layers":
			if err := parseLayers(valueNode, result); err != nil {
				return err
			}
		}
	}

	return nil
}

// parseSchemaVersion extracts the schema version from a YAML node.
func parseSchemaVersion(node *yaml.Node, result *ParseResult) error {
	result.NodeInfos["schema_version"] = NodeInfo{Line: node.Line, Column: node.Column}
	result.Config.SchemaVersion = node.Value
	return nil
}

// parseDAGMetadata extracts the DAG metadata from a YAML node.
func parseDAGMetadata(node *yaml.Node, result *ParseResult) error {
	result.NodeInfos["dag"] = NodeInfo{Line: node.Line, Column: node.Column}

	if node.Kind != yaml.MappingNode {
		return &ParseError{Line: node.Line, Column: node.Column, Message: "expected mapping for 'dag' field"}
	}

	for i := 0; i < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		valueNode := node.Content[i+1]

		if keyNode.Value == "name" {
			result.NodeInfos["dag.name"] = NodeInfo{Line: valueNode.Line, Column: valueNode.Column}
			result.Config.DAG.Name = valueNode.Value
		}
	}

	return nil
}

// parseLayers extracts the layers list from a YAML node.
func parseLayers(node *yaml.Node, result *ParseResult) error {
	result.NodeInfos["layers"] = NodeInfo{Line: node.Line, Column: node.Column}

	if node.Kind != yaml.SequenceNode {
		return &ParseError{Line: node.Line, Column: node.Column, Message: "expected sequence for 'layers' field"}
	}

	for i, layerNode := range node.Content {
		layer, err := parseLayer(layerNode, i, result)
		if err != nil {
			return err
		}
		result.Config.Layers = append(result.Config.Layers, layer)
	}

	return nil
}

// parseLayer extracts a single layer from a YAML node.
func parseLayer(node *yaml.Node, idx int, result *ParseResult) (Layer, error) {
	prefix := fmt.Sprintf("layers[%d]", idx)
	result.NodeInfos[prefix] = NodeInfo{Line: node.Line, Column: node.Column}

	if node.Kind != yaml.MappingNode {
		return Layer{}, &ParseError{Line: node.Line, Column: node.Column, Message: "expected mapping for layer"}
	}

	var layer Layer
	for i := 0; i < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		valueNode := node.Content[i+1]

		switch keyNode.Value {
		case "id":
			result.NodeInfos[prefix+".id"] = NodeInfo{Line: valueNode.Line, Column: valueNode.Column}
			layer.ID = valueNode.Value
		case "name":
			layer.Name = valueNode.Value
		case "depends_on":
			layer.DependsOn = parseStringList(valueNode)
		case "features":
			features, err := parseFeatures(valueNode, prefix, result)
			if err != nil {
				return Layer{}, err
			}
			layer.Features = features
		}
	}

	return layer, nil
}

// parseFeatures extracts the features list from a YAML node.
func parseFeatures(node *yaml.Node, prefix string, result *ParseResult) ([]Feature, error) {
	result.NodeInfos[prefix+".features"] = NodeInfo{Line: node.Line, Column: node.Column}

	if node.Kind != yaml.SequenceNode {
		return nil, &ParseError{Line: node.Line, Column: node.Column, Message: "expected sequence for 'features' field"}
	}

	var features []Feature
	for i, featureNode := range node.Content {
		feature, err := parseFeature(featureNode, fmt.Sprintf("%s.features[%d]", prefix, i), result)
		if err != nil {
			return nil, err
		}
		features = append(features, feature)
	}

	return features, nil
}

// parseFeature extracts a single feature from a YAML node.
func parseFeature(node *yaml.Node, prefix string, result *ParseResult) (Feature, error) {
	result.NodeInfos[prefix] = NodeInfo{Line: node.Line, Column: node.Column}

	if node.Kind != yaml.MappingNode {
		return Feature{}, &ParseError{Line: node.Line, Column: node.Column, Message: "expected mapping for feature"}
	}

	var feature Feature
	for i := 0; i < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		valueNode := node.Content[i+1]

		switch keyNode.Value {
		case "id":
			result.NodeInfos[prefix+".id"] = NodeInfo{Line: valueNode.Line, Column: valueNode.Column}
			feature.ID = valueNode.Value
		case "description":
			feature.Description = valueNode.Value
		case "depends_on":
			feature.DependsOn = parseStringList(valueNode)
		case "timeout":
			feature.Timeout = valueNode.Value
		}
	}

	return feature, nil
}

// parseStringList extracts a list of strings from a YAML sequence node.
func parseStringList(node *yaml.Node) []string {
	if node.Kind != yaml.SequenceNode {
		return nil
	}

	var items []string
	for _, item := range node.Content {
		items = append(items, item.Value)
	}
	return items
}

// ParseError represents an error during YAML parsing with location information.
type ParseError struct {
	Line    int
	Column  int
	Message string
}

// Error implements the error interface.
func (e *ParseError) Error() string {
	return fmt.Sprintf("line %d, column %d: %s", e.Line, e.Column, e.Message)
}
