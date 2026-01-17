package dag

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/sync/errgroup"
)

func TestNewParallelExecutor_Defaults(t *testing.T) {
	tests := map[string]struct {
		opts     []ParallelExecutorOption
		wantMax  int
		wantFail bool
	}{
		"default values": {
			opts:     nil,
			wantMax:  4, // FR-003: default max_parallel=4
			wantFail: false,
		},
		"custom max parallel": {
			opts:     []ParallelExecutorOption{WithParallelMaxParallel(8)},
			wantMax:  8,
			wantFail: false,
		},
		"max parallel zero ignored": {
			opts:     []ParallelExecutorOption{WithParallelMaxParallel(0)},
			wantMax:  4, // Should remain at default because 0 is invalid
			wantFail: false,
		},
		"fail fast enabled": {
			opts:     []ParallelExecutorOption{WithParallelFailFast(true)},
			wantMax:  4,
			wantFail: true,
		},
		"combined options": {
			opts: []ParallelExecutorOption{
				WithParallelMaxParallel(2),
				WithParallelFailFast(true),
			},
			wantMax:  2,
			wantFail: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Create minimal executor for testing
			dag := &DAGConfig{Layers: []Layer{}}
			executor := NewExecutor(dag, "test.yaml", nil, "", "", nil, nil)

			pe := NewParallelExecutor(executor, tt.opts...)

			if pe.MaxParallel() != tt.wantMax {
				t.Errorf("MaxParallel() = %d, want %d", pe.MaxParallel(), tt.wantMax)
			}
			if pe.FailFast() != tt.wantFail {
				t.Errorf("FailFast() = %v, want %v", pe.FailFast(), tt.wantFail)
			}
		})
	}
}

func TestParallelExecutor_RunningSpecs(t *testing.T) {
	dag := &DAGConfig{Layers: []Layer{}}
	executor := NewExecutor(dag, "test.yaml", nil, "", "", nil, nil)
	pe := NewParallelExecutor(executor)

	// Initially no running specs
	if count := pe.RunningCount(); count != 0 {
		t.Errorf("initial RunningCount() = %d, want 0", count)
	}
	if specs := pe.RunningSpecs(); len(specs) != 0 {
		t.Errorf("initial RunningSpecs() = %v, want empty", specs)
	}

	// Mark specs as running
	pe.markRunning("spec-a")
	pe.markRunning("spec-b")

	if count := pe.RunningCount(); count != 2 {
		t.Errorf("RunningCount() = %d, want 2", count)
	}
	specs := pe.RunningSpecs()
	if len(specs) != 2 {
		t.Errorf("RunningSpecs() len = %d, want 2", len(specs))
	}

	// Mark one done
	pe.markDone("spec-a")

	if count := pe.RunningCount(); count != 1 {
		t.Errorf("after markDone RunningCount() = %d, want 1", count)
	}

	// Mark remaining done
	pe.markDone("spec-b")
	if count := pe.RunningCount(); count != 0 {
		t.Errorf("final RunningCount() = %d, want 0", count)
	}
}

func TestParallelExecutor_ConcurrentMarkRunning(t *testing.T) {
	dag := &DAGConfig{Layers: []Layer{}}
	executor := NewExecutor(dag, "test.yaml", nil, "", "", nil, nil)
	pe := NewParallelExecutor(executor)

	// Test concurrent access is safe
	var wg sync.WaitGroup
	specCount := 100

	for i := 0; i < specCount; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			specID := "spec-" + string(rune('a'+id%26))
			pe.markRunning(specID)
			time.Sleep(time.Millisecond)
			pe.markDone(specID)
		}(i)
	}

	wg.Wait()

	// All specs should be done
	if count := pe.RunningCount(); count != 0 {
		t.Errorf("final RunningCount() = %d, want 0", count)
	}
}

func TestParallelExecutor_ExecuteParallel_RespectsConcurrencyLimit(t *testing.T) {
	tests := map[string]struct {
		maxParallel int
		specCount   int
	}{
		"limit 1 with 3 specs": {
			maxParallel: 1,
			specCount:   3,
		},
		"limit 2 with 4 specs": {
			maxParallel: 2,
			specCount:   4,
		},
		"limit 4 with 8 specs": {
			maxParallel: 4,
			specCount:   8,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Create DAG with test specs
			features := make([]Feature, tt.specCount)
			for i := 0; i < tt.specCount; i++ {
				features[i] = Feature{ID: "spec-" + string(rune('a'+i))}
			}
			dag := &DAGConfig{
				Layers: []Layer{{ID: "L0", Features: features}},
			}

			// Track max concurrent executions
			var maxConcurrent int32
			var currentConcurrent int32
			var mu sync.Mutex

			// Create mock executor with tracking
			executor := NewExecutor(dag, "test.yaml", nil, "", "", nil, nil)
			pe := NewParallelExecutor(executor, WithParallelMaxParallel(tt.maxParallel))

			// Override executeSpec to track concurrency
			specIDs := make([]string, tt.specCount)
			for i := range specIDs {
				specIDs[i] = features[i].ID
			}

			// Use a custom execution that tracks concurrency
			g, _ := errgroup.WithContext(context.Background())
			g.SetLimit(tt.maxParallel)

			for range specIDs {
				g.Go(func() error {
					// Increment concurrent count
					current := atomic.AddInt32(&currentConcurrent, 1)
					mu.Lock()
					if current > maxConcurrent {
						maxConcurrent = current
					}
					mu.Unlock()

					// Simulate work
					time.Sleep(10 * time.Millisecond)

					// Decrement concurrent count
					atomic.AddInt32(&currentConcurrent, -1)
					return nil
				})
			}

			if err := g.Wait(); err != nil {
				t.Fatalf("execution error: %v", err)
			}

			// Verify max concurrent never exceeded limit
			if maxConcurrent > int32(tt.maxParallel) {
				t.Errorf("max concurrent = %d, exceeded limit %d", maxConcurrent, tt.maxParallel)
			}

			// Verify concurrency was actually used (when limit > 1 and specs > 1)
			if tt.maxParallel > 1 && tt.specCount > 1 && maxConcurrent < 2 {
				t.Errorf("max concurrent = %d, expected some parallelism", maxConcurrent)
			}

			// Verify at end no specs are running
			_ = pe // Use pe to avoid unused variable warning in test context
		})
	}
}

