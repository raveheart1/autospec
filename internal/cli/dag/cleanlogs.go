package dag

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ariel-frischer/autospec/internal/config"
	"github.com/ariel-frischer/autospec/internal/dag"
	"github.com/ariel-frischer/autospec/internal/history"
	"github.com/ariel-frischer/autospec/internal/lifecycle"
	"github.com/ariel-frischer/autospec/internal/notify"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var cleanLogsCmd = &cobra.Command{
	Use:     "clean-logs",
	Aliases: []string{"cleanlogs"},
	Short:   "Clean up DAG log files from cache",
	Long: `Clean up DAG log files stored in the user cache directory.

The dag clean-logs command helps reclaim disk space by removing log files
from completed DAG runs. Logs are stored in the XDG cache directory
(~/.cache/autospec/dag-logs/ or $XDG_CACHE_HOME/autospec/dag-logs/).

By default, this command shows logs for the current project only.
Use --all to see and clean logs across all projects.

Exit codes:
  0 - Cleanup completed successfully
  1 - Error during cleanup`,
	Example: `  # List and optionally delete logs for current project
  autospec dag clean-logs

  # List and optionally delete logs for all projects
  autospec dag clean-logs --all

  # Delete without prompting (current project)
  autospec dag clean-logs --force

  # Delete without prompting (all projects)
  autospec dag clean-logs --all --force`,
	RunE: runDagCleanLogs,
}

func init() {
	cleanLogsCmd.Flags().Bool("all", false, "Clean logs from all projects")
	cleanLogsCmd.Flags().BoolP("force", "f", false, "Delete without prompting")
	DagCmd.AddCommand(cleanLogsCmd)
}

func runDagCleanLogs(cmd *cobra.Command, _ []string) error {
	cmd.SilenceUsage = true

	allProjects, _ := cmd.Flags().GetBool("all")
	force, _ := cmd.Flags().GetBool("force")

	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	notifHandler := notify.NewHandler(cfg.Notifications)
	historyLogger := history.NewWriter(cfg.StateDir, cfg.MaxHistoryEntries)

	scopeLabel := "current project"
	if allProjects {
		scopeLabel = "all projects"
	}

	return lifecycle.RunWithHistoryContext(cmd.Context(), notifHandler, historyLogger, "dag-clean-logs", scopeLabel, func(_ context.Context) error {
		return executeDagCleanLogs(allProjects, force)
	})
}

func executeDagCleanLogs(allProjects, force bool) error {
	if allProjects {
		return cleanLogsAllProjects(force)
	}
	return cleanLogsCurrentProject(force)
}

// cleanLogsCurrentProject cleans logs for the current project only.
func cleanLogsCurrentProject(force bool) error {
	projectID := dag.GetProjectID()
	logBase := dag.GetCacheLogBase(nil)
	projectDir := filepath.Join(logBase, projectID)

	entries, err := listDAGsInDir(projectDir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No log files found for current project.")
			return nil
		}
		return fmt.Errorf("listing DAGs: %w", err)
	}

	if len(entries) == 0 {
		fmt.Println("No log files found for current project.")
		return nil
	}

	totalSize, dagSizes := calculateProjectLogSizes(projectDir, entries)

	printCurrentProjectLogs(projectID, dagSizes, totalSize)

	if !force && !promptConfirmation(dag.FormatBytes(totalSize)) {
		fmt.Println("→ Logs kept")
		return nil
	}

	return deleteProjectLogs(projectDir, entries)
}

// cleanLogsAllProjects cleans logs across all projects.
func cleanLogsAllProjects(force bool) error {
	logBase := dag.GetCacheLogBase(nil)

	projects, err := os.ReadDir(logBase)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No log files found.")
			return nil
		}
		return fmt.Errorf("listing projects: %w", err)
	}

	projectInfo := collectProjectInfo(logBase, projects)
	if len(projectInfo) == 0 {
		fmt.Println("No log files found.")
		return nil
	}

	totalSize := printAllProjectsLogs(projectInfo)

	if !force && !promptConfirmation(dag.FormatBytes(totalSize)) {
		fmt.Println("→ Logs kept")
		return nil
	}

	return deleteAllProjectLogs(logBase, projectInfo)
}

// dagLogInfo holds information about a DAG's logs.
type dagLogInfo struct {
	dagID         string
	sizeBytes     int64
	sizeFormatted string
}

// projectLogInfo holds information about a project's logs.
type projectLogInfo struct {
	projectID      string
	dags           []dagLogInfo
	totalBytes     int64
	totalFormatted string
}

// listDAGsInDir returns directory entries that are directories (DAGs).
func listDAGsInDir(dir string) ([]os.DirEntry, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var dags []os.DirEntry
	for _, e := range entries {
		if e.IsDir() {
			dags = append(dags, e)
		}
	}
	return dags, nil
}

