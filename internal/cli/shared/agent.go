// Package shared provides constants and types used across CLI subpackages.
package shared

import (
	"fmt"
	"strings"

	"github.com/ariel-frischer/autospec/internal/cliagent"
	"github.com/ariel-frischer/autospec/internal/config"
	"github.com/spf13/cobra"
)

// AgentFlagName is the flag name for agent override.
const AgentFlagName = "agent"

// AddAgentFlag adds the --agent flag to a command.
// The flag allows users to override the configured agent for a single execution.
func AddAgentFlag(cmd *cobra.Command) {
	cmd.Flags().String(AgentFlagName, "", fmt.Sprintf("Override agent (available: %s)", strings.Join(cliagent.List(), ", ")))
}

// ResolveAgent resolves the agent to use based on CLI flag and config.
// Priority: CLI flag > config (agent_preset/custom_agent_cmd) > legacy fields > default (claude).
func ResolveAgent(cmd *cobra.Command, cfg *config.Configuration) (cliagent.Agent, error) {
	// Check for CLI flag override
	agentName, _ := cmd.Flags().GetString(AgentFlagName)
	if agentName != "" {
		agent := cliagent.Get(agentName)
		if agent == nil {
			return nil, fmt.Errorf("unknown agent %q; available: %s", agentName, strings.Join(cliagent.List(), ", "))
		}
		return agent, nil
	}

	// Fall back to config resolution
	return cfg.GetAgent()
}

// ApplyAgentOverride updates the configuration with an agent override from CLI flag.
// This modifies the config's AgentPreset field so that workflow orchestrator picks it up.
// Returns true if an override was applied.
func ApplyAgentOverride(cmd *cobra.Command, cfg *config.Configuration) (bool, error) {
	agentName, _ := cmd.Flags().GetString(AgentFlagName)
	if agentName == "" {
		return false, nil
	}

	// Validate agent exists
	agent := cliagent.Get(agentName)
	if agent == nil {
		return false, fmt.Errorf("unknown agent %q; available: %s", agentName, strings.Join(cliagent.List(), ", "))
	}

	// Override config to use this agent
	cfg.AgentPreset = agentName
	// Clear custom_agent_cmd to ensure preset takes effect
	cfg.CustomAgentCmd = ""

	return true, nil
}
