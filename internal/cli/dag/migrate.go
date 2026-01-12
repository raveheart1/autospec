package dag

import (
	"fmt"
	"os"

	"github.com/ariel-frischer/autospec/internal/dag"
	clierrors "github.com/ariel-frischer/autospec/internal/errors"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var migrateCmd = &cobra.Command{
	Use:   "migrate-state <workflow-file>",
	Short: "Migrate legacy state file to inline state in dag.yaml",
	Long: `Migrate state from a legacy state file to inline state in dag.yaml.

The dag migrate-state command:
- Detects legacy state files in .autospec/state/dag-runs/
- Converts state to the new inline format (run, specs, staging sections)
- Writes state directly to the dag.yaml file
- Removes the legacy state file after successful migration

This migration happens automatically on 'dag run', but this command
allows explicit migration without running the DAG.

Exit codes:
  0 - Migration completed (or no legacy state to migrate)
  1 - Migration failed`,
	Example: `  # Migrate state for a specific DAG
  autospec dag migrate-state .autospec/dags/my-workflow.yaml`,
	Args: cobra.ExactArgs(1),
	RunE: runDagMigrateState,
}

func init() {
	DagCmd.AddCommand(migrateCmd)
}

func runDagMigrateState(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true

	dagPath := args[0]
	if dagPath == "" {
		cliErr := clierrors.NewArgumentError("workflow-file is required")
		clierrors.PrintError(cliErr)
		return cliErr
	}

	return executeMigrateState(dagPath)
}

func executeMigrateState(dagPath string) error {
	// Check if legacy state exists
	legacyPath, found := dag.DetectLegacyStateFile(dagPath)
	if !found {
		fmt.Println("No legacy state file found for migration.")
		return checkExistingInlineState(dagPath)
	}

	fmt.Printf("Found legacy state file: %s\n", legacyPath)
	fmt.Printf("Migrating to inline state in: %s\n\n", dagPath)

	// Perform migration
	if err := dag.MigrateLegacyState(dagPath); err != nil {
		printMigrateError(dagPath, err)
		return err
	}

	printMigrateSuccess(dagPath, legacyPath)
	return nil
}

// checkExistingInlineState reports on existing inline state when no legacy state found.
func checkExistingInlineState(dagPath string) error {
	config, err := dag.LoadDAGConfigFull(dagPath)
	if err != nil {
		return fmt.Errorf("loading DAG config: %w", err)
	}

	if dag.HasInlineState(config) {
		fmt.Println("DAG already has inline state - no migration needed.")
	} else {
		fmt.Println("DAG has no state (neither legacy nor inline).")
	}

	return nil
}

func printMigrateError(dagPath string, err error) {
	red := color.New(color.FgRed, color.Bold)
	red.Fprintf(os.Stderr, "Error: ")
	fmt.Fprintf(os.Stderr, "Failed to migrate state for %s\n", dagPath)
	fmt.Fprintf(os.Stderr, "  %v\n", err)
}

func printMigrateSuccess(dagPath, legacyPath string) {
	green := color.New(color.FgGreen, color.Bold)
	green.Print("✓ Migration Complete\n")
	fmt.Printf("  State migrated to: %s\n", dagPath)
	fmt.Printf("  Legacy file removed: %s\n", legacyPath)
}
