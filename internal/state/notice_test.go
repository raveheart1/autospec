package state

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadNoticeState(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		setup       func(t *testing.T, stateDir string)
		wantShown   bool
		wantErr     bool
		description string
	}{
		"returns default state when file does not exist": {
			setup:       func(t *testing.T, stateDir string) {},
			wantShown:   false,
			wantErr:     false,
			description: "When state file doesn't exist, NoticeShown should be false",
		},
		"loads existing state with NoticeShown true": {
			setup: func(t *testing.T, stateDir string) {
				data := `{"notice_shown": true, "shown_at": "2025-12-21T10:00:00Z"}`
				err := os.WriteFile(filepath.Join(stateDir, NoticeFileName), []byte(data), 0644)
				require.NoError(t, err)
			},
			wantShown:   true,
			wantErr:     false,
			description: "Should load NoticeShown=true from existing file",
		},
		"loads existing state with NoticeShown false": {
			setup: func(t *testing.T, stateDir string) {
				data := `{"notice_shown": false}`
				err := os.WriteFile(filepath.Join(stateDir, NoticeFileName), []byte(data), 0644)
				require.NoError(t, err)
			},
			wantShown:   false,
			wantErr:     false,
			description: "Should load NoticeShown=false from existing file",
		},
		"returns default state on corrupted JSON": {
			setup: func(t *testing.T, stateDir string) {
				data := `{this is not valid json`
				err := os.WriteFile(filepath.Join(stateDir, NoticeFileName), []byte(data), 0644)
				require.NoError(t, err)
			},
			wantShown:   false,
			wantErr:     false,
			description: "Corrupted JSON should return default state, not error",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			stateDir := t.TempDir()
			tt.setup(t, stateDir)

			state, err := LoadNoticeState(stateDir)

			if tt.wantErr {
				assert.Error(t, err, tt.description)
			} else {
				assert.NoError(t, err, tt.description)
				require.NotNil(t, state)
				assert.Equal(t, tt.wantShown, state.NoticeShown, tt.description)
			}
		})
	}
}

func TestSaveNoticeState(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		state       *AutoCommitNoticeState
		wantErr     bool
		description string
	}{
		"saves state with NoticeShown true": {
			state: &AutoCommitNoticeState{
				NoticeShown: true,
				ShownAt:     time.Date(2025, 12, 21, 10, 0, 0, 0, time.UTC),
			},
			wantErr:     false,
			description: "Should persist NoticeShown=true to file",
		},
		"saves state with NoticeShown false": {
			state: &AutoCommitNoticeState{
				NoticeShown: false,
			},
			wantErr:     false,
			description: "Should persist NoticeShown=false to file",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			stateDir := t.TempDir()

			err := SaveNoticeState(stateDir, tt.state)

			if tt.wantErr {
				assert.Error(t, err, tt.description)
			} else {
				assert.NoError(t, err, tt.description)

				// Verify the file was created
				noticePath := filepath.Join(stateDir, NoticeFileName)
				assert.FileExists(t, noticePath)

				// Verify we can load it back
				loaded, err := LoadNoticeState(stateDir)
				require.NoError(t, err)
				assert.Equal(t, tt.state.NoticeShown, loaded.NoticeShown)
			}
		})
	}
}

func TestSaveNoticeState_CreatesStateDir(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	stateDir := filepath.Join(tempDir, "nested", "state", "dir")

	state := &AutoCommitNoticeState{
		NoticeShown: true,
		ShownAt:     time.Now(),
	}

	err := SaveNoticeState(stateDir, state)
	assert.NoError(t, err, "Should create nested state directory")

	// Verify directory was created
	assert.DirExists(t, stateDir)

	// Verify file was created
	assert.FileExists(t, filepath.Join(stateDir, NoticeFileName))
}

func TestMarkNoticeShown(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()

	err := MarkNoticeShown(stateDir)
	require.NoError(t, err)

	// Verify state was saved
	state, err := LoadNoticeState(stateDir)
	require.NoError(t, err)
	assert.True(t, state.NoticeShown, "NoticeShown should be true after MarkNoticeShown")
	assert.False(t, state.ShownAt.IsZero(), "ShownAt should be set")
}

