package cliagent

import (
	"fmt"

	"github.com/ariel-frischer/autospec/internal/commands"
	"github.com/ariel-frischer/autospec/internal/opencode"
)

// OpenCode implements the Agent interface for OpenCode CLI.
// Command: opencode run <prompt> --command <command-name>
type OpenCode struct {
	BaseAgent
}

// NewOpenCode creates a new OpenCode agent.
func NewOpenCode() *OpenCode {
	return &OpenCode{
		BaseAgent: BaseAgent{
			AgentName:   "opencode",
			Cmd:         "opencode",
			VersionFlag: "--version",
			AgentCaps: Caps{
				Automatable: true,
				PromptDelivery: PromptDelivery{
					Method:          PromptMethodSubcommandWithFlag,
					Flag:            "run",
					CommandFlag:     "--command",
					InteractiveFlag: "--prompt",
					ContextFileFlag: "-f",
				},
				// run subcommand is inherently non-interactive
				AutonomousFlag: "",
				RequiredEnv:    []string{},
				OptionalEnv:    []string{"ANTHROPIC_API_KEY", "OPENAI_API_KEY", "GEMINI_API_KEY"},
			},
		},
	}
}

// ConfigureProject implements the Configurator interface for OpenCode.
// It configures the OpenCode agent for autospec:
//   - Installs command templates to .opencode/command/
//   - Adds 'autospec *': 'allow' permission to ./opencode.json
//
// This method is idempotent - calling it multiple times produces the same result.
func (o *OpenCode) ConfigureProject(projectDir, specsDir string) (ConfigResult, error) {
	// Install command templates
	if _, err := commands.InstallTemplatesForAgent("opencode", projectDir); err != nil {
		return ConfigResult{}, fmt.Errorf("installing opencode commands: %w", err)
	}

	// Configure opencode.json permissions
	settings, err := opencode.Load(projectDir)
	if err != nil {
		return ConfigResult{}, fmt.Errorf("loading opencode settings: %w", err)
	}

	if settings.HasRequiredPermission() {
		return ConfigResult{
			AlreadyConfigured: true,
		}, nil
	}

	// Check for explicit deny
	var warning string
	if settings.IsPermissionDenied() {
		warning = fmt.Sprintf("permission '%s' is explicitly denied in opencode.json", opencode.RequiredPattern)
	}

	settings.AddBashPermission(opencode.RequiredPattern, opencode.PermissionAllow)

	if err := settings.Save(); err != nil {
		return ConfigResult{}, fmt.Errorf("saving opencode settings: %w", err)
	}

	return ConfigResult{
		PermissionsAdded: []string{fmt.Sprintf("%s: %s", opencode.RequiredPattern, opencode.PermissionAllow)},
		Warning:          warning,
	}, nil
}
