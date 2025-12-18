// Package cli_test tests command alias definitions and resolution for all workflow commands.
// Related: internal/cli/*.go (specify.go, plan.go, tasks.go, implement.go, etc.)
// Tags: cli, aliases, commands, shortcuts, usability
package cli

import (
	"strings"
	"testing"
)

func TestCommandAliases(t *testing.T) {
	tests := map[string]struct {
		commandName   string
		expectedAlias []string
	}{
		"specify command has aliases spec and s": {
			commandName:   "specify",
			expectedAlias: []string{"spec", "s"},
		},
		"plan command has alias p": {
			commandName:   "plan",
			expectedAlias: []string{"p"},
		},
		"tasks command has alias t": {
			commandName:   "tasks",
			expectedAlias: []string{"t"},
		},
		"implement command has aliases impl and i": {
			commandName:   "implement",
			expectedAlias: []string{"impl", "i"},
		},
		"status command has alias st": {
			commandName:   "status",
			expectedAlias: []string{"st"},
		},
		"doctor command has alias doc": {
			commandName:   "doctor",
			expectedAlias: []string{"doc"},
		},
		"constitution command has alias const": {
			commandName:   "constitution",
			expectedAlias: []string{"const"},
		},
		"clarify command has alias cl": {
			commandName:   "clarify",
			expectedAlias: []string{"cl"},
		},
		"checklist command has alias chk": {
			commandName:   "checklist",
			expectedAlias: []string{"chk"},
		},
		"analyze command has alias az": {
			commandName:   "analyze",
			expectedAlias: []string{"az"},
		},
		"version command has alias v": {
			commandName:   "version",
			expectedAlias: []string{"v"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Find the command by name
			cmd, _, err := rootCmd.Find([]string{tt.commandName})
			if err != nil {
				t.Fatalf("command %q not found: %v", tt.commandName, err)
			}

			// Verify aliases
			if len(cmd.Aliases) != len(tt.expectedAlias) {
				t.Errorf("command %q has %d aliases, want %d; got %v, want %v",
					tt.commandName, len(cmd.Aliases), len(tt.expectedAlias), cmd.Aliases, tt.expectedAlias)
				return
			}

			for i, expected := range tt.expectedAlias {
				if cmd.Aliases[i] != expected {
					t.Errorf("command %q alias[%d] = %q, want %q", tt.commandName, i, cmd.Aliases[i], expected)
				}
			}
		})
	}
}

func TestAliasResolution(t *testing.T) {
	tests := map[string]struct {
		alias       string
		commandName string
	}{
		"spec resolves to specify":       {alias: "spec", commandName: "specify"},
		"s resolves to specify":          {alias: "s", commandName: "specify"},
		"p resolves to plan":             {alias: "p", commandName: "plan"},
		"t resolves to tasks":            {alias: "t", commandName: "tasks"},
		"impl resolves to implement":     {alias: "impl", commandName: "implement"},
		"i resolves to implement":        {alias: "i", commandName: "implement"},
		"st resolves to status":          {alias: "st", commandName: "status"},
		"doc resolves to doctor":         {alias: "doc", commandName: "doctor"},
		"const resolves to constitution": {alias: "const", commandName: "constitution"},
		"cl resolves to clarify":         {alias: "cl", commandName: "clarify"},
		"chk resolves to checklist":      {alias: "chk", commandName: "checklist"},
		"az resolves to analyze":         {alias: "az", commandName: "analyze"},
		"v resolves to version":          {alias: "v", commandName: "version"},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			cmd, _, err := rootCmd.Find([]string{tt.alias})
			if err != nil {
				t.Fatalf("alias %q not found: %v", tt.alias, err)
			}

			if cmd.Name() != tt.commandName {
				t.Errorf("alias %q resolved to %q, want %q", tt.alias, cmd.Name(), tt.commandName)
			}
		})
	}
}

func TestHelpTextShowsAliases(t *testing.T) {
	tests := map[string]struct {
		commandName    string
		expectedInHelp string
	}{
		"specify help shows aliases":      {commandName: "specify", expectedInHelp: "Aliases:"},
		"plan help shows aliases":         {commandName: "plan", expectedInHelp: "Aliases:"},
		"tasks help shows aliases":        {commandName: "tasks", expectedInHelp: "Aliases:"},
		"implement help shows aliases":    {commandName: "implement", expectedInHelp: "Aliases:"},
		"status help shows aliases":       {commandName: "status", expectedInHelp: "Aliases:"},
		"doctor help shows aliases":       {commandName: "doctor", expectedInHelp: "Aliases:"},
		"constitution help shows aliases": {commandName: "constitution", expectedInHelp: "Aliases:"},
		"clarify help shows aliases":      {commandName: "clarify", expectedInHelp: "Aliases:"},
		"checklist help shows aliases":    {commandName: "checklist", expectedInHelp: "Aliases:"},
		"analyze help shows aliases":      {commandName: "analyze", expectedInHelp: "Aliases:"},
		"version help shows aliases":      {commandName: "version", expectedInHelp: "Aliases:"},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			cmd, _, err := rootCmd.Find([]string{tt.commandName})
			if err != nil {
				t.Fatalf("command %q not found: %v", tt.commandName, err)
			}

			// Get the usage string which includes aliases
			usage := cmd.UsageString()
			if !strings.Contains(usage, tt.expectedInHelp) {
				t.Errorf("command %q help does not contain %q:\n%s", tt.commandName, tt.expectedInHelp, usage)
			}
		})
	}
}
