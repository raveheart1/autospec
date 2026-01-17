package dag

import (
	"os"
	"path/filepath"
)

// GetCacheLogBase returns the base log directory for DAG logs, checking config override first.
// If cfg is non-nil and has a LogDir set, that path is used directly.
// Otherwise, falls back to the XDG cache path.
//
// This is the primary function to use when determining log paths,
// as it respects the AUTOSPEC_DAG_LOG_DIR environment variable
// and dag.log_dir config setting.
func GetCacheLogBase(cfg *DAGExecutionConfig) string {
	if cfg != nil && cfg.LogDir != "" {
		return cfg.LogDir
	}
	return filepath.Join(GetCacheBase(), "autospec", "dag-logs")
}

// GetCacheLogDir returns the full path to the log directory for a specific DAG run.
// The path follows the structure: <cache-base>/autospec/dag-logs/<project-id>/<dag-id>/
//
// Parameters:
//   - projectID: The unique project identifier (from GetProjectID)
//   - dagID: The DAG identifier (from ResolveDAGID)
//
// Returns the full path where log files for this DAG should be stored.
// Note: For config-aware paths, use GetCacheLogDirWithConfig instead.
func GetCacheLogDir(projectID, dagID string) string {
	return filepath.Join(GetCacheBase(), "autospec", "dag-logs", projectID, dagID)
}

// GetCacheLogDirWithConfig returns the full path to the log directory for a DAG run,
// respecting the config override if set.
func GetCacheLogDirWithConfig(cfg *DAGExecutionConfig, projectID, dagID string) string {
	return filepath.Join(GetCacheLogBase(cfg), projectID, dagID)
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
// Note: For config-aware paths, use GetLogFilePathWithConfig instead.
func GetLogFilePath(projectID, dagID, specID string) string {
	return filepath.Join(GetCacheLogDir(projectID, dagID), specID+".log")
}

// GetLogFilePathWithConfig returns the full path to a spec's log file,
// respecting the config override if set.
func GetLogFilePathWithConfig(cfg *DAGExecutionConfig, projectID, dagID, specID string) string {
	return filepath.Join(GetCacheLogDirWithConfig(cfg, projectID, dagID), specID+".log")
}

// EnsureCacheLogDir creates the cache log directory if it doesn't exist.
// Note: For config-aware directories, use EnsureCacheLogDirWithConfig instead.
func EnsureCacheLogDir(projectID, dagID string) error {
	logDir := GetCacheLogDir(projectID, dagID)
	return os.MkdirAll(logDir, 0o755)
}

// EnsureCacheLogDirWithConfig creates the cache log directory if it doesn't exist,
// respecting the config override if set.
func EnsureCacheLogDirWithConfig(cfg *DAGExecutionConfig, projectID, dagID string) error {
	logDir := GetCacheLogDirWithConfig(cfg, projectID, dagID)
	return os.MkdirAll(logDir, 0o755)
}

// CalculateLogDirSize calculates the total size of all files in the log directory.
// Returns the total size in bytes and a human-readable string (e.g., "127MB").
// If the directory doesn't exist or is empty, returns 0 and "0B".
// Errors reading individual files are ignored; they simply don't contribute to the total.
func CalculateLogDirSize(logDir string) (int64, string) {
	var totalSize int64

	entries, err := os.ReadDir(logDir)
	if err != nil {
		// Directory doesn't exist or can't be read
		return 0, "0B"
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue // Skip files we can't stat
		}
		totalSize += info.Size()
	}

	return totalSize, FormatBytes(totalSize)
}

// FormatBytes converts a byte count to a human-readable string.
// Uses SI units (KB, MB, GB) with appropriate precision.
func FormatBytes(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return formatSize(bytes, GB, "GB")
	case bytes >= MB:
		return formatSize(bytes, MB, "MB")
	case bytes >= KB:
		return formatSize(bytes, KB, "KB")
	default:
		return formatSizeInt(bytes, "B")
	}
}

// formatSize formats bytes as a float with 1 decimal place and unit suffix.
func formatSize(bytes int64, unit int64, suffix string) string {
	value := float64(bytes) / float64(unit)
	if value >= 100 {
		return formatSizeInt(int64(value), suffix)
	}
	return formatFloat(value, suffix)
}

// formatSizeInt formats a size as an integer with unit suffix.
func formatSizeInt(value int64, suffix string) string {
	return intToString(value) + suffix
}

// formatFloat formats a float with 1 decimal place and suffix.
func formatFloat(value float64, suffix string) string {
	// Manual formatting to avoid fmt import for this simple case
	intPart := int64(value)
	decPart := int64((value - float64(intPart)) * 10)
	if decPart == 0 {
		return intToString(intPart) + suffix
	}
	return intToString(intPart) + "." + intToString(decPart) + suffix
}

// intToString converts an int64 to string without fmt package.
func intToString(n int64) string {
	if n == 0 {
		return "0"
	}
	if n < 0 {
		return "-" + intToString(-n)
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}
