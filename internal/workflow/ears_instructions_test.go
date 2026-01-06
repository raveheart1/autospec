package workflow

import (
	"strings"
	"testing"
)

func TestBuildEarsInstructions(t *testing.T) {
	t.Parallel()

	instruction := BuildEarsInstructions()

	if instruction.Name != "EarsRequirements" {
		t.Errorf("Name = %q, want %q", instruction.Name, "EarsRequirements")
	}

	if instruction.DisplayHint == "" {
		t.Error("DisplayHint should not be empty")
	}

	if instruction.Content == "" {
		t.Error("Content should not be empty")
	}

	patterns := []string{"ubiquitous", "event-driven", "state-driven", "unwanted", "optional"}
	for _, pattern := range patterns {
		if !strings.Contains(instruction.Content, pattern) {
			t.Errorf("Content should contain pattern %q", pattern)
		}
	}

	templates := []string{
		"The [system] shall [action]",
		"When [trigger], the [system] shall [action]",
		"While [state], the [system] shall [action]",
		"If [condition], then the [system] shall [action]",
		"Where [feature], the [system] shall [action]",
	}
	for _, template := range templates {
		if !strings.Contains(instruction.Content, template) {
			t.Errorf("Content should contain template %q", template)
		}
	}
}

func TestInjectEarsInstructions(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		command string
		enabled bool
		check   func(result string) bool
		desc    string
	}{
		"enabled adds EARS content": {
			command: "/autospec.specify \"feature\"",
			enabled: true,
			check: func(result string) bool {
				return strings.Contains(result, "EARS") &&
					strings.Contains(result, "ears_requirements") &&
					strings.Contains(result, "/autospec.specify")
			},
			desc: "should inject EARS instructions when enabled",
		},
		"disabled returns original": {
			command: "/autospec.specify \"feature\"",
			enabled: false,
			check: func(result string) bool {
				return result == "/autospec.specify \"feature\""
			},
			desc: "should return original command when disabled",
		},
		"empty command with enabled": {
			command: "",
			enabled: true,
			check: func(result string) bool {
				return strings.Contains(result, "EARS")
			},
			desc: "should still inject EARS even with empty command",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			result := InjectEarsInstructions(tt.command, tt.enabled)
			if !tt.check(result) {
				t.Errorf("InjectEarsInstructions(%q, %v): %s\nGot: %s", tt.command, tt.enabled, tt.desc, result)
			}
		})
	}
}

func TestEarsInstructionsContainAllPatterns(t *testing.T) {
	t.Parallel()

	instruction := BuildEarsInstructions()

	testTypes := []string{"invariant", "property", "state-machine", "exception", "feature-flag"}
	for _, testType := range testTypes {
		if !strings.Contains(instruction.Content, testType) {
			t.Errorf("Content should contain test_type %q", testType)
		}
	}

	requiredFields := []string{"trigger", "expected", "state", "condition", "feature"}
	for _, field := range requiredFields {
		if !strings.Contains(instruction.Content, field) {
			t.Errorf("Content should mention field %q", field)
		}
	}
}
