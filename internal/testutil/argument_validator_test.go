package testutil

import (
	"testing"
)

func TestValidateArgs(t *testing.T) {
	tests := map[string]struct {
		agentName string
		args      []string
		wantErr   bool
		errMsg    string
	}{
		"claude valid args with -p flag": {
			agentName: "claude",
			args:      []string{"-p", "test prompt", "--output-format", "json"},
			wantErr:   false,
		},
		"claude valid args with --print flag": {
			agentName: "claude",
			args:      []string{"--print", "--verbose"},
			wantErr:   false,
		},
		"claude invalid output-format": {
			agentName: "claude",
			args:      []string{"--output-format", "invalid"},
			wantErr:   true,
			errMsg:    "does not match pattern",
		},
		"claude valid output-format text": {
			agentName: "claude",
			args:      []string{"--output-format", "text"},
			wantErr:   false,
		},
		"claude valid output-format stream-json": {
			agentName: "claude",
			args:      []string{"--output-format", "stream-json"},
			wantErr:   false,
		},
		"opencode valid args with -p flag": {
			agentName: "opencode",
			args:      []string{"-p", "test prompt", "--non-interactive"},
			wantErr:   false,
		},
		"opencode valid args with --prompt flag": {
			agentName: "opencode",
			args:      []string{"--prompt", "test prompt"},
			wantErr:   false,
		},
		"opencode invalid output-format": {
			agentName: "opencode",
			args:      []string{"--output-format", "stream-json"},
			wantErr:   true,
			errMsg:    "does not match pattern",
		},
		"opencode valid output-format json": {
			agentName: "opencode",
			args:      []string{"--output-format", "json"},
			wantErr:   false,
		},
		"unknown agent": {
			agentName: "unknown",
			args:      []string{"-p", "test"},
			wantErr:   true,
			errMsg:    "unknown agent",
		},
		"empty args": {
			agentName: "claude",
			args:      []string{},
			wantErr:   false,
		},
		"claude with equals syntax": {
			agentName: "claude",
			args:      []string{"--output-format=json"},
			wantErr:   false,
		},
		"claude with model flag": {
			agentName: "claude",
			args:      []string{"--model", "claude-3-opus"},
			wantErr:   false,
		},
		"opencode with debug and quiet": {
			agentName: "opencode",
			args:      []string{"--debug", "--quiet"},
			wantErr:   false,
		},
	}

	validator := GetDefaultValidator()

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			err := validator.ValidateArgs(tt.agentName, tt.args)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errMsg)
					return
				}
				if tt.errMsg != "" && !containsString(err.Error(), tt.errMsg) {
					t.Errorf("expected error containing %q, got %q", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestValidateArgsRequiredFlags(t *testing.T) {
	tests := map[string]struct {
		schema  *ArgumentSchema
		args    []string
		wantErr bool
		errMsg  string
	}{
		"missing required flag": {
			schema: &ArgumentSchema{
				AgentName:     "test",
				RequiredFlags: []string{"required-flag"},
				ValidFlags:    map[string]FlagSpec{},
			},
			args:    []string{"-p", "test"},
			wantErr: true,
			errMsg:  "missing required flag",
		},
		"required flag present with dashes": {
			schema: &ArgumentSchema{
				AgentName:     "test",
				RequiredFlags: []string{"required-flag"},
				ValidFlags:    map[string]FlagSpec{},
			},
			args:    []string{"--required-flag", "value"},
			wantErr: false,
		},
		"required flag present without dashes in schema": {
			schema: &ArgumentSchema{
				AgentName:     "test",
				RequiredFlags: []string{"flag"},
				ValidFlags:    map[string]FlagSpec{},
			},
			args:    []string{"-flag", "value"},
			wantErr: false,
		},
		"multiple required flags all present": {
			schema: &ArgumentSchema{
				AgentName:     "test",
				RequiredFlags: []string{"flag1", "flag2"},
				ValidFlags:    map[string]FlagSpec{},
			},
			args:    []string{"--flag1", "v1", "--flag2", "v2"},
			wantErr: false,
		},
		"multiple required flags one missing": {
			schema: &ArgumentSchema{
				AgentName:     "test",
				RequiredFlags: []string{"flag1", "flag2"},
				ValidFlags:    map[string]FlagSpec{},
			},
			args:    []string{"--flag1", "v1"},
			wantErr: true,
			errMsg:  "missing required flag: flag2",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			validator := NewArgumentValidator()
			validator.RegisterSchema(tt.schema)

			err := validator.ValidateArgs(tt.schema.AgentName, tt.args)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errMsg)
					return
				}
				if tt.errMsg != "" && !containsString(err.Error(), tt.errMsg) {
					t.Errorf("expected error containing %q, got %q", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestValidateArgsEdgeCases(t *testing.T) {
	tests := map[string]struct {
		agentName string
		args      []string
		wantErr   bool
		errMsg    string
	}{
		"special characters in prompt": {
			agentName: "claude",
			args:      []string{"-p", `test "quoted" prompt with 'single' quotes`},
			wantErr:   false,
		},
		"newlines in prompt": {
			agentName: "claude",
			args:      []string{"-p", "line1\nline2\nline3"},
			wantErr:   false,
		},
		"unicode in prompt": {
			agentName: "claude",
			args:      []string{"-p", "test with émojis 🎉 and ünïcödé"},
			wantErr:   false,
		},
		"empty string value": {
			agentName: "claude",
			args:      []string{"-p", ""},
			wantErr:   false,
		},
		"multiple equals in value": {
			agentName: "claude",
			args:      []string{"--system-prompt=key=value=extra"},
			wantErr:   false,
		},
		"mixed short and long flags": {
			agentName: "claude",
			args:      []string{"-p", "test", "--verbose", "--output-format", "json"},
			wantErr:   false,
		},
		"flag without value followed by another flag": {
			agentName: "claude",
			args:      []string{"--verbose", "--print"},
			wantErr:   false,
		},
	}

	validator := GetDefaultValidator()

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			err := validator.ValidateArgs(tt.agentName, tt.args)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
					return
				}
				if tt.errMsg != "" && !containsString(err.Error(), tt.errMsg) {
					t.Errorf("expected error containing %q, got %q", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestAgentSpecificSchemaDifferences(t *testing.T) {
	tests := map[string]struct {
		agentName string
		args      []string
		wantErr   bool
		errMsg    string
	}{
		"claude allows stream-json output": {
			agentName: "claude",
			args:      []string{"--output-format", "stream-json"},
			wantErr:   false,
		},
		"opencode does not allow stream-json output": {
			agentName: "opencode",
			args:      []string{"--output-format", "stream-json"},
			wantErr:   true,
			errMsg:    "does not match pattern",
		},
		"claude has dangerously-skip-permissions flag": {
			agentName: "claude",
			args:      []string{"--dangerously-skip-permissions"},
			wantErr:   false,
		},
		"opencode has non-interactive flag": {
			agentName: "opencode",
			args:      []string{"--non-interactive"},
			wantErr:   false,
		},
		"opencode has provider flag": {
			agentName: "opencode",
			args:      []string{"--provider", "anthropic"},
			wantErr:   false,
		},
	}

	validator := GetDefaultValidator()

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			err := validator.ValidateArgs(tt.agentName, tt.args)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errMsg)
					return
				}
				if tt.errMsg != "" && !containsString(err.Error(), tt.errMsg) {
					t.Errorf("expected error containing %q, got %q", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestGetSchema(t *testing.T) {
	tests := map[string]struct {
		agentName string
		wantFound bool
	}{
		"claude schema exists": {
			agentName: "claude",
			wantFound: true,
		},
		"opencode schema exists": {
			agentName: "opencode",
			wantFound: true,
		},
		"unknown schema not found": {
			agentName: "unknown",
			wantFound: false,
		},
	}

	validator := GetDefaultValidator()

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			schema, found := validator.GetSchema(tt.agentName)

			if found != tt.wantFound {
				t.Errorf("expected found=%v, got %v", tt.wantFound, found)
			}

			if tt.wantFound && schema == nil {
				t.Error("expected non-nil schema when found=true")
			}

			if tt.wantFound && schema.AgentName != tt.agentName {
				t.Errorf("expected schema.AgentName=%q, got %q", tt.agentName, schema.AgentName)
			}
		})
	}
}

func TestIsRealCLIAvailable(t *testing.T) {
	tests := map[string]struct {
		agentName   string
		wantAvail   bool
		wantMsgPart string
	}{
		"unknown agent returns false": {
			agentName:   "nonexistent-agent-xyz",
			wantAvail:   false,
			wantMsgPart: "unknown agent",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			avail, msg := IsRealCLIAvailable(tt.agentName)

			if avail != tt.wantAvail {
				t.Errorf("expected available=%v, got %v", tt.wantAvail, avail)
			}

			if tt.wantMsgPart != "" && !containsString(msg, tt.wantMsgPart) {
				t.Errorf("expected message containing %q, got %q", tt.wantMsgPart, msg)
			}
		})
	}
}

func TestClaudeSchemaFields(t *testing.T) {
	schema := ClaudeSchema()

	if schema.AgentName != "claude" {
		t.Errorf("expected AgentName=claude, got %q", schema.AgentName)
	}

	if schema.BinaryName != "claude" {
		t.Errorf("expected BinaryName=claude, got %q", schema.BinaryName)
	}

	expectedFlags := []string{"p", "print", "output-format", "verbose", "model", "max-turns"}
	for _, flag := range expectedFlags {
		if _, ok := schema.ValidFlags[flag]; !ok {
			t.Errorf("expected flag %q in ValidFlags", flag)
		}
	}

	expectedMethods := []string{"-p", "--print", "stdin"}
	if len(schema.PromptDeliveryMethods) != len(expectedMethods) {
		t.Errorf("expected %d prompt delivery methods, got %d",
			len(expectedMethods), len(schema.PromptDeliveryMethods))
	}
}

func TestOpenCodeSchemaFields(t *testing.T) {
	schema := OpenCodeSchema()

	if schema.AgentName != "opencode" {
		t.Errorf("expected AgentName=opencode, got %q", schema.AgentName)
	}

	if schema.BinaryName != "opencode" {
		t.Errorf("expected BinaryName=opencode, got %q", schema.BinaryName)
	}

	expectedFlags := []string{"p", "prompt", "non-interactive", "output-format", "provider"}
	for _, flag := range expectedFlags {
		if _, ok := schema.ValidFlags[flag]; !ok {
			t.Errorf("expected flag %q in ValidFlags", flag)
		}
	}

	expectedMethods := []string{"-p", "--prompt", "stdin"}
	if len(schema.PromptDeliveryMethods) != len(expectedMethods) {
		t.Errorf("expected %d prompt delivery methods, got %d",
			len(expectedMethods), len(schema.PromptDeliveryMethods))
	}
}

func TestRegisterSchema(t *testing.T) {
	validator := NewArgumentValidator()

	schema := &ArgumentSchema{
		AgentName:     "custom",
		BinaryName:    "custom-cli",
		RequiredFlags: []string{"required"},
		ValidFlags: map[string]FlagSpec{
			"flag1": {Type: "string"},
		},
		PromptDeliveryMethods: []string{"-p"},
	}

	validator.RegisterSchema(schema)

	retrieved, found := validator.GetSchema("custom")
	if !found {
		t.Fatal("expected to find registered schema")
	}

	if retrieved.BinaryName != "custom-cli" {
		t.Errorf("expected BinaryName=custom-cli, got %q", retrieved.BinaryName)
	}

	if len(retrieved.RequiredFlags) != 1 || retrieved.RequiredFlags[0] != "required" {
		t.Error("RequiredFlags not preserved correctly")
	}
}

func TestNormalizeFlag(t *testing.T) {
	tests := map[string]struct {
		input string
		want  string
	}{
		"single dash": {
			input: "-p",
			want:  "p",
		},
		"double dash": {
			input: "--output-format",
			want:  "output-format",
		},
		"no dash": {
			input: "flag",
			want:  "flag",
		},
		"uppercase": {
			input: "--OUTPUT-FORMAT",
			want:  "output-format",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := normalizeFlag(tt.input)
			if got != tt.want {
				t.Errorf("normalizeFlag(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
