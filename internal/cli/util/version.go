package util

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/ariel-frischer/autospec/internal/cli/shared"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	// Version information - set via ldflags during build
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)

// IsDevBuild returns true if running a development build (not a release).
// Used to gate experimental features that aren't ready for production.
func IsDevBuild() bool {
	return Version == "dev"
}

var versionPlain bool

var versionCmd = &cobra.Command{
	Use:     "version",
	Aliases: []string{"v"},
	Short:   "Display version information (v)",
	Long:    "Display version, commit, build date, and Go version information for autospec",
	Example: `  # Show version info
  autospec version

  # Plain output (for scripts)
  autospec version --plain`,
	Run: func(cmd *cobra.Command, args []string) {
		if versionPlain {
			printPlainVersion()
		} else {
			printPrettyVersion()
		}
	},
}

func init() {
	versionCmd.GroupID = shared.GroupGettingStarted
	versionCmd.Flags().BoolVar(&versionPlain, "plain", false, "Plain output without formatting")
}

// printPlainVersion prints a simple version output for scripting
func printPlainVersion() {
	fmt.Printf("autospec %s\n", Version)
	fmt.Printf("commit: %s\n", Commit)
	fmt.Printf("built: %s\n", BuildDate)
	fmt.Printf("go: %s\n", runtime.Version())
	fmt.Printf("platform: %s/%s\n", runtime.GOOS, runtime.GOARCH)
}

// SourceURL is the project source URL
const SourceURL = "https://github.com/ariel-frischer/autospec"

var sauceCmd = &cobra.Command{
	Use:   "sauce",
	Short: "Display the source URL",
	Long:  "Display the source URL for the autospec project",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Println(SourceURL)
	},
}

// printPrettyVersion prints a styled version output with logo and box
func printPrettyVersion() {
	termWidth := shared.GetTerminalWidth()

	// Color setup
	cyan := color.New(color.FgCyan, color.Bold).SprintFunc()
	dim := color.New(color.Faint).SprintFunc()
	white := color.New(color.FgWhite, color.Bold).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()

	// Print logo centered (use fixed display width for unicode block chars)
	fmt.Println()
	logoPadding := (termWidth - shared.LogoDisplayWidth) / 2
	for _, line := range shared.Logo {
		fmt.Println(cyan(strings.Repeat(" ", logoPadding) + line))
	}
	fmt.Println()

	// Tagline
	fmt.Println(dim(shared.CenterText(shared.Tagline, termWidth)))
	fmt.Println()

	// Build version info content
	info := []struct {
		label string
		value string
	}{
		{"Version", Version},
		{"Commit", truncateCommit(Commit)},
		{"Built", BuildDate},
		{"Go", runtime.Version()},
		{"Platform", fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)},
	}

	// Calculate box width (minimum 40, max 60)
	boxWidth := 44
	if termWidth < 50 {
		boxWidth = termWidth - 6
	}
	contentWidth := boxWidth - 4 // Account for borders and padding

	// Print box centered
	boxPadding := (termWidth - boxWidth) / 2
	pad := strings.Repeat(" ", boxPadding)

	// Top border
	fmt.Println(pad + shared.BoxTopLeft + strings.Repeat(shared.BoxHorizontal, boxWidth-2) + shared.BoxTopRight)

	// Empty line
	fmt.Println(pad + shared.BoxVertical + strings.Repeat(" ", boxWidth-2) + shared.BoxVertical)

	// Content lines
	for _, item := range info {
		label := yellow(fmt.Sprintf("%12s", item.label))
		value := white(item.value)
		line := fmt.Sprintf("  %s    %s", label, value)
		// Pad to fill the box
		lineLen := 12 + 4 + len(item.value) + 2 // label width + spacing + value + margin
		if lineLen < contentWidth {
			line += strings.Repeat(" ", contentWidth-lineLen)
		}
		fmt.Println(pad + shared.BoxVertical + " " + line + " " + shared.BoxVertical)
	}

	// Empty line
	fmt.Println(pad + shared.BoxVertical + strings.Repeat(" ", boxWidth-2) + shared.BoxVertical)

	// Bottom border
	fmt.Println(pad + shared.BoxBottomLeft + strings.Repeat(shared.BoxHorizontal, boxWidth-2) + shared.BoxBottomRight)
	fmt.Println()
}

// truncateCommit shortens commit hash if it's too long
func truncateCommit(commit string) string {
	if len(commit) > 8 {
		return commit[:8]
	}
	return commit
}
