package workflow

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInjectableInstruction(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		inst     InjectableInstruction
		wantName string
		wantHint string
	}{
		"all fields populated": {
			inst: InjectableInstruction{
				Name:        "AutoCommit",
				DisplayHint: "post-work git commit with conventional format",
				Content:     "Do the commit",
			},
			wantName: "AutoCommit",
			wantHint: "post-work git commit with conventional format",
		},
		"name only with content": {
			inst: InjectableInstruction{
				Name:    "TestInstruction",
				Content: "Test content",
			},
			wantName: "TestInstruction",
			wantHint: "",
		},
		"empty display hint is valid": {
			inst: InjectableInstruction{
				Name:        "MinimalInst",
				DisplayHint: "",
				Content:     "Content here",
			},
			wantName: "MinimalInst",
			wantHint: "",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.wantName, tc.inst.Name)
			assert.Equal(t, tc.wantHint, tc.inst.DisplayHint)
			assert.NotEmpty(t, tc.inst.Content)
		})
	}
}

func TestInjectInstructions(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		command      string
		instructions []InjectableInstruction
		wantContains []string
		wantExact    string
	}{
		"empty instruction list returns original": {
			command:      "original command",
			instructions: []InjectableInstruction{},
			wantExact:    "original command",
		},
		"nil instruction list returns original": {
			command:      "original command",
			instructions: nil,
			wantExact:    "original command",
		},
		"single instruction with markers": {
			command: "base command",
			instructions: []InjectableInstruction{
				{Name: "AutoCommit", Content: "Commit instructions"},
			},
			wantContains: []string{
				"base command",
				"<!-- AUTOSPEC_INJECT:AutoCommit -->",
				"Commit instructions",
				"<!-- /AUTOSPEC_INJECT:AutoCommit -->",
			},
		},
		"multiple instructions": {
			command: "multi command",
			instructions: []InjectableInstruction{
				{Name: "First", Content: "First content"},
				{Name: "Second", Content: "Second content"},
			},
			wantContains: []string{
				"multi command",
				"<!-- AUTOSPEC_INJECT:First -->",
				"First content",
				"<!-- /AUTOSPEC_INJECT:First -->",
				"<!-- AUTOSPEC_INJECT:Second -->",
				"Second content",
				"<!-- /AUTOSPEC_INJECT:Second -->",
			},
		},
		"skips instruction with empty name": {
			command: "cmd",
			instructions: []InjectableInstruction{
				{Name: "", Content: "Should be skipped"},
				{Name: "Valid", Content: "Should be included"},
			},
			wantContains: []string{
				"<!-- AUTOSPEC_INJECT:Valid -->",
				"Should be included",
			},
		},
		"skips instruction with empty content": {
			command: "cmd",
			instructions: []InjectableInstruction{
				{Name: "EmptyContent", Content: ""},
				{Name: "Valid", Content: "Valid content"},
			},
			wantContains: []string{
				"<!-- AUTOSPEC_INJECT:Valid -->",
				"Valid content",
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result := InjectInstructions(tc.command, tc.instructions)

			if tc.wantExact != "" {
				assert.Equal(t, tc.wantExact, result)
				return
			}

			for _, want := range tc.wantContains {
				assert.Contains(t, result, want,
					"result should contain: %s", want)
			}
		})
	}
}

func TestInjectInstructionsMarkerFormat(t *testing.T) {
	t.Parallel()

	command := "test"
	instructions := []InjectableInstruction{
		{Name: "TestMarker", Content: "Test content"},
	}

	result := InjectInstructions(command, instructions)

	// Verify marker structure is correct for parsing
	require.Contains(t, result, "<!-- AUTOSPEC_INJECT:TestMarker -->")
	require.Contains(t, result, "<!-- /AUTOSPEC_INJECT:TestMarker -->")

	// Verify content is between markers
	startIdx := strings.Index(result, "<!-- AUTOSPEC_INJECT:TestMarker -->")
	endIdx := strings.Index(result, "<!-- /AUTOSPEC_INJECT:TestMarker -->")
	contentIdx := strings.Index(result, "Test content")

	assert.Less(t, startIdx, contentIdx, "start marker should come before content")
	assert.Less(t, contentIdx, endIdx, "content should come before end marker")
}

