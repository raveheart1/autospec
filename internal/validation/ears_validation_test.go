package validation

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestValidateEarsRequirements(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		yamlContent string
		wantValid   bool
		wantErrors  []string
	}{
		"valid ubiquitous pattern": {
			yamlContent: `
- id: "EARS-001"
  pattern: "ubiquitous"
  text: "The system shall validate inputs"
  test_type: "invariant"
`,
			wantValid: true,
		},
		"valid event-driven pattern": {
			yamlContent: `
- id: "EARS-001"
  pattern: "event-driven"
  text: "When user submits, the system shall confirm"
  test_type: "property"
  trigger: "user submits form"
  expected: "confirmation shown"
`,
			wantValid: true,
		},
		"valid state-driven pattern": {
			yamlContent: `
- id: "EARS-001"
  pattern: "state-driven"
  text: "While logged in, the system shall show dashboard"
  test_type: "state-machine"
  state: "user is logged in"
`,
			wantValid: true,
		},
		"valid unwanted pattern": {
			yamlContent: `
- id: "EARS-001"
  pattern: "unwanted"
  text: "If error occurs, then the system shall retry"
  test_type: "exception"
  condition: "error occurs"
`,
			wantValid: true,
		},
		"valid optional pattern": {
			yamlContent: `
- id: "EARS-001"
  pattern: "optional"
  text: "Where dark mode, the system shall use dark colors"
  test_type: "feature-flag"
  feature: "dark mode"
`,
			wantValid: true,
		},
		"empty array is valid": {
			yamlContent: `[]`,
			wantValid:   true,
		},
		"missing trigger for event-driven": {
			yamlContent: `
- id: "EARS-001"
  pattern: "event-driven"
  text: "When X, system does Y"
  test_type: "property"
`,
			wantValid:  false,
			wantErrors: []string{"missing required field 'trigger'", "missing required field 'expected'"},
		},
		"missing state for state-driven": {
			yamlContent: `
- id: "EARS-001"
  pattern: "state-driven"
  text: "While X, system does Y"
  test_type: "state-machine"
`,
			wantValid:  false,
			wantErrors: []string{"missing required field 'state'"},
		},
		"missing condition for unwanted": {
			yamlContent: `
- id: "EARS-001"
  pattern: "unwanted"
  text: "If X, then system does Y"
  test_type: "exception"
`,
			wantValid:  false,
			wantErrors: []string{"missing required field 'condition'"},
		},
		"missing feature for optional": {
			yamlContent: `
- id: "EARS-001"
  pattern: "optional"
  text: "Where X, system does Y"
  test_type: "feature-flag"
`,
			wantValid:  false,
			wantErrors: []string{"missing required field 'feature'"},
		},
		"invalid pattern value": {
			yamlContent: `
- id: "EARS-001"
  pattern: "invalid-pattern"
  text: "System does something"
  test_type: "invariant"
`,
			wantValid:  false,
			wantErrors: []string{"invalid EARS pattern"},
		},
		"invalid test_type value": {
			yamlContent: `
- id: "EARS-001"
  pattern: "ubiquitous"
  text: "System does something"
  test_type: "invalid-type"
`,
			wantValid:  false,
			wantErrors: []string{"invalid test_type"},
		},
		"invalid ID format": {
			yamlContent: `
- id: "REQ-001"
  pattern: "ubiquitous"
  text: "System does something"
  test_type: "invariant"
`,
			wantValid:  false,
			wantErrors: []string{"invalid EARS ID format"},
		},
		"duplicate EARS IDs": {
			yamlContent: `
- id: "EARS-001"
  pattern: "ubiquitous"
  text: "System does X"
  test_type: "invariant"
- id: "EARS-001"
  pattern: "ubiquitous"
  text: "System does Y"
  test_type: "invariant"
`,
			wantValid:  false,
			wantErrors: []string{"duplicate EARS ID"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var node yaml.Node
			if err := yaml.Unmarshal([]byte(tt.yamlContent), &node); err != nil {
				t.Fatalf("failed to parse test YAML: %v", err)
			}

			var earsNode *yaml.Node
			if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
				earsNode = node.Content[0]
			}

			result := &ValidationResult{Valid: true}
			ValidateEarsRequirements(earsNode, result)

			if result.Valid != tt.wantValid {
				t.Errorf("ValidateEarsRequirements() valid = %v, want %v", result.Valid, tt.wantValid)
				for _, err := range result.Errors {
					t.Logf("  error: %s", err.Message)
				}
			}

			for _, wantErr := range tt.wantErrors {
				found := false
				for _, err := range result.Errors {
					if strings.Contains(err.Message, wantErr) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected error containing %q, got errors: %v", wantErr, errorMessages(result.Errors))
				}
			}
		})
	}
}

