package dag

import (
	"os"
	"path/filepath"
)

// GetCacheLogDir returns the full path to the log directory for a specific DAG run.
// The path follows the structure: <cache-base>/autospec/dag-logs/<project-id>/<dag-id>/
//
// Parameters:
//   - projectID: The unique project identifier (from GetProjectID)
//   - dagID: The DAG identifier (from ResolveDAGID)
//
// Returns the full path where log files for this DAG should be stored.
func GetCacheLogDir(projectID, dagID string) string {
	return filepath.Join(GetCacheBase(), "autospec", "dag-logs", projectID, dagID)
}

// GetCacheBase returns the base cache directory for autospec.
// It uses the XDG_CACHE_HOME environment variable if set,
// otherwise falls back to os.UserCacheDir() for cross-platform support.
//
// On Linux: ~/.cache (or $XDG_CACHE_HOME)
// On macOS: ~/Library/Caches
// On Windows: %LocalAppData%
func GetCacheBase() string {
	// Check XDG_CACHE_HOME first (takes precedence for consistency)
	if xdgCache := os.Getenv("XDG_CACHE_HOME"); xdgCache != "" {
		return xdgCache
	}

	// Fall back to os.UserCacheDir() for cross-platform support
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		// Last resort: use ~/.cache
		home, err := os.UserHomeDir()
		if err != nil {
			return filepath.Join(".", ".cache")
		}
		return filepath.Join(home, ".cache")
	}

	return cacheDir
}

// GetLogFilePath returns the full path to a specific spec's log file.
// The path follows the structure: <cache-log-dir>/<spec-id>.log
func GetLogFilePath(projectID, dagID, specID string) string {
	return filepath.Join(GetCacheLogDir(projectID, dagID), specID+".log")
}

// EnsureCacheLogDir creates the cache log directory if it doesn't exist.
func EnsureCacheLogDir(projectID, dagID string) error {
	logDir := GetCacheLogDir(projectID, dagID)
	return os.MkdirAll(logDir, 0o755)
}
