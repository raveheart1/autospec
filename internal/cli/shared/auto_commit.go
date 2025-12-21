// Package shared provides constants and types used across CLI subpackages.
package shared

import (
	"github.com/ariel-frischer/autospec/internal/config"
	"github.com/spf13/cobra"
)

// AutoCommitFlagName is the flag name for enabling auto-commit.
const AutoCommitFlagName = "auto-commit"

// NoAutoCommitFlagName is the flag name for disabling auto-commit.
const NoAutoCommitFlagName = "no-auto-commit"

// AddAutoCommitFlags adds --auto-commit and --no-auto-commit flags to a command.
// These flags allow users to override the configured auto_commit behavior for a single execution.
// The flags are marked mutually exclusive.
func AddAutoCommitFlags(cmd *cobra.Command) {
	cmd.Flags().Bool(AutoCommitFlagName, false, "Enable automatic git commit after workflow completion")
	cmd.Flags().Bool(NoAutoCommitFlagName, false, "Disable automatic git commit after workflow completion")
	cmd.MarkFlagsMutuallyExclusive(AutoCommitFlagName, NoAutoCommitFlagName)
}

// ApplyAutoCommitOverride updates the configuration's AutoCommit field based on CLI flags.
// Returns true if an override was applied.
// Priority: --auto-commit or --no-auto-commit flag > config file > default (true).
// Also updates AutoCommitSource to SourceFlag when a flag is used.
func ApplyAutoCommitOverride(cmd *cobra.Command, cfg *config.Configuration) bool {
	if cmd.Flags().Changed(AutoCommitFlagName) {
		autoCommit, _ := cmd.Flags().GetBool(AutoCommitFlagName)
		cfg.AutoCommit = autoCommit
		cfg.AutoCommitSource = config.SourceFlag
		return true
	}
	if cmd.Flags().Changed(NoAutoCommitFlagName) {
		noAutoCommit, _ := cmd.Flags().GetBool(NoAutoCommitFlagName)
		cfg.AutoCommit = !noAutoCommit
		cfg.AutoCommitSource = config.SourceFlag
		return true
	}
	return false
}
