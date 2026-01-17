package util

import (
	"context"
	"fmt"
	"strings"

	"github.com/ariel-frischer/autospec/internal/changelog"
	"github.com/ariel-frischer/autospec/internal/cli/shared"
	"github.com/ariel-frischer/autospec/internal/update"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var ckPlain bool

// ckCmd is the command for checking if an update is available.
var ckCmd = &cobra.Command{
	Use:     "ck",
	Aliases: []string{"check"},
	Short:   "Check if an update is available",
	Long:    "Check if a newer version of autospec is available on GitHub releases.",
	Example: `  # Check for available updates
  autospec ck

  # Plain output (for scripts)
  autospec ck --plain

  # Using the longer alias
  autospec check`,
	RunE: runCheck,
}

func init() {
	ckCmd.GroupID = shared.GroupGettingStarted
	ckCmd.Flags().BoolVar(&ckPlain, "plain", false, "Plain output without formatting")
}

// runCheck executes the update check command.
func runCheck(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	checker := update.NewChecker(update.DefaultHTTPTimeout)
	output, err := executeCheck(ctx, checker, Version, ckPlain)
	if err != nil {
		return err
	}

	fmt.Fprint(cmd.OutOrStdout(), output)
	return nil
}

// executeCheck performs the update check and returns formatted output.
// The plain parameter controls whether output is formatted for scripts.
func executeCheck(ctx context.Context, checker *update.Checker, version string, plain bool) (string, error) {
	// Handle dev builds without making network calls
	if version == "dev" || version == "" {
		return formatDevBuildMessage(version, plain), nil
	}

	// Perform the update check
	check, err := checker.CheckForUpdate(ctx, version)
	if err != nil {
		return handleCheckError(err, plain)
	}

	return formatCheckResult(check, plain), nil
}

// formatDevBuildMessage returns a message for dev builds.
func formatDevBuildMessage(version string, plain bool) string {
	if plain {
		return fmt.Sprintf("version: %s\nstatus: dev-build\nmessage: update check not applicable\n", version)
	}

	yellow := color.New(color.FgYellow).SprintFunc()
	dim := color.New(color.Faint).SprintFunc()
	return fmt.Sprintf("%s Running dev build - update check not applicable\n%s\n",
		yellow("⚠"),
		dim("  Install a release version to enable update checks"))
}

// handleCheckError converts errors to user-friendly messages.
func handleCheckError(err error, plain bool) (string, error) {
	errStr := err.Error()

	switch {
	case strings.Contains(errStr, "rate limit"):
		return formatErrorMessage("GitHub API rate limit exceeded", "Please try again later", plain), nil
	case strings.Contains(errStr, "no releases"):
		return formatErrorMessage("No releases found", "No releases found on GitHub", plain), nil
	case strings.Contains(errStr, "deadline exceeded") || strings.Contains(errStr, "timeout"):
		return formatErrorMessage("Network timeout", "Network timeout while checking for updates", plain), nil
	case strings.Contains(errStr, "no asset found"):
		return formatErrorMessage("Platform not supported", "No release asset for this platform", plain), nil
	case strings.Contains(errStr, "context canceled"):
		return "", err
	default:
		return "", fmt.Errorf("checking for update: %w", err)
	}
}

// formatErrorMessage returns a formatted error message.
func formatErrorMessage(title, detail string, plain bool) string {
	if plain {
		return fmt.Sprintf("error: %s - %s\n", title, detail)
	}
	red := color.New(color.FgRed).SprintFunc()
	return fmt.Sprintf("%s %s\n  %s\n", red("✗"), title, detail)
}

// formatCheckResult formats the update check result for display.
func formatCheckResult(check *update.UpdateCheck, plain bool) string {
	if check.UpdateAvailable {
		return formatUpdateAvailable(check, plain)
	}
	return formatUpToDate(check, plain)
}

// formatUpdateAvailable returns output when an update is available.
func formatUpdateAvailable(check *update.UpdateCheck, plain bool) string {
	if plain {
		return formatUpdateAvailablePlain(check)
	}
	return formatUpdateAvailableStyled(check)
}

// formatUpdateAvailablePlain returns plain text output for update available.
func formatUpdateAvailablePlain(check *update.UpdateCheck) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("current: %s\nlatest: %s\nupdate_available: true\n",
		check.CurrentVersion, check.LatestVersion))
	sb.WriteString(formatChangelogPreview(check.LatestVersion, true))
	return sb.String()
}

// formatUpdateAvailableStyled returns styled output for update available.
func formatUpdateAvailableStyled(check *update.UpdateCheck) string {
	green := color.New(color.FgGreen, color.Bold).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()
	dim := color.New(color.Faint).SprintFunc()

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s Update available: %s → %s\n%s\n",
		green("✓"),
		dim(check.CurrentVersion),
		cyan(check.LatestVersion),
		dim("  Run 'autospec update' to upgrade")))

	sb.WriteString(formatChangelogPreview(check.LatestVersion, false))
	return sb.String()
}

// formatChangelogPreview returns a preview of changelog highlights for the given version.
// Shows 2-3 entries with truncation indicator if more exist.
// Tries remote changelog first (for newer versions), then falls back to embedded.
func formatChangelogPreview(version string, plain bool) string {
	ctx, cancel := context.WithTimeout(context.Background(), changelog.DefaultRemoteTimeout)
	defer cancel()

	v, _, err := changelog.FetchVersionFromRemote(ctx, version)
	if err != nil {
		return ""
	}

	entries := v.Entries()
	if len(entries) == 0 {
		return ""
	}

	return renderChangelogPreview(entries, version, plain)
}

// renderChangelogPreview renders the preview entries with optional truncation message.
func renderChangelogPreview(entries []changelog.Entry, version string, plain bool) string {
	opts := changelog.FormatOptions{Plain: plain}
	maxPreview := 3

	var sb strings.Builder
	sb.WriteString("\n")

	if plain {
		sb.WriteString(fmt.Sprintf("what's new in %s:\n", version))
	} else {
		dim := color.New(color.Faint).SprintFunc()
		sb.WriteString(dim(fmt.Sprintf("What's new in %s:", version)) + "\n")
	}

	displayCount := min(len(entries), maxPreview)
	for i := 0; i < displayCount; i++ {
		sb.WriteString("  " + changelog.FormatEntrySummary(entries[i], opts) + "\n")
	}

	if len(entries) > maxPreview {
		sb.WriteString(formatTruncationMessage(len(entries)-maxPreview, version, plain))
	}

	return sb.String()
}

// formatTruncationMessage returns the truncation indicator with suggestion.
func formatTruncationMessage(remaining int, version string, plain bool) string {
	if plain {
		return fmt.Sprintf("  ...and %d more. Run 'autospec changelog %s' for details.\n", remaining, version)
	}
	dim := color.New(color.Faint).SprintFunc()
	return dim(fmt.Sprintf("  ...and %d more. Run 'autospec changelog %s' for details.\n", remaining, version))
}

// formatUpToDate returns output when already on latest version.
func formatUpToDate(check *update.UpdateCheck, plain bool) string {
	if plain {
		return fmt.Sprintf("current: %s\nlatest: %s\nupdate_available: false\n",
			check.CurrentVersion, check.LatestVersion)
	}

	green := color.New(color.FgGreen).SprintFunc()
	return fmt.Sprintf("%s Already on latest version (%s)\n",
		green("✓"),
		check.CurrentVersion)
}