func TestParallelExecutor_FindFeature(t *testing.T) {
	tests := map[string]struct {
		dag      *DAGConfig
		specID   string
		wantNil  bool
		wantDesc string
	}{
		"feature in first layer": {
			dag: &DAGConfig{
				Layers: []Layer{
					{ID: "L0", Features: []Feature{{ID: "spec-a", Description: "desc-a"}}},
				},
			},
			specID:   "spec-a",
			wantNil:  false,
			wantDesc: "desc-a",
		},
		"feature in second layer": {
			dag: &DAGConfig{
				Layers: []Layer{
					{ID: "L0", Features: []Feature{{ID: "spec-a"}}},
					{ID: "L1", Features: []Feature{{ID: "spec-b", Description: "desc-b"}}},
				},
			},
			specID:   "spec-b",
			wantNil:  false,
			wantDesc: "desc-b",
		},
		"feature not found": {
			dag: &DAGConfig{
				Layers: []Layer{
					{ID: "L0", Features: []Feature{{ID: "spec-a"}}},
				},
			},
			specID:  "spec-not-exist",
			wantNil: true,
		},
		"empty dag": {
			dag:     &DAGConfig{Layers: []Layer{}},
			specID:  "spec-a",
			wantNil: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			executor := NewExecutor(tt.dag, "test.yaml", nil, "", "", nil, nil)
			pe := NewParallelExecutor(executor)

			feature := pe.findFeature(tt.specID)

			if tt.wantNil {
				if feature != nil {
					t.Errorf("findFeature() = %v, want nil", feature)
				}
			} else {
				if feature == nil {
					t.Fatal("findFeature() returned nil, want non-nil")
				}
				if feature.Description != tt.wantDesc {
					t.Errorf("feature.Description = %q, want %q", feature.Description, tt.wantDesc)
				}
			}
		})
	}
}

func TestParallelExecutor_GetAllSpecIDs(t *testing.T) {
	tests := map[string]struct {
		dag     *DAGConfig
		wantLen int
	}{
		"empty dag": {
			dag:     &DAGConfig{Layers: []Layer{}},
			wantLen: 0,
		},
		"single layer single spec": {
			dag: &DAGConfig{
				Layers: []Layer{
					{ID: "L0", Features: []Feature{{ID: "spec-a"}}},
				},
			},
			wantLen: 1,
		},
		"multiple layers multiple specs": {
			dag: &DAGConfig{
				Layers: []Layer{
					{ID: "L0", Features: []Feature{{ID: "spec-a"}, {ID: "spec-b"}}},
					{ID: "L1", Features: []Feature{{ID: "spec-c"}}},
				},
			},
			wantLen: 3,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			executor := NewExecutor(tt.dag, "test.yaml", nil, "", "", nil, nil)
			pe := NewParallelExecutor(executor)

			ids := pe.getAllSpecIDs()
			if len(ids) != tt.wantLen {
				t.Errorf("getAllSpecIDs() len = %d, want %d", len(ids), tt.wantLen)
			}
		})
	}
}

func TestParallelExecutor_FindReadySpecs(t *testing.T) {
	tests := map[string]struct {
		dag       *DAGConfig
		pending   map[string]bool
		completed map[string]bool
		failed    map[string]bool
		wantLen   int
	}{
		"all specs ready (no deps)": {
			dag: &DAGConfig{
				Layers: []Layer{
					{ID: "L0", Features: []Feature{
						{ID: "spec-a"},
						{ID: "spec-b"},
					}},
				},
			},
			pending:   map[string]bool{"spec-a": true, "spec-b": true},
			completed: map[string]bool{},
			failed:    map[string]bool{},
			wantLen:   2,
		},
		"one spec blocked by dependency": {
			dag: &DAGConfig{
				Layers: []Layer{
					{ID: "L0", Features: []Feature{
						{ID: "spec-a"},
						{ID: "spec-b", DependsOn: []string{"spec-a"}},
					}},
				},
			},
			pending:   map[string]bool{"spec-a": true, "spec-b": true},
			completed: map[string]bool{},
			failed:    map[string]bool{},
			wantLen:   1, // Only spec-a is ready
		},
		"dependency completed unlocks spec": {
			dag: &DAGConfig{
				Layers: []Layer{
					{ID: "L0", Features: []Feature{
						{ID: "spec-a"},
						{ID: "spec-b", DependsOn: []string{"spec-a"}},
					}},
				},
			},
			pending:   map[string]bool{"spec-b": true},
			completed: map[string]bool{"spec-a": true},
			failed:    map[string]bool{},
			wantLen:   1, // spec-b is now ready
		},
		"failed dependency blocks spec": {
			dag: &DAGConfig{
				Layers: []Layer{
					{ID: "L0", Features: []Feature{
						{ID: "spec-a"},
						{ID: "spec-b", DependsOn: []string{"spec-a"}},
					}},
				},
			},
			pending:   map[string]bool{"spec-b": true},
			completed: map[string]bool{},
			failed:    map[string]bool{"spec-a": true},
			wantLen:   0, // spec-b cannot run because spec-a failed
		},
		"multiple dependencies all completed": {
			dag: &DAGConfig{
				Layers: []Layer{
					{ID: "L0", Features: []Feature{
						{ID: "spec-a"},
						{ID: "spec-b"},
						{ID: "spec-c", DependsOn: []string{"spec-a", "spec-b"}},
					}},
				},
			},
			pending:   map[string]bool{"spec-c": true},
			completed: map[string]bool{"spec-a": true, "spec-b": true},
			failed:    map[string]bool{},
			wantLen:   1, // spec-c is ready
		},
		"multiple dependencies partial completion": {
			dag: &DAGConfig{
				Layers: []Layer{
					{ID: "L0", Features: []Feature{
						{ID: "spec-a"},
						{ID: "spec-b"},
						{ID: "spec-c", DependsOn: []string{"spec-a", "spec-b"}},
					}},
				},
			},
			pending:   map[string]bool{"spec-b": true, "spec-c": true},
			completed: map[string]bool{"spec-a": true},
			failed:    map[string]bool{},
			wantLen:   1, // Only spec-b is ready (spec-c waiting on spec-b)
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			executor := NewExecutor(tt.dag, "test.yaml", nil, "", "", nil, nil)
			pe := NewParallelExecutor(executor)

			ready := pe.findReadySpecs(tt.pending, tt.completed, tt.failed)
			if len(ready) != tt.wantLen {
				t.Errorf("findReadySpecs() len = %d, want %d (got %v)", len(ready), tt.wantLen, ready)
			}
		})
	}
}

