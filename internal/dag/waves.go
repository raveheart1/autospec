package dag

import (
	"fmt"
	"sort"
)

// WaveStatus represents the execution status of a wave.
type WaveStatus int

const (
	// WavePending indicates the wave has not started.
	WavePending WaveStatus = iota
	// WaveRunning indicates the wave is currently executing.
	WaveRunning
	// WaveCompleted indicates all tasks in the wave finished successfully.
	WaveCompleted
	// WavePartialFailed indicates some tasks in the wave failed.
	WavePartialFailed
)

// String returns the string representation of a WaveStatus.
func (s WaveStatus) String() string {
	switch s {
	case WavePending:
		return "Pending"
	case WaveRunning:
		return "Running"
	case WaveCompleted:
		return "Completed"
	case WavePartialFailed:
		return "PartialFailed"
	default:
		return "Unknown"
	}
}

// TaskResult represents the outcome of a single task execution.
type TaskResult struct {
	TaskID       string // Task identifier
	Success      bool   // Whether task completed successfully
	Error        error  // Error message if failed
	Duration     int64  // Execution time in milliseconds
	WorktreePath string // Path to worktree if used (empty otherwise)
}

// ExecutionWave represents a group of tasks that can execute concurrently.
type ExecutionWave struct {
	Number  int                    // Wave number (1, 2, 3...)
	TaskIDs []string               // Tasks in this wave
	Status  WaveStatus             // Wave execution status
	Results map[string]*TaskResult // Execution results per task
}

// NewExecutionWave creates a new execution wave.
func NewExecutionWave(number int, taskIDs []string) *ExecutionWave {
	return &ExecutionWave{
		Number:  number,
		TaskIDs: taskIDs,
		Status:  WavePending,
		Results: make(map[string]*TaskResult),
	}
}

// Size returns the number of tasks in the wave.
func (w *ExecutionWave) Size() int {
	return len(w.TaskIDs)
}

// IsComplete returns true if the wave has finished execution.
func (w *ExecutionWave) IsComplete() bool {
	return w.Status == WaveCompleted || w.Status == WavePartialFailed
}

// ComputeWaves calculates execution waves based on task dependencies.
// Tasks are grouped by their maximum dependency depth (level).
// Wave N contains all tasks whose longest dependency chain has length N-1.
// Returns the computed waves and stores them in the graph.
func (g *DependencyGraph) ComputeWaves() ([]ExecutionWave, error) {
	if err := g.Validate(); err != nil {
		return nil, fmt.Errorf("computing waves: %w", err)
	}

	if len(g.nodes) == 0 {
		g.waves = []ExecutionWave{}
		return g.waves, nil
	}

	// Compute depths using BFS from roots
	if err := g.computeDepths(); err != nil {
		return nil, err
	}

	// Group tasks by depth
	depthGroups := g.groupByDepth()

	// Create waves from depth groups
	g.waves = g.createWavesFromGroups(depthGroups)

	return g.waves, nil
}

// computeDepths calculates the maximum depth for each node.
// Depth is defined as the longest path from any root to the node.
// Uses Kahn's algorithm (BFS topological order) for efficiency.
func (g *DependencyGraph) computeDepths() error {
	// Initialize all depths to 0 and count incoming edges
	inDegree := make(map[string]int)
	for id, node := range g.nodes {
		node.Depth = 0
		inDegree[id] = len(node.Dependencies)
	}

	// Start with roots (nodes with no dependencies)
	queue := make([]string, 0, len(g.roots))
	queue = append(queue, g.roots...)

	// Process nodes in topological order
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		node := g.nodes[id]

		// Process all dependents (nodes that depend on this one)
		for _, depID := range node.Dependents {
			depNode := g.nodes[depID]
			// Update dependent's depth to max of current paths
			newDepth := node.Depth + 1
			if newDepth > depNode.Depth {
				depNode.Depth = newDepth
			}

			// Decrease in-degree and add to queue when all deps processed
			inDegree[depID]--
			if inDegree[depID] == 0 {
				queue = append(queue, depID)
			}
		}
	}

	return nil
}

// groupByDepth groups task IDs by their depth level.
func (g *DependencyGraph) groupByDepth() map[int][]string {
	groups := make(map[int][]string)
	for id, node := range g.nodes {
		groups[node.Depth] = append(groups[node.Depth], id)
	}
	return groups
}

// createWavesFromGroups creates ExecutionWave structs from depth groups.
func (g *DependencyGraph) createWavesFromGroups(groups map[int][]string) []ExecutionWave {
	// Get sorted depth levels
	depths := make([]int, 0, len(groups))
	for depth := range groups {
		depths = append(depths, depth)
	}
	sort.Ints(depths)

	// Create waves
	waves := make([]ExecutionWave, 0, len(depths))
	for i, depth := range depths {
		taskIDs := groups[depth]
		// Sort task IDs for consistent ordering
		sort.Strings(taskIDs)
		wave := ExecutionWave{
			Number:  i + 1, // 1-indexed wave numbers
			TaskIDs: taskIDs,
			Status:  WavePending,
			Results: make(map[string]*TaskResult),
		}
		waves = append(waves, wave)
	}

	return waves
}

// GetWaveForTask returns the wave number (1-indexed) for a given task ID.
// Returns 0 if the task is not found.
func (g *DependencyGraph) GetWaveForTask(taskID string) int {
	for _, wave := range g.waves {
		for _, id := range wave.TaskIDs {
			if id == taskID {
				return wave.Number
			}
		}
	}
	return 0
}

// GetWavesFromTask returns all waves starting from the wave containing the given task.
// Useful for resuming execution from a specific point.
func (g *DependencyGraph) GetWavesFromTask(taskID string) []ExecutionWave {
	startWave := g.GetWaveForTask(taskID)
	if startWave == 0 {
		return nil
	}

	result := make([]ExecutionWave, 0)
	for _, wave := range g.waves {
		if wave.Number >= startWave {
			result = append(result, wave)
		}
	}
	return result
}

// WaveStats returns summary statistics about the waves.
type WaveStats struct {
	TotalWaves  int // Number of waves
	TotalTasks  int // Total tasks across all waves
	MaxWaveSize int // Size of the largest wave
	MinWaveSize int // Size of the smallest wave
}

// GetWaveStats returns statistics about the computed waves.
func (g *DependencyGraph) GetWaveStats() WaveStats {
	if len(g.waves) == 0 {
		return WaveStats{}
	}

	stats := WaveStats{
		TotalWaves:  len(g.waves),
		MaxWaveSize: 0,
		MinWaveSize: g.waves[0].Size(),
	}

	for _, wave := range g.waves {
		size := wave.Size()
		stats.TotalTasks += size
		if size > stats.MaxWaveSize {
			stats.MaxWaveSize = size
		}
		if size < stats.MinWaveSize {
			stats.MinWaveSize = size
		}
	}

	return stats
}
