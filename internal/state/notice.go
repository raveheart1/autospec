// Package state provides persistent state management for autospec.
// It handles storing and retrieving state that persists across CLI invocations,
// such as one-time notices and feature flags.
package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// NoticeFileName is the name of the file that stores notice state
const NoticeFileName = "auto_commit_notice.json"

// AutoCommitNoticeState tracks whether the one-time auto-commit migration notice
// has been shown to the user. This is persisted to the state directory to ensure
// the notice is only displayed once per user.
type AutoCommitNoticeState struct {
	// NoticeShown indicates whether the migration notice has been displayed
	NoticeShown bool `json:"notice_shown"`
	// ShownAt is the timestamp when the notice was first shown (zero if not shown)
	ShownAt time.Time `json:"shown_at,omitempty"`
}

// LoadNoticeState loads the auto-commit notice state from the state directory.
// If the state file doesn't exist, returns a default state with NoticeShown=false.
// Performance contract: <10ms
func LoadNoticeState(stateDir string) (*AutoCommitNoticeState, error) {
	noticePath := filepath.Join(stateDir, NoticeFileName)
	data, err := os.ReadFile(noticePath)
	if err != nil {
		if os.IsNotExist(err) {
			// Return default state if file doesn't exist
			return &AutoCommitNoticeState{
				NoticeShown: false,
			}, nil
		}
		return nil, fmt.Errorf("reading notice state: %w", err)
	}

	var state AutoCommitNoticeState
	if err := json.Unmarshal(data, &state); err != nil {
		// If JSON is corrupted, return default state
		return &AutoCommitNoticeState{
			NoticeShown: false,
		}, nil
	}

	return &state, nil
}

// SaveNoticeState persists the notice state to the state directory using atomic write.
// Creates the state directory if it doesn't exist.
// Uses atomic write pattern: write to .tmp file, then rename to final path.
func SaveNoticeState(stateDir string, state *AutoCommitNoticeState) error {
	// Ensure state directory exists
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return fmt.Errorf("creating state directory: %w", err)
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling notice state: %w", err)
	}

	// Write to temp file
	noticePath := filepath.Join(stateDir, NoticeFileName)
	tmpPath := noticePath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("writing temp file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tmpPath, noticePath); err != nil {
		return fmt.Errorf("renaming temp file: %w", err)
	}

	return nil
}

// MarkNoticeShown updates the state to indicate the notice has been shown
// and persists it immediately.
func MarkNoticeShown(stateDir string) error {
	state := &AutoCommitNoticeState{
		NoticeShown: true,
		ShownAt:     time.Now(),
	}
	return SaveNoticeState(stateDir, state)
}

// ShouldShowNotice determines whether the auto-commit migration notice should
// be displayed. Returns true only if:
//   - The notice hasn't been shown before (NoticeShown=false)
//   - The config is using the default value (not explicitly set by user)
//
// Parameters:
//   - stateDir: path to the state directory
//   - isExplicitConfig: true if the user explicitly set auto_commit in config or via flag
//
// Returns:
//   - bool: true if notice should be shown
//   - error: any error from loading state (nil if state file doesn't exist)
func ShouldShowNotice(stateDir string, isExplicitConfig bool) (bool, error) {
	// If user explicitly configured auto_commit, don't show notice
	if isExplicitConfig {
		return false, nil
	}

	state, err := LoadNoticeState(stateDir)
	if err != nil {
		return false, err
	}

	// Show notice if it hasn't been shown before
	return !state.NoticeShown, nil
}