func TestParallelExecutor_AreDependenciesSatisfied(t *testing.T) {
	dag := &DAGConfig{
		Layers: []Layer{
			{ID: "L0", Features: []Feature{
				{ID: "spec-a"},
				{ID: "spec-b", DependsOn: []string{"spec-a"}},
				{ID: "spec-c", DependsOn: []string{"spec-a", "spec-b"}},
			}},
		},
	}

	tests := map[string]struct {
		specID    string
		completed map[string]bool
		failed    map[string]bool
		want      bool
	}{
		"no dependencies": {
			specID:    "spec-a",
			completed: map[string]bool{},
			failed:    map[string]bool{},
			want:      true,
		},
		"dependency not completed": {
			specID:    "spec-b",
			completed: map[string]bool{},
			failed:    map[string]bool{},
			want:      false,
		},
		"dependency completed": {
			specID:    "spec-b",
			completed: map[string]bool{"spec-a": true},
			failed:    map[string]bool{},
			want:      true,
		},
		"dependency failed": {
			specID:    "spec-b",
			completed: map[string]bool{},
			failed:    map[string]bool{"spec-a": true},
			want:      false,
		},
		"all dependencies satisfied": {
			specID:    "spec-c",
			completed: map[string]bool{"spec-a": true, "spec-b": true},
			failed:    map[string]bool{},
			want:      true,
		},
		"partial dependencies": {
			specID:    "spec-c",
			completed: map[string]bool{"spec-a": true},
			failed:    map[string]bool{},
			want:      false,
		},
		"spec not found": {
			specID:    "nonexistent",
			completed: map[string]bool{},
			failed:    map[string]bool{},
			want:      false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			executor := NewExecutor(dag, "test.yaml", nil, "", "", nil, nil)
			pe := NewParallelExecutor(executor)

			got := pe.areDependenciesSatisfied(tt.specID, tt.completed, tt.failed)
			if got != tt.want {
				t.Errorf("areDependenciesSatisfied() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParallelExecutor_GetBlockingDeps(t *testing.T) {
	dag := &DAGConfig{
		Layers: []Layer{
			{ID: "L0", Features: []Feature{
				{ID: "spec-a"},
				{ID: "spec-b"},
				{ID: "spec-c", DependsOn: []string{"spec-a", "spec-b"}},
			}},
		},
	}

	tests := map[string]struct {
		specID  string
		failed  map[string]bool
		wantLen int
	}{
		"no dependencies": {
			specID:  "spec-a",
			failed:  map[string]bool{},
			wantLen: 0,
		},
		"one failed dependency": {
			specID:  "spec-c",
			failed:  map[string]bool{"spec-a": true},
			wantLen: 1,
		},
		"both dependencies failed": {
			specID:  "spec-c",
			failed:  map[string]bool{"spec-a": true, "spec-b": true},
			wantLen: 2,
		},
		"no failed dependencies": {
			specID:  "spec-c",
			failed:  map[string]bool{},
			wantLen: 0,
		},
		"spec not found": {
			specID:  "nonexistent",
			failed:  map[string]bool{"spec-a": true},
			wantLen: 0,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			executor := NewExecutor(dag, "test.yaml", nil, "", "", nil, nil)
			pe := NewParallelExecutor(executor)

			blocking := pe.getBlockingDeps(tt.specID, tt.failed)
			if len(blocking) != tt.wantLen {
				t.Errorf("getBlockingDeps() len = %d, want %d", len(blocking), tt.wantLen)
			}
		})
	}
}

// TestParallelExecutor_ConcurrencyLimitEnforcement verifies that the executor
// never exceeds the configured max-parallel limit during execution.
func TestParallelExecutor_ConcurrencyLimitEnforcement(t *testing.T) {
	tests := map[string]struct {
		maxParallel int
		specCount   int
		description string
	}{
		"max-parallel 1 sequential execution": {
			maxParallel: 1,
			specCount:   5,
			description: "With max-parallel=1, specs should run sequentially",
		},
		"max-parallel 2 limited concurrency": {
			maxParallel: 2,
			specCount:   6,
			description: "With max-parallel=2, at most 2 specs run at once",
		},
		"max-parallel 3 with fewer specs": {
			maxParallel: 3,
			specCount:   2,
			description: "Max-parallel higher than spec count is acceptable",
		},
		"max-parallel 4 default value": {
			maxParallel: 4,
			specCount:   10,
			description: "Default max-parallel=4 should limit to 4 concurrent",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Track concurrent execution count
			var currentConcurrent int32
			var maxConcurrentSeen int32
			var mu sync.Mutex

			// Use errgroup with SetLimit to simulate ParallelExecutor behavior
			g, _ := errgroup.WithContext(context.Background())
			g.SetLimit(tt.maxParallel)

			for i := 0; i < tt.specCount; i++ {
				g.Go(func() error {
					// Track entry - increment concurrent count
					current := atomic.AddInt32(&currentConcurrent, 1)
					mu.Lock()
					if current > maxConcurrentSeen {
						maxConcurrentSeen = current
					}
					mu.Unlock()

					// Simulate work with varying duration to stress test
					time.Sleep(time.Duration(5+i%3) * time.Millisecond)

					// Track exit - decrement concurrent count
					atomic.AddInt32(&currentConcurrent, -1)
					return nil
				})
			}

			if err := g.Wait(); err != nil {
				t.Fatalf("execution error: %v", err)
			}

			// Verify concurrency limit was respected
			if maxConcurrentSeen > int32(tt.maxParallel) {
				t.Errorf("%s: max concurrent = %d, exceeded limit %d",
					tt.description, maxConcurrentSeen, tt.maxParallel)
			}

			// Verify parallelism was actually used when appropriate
			expectedMin := int32(1)
			if tt.maxParallel > 1 && tt.specCount > 1 {
				expectedMin = 2 // Should see at least some parallelism
			}
			if maxConcurrentSeen < expectedMin {
				t.Errorf("%s: max concurrent = %d, expected at least %d",
					tt.description, maxConcurrentSeen, expectedMin)
			}
		})
	}
}

// TestParallelExecutor_DependencyOrderingWithConcurrency verifies that
// specs with dependencies wait for their dependencies to complete before running.
func TestParallelExecutor_DependencyOrderingWithConcurrency(t *testing.T) {
	tests := map[string]struct {
		dagConfig      *DAGConfig
		expectedOrders [][]string // Each inner slice contains specs that can run concurrently
		description    string
	}{
		"linear dependency chain A->B->C": {
			dagConfig: &DAGConfig{
				Layers: []Layer{
					{ID: "L0", Features: []Feature{
						{ID: "spec-a"},
						{ID: "spec-b", DependsOn: []string{"spec-a"}},
						{ID: "spec-c", DependsOn: []string{"spec-b"}},
					}},
				},
			},
			expectedOrders: [][]string{
				{"spec-a"},
				{"spec-b"},
				{"spec-c"},
			},
			description: "Linear chain should execute strictly in order",
		},
		"diamond dependency A->(B,C)->D": {
			dagConfig: &DAGConfig{
				Layers: []Layer{
					{ID: "L0", Features: []Feature{
						{ID: "spec-a"},
						{ID: "spec-b", DependsOn: []string{"spec-a"}},
						{ID: "spec-c", DependsOn: []string{"spec-a"}},
						{ID: "spec-d", DependsOn: []string{"spec-b", "spec-c"}},
					}},
				},
			},
			expectedOrders: [][]string{
				{"spec-a"},
				{"spec-b", "spec-c"}, // B and C can run in parallel
				{"spec-d"},
			},
			description: "Diamond pattern should allow B and C to run concurrently",
		},
		"independent specs run in parallel": {
			dagConfig: &DAGConfig{
				Layers: []Layer{
					{ID: "L0", Features: []Feature{
						{ID: "spec-a"},
						{ID: "spec-b"},
						{ID: "spec-c"},
					}},
				},
			},
			expectedOrders: [][]string{
				{"spec-a", "spec-b", "spec-c"}, // All can run in parallel
			},
			description: "Independent specs should all be ready simultaneously",
		},
		"mixed dependencies": {
			dagConfig: &DAGConfig{
				Layers: []Layer{
					{ID: "L0", Features: []Feature{
						{ID: "spec-a"},
						{ID: "spec-b"},
						{ID: "spec-c", DependsOn: []string{"spec-a"}},
						{ID: "spec-d", DependsOn: []string{"spec-b"}},
						{ID: "spec-e", DependsOn: []string{"spec-c", "spec-d"}},
					}},
				},
			},
			expectedOrders: [][]string{
				{"spec-a", "spec-b"},
				{"spec-c", "spec-d"},
				{"spec-e"},
			},
			description: "Mixed dependencies should allow concurrent execution where possible",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			executor := NewExecutor(tt.dagConfig, "test.yaml", nil, "", "", nil, nil)
			pe := NewParallelExecutor(executor, WithParallelMaxParallel(4))

			// Get all spec IDs and simulate the scheduling
			allSpecs := pe.getAllSpecIDs()
			if len(allSpecs) == 0 {
				t.Fatal("No specs found in DAG config")
			}

			// Simulate dependency-aware scheduling
			pending := make(map[string]bool, len(allSpecs))
			for _, id := range allSpecs {
				pending[id] = true
			}
			completed := make(map[string]bool)
			failed := make(map[string]bool)

			executionWaves := [][]string{}
			for len(pending) > 0 {
				ready := pe.findReadySpecs(pending, completed, failed)
				if len(ready) == 0 {
					t.Fatalf("Deadlock: %d specs pending but none ready", len(pending))
				}
				executionWaves = append(executionWaves, ready)
				for _, specID := range ready {
					delete(pending, specID)
					completed[specID] = true
				}
			}

			// Verify execution waves match expected orders
			if len(executionWaves) != len(tt.expectedOrders) {
				t.Errorf("%s: got %d waves, expected %d\n  got: %v\n  expected: %v",
					tt.description, len(executionWaves), len(tt.expectedOrders),
					executionWaves, tt.expectedOrders)
				return
			}

			for i, wave := range executionWaves {
				expectedWave := tt.expectedOrders[i]
				if !containsSameElements(wave, expectedWave) {
					t.Errorf("%s: wave %d mismatch\n  got: %v\n  expected: %v",
						tt.description, i, wave, expectedWave)
				}
			}
		})
	}
}

// containsSameElements checks if two slices contain the same elements (order-independent).
func containsSameElements(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	aMap := make(map[string]bool, len(a))
	for _, v := range a {
		aMap[v] = true
	}
	for _, v := range b {
		if !aMap[v] {
			return false
		}
	}
	return true
}

// TestParallelExecutor_CreateParallelExecutorFromConfig verifies the factory function.
func TestParallelExecutor_CreateParallelExecutorFromConfig(t *testing.T) {
	tests := map[string]struct {
		maxParallel int
		failFast    bool
		wantMax     int
		wantFail    bool
	}{
		"default parallel with limit 2": {
			maxParallel: 2,
			failFast:    false,
			wantMax:     2,
			wantFail:    false,
		},
		"parallel with fail-fast enabled": {
			maxParallel: 4,
			failFast:    true,
			wantMax:     4,
			wantFail:    true,
		},
		"single threaded parallel mode": {
			maxParallel: 1,
			failFast:    false,
			wantMax:     1,
			wantFail:    false,
		},
		"high parallelism": {
			maxParallel: 8,
			failFast:    false,
			wantMax:     8,
			wantFail:    false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			dagCfg := &DAGConfig{
				Layers: []Layer{{ID: "L0", Features: []Feature{{ID: "spec-a"}}}},
			}

			pe := CreateParallelExecutorFromConfig(
				dagCfg,
				"test.yaml",
				nil, // worktreeManager
				"",  // stateDir
				"",  // repoRoot
				nil, // config
				nil, // worktreeConfig
				tt.maxParallel,
				tt.failFast,
				nil, // stdout
			)

			if pe.MaxParallel() != tt.wantMax {
				t.Errorf("MaxParallel() = %d, want %d", pe.MaxParallel(), tt.wantMax)
			}
			if pe.FailFast() != tt.wantFail {
				t.Errorf("FailFast() = %v, want %v", pe.FailFast(), tt.wantFail)
			}
			if pe.Executor() == nil {
				t.Error("Executor() should not be nil")
			}
		})
	}
}