func TestInjectInstructionsPreservesOriginal(t *testing.T) {
	t.Parallel()

	// Original command with complex content
	original := `Some complex command
with multiple lines
and special chars: $VAR && || > <`

	instructions := []InjectableInstruction{
		{Name: "Append", Content: "Appended content"},
	}

	result := InjectInstructions(original, instructions)

	// Original should be fully preserved at the start
	assert.True(t, strings.HasPrefix(result, original),
		"result should start with original command")
}

func TestBuildAutoCommitInstructions(t *testing.T) {
	t.Parallel()

	inst := BuildAutoCommitInstructions()

	// Verify InjectableInstruction fields
	assert.Equal(t, "AutoCommit", inst.Name)
	assert.Equal(t, "post-work git commit with conventional format", inst.DisplayHint)
	assert.NotEmpty(t, inst.Content, "content should not be empty")

	t.Run("contains essential git workflow", func(t *testing.T) {
		t.Parallel()

		content := inst.Content
		// Core workflow steps
		assert.Contains(t, content, "git status",
			"instructions should include git status")
		assert.Contains(t, content, ".gitignore",
			"instructions should mention .gitignore")
		assert.Contains(t, content, "git add",
			"instructions should include git add command")
		assert.Contains(t, content, "git commit",
			"instructions should include git commit command")
	})

	t.Run("contains conventional commit format", func(t *testing.T) {
		t.Parallel()

		content := inst.Content
		// Should include commit format and types
		assert.Contains(t, content, "type(scope): description",
			"instructions should include commit format template")
		assert.Contains(t, content, "feat",
			"instructions should mention feat commit type")
		assert.Contains(t, content, "fix",
			"instructions should mention fix commit type")
	})

	t.Run("is agent-agnostic", func(t *testing.T) {
		t.Parallel()

		// Should NOT contain agent-specific references
		agentSpecificTerms := []string{
			"Claude",
			"Gemini",
			"GPT",
			"OpenAI",
			"Anthropic",
			"Google AI",
			"Copilot",
		}
		contentLower := strings.ToLower(inst.Content)
		for _, term := range agentSpecificTerms {
			assert.NotContains(t, contentLower, strings.ToLower(term),
				"instructions should not contain agent-specific term: %s", term)
		}
	})

	t.Run("handles edge cases", func(t *testing.T) {
		t.Parallel()

		content := inst.Content
		// Should mention what to do when there are no changes
		assert.Contains(t, content, "no changes",
			"instructions should handle no-changes case")

		// Should mention detached HEAD state
		assert.Contains(t, content, "detached HEAD",
			"instructions should handle detached HEAD case")
	})
}

func TestBuildAutoCommitInstructionsIdempotent(t *testing.T) {
	t.Parallel()

	// Calling the function multiple times should return the same result
	first := BuildAutoCommitInstructions()
	second := BuildAutoCommitInstructions()

	assert.Equal(t, first, second,
		"BuildAutoCommitInstructions should be idempotent")
}

func TestMinimalAutoCommitInstructions(t *testing.T) {
	t.Parallel()

	inst := BuildAutoCommitInstructions()
	content := inst.Content
	lines := strings.Split(strings.TrimSpace(content), "\n")

	tests := map[string]struct {
		check       func() bool
		description string
	}{
		"instructions are minimal (≤20 lines)": {
			check: func() bool {
				return len(lines) <= 20
			},
			description: "auto-commit instructions should be ≤20 lines",
		},
		"git status appears before git add": {
			check: func() bool {
				statusIdx := strings.Index(content, "git status")
				addIdx := strings.Index(content, "git add")
				return statusIdx >= 0 && addIdx >= 0 && statusIdx < addIdx
			},
			description: "git status should appear before git add",
		},
		"returns valid InjectableInstruction": {
			check: func() bool {
				return inst.Name == "AutoCommit" &&
					inst.DisplayHint != "" &&
					inst.Content != ""
			},
			description: "BuildAutoCommitInstructions should return valid InjectableInstruction",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.True(t, tc.check(), tc.description)
		})
	}
}

func TestAutoCommitInstructionsStructure(t *testing.T) {
	t.Parallel()

	inst := BuildAutoCommitInstructions()
	content := inst.Content

	tests := map[string]struct {
		check       string
		description string
	}{
		"has numbered steps": {
			check:       "1.",
			description: "numbered workflow steps",
		},
		"mentions conventional format": {
			check:       "conventional",
			description: "conventional commit format",
		},
		"mentions skip conditions": {
			check:       "Skip commit",
			description: "skip conditions for edge cases",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Contains(t, content, tc.check,
				"instructions should contain %s for %s", tc.check, tc.description)
		})
	}
}
