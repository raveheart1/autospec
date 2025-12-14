package cli

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "autospec",
	Short: "Auto Claude SpecKit workflow automation",
	Long: `Auto Claude SpecKit workflow automation

Cross-platform CLI tool for SpecKit workflow validation and orchestration.
Replaces bash-based scripts with a single, performant Go binary.`,
	Example: `  # Complete workflow: specify -> plan -> tasks -> implement
  autospec all "Add user authentication feature"

  # Flexible phase selection with run command
  autospec run -spti "Add user authentication"  # All core phases
  autospec run -pi                              # Plan + implement on current spec

  # Individual phase commands
  autospec specify "Add dark mode support"
  autospec plan
  autospec tasks
  autospec implement

  # Check system dependencies
  autospec doctor`,
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringP("config", "c", ".autospec/config.json", "Path to config file")
	rootCmd.PersistentFlags().String("specs-dir", "./specs", "Directory containing feature specs")
	rootCmd.PersistentFlags().Bool("skip-preflight", false, "Skip pre-flight validation checks")
	rootCmd.PersistentFlags().BoolP("debug", "d", false, "Enable debug logging")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose output")
}