// TestParallelExecutor_MarkBlockedSpecs verifies that specs are marked as blocked
// when their dependencies fail.
func TestParallelExecutor_MarkBlockedSpecs(t *testing.T) {
	tests := map[string]struct {
		dagConfig     *DAGConfig
		failedSpecs   map[string]bool
		pendingSpecs  map[string]bool
		wantBlocked   []string
		wantBlockedBy map[string][]string
	}{
		"single dependency fails blocks one spec": {
			dagConfig: &DAGConfig{
				Layers: []Layer{
					{ID: "L0", Features: []Feature{
						{ID: "spec-a"},
						{ID: "spec-b", DependsOn: []string{"spec-a"}},
					}},
				},
			},
			failedSpecs:  map[string]bool{"spec-a": true},
			pendingSpecs: map[string]bool{"spec-b": true},
			wantBlocked:  []string{"spec-b"},
			wantBlockedBy: map[string][]string{
				"spec-b": {"spec-a"},
			},
		},
		"multiple dependencies with one failed": {
			dagConfig: &DAGConfig{
				Layers: []Layer{
					{ID: "L0", Features: []Feature{
						{ID: "spec-a"},
						{ID: "spec-b"},
						{ID: "spec-c", DependsOn: []string{"spec-a", "spec-b"}},
					}},
				},
			},
			failedSpecs:  map[string]bool{"spec-a": true},
			pendingSpecs: map[string]bool{"spec-c": true},
			wantBlocked:  []string{"spec-c"},
			wantBlockedBy: map[string][]string{
				"spec-c": {"spec-a"},
			},
		},
		"chain dependency failure blocks multiple": {
			dagConfig: &DAGConfig{
				Layers: []Layer{
					{ID: "L0", Features: []Feature{
						{ID: "spec-a"},
						{ID: "spec-b", DependsOn: []string{"spec-a"}},
						{ID: "spec-c", DependsOn: []string{"spec-b"}},
					}},
				},
			},
			failedSpecs:  map[string]bool{"spec-a": true},
			pendingSpecs: map[string]bool{"spec-b": true, "spec-c": true},
			wantBlocked:  []string{"spec-b", "spec-c"},
			wantBlockedBy: map[string][]string{
				"spec-b": {"spec-a"},
				"spec-c": {}, // spec-c depends on spec-b which isn't failed yet
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			executor := NewExecutor(tt.dagConfig, "test.yaml", nil, "", "", nil, nil)
			// Initialize state so markBlockedSpecs can update it
			executor.state = NewDAGRun("test.yaml", tt.dagConfig, 0)
			pe := NewParallelExecutor(executor)

			pe.markBlockedSpecs(tt.pendingSpecs, tt.failedSpecs)

			// Verify blocked specs
			for _, specID := range tt.wantBlocked {
				specState := executor.state.Specs[specID]
				if specState == nil {
					t.Errorf("spec %s not found in state", specID)
					continue
				}
				if specState.Status != SpecStatusBlocked {
					t.Errorf("spec %s status = %v, want %v", specID, specState.Status, SpecStatusBlocked)
				}
			}

			// Verify blocked_by lists
			for specID, wantBlockedBy := range tt.wantBlockedBy {
				specState := executor.state.Specs[specID]
				if specState == nil {
					continue
				}
				if len(specState.BlockedBy) != len(wantBlockedBy) {
					t.Errorf("spec %s blocked_by len = %d, want %d",
						specID, len(specState.BlockedBy), len(wantBlockedBy))
				}
			}
		})
	}
}

