package cli

import (
	"fmt"
	"strings"

	"github.com/anthropics/auto-claude-speckit/internal/config"
	"github.com/anthropics/auto-claude-speckit/internal/workflow"
	"github.com/spf13/cobra"
)

var planCmd = &cobra.Command{
	Use:   "plan [optional-prompt]",
	Short: "Execute the planning phase for the current spec",
	Long: `Execute the /autospec.plan command for the current specification.

The plan command will:
- Auto-detect the current spec from git branch or most recent spec
- Execute the planning workflow
- Create plan.md, research.md, and data-model.md

You can optionally provide a prompt to guide the planning process:
  autospec plan "Focus on security best practices"
  autospec plan "Optimize for performance"

Examples:
  autospec plan                           # Run planning with no additional guidance
  autospec plan "Consider scalability"    # Run planning with focus on scalability`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get optional prompt from args
		var prompt string
		if len(args) > 0 {
			prompt = strings.Join(args, " ")
		}

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
		orch := workflow.NewWorkflowOrchestrator(cfg)

		// Execute plan phase
		if err := orch.ExecutePlan("", prompt); err != nil {
			return err
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(planCmd)
}