func TestValidateCrossSectionIDs(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		earsYAML  string
		reqsYAML  string
		wantValid bool
		wantError string
	}{
		"no conflict": {
			earsYAML: `
- id: "EARS-001"
  pattern: "ubiquitous"
  text: "System does X"
  test_type: "invariant"
`,
			reqsYAML: `
functional:
  - id: "FR-001"
    description: "Must do X"
`,
			wantValid: true,
		},
		"ID conflict between EARS and FR": {
			earsYAML: `
- id: "FR-001"
  pattern: "ubiquitous"
  text: "System does X"
  test_type: "invariant"
`,
			reqsYAML: `
functional:
  - id: "FR-001"
    description: "Must do X"
`,
			wantValid: false,
			wantError: "conflicts with functional requirement ID",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var earsDoc, reqsDoc yaml.Node
			if err := yaml.Unmarshal([]byte(tt.earsYAML), &earsDoc); err != nil {
				t.Fatalf("failed to parse ears YAML: %v", err)
			}
			if err := yaml.Unmarshal([]byte(tt.reqsYAML), &reqsDoc); err != nil {
				t.Fatalf("failed to parse reqs YAML: %v", err)
			}

			var earsNode, reqsNode *yaml.Node
			if earsDoc.Kind == yaml.DocumentNode && len(earsDoc.Content) > 0 {
				earsNode = earsDoc.Content[0]
			}
			if reqsDoc.Kind == yaml.DocumentNode && len(reqsDoc.Content) > 0 {
				reqsNode = reqsDoc.Content[0]
			}

			result := &ValidationResult{Valid: true}
			ValidateCrossSectionIDs(earsNode, reqsNode, result)

			if result.Valid != tt.wantValid {
				t.Errorf("ValidateCrossSectionIDs() valid = %v, want %v", result.Valid, tt.wantValid)
			}

			if tt.wantError != "" {
				found := false
				for _, err := range result.Errors {
					if strings.Contains(err.Message, tt.wantError) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected error containing %q", tt.wantError)
				}
			}
		})
	}
}

func TestPatternTemplates(t *testing.T) {
	t.Parallel()

	expectedTemplates := map[EarsPattern]string{
		PatternUbiquitous:  "The [system] shall [action]",
		PatternEventDriven: "When [trigger], the [system] shall [action]",
		PatternStateDriven: "While [state], the [system] shall [action]",
		PatternUnwanted:    "If [condition], then the [system] shall [action]",
		PatternOptional:    "Where [feature], the [system] shall [action]",
	}

	for pattern, expected := range expectedTemplates {
		t.Run(string(pattern), func(t *testing.T) {
			t.Parallel()
			if got := PatternTemplates[pattern]; got != expected {
				t.Errorf("PatternTemplates[%s] = %q, want %q", pattern, got, expected)
			}
		})
	}
}

func TestPatternRequiredFields(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		pattern EarsPattern
		want    []string
	}{
		"ubiquitous has no required fields": {
			pattern: PatternUbiquitous,
			want:    []string{},
		},
		"event-driven requires trigger and expected": {
			pattern: PatternEventDriven,
			want:    []string{"trigger", "expected"},
		},
		"state-driven requires state": {
			pattern: PatternStateDriven,
			want:    []string{"state"},
		},
		"unwanted requires condition": {
			pattern: PatternUnwanted,
			want:    []string{"condition"},
		},
		"optional requires feature": {
			pattern: PatternOptional,
			want:    []string{"feature"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := PatternRequiredFields[tt.pattern]
			if len(got) != len(tt.want) {
				t.Errorf("PatternRequiredFields[%s] len = %d, want %d", tt.pattern, len(got), len(tt.want))
				return
			}
			for i, field := range tt.want {
				if got[i] != field {
					t.Errorf("PatternRequiredFields[%s][%d] = %q, want %q", tt.pattern, i, got[i], field)
				}
			}
		})
	}
}

func errorMessages(errors []*ValidationError) []string {
	msgs := make([]string, len(errors))
	for i, err := range errors {
		msgs[i] = err.Message
	}
	return msgs
}
