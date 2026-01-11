package dag

import (
	"fmt"
	"os"
	"path/filepath"
)

// ValidateDAG validates a DAGConfig for structural correctness.
// It checks required fields, validates dependencies, detects cycles,
// and verifies spec folders exist.
// Returns a slice of errors, empty if valid.
func ValidateDAG(cfg *DAGConfig, result *ParseResult, specsDir string) []error {
	var errs []error

	errs = append(errs, validateRequiredFields(cfg, result)...)
	errs = append(errs, validateLayerDependencies(cfg, result)...)
	errs = append(errs, validateFeatureUniqueness(cfg, result)...)
	errs = append(errs, validateFeatureDependencies(cfg, result)...)
	errs = append(errs, detectFeatureCycles(cfg)...)
	errs = append(errs, validateSpecFolders(cfg, result, specsDir)...)

	return errs
}

// validateRequiredFields checks that all required fields are present.
func validateRequiredFields(cfg *DAGConfig, result *ParseResult) []error {
	var errs []error

	if cfg.SchemaVersion == "" {
		info := result.NodeInfos["schema_version"]
		errs = append(errs, &MissingFieldError{
			Field: "schema_version", Context: "root", Line: info.Line, Column: info.Column,
		})
	}

	if cfg.DAG.Name == "" {
		info := result.NodeInfos["dag.name"]
		if info.Line == 0 {
			info = result.NodeInfos["dag"]
		}
		errs = append(errs, &MissingFieldError{
			Field: "name", Context: "dag", Line: info.Line, Column: info.Column,
		})
	}

	if len(cfg.Layers) == 0 {
		info := result.NodeInfos["layers"]
		errs = append(errs, &MissingFieldError{
			Field: "layers", Context: "root (at least one layer required)", Line: info.Line, Column: info.Column,
		})
	}

	errs = append(errs, validateLayerFields(cfg, result)...)

	return errs
}

// validateLayerFields checks required fields within each layer.
func validateLayerFields(cfg *DAGConfig, result *ParseResult) []error {
	var errs []error

	for i, layer := range cfg.Layers {
		prefix := fmt.Sprintf("layers[%d]", i)

		if layer.ID == "" {
			info := result.NodeInfos[prefix]
			errs = append(errs, &MissingFieldError{
				Field: "id", Context: fmt.Sprintf("layer at index %d", i), Line: info.Line, Column: info.Column,
			})
		}

		if len(layer.Features) == 0 {
			info := result.NodeInfos[prefix]
			errs = append(errs, &MissingFieldError{
				Field: "features", Context: fmt.Sprintf("layer %q", layer.ID), Line: info.Line, Column: info.Column,
			})
		}

		errs = append(errs, validateFeatureFields(layer, i, result)...)
	}

	return errs
}

// validateFeatureFields checks required fields within each feature.
func validateFeatureFields(layer Layer, layerIdx int, result *ParseResult) []error {
	var errs []error

	for j, feature := range layer.Features {
		prefix := fmt.Sprintf("layers[%d].features[%d]", layerIdx, j)

		if feature.ID == "" {
			info := result.NodeInfos[prefix]
			errs = append(errs, &MissingFieldError{
				Field: "id", Context: fmt.Sprintf("feature at layer %q index %d", layer.ID, j),
				Line: info.Line, Column: info.Column,
			})
		}

		if feature.Description == "" {
			info := result.NodeInfos[prefix]
			errs = append(errs, &MissingFieldError{
				Field: "description", Context: fmt.Sprintf("feature %q", feature.ID),
				Line: info.Line, Column: info.Column,
			})
		}
	}

	return errs
}

// validateLayerDependencies checks that layer depends_on references valid layers.
func validateLayerDependencies(cfg *DAGConfig, result *ParseResult) []error {
	var errs []error

	layerIDs := collectLayerIDs(cfg)

	for i, layer := range cfg.Layers {
		for _, depID := range layer.DependsOn {
			if !layerIDs[depID] {
				info := result.NodeInfos[fmt.Sprintf("layers[%d]", i)]
				errs = append(errs, &InvalidLayerRefError{
					LayerID: layer.ID, InvalidRef: depID,
					ValidLayerIDs: mapKeys(layerIDs), Line: info.Line, Column: info.Column,
				})
			}
		}
	}

	return errs
}

// collectLayerIDs returns a set of all layer IDs.
func collectLayerIDs(cfg *DAGConfig) map[string]bool {
	ids := make(map[string]bool)
	for _, layer := range cfg.Layers {
		ids[layer.ID] = true
	}
	return ids
}

