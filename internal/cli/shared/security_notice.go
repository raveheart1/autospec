// Package shared provides constants and types used across CLI subpackages.
package shared

import (
	"io"

	"github.com/ariel-frischer/autospec/internal/config"
	"github.com/ariel-frischer/autospec/internal/workflow"
)

// ShowSecurityNotice displays the security notice if not already shown.
// This is a convenience wrapper around workflow.ShowSecurityNoticeOnce.
// Call this after resolving the agent but before starting any workflow execution.
//
// The notice is only shown for Claude since it relates to --dangerously-skip-permissions.
// For other agents (opencode, gemini, etc.), the notice is skipped.
//
// Example usage in a command:
//
//	cfg, err := config.Load(configPath)
//	if err != nil { return err }
//	agent, err := shared.ResolveAgent(cmd, cfg)
//	if err != nil { return err }
//	shared.ShowSecurityNotice(cmd.OutOrStdout(), cfg, agent.Name())
//	// ... continue with workflow execution
func ShowSecurityNotice(out io.Writer, cfg *config.Configuration, agentName string) {
	workflow.ShowSecurityNoticeOnce(out, cfg, agentName)
}
