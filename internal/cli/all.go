package cli

import (
	"fmt"
	"os"

	"github.com/anthropics/auto-claude-speckit/internal/config"
	"github.com/anthropics/auto-claude-speckit/internal/workflow"
	"github.com/spf13/cobra"
)

// allCmd is the primary command for running all workflow phases
var allCmd = &cobra.Command{
	Use:   "all <feature-description>",
	Short: "Run complete specify -> plan -> tasks -> implement workflow",
	Long: `Run the complete SpecKit workflow including implementation with automatic validation and retry.

This command will:
1. Run pre-flight checks (unless --skip-preflight)
2. Execute /autospec.specify with the feature description
3. Validate spec.yaml exists
4. Execute /autospec.plan
5. Validate plan.yaml exists
6. Execute /autospec.tasks
7. Validate tasks.yaml exists
8. Execute /autospec.implement
9. Validate all tasks are completed

Each phase is validated and will retry up to max_retries times if validation fails.
This is equivalent to running 'autospec run -a <feature-description>'.`,
	Args: cobra.ExactArgs(1),
	RunE: runAllWorkflow,
}

// fullCmd is a deprecated alias for allCmd
var fullCmd = &cobra.Command{
	Use:        "full <feature-description>",
	Short:      "[DEPRECATED] Use 'all' instead. Run complete workflow",
	Long:       "DEPRECATED: This command has been renamed to 'all'. Please use 'autospec all' instead.",
	Args:       cobra.ExactArgs(1),
	Deprecated: "use 'autospec all' instead",
	Hidden:     false, // Keep visible but show deprecation warning
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Fprintln(os.Stderr, "WARNING: 'autospec full' is deprecated and will be removed in a future release.")
		fmt.Fprintln(os.Stderr, "Please use 'autospec all' instead.")
		fmt.Fprintln(os.Stderr)
		return runAllWorkflow(cmd, args)
	},
}

// runAllWorkflow is the shared implementation for both all and full commands
func runAllWorkflow(cmd *cobra.Command, args []string) error {
	featureDescription := args[0]

	// Get flags
	configPath, _ := cmd.Flags().GetString("config")
	skipPreflight, _ := cmd.Flags().GetBool("skip-preflight")
	maxRetries, _ := cmd.Flags().GetInt("max-retries")
	resume, _ := cmd.Flags().GetBool("resume")
	debug, _ := cmd.Flags().GetBool("debug")
	progress, _ := cmd.Flags().GetBool("progress")

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

	// Override show-progress from flag if set
	if cmd.Flags().Changed("progress") {
		cfg.ShowProgress = progress
	}

	// Check if constitution exists (required for all workflow phases)
	constitutionCheck := workflow.CheckConstitutionExists()
	if !constitutionCheck.Exists {
		fmt.Fprint(os.Stderr, constitutionCheck.ErrorMessage)
		return fmt.Errorf("constitution required")
	}

	// Create workflow orchestrator
	orchestrator := workflow.NewWorkflowOrchestrator(cfg)
	orchestrator.Debug = debug
	orchestrator.Executor.Debug = debug // Propagate debug to executor

	if debug {
		fmt.Println("[DEBUG] Debug mode enabled")
		fmt.Printf("[DEBUG] Config: %+v\n", cfg)
	}

	// Run full workflow
	if err := orchestrator.RunFullWorkflow(featureDescription, resume); err != nil {
		return err
	}

	return nil
}

func init() {
	// Register both commands
	rootCmd.AddCommand(allCmd)
	rootCmd.AddCommand(fullCmd)

	// Command-specific flags for allCmd
	allCmd.Flags().IntP("max-retries", "r", 0, "Override max retry attempts (0 = use config)")
	allCmd.Flags().Bool("resume", false, "Resume implementation from where it left off")
	allCmd.Flags().Bool("progress", false, "Show progress indicators (spinners) during execution")

	// Same flags for fullCmd (deprecated alias)
	fullCmd.Flags().IntP("max-retries", "r", 0, "Override max retry attempts (0 = use config)")
	fullCmd.Flags().Bool("resume", false, "Resume implementation from where it left off")
	fullCmd.Flags().Bool("progress", false, "Show progress indicators (spinners) during execution")
}
