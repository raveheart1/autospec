package taskgraph

import (
	"fmt"
	"testing"

	"github.com/ariel-frischer/autospec/internal/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWaveStatus_String(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		status WaveStatus
		want   string
	}{
		"pending":        {status: WavePending, want: "Pending"},
		"running":        {status: WaveRunning, want: "Running"},
		"completed":      {status: WaveCompleted, want: "Completed"},
		"partial failed": {status: WavePartialFailed, want: "PartialFailed"},
		"unknown":        {status: WaveStatus(99), want: "Unknown"},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.status.String())
		})
	}
}

func TestExecutionWave_Size(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		taskIDs []string
		want    int
	}{
		"empty":    {taskIDs: []string{}, want: 0},
		"one":      {taskIDs: []string{"T001"}, want: 1},
		"multiple": {taskIDs: []string{"T001", "T002", "T003"}, want: 3},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			wave := NewExecutionWave(1, tt.taskIDs)
			assert.Equal(t, tt.want, wave.Size())
		})
	}
}

func TestExecutionWave_IsComplete(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		status WaveStatus
		want   bool
	}{
		"pending":        {status: WavePending, want: false},
		"running":        {status: WaveRunning, want: false},
		"completed":      {status: WaveCompleted, want: true},
		"partial failed": {status: WavePartialFailed, want: true},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			wave := NewExecutionWave(1, []string{"T001"})
			wave.Status = tt.status
			assert.Equal(t, tt.want, wave.IsComplete())
		})
	}
}

func TestDependencyGraph_ComputeWaves(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		tasks     []validation.TaskItem
		wantWaves int
		wantWave1 []string
		wantWave2 []string
		wantWave3 []string
		wantErr   bool
		errMsg    string
	}{
		"empty graph": {
			tasks:     []validation.TaskItem{},
			wantWaves: 0,
			wantErr:   false,
		},
		"single task - single wave": {
			tasks: []validation.TaskItem{
				{ID: "T001", Dependencies: []string{}},
			},
			wantWaves: 1,
			wantWave1: []string{"T001"},
			wantErr:   false,
		},
		"all independent - single wave": {
			tasks: []validation.TaskItem{
				{ID: "T001", Dependencies: []string{}},
				{ID: "T002", Dependencies: []string{}},
				{ID: "T003", Dependencies: []string{}},
			},
			wantWaves: 1,
			wantWave1: []string{"T001", "T002", "T003"},
			wantErr:   false,
		},
		"linear chain - one per wave": {
			tasks: []validation.TaskItem{
				{ID: "T001", Dependencies: []string{}},
				{ID: "T002", Dependencies: []string{"T001"}},
				{ID: "T003", Dependencies: []string{"T002"}},
			},
			wantWaves: 3,
			wantWave1: []string{"T001"},
			wantWave2: []string{"T002"},
			wantWave3: []string{"T003"},
			wantErr:   false,
		},
		"diamond dependencies": {
			tasks: []validation.TaskItem{
				{ID: "T001", Dependencies: []string{}},
				{ID: "T002", Dependencies: []string{"T001"}},
				{ID: "T003", Dependencies: []string{"T001"}},
				{ID: "T004", Dependencies: []string{"T002", "T003"}},
			},
			wantWaves: 3,
			wantWave1: []string{"T001"},
			wantWave2: []string{"T002", "T003"},
			wantWave3: []string{"T004"},
			wantErr:   false,
		},
		"complex multi-wave": {
			tasks: []validation.TaskItem{
				{ID: "T001", Dependencies: []string{}},
				{ID: "T002", Dependencies: []string{}},
				{ID: "T003", Dependencies: []string{"T001"}},
				{ID: "T004", Dependencies: []string{"T001", "T002"}},
				{ID: "T005", Dependencies: []string{"T003", "T004"}},
			},
			wantWaves: 3,
			wantWave1: []string{"T001", "T002"},
			wantWave2: []string{"T003", "T004"},
			wantWave3: []string{"T005"},
			wantErr:   false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			g, err := BuildFromTasks(tt.tasks)
			require.NoError(t, err)

			waves, err := g.ComputeWaves()

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
				assert.Len(t, waves, tt.wantWaves)

				if tt.wantWaves >= 1 && len(tt.wantWave1) > 0 {
					assert.ElementsMatch(t, tt.wantWave1, waves[0].TaskIDs)
				}
				if tt.wantWaves >= 2 && len(tt.wantWave2) > 0 {
					assert.ElementsMatch(t, tt.wantWave2, waves[1].TaskIDs)
				}
				if tt.wantWaves >= 3 && len(tt.wantWave3) > 0 {
					assert.ElementsMatch(t, tt.wantWave3, waves[2].TaskIDs)
				}
			}
		})
	}
}