// calculateProjectLogSizes calculates log sizes for all DAGs in a project.
func calculateProjectLogSizes(projectDir string, entries []os.DirEntry) (int64, []dagLogInfo) {
	var totalSize int64
	var dagSizes []dagLogInfo

	for _, entry := range entries {
		dagDir := filepath.Join(projectDir, entry.Name())
		sizeBytes, sizeFormatted := dag.CalculateLogDirSize(dagDir)
		totalSize += sizeBytes
		dagSizes = append(dagSizes, dagLogInfo{
			dagID:         entry.Name(),
			sizeBytes:     sizeBytes,
			sizeFormatted: sizeFormatted,
		})
	}

	return totalSize, dagSizes
}

// collectProjectInfo gathers log info for all projects.
func collectProjectInfo(logBase string, projects []os.DirEntry) []projectLogInfo {
	var infos []projectLogInfo

	for _, proj := range projects {
		if !proj.IsDir() {
			continue
		}

		projectDir := filepath.Join(logBase, proj.Name())
		dags, err := listDAGsInDir(projectDir)
		if err != nil || len(dags) == 0 {
			continue
		}

		totalSize, dagSizes := calculateProjectLogSizes(projectDir, dags)
		if totalSize == 0 {
			continue
		}

		infos = append(infos, projectLogInfo{
			projectID:      proj.Name(),
			dags:           dagSizes,
			totalBytes:     totalSize,
			totalFormatted: dag.FormatBytes(totalSize),
		})
	}

	return infos
}

// printCurrentProjectLogs displays log information for the current project.
func printCurrentProjectLogs(projectID string, dagSizes []dagLogInfo, totalSize int64) {
	fmt.Println("=== DAG Logs for Current Project ===")
	fmt.Printf("Project: %s\n\n", projectID)

	for _, d := range dagSizes {
		fmt.Printf("  • %s: %s\n", d.dagID, d.sizeFormatted)
	}

	fmt.Printf("\nTotal: %s\n", dag.FormatBytes(totalSize))
}

// printAllProjectsLogs displays log information for all projects.
func printAllProjectsLogs(projectInfo []projectLogInfo) int64 {
	fmt.Println("=== DAG Logs Across All Projects ===")
	fmt.Println()

	var grandTotal int64
	for _, proj := range projectInfo {
		grandTotal += proj.totalBytes
		fmt.Printf("Project: %s (%s)\n", proj.projectID, proj.totalFormatted)
		for _, d := range proj.dags {
			fmt.Printf("  • %s: %s\n", d.dagID, d.sizeFormatted)
		}
		fmt.Println()
	}

	fmt.Printf("Grand Total: %s\n", dag.FormatBytes(grandTotal))
	return grandTotal
}

// promptConfirmation asks user to confirm deletion.
// Returns true if user confirms, false otherwise.
func promptConfirmation(totalSize string) bool {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		fmt.Printf("→ Logs kept [non-interactive mode]\n")
		return false
	}

	fmt.Printf("\nDelete all logs? (%s) [y/N] ", totalSize)

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	input = strings.TrimSpace(strings.ToLower(input))
	return input == "y" || input == "yes"
}

// deleteProjectLogs removes all log directories for a project.
func deleteProjectLogs(projectDir string, entries []os.DirEntry) error {
	var totalDeleted int64

	for _, entry := range entries {
		dagDir := filepath.Join(projectDir, entry.Name())
		sizeBytes, _ := dag.CalculateLogDirSize(dagDir)

		if err := os.RemoveAll(dagDir); err != nil {
			red := color.New(color.FgRed)
			red.Fprintf(os.Stderr, "✗ Failed to remove %s: %v\n", entry.Name(), err)
			continue
		}

		totalDeleted += sizeBytes
		fmt.Printf("✓ Deleted %s\n", entry.Name())
	}

	green := color.New(color.FgGreen, color.Bold)
	green.Print("\n✓ Total freed: ")
	fmt.Printf("%s\n", dag.FormatBytes(totalDeleted))
	return nil
}

// deleteAllProjectLogs removes log directories for all projects.
func deleteAllProjectLogs(logBase string, projectInfo []projectLogInfo) error {
	var totalDeleted int64

	for _, proj := range projectInfo {
		projectDir := filepath.Join(logBase, proj.projectID)
		sizeBytes := proj.totalBytes

		if err := os.RemoveAll(projectDir); err != nil {
			red := color.New(color.FgRed)
			red.Fprintf(os.Stderr, "✗ Failed to remove project %s: %v\n", proj.projectID, err)
			continue
		}

		totalDeleted += sizeBytes
		fmt.Printf("✓ Deleted project %s (%s)\n", proj.projectID, proj.totalFormatted)
	}

	green := color.New(color.FgGreen, color.Bold)
	green.Print("\n✓ Total freed: ")
	fmt.Printf("%s\n", dag.FormatBytes(totalDeleted))
	return nil
}
