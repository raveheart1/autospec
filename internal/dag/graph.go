// Package dag provides dependency graph construction and wave computation for parallel task execution.
package dag

import (
	"fmt"
	"strings"

	"github.com/ariel-frischer/autospec/internal/validation"
)

// TaskStatus represents the execution status of a task node.
type TaskStatus int

const (
	// StatusPending indicates the task has not started.
	StatusPending TaskStatus = iota
	// StatusRunning indicates the task is currently executing.
	StatusRunning
	// StatusCompleted indicates the task finished successfully.
	StatusCompleted
	// StatusFailed indicates the task execution failed.
	StatusFailed
	// StatusSkipped indicates the task was skipped due to failed dependencies.
	StatusSkipped
)

// String returns the string representation of a TaskStatus.
func (s TaskStatus) String() string {
	switch s {
	case StatusPending:
		return "Pending"
	case StatusRunning:
		return "Running"
	case StatusCompleted:
		return "Completed"
	case StatusFailed:
		return "Failed"
	case StatusSkipped:
		return "Skipped"
	default:
		return "Unknown"
	}
}

// TaskNode represents a node in the dependency graph.
type TaskNode struct {
	ID           string               // Task identifier (T001, T002, etc.)
	Dependencies []string             // IDs of tasks this depends on
	Dependents   []string             // IDs of tasks that depend on this
	Depth        int                  // Maximum distance from any root node (determines wave)
	Status       TaskStatus           // Current execution status
	Task         *validation.TaskItem // Original task reference
}

// DependencyGraph represents a directed acyclic graph of task dependencies.
type DependencyGraph struct {
	nodes map[string]*TaskNode // Task ID to node mapping
	roots []string             // Task IDs with no dependencies (wave 1 candidates)
	waves []ExecutionWave      // Computed execution waves in order
}

// NewDependencyGraph creates an empty dependency graph.
func NewDependencyGraph() *DependencyGraph {
	return &DependencyGraph{
		nodes: make(map[string]*TaskNode),
		roots: []string{},
		waves: []ExecutionWave{},
	}
}

// AddTask adds a task with its dependencies to the graph.
// Returns an error if a dependency references a non-existent task.
func (g *DependencyGraph) AddTask(id string, deps []string) error {
	if _, exists := g.nodes[id]; exists {
		return fmt.Errorf("adding task: duplicate task ID %s", id)
	}

	node := &TaskNode{
		ID:           id,
		Dependencies: deps,
		Dependents:   []string{},
		Depth:        0,
		Status:       StatusPending,
	}

	g.nodes[id] = node
	return nil
}

// BuildFromTasks constructs a dependency graph from a list of tasks.
// Returns an error if task dependencies are invalid.
func BuildFromTasks(tasks []validation.TaskItem) (*DependencyGraph, error) {
	g := NewDependencyGraph()

	// First pass: add all tasks
	for i := range tasks {
		task := &tasks[i]
		if err := g.AddTask(task.ID, task.Dependencies); err != nil {
			return nil, fmt.Errorf("building graph: %w", err)
		}
		g.nodes[task.ID].Task = task
	}

	// Second pass: validate dependencies and build dependents
	if err := g.buildDependentsAndValidate(); err != nil {
		return nil, err
	}

	// Third pass: identify roots
	g.identifyRoots()

	return g, nil
}

// buildDependentsAndValidate validates dependencies exist and builds the dependents lists.
func (g *DependencyGraph) buildDependentsAndValidate() error {
	for id, node := range g.nodes {
		for _, depID := range node.Dependencies {
			depNode, exists := g.nodes[depID]
			if !exists {
				return fmt.Errorf("validating dependencies: task %s depends on non-existent task %s", id, depID)
			}
			depNode.Dependents = append(depNode.Dependents, id)
		}
	}
	return nil
}

// identifyRoots finds all tasks with no dependencies.
func (g *DependencyGraph) identifyRoots() {
	g.roots = []string{}
	for id, node := range g.nodes {
		if len(node.Dependencies) == 0 {
			g.roots = append(g.roots, id)
		}
	}
}

// Nodes returns the map of task nodes.
func (g *DependencyGraph) Nodes() map[string]*TaskNode {
	return g.nodes
}

// Roots returns the list of root task IDs (tasks with no dependencies).
func (g *DependencyGraph) Roots() []string {
	return g.roots
}

// Waves returns the computed execution waves.
func (g *DependencyGraph) Waves() []ExecutionWave {
	return g.waves
}

// GetNode returns a task node by ID, or nil if not found.
func (g *DependencyGraph) GetNode(id string) *TaskNode {
	return g.nodes[id]
}

// Size returns the number of tasks in the graph.
func (g *DependencyGraph) Size() int {
	return len(g.nodes)
}

// DetectCycle checks for circular dependencies in the graph.
// Returns an error with the cycle path if found, nil otherwise.
// Uses DFS-based cycle detection algorithm.
func (g *DependencyGraph) DetectCycle() error {
	visited := make(map[string]bool)
	recStack := make(map[string]bool)
	path := make([]string, 0)

	for id := range g.nodes {
		if !visited[id] {
			if cycle := g.detectCycleDFS(id, visited, recStack, path); cycle != nil {
				return fmt.Errorf("circular dependency detected: %s", strings.Join(cycle, " -> "))
			}
		}
	}
	return nil
}

// detectCycleDFS performs depth-first search for cycle detection.
// Returns the cycle path if found, nil otherwise.
func (g *DependencyGraph) detectCycleDFS(id string, visited, recStack map[string]bool, path []string) []string {
	visited[id] = true
	recStack[id] = true
	path = append(path, id)

	node := g.nodes[id]
	for _, depID := range node.Dependencies {
		if !visited[depID] {
			if cycle := g.detectCycleDFS(depID, visited, recStack, path); cycle != nil {
				return cycle
			}
		} else if recStack[depID] {
			// Found a cycle - build the cycle path
			return g.buildCyclePath(path, depID)
		}
	}

	recStack[id] = false
	return nil
}

// buildCyclePath constructs the cycle path from the DFS path.
func (g *DependencyGraph) buildCyclePath(path []string, cycleStart string) []string {
	startIdx := -1
	for i, id := range path {
		if id == cycleStart {
			startIdx = i
			break
		}
	}
	if startIdx >= 0 {
		cycle := append(path[startIdx:], cycleStart)
		return cycle
	}
	return append(path, cycleStart)
}

// Validate performs full validation of the graph including cycle detection.
// Returns an error if any validation fails.
func (g *DependencyGraph) Validate() error {
	if len(g.nodes) == 0 {
		return nil // Empty graph is valid
	}

	// Check for cycles first (a cycle means no valid roots)
	if err := g.DetectCycle(); err != nil {
		return err
	}

	if len(g.roots) == 0 {
		return fmt.Errorf("validating graph: no root tasks found (all tasks have dependencies)")
	}

	return nil
}

// SetNodeStatus updates the status of a task node.
// Returns an error if the task ID is not found.
func (g *DependencyGraph) SetNodeStatus(id string, status TaskStatus) error {
	node := g.nodes[id]
	if node == nil {
		return fmt.Errorf("setting node status: task %s not found", id)
	}
	node.Status = status
	return nil
}
