// Package workflow provides EARS instruction generation for agent prompt injection.
package workflow

const earsRequirementsInstructions = `## EARS Requirements Format

Include EARS (Easy Approach to Requirements Syntax) requirements in spec.yaml:

` + "```yaml" + `
ears_requirements:
  - id: "EARS-001"
    pattern: "ubiquitous"
    text: "The system shall validate all user inputs"
    test_type: "invariant"

  - id: "EARS-002"
    pattern: "event-driven"
    text: "When a user submits a form, the system shall display a confirmation message"
    test_type: "property"
    trigger: "user submits a form"
    expected: "confirmation message displayed"

  - id: "EARS-003"
    pattern: "state-driven"
    text: "While the user is logged in, the system shall show the dashboard"
    test_type: "state-machine"
    state: "user is logged in"

  - id: "EARS-004"
    pattern: "unwanted"
    text: "If the database connection fails, then the system shall retry three times"
    test_type: "exception"
    condition: "database connection fails"

  - id: "EARS-005"
    pattern: "optional"
    text: "Where dark mode is enabled, the system shall use a dark color scheme"
    test_type: "feature-flag"
    feature: "dark mode"
` + "```" + `

### Pattern Reference

| Pattern | Template | Required Fields | Test Type |
|---------|----------|-----------------|-----------|
| ubiquitous | "The [system] shall [action]" | - | invariant |
| event-driven | "When [trigger], the [system] shall [action]" | trigger, expected | property |
| state-driven | "While [state], the [system] shall [action]" | state | state-machine |
| unwanted | "If [condition], then the [system] shall [action]" | condition | exception |
| optional | "Where [feature], the [system] shall [action]" | feature | feature-flag |

### Guidelines
- Use EARS-NNN format for IDs (e.g., EARS-001)
- IDs must be unique across both functional requirements and EARS requirements
- Choose the pattern that best captures the requirement's nature
- Empty array is valid if no EARS requirements apply
`

func BuildEarsInstructions() InjectableInstruction {
	return InjectableInstruction{
		Name:        "EarsRequirements",
		DisplayHint: "include EARS-formatted requirements in spec.yaml",
		Content:     earsRequirementsInstructions,
	}
}

func InjectEarsInstructions(command string, enabled bool) string {
	if !enabled {
		return command
	}
	instruction := BuildEarsInstructions()
	return InjectInstructions(command, []InjectableInstruction{instruction})
}
