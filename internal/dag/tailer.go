package dag

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// LogTailer streams new lines from a log file as they are written.
// It uses fsnotify for efficient file change detection.
type LogTailer struct {
	path    string
	watcher *fsnotify.Watcher
	mu      sync.Mutex
	closed  bool
}

// NewLogTailer creates a new LogTailer for the given file path.
// The file does not need to exist yet; the tailer will wait for creation.
func NewLogTailer(path string) (*LogTailer, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("creating fsnotify watcher: %w", err)
	}

	return &LogTailer{
		path:    path,
		watcher: watcher,
	}, nil
}

// Tail streams lines from the log file.
// Returns a channel that receives new lines as they are written.
// The channel is closed when the context is cancelled or Close is called.
// If follow is false, dumps existing content and returns immediately.
func (t *LogTailer) Tail(ctx context.Context, follow bool) (<-chan string, error) {
	lines := make(chan string, 100)

	go t.tailLoop(ctx, lines, follow)

	return lines, nil
}

// tailLoop is the main loop for streaming file content.
func (t *LogTailer) tailLoop(ctx context.Context, lines chan<- string, follow bool) {
	defer close(lines)

	// Wait for file to exist if needed
	if err := t.waitForFile(ctx); err != nil {
		return
	}

	// Initial read of existing content
	offset, err := t.readExistingContent(ctx, lines)
	if err != nil {
		return
	}

	// If not following, we're done after dumping content
	if !follow {
		return
	}

	// Watch for changes and stream new content
	t.streamNewContent(ctx, lines, offset)
}

// waitForFile waits until the log file exists.
// Returns nil when file exists, or an error if context is cancelled.
func (t *LogTailer) waitForFile(ctx context.Context) error {
	// Check if file already exists
	if _, err := os.Stat(t.path); err == nil {
		return nil
	}

	// Watch parent directory for file creation
	parentDir := filepath.Dir(t.path)
	if err := t.ensureParentAndWatch(parentDir); err != nil {
		return err
	}

	return t.pollForFileCreation(ctx)
}

// ensureParentAndWatch creates parent directory if needed and starts watching.
func (t *LogTailer) ensureParentAndWatch(parentDir string) error {
	if err := os.MkdirAll(parentDir, 0o755); err != nil {
		return fmt.Errorf("creating parent directory: %w", err)
	}

	if err := t.watcher.Add(parentDir); err != nil {
		return fmt.Errorf("watching parent directory: %w", err)
	}

	return nil
}

// pollForFileCreation polls for file creation events.
func (t *LogTailer) pollForFileCreation(ctx context.Context) error {
	// Also poll periodically in case we miss events
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case event, ok := <-t.watcher.Events:
			if !ok {
				return fmt.Errorf("watcher closed")
			}
			if event.Name == t.path && (event.Has(fsnotify.Create) || event.Has(fsnotify.Write)) {
				return nil
			}
		case <-ticker.C:
			if _, err := os.Stat(t.path); err == nil {
				return nil
			}
		case err, ok := <-t.watcher.Errors:
			if !ok {
				return fmt.Errorf("watcher closed")
			}
			return fmt.Errorf("watcher error: %w", err)
		}
	}
}

// readExistingContent reads all existing content from the file.
// Returns the ending offset for subsequent streaming.
func (t *LogTailer) readExistingContent(ctx context.Context, lines chan<- string) (int64, error) {
	file, err := os.Open(t.path)
	if err != nil {
		return 0, fmt.Errorf("opening log file: %w", err)
	}
	defer file.Close()

	return t.scanAndSendLines(ctx, file, lines)
}

// scanAndSendLines reads lines from a reader and sends to channel.
func (t *LogTailer) scanAndSendLines(ctx context.Context, r io.Reader, lines chan<- string) (int64, error) {
	scanner := bufio.NewScanner(r)
	var offset int64

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return offset, ctx.Err()
		case lines <- scanner.Text():
			offset += int64(len(scanner.Bytes())) + 1 // +1 for newline
		}
	}

	if err := scanner.Err(); err != nil {
		return offset, fmt.Errorf("scanning log file: %w", err)
	}

	return offset, nil
}

// streamNewContent watches for file changes and streams new lines.
func (t *LogTailer) streamNewContent(ctx context.Context, lines chan<- string, offset int64) {
	// Watch the file itself for changes
	if err := t.watcher.Add(t.path); err != nil {
		return
	}

	// Poll periodically as backup for missed events
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-t.watcher.Events:
			if !ok {
				return
			}
			offset = t.handleFileEvent(ctx, event, lines, offset)
		case <-ticker.C:
			offset = t.readNewLines(ctx, lines, offset)
		case _, ok := <-t.watcher.Errors:
			if !ok {
				return
			}
			// Continue on errors, polling will handle reads
		}
	}
}

// handleFileEvent handles a file system event and returns new offset.
func (t *LogTailer) handleFileEvent(ctx context.Context, event fsnotify.Event, lines chan<- string, offset int64) int64 {
	if event.Name != t.path {
		return offset
	}

	// Handle truncation by resetting offset
	if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) {
		return t.readNewLines(ctx, lines, offset)
	}

	return offset
}

// readNewLines reads any new content from the file starting at offset.
// Handles truncation by resetting offset if file is smaller than offset.
func (t *LogTailer) readNewLines(ctx context.Context, lines chan<- string, offset int64) int64 {
	file, err := os.Open(t.path)
	if err != nil {
		return offset
	}
	defer file.Close()

	// Check for truncation
	info, err := file.Stat()
	if err != nil {
		return offset
	}

	if info.Size() < offset {
		// File was truncated, reset to beginning
		offset = 0
	}

	// Seek to offset
	if _, err := file.Seek(offset, io.SeekStart); err != nil {
		return offset
	}

	newOffset, _ := t.scanAndSendLines(ctx, file, lines)
	return offset + newOffset
}

// Close stops the tailer and releases resources.
func (t *LogTailer) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		return nil
	}
	t.closed = true

	if t.watcher != nil {
		return t.watcher.Close()
	}
	return nil
}

// Path returns the path being tailed.
func (t *LogTailer) Path() string {
	return t.path
}