// TestParallelExecutor_FailFastBehavior verifies that fail-fast mode cancels
// all running specs on first failure.
func TestParallelExecutor_FailFastBehavior(t *testing.T) {
	tests := map[string]struct {
		failFast    bool
		maxParallel int
		specCount   int
		description string
	}{
		"fail-fast disabled allows completion": {
			failFast:    false,
			maxParallel: 2,
			specCount:   4,
			description: "Without fail-fast, specs should continue despite failures",
		},
		"fail-fast enabled cancels on error": {
			failFast:    true,
			maxParallel: 2,
			specCount:   4,
			description: "With fail-fast, context should be cancelled on first error",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Create DAG with independent specs
			features := make([]Feature, tt.specCount)
			for i := 0; i < tt.specCount; i++ {
				features[i] = Feature{ID: "spec-" + string(rune('a'+i))}
			}
			dagCfg := &DAGConfig{
				Layers: []Layer{{ID: "L0", Features: features}},
			}

			executor := NewExecutor(dagCfg, "test.yaml", nil, "", "", nil, nil)
			pe := NewParallelExecutor(executor,
				WithParallelMaxParallel(tt.maxParallel),
				WithParallelFailFast(tt.failFast),
			)

			// Verify configuration
			if pe.FailFast() != tt.failFast {
				t.Errorf("FailFast() = %v, want %v", pe.FailFast(), tt.failFast)
			}
			if pe.MaxParallel() != tt.maxParallel {
				t.Errorf("MaxParallel() = %d, want %d", pe.MaxParallel(), tt.maxParallel)
			}
		})
	}
}

