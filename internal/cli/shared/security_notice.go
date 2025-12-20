// Package shared provides constants and types used across CLI subpackages.
package shared

import (
	"io"

	"github.com/ariel-frischer/autospec/internal/config"
	"github.com/ariel-frischer/autospec/internal/workflow"
)

// ShowSecurityNotice displays the security notice if not already shown.
// This is a convenience wrapper around workflow.ShowSecurityNoticeOnce.
// Call this after loading config but before starting any workflow execution.
//
// Example usage in a command:
//
//	cfg, err := config.Load(configPath)
//	if err != nil { return err }
//	shared.ShowSecurityNotice(cmd.OutOrStdout(), cfg)
//	// ... continue with workflow execution
func ShowSecurityNotice(out io.Writer, cfg *config.Configuration) {
	workflow.ShowSecurityNoticeOnce(out, cfg)
}
