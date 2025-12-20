package cliagent

import (
	"fmt"

	"github.com/ariel-frischer/autospec/internal/claude"
)

// Claude implements the Agent interface for Claude Code CLI.
// Command: claude -p <prompt> [--dangerously-skip-permissions]
type Claude struct {
	BaseAgent
}

// NewClaude creates a new Claude Code agent.
func NewClaude() *Claude {
	return &Claude{
		BaseAgent: BaseAgent{
			AgentName:   "claude",
			Cmd:         "claude",
			VersionFlag: "--version",
			AgentCaps: Caps{
				Automatable: true,
				PromptDelivery: PromptDelivery{
					Method: PromptMethodArg,
					Flag:   "-p",
				},
				AutonomousFlag: "--dangerously-skip-permissions",
				RequiredEnv:    []string{"ANTHROPIC_API_KEY"},
				OptionalEnv:    []string{"CLAUDE_MODEL"},
			},
		},
	}
}

// ConfigureProject implements the Configurator interface for Claude.
// It configures .claude/settings.local.json with required permissions for autospec:
//   - Bash(autospec:*) - run autospec commands
//   - Write(.autospec/**) - write to .autospec directory
//   - Edit(.autospec/**) - edit files in .autospec directory
//   - Write({specsDir}/**) - write to specs directory
//   - Edit({specsDir}/**) - edit files in specs directory
//
// This method is idempotent - calling it multiple times produces the same result.
func (c *Claude) ConfigureProject(projectDir, specsDir string) (ConfigResult, error) {
	settings, err := claude.Load(projectDir)
	if err != nil {
		return ConfigResult{}, fmt.Errorf("loading claude settings: %w", err)
	}

	permissions := buildClaudePermissions(specsDir)

	// Check for deny list conflicts before adding permissions
	warning := checkDenyConflicts(settings, permissions)

	added := settings.AddPermissions(permissions)

	if len(added) == 0 {
		return ConfigResult{
			AlreadyConfigured: true,
			Warning:           warning,
		}, nil
	}

	if err := settings.Save(); err != nil {
		return ConfigResult{}, fmt.Errorf("saving claude settings: %w", err)
	}

	return ConfigResult{
		PermissionsAdded: added,
		Warning:          warning,
	}, nil
}

// buildClaudePermissions generates the list of permissions required for autospec.
func buildClaudePermissions(specsDir string) []string {
	return []string{
		"Bash(autospec:*)",
		"Write(.autospec/**)",
		"Edit(.autospec/**)",
		fmt.Sprintf("Write(%s/**)", specsDir),
		fmt.Sprintf("Edit(%s/**)", specsDir),
	}
}

// checkDenyConflicts checks if any required permissions are in the deny list.
// Returns a warning message if conflicts are found, empty string otherwise.
func checkDenyConflicts(settings *claude.Settings, permissions []string) string {
	var denied []string
	for _, perm := range permissions {
		if settings.CheckDenyList(perm) {
			denied = append(denied, perm)
		}
	}

	if len(denied) == 0 {
		return ""
	}

	if len(denied) == 1 {
		return fmt.Sprintf("permission %s is explicitly denied in settings", denied[0])
	}
	return fmt.Sprintf("permissions %v are explicitly denied in settings", denied)
}
