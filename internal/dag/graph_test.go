package dag

import (
	"strings"
	"testing"

	"github.com/ariel-frischer/autospec/internal/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTaskStatus_String(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		status TaskStatus
		want   string
	}{
		"pending":   {status: StatusPending, want: "Pending"},
		"running":   {status: StatusRunning, want: "Running"},
		"completed": {status: StatusCompleted, want: "Completed"},
		"failed":    {status: StatusFailed, want: "Failed"},
		"skipped":   {status: StatusSkipped, want: "Skipped"},
		"unknown":   {status: TaskStatus(99), want: "Unknown"},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.status.String())
		})
	}
}

func TestNewDependencyGraph(t *testing.T) {
	t.Parallel()

	g := NewDependencyGraph()
	assert.NotNil(t, g)
	assert.NotNil(t, g.nodes)
	assert.Empty(t, g.roots)
	assert.Empty(t, g.waves)
	assert.Equal(t, 0, g.Size())
}

func TestDependencyGraph_AddTask(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		setup   func(*DependencyGraph)
		id      string
		deps    []string
		wantErr bool
		errMsg  string
	}{
		"add single task no deps": {
			setup:   func(g *DependencyGraph) {},
			id:      "T001",
			deps:    []string{},
			wantErr: false,
		},
		"add task with deps": {
			setup:   func(g *DependencyGraph) {},
			id:      "T002",
			deps:    []string{"T001"},
			wantErr: false,
		},
		"duplicate task ID": {
			setup: func(g *DependencyGraph) {
				_ = g.AddTask("T001", []string{})
			},
			id:      "T001",
			deps:    []string{},
			wantErr: true,
			errMsg:  "duplicate task ID",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			g := NewDependencyGraph()
			tt.setup(g)

			err := g.AddTask(tt.id, tt.deps)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
				node := g.GetNode(tt.id)
				assert.NotNil(t, node)
				assert.Equal(t, tt.id, node.ID)
				assert.Equal(t, tt.deps, node.Dependencies)
			}
		})
	}
}

func TestBuildFromTasks(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		tasks     []validation.TaskItem
		wantErr   bool
		errMsg    string
		wantSize  int
		wantRoots []string
	}{
		"empty task list": {
			tasks:     []validation.TaskItem{},
			wantErr:   false,
			wantSize:  0,
			wantRoots: []string{},
		},
		"single task no deps": {
			tasks: []validation.TaskItem{
				{ID: "T001", Dependencies: []string{}},
			},
			wantErr:   false,
			wantSize:  1,
			wantRoots: []string{"T001"},
		},
		"linear chain": {
			tasks: []validation.TaskItem{
				{ID: "T001", Dependencies: []string{}},
				{ID: "T002", Dependencies: []string{"T001"}},
				{ID: "T003", Dependencies: []string{"T002"}},
			},
			wantErr:   false,
			wantSize:  3,
			wantRoots: []string{"T001"},
		},
		"diamond pattern": {
			tasks: []validation.TaskItem{
				{ID: "T001", Dependencies: []string{}},
				{ID: "T002", Dependencies: []string{"T001"}},
				{ID: "T003", Dependencies: []string{"T001"}},
				{ID: "T004", Dependencies: []string{"T002", "T003"}},
			},
			wantErr:   false,
			wantSize:  4,
			wantRoots: []string{"T001"},
		},
		"multiple roots": {
			tasks: []validation.TaskItem{
				{ID: "T001", Dependencies: []string{}},
				{ID: "T002", Dependencies: []string{}},
				{ID: "T003", Dependencies: []string{"T001", "T002"}},
			},
			wantErr:  false,
			wantSize: 3,
			// wantRoots not checked for order
		},
		"invalid dependency": {
			tasks: []validation.TaskItem{
				{ID: "T001", Dependencies: []string{"T999"}},
			},
			wantErr: true,
			errMsg:  "non-existent task",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			g, err := BuildFromTasks(tt.tasks)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, g)
				assert.Equal(t, tt.wantSize, g.Size())

				if len(tt.wantRoots) == 1 {
					assert.Contains(t, g.Roots(), tt.wantRoots[0])
				}
			}
		})
	}
}

