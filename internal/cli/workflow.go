package cli

import (
	"fmt"
	"os"

	"github.com/anthropics/auto-claude-speckit/internal/config"
	clierrors "github.com/anthropics/auto-claude-speckit/internal/errors"
	"github.com/anthropics/auto-claude-speckit/internal/workflow"
	"github.com/spf13/cobra"
)

var workflowCmd = &cobra.Command{
	Use:   "workflow <feature-description>",
	Short: "Run complete specify → plan → tasks workflow",
	Long: `Run the complete SpecKit workflow with automatic validation and retry.

This command will:
1. Run pre-flight checks (unless --skip-preflight)
2. Execute /autospec.specify with the feature description
3. Validate spec.yaml exists
4. Execute /autospec.plan
5. Validate plan.yaml exists
6. Execute /autospec.tasks
7. Validate tasks.yaml exists

Each phase is validated and will retry up to max_retries times if validation fails.`,
	Example: `  # Generate spec, plan, and tasks (no implementation)
  autospec workflow "Add user authentication feature"

  # Useful for review before implementation
  autospec workflow "Refactor database layer"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		featureDescription := args[0]

		// Get flags
		configPath, _ := cmd.Flags().GetString("config")
		skipPreflight, _ := cmd.Flags().GetBool("skip-preflight")
		maxRetries, _ := cmd.Flags().GetInt("max-retries")

		// Load configuration
		cfg, err := config.Load(configPath)
		if err != nil {
			cliErr := clierrors.ConfigParseError(configPath, err)
			clierrors.PrintError(cliErr)
			return cliErr
		}

		// Override skip-preflight from flag if set
		if cmd.Flags().Changed("skip-preflight") {
			cfg.SkipPreflight = skipPreflight
		}

		// Override max-retries from flag if set
		if cmd.Flags().Changed("max-retries") {
			cfg.MaxRetries = maxRetries
		}

		// Check if constitution exists (required for all workflow phases)
		constitutionCheck := workflow.CheckConstitutionExists()
		if !constitutionCheck.Exists {
			fmt.Fprint(os.Stderr, constitutionCheck.ErrorMessage)
			return fmt.Errorf("constitution required")
		}

		// Create workflow orchestrator
		orchestrator := workflow.NewWorkflowOrchestrator(cfg)

		// Run complete workflow
		if err := orchestrator.RunCompleteWorkflow(featureDescription); err != nil {
			return err
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(workflowCmd)

	// Command-specific flags
	workflowCmd.Flags().IntP("max-retries", "r", 0, "Override max retry attempts (0 = use config)")
}
