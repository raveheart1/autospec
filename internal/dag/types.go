package dag

// DAGConfig represents the root configuration structure for a DAG file.
// It contains schema version information, DAG metadata, and the ordered list of layers.
type DAGConfig struct {
	// SchemaVersion is the version of the DAG schema format (e.g., "1.0").
	SchemaVersion string `yaml:"schema_version"`
	// DAG contains metadata about the DAG.
	DAG DAGMetadata `yaml:"dag"`
	// Layers is an ordered list of execution layers.
	Layers []Layer `yaml:"layers"`
}

// DAGMetadata contains metadata about the DAG.
type DAGMetadata struct {
	// Name is the human-readable name for the DAG.
	Name string `yaml:"name"`
}

// Layer represents a grouping of features that can be processed together.
// Layers define execution ordering through their dependencies.
type Layer struct {
	// ID is the unique identifier for this layer (e.g., "L0", "L1").
	ID string `yaml:"id"`
	// Name is an optional human-readable name for the layer.
	Name string `yaml:"name,omitempty"`
	// DependsOn lists layer IDs that must complete before this layer can start.
	DependsOn []string `yaml:"depends_on,omitempty"`
	// Features is the list of features in this layer.
	Features []Feature `yaml:"features"`
}

// Feature represents a reference to a spec folder for a single feature.
// Features define fine-grained dependencies within and across layers.
type Feature struct {
	// ID is the spec folder name (must exist in specs/<id>/).
	ID string `yaml:"id"`
	// Description is a human-readable description used by autospec run to create
	// the spec if the folder doesn't exist.
	Description string `yaml:"description"`
	// DependsOn lists feature IDs that must complete before this feature can start.
	DependsOn []string `yaml:"depends_on,omitempty"`
	// Timeout overrides the default timeout for this feature (e.g., "30m", "1h").
	Timeout string `yaml:"timeout,omitempty"`
}