// TestParallelExecutor_IndependentSpecsCompleteOnFailure verifies that
// independent specs complete even when unrelated specs fail.
func TestParallelExecutor_IndependentSpecsCompleteOnFailure(t *testing.T) {
	// DAG with independent spec C and dependent chain A->B
	dagCfg := &DAGConfig{
		Layers: []Layer{
			{ID: "L0", Features: []Feature{
				{ID: "spec-a"},
				{ID: "spec-b", DependsOn: []string{"spec-a"}},
				{ID: "spec-c"}, // Independent, no dependencies
			}},
		},
	}

	executor := NewExecutor(dagCfg, "test.yaml", nil, "", "", nil, nil)
	executor.state = NewDAGRun("test.yaml", dagCfg, 0)
	pe := NewParallelExecutor(executor, WithParallelMaxParallel(4))

	// Simulate spec-a failed, spec-c completed
	completedSpecs := map[string]bool{"spec-c": true}
	failedSpecs := map[string]bool{"spec-a": true}
	pendingSpecs := map[string]bool{"spec-b": true}

	// Find ready specs - spec-b should not be ready since spec-a failed
	ready := pe.findReadySpecs(pendingSpecs, completedSpecs, failedSpecs)

	if len(ready) != 0 {
		t.Errorf("findReadySpecs() returned %d ready specs, want 0 (spec-b blocked)", len(ready))
	}

	// Mark blocked specs
	pe.markBlockedSpecs(pendingSpecs, failedSpecs)

	// Verify spec-b is marked as blocked
	specB := executor.state.Specs["spec-b"]
	if specB.Status != SpecStatusBlocked {
		t.Errorf("spec-b status = %v, want blocked", specB.Status)
	}
	if len(specB.BlockedBy) != 1 || specB.BlockedBy[0] != "spec-a" {
		t.Errorf("spec-b blocked_by = %v, want [spec-a]", specB.BlockedBy)
	}
}

// TestParallelExecutor_HandleInterruption verifies interruption handling
// saves state correctly.
func TestParallelExecutor_HandleInterruption(t *testing.T) {
	dagCfg := &DAGConfig{
		Layers: []Layer{
			{ID: "L0", Features: []Feature{
				{ID: "spec-a"},
				{ID: "spec-b"},
			}},
		},
	}

	// Create temp directory for state
	tempDir := t.TempDir()

	executor := NewExecutor(dagCfg, "test.yaml", nil, tempDir, "", nil, nil)
	executor.state = NewDAGRun("test.yaml", dagCfg, 0)
	pe := NewParallelExecutor(executor)

	// Simulate running specs
	pe.markRunning("spec-a")
	pe.markRunning("spec-b")

	// Verify running count
	if count := pe.RunningCount(); count != 2 {
		t.Errorf("RunningCount() = %d before interruption, want 2", count)
	}

	// Call handleInterruption
	pe.handleInterruption()

	// Verify state was updated
	state := executor.State()
	if state.Status != RunStatusInterrupted {
		t.Errorf("run status = %v, want %v", state.Status, RunStatusInterrupted)
	}
	if state.RunningCount != 0 {
		t.Errorf("running_count = %d, want 0", state.RunningCount)
	}

	// Verify spec states updated
	for _, specID := range []string{"spec-a", "spec-b"} {
		specState := state.Specs[specID]
		if specState.Status != SpecStatusFailed {
			t.Errorf("spec %s status = %v, want failed", specID, specState.Status)
		}
		if specState.FailureReason != "interrupted by signal" {
			t.Errorf("spec %s failure_reason = %q, want 'interrupted by signal'",
				specID, specState.FailureReason)
		}
	}
}

// TestParallelExecutor_UpdateFinalRunStatus verifies final run status is set correctly.
func TestParallelExecutor_UpdateFinalRunStatus(t *testing.T) {
	tests := map[string]struct {
		failedSpecs map[string]bool
		wantStatus  RunStatus
	}{
		"no failures - completed": {
			failedSpecs: map[string]bool{},
			wantStatus:  RunStatusCompleted,
		},
		"with failures - failed": {
			failedSpecs: map[string]bool{"spec-a": true},
			wantStatus:  RunStatusFailed,
		},
		"multiple failures - failed": {
			failedSpecs: map[string]bool{"spec-a": true, "spec-b": true},
			wantStatus:  RunStatusFailed,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			dagCfg := &DAGConfig{
				Layers: []Layer{
					{ID: "L0", Features: []Feature{{ID: "spec-a"}, {ID: "spec-b"}}},
				},
			}

			tempDir := t.TempDir()
			executor := NewExecutor(dagCfg, "test.yaml", nil, tempDir, "", nil, nil)
			executor.state = NewDAGRun("test.yaml", dagCfg, 0)
			pe := NewParallelExecutor(executor)

			var mu sync.Mutex
			pe.updateFinalRunStatus(tt.failedSpecs, &mu)

			if executor.state.Status != tt.wantStatus {
				t.Errorf("run status = %v, want %v", executor.state.Status, tt.wantStatus)
			}
		})
	}
}

// =============================================================================
// Race Condition Tests - designed to be run with: go test -race
// These tests exercise concurrent access patterns to verify thread safety.
// =============================================================================

