// Package testutil provides test utilities and helpers for autospec tests.
package testutil

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

// FlagSpec defines constraints for a CLI flag value.
type FlagSpec struct {
	// Type specifies the expected value type: "bool", "string", "path", "int"
	Type string
	// Pattern is an optional regex pattern for validating string values
	Pattern string
	// Required indicates if this flag must always be present
	Required bool
	// Description provides documentation for the flag
	Description string
}

// ArgumentSchema defines the valid CLI argument structure for an agent.
type ArgumentSchema struct {
	// AgentName identifies the CLI agent (e.g., "claude", "opencode")
	AgentName string
	// RequiredFlags lists flags that must be present in every invocation
	RequiredFlags []string
	// ValidFlags maps flag names to their specifications
	ValidFlags map[string]FlagSpec
	// PromptDeliveryMethods lists valid ways to deliver prompts (-p, --print, stdin)
	PromptDeliveryMethods []string
	// BinaryName is the executable name to look for in PATH
	BinaryName string
}

// ArgumentValidator validates CLI arguments against an agent schema.
type ArgumentValidator struct {
	schemas map[string]*ArgumentSchema
}

// NewArgumentValidator creates a new validator with pre-registered schemas.
func NewArgumentValidator() *ArgumentValidator {
	v := &ArgumentValidator{
		schemas: make(map[string]*ArgumentSchema),
	}
	return v
}

// RegisterSchema adds an argument schema to the validator.
func (v *ArgumentValidator) RegisterSchema(schema *ArgumentSchema) {
	v.schemas[schema.AgentName] = schema
}

// GetSchema returns the schema for the given agent name.
func (v *ArgumentValidator) GetSchema(agentName string) (*ArgumentSchema, bool) {
	schema, ok := v.schemas[agentName]
	return schema, ok
}

// ValidateArgs validates arguments against the schema for the given agent.
// Returns an error describing any validation failures.
func (v *ArgumentValidator) ValidateArgs(agentName string, args []string) error {
	schema, ok := v.schemas[agentName]
	if !ok {
		return fmt.Errorf("unknown agent: %s", agentName)
	}

	if err := v.checkRequiredFlags(schema, args); err != nil {
		return err
	}

	if err := v.checkFlagValues(schema, args); err != nil {
		return err
	}

	return nil
}

// checkRequiredFlags verifies all required flags are present.
func (v *ArgumentValidator) checkRequiredFlags(schema *ArgumentSchema, args []string) error {
	argSet := makeArgSet(args)

	for _, required := range schema.RequiredFlags {
		if !argSet[required] && !argSet["-"+required] && !argSet["--"+required] {
			normalizedRequired := normalizeFlag(required)
			found := false
			for arg := range argSet {
				if normalizeFlag(arg) == normalizedRequired {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("missing required flag: %s", required)
			}
		}
	}
	return nil
}

// checkFlagValues validates flag values against their specifications.
func (v *ArgumentValidator) checkFlagValues(schema *ArgumentSchema, args []string) error {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !strings.HasPrefix(arg, "-") {
			continue
		}

		flagName, flagValue := parseFlagAndValue(arg, args, i)
		if flagName == "" {
			continue
		}

		spec, ok := schema.ValidFlags[normalizeFlag(flagName)]
		if !ok {
			spec, ok = schema.ValidFlags[flagName]
		}
		if !ok {
			continue
		}

		if err := validateFlagValue(flagName, flagValue, spec); err != nil {
			return err
		}

		if flagValue != "" && !strings.Contains(arg, "=") && i+1 < len(args) && args[i+1] == flagValue {
			i++
		}
	}
	return nil
}

// parseFlagAndValue extracts flag name and value from arguments.
func parseFlagAndValue(arg string, args []string, index int) (string, string) {
	if strings.Contains(arg, "=") {
		parts := strings.SplitN(arg, "=", 2)
		return parts[0], parts[1]
	}

	flagName := arg
	var flagValue string
	if index+1 < len(args) && !strings.HasPrefix(args[index+1], "-") {
		flagValue = args[index+1]
	}
	return flagName, flagValue
}

// validateFlagValue checks if a flag value matches its specification.
func validateFlagValue(flagName, flagValue string, spec FlagSpec) error {
	if spec.Type == "bool" {
		return nil
	}

	if spec.Pattern != "" && flagValue != "" {
		matched, err := regexp.MatchString(spec.Pattern, flagValue)
		if err != nil {
			return fmt.Errorf("invalid pattern for flag %s: %w", flagName, err)
		}
		if !matched {
			return fmt.Errorf("flag %s value %q does not match pattern %s", flagName, flagValue, spec.Pattern)
		}
	}

	return nil
}

// makeArgSet creates a set of flags present in args for quick lookup.
func makeArgSet(args []string) map[string]bool {
	set := make(map[string]bool)
	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			if strings.Contains(arg, "=") {
				parts := strings.SplitN(arg, "=", 2)
				set[parts[0]] = true
			} else {
				set[arg] = true
			}
		}
	}
	return set
}

