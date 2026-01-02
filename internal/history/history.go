// Package history provides command execution history storage and retrieval.
package history

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	// HistoryFileName is the name of the history file.
	HistoryFileName = "history.yaml"
	// BackupSuffix is the suffix for backup files when corruption is detected.
	BackupSuffix = ".backup"
)

// Status constants for history entries.
const (
	// StatusRunning indicates the command is currently executing.
	StatusRunning = "running"
	// StatusCompleted indicates the command finished successfully.
	StatusCompleted = "completed"
	// StatusFailed indicates the command finished with an error.
	StatusFailed = "failed"
	// StatusCancelled indicates the command was interrupted by the user.
	StatusCancelled = "cancelled"
)

// HistoryEntry represents a single command execution record.
type HistoryEntry struct {
	// ID is a unique identifier in adjective_noun_YYYYMMDD_HHMMSS format.
	// Optional for backward compatibility with old entries.
	ID string `yaml:"id,omitempty"`
	// Timestamp is when the command started executing (RFC3339 format in YAML).
	// Kept for backward compatibility with existing entries.
	Timestamp time.Time `yaml:"timestamp"`
	// Command is the name of the autospec command (e.g., "specify", "run").
	Command string `yaml:"command"`
	// Spec is the name or path of the spec being worked on (may be empty).
	Spec string `yaml:"spec,omitempty"`
	// Status is the current state: running, completed, failed, cancelled.
	// Optional for backward compatibility with old entries.
	Status string `yaml:"status,omitempty"`
	// CreatedAt is when the command started (explicit field, same as Timestamp).
	// Optional for backward compatibility with old entries.
	CreatedAt time.Time `yaml:"created_at,omitempty"`
	// CompletedAt is when the command finished (nil if still running).
	// Pointer allows distinguishing between "not set" and "zero time".
	CompletedAt *time.Time `yaml:"completed_at,omitempty"`
	// ExitCode is the exit code of the command (0=success).
	ExitCode int `yaml:"exit_code"`
	// Duration is the execution duration in Go duration format (e.g., "2m15.123s").
	Duration string `yaml:"duration"`
}

// HistoryFile represents the YAML file containing all history entries.
type HistoryFile struct {
	// Entries is an ordered list of command executions (newest entries appended at end).
	Entries []HistoryEntry `yaml:"entries"`
}

// DefaultHistoryPath returns the default path for the history file.
// Location: ~/.autospec/state/history.yaml
func DefaultHistoryPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("getting home directory: %w", err)
	}
	return filepath.Join(homeDir, ".autospec", "state", HistoryFileName), nil
}

// LoadHistory loads the history file from the given state directory.
// Returns empty history if file doesn't exist.
// Handles corrupted files by backing them up and creating a fresh history.
func LoadHistory(stateDir string) (*HistoryFile, error) {
	historyPath := filepath.Join(stateDir, HistoryFileName)

	data, err := os.ReadFile(historyPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &HistoryFile{Entries: []HistoryEntry{}}, nil
		}
		return nil, fmt.Errorf("reading history file: %w", err)
	}

	var history HistoryFile
	if err := yaml.Unmarshal(data, &history); err != nil {
		if backupErr := backupCorruptedFile(historyPath); backupErr != nil {
			return nil, fmt.Errorf("backing up corrupted history file: %w", backupErr)
		}
		return &HistoryFile{Entries: []HistoryEntry{}}, nil
	}

	if history.Entries == nil {
		history.Entries = []HistoryEntry{}
	}

	return &history, nil
}

// backupCorruptedFile renames a corrupted file with a .backup suffix.
func backupCorruptedFile(path string) error {
	backupPath := path + BackupSuffix
	if err := os.Rename(path, backupPath); err != nil {
		return fmt.Errorf("renaming corrupted file to backup: %w", err)
	}
	return nil
}

// SaveHistory saves the history file to the given state directory using atomic writes.
// Creates parent directories if needed.
func SaveHistory(stateDir string, history *HistoryFile) error {
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		return fmt.Errorf("creating state directory: %w", err)
	}

	data, err := yaml.Marshal(history)
	if err != nil {
		return fmt.Errorf("marshaling history: %w", err)
	}

	historyPath := filepath.Join(stateDir, HistoryFileName)
	tmpPath := historyPath + ".tmp"

	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return fmt.Errorf("writing temp history file: %w", err)
	}

	if err := os.Rename(tmpPath, historyPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("renaming temp history file: %w", err)
	}

	return nil
}

// ClearHistory removes all entries from the history file.
func ClearHistory(stateDir string) error {
	return SaveHistory(stateDir, &HistoryFile{Entries: []HistoryEntry{}})
}
