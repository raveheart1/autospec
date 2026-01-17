package util

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/ariel-frischer/autospec/internal/changelog"
	"github.com/ariel-frischer/autospec/internal/cli/shared"
	"github.com/ariel-frischer/autospec/internal/config"
	"github.com/ariel-frischer/autospec/internal/update"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

const (
	// updateHTTPTimeout is the timeout for update-related HTTP requests.
	updateHTTPTimeout = 30 * time.Second
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update autospec to the latest version",
	Long:  "Download and install the latest version of autospec from GitHub releases.",
	Example: `  # Update to latest version
  autospec update`,
	RunE: runUpdate,
}

func init() {
	updateCmd.GroupID = shared.GroupGettingStarted
}

// runUpdate executes the update command.
func runUpdate(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	green := color.New(color.FgGreen, color.Bold).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	dim := color.New(color.Faint).SprintFunc()

	// Check for dev build
	if IsDevBuild() {
		return fmt.Errorf("cannot update dev builds; please build from source or use a release version")
	}

	fmt.Printf("%s Checking for updates...\n", yellow("→"))

	// Check for update
	checker := update.NewChecker(updateHTTPTimeout)
	check, err := checker.CheckForUpdate(ctx, Version)
	if err != nil {
		return fmt.Errorf("checking for update: %w", err)
	}

	if !check.UpdateAvailable {
		fmt.Printf("%s Already running the latest version (%s)\n", green("✓"), Version)
		return nil
	}

	fmt.Printf("%s New version available: %s → %s\n",
		green("→"), Version, green(check.LatestVersion))

	// Create installer and check permissions
	installer, err := update.NewInstaller()
	if err != nil {
		return fmt.Errorf("initializing installer: %w", err)
	}

	if err := installer.CheckWritePermission(); err != nil {
		return fmt.Errorf("permission check failed: %w", err)
	}

	// Download binary
	fmt.Printf("%s Downloading %s...\n", yellow("→"), check.AssetName)

	httpClient := &http.Client{Timeout: updateHTTPTimeout}
	downloader := update.NewDownloader(httpClient)

	archivePath, err := downloader.DownloadBinary(ctx, check.DownloadURL, func(current, total int64) {
		printProgress(current, total)
	})
	if err != nil {
		return fmt.Errorf("downloading binary: %w", err)
	}
	defer os.Remove(archivePath)
	fmt.Println() // New line after progress

	// Verify checksum if available
	if check.ChecksumURL != "" {
		fmt.Printf("%s Verifying checksum...\n", yellow("→"))

		checksum, err := downloader.FetchChecksum(ctx, check.ChecksumURL, check.AssetName)
		if err != nil {
			return fmt.Errorf("fetching checksum: %w", err)
		}

		if err := update.VerifyChecksum(archivePath, checksum); err != nil {
			return fmt.Errorf("checksum verification failed: %w", err)
		}
		fmt.Printf("%s Checksum verified\n", green("✓"))
	} else {
		fmt.Printf("%s No checksum file available, skipping verification\n", dim("!"))
	}

	// Extract binary
	fmt.Printf("%s Extracting binary...\n", yellow("→"))

	tmpDir, err := os.MkdirTemp("", "autospec-update-extract-*")
	if err != nil {
		return fmt.Errorf("creating temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	newBinaryPath, err := update.ExtractBinary(archivePath, tmpDir)
	if err != nil {
		return fmt.Errorf("extracting binary: %w", err)
	}

	// Install
	fmt.Printf("%s Installing update...\n", yellow("→"))

	// Create backup
	if err := installer.CreateBackup(); err != nil {
		return fmt.Errorf("creating backup: %w", err)
	}

	// Install new binary
	if err := installer.InstallBinary(newBinaryPath); err != nil {
		// Try to rollback
		if rollbackErr := installer.Rollback(); rollbackErr != nil {
			return fmt.Errorf("installation failed (%v) and rollback failed (%v)", err, rollbackErr)
		}
		return fmt.Errorf("installing binary: %w (rolled back to previous version)", err)
	}

	// Set permissions
	if err := installer.SetPermissions(); err != nil {
		// Try to rollback
		if rollbackErr := installer.Rollback(); rollbackErr != nil {
			return fmt.Errorf("setting permissions failed (%v) and rollback failed (%v)", err, rollbackErr)
		}
		return fmt.Errorf("setting permissions: %w (rolled back to previous version)", err)
	}

	// Cleanup backup
	if err := installer.CleanupBackup(); err != nil {
		// Non-fatal, just warn
		fmt.Printf("%s Warning: failed to cleanup backup: %v\n", dim("!"), err)
	}

	fmt.Printf("%s Successfully updated to %s\n", green("✓"), green(check.LatestVersion))

	// Sync user config with new schema
	syncUserConfig(yellow, green, dim)

	// Show changelog highlights for the new version
	showUpdateChangelog(ctx, check.LatestVersion, os.Stdout)

	fmt.Printf("\n")
	fmt.Printf("  Run 'autospec version' to verify the update.\n")
	fmt.Printf("\n")
	fmt.Printf("%s Run 'autospec init' in each project to update slash commands.\n", yellow("!"))
	fmt.Printf("  Command templates may have been improved in this release.\n")

	return nil
}

// syncUserConfig attempts to sync the user config with the current schema.
// This is a non-fatal operation - errors are logged but don't fail the update.
func syncUserConfig(yellow, green, dim func(a ...interface{}) string) {
	userConfigPath, err := config.UserConfigPath()
	if err != nil {
		return // Silently skip if we can't get the path
	}

	// Check if config file exists
	if _, err := os.Stat(userConfigPath); os.IsNotExist(err) {
		return // No config file to sync
	}

	fmt.Printf("%s Syncing configuration...\n", yellow("→"))

	result, syncErr := config.SyncConfig(userConfigPath, config.SyncOptions{})
	if syncErr != nil {
		fmt.Printf("%s Warning: config sync failed: %v\n", dim("!"), syncErr)
		return
	}

	if !result.Changed {
		fmt.Printf("%s Config is up to date\n", green("✓"))
		return
	}

	fmt.Printf("%s Config synced: %d new options added, %d deprecated removed\n",
		green("✓"), len(result.Added), len(result.Removed))
}

// printProgress prints a download progress bar.
func printProgress(current, total int64) {
	if total <= 0 {
		fmt.Printf("\r  Downloaded %s", formatBytes(current))
		return
	}

	percent := float64(current) / float64(total) * 100
	barWidth := 30
	filled := int(float64(barWidth) * float64(current) / float64(total))

	bar := ""
	for i := 0; i < barWidth; i++ {
		if i < filled {
			bar += "█"
		} else {
			bar += "░"
		}
	}

	fmt.Printf("\r  [%s] %.1f%% (%s/%s)", bar, percent,
		formatBytes(current), formatBytes(total))
}

// formatBytes formats a byte count as a human-readable string.
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// showUpdateChangelog displays changelog highlights for the newly installed version.
// This is a non-fatal operation - errors are silently ignored.
func showUpdateChangelog(ctx context.Context, version string, w io.Writer) {
	fetchCtx, cancel := context.WithTimeout(ctx, changelog.DefaultRemoteTimeout)
	defer cancel()

	v, _, err := changelog.FetchVersionFromRemote(fetchCtx, version)
	if err != nil {
		return
	}

	entries := v.Entries()
	if len(entries) == 0 {
		return
	}

	renderUpdateChangelog(entries, version, w)
}

// renderUpdateChangelog renders changelog entries for the update command output.
func renderUpdateChangelog(entries []changelog.Entry, version string, w io.Writer) {
	dim := color.New(color.Faint).SprintFunc()
	opts := changelog.FormatOptions{Plain: false}
	maxEntries := 5

	fmt.Fprintln(w)
	fmt.Fprintf(w, "%s\n", dim(fmt.Sprintf("What's new in %s:", version)))

	displayCount := min(len(entries), maxEntries)
	for i := 0; i < displayCount; i++ {
		fmt.Fprintf(w, "  %s\n", changelog.FormatEntrySummary(entries[i], opts))
	}

	if len(entries) > maxEntries {
		fmt.Fprintf(w, "  %s\n", dim(fmt.Sprintf("...and %d more.", len(entries)-maxEntries)))
	}

	fmt.Fprintf(w, "\n  %s\n", dim(fmt.Sprintf("Run 'autospec changelog %s' for full details.", version)))
}