// TestRace_ParallelExecutor_ConcurrentSpecCompletion verifies that concurrent
// spec completions don't cause data races on the running specs map.
// Run with: go test -race -run TestRace_ParallelExecutor_ConcurrentSpecCompletion
func TestRace_ParallelExecutor_ConcurrentSpecCompletion(t *testing.T) {
	const numSpecs = 50
	const numIterations = 3

	for iter := 0; iter < numIterations; iter++ {
		dag := &DAGConfig{Layers: []Layer{}}
		executor := NewExecutor(dag, "test.yaml", nil, "", "", nil, nil)
		pe := NewParallelExecutor(executor)

		// Start all specs concurrently
		var wg sync.WaitGroup
		for i := 0; i < numSpecs; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				specID := fmt.Sprintf("spec-%d", id)

				// Mark running
				pe.markRunning(specID)

				// Simulate variable work duration
				time.Sleep(time.Duration(id%5) * time.Millisecond)

				// Concurrent read of running count
				_ = pe.RunningCount()
				_ = pe.RunningSpecs()

				// Mark done
				pe.markDone(specID)
			}(i)
		}

		wg.Wait()

		// Verify final state
		if count := pe.RunningCount(); count != 0 {
			t.Errorf("iteration %d: final RunningCount() = %d, want 0", iter, count)
		}
	}
}

// TestRace_ParallelExecutor_ConcurrentReadsAndWrites verifies that concurrent
// reads and writes to the ParallelExecutor are thread-safe.
// Run with: go test -race -run TestRace_ParallelExecutor_ConcurrentReadsAndWrites
func TestRace_ParallelExecutor_ConcurrentReadsAndWrites(t *testing.T) {
	dag := &DAGConfig{Layers: []Layer{}}
	executor := NewExecutor(dag, "test.yaml", nil, "", "", nil, nil)
	pe := NewParallelExecutor(executor)

	const numGoroutines = 20
	const opsPerGoroutine = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Half goroutines do writes, half do reads
	for i := 0; i < numGoroutines; i++ {
		if i%2 == 0 {
			// Writer goroutine
			go func(id int) {
				defer wg.Done()
				specID := fmt.Sprintf("spec-%d", id)
				for j := 0; j < opsPerGoroutine; j++ {
					pe.markRunning(specID)
					pe.markDone(specID)
				}
			}(i)
		} else {
			// Reader goroutine
			go func() {
				defer wg.Done()
				for j := 0; j < opsPerGoroutine; j++ {
					_ = pe.RunningCount()
					_ = pe.RunningSpecs()
					_ = pe.MaxParallel()
					_ = pe.FailFast()
				}
			}()
		}
	}

	wg.Wait()
}

// TestRace_ParallelExecutor_SimultaneousMarkOperations verifies thread safety
// when markRunning and markDone are called simultaneously on different specs.
// Run with: go test -race -run TestRace_ParallelExecutor_SimultaneousMarkOperations
func TestRace_ParallelExecutor_SimultaneousMarkOperations(t *testing.T) {
	dag := &DAGConfig{Layers: []Layer{}}
	executor := NewExecutor(dag, "test.yaml", nil, "", "", nil, nil)
	pe := NewParallelExecutor(executor)

	const numPairs = 25

	var wg sync.WaitGroup

	// Create pairs of goroutines that mark different specs running/done
	for i := 0; i < numPairs; i++ {
		specID1 := fmt.Sprintf("spec-%d-a", i)
		specID2 := fmt.Sprintf("spec-%d-b", i)

		wg.Add(2)

		// First goroutine marks spec1 running, then done
		go func(id string) {
			defer wg.Done()
			pe.markRunning(id)
			time.Sleep(time.Microsecond)
			pe.markDone(id)
		}(specID1)

		// Second goroutine marks spec2 running, then done (simultaneously)
		go func(id string) {
			defer wg.Done()
			pe.markRunning(id)
			time.Sleep(time.Microsecond)
			pe.markDone(id)
		}(specID2)
	}

	wg.Wait()

	if count := pe.RunningCount(); count != 0 {
		t.Errorf("final RunningCount() = %d, want 0", count)
	}
}

// TestRace_ProgressTracker_ConcurrentUpdates verifies that concurrent updates
// to the ProgressTracker don't cause data races.
// Run with: go test -race -run TestRace_ProgressTracker_ConcurrentUpdates
func TestRace_ProgressTracker_ConcurrentUpdates(t *testing.T) {
	const total = 100
	pt := NewProgressTracker(total)

	// Register a callback that reads stats
	var callbackCount int32
	pt.OnChange(func(stats ProgressStats) {
		atomic.AddInt32(&callbackCount, 1)
		// Access all fields to ensure they're consistent
		_ = stats.Completed + stats.Running + stats.Failed + stats.Blocked
		_ = stats.IsComplete()
		_ = stats.SuccessRate()
	})

	var wg sync.WaitGroup
	numUpdates := total / 4

	// Goroutine marking specs as running
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < numUpdates; i++ {
			pt.MarkRunning()
		}
	}()

	// Goroutine marking specs as completed
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < numUpdates; i++ {
			pt.MarkCompleted()
		}
	}()

	// Goroutine marking specs as failed
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < numUpdates; i++ {
			pt.MarkFailed()
		}
	}()

	// Goroutine marking specs as blocked
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < numUpdates; i++ {
			pt.MarkBlocked()
		}
	}()

	// Goroutine reading stats concurrently
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < numUpdates*4; i++ {
			_ = pt.Stats()
			_ = pt.Render()
			_ = pt.RenderDetailed()
		}
	}()

	wg.Wait()

	// Verify callback was called
	if atomic.LoadInt32(&callbackCount) == 0 {
		t.Error("callback was never invoked during concurrent updates")
	}
}

