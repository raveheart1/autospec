package dag

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ariel-frischer/autospec/internal/cliagent"
)

// ConflictContext contains all information needed for conflict resolution.
type ConflictContext struct {
	// FilePath is the relative path to the conflicted file.
	FilePath string
	// ConflictDiff contains the conflict markers and content from the file.
	ConflictDiff string
	// SpecID is the identifier of the spec being merged.
	SpecID string
	// SpecName is the human-readable name from dag.yaml.
	SpecName string
	// SpecDescription is the full description from dag.yaml.
	SpecDescription string
	// SourceBranch is the branch being merged from.
	SourceBranch string
	// TargetBranch is the branch being merged into.
	TargetBranch string
}

// ConflictResolver handles merge conflict resolution.
type ConflictResolver struct {
	repoRoot string
	agent    cliagent.Agent
	stdout   io.Writer
}

// NewConflictResolver creates a new ConflictResolver.
func NewConflictResolver(repoRoot string, agent cliagent.Agent, stdout io.Writer) *ConflictResolver {
	if stdout == nil {
		stdout = os.Stdout
	}
	return &ConflictResolver{
		repoRoot: repoRoot,
		agent:    agent,
		stdout:   stdout,
	}
}

// BuildConflictContext creates a ConflictContext from a conflicted file.
func (cr *ConflictResolver) BuildConflictContext(
	filePath string,
	specID string,
	dag *DAGConfig,
	sourceBranch, targetBranch string,
) (*ConflictContext, error) {
	conflictDiff, err := extractConflictMarkers(filepath.Join(cr.repoRoot, filePath))
	if err != nil {
		return nil, fmt.Errorf("extracting conflict markers from %s: %w", filePath, err)
	}

	specName, specDesc := lookupSpecInfo(dag, specID)

	return &ConflictContext{
		FilePath:        filePath,
		ConflictDiff:    conflictDiff,
		SpecID:          specID,
		SpecName:        specName,
		SpecDescription: specDesc,
		SourceBranch:    sourceBranch,
		TargetBranch:    targetBranch,
	}, nil
}

// extractConflictMarkers reads a file and extracts sections with conflict markers.
func extractConflictMarkers(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("opening file: %w", err)
	}
	defer file.Close()

	return parseConflictMarkers(file)
}

// parseConflictMarkers extracts conflict blocks from a reader.
func parseConflictMarkers(r io.Reader) (string, error) {
	var result strings.Builder
	scanner := bufio.NewScanner(r)
	inConflict := false
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		if strings.HasPrefix(line, "<<<<<<<") {
			inConflict = true
			if result.Len() > 0 {
				result.WriteString("\n---\n\n")
			}
			fmt.Fprintf(&result, "Line %d:\n", lineNum)
		}

		if inConflict {
			result.WriteString(line)
			result.WriteString("\n")
		}

		if strings.HasPrefix(line, ">>>>>>>") {
			inConflict = false
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("scanning file: %w", err)
	}

	return result.String(), nil
}

// lookupSpecInfo finds the name and description for a spec in the DAG.
func lookupSpecInfo(dag *DAGConfig, specID string) (name, description string) {
	if dag == nil {
		return specID, ""
	}

	for _, layer := range dag.Layers {
		for _, feature := range layer.Features {
			if feature.ID == specID {
				if feature.Description != "" {
					return feature.ID, feature.Description
				}
				return feature.ID, ""
			}
		}
	}

	return specID, ""
}

// ResolveWithAgent attempts to resolve conflicts using an AI agent.
func (cr *ConflictResolver) ResolveWithAgent(
	ctx context.Context,
	conflicts []*ConflictContext,
) error {
	if cr.agent == nil {
		return fmt.Errorf("no agent configured for conflict resolution")
	}

	for _, conflict := range conflicts {
		prompt := buildAgentPrompt(conflict)

		result, err := cr.agent.Execute(ctx, prompt, cliagent.ExecOptions{
			Autonomous: true,
			WorkDir:    cr.repoRoot,
		})

		if err != nil {
			return fmt.Errorf("agent execution failed for %s: %w", conflict.FilePath, err)
		}

		if result.ExitCode != 0 {
			return fmt.Errorf("agent failed to resolve %s: exit code %d", conflict.FilePath, result.ExitCode)
		}

		if err := verifyConflictResolved(cr.repoRoot, conflict.FilePath); err != nil {
			return fmt.Errorf("conflict not fully resolved in %s: %w", conflict.FilePath, err)
		}

		fmt.Fprintf(cr.stdout, "✓ Agent resolved conflict in %s\n", conflict.FilePath)
	}

	return nil
}

// buildAgentPrompt creates the prompt for the agent to resolve a conflict.
func buildAgentPrompt(conflict *ConflictContext) string {
	var sb strings.Builder

	sb.WriteString("Resolve the following git merge conflict. ")
	sb.WriteString("Edit the file to remove ALL conflict markers ")
	sb.WriteString("(<<<<<<< ======= >>>>>>>) and produce correct code.\n\n")

	sb.WriteString("## Context\n\n")
	fmt.Fprintf(&sb, "- **Spec**: %s\n", conflict.SpecID)
	if conflict.SpecDescription != "" {
		fmt.Fprintf(&sb, "- **Description**: %s\n", conflict.SpecDescription)
	}
	fmt.Fprintf(&sb, "- **Source branch**: %s (being merged)\n", conflict.SourceBranch)
	fmt.Fprintf(&sb, "- **Target branch**: %s (merge destination)\n", conflict.TargetBranch)
	fmt.Fprintf(&sb, "- **File**: %s\n\n", conflict.FilePath)

	sb.WriteString("## Conflict\n\n```\n")
	sb.WriteString(conflict.ConflictDiff)
	sb.WriteString("```\n\n")

	sb.WriteString("## Instructions\n\n")
	sb.WriteString("1. Read the conflicted file\n")
	sb.WriteString("2. Understand both changes and their intent\n")
	sb.WriteString("3. Edit the file to merge both changes correctly\n")
	sb.WriteString("4. Remove ALL conflict markers\n")
	sb.WriteString("5. Stage the resolved file with: git add ")
	sb.WriteString(conflict.FilePath)
	sb.WriteString("\n")

	return sb.String()
}

