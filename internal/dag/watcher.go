package dag

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/fatih/color"
	"golang.org/x/term"
)

// Watcher provides live-updating status table display for DAG runs.
// It manages terminal state, renders spec status tables, and handles
// keyboard input for clean exit.
type Watcher struct {
	stateDir     string
	runID        string
	interval     time.Duration
	out          io.Writer
	mu           sync.Mutex
	lastRowCount int
	oldState     *term.State
	isRawMode    bool
	stdinFd      int
}

// WatcherOption configures a Watcher.
type WatcherOption func(*Watcher)

// WithOutput sets the output writer for the watcher.
func WithOutput(w io.Writer) WatcherOption {
	return func(watcher *Watcher) {
		watcher.out = w
	}
}

// WithInterval sets the refresh interval for the watcher.
func WithInterval(d time.Duration) WatcherOption {
	return func(watcher *Watcher) {
		watcher.interval = d
	}
}

// NewWatcher creates a new Watcher for monitoring a DAG run.
func NewWatcher(stateDir, runID string, opts ...WatcherOption) *Watcher {
	w := &Watcher{
		stateDir: stateDir,
		runID:    runID,
		interval: 2 * time.Second,
		out:      os.Stdout,
		stdinFd:  int(os.Stdin.Fd()),
	}

	for _, opt := range opts {
		opt(w)
	}

	return w
}

// Watch starts the live-updating status table display.
// Returns when the user presses 'q', Ctrl+C, or the context is cancelled.
func (w *Watcher) Watch(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Set up signal handling
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	// Start keyboard listener
	keyCh := w.startKeyboardListener(ctx)

	// Initial render
	if err := w.renderTable(); err != nil {
		return fmt.Errorf("initial render: %w", err)
	}

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	return w.watchLoop(ctx, ticker.C, keyCh, sigCh)
}

// watchLoop is the main event loop for the watcher.
func (w *Watcher) watchLoop(ctx context.Context, tickCh <-chan time.Time, keyCh <-chan byte, sigCh <-chan os.Signal) error {
	for {
		select {
		case <-ctx.Done():
			w.restoreTerminal()
			return nil
		case <-sigCh:
			w.restoreTerminal()
			return nil
		case key := <-keyCh:
			if key == 'q' || key == 'Q' || key == 3 { // 3 = Ctrl+C
				w.restoreTerminal()
				return nil
			}
		case <-tickCh:
			if err := w.renderTable(); err != nil {
				return fmt.Errorf("rendering table: %w", err)
			}
		}
	}
}

// renderTable renders the current state as a table.
func (w *Watcher) renderTable() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	run, err := LoadState(w.stateDir, w.runID)
	if err != nil {
		return fmt.Errorf("loading state: %w", err)
	}
	if run == nil {
		return fmt.Errorf("run not found: %s", w.runID)
	}

	// Clear previous output by moving cursor up
	w.clearPreviousOutput()

	// Render new table
	lines := w.buildTableLines(run)
	for _, line := range lines {
		fmt.Fprintln(w.out, line)
	}
	w.lastRowCount = len(lines)

	return nil
}

// clearPreviousOutput moves cursor up and clears previous table.
func (w *Watcher) clearPreviousOutput() {
	if w.lastRowCount > 0 {
		// Move cursor up and clear each line
		for range w.lastRowCount {
			fmt.Fprint(w.out, "\033[1A\033[2K")
		}
	}
}

// buildTableLines constructs the table output lines.
func (w *Watcher) buildTableLines(run *DAGRun) []string {
	lines := []string{}

	// Header with run info
	lines = append(lines, fmt.Sprintf("DAG Run: %s  Status: %s", run.RunID, formatRunStatus(run.Status)))
	lines = append(lines, "")

	// Table header
	header := fmt.Sprintf("%-25s  %-12s  %-12s  %-12s  %-20s", "SPEC", "STATUS", "PROGRESS", "DURATION", "LAST UPDATE")
	lines = append(lines, header)
	lines = append(lines, strings.Repeat("-", 85))

	// Spec rows
	for specID, spec := range run.Specs {
		row := w.formatSpecRow(specID, spec)
		lines = append(lines, row)
	}

	// Footer
	lines = append(lines, "")
	lines = append(lines, "Press 'q' to exit")

	return lines
}

