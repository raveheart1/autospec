// Package validation_test tests core YAML artifact validation and error reporting.
// Related: internal/validation/artifact.go
// Tags: validation, artifact, yaml, error, node, parsing, schema
package validation

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestValidationError_Error(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		err          *ValidationError
		wantContains []string
	}{
		"full error with all fields": {
			err: &ValidationError{
				Path:    "user_stories[0].id",
				Line:    10,
				Column:  5,
				Message: "missing required field",
			},
			wantContains: []string{"line 10:5", "user_stories[0].id", "missing required field"},
		},
		"error with line only": {
			err: &ValidationError{
				Line:    15,
				Message: "syntax error",
			},
			wantContains: []string{"line 15", "syntax error"},
		},
		"error with path only": {
			err: &ValidationError{
				Path:    "feature.name",
				Message: "field is empty",
			},
			wantContains: []string{"feature.name", "field is empty"},
		},
		"error with message only": {
			err: &ValidationError{
				Message: "invalid yaml",
			},
			wantContains: []string{"invalid yaml"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			errStr := tc.err.Error()

			for _, want := range tc.wantContains {
				if !strings.Contains(errStr, want) {
					t.Errorf("Error() = %q, want to contain %q", errStr, want)
				}
			}
		})
	}
}

func TestValidationError_FormatFull(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		err          *ValidationError
		wantContains []string
	}{
		"full error with all fields": {
			err: &ValidationError{
				Path:     "user_stories[0].id",
				Line:     10,
				Column:   5,
				Message:  "missing required field",
				Expected: "string",
				Actual:   "null",
				Hint:     "Add the id field",
			},
			wantContains: []string{
				"Line 10",
				"Column 5",
				"Path: user_stories[0].id",
				"Error: missing required field",
				"Expected: string",
				"Got: null",
				"Hint: Add the id field",
			},
		},
		"error without expected/actual": {
			err: &ValidationError{
				Path:    "feature",
				Line:    1,
				Message: "malformed yaml",
			},
			wantContains: []string{
				"Line 1",
				"Path: feature",
				"Error: malformed yaml",
			},
		},
		"error with line and column zero": {
			err: &ValidationError{
				Message: "parse error",
			},
			wantContains: []string{"Error: parse error"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			formatted := tc.err.FormatFull()

			for _, want := range tc.wantContains {
				if !strings.Contains(formatted, want) {
					t.Errorf("FormatFull() = %q, want to contain %q", formatted, want)
				}
			}
		})
	}
}