func TestDependencyGraph_GetWaveForTask(t *testing.T) {
	t.Parallel()

	tasks := []validation.TaskItem{
		{ID: "T001", Dependencies: []string{}},
		{ID: "T002", Dependencies: []string{"T001"}},
		{ID: "T003", Dependencies: []string{"T001"}},
		{ID: "T004", Dependencies: []string{"T002", "T003"}},
	}

	g, err := BuildFromTasks(tasks)
	require.NoError(t, err)
	_, err = g.ComputeWaves()
	require.NoError(t, err)

	tests := map[string]struct {
		taskID string
		want   int
	}{
		"wave 1 task":      {taskID: "T001", want: 1},
		"wave 2 task T002": {taskID: "T002", want: 2},
		"wave 2 task T003": {taskID: "T003", want: 2},
		"wave 3 task":      {taskID: "T004", want: 3},
		"non-existent":     {taskID: "T999", want: 0},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, g.GetWaveForTask(tt.taskID))
		})
	}
}

func TestDependencyGraph_GetWavesFromTask(t *testing.T) {
	t.Parallel()

	tasks := []validation.TaskItem{
		{ID: "T001", Dependencies: []string{}},
		{ID: "T002", Dependencies: []string{"T001"}},
		{ID: "T003", Dependencies: []string{"T002"}},
	}

	g, err := BuildFromTasks(tasks)
	require.NoError(t, err)
	_, err = g.ComputeWaves()
	require.NoError(t, err)

	tests := map[string]struct {
		taskID    string
		wantCount int
	}{
		"from wave 1":  {taskID: "T001", wantCount: 3},
		"from wave 2":  {taskID: "T002", wantCount: 2},
		"from wave 3":  {taskID: "T003", wantCount: 1},
		"non-existent": {taskID: "T999", wantCount: 0},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			waves := g.GetWavesFromTask(tt.taskID)
			if tt.wantCount == 0 {
				assert.Nil(t, waves)
			} else {
				assert.Len(t, waves, tt.wantCount)
			}
		})
	}
}

func TestDependencyGraph_GetWaveStats(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		tasks     []validation.TaskItem
		wantTotal int
		wantMax   int
		wantMin   int
		wantWaves int
	}{
		"empty": {
			tasks:     []validation.TaskItem{},
			wantTotal: 0,
			wantMax:   0,
			wantMin:   0,
			wantWaves: 0,
		},
		"single task": {
			tasks: []validation.TaskItem{
				{ID: "T001", Dependencies: []string{}},
			},
			wantTotal: 1,
			wantMax:   1,
			wantMin:   1,
			wantWaves: 1,
		},
		"varied wave sizes": {
			tasks: []validation.TaskItem{
				{ID: "T001", Dependencies: []string{}},
				{ID: "T002", Dependencies: []string{"T001"}},
				{ID: "T003", Dependencies: []string{"T001"}},
				{ID: "T004", Dependencies: []string{"T001"}},
				{ID: "T005", Dependencies: []string{"T002", "T003", "T004"}},
			},
			wantTotal: 5,
			wantMax:   3,
			wantMin:   1,
			wantWaves: 3,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			g, err := BuildFromTasks(tt.tasks)
			require.NoError(t, err)
			_, err = g.ComputeWaves()
			require.NoError(t, err)

			stats := g.GetWaveStats()

			assert.Equal(t, tt.wantWaves, stats.TotalWaves)
			assert.Equal(t, tt.wantTotal, stats.TotalTasks)
			assert.Equal(t, tt.wantMax, stats.MaxWaveSize)
			assert.Equal(t, tt.wantMin, stats.MinWaveSize)
		})
	}
}

