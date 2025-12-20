package config

import (
	"bufio"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	"github.com/ariel-frischer/autospec/internal/cliagent"
)

// AgentOption represents an agent displayed in the multi-select prompt.
// Used by promptAgentSelection to show available agents with selection state.
type AgentOption struct {
	// Name is the unique identifier for the agent (e.g., "claude", "gemini").
	Name string

	// DisplayName is the human-readable name shown in prompts.
	DisplayName string

	// Recommended indicates whether this agent should be pre-selected by default.
	// Only Claude is recommended.
	Recommended bool

	// Selected indicates whether the agent is currently selected in the prompt.
	Selected bool
}

// agentDisplayNames maps agent names to their human-readable display names.
var agentDisplayNames = map[string]string{
	"claude":   "Claude Code",
	"cline":    "Cline",
	"codex":    "Codex CLI",
	"gemini":   "Gemini CLI",
	"goose":    "Goose",
	"opencode": "OpenCode",
}

// GetSupportedAgents returns all supported agents as AgentOptions.
// Claude is marked as Recommended by default. Agents are returned in
// alphabetical order by name for consistent display.
func GetSupportedAgents() []AgentOption {
	registeredAgents := cliagent.List()
	options := make([]AgentOption, 0, len(registeredAgents))

	for _, name := range registeredAgents {
		displayName := agentDisplayNames[name]
		if displayName == "" {
			// Fallback: capitalize first letter
			displayName = strings.ToUpper(name[:1]) + name[1:]
		}

		options = append(options, AgentOption{
			Name:        name,
			DisplayName: displayName,
			Recommended: name == "claude",
			Selected:    false,
		})
	}

	// Sort alphabetically by name for consistent display
	sort.Slice(options, func(i, j int) bool {
		return options[i].Name < options[j].Name
	})

	return options
}

// GetSupportedAgentsWithDefaults returns agents with selections pre-applied.
// If defaultAgents is empty, only Claude is pre-selected (as recommended).
// Otherwise, agents in defaultAgents are pre-selected.
// Unknown agent names in defaultAgents are ignored.
func GetSupportedAgentsWithDefaults(defaultAgents []string) []AgentOption {
	options := GetSupportedAgents()

	if len(defaultAgents) == 0 {
		// No defaults configured - pre-select recommended (Claude)
		for i := range options {
			if options[i].Recommended {
				options[i].Selected = true
			}
		}
		return options
	}

	// Build a set of default agent names for O(1) lookup
	defaultSet := make(map[string]bool)
	for _, name := range defaultAgents {
		defaultSet[name] = true
	}

	// Pre-select agents that are in the default set
	for i := range options {
		if defaultSet[options[i].Name] {
			options[i].Selected = true
		}
	}

	return options
}

// promptAgentSelection displays an interactive multi-select prompt for agent selection.
// It shows a numbered list of agents with selection state, accepts space-separated
// numbers to toggle selections, and supports 'done' or empty input to confirm.
//
// Returns the list of selected agent names.
//
// Parameters:
//   - r: Reader for user input (typically os.Stdin)
//   - w: Writer for output (typically os.Stdout)
//   - agents: List of agent options to display
//
// Example output:
//
//	Select agents to configure (space-separated numbers, or 'done' to confirm):
//	  [1] [x] Claude Code (Recommended)
//	  [2] [ ] Cline
//	  [3] [ ] Codex CLI
//	  [4] [ ] Gemini CLI
//	  [5] [ ] Goose
//	  [6] [ ] OpenCode
//
//	Toggle selections: 2 3
//	Done? [Y/n]:
func promptAgentSelection(r io.Reader, w io.Writer, agents []AgentOption) []string {
	scanner := bufio.NewScanner(r)

	for {
		displayAgentList(w, agents)

		fmt.Fprint(w, "\nToggle selections (space-separated numbers), or press Enter when done: ")

		if !scanner.Scan() {
			// EOF or error - return currently selected agents
			break
		}

		input := strings.TrimSpace(scanner.Text())

		// Empty input or 'done' confirms selection
		if input == "" || strings.ToLower(input) == "done" {
			break
		}

		// Parse and toggle selections
		toggleAgentSelections(agents, input)
	}

	return getSelectedAgentNames(agents)
}

// displayAgentList prints the numbered list of agents with selection state.
func displayAgentList(w io.Writer, agents []AgentOption) {
	fmt.Fprintln(w, "\nSelect AI coding agents to configure:")
	fmt.Fprintln(w)

	for i, agent := range agents {
		checkbox := "[ ]"
		if agent.Selected {
			checkbox = "[x]"
		}

		label := agent.DisplayName
		if agent.Recommended {
			label += " (Recommended)"
		}

		fmt.Fprintf(w, "  [%d] %s %s\n", i+1, checkbox, label)
	}
}

// toggleAgentSelections parses the input string and toggles agent selections.
// Input format: space-separated numbers (1-indexed).
// Invalid numbers are silently ignored.
func toggleAgentSelections(agents []AgentOption, input string) {
	parts := strings.Fields(input)

	for _, part := range parts {
		num, err := strconv.Atoi(part)
		if err != nil {
			continue // Ignore non-numeric input
		}

		// Convert to 0-indexed and validate range
		idx := num - 1
		if idx >= 0 && idx < len(agents) {
			agents[idx].Selected = !agents[idx].Selected
		}
	}
}

// getSelectedAgentNames returns the names of all selected agents.
func getSelectedAgentNames(agents []AgentOption) []string {
	var selected []string
	for _, agent := range agents {
		if agent.Selected {
			selected = append(selected, agent.Name)
		}
	}
	return selected
}
