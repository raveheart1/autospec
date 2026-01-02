// Package output provides terminal output formatting utilities for autospec CLI.
// This package is designed to have minimal dependencies to avoid import cycles.
package output

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/fatih/color"
	"golang.org/x/term"
)

// GetTerminalWidth returns the terminal width, defaulting to 80 if unavailable.
func GetTerminalWidth() int {
	if width, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil && width > 0 {
		return width
	}
	return 80
}

// PrintAgentOutputEnd prints a colored separator after agent output ends.
// Uses dim magenta styling to create visual distinction from agent output.
func PrintAgentOutputEnd(out io.Writer) {
	termWidth := GetTerminalWidth()
	magenta := color.New(color.FgMagenta, color.Faint).SprintFunc()

	label := " autospec "
	lineLen := (termWidth - len(label)) / 2
	if lineLen < 3 {
		lineLen = 3
	}

	line := strings.Repeat("─", lineLen)
	fmt.Fprintf(out, "\n%s%s%s\n", magenta(line), magenta(label), magenta(line))
}

// PrintStageHeader prints a colored stage header (e.g., "[Stage 1/4] Specify...").
// Uses cyan for the stage indicator and white for the stage name.
func PrintStageHeader(out io.Writer, stageNum, totalStages int, stageName string) {
	cyan := color.New(color.FgCyan, color.Bold).SprintFunc()
	white := color.New(color.FgWhite, color.Bold).SprintFunc()
	fmt.Fprintf(out, "%s %s\n", cyan(fmt.Sprintf("[Stage %d/%d]", stageNum, totalStages)), white(stageName+"..."))
}

// PrintStageSuccess prints a colored success message for completed artifacts.
// Uses green checkmark and cyan for the artifact path.
func PrintStageSuccess(out io.Writer, message string) {
	green := color.New(color.FgGreen, color.Bold).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()
	fmt.Fprintf(out, "%s %s\n\n", green("✓"), cyan(message))
}

// PrintExecutingCommand prints the command being executed with colored styling.
// Uses magenta arrow and dim text for the command details.
func PrintExecutingCommand(out io.Writer, command string) {
	magenta := color.New(color.FgMagenta).SprintFunc()
	dim := color.New(color.Faint).SprintFunc()
	fmt.Fprintf(out, "\n%s %s\n\n", magenta("→ Executing:"), dim(command))
}
