package history

import (
	"fmt"
	"os"
	"time"
)

// Writer provides thread-safe history logging with automatic pruning.
type Writer struct {
	// StateDir is the directory containing the history file.
	StateDir string
	// MaxEntries is the maximum number of entries to retain.
	MaxEntries int
}

// NewWriter creates a new history writer.
func NewWriter(stateDir string, maxEntries int) *Writer {
	return &Writer{
		StateDir:   stateDir,
		MaxEntries: maxEntries,
	}
}

// LogEntry adds a new entry to the history file.
// It loads the existing history, appends the new entry, prunes if needed, and saves.
// Errors are non-fatal: they are written to stderr and don't cause command failures.
func (w *Writer) LogEntry(entry HistoryEntry) {
	if err := w.logEntryInternal(entry); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log history: %v\n", err)
	}
}

// logEntryInternal handles the actual logging logic.
func (w *Writer) logEntryInternal(entry HistoryEntry) error {
	history, err := LoadHistory(w.StateDir)
	if err != nil {
		return fmt.Errorf("loading history: %w", err)
	}

	history.Entries = append(history.Entries, entry)

	// Prune oldest entries if over limit
	if w.MaxEntries > 0 && len(history.Entries) > w.MaxEntries {
		excess := len(history.Entries) - w.MaxEntries
		history.Entries = history.Entries[excess:]
	}

	if err := SaveHistory(w.StateDir, history); err != nil {
		return fmt.Errorf("saving history: %w", err)
	}

	return nil
}

// LogCommand is a convenience method to log a command execution.
func (w *Writer) LogCommand(command, spec string, exitCode int, duration time.Duration) {
	entry := HistoryEntry{
		Timestamp: time.Now(),
		Command:   command,
		Spec:      spec,
		ExitCode:  exitCode,
		Duration:  duration.String(),
	}
	w.LogEntry(entry)
}