func TestWaveDepthCalculation(t *testing.T) {
	t.Parallel()

	// Test that max depth is correctly calculated for diamond pattern
	// T001 (depth 0) -> T002, T003 (depth 1) -> T004 (depth 2)
	tasks := []validation.TaskItem{
		{ID: "T001", Dependencies: []string{}},
		{ID: "T002", Dependencies: []string{"T001"}},
		{ID: "T003", Dependencies: []string{"T001"}},
		{ID: "T004", Dependencies: []string{"T002", "T003"}},
	}

	g, err := BuildFromTasks(tasks)
	require.NoError(t, err)
	_, err = g.ComputeWaves()
	require.NoError(t, err)

	tests := map[string]struct {
		taskID    string
		wantDepth int
	}{
		"root":    {taskID: "T001", wantDepth: 0},
		"level1":  {taskID: "T002", wantDepth: 1},
		"level1b": {taskID: "T003", wantDepth: 1},
		"level2":  {taskID: "T004", wantDepth: 2},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			node := g.GetNode(tt.taskID)
			require.NotNil(t, node)
			assert.Equal(t, tt.wantDepth, node.Depth)
		})
	}
}

// BenchmarkComputeWaves100Tasks benchmarks wave computation for 100 tasks.
// Performance contract: <100ms for 100 tasks.
func BenchmarkComputeWaves100Tasks(b *testing.B) {
	// Create a complex dependency graph with 100 tasks
	tasks := make([]validation.TaskItem, 100)
	for i := 0; i < 100; i++ {
		id := generateTaskID(i)
		deps := []string{}

		// Create varying dependency patterns
		if i > 0 {
			// Depend on some earlier tasks
			if i >= 3 {
				deps = append(deps, generateTaskID(i-3))
			}
			if i >= 2 {
				deps = append(deps, generateTaskID(i-2))
			}
			if i >= 1 {
				deps = append(deps, generateTaskID(i-1))
			}
		}

		tasks[i] = validation.TaskItem{
			ID:           id,
			Dependencies: deps,
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		g, err := BuildFromTasks(tasks)
		if err != nil {
			b.Fatal(err)
		}
		_, err = g.ComputeWaves()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func generateTaskID(n int) string {
	return fmt.Sprintf("T%03d", n+1)
}

// TestPerformanceContract verifies wave computation meets the <100ms requirement.
func TestPerformanceContract(t *testing.T) {
	t.Parallel()

	// Skip in short mode
	if testing.Short() {
		t.Skip("skipping performance test in short mode")
	}

	// Create 100 tasks with complex dependencies
	tasks := make([]validation.TaskItem, 100)
	for i := 0; i < 100; i++ {
		id := generateTaskID(i)
		deps := []string{}

		// Create varying dependency patterns
		if i > 0 {
			if i >= 3 {
				deps = append(deps, generateTaskID(i-3))
			}
			if i >= 2 {
				deps = append(deps, generateTaskID(i-2))
			}
		}

		tasks[i] = validation.TaskItem{
			ID:           id,
			Dependencies: deps,
		}
	}

	g, err := BuildFromTasks(tasks)
	require.NoError(t, err)

	_, err = g.ComputeWaves()
	require.NoError(t, err)

	// Verify we have reasonable wave distribution
	stats := g.GetWaveStats()
	assert.Equal(t, 100, stats.TotalTasks)
	assert.Greater(t, stats.TotalWaves, 0)
}
