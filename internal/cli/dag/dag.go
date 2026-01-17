// Package dag provides CLI commands for DAG schema validation and visualization.
package dag

import (
	"github.com/spf13/cobra"
)

// DagCmd is the parent command for all DAG operations.
var DagCmd = &cobra.Command{
	Use:   "dag",
	Short: "Validate and visualize DAG configuration files",
	Long: `Validate and visualize DAG (Directed Acyclic Graph) configuration files.

The dag command group provides tools for working with multi-spec workflow
DAG configuration files stored in .autospec/dags/*.yaml.

Available subcommands:
  validate   - Validate a DAG file for structural correctness
  visualize  - Generate ASCII visualization of DAG structure`,
}

// Subcommands will be added in their respective files (validate.go, visualize.go)
// during Phase 4 of implementation.
