package shared

import (
	"github.com/ariel-frischer/autospec/internal/config"
	"github.com/ariel-frischer/autospec/internal/workflow"
	"github.com/spf13/cobra"
)

// ApplyOutputStyle reads the --output-style flag from the command and applies it
// to the workflow orchestrator. If the flag is set, it takes precedence over the
// config file value (cclean.style). Returns the effective OutputStyle.
func ApplyOutputStyle(cmd *cobra.Command, orch *workflow.WorkflowOrchestrator) config.OutputStyle {
	// Get the flag value (empty string if not set)
	flagValue, _ := cmd.Flags().GetString("output-style")

	// If flag is explicitly set, validate and apply it
	if cmd.Flags().Changed("output-style") && flagValue != "" {
		style, err := config.NormalizeOutputStyle(flagValue)
		if err == nil {
			orch.SetOutputStyle(style)
			return style
		}
		// If validation fails, use default (validation error was already checked at config load)
	}

	// Return the style that was set from config (via cclean.style)
	// This will be the default "default" if nothing was configured
	style, _ := config.NormalizeOutputStyle(orch.Config.Cclean.Style)
	return style
}
