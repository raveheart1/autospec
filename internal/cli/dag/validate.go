package dag

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ariel-frischer/autospec/internal/dag"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var validateCmd = &cobra.Command{
	Use:   "validate <file>",
	Short: "Validate a DAG configuration file",
	Long: `Validate a DAG configuration file for structural correctness.

Checks for:
- Required fields (schema_version, dag.name, layers)
- Valid layer and feature dependencies
- No duplicate feature IDs
- No cycles in dependencies

Note: Missing spec folders are NOT errors. They will be created dynamically
when dag run executes using each feature's description field.

Exit codes:
  0 - Valid DAG file
  1 - Invalid DAG file or validation errors`,
	Example: `  # Validate a DAG file
  autospec dag validate .autospec/dags/my-workflow.yaml`,
	Args: cobra.ExactArgs(1),
	RunE: runValidate,
}

func runValidate(cmd *cobra.Command, args []string) error {
	filePath := args[0]

	if err := validateFileArg(filePath); err != nil {
		return err
	}

	result, err := dag.ParseDAGFile(filePath)
	if err != nil {
		return formatParseError(filePath, err)
	}

	specsDir, _ := cmd.Flags().GetString("specs-dir")
	if specsDir == "" {
		specsDir = "specs"
	}

	vr := dag.ValidateDAG(result.Config, result, specsDir)
	if vr.HasErrors() {
		return formatValidationErrors(filePath, vr.Errors)
	}

	printValidMessage(result.Config, vr.MissingSpecs)
	return nil
}

func validateFileArg(filePath string) error {
	info, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file not found: %s", filePath)
		}
		return fmt.Errorf("accessing file: %w", err)
	}

	if info.IsDir() {
		return fmt.Errorf("expected file, got directory: %s", filePath)
	}

	return nil
}

func validateMaxParallel(maxParallel int) error {
	if maxParallel < 1 {
		return fmt.Errorf("--max-parallel must be at least 1, got %d", maxParallel)
	}
	return nil
}

func formatParseError(filePath string, err error) error {
	red := color.New(color.FgRed, color.Bold)
	red.Fprintf(os.Stderr, "Error: ")
	fmt.Fprintf(os.Stderr, "Failed to parse %s\n", filepath.Base(filePath))
	fmt.Fprintf(os.Stderr, "  %v\n", err)
	os.Exit(1)
	return nil
}

func formatValidationErrors(filePath string, errs []error) error {
	red := color.New(color.FgRed, color.Bold)
	red.Fprintf(os.Stderr, "Error: ")
	fmt.Fprintf(os.Stderr, "Validation failed for %s\n\n", filepath.Base(filePath))

	for i, err := range errs {
		fmt.Fprintf(os.Stderr, "  %d. %v\n", i+1, err)
	}

	fmt.Fprintf(os.Stderr, "\nFound %d validation error(s)\n", len(errs))
	os.Exit(1)
	return nil
}

func printValidMessage(cfg *dag.DAGConfig, missingSpecs []*dag.MissingSpecError) {
	green := color.New(color.FgGreen, color.Bold)
	green.Print("Valid")
	fmt.Printf(" - DAG %q\n", cfg.DAG.Name)

	layerCount := len(cfg.Layers)
	featureCount := countFeatures(cfg)
	fmt.Printf("  %d layer(s), %d feature(s)\n", layerCount, featureCount)

	if len(missingSpecs) > 0 {
		cyan := color.New(color.FgCyan)
		fmt.Println()
		cyan.Printf("  📝 %d spec(s) will be created during execution:\n", len(missingSpecs))
		for _, spec := range missingSpecs {
			fmt.Printf("     - %s\n", spec.FeatureID)
		}
	}
}

func countFeatures(cfg *dag.DAGConfig) int {
	count := 0
	for _, layer := range cfg.Layers {
		count += len(layer.Features)
	}
	return count
}

func init() {
	DagCmd.AddCommand(validateCmd)
}