// verifyConflictResolved checks that a file no longer contains conflict markers.
func verifyConflictResolved(repoRoot, filePath string) error {
	fullPath := filepath.Join(repoRoot, filePath)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}

	if hasConflictMarkers(string(content)) {
		return fmt.Errorf("file still contains conflict markers")
	}

	return nil
}

// hasConflictMarkers checks if content contains git conflict markers.
func hasConflictMarkers(content string) bool {
	return strings.Contains(content, "<<<<<<<") ||
		strings.Contains(content, "=======") ||
		strings.Contains(content, ">>>>>>>")
}

// OutputManualContext prints a copy-pastable context block for manual resolution.
func (cr *ConflictResolver) OutputManualContext(conflicts []*ConflictContext) {
	fmt.Fprintln(cr.stdout, "\n"+strings.Repeat("=", 80))
	fmt.Fprintln(cr.stdout, "MERGE CONFLICT - Manual Resolution Required")
	fmt.Fprintln(cr.stdout, strings.Repeat("=", 80))

	for i, conflict := range conflicts {
		if i > 0 {
			fmt.Fprintln(cr.stdout, "\n"+strings.Repeat("-", 80)+"\n")
		}
		cr.outputSingleConflict(conflict)
	}

	fmt.Fprintln(cr.stdout, "\n"+strings.Repeat("=", 80))
	fmt.Fprintln(cr.stdout, "Copy the above context to your AI assistant for help resolving.")
	fmt.Fprintln(cr.stdout, "After resolving, run: autospec dag merge --continue")
	fmt.Fprintln(cr.stdout, strings.Repeat("=", 80)+"\n")
}

// outputSingleConflict outputs the context for a single conflict.
func (cr *ConflictResolver) outputSingleConflict(conflict *ConflictContext) {
	fmt.Fprintf(cr.stdout, "## File: %s\n\n", conflict.FilePath)

	fmt.Fprintln(cr.stdout, "### Context")
	fmt.Fprintf(cr.stdout, "- Spec ID: %s\n", conflict.SpecID)
	if conflict.SpecDescription != "" {
		fmt.Fprintf(cr.stdout, "- Description: %s\n", conflict.SpecDescription)
	}
	fmt.Fprintf(cr.stdout, "- Source Branch: %s (being merged)\n", conflict.SourceBranch)
	fmt.Fprintf(cr.stdout, "- Target Branch: %s (merge destination)\n", conflict.TargetBranch)

	fmt.Fprintln(cr.stdout, "\n### Conflict Markers")
	fmt.Fprintln(cr.stdout, "```")
	fmt.Fprint(cr.stdout, conflict.ConflictDiff)
	fmt.Fprintln(cr.stdout, "```")

	fmt.Fprintln(cr.stdout, "\n### Resolution Steps")
	fmt.Fprintln(cr.stdout, "1. Open the file and locate the conflict markers")
	fmt.Fprintln(cr.stdout, "2. Understand the intent of both changes")
	fmt.Fprintln(cr.stdout, "3. Edit to produce the correct merged result")
	fmt.Fprintln(cr.stdout, "4. Remove ALL conflict markers (<<<<<<< ======= >>>>>>>)")
	fmt.Fprintf(cr.stdout, "5. Stage the file: git add %s\n", conflict.FilePath)
}

// DetectConflicts returns a list of files with merge conflicts in the repository.
// This is a convenience wrapper around DetectConflictedFiles for consistency.
func DetectConflicts(repoRoot string) []string {
	return DetectConflictedFiles(repoRoot)
}

// BuildAllConflictContexts builds ConflictContext for each conflicted file.
func (cr *ConflictResolver) BuildAllConflictContexts(
	conflictedFiles []string,
	specID string,
	dag *DAGConfig,
	sourceBranch, targetBranch string,
) ([]*ConflictContext, error) {
	contexts := make([]*ConflictContext, 0, len(conflictedFiles))

	for _, filePath := range conflictedFiles {
		ctx, err := cr.BuildConflictContext(filePath, specID, dag, sourceBranch, targetBranch)
		if err != nil {
			return nil, fmt.Errorf("building context for %s: %w", filePath, err)
		}
		contexts = append(contexts, ctx)
	}

	return contexts, nil
}

// AbortMerge aborts an in-progress git merge.
func AbortMerge(repoRoot string) error {
	cmd := exec.Command("git", "merge", "--abort")
	cmd.Dir = repoRoot
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("aborting merge: %w", err)
	}
	return nil
}

// CompleteMerge stages all changes and completes the merge.
func CompleteMerge(repoRoot string) error {
	addCmd := exec.Command("git", "add", "-A")
	addCmd.Dir = repoRoot
	if err := addCmd.Run(); err != nil {
		return fmt.Errorf("staging changes: %w", err)
	}

	commitCmd := exec.Command("git", "commit", "--no-edit")
	commitCmd.Dir = repoRoot
	if err := commitCmd.Run(); err != nil {
		return fmt.Errorf("completing merge commit: %w", err)
	}

	return nil
}