// normalizeFlag removes leading dashes and converts to lowercase.
func normalizeFlag(flag string) string {
	flag = strings.TrimPrefix(flag, "--")
	flag = strings.TrimPrefix(flag, "-")
	return strings.ToLower(flag)
}

// IsRealCLIAvailable checks if the real CLI binary is available in PATH.
// Returns true if the binary exists, false otherwise with a descriptive message.
func IsRealCLIAvailable(agentName string) (bool, string) {
	schema, ok := defaultValidator.GetSchema(agentName)
	if !ok {
		return false, fmt.Sprintf("unknown agent: %s", agentName)
	}

	binaryName := schema.BinaryName
	if binaryName == "" {
		binaryName = agentName
	}

	path, err := exec.LookPath(binaryName)
	if err != nil {
		return false, fmt.Sprintf("%s CLI not found in PATH (looking for %q)", agentName, binaryName)
	}

	return true, fmt.Sprintf("%s CLI found at: %s", agentName, path)
}

// SkipIfCLIUnavailable is a test helper that skips the test if the CLI is not available.
func SkipIfCLIUnavailable(t interface{ Skip(...any) }, agentName string) {
	if available, msg := IsRealCLIAvailable(agentName); !available {
		t.Skip(msg)
	}
}

// defaultValidator is the global validator instance with pre-registered schemas.
var defaultValidator *ArgumentValidator

func init() {
	defaultValidator = NewArgumentValidator()
	defaultValidator.RegisterSchema(ClaudeSchema())
	defaultValidator.RegisterSchema(OpenCodeSchema())
}

// GetDefaultValidator returns the global validator with pre-registered schemas.
func GetDefaultValidator() *ArgumentValidator {
	return defaultValidator
}

// ClaudeSchema returns the argument schema for the Claude CLI.
func ClaudeSchema() *ArgumentSchema {
	return &ArgumentSchema{
		AgentName:     "claude",
		BinaryName:    "claude",
		RequiredFlags: []string{},
		ValidFlags: map[string]FlagSpec{
			"p": {
				Type:        "string",
				Description: "Prompt to send to Claude",
			},
			"print": {
				Type:        "bool",
				Description: "Print the prompt to stdout",
			},
			"output-format": {
				Type:        "string",
				Pattern:     "^(text|json|stream-json)$",
				Description: "Output format (text, json, stream-json)",
			},
			"verbose": {
				Type:        "bool",
				Description: "Enable verbose output",
			},
			"model": {
				Type:        "string",
				Description: "Model to use",
			},
			"max-turns": {
				Type:        "int",
				Description: "Maximum conversation turns",
			},
			"system-prompt": {
				Type:        "string",
				Description: "System prompt to use",
			},
			"append-system-prompt": {
				Type:        "string",
				Description: "Additional system prompt to append",
			},
			"allowedtools": {
				Type:        "string",
				Description: "Comma-separated list of allowed tools",
			},
			"disallowedtools": {
				Type:        "string",
				Description: "Comma-separated list of disallowed tools",
			},
			"mcp-config": {
				Type:        "path",
				Description: "Path to MCP configuration file",
			},
			"permission-prompt-tool": {
				Type:        "string",
				Description: "Tool for permission prompts",
			},
			"resume": {
				Type:        "string",
				Description: "Resume a previous conversation",
			},
			"continue": {
				Type:        "bool",
				Description: "Continue the most recent conversation",
			},
			"dangerously-skip-permissions": {
				Type:        "bool",
				Description: "Skip permission prompts",
			},
		},
		PromptDeliveryMethods: []string{"-p", "--print", "stdin"},
	}
}

// OpenCodeSchema returns the argument schema for the OpenCode CLI.
func OpenCodeSchema() *ArgumentSchema {
	return &ArgumentSchema{
		AgentName:     "opencode",
		BinaryName:    "opencode",
		RequiredFlags: []string{},
		ValidFlags: map[string]FlagSpec{
			"p": {
				Type:        "string",
				Description: "Prompt to send",
			},
			"prompt": {
				Type:        "string",
				Description: "Prompt to send (long form)",
			},
			"non-interactive": {
				Type:        "bool",
				Description: "Run in non-interactive mode",
			},
			"output-format": {
				Type:        "string",
				Pattern:     "^(text|json)$",
				Description: "Output format (text, json)",
			},
			"verbose": {
				Type:        "bool",
				Description: "Enable verbose output",
			},
			"model": {
				Type:        "string",
				Description: "Model to use",
			},
			"provider": {
				Type:        "string",
				Description: "LLM provider to use",
			},
			"cwd": {
				Type:        "path",
				Description: "Working directory",
			},
			"debug": {
				Type:        "bool",
				Description: "Enable debug mode",
			},
			"quiet": {
				Type:        "bool",
				Description: "Suppress output",
			},
		},
		PromptDeliveryMethods: []string{"-p", "--prompt", "stdin"},
	}
}
