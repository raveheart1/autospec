package util

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/ariel-frischer/autospec/internal/cli/shared"
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
	fmt.Printf("  Run 'autospec version' to verify the update.\n")

	return nil
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