// formatSpecRow formats a single spec row for the table.
func (w *Watcher) formatSpecRow(specID string, spec *SpecState) string {
	status := formatSpecStatus(spec.Status)
	progress := formatProgress(spec)
	duration := formatDuration(spec)
	lastUpdate := formatLastUpdate(spec)

	return fmt.Sprintf("%-25s  %-12s  %-12s  %-12s  %-20s", specID, status, progress, duration, lastUpdate)
}

// formatRunStatus formats the run status with color.
func formatRunStatus(status RunStatus) string {
	switch status {
	case RunStatusRunning:
		return color.CyanString("running")
	case RunStatusCompleted:
		return color.GreenString("completed")
	case RunStatusFailed:
		return color.RedString("failed")
	case RunStatusInterrupted:
		return color.YellowString("interrupted")
	default:
		return string(status)
	}
}

// formatSpecStatus formats the spec status with color.
func formatSpecStatus(status SpecStatus) string {
	switch status {
	case SpecStatusPending:
		return color.WhiteString("pending")
	case SpecStatusRunning:
		return color.CyanString("running")
	case SpecStatusCompleted:
		return color.GreenString("completed")
	case SpecStatusFailed:
		return color.RedString("failed")
	case SpecStatusBlocked:
		return color.YellowString("blocked")
	default:
		return string(status)
	}
}

// formatProgress formats the task progress.
func formatProgress(spec *SpecState) string {
	if spec.CurrentTask != "" {
		return spec.CurrentTask
	}
	if spec.CurrentStage != "" {
		return spec.CurrentStage
	}
	return "-"
}

// formatDuration formats the duration since spec started.
func formatDuration(spec *SpecState) string {
	if spec.StartedAt == nil {
		return "-"
	}

	end := time.Now()
	if spec.CompletedAt != nil {
		end = *spec.CompletedAt
	}

	d := end.Sub(*spec.StartedAt)
	return formatDurationHuman(d)
}

// formatDurationHuman formats a duration in human-readable form.
func formatDurationHuman(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm %ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%dh %dm", int(d.Hours()), int(d.Minutes())%60)
}

// formatLastUpdate formats the last update timestamp.
func formatLastUpdate(spec *SpecState) string {
	var lastTime *time.Time

	if spec.CompletedAt != nil {
		lastTime = spec.CompletedAt
	} else if spec.StartedAt != nil {
		lastTime = spec.StartedAt
	}

	if lastTime == nil {
		return "-"
	}

	return formatRelativeTime(*lastTime)
}

// formatRelativeTime formats a time as relative to now.
func formatRelativeTime(t time.Time) string {
	d := time.Since(t)

	if d < time.Minute {
		return "just now"
	}
	if d < time.Hour {
		mins := int(d.Minutes())
		if mins == 1 {
			return "1 min ago"
		}
		return fmt.Sprintf("%d mins ago", mins)
	}
	if d < 24*time.Hour {
		hours := int(d.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	}

	return t.Format("2006-01-02 15:04")
}

// startKeyboardListener starts a goroutine that listens for keyboard input.
// Returns a channel that receives key presses.
func (w *Watcher) startKeyboardListener(ctx context.Context) <-chan byte {
	keyCh := make(chan byte, 1)

	// Only enable raw mode if stdout is a terminal
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		return keyCh
	}

	go w.keyboardLoop(ctx, keyCh)

	return keyCh
}

// keyboardLoop reads keyboard input in raw mode.
func (w *Watcher) keyboardLoop(ctx context.Context, keyCh chan<- byte) {
	// Put terminal in raw mode for immediate key detection
	oldState, err := term.MakeRaw(w.stdinFd)
	if err != nil {
		return
	}

	w.mu.Lock()
	w.oldState = oldState
	w.isRawMode = true
	w.mu.Unlock()

	buf := make([]byte, 1)
	for {
		select {
		case <-ctx.Done():
			return
		default:
			// Read with small timeout to check context periodically
			n, err := os.Stdin.Read(buf)
			if err != nil || n == 0 {
				continue
			}
			select {
			case keyCh <- buf[0]:
			default:
			}
		}
	}
}

// restoreTerminal restores the terminal to its original state.
func (w *Watcher) restoreTerminal() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.isRawMode && w.oldState != nil {
		term.Restore(w.stdinFd, w.oldState)
		w.isRawMode = false
	}
}
