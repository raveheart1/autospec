package dag

import "testing"

func TestSlugify(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		input    string
		expected string
	}{
		"basic name with spaces": {
			input:    "GitStats CLI v1",
			expected: "gitstats-cli-v1",
		},
		"special characters and ampersand": {
			input:    "Feature: Auth & Sessions",
			expected: "feature-auth-sessions",
		},
		"extra whitespace": {
			input:    "  My  DAG  ",
			expected: "my-dag",
		},
		"empty string": {
			input:    "",
			expected: "",
		},
		"special characters only": {
			input:    "!@#$%^&*()",
			expected: "",
		},
		"truncation at 50 chars": {
			input:    "this is a very long dag name that exceeds the fifty character limit for slugs",
			expected: "this-is-a-very-long-dag-name-that-exceeds-the-fift",
		},
		"unicode characters": {
			input:    "日本語テスト",
			expected: "",
		},
		"mixed unicode and ascii": {
			input:    "Feature: 日本語 Test",
			expected: "feature-test",
		},
		"numbers preserved": {
			input:    "v1.2.3 Release",
			expected: "v1-2-3-release",
		},
		"already lowercase": {
			input:    "my-slug-name",
			expected: "my-slug-name",
		},
		"consecutive special chars": {
			input:    "foo---bar___baz",
			expected: "foo-bar-baz",
		},
		"leading special chars": {
			input:    "---foo",
			expected: "foo",
		},
		"trailing special chars": {
			input:    "foo---",
			expected: "foo",
		},
		"single word": {
			input:    "Feature",
			expected: "feature",
		},
		"truncation avoids trailing hyphen": {
			input:    "a-b-c-d-e-f-g-h-i-j-k-l-m-n-o-p-q-r-s-t-u-v-w-x-y-z",
			expected: "a-b-c-d-e-f-g-h-i-j-k-l-m-n-o-p-q-r-s-t-u-v-w-x-y",
		},
		"exactly 50 chars": {
			input:    "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwx",
			expected: "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwx",
		},
		"51 chars truncated": {
			input:    "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxy",
			expected: "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwx",
		},
		"whitespace only": {
			input:    "   ",
			expected: "",
		},
		"tabs and newlines": {
			input:    "foo\tbar\nbaz",
			expected: "foo-bar-baz",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := Slugify(tt.input)
			if got != tt.expected {
				t.Errorf("Slugify(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestMaxSlugLength(t *testing.T) {
	t.Parallel()

	// Verify the constant is set correctly
	if MaxSlugLength != 50 {
		t.Errorf("MaxSlugLength = %d, want 50", MaxSlugLength)
	}
}

func TestSlugifyDeterministic(t *testing.T) {
	t.Parallel()

	// NFR-006: Slug generation MUST be deterministic
	input := "GitStats CLI v1"
	expected := "gitstats-cli-v1"

	for i := range 100 {
		got := Slugify(input)
		if got != expected {
			t.Errorf("Slugify not deterministic: iteration %d got %q, want %q", i, got, expected)
		}
	}
}

func TestResolveDAGID(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		dag          *DAGMetadata
		workflowPath string
		expected     string
	}{
		"explicit ID takes priority": {
			dag:          &DAGMetadata{ID: "my-custom-id", Name: "GitStats CLI v1"},
			workflowPath: "workflows/v1.yaml",
			expected:     "my-custom-id",
		},
		"explicit ID is slugified": {
			dag:          &DAGMetadata{ID: "My Custom ID", Name: "GitStats CLI v1"},
			workflowPath: "workflows/v1.yaml",
			expected:     "my-custom-id",
		},
		"slugified name when no ID": {
			dag:          &DAGMetadata{Name: "GitStats CLI v1"},
			workflowPath: "workflows/v1.yaml",
			expected:     "gitstats-cli-v1",
		},
		"workflow filename fallback when no ID or name": {
			dag:          &DAGMetadata{},
			workflowPath: "workflows/my-workflow.yaml",
			expected:     "my-workflow",
		},
		"workflow filename fallback with nested path": {
			dag:          &DAGMetadata{},
			workflowPath: ".autospec/dags/gitstats-v1.yaml",
			expected:     "gitstats-v1",
		},
		"empty ID falls through to name": {
			dag:          &DAGMetadata{ID: "", Name: "My Feature"},
			workflowPath: "workflows/v1.yaml",
			expected:     "my-feature",
		},
		"ID with only special chars falls through to name": {
			dag:          &DAGMetadata{ID: "!@#$%", Name: "My Feature"},
			workflowPath: "workflows/v1.yaml",
			expected:     "my-feature",
		},
		"name with only special chars falls through to filename": {
			dag:          &DAGMetadata{Name: "!@#$%"},
			workflowPath: "workflows/v1.yaml",
			expected:     "v1",
		},
		"all empty falls through to filename": {
			dag:          &DAGMetadata{ID: "", Name: ""},
			workflowPath: "workflows/feature.yaml",
			expected:     "feature",
		},
		"complex workflow path extracts filename": {
			dag:          &DAGMetadata{},
			workflowPath: "/home/user/project/.autospec/dags/complex-feature.yaml",
			expected:     "complex-feature",
		},
		"yml extension handled": {
			dag:          &DAGMetadata{},
			workflowPath: "dags/my-dag.yml",
			expected:     "my-dag",
		},
		"priority order: ID > Name > filename": {
			dag:          &DAGMetadata{ID: "explicit", Name: "ignored"},
			workflowPath: "also-ignored.yaml",
			expected:     "explicit",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := ResolveDAGID(tt.dag, tt.workflowPath)
			if got != tt.expected {
				t.Errorf("ResolveDAGID() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestResolveDAGIDDeterministic(t *testing.T) {
	t.Parallel()

	dag := &DAGMetadata{Name: "GitStats CLI v1"}
	workflowPath := "workflows/v1.yaml"
	expected := "gitstats-cli-v1"

	for i := range 100 {
		got := ResolveDAGID(dag, workflowPath)
		if got != expected {
			t.Errorf("ResolveDAGID not deterministic: iteration %d got %q, want %q", i, got, expected)
		}
	}
}
