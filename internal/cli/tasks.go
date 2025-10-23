package cli

import (
	"fmt"
	"strings"

	"github.com/anthropics/auto-claude-speckit/internal/config"
	"github.com/anthropics/auto-claude-speckit/internal/workflow"
	"github.com/spf13/cobra"
)

var tasksCmd = &cobra.Command{
	Use:   "tasks [optional-prompt]",
	Short: "Execute the task generation phase for the current spec",
	Long: `Execute the /speckit.tasks command for the current specification.

The tasks command will:
- Auto-detect the current spec from git branch or most recent spec
- Execute the task generation workflow
- Create tasks.md with actionable, dependency-ordered tasks

You can optionally provide a prompt to guide the task generation:
  autospec tasks "Break into small incremental steps"
  autospec tasks "Prioritize testing tasks"
  autospec tasks "Focus on refactoring safety"

Examples:
  autospec tasks                              # Generate tasks with no additional guidance
  autospec tasks "Make tasks very granular"   # Generate more fine-grained tasks`,
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

		// Execute tasks phase
		if err := orch.ExecuteTasks("", prompt); err != nil {
			return err
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(tasksCmd)
}
