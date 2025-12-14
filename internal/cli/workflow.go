package cli

import (
	"fmt"

	"github.com/anthropics/auto-claude-speckit/internal/config"
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
3. Validate spec.md exists
4. Execute /autospec.plan
5. Validate plan.md exists
6. Execute /autospec.tasks
7. Validate tasks.md exists

Each phase is validated and will retry up to max_retries times if validation fails.`,
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
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Override skip-preflight from flag if set
		if cmd.Flags().Changed("skip-preflight") {
			cfg.SkipPreflight = skipPreflight
		}

		// Override max-retries from flag if set
		if cmd.Flags().Changed("max-retries") {
			cfg.MaxRetries = maxRetries
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