// validateFeatureUniqueness checks that feature IDs are unique across all layers.
func validateFeatureUniqueness(cfg *DAGConfig, result *ParseResult) []error {
	var errs []error
	seen := make(map[string]string) // featureID -> layerID

	for i, layer := range cfg.Layers {
		for j, feature := range layer.Features {
			if prevLayer, exists := seen[feature.ID]; exists {
				info := result.NodeInfos[fmt.Sprintf("layers[%d].features[%d]", i, j)]
				errs = append(errs, &DuplicateFeatureError{
					FeatureID: feature.ID, FirstLocation: fmt.Sprintf("layer %q", prevLayer),
					SecondLocation: fmt.Sprintf("layer %q", layer.ID), Line: info.Line, Column: info.Column,
				})
			}
			seen[feature.ID] = layer.ID
		}
	}

	return errs
}

// validateFeatureDependencies checks that feature depends_on references valid features.
func validateFeatureDependencies(cfg *DAGConfig, result *ParseResult) []error {
	var errs []error

	featureIDs := collectFeatureIDs(cfg)

	for i, layer := range cfg.Layers {
		for j, feature := range layer.Features {
			for _, depID := range feature.DependsOn {
				if !featureIDs[depID] {
					info := result.NodeInfos[fmt.Sprintf("layers[%d].features[%d]", i, j)]
					errs = append(errs, &InvalidFeatureRefError{
						FeatureID: feature.ID, InvalidRef: depID, Line: info.Line, Column: info.Column,
					})
				}
			}
		}
	}

	return errs
}

// collectFeatureIDs returns a set of all feature IDs across all layers.
func collectFeatureIDs(cfg *DAGConfig) map[string]bool {
	ids := make(map[string]bool)
	for _, layer := range cfg.Layers {
		for _, feature := range layer.Features {
			ids[feature.ID] = true
		}
	}
	return ids
}

// detectFeatureCycles detects cycles in feature dependencies using DFS.
func detectFeatureCycles(cfg *DAGConfig) []error {
	var errs []error

	deps := buildFeatureDependencyMap(cfg)
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	for featureID := range deps {
		if !visited[featureID] {
			if cycle := detectCycleDFS(featureID, deps, visited, recStack, nil); cycle != nil {
				errs = append(errs, &CycleError{Path: cycle})
			}
		}
	}

	return errs
}

// buildFeatureDependencyMap builds a map of feature ID to its dependencies.
func buildFeatureDependencyMap(cfg *DAGConfig) map[string][]string {
	deps := make(map[string][]string)
	for _, layer := range cfg.Layers {
		for _, feature := range layer.Features {
			deps[feature.ID] = feature.DependsOn
		}
	}
	return deps
}

// detectCycleDFS performs depth-first search for cycle detection.
func detectCycleDFS(id string, deps map[string][]string, visited, recStack map[string]bool, path []string) []string {
	visited[id] = true
	recStack[id] = true
	path = append(path, id)

	for _, depID := range deps[id] {
		if !visited[depID] {
			if cycle := detectCycleDFS(depID, deps, visited, recStack, path); cycle != nil {
				return cycle
			}
		} else if recStack[depID] {
			return buildCyclePath(path, depID)
		}
	}

	recStack[id] = false
	return nil
}

// buildCyclePath constructs the cycle path from the DFS path.
func buildCyclePath(path []string, cycleStart string) []string {
	startIdx := -1
	for i, id := range path {
		if id == cycleStart {
			startIdx = i
			break
		}
	}
	if startIdx >= 0 {
		return append(path[startIdx:], cycleStart)
	}
	return append(path, cycleStart)
}

// validateSpecFolders checks that spec folders exist for all features.
func validateSpecFolders(cfg *DAGConfig, result *ParseResult, specsDir string) []error {
	var errs []error

	for i, layer := range cfg.Layers {
		for j, feature := range layer.Features {
			specPath := filepath.Join(specsDir, feature.ID)
			if !dirExists(specPath) {
				info := result.NodeInfos[fmt.Sprintf("layers[%d].features[%d]", i, j)]
				errs = append(errs, &MissingSpecError{
					FeatureID: feature.ID, ExpectedPath: specPath, Line: info.Line, Column: info.Column,
				})
			}
		}
	}

	return errs
}

// dirExists checks if a directory exists.
func dirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// mapKeys returns the keys of a map as a slice.
func mapKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
