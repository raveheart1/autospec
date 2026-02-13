package cliagent

// Codex implements the Agent interface for OpenAI Codex CLI.
// Command: codex exec <prompt>
type Codex struct {
	BaseAgent
}

// NewCodex creates a new Codex CLI agent.
func NewCodex() *Codex {
	return &Codex{
		BaseAgent: BaseAgent{
			AgentName:   "codex",
			Cmd:         "codex",
			VersionFlag: "--version",
			AgentCaps: Caps{
				Automatable: true,
				PromptDelivery: PromptDelivery{
					Method: PromptMethodSubcommand,
					Flag:   "exec",
				},
				// exec mode is inherently autonomous, no extra flag needed
				AutonomousFlag: "",
				// Browser OAuth auth is the default Codex login flow; API keys are optional fallback.
				// Ref: https://developers.openai.com/codex/cli/reference/#codex-login
				RequiredEnv: []string{},
				OptionalEnv: []string{"OPENAI_API_KEY", "CODEX_API_KEY"},
			},
		},
	}
}