func TestShouldShowNotice(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		setup            func(t *testing.T, stateDir string)
		isExplicitConfig bool
		want             bool
		wantErr          bool
		description      string
	}{
		"shows notice when file doesn't exist and using default config": {
			setup:            func(t *testing.T, stateDir string) {},
			isExplicitConfig: false,
			want:             true,
			wantErr:          false,
			description:      "Should show notice for new user using defaults",
		},
		"does not show notice when file doesn't exist but config is explicit": {
			setup:            func(t *testing.T, stateDir string) {},
			isExplicitConfig: true,
			want:             false,
			wantErr:          false,
			description:      "Explicit config should suppress notice even for new users",
		},
		"does not show notice when already shown": {
			setup: func(t *testing.T, stateDir string) {
				state := &AutoCommitNoticeState{
					NoticeShown: true,
					ShownAt:     time.Now(),
				}
				err := SaveNoticeState(stateDir, state)
				require.NoError(t, err)
			},
			isExplicitConfig: false,
			want:             false,
			wantErr:          false,
			description:      "Should not show notice twice",
		},
		"does not show notice when already shown and config is explicit": {
			setup: func(t *testing.T, stateDir string) {
				state := &AutoCommitNoticeState{
					NoticeShown: true,
					ShownAt:     time.Now(),
				}
				err := SaveNoticeState(stateDir, state)
				require.NoError(t, err)
			},
			isExplicitConfig: true,
			want:             false,
			wantErr:          false,
			description:      "Already shown + explicit config = no notice",
		},
		"shows notice when NoticeShown is false and using default config": {
			setup: func(t *testing.T, stateDir string) {
				state := &AutoCommitNoticeState{
					NoticeShown: false,
				}
				err := SaveNoticeState(stateDir, state)
				require.NoError(t, err)
			},
			isExplicitConfig: false,
			want:             true,
			wantErr:          false,
			description:      "NoticeShown=false with default config should show notice",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			stateDir := t.TempDir()
			tt.setup(t, stateDir)

			got, err := ShouldShowNotice(stateDir, tt.isExplicitConfig)

			if tt.wantErr {
				assert.Error(t, err, tt.description)
			} else {
				assert.NoError(t, err, tt.description)
				assert.Equal(t, tt.want, got, tt.description)
			}
		})
	}
}

func TestNoticeShownOnlyOnce(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()

	// First check - should show
	shouldShow, err := ShouldShowNotice(stateDir, false)
	require.NoError(t, err)
	assert.True(t, shouldShow, "First check should return true")

	// Mark as shown
	err = MarkNoticeShown(stateDir)
	require.NoError(t, err)

	// Second check - should not show
	shouldShow, err = ShouldShowNotice(stateDir, false)
	require.NoError(t, err)
	assert.False(t, shouldShow, "Second check should return false after marking shown")

	// Third check - still should not show
	shouldShow, err = ShouldShowNotice(stateDir, false)
	require.NoError(t, err)
	assert.False(t, shouldShow, "Third check should still return false")
}

func TestNoticeNotShownWithExplicitConfig(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()

	// With explicit config, should never show even for new user
	shouldShow, err := ShouldShowNotice(stateDir, true)
	require.NoError(t, err)
	assert.False(t, shouldShow, "Explicit config should suppress notice for new user")

	// Also verify with existing state file
	err = SaveNoticeState(stateDir, &AutoCommitNoticeState{NoticeShown: false})
	require.NoError(t, err)

	shouldShow, err = ShouldShowNotice(stateDir, true)
	require.NoError(t, err)
	assert.False(t, shouldShow, "Explicit config should suppress notice even with NoticeShown=false in state")
}

func TestAtomicWritePattern(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()
	state := &AutoCommitNoticeState{
		NoticeShown: true,
		ShownAt:     time.Now(),
	}

	err := SaveNoticeState(stateDir, state)
	require.NoError(t, err)

	// Verify no temp file remains
	tmpPath := filepath.Join(stateDir, NoticeFileName+".tmp")
	assert.NoFileExists(t, tmpPath, "Temp file should not exist after successful save")

	// Verify main file exists
	assert.FileExists(t, filepath.Join(stateDir, NoticeFileName))
}
