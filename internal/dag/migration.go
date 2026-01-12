package dag

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// MigrationResult contains the outcome of a log migration operation.
type MigrationResult struct {
	// Migrated is the number of log files successfully migrated.
	Migrated int
	// Skipped is the number of log files already in the cache location.
	Skipped int
	// TotalBytes is the total size of migrated log files.
	TotalBytes int64
	// Errors contains spec IDs and their migration errors.
	Errors map[string]string
}

// MigrateLogs moves existing log files from the project directory to the cache.
// If the run has no logs in the old location, or all logs are already migrated,
// this function is a no-op. The state file is updated with new log paths.
func MigrateLogs(stateDir string, run *DAGRun) (*MigrationResult, error) {
	if run == nil {
		return nil, fmt.Errorf("run is nil")
	}

	result := &MigrationResult{Errors: make(map[string]string)}

	// Ensure run has project_id and log_base set for cache paths
	if err := ensureLogBaseFields(run); err != nil {
		return nil, fmt.Errorf("setting log base fields: %w", err)
	}

	// Check for legacy log directory
	legacyLogDir := GetLogDir(stateDir, run.RunID)
	if !dirExists(legacyLogDir) {
		return result, nil // No legacy logs to migrate
	}

	// Migrate each log file
	if err := migrateLogFiles(run, legacyLogDir, result); err != nil {
		return result, fmt.Errorf("migrating log files: %w", err)
	}

	// Save updated state file with new log paths
	if result.Migrated > 0 {
		if err := SaveStateByWorkflow(stateDir, run); err != nil {
			return result, fmt.Errorf("saving updated state: %w", err)
		}
	}

	return result, nil
}

// ensureLogBaseFields populates ProjectID and LogBase if not already set.
func ensureLogBaseFields(run *DAGRun) error {
	if run.ProjectID == "" {
		run.ProjectID = GetProjectID()
	}
	if run.LogBase == "" {
		run.LogBase = GetCacheLogDir(run.ProjectID, run.DAGId)
	}
	return nil
}

// migrateLogFiles moves log files from legacy dir to cache dir.
func migrateLogFiles(run *DAGRun, legacyDir string, result *MigrationResult) error {
	if err := EnsureCacheLogDir(run.ProjectID, run.DAGId); err != nil {
		return fmt.Errorf("creating cache log directory: %w", err)
	}

	for specID, specState := range run.Specs {
		if err := migrateSpecLog(run, specID, specState, legacyDir, result); err != nil {
			result.Errors[specID] = err.Error()
		}
	}

	// Clean up legacy directory if empty
	cleanupLegacyDir(legacyDir)

	return nil
}

// migrateSpecLog migrates a single spec's log file.
func migrateSpecLog(run *DAGRun, specID string, specState *SpecState, legacyDir string, result *MigrationResult) error {
	legacyPath := filepath.Join(legacyDir, specID+".log")
	if !fileExists(legacyPath) {
		return nil // No legacy log for this spec
	}

	// Check if already in cache location
	cachePath := GetLogFilePath(run.ProjectID, run.DAGId, specID)
	if fileExists(cachePath) {
		result.Skipped++
		specState.LogFile = specID + ".log"
		return nil
	}

	// Move the file
	bytesWritten, err := moveFile(legacyPath, cachePath)
	if err != nil {
		return fmt.Errorf("moving log file: %w", err)
	}

	// Update spec state
	specState.LogFile = specID + ".log"
	result.Migrated++
	result.TotalBytes += bytesWritten

	return nil
}

// moveFile copies src to dst and removes src.
func moveFile(src, dst string) (int64, error) {
	srcFile, err := os.Open(src)
	if err != nil {
		return 0, fmt.Errorf("opening source: %w", err)
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return 0, fmt.Errorf("creating destination: %w", err)
	}
	defer dstFile.Close()

	bytesWritten, err := io.Copy(dstFile, srcFile)
	if err != nil {
		return bytesWritten, fmt.Errorf("copying data: %w", err)
	}

	// Close files before removing source
	srcFile.Close()
	dstFile.Close()

	if err := os.Remove(src); err != nil {
		return bytesWritten, fmt.Errorf("removing source: %w", err)
	}

	return bytesWritten, nil
}

// cleanupLegacyDir removes the legacy log directory if empty.
func cleanupLegacyDir(legacyDir string) {
	entries, err := os.ReadDir(legacyDir)
	if err != nil || len(entries) > 0 {
		return
	}
	os.Remove(legacyDir) // Best effort removal of empty directory
}

// fileExists returns true if the path exists and is a file.
func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// HasOldLogs checks if a run has logs in the old project directory location.
// This is used to determine if migration is needed on resume.
func HasOldLogs(stateDir string, run *DAGRun) bool {
	if run == nil {
		return false
	}
	legacyLogDir := GetLogDir(stateDir, run.RunID)
	if !dirExists(legacyLogDir) {
		return false
	}
	// Check if any log files exist for the specs in this run
	for specID := range run.Specs {
		legacyPath := filepath.Join(legacyLogDir, specID+".log")
		if fileExists(legacyPath) {
			return true
		}
	}
	return false
}
