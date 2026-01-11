package dag

import (
	"fmt"
	"sort"
	"strings"
)

// RenderASCII generates an ASCII representation of the DAG structure.
// The output shows layers with their features and dependency arrows.
// Uses portable ASCII characters only (no Unicode).
func RenderASCII(cfg *DAGConfig) string {
	if len(cfg.Layers) == 0 {
		return "DAG has no layers to visualize."
	}

	var sb strings.Builder
	sb.WriteString(renderHeader(cfg))
	sb.WriteString("\n")

	for i, layer := range cfg.Layers {
		sb.WriteString(renderLayer(layer))

		if i < len(cfg.Layers)-1 {
			sb.WriteString(renderLayerConnector())
		}
	}

	sb.WriteString("\n")
	sb.WriteString(renderDependencies(cfg))
	sb.WriteString("\n")
	sb.WriteString(renderLegend())

	return sb.String()
}

// renderHeader renders the DAG title and summary.
func renderHeader(cfg *DAGConfig) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "DAG: %s\n", cfg.DAG.Name)
	sb.WriteString(strings.Repeat("=", len(cfg.DAG.Name)+5) + "\n")

	layerCount := len(cfg.Layers)
	featureCount := countAllFeatures(cfg)
	fmt.Fprintf(&sb, "Layers: %d  |  Features: %d\n", layerCount, featureCount)

	return sb.String()
}

// countAllFeatures counts total features across all layers.
func countAllFeatures(cfg *DAGConfig) int {
	count := 0
	for _, layer := range cfg.Layers {
		count += len(layer.Features)
	}
	return count
}

// renderLayer renders a single layer with its features.
func renderLayer(layer Layer) string {
	var sb strings.Builder

	layerName := layer.ID
	if layer.Name != "" {
		layerName = fmt.Sprintf("%s (%s)", layer.ID, layer.Name)
	}

	fmt.Fprintf(&sb, "[%s]\n", layerName)
	sb.WriteString(renderLayerFeatures(layer.Features))

	return sb.String()
}

// renderLayerFeatures renders the features in a layer.
func renderLayerFeatures(features []Feature) string {
	if len(features) == 0 {
		return "  (no features)\n"
	}

	// Sort features by ID for consistent output
	sorted := make([]Feature, len(features))
	copy(sorted, features)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].ID < sorted[j].ID
	})

	var sb strings.Builder
	for i, f := range sorted {
		prefix := "  |-"
		if i == len(sorted)-1 {
			prefix = "  +-"
		}
		sb.WriteString(renderFeatureLine(prefix, f))
	}

	return sb.String()
}

// renderFeatureLine renders a single feature line.
func renderFeatureLine(prefix string, f Feature) string {
	depMarker := ""
	if len(f.DependsOn) > 0 {
		depMarker = " *"
	}
	return fmt.Sprintf("%s %s%s\n", prefix, f.ID, depMarker)
}

// renderLayerConnector renders the connector between layers.
func renderLayerConnector() string {
	return "    |\n    v\n"
}

// renderDependencies renders the feature dependency section.
func renderDependencies(cfg *DAGConfig) string {
	deps := collectDependencyLines(cfg)
	if len(deps) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("Feature Dependencies:\n")
	sb.WriteString("---------------------\n")

	for _, line := range deps {
		sb.WriteString(line)
	}

	return sb.String()
}

// collectDependencyLines collects formatted dependency lines.
func collectDependencyLines(cfg *DAGConfig) []string {
	var lines []string

	for _, layer := range cfg.Layers {
		for _, f := range layer.Features {
			if len(f.DependsOn) > 0 {
				deps := make([]string, len(f.DependsOn))
				copy(deps, f.DependsOn)
				sort.Strings(deps)
				lines = append(lines, fmt.Sprintf("  %s --> %s\n", f.ID, strings.Join(deps, ", ")))
			}
		}
	}

	sort.Strings(lines)
	return lines
}

// renderLegend renders the legend explaining symbols.
func renderLegend() string {
	var sb strings.Builder
	sb.WriteString("Legend:\n")
	sb.WriteString("  * = has dependencies (see list above)\n")
	sb.WriteString("  --> = depends on\n")
	return sb.String()
}

// RenderCompact generates a compact single-line representation.
// Format: L0: [feat-a, feat-b] -> L1: [feat-c] -> L2: [feat-d, feat-e]
func RenderCompact(cfg *DAGConfig) string {
	if len(cfg.Layers) == 0 {
		return "Empty DAG"
	}

	parts := make([]string, len(cfg.Layers))
	for i, layer := range cfg.Layers {
		featureIDs := extractFeatureIDs(layer.Features)
		sort.Strings(featureIDs)
		parts[i] = fmt.Sprintf("%s: [%s]", layer.ID, strings.Join(featureIDs, ", "))
	}

	return strings.Join(parts, " -> ")
}

// extractFeatureIDs extracts feature IDs from a list of features.
func extractFeatureIDs(features []Feature) []string {
	ids := make([]string, len(features))
	for i, f := range features {
		ids[i] = f.ID
	}
	return ids
}