func TestDependencyGraph_DetectCycle(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		tasks   []validation.TaskItem
		wantErr bool
		errMsg  string
	}{
		"no cycle - empty graph": {
			tasks:   []validation.TaskItem{},
			wantErr: false,
		},
		"no cycle - single task": {
			tasks: []validation.TaskItem{
				{ID: "T001", Dependencies: []string{}},
			},
			wantErr: false,
		},
		"no cycle - linear chain": {
			tasks: []validation.TaskItem{
				{ID: "T001", Dependencies: []string{}},
				{ID: "T002", Dependencies: []string{"T001"}},
				{ID: "T003", Dependencies: []string{"T002"}},
			},
			wantErr: false,
		},
		"no cycle - diamond pattern": {
			tasks: []validation.TaskItem{
				{ID: "T001", Dependencies: []string{}},
				{ID: "T002", Dependencies: []string{"T001"}},
				{ID: "T003", Dependencies: []string{"T001"}},
				{ID: "T004", Dependencies: []string{"T002", "T003"}},
			},
			wantErr: false,
		},
		"simple cycle - two nodes": {
			tasks: []validation.TaskItem{
				{ID: "T001", Dependencies: []string{"T002"}},
				{ID: "T002", Dependencies: []string{"T001"}},
			},
			wantErr: true,
			errMsg:  "circular dependency detected",
		},
		"complex cycle - three nodes": {
			tasks: []validation.TaskItem{
				{ID: "T001", Dependencies: []string{"T003"}},
				{ID: "T002", Dependencies: []string{"T001"}},
				{ID: "T003", Dependencies: []string{"T002"}},
			},
			wantErr: true,
			errMsg:  "circular dependency detected",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			g, err := buildGraphWithoutValidation(tt.tasks)
			require.NoError(t, err)

			err = g.DetectCycle()

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestDependencyGraph_Validate(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		tasks   []validation.TaskItem
		wantErr bool
		errMsg  string
	}{
		"valid empty graph": {
			tasks:   []validation.TaskItem{},
			wantErr: false,
		},
		"valid single root": {
			tasks: []validation.TaskItem{
				{ID: "T001", Dependencies: []string{}},
			},
			wantErr: false,
		},
		"valid diamond": {
			tasks: []validation.TaskItem{
				{ID: "T001", Dependencies: []string{}},
				{ID: "T002", Dependencies: []string{"T001"}},
				{ID: "T003", Dependencies: []string{"T001"}},
				{ID: "T004", Dependencies: []string{"T002", "T003"}},
			},
			wantErr: false,
		},
		"invalid - cycle detected": {
			tasks: []validation.TaskItem{
				{ID: "T001", Dependencies: []string{"T002"}},
				{ID: "T002", Dependencies: []string{"T001"}},
			},
			wantErr: true,
			errMsg:  "circular dependency",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			g, err := buildGraphWithoutValidation(tt.tasks)
			require.NoError(t, err)

			err = g.Validate()

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestDependencyGraph_SetNodeStatus(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		taskID  string
		status  TaskStatus
		wantErr bool
		errMsg  string
	}{
		"set pending to running": {
			taskID:  "T001",
			status:  StatusRunning,
			wantErr: false,
		},
		"set running to completed": {
			taskID:  "T001",
			status:  StatusCompleted,
			wantErr: false,
		},
		"task not found": {
			taskID:  "T999",
			status:  StatusRunning,
			wantErr: true,
			errMsg:  "not found",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			g, err := BuildFromTasks([]validation.TaskItem{
				{ID: "T001", Dependencies: []string{}},
			})
			require.NoError(t, err)

			err = g.SetNodeStatus(tt.taskID, tt.status)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
				node := g.GetNode(tt.taskID)
				assert.Equal(t, tt.status, node.Status)
			}
		})
	}
}

func TestDependencyGraph_Dependents(t *testing.T) {
	t.Parallel()

	tasks := []validation.TaskItem{
		{ID: "T001", Dependencies: []string{}},
		{ID: "T002", Dependencies: []string{"T001"}},
		{ID: "T003", Dependencies: []string{"T001"}},
		{ID: "T004", Dependencies: []string{"T002", "T003"}},
	}

	g, err := BuildFromTasks(tasks)
	require.NoError(t, err)

	tests := map[string]struct {
		taskID       string
		wantDeps     []string
		wantChildren []string
	}{
		"root task": {
			taskID:       "T001",
			wantDeps:     []string{},
			wantChildren: []string{"T002", "T003"},
		},
		"middle task T002": {
			taskID:       "T002",
			wantDeps:     []string{"T001"},
			wantChildren: []string{"T004"},
		},
		"leaf task": {
			taskID:       "T004",
			wantDeps:     []string{"T002", "T003"},
			wantChildren: []string{},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			node := g.GetNode(tt.taskID)
			require.NotNil(t, node)

			assert.ElementsMatch(t, tt.wantDeps, node.Dependencies)
			assert.ElementsMatch(t, tt.wantChildren, node.Dependents)
		})
	}
}

// buildGraphWithoutValidation builds a graph without running full validation.
// This allows testing cycle detection independently.
func buildGraphWithoutValidation(tasks []validation.TaskItem) (*DependencyGraph, error) {
	g := NewDependencyGraph()

	// Add all tasks first
	for i := range tasks {
		task := &tasks[i]
		if err := g.AddTask(task.ID, task.Dependencies); err != nil {
			return nil, err
		}
		g.nodes[task.ID].Task = task
	}

	// Build dependents (skip validation of missing deps for cycle tests)
	for id, node := range g.nodes {
		for _, depID := range node.Dependencies {
			if depNode, exists := g.nodes[depID]; exists {
				depNode.Dependents = append(depNode.Dependents, id)
			}
		}
	}

	// Identify roots
	for id, node := range g.nodes {
		allDepsExist := true
		for _, depID := range node.Dependencies {
			if _, exists := g.nodes[depID]; !exists {
				allDepsExist = false
				break
			}
		}
		if len(node.Dependencies) == 0 || !allDepsExist {
			hasValidDep := false
			for _, depID := range node.Dependencies {
				if _, exists := g.nodes[depID]; exists {
					hasValidDep = true
					break
				}
			}
			if !hasValidDep {
				g.roots = append(g.roots, id)
			}
		}
	}

	return g, nil
}

func TestCycleErrorFormat(t *testing.T) {
	t.Parallel()

	tasks := []validation.TaskItem{
		{ID: "T001", Dependencies: []string{"T002"}},
		{ID: "T002", Dependencies: []string{"T001"}},
	}

	g, err := buildGraphWithoutValidation(tasks)
	require.NoError(t, err)

	err = g.DetectCycle()
	require.Error(t, err)

	// Check format: "circular dependency detected: T001 -> T002 -> T001"
	errStr := err.Error()
	assert.True(t, strings.HasPrefix(errStr, "circular dependency detected:"))
	assert.Contains(t, errStr, "->")
}
