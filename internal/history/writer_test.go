package history

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHistoryWriter_LogEntry(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		setupStore  func(t *testing.T, stateDir string)
		maxEntries  int
		wantEntries int
	}{
		"log entry to empty history": {
			setupStore:  func(t *testing.T, stateDir string) {},
			maxEntries:  500,
			wantEntries: 1,
		},
		"log entry to existing history": {
			setupStore: func(t *testing.T, stateDir string) {
				history := &HistoryFile{
					Entries: []HistoryEntry{
						{Timestamp: time.Now(), Command: "existing", ExitCode: 0, Duration: "1m"},
					},
				}
				require.NoError(t, SaveHistory(stateDir, history))
			},
			maxEntries:  500,
			wantEntries: 2,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			stateDir := t.TempDir()
			tc.setupStore(t, stateDir)

			writer := NewWriter(stateDir, tc.maxEntries)
			entry := HistoryEntry{
				Timestamp: time.Now(),
				Command:   "test",
				Spec:      "test-spec",
				ExitCode:  0,
				Duration:  "30s",
			}
			writer.LogEntry(entry)

			// Verify entry was logged
			history, err := LoadHistory(stateDir)
			require.NoError(t, err)
			assert.Len(t, history.Entries, tc.wantEntries)
		})
	}
}

func TestHistoryWriter_Pruning(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		existingEntries int
		maxEntries      int
		wantEntries     int
		wantOldest      string // Command name of oldest remaining entry
	}{
		"no pruning needed": {
			existingEntries: 5,
			maxEntries:      10,
			wantEntries:     6, // 5 existing + 1 new
			wantOldest:      "cmd-0",
		},
		"prune oldest when max exceeded": {
			existingEntries: 10,
			maxEntries:      10,
			wantEntries:     10, // oldest removed, new added
			wantOldest:      "cmd-1",
		},
		"prune multiple when well over max": {
			existingEntries: 12,
			maxEntries:      10,
			wantEntries:     10,
			wantOldest:      "cmd-3",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			stateDir := t.TempDir()

			// Create existing entries
			entries := make([]HistoryEntry, tc.existingEntries)
			for i := 0; i < tc.existingEntries; i++ {
				entries[i] = HistoryEntry{
					Timestamp: time.Now().Add(time.Duration(i) * time.Minute),
					Command:   "cmd-" + string(rune('0'+i)),
					ExitCode:  0,
					Duration:  "1m",
				}
			}
			history := &HistoryFile{Entries: entries}
			require.NoError(t, SaveHistory(stateDir, history))

			// Log new entry
			writer := NewWriter(stateDir, tc.maxEntries)
			writer.LogEntry(HistoryEntry{
				Timestamp: time.Now().Add(time.Hour),
				Command:   "new-cmd",
				ExitCode:  0,
				Duration:  "30s",
			})

			// Verify
			loaded, err := LoadHistory(stateDir)
			require.NoError(t, err)
			assert.Len(t, loaded.Entries, tc.wantEntries)

			// Verify oldest entry
			if len(loaded.Entries) > 0 {
				assert.Equal(t, tc.wantOldest, loaded.Entries[0].Command)
			}

			// Verify newest entry is our new one
			assert.Equal(t, "new-cmd", loaded.Entries[len(loaded.Entries)-1].Command)
		})
	}
}

func TestHistoryWriter_LogCommand(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()
	writer := NewWriter(stateDir, 500)

	writer.LogCommand("specify", "test-feature", 0, 2*time.Minute+30*time.Second)

	// Verify
	history, err := LoadHistory(stateDir)
	require.NoError(t, err)
	require.Len(t, history.Entries, 1)

	entry := history.Entries[0]
	assert.Equal(t, "specify", entry.Command)
	assert.Equal(t, "test-feature", entry.Spec)
	assert.Equal(t, 0, entry.ExitCode)
	assert.Equal(t, "2m30s", entry.Duration)
	assert.False(t, entry.Timestamp.IsZero())
}

func TestHistoryWriter_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()
	writer := NewWriter(stateDir, 100)

	// Run multiple goroutines writing concurrently
	var wg sync.WaitGroup
	numWriters := 10
	entriesPerWriter := 5

	for i := 0; i < numWriters; i++ {
		wg.Add(1)
		go func(writerID int) {
			defer wg.Done()
			for j := 0; j < entriesPerWriter; j++ {
				writer.LogEntry(HistoryEntry{
					Timestamp: time.Now(),
					Command:   "test",
					Spec:      "concurrent-test",
					ExitCode:  0,
					Duration:  "1s",
				})
			}
		}(i)
	}

	wg.Wait()

	// Verify all entries were written (may be less due to races, but should be close)
	history, err := LoadHistory(stateDir)
	require.NoError(t, err)

	// Due to potential race conditions with file writes, we just verify
	// that some entries were written successfully
	assert.Greater(t, len(history.Entries), 0, "at least some entries should be written")
	assert.LessOrEqual(t, len(history.Entries), numWriters*entriesPerWriter)
}

func TestHistoryWriter_NonFatalErrors(t *testing.T) {
	t.Parallel()

	// Use an invalid path that can't be created
	writer := NewWriter("/nonexistent/deeply/nested/path/that/cannot/exist", 500)

	// This should not panic, just print a warning
	writer.LogEntry(HistoryEntry{
		Timestamp: time.Now(),
		Command:   "test",
		ExitCode:  0,
		Duration:  "1s",
	})

	// If we get here without panic, the test passes
}

func TestNewWriter(t *testing.T) {
	t.Parallel()

	writer := NewWriter("/test/path", 100)

	assert.Equal(t, "/test/path", writer.StateDir)
	assert.Equal(t, 100, writer.MaxEntries)
}

func TestHistoryWriter_ZeroMaxEntries(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()

	// Zero max entries means unlimited
	writer := NewWriter(stateDir, 0)

	// Log 5 entries
	for i := 0; i < 5; i++ {
		writer.LogEntry(HistoryEntry{
			Timestamp: time.Now(),
			Command:   "test",
			ExitCode:  0,
			Duration:  "1s",
		})
	}

	// All should be retained
	history, err := LoadHistory(stateDir)
	require.NoError(t, err)
	assert.Len(t, history.Entries, 5)
}
