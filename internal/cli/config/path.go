// Package config provides CLI configuration commands including init.
// path.go contains path resolution utilities for the init command,
// enabling initialization at arbitrary filesystem locations.
package config

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

// ResolvePath converts a raw path argument to an absolute path.
// It handles the following cases:
//   - Empty string or ".": returns current working directory
//   - "~" or "~/...": expands tilde to user home directory
//   - Relative path: resolves against current working directory
//   - Absolute path: returns unchanged
//
// Returns an error if the path cannot be resolved.
func ResolvePath(rawPath string) (string, error) {
	// Handle empty path or current directory marker
	if rawPath == "" || rawPath == "." {
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("getting current directory: %w", err)
		}
		return cwd, nil
	}

	// Handle tilde expansion
	if strings.HasPrefix(rawPath, "~") {
		expanded, err := expandTilde(rawPath)
		if err != nil {
			return "", fmt.Errorf("expanding tilde in path: %w", err)
		}
		rawPath = expanded
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(rawPath)
	if err != nil {
		return "", fmt.Errorf("resolving absolute path: %w", err)
	}

	return absPath, nil
}

// expandTilde expands a leading tilde (~) to the user's home directory.
// Supports both "~" alone and "~/path" forms.
func expandTilde(path string) (string, error) {
	if path == "~" {
		u, err := user.Current()
		if err != nil {
			return "", fmt.Errorf("getting current user: %w", err)
		}
		return u.HomeDir, nil
	}

	if strings.HasPrefix(path, "~/") {
		u, err := user.Current()
		if err != nil {
			return "", fmt.Errorf("getting current user: %w", err)
		}
		return filepath.Join(u.HomeDir, path[2:]), nil
	}

	// Path doesn't start with tilde, return as-is
	return path, nil
}

// EnsureDirectory ensures the target path exists as a directory.
// Creates the directory (and any parents) with 0755 permissions if needed.
// Returns an error if:
//   - The path exists but is a file (not a directory)
//   - The directory cannot be created (e.g., permission denied)
func EnsureDirectory(path string) error {
	info, err := os.Stat(path)
	if err == nil {
		// Path exists - check if it's a directory
		if !info.IsDir() {
			return fmt.Errorf("path exists and is not a directory: %s", path)
		}
		// Directory already exists
		return nil
	}

	// Path doesn't exist - create it
	if os.IsNotExist(err) {
		if err := os.MkdirAll(path, 0o755); err != nil {
			return fmt.Errorf("creating directory %s: %w", path, err)
		}
		return nil
	}

	// Some other error (e.g., permission denied on parent)
	return fmt.Errorf("checking path %s: %w", path, err)
}

// resolveTargetDirectory determines the target directory for init command.
// Priority: path argument > --here flag > current directory.
// Returns empty string if the resolved path is the current directory.
func resolveTargetDirectory(args []string, here bool) (string, error) {
	var rawPath string

	// Path argument takes precedence over --here flag
	if len(args) > 0 && args[0] != "" {
		rawPath = args[0]
	} else if here {
		rawPath = "."
	}

	// No path specified - use current directory (return empty to skip chdir)
	if rawPath == "" || rawPath == "." {
		return "", nil
	}

	// Resolve the path to absolute
	resolved, err := ResolvePath(rawPath)
	if err != nil {
		return "", err
	}

	// Check if resolved path is current directory
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getting current directory: %w", err)
	}

	if resolved == cwd {
		return "", nil // Already in target directory
	}

	return resolved, nil
}
