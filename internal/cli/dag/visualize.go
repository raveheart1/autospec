package dag

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ariel-frischer/autospec/internal/dag"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var visualizeCmd = &cobra.Command{
	Use:   "visualize <file>",
	Short: "Generate ASCII visualization of DAG structure",
	Long: `Generate an ASCII visualization of a DAG configuration file.

The visualization shows:
- Layers with their features
- Dependency relationships between features
- Summary statistics (layer count, feature count)

The DAG is validated before visualization. If validation fails,
errors are displayed instead of the diagram.

Exit codes:
  0 - Visualization successful
  1 - Invalid DAG file or visualization errors`,
	Example: `  # Visualize a DAG file
  autospec dag visualize .autospec/dags/my-workflow.yaml

  # Visualize with compact output
  autospec dag visualize --compact .autospec/dags/my-workflow.yaml`,
	Args: cobra.ExactArgs(1),
	RunE: runVisualize,
}

var compactFlag bool

func runVisualize(cmd *cobra.Command, args []string) error {
	filePath := args[0]

	if err := validateFileArg(filePath); err != nil {
		return err
	}

	result, err := dag.ParseDAGFile(filePath)
	if err != nil {
		return formatVisualizationParseError(filePath, err)
	}

	specsDir, _ := cmd.Flags().GetString("specs-dir")
	if specsDir == "" {
		specsDir = "specs"
	}

	vr := dag.ValidateDAG(result.Config, result, specsDir)
	if vr.HasErrors() {
		return formatVisualizationErrors(filePath, vr.Errors)
	}

	output := renderVisualization(result.Config, compactFlag)
	fmt.Print(output)

	return nil
}

// renderVisualization generates the visualization output.
func renderVisualization(cfg *dag.DAGConfig, compact bool) string {
	if compact {
		return dag.RenderCompact(cfg) + "\n"
	}
	return dag.RenderASCII(cfg)
}

// formatVisualizationParseError formats a parse error for visualization.
func formatVisualizationParseError(filePath string, err error) error {
	red := color.New(color.FgRed, color.Bold)
	red.Fprintf(os.Stderr, "Error: ")
	fmt.Fprintf(os.Stderr, "Cannot visualize %s\n", filepath.Base(filePath))
	fmt.Fprintf(os.Stderr, "  Parse error: %v\n", err)
	os.Exit(1)
	return nil
}

// formatVisualizationErrors formats validation errors for visualization.
func formatVisualizationErrors(filePath string, errs []error) error {
	red := color.New(color.FgRed, color.Bold)
	red.Fprintf(os.Stderr, "Error: ")
	fmt.Fprintf(os.Stderr, "Cannot visualize %s (validation failed)\n\n", filepath.Base(filePath))

	for i, err := range errs {
		fmt.Fprintf(os.Stderr, "  %d. %v\n", i+1, err)
	}

	fmt.Fprintf(os.Stderr, "\nFix %d validation error(s) before visualizing\n", len(errs))
	os.Exit(1)
	return nil
}

func init() {
	visualizeCmd.Flags().BoolVar(&compactFlag, "compact", false, "Use compact single-line output")
	visualizeCmd.Flags().String("specs-dir", "", "Directory containing spec folders (default: specs)")
	DagCmd.AddCommand(visualizeCmd)
}
