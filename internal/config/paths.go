package config

import (
	"os"
	"path/filepath"
)

// UserConfigPath returns the path to the user-level config file.
// This follows the XDG Base Directory Specification:
// - Linux: ~/.config/autospec/config.yml
// - macOS: ~/Library/Application Support/autospec/config.yml
// - Windows: %APPDATA%\autospec\config.yml
//
// If XDG_CONFIG_HOME is set, it will be respected on Linux.
func UserConfigPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "autospec", "config.yml"), nil
}

// UserConfigDir returns the path to the user-level config directory.
func UserConfigDir() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "autospec"), nil
}

// ProjectConfigPath returns the path to the project-level config file.
// This is always .autospec/config.yml relative to the current directory.
func ProjectConfigPath() string {
	return filepath.Join(".autospec", "config.yml")
}

// ProjectConfigDir returns the path to the project-level config directory.
func ProjectConfigDir() string {
	return ".autospec"
}

// LegacyUserConfigPath returns the path to the legacy user-level JSON config file.
// This was the old location: ~/.autospec/config.json
func LegacyUserConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".autospec", "config.json"), nil
}

// LegacyProjectConfigPath returns the path to the legacy project-level JSON config file.
// This was the old location: .autospec/config.json
func LegacyProjectConfigPath() string {
	return filepath.Join(".autospec", "config.json")
}

// LegacyGlobalConfigPath returns the path to the legacy global JSON config file.
// This is kept for backward compatibility during migration.
func LegacyGlobalConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".autospec", "config.json"), nil
}