func TestValidationResult_HasErrors(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		result *ValidationResult
		want   bool
	}{
		"no errors": {
			result: &ValidationResult{
				Valid:  true,
				Errors: []*ValidationError{},
			},
			want: false,
		},
		"with errors": {
			result: &ValidationResult{
				Valid:  false,
				Errors: []*ValidationError{{Message: "error"}},
			},
			want: true,
		},
		"nil errors slice": {
			result: &ValidationResult{
				Valid:  true,
				Errors: nil,
			},
			want: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if got := tc.result.HasErrors(); got != tc.want {
				t.Errorf("HasErrors() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestBaseValidator_Type(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		validator *baseValidator
		want      ArtifactType
	}{
		"spec type": {
			validator: &baseValidator{artifactType: ArtifactTypeSpec},
			want:      ArtifactTypeSpec,
		},
		"plan type": {
			validator: &baseValidator{artifactType: ArtifactTypePlan},
			want:      ArtifactTypePlan,
		},
		"tasks type": {
			validator: &baseValidator{artifactType: ArtifactTypeTasks},
			want:      ArtifactTypeTasks,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if got := tc.validator.Type(); got != tc.want {
				t.Errorf("Type() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestNewArtifactValidator(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		artifactType ArtifactType
		wantErr      bool
	}{
		"spec validator": {
			artifactType: ArtifactTypeSpec,
			wantErr:      false,
		},
		"plan validator": {
			artifactType: ArtifactTypePlan,
			wantErr:      false,
		},
		"tasks validator": {
			artifactType: ArtifactTypeTasks,
			wantErr:      false,
		},
		"analysis validator": {
			artifactType: ArtifactTypeAnalysis,
			wantErr:      false,
		},
		"checklist validator": {
			artifactType: ArtifactTypeChecklist,
			wantErr:      false,
		},
		"constitution validator": {
			artifactType: ArtifactTypeConstitution,
			wantErr:      false,
		},
		"unknown type": {
			artifactType: ArtifactType("unknown"),
			wantErr:      true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			v, err := NewArtifactValidator(tc.artifactType)

			if tc.wantErr {
				if err == nil {
					t.Error("NewArtifactValidator() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("NewArtifactValidator() error = %v", err)
				}
				if v == nil {
					t.Error("NewArtifactValidator() returned nil validator")
				}
			}
		})
	}
}

func TestNodeKindToString(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		kind yaml.Kind
		want string
	}{
		"document node": {
			kind: yaml.DocumentNode,
			want: "document",
		},
		"sequence node": {
			kind: yaml.SequenceNode,
			want: "array",
		},
		"mapping node": {
			kind: yaml.MappingNode,
			want: "object",
		},
		"scalar node": {
			kind: yaml.ScalarNode,
			want: "scalar",
		},
		"alias node": {
			kind: yaml.AliasNode,
			want: "alias",
		},
		"unknown kind": {
			kind: yaml.Kind(255),
			want: "unknown",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if got := nodeKindToString(tc.kind); got != tc.want {
				t.Errorf("nodeKindToString(%v) = %q, want %q", tc.kind, got, tc.want)
			}
		})
	}
}

func TestGetNodeLine(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		node *yaml.Node
		want int
	}{
		"nil node": {
			node: nil,
			want: 0,
		},
		"node with line": {
			node: &yaml.Node{Line: 10},
			want: 10,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if got := getNodeLine(tc.node); got != tc.want {
				t.Errorf("getNodeLine() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestGetNodeColumn(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		node *yaml.Node
		want int
	}{
		"nil node": {
			node: nil,
			want: 0,
		},
		"node with column": {
			node: &yaml.Node{Column: 5},
			want: 5,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if got := getNodeColumn(tc.node); got != tc.want {
				t.Errorf("getNodeColumn() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestFindNode(t *testing.T) {
	t.Parallel()

	// Build test YAML structure
	mappingNode := &yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "key1"},
			{Kind: yaml.ScalarNode, Value: "value1"},
			{Kind: yaml.ScalarNode, Value: "key2"},
			{Kind: yaml.ScalarNode, Value: "value2"},
		},
	}

	documentNode := &yaml.Node{
		Kind:    yaml.DocumentNode,
		Content: []*yaml.Node{mappingNode},
	}

	tests := map[string]struct {
		root      *yaml.Node
		key       string
		wantFound bool
		wantValue string
	}{
		"find existing key in mapping": {
			root:      mappingNode,
			key:       "key1",
			wantFound: true,
			wantValue: "value1",
		},
		"find second key in mapping": {
			root:      mappingNode,
			key:       "key2",
			wantFound: true,
			wantValue: "value2",
		},
		"key not found": {
			root:      mappingNode,
			key:       "nonexistent",
			wantFound: false,
		},
		"nil root": {
			root:      nil,
			key:       "key",
			wantFound: false,
		},
		"document node wrapping mapping": {
			root:      documentNode,
			key:       "key1",
			wantFound: true,
			wantValue: "value1",
		},
		"scalar node (not mapping)": {
			root:      &yaml.Node{Kind: yaml.ScalarNode, Value: "test"},
			key:       "key",
			wantFound: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result := findNode(tc.root, tc.key)

			if tc.wantFound {
				if result == nil {
					t.Errorf("findNode() = nil, want node with value %q", tc.wantValue)
				} else if result.Value != tc.wantValue {
					t.Errorf("findNode().Value = %q, want %q", result.Value, tc.wantValue)
				}
			} else {
				if result != nil {
					t.Errorf("findNode() = %v, want nil", result)
				}
			}
		})
	}
}

func TestGetRootMapping(t *testing.T) {
	t.Parallel()

	mappingNode := &yaml.Node{Kind: yaml.MappingNode}
	documentNode := &yaml.Node{
		Kind:    yaml.DocumentNode,
		Content: []*yaml.Node{mappingNode},
	}

	tests := map[string]struct {
		root     *yaml.Node
		wantNil  bool
		wantKind yaml.Kind
	}{
		"nil node": {
			root:    nil,
			wantNil: true,
		},
		"mapping node": {
			root:     mappingNode,
			wantNil:  false,
			wantKind: yaml.MappingNode,
		},
		"document node wrapping mapping": {
			root:     documentNode,
			wantNil:  false,
			wantKind: yaml.MappingNode,
		},
		"scalar node": {
			root:    &yaml.Node{Kind: yaml.ScalarNode},
			wantNil: true,
		},
		"sequence node": {
			root:    &yaml.Node{Kind: yaml.SequenceNode},
			wantNil: true,
		},
		"empty document node": {
			root:    &yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{}},
			wantNil: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result := getRootMapping(tc.root)

			if tc.wantNil {
				if result != nil {
					t.Errorf("getRootMapping() = %v, want nil", result)
				}
			} else {
				if result == nil {
					t.Error("getRootMapping() = nil, want non-nil")
				} else if result.Kind != tc.wantKind {
					t.Errorf("getRootMapping().Kind = %v, want %v", result.Kind, tc.wantKind)
				}
			}
		})
	}
}

func TestParseYAMLReader(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		yaml    string
		wantErr bool
	}{
		"valid yaml": {
			yaml:    "key: value",
			wantErr: false,
		},
		"empty string": {
			yaml:    "",
			wantErr: true,
		},
		"only comments": {
			yaml:    "# comment only",
			wantErr: true,
		},
		"invalid yaml": {
			yaml:    "invalid: - yaml: -",
			wantErr: true,
		},
		"complex yaml": {
			yaml: `
feature:
  name: test
  status: Draft
user_stories:
  - id: US-001
    title: Test Story
`,
			wantErr: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			reader := strings.NewReader(tc.yaml)
			_, err := parseYAMLReader(reader)

			if tc.wantErr && err == nil {
				t.Error("parseYAMLReader() expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("parseYAMLReader() error = %v", err)
			}
		})
	}
}
