package dag

import (
	"fmt"
	"sync"
)

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
	// mu protects all counter fields.
	mu sync.RWMutex
}

// NewProgressTracker creates a new ProgressTracker for the given total specs.
func NewProgressTracker(total int) *ProgressTracker {
	return &ProgressTracker{
		total: total,
	}
}

// MarkRunning increments the running count.
func (pt *ProgressTracker) MarkRunning() {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	pt.running++
}

// MarkCompleted decrements running and increments completed.
func (pt *ProgressTracker) MarkCompleted() {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	if pt.running > 0 {
		pt.running--
	}
	pt.completed++
}

// MarkFailed decrements running and increments failed.
func (pt *ProgressTracker) MarkFailed() {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	if pt.running > 0 {
		pt.running--
	}
	pt.failed++
}

// MarkBlocked increments the blocked count.
func (pt *ProgressTracker) MarkBlocked() {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	pt.blocked++
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
