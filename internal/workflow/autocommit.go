// Package workflow provides auto-commit instruction generation for agent prompt injection.
package workflow

import (
	"fmt"
	"strings"
)

// Marker constants for injectable instruction blocks.
// These markers enable reliable detection and extraction of injected instructions.
const (
	// InjectMarkerPrefix is the opening marker for an injectable instruction block.
	// Format: <!-- AUTOSPEC_INJECT:Name -->
	InjectMarkerPrefix = "<!-- AUTOSPEC_INJECT:"

	// InjectMarkerSuffix closes the opening marker.
	InjectMarkerSuffix = " -->"

	// InjectMarkerEndPrefix is the closing marker prefix for an injectable instruction block.
	// Format: <!-- /AUTOSPEC_INJECT:Name -->
	InjectMarkerEndPrefix = "<!-- /AUTOSPEC_INJECT:"
)

// InjectableInstruction represents a system instruction that can be injected into
// agent prompts with metadata for display. This abstraction enables compact output
// display (e.g., [+AutoCommit]) while preserving full instruction content for agents.
type InjectableInstruction struct {
	// Name is a short identifier for the instruction type (e.g., "AutoCommit").
	// Must be non-empty and alphanumeric with underscores.
	Name string

	// DisplayHint is a brief description for verbose display mode.
	// Optional, max 80 characters. Shown as [+Name: DisplayHint] in verbose mode.
	DisplayHint string

	// Content is the full instruction text for agent consumption.
	// Must be non-empty.
	Content string
}

// InjectInstructions appends marked instruction blocks to a command string.
// Each instruction is wrapped with markers for reliable detection and extraction:
//
//	<!-- AUTOSPEC_INJECT:Name -->
//	content
//	<!-- /AUTOSPEC_INJECT:Name -->
//
// Returns the original command with all instruction blocks appended.
func InjectInstructions(command string, instructions []InjectableInstruction) string {
	if len(instructions) == 0 {
		return command
	}

	var builder strings.Builder
	builder.WriteString(command)

	for _, inst := range instructions {
		if inst.Name == "" || inst.Content == "" {
			continue
		}
		builder.WriteString("\n\n")
		builder.WriteString(formatInstructionBlock(inst))
	}

	return builder.String()
}

// formatInstructionBlock wraps an instruction's content with start/end markers.
// The opening marker includes the DisplayHint (if present) for verbose display:
// <!-- AUTOSPEC_INJECT:Name:DisplayHint -->
func formatInstructionBlock(inst InjectableInstruction) string {
	// Include DisplayHint in marker if present: <!-- AUTOSPEC_INJECT:Name:Hint -->
	openMarker := fmt.Sprintf("%s%s", InjectMarkerPrefix, inst.Name)
	if inst.DisplayHint != "" {
		openMarker += ":" + inst.DisplayHint
	}
	openMarker += InjectMarkerSuffix

	return fmt.Sprintf("%s\n%s\n%s%s%s",
		openMarker,
		inst.Content,
		InjectMarkerEndPrefix, inst.Name, InjectMarkerSuffix,
	)
}

// autoCommitInstructions contains minimal instructions for agents to create
// clean git commits. Agents already understand git conventions, so this focuses
// on the essential workflow steps without redundant explanations.
const autoCommitInstructions = `## Auto-Commit

After completing work, commit your changes:

1. Run ` + "`git status --short`" + ` to see what changed
2. Update .gitignore if needed (add common patterns for untracked build/deps folders)
3. Stage changes: ` + "`git add -A`" + `
4. Commit with conventional format: ` + "`git commit -m \"type(scope): description\"`" + `

Skip commit if: no changes, detached HEAD, or only ignored files.
`

// BuildAutoCommitInstructions returns an InjectableInstruction for auto-commit.
// The instruction contains minimal guidance since agents already understand
// git conventions. This is used with InjectInstructions for proper marker wrapping.
func BuildAutoCommitInstructions() InjectableInstruction {
	return InjectableInstruction{
		Name:        "AutoCommit",
		DisplayHint: "post-work git commit with conventional format",
		Content:     autoCommitInstructions,
	}
}