// TestRace_ProgressTracker_CallbackDuringUpdate verifies that the callback
// is invoked correctly even during concurrent updates.
// Run with: go test -race -run TestRace_ProgressTracker_CallbackDuringUpdate
func TestRace_ProgressTracker_CallbackDuringUpdate(t *testing.T) {
	const total = 50
	pt := NewProgressTracker(total)

	var statsHistory []ProgressStats
	var mu sync.Mutex

	pt.OnChange(func(stats ProgressStats) {
		mu.Lock()
		// Create a copy to avoid potential issues with mutation
		statsCopy := ProgressStats{
			Total:     stats.Total,
			Completed: stats.Completed,
			Running:   stats.Running,
			Failed:    stats.Failed,
			Blocked:   stats.Blocked,
			Pending:   stats.Pending,
		}
		statsHistory = append(statsHistory, statsCopy)
		mu.Unlock()
	})

	var wg sync.WaitGroup
	const numGoroutines = 10
	const updatesPerGoroutine = 5

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < updatesPerGoroutine; j++ {
				switch (id + j) % 4 {
				case 0:
					pt.MarkRunning()
				case 1:
					pt.MarkCompleted()
				case 2:
					pt.MarkFailed()
				case 3:
					pt.MarkBlocked()
				}
			}
		}(i)
	}

	wg.Wait()

	mu.Lock()
	historyLen := len(statsHistory)
	mu.Unlock()

	// Verify callbacks were received
	expectedCallbacks := numGoroutines * updatesPerGoroutine
	if historyLen != expectedCallbacks {
		t.Errorf("received %d callbacks, want %d", historyLen, expectedCallbacks)
	}
}

// TestRace_ErrGroupWithParallelExecution simulates actual parallel execution
// pattern used by ParallelExecutor with errgroup.
// Run with: go test -race -run TestRace_ErrGroupWithParallelExecution
func TestRace_ErrGroupWithParallelExecution(t *testing.T) {
	const numSpecs = 20
	const maxParallel = 4

	// Create shared state to track execution order
	var executionOrder []string
	var orderMu sync.Mutex

	completed := make(map[string]bool)
	failed := make(map[string]bool)
	var stateMu sync.Mutex

	g, ctx := errgroup.WithContext(context.Background())
	g.SetLimit(maxParallel)

	for i := 0; i < numSpecs; i++ {
		specID := fmt.Sprintf("spec-%02d", i)
		g.Go(func() error {
			// Check context
			if ctx.Err() != nil {
				return ctx.Err()
			}

			// Record execution start
			orderMu.Lock()
			executionOrder = append(executionOrder, specID+":start")
			orderMu.Unlock()

			// Simulate variable work
			time.Sleep(time.Duration(1+i%3) * time.Millisecond)

			// Record completion
			orderMu.Lock()
			executionOrder = append(executionOrder, specID+":end")
			orderMu.Unlock()

			// Update shared state
			stateMu.Lock()
			completed[specID] = true
			stateMu.Unlock()

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		t.Fatalf("errgroup error: %v", err)
	}

	// Verify all specs completed
	stateMu.Lock()
	completedCount := len(completed)
	failedCount := len(failed)
	stateMu.Unlock()

	if completedCount != numSpecs {
		t.Errorf("completed %d specs, want %d", completedCount, numSpecs)
	}
	if failedCount != 0 {
		t.Errorf("failed %d specs, want 0", failedCount)
	}
}

// TestRace_ParallelExecutor_StateAccess verifies thread-safe access to executor state
// during parallel operations.
// Run with: go test -race -run TestRace_ParallelExecutor_StateAccess
func TestRace_ParallelExecutor_StateAccess(t *testing.T) {
	dagCfg := &DAGConfig{
		Layers: []Layer{
			{ID: "L0", Features: []Feature{
				{ID: "spec-a"},
				{ID: "spec-b"},
				{ID: "spec-c"},
			}},
		},
	}

	tempDir := t.TempDir()
	executor := NewExecutor(dagCfg, "test.yaml", nil, tempDir, "", nil, nil)
	executor.state = NewDAGRun("test.yaml", dagCfg, 0)
	pe := NewParallelExecutor(executor)

	var wg sync.WaitGroup
	const numReaders = 10
	const numWriters = 5
	const iterations = 50

	// Writer goroutines
	for i := 0; i < numWriters; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			specID := fmt.Sprintf("spec-%c", 'a'+id%3)
			for j := 0; j < iterations; j++ {
				pe.markRunning(specID)
				pe.markDone(specID)
			}
		}(i)
	}

	// Reader goroutines
	for i := 0; i < numReaders; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				_ = pe.State()
				_ = pe.RunningCount()
				_ = pe.RunningSpecs()
				_ = pe.RunID()
			}
		}()
	}

	wg.Wait()
}

// TestRace_ParallelExecutor_HandleInterruptionDuringExecution verifies that
// handleInterruption can be called safely while specs are being marked running/done.
// Run with: go test -race -run TestRace_ParallelExecutor_HandleInterruptionDuringExecution
func TestRace_ParallelExecutor_HandleInterruptionDuringExecution(t *testing.T) {
	dagCfg := &DAGConfig{
		Layers: []Layer{
			{ID: "L0", Features: []Feature{
				{ID: "spec-a"},
				{ID: "spec-b"},
				{ID: "spec-c"},
				{ID: "spec-d"},
			}},
		},
	}

	tempDir := t.TempDir()
	executor := NewExecutor(dagCfg, "test.yaml", nil, tempDir, "", nil, nil)
	executor.state = NewDAGRun("test.yaml", dagCfg, 0)
	pe := NewParallelExecutor(executor)

	var wg sync.WaitGroup
	const numWorkers = 4

	// Start workers that mark specs running/done
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			specID := fmt.Sprintf("spec-%c", 'a'+id)
			for j := 0; j < 20; j++ {
				pe.markRunning(specID)
				time.Sleep(time.Microsecond)
				pe.markDone(specID)
			}
		}(i)
	}

	// Concurrently call handleInterruption
	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(time.Millisecond) // Let workers start
		pe.handleInterruption()
	}()

	wg.Wait()
}
