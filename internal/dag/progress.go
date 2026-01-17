package dag

import (
	"fmt"
	"io"
	"sync"
)

// ProgressCallback is called when progress changes occur.
// It receives the current progress stats snapshot.
type ProgressCallback func(stats ProgressStats)

// ProgressTracker tracks the execution progress of specs in a DAG run.
// It provides thread-safe updates and rendering of progress information.
type ProgressTracker struct {
	// total is the total number of specs in the DAG.
	total int
	// completed is the count of successfully completed specs.
	completed int
	// running is the count of currently executing specs.
	running int
	// failed is the count of failed specs.
	failed int
	// blocked is the count of blocked specs (waiting on failed dependencies).
	blocked int
	// callback is called when progress changes (if set).
	callback ProgressCallback
	// mu protects all counter fields and callback.
	mu sync.RWMutex
}

// NewProgressTracker creates a new ProgressTracker for the given total specs.
func NewProgressTracker(total int) *ProgressTracker {
	return &ProgressTracker{
		total: total,
	}
}

// OnChange registers a callback to be invoked when progress changes.
// The callback is called with a snapshot of current stats after each change.
// Only one callback can be registered; subsequent calls replace the previous.
func (pt *ProgressTracker) OnChange(callback ProgressCallback) {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	pt.callback = callback
}

// notifyChange calls the registered callback with current stats (must hold lock).
func (pt *ProgressTracker) notifyChange() {
	if pt.callback == nil {
		return
	}
	stats := ProgressStats{
		Total:     pt.total,
		Completed: pt.completed,
		Running:   pt.running,
		Failed:    pt.failed,
		Blocked:   pt.blocked,
		Pending:   pt.total - pt.completed - pt.running - pt.failed - pt.blocked,
	}
	pt.callback(stats)
}

// MarkRunning increments the running count.
func (pt *ProgressTracker) MarkRunning() {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	pt.running++
	pt.notifyChange()
}

// MarkCompleted decrements running and increments completed.
func (pt *ProgressTracker) MarkCompleted() {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	if pt.running > 0 {
		pt.running--
	}
	pt.completed++
	pt.notifyChange()
}

// MarkFailed decrements running and increments failed.
func (pt *ProgressTracker) MarkFailed() {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	if pt.running > 0 {
		pt.running--
	}
	pt.failed++
	pt.notifyChange()
}

// MarkBlocked increments the blocked count.
func (pt *ProgressTracker) MarkBlocked() {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	pt.blocked++
	pt.notifyChange()
}

// Render returns a formatted progress string (e.g., "2/5 specs complete").
func (pt *ProgressTracker) Render() string {
	pt.mu.RLock()
	defer pt.mu.RUnlock()
	return fmt.Sprintf("%d/%d specs complete", pt.completed, pt.total)
}

// RenderDetailed returns detailed progress with all states.
func (pt *ProgressTracker) RenderDetailed() string {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	return fmt.Sprintf(
		"%d/%d specs complete (%d running, %d failed, %d blocked)",
		pt.completed, pt.total, pt.running, pt.failed, pt.blocked,
	)
}

// Stats returns current progress statistics.
func (pt *ProgressTracker) Stats() ProgressStats {
	pt.mu.RLock()
	defer pt.mu.RUnlock()
	return ProgressStats{
		Total:     pt.total,
		Completed: pt.completed,
		Running:   pt.running,
		Failed:    pt.failed,
		Blocked:   pt.blocked,
		Pending:   pt.total - pt.completed - pt.running - pt.failed - pt.blocked,
	}
}

// ProgressStats holds a snapshot of progress statistics.
type ProgressStats struct {
	Total     int
	Completed int
	Running   int
	Failed    int
	Blocked   int
	Pending   int
}

// IsComplete returns true if all specs have finished (completed + failed + blocked).
func (ps ProgressStats) IsComplete() bool {
	finished := ps.Completed + ps.Failed + ps.Blocked
	return finished >= ps.Total
}

// SuccessRate returns the completion success rate as a percentage (0-100).
func (ps ProgressStats) SuccessRate() float64 {
	if ps.Total == 0 {
		return 100.0
	}
	return float64(ps.Completed) / float64(ps.Total) * 100.0
}

// NewProgressTrackerFromState creates a ProgressTracker initialized from DAGRun state.
func NewProgressTrackerFromState(run *DAGRun) *ProgressTracker {
	pt := &ProgressTracker{
		total: len(run.Specs),
	}

	for _, spec := range run.Specs {
		switch spec.Status {
		case SpecStatusCompleted:
			pt.completed++
		case SpecStatusRunning:
			pt.running++
		case SpecStatusFailed:
			pt.failed++
		case SpecStatusBlocked:
			pt.blocked++
		}
	}

	return pt
}

// WriteProgressCallback creates a callback that writes progress to the given writer.
// The output format is "X/Y specs complete" for simple mode, or includes details.
func WriteProgressCallback(w io.Writer, detailed bool) ProgressCallback {
	return func(stats ProgressStats) {
		if detailed {
			fmt.Fprintf(w, "Progress: %d/%d specs complete (%d running, %d failed, %d blocked)\n",
				stats.Completed, stats.Total, stats.Running, stats.Failed, stats.Blocked)
		} else {
			fmt.Fprintf(w, "Progress: %d/%d specs complete\n", stats.Completed, stats.Total)
		}
	}
}

// Render returns the progress string for the given stats.
func (ps ProgressStats) Render() string {
	return fmt.Sprintf("%d/%d specs complete", ps.Completed, ps.Total)
}

// RenderDetailed returns detailed progress string for the given stats.
func (ps ProgressStats) RenderDetailed() string {
	return fmt.Sprintf("%d/%d specs complete (%d running, %d failed, %d blocked)",
		ps.Completed, ps.Total, ps.Running, ps.Failed, ps.Blocked)
}
