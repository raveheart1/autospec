// Package workflow tests workflow stage configuration and artifact dependencies.
// Related: internal/workflow/stage_config.go
// Tags: workflow, stages, configuration, artifacts, dependencies, validation
package workflow

import (
	"testing"
)

func TestNewStageConfig(t *testing.T) {
	sc := NewStageConfig()
	if sc == nil {
		t.Fatal("NewStageConfig returned nil")
	}
	// Check core stages are disabled
	if sc.Specify || sc.Plan || sc.Tasks || sc.Implement {
		t.Error("NewStageConfig should have all core stages disabled")
	}
	// Check optional stages are disabled
	if sc.Constitution || sc.Clarify || sc.Checklist || sc.Analyze {
		t.Error("NewStageConfig should have all optional stages disabled")
	}
}

func TestNewStageConfigAll(t *testing.T) {
	sc := NewStageConfigAll()
	if sc == nil {
		t.Fatal("NewStageConfigAll returned nil")
	}
	if !sc.Specify || !sc.Plan || !sc.Tasks || !sc.Implement {
		t.Error("NewStageConfigAll should have all core stages enabled")
	}
}

func TestHasAnyStage(t *testing.T) {
	tests := []struct {
		name     string
		config   StageConfig
		expected bool
	}{
		{
			name:     "no stages selected",
			config:   StageConfig{},
			expected: false,
		},
		{
			name:     "only specify selected",
			config:   StageConfig{Specify: true},
			expected: true,
		},
		{
			name:     "only plan selected",
			config:   StageConfig{Plan: true},
			expected: true,
		},
		{
			name:     "only tasks selected",
			config:   StageConfig{Tasks: true},
			expected: true,
		},
		{
			name:     "only implement selected",
			config:   StageConfig{Implement: true},
			expected: true,
		},
		{
			name:     "all stages selected",
			config:   StageConfig{Specify: true, Plan: true, Tasks: true, Implement: true},
			expected: true,
		},
		{
			name:     "plan and implement selected",
			config:   StageConfig{Plan: true, Implement: true},
			expected: true,
		},
		// Optional stage tests
		{
			name:     "only constitution selected",
			config:   StageConfig{Constitution: true},
			expected: true,
		},
		{
			name:     "only clarify selected",
			config:   StageConfig{Clarify: true},
			expected: true,
		},
		{
			name:     "only checklist selected",
			config:   StageConfig{Checklist: true},
			expected: true,
		},
		{
			name:     "only analyze selected",
			config:   StageConfig{Analyze: true},
			expected: true,
		},
		{
			name:     "mixed core and optional stages",
			config:   StageConfig{Specify: true, Clarify: true, Checklist: true},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.config.HasAnyStage(); got != tt.expected {
				t.Errorf("HasAnyStage() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGetSelectedStages(t *testing.T) {
	tests := []struct {
		name     string
		config   StageConfig
		expected []Stage
	}{
		{
			name:     "no stages selected",
			config:   StageConfig{},
			expected: []Stage{},
		},
		{
			name:     "only specify",
			config:   StageConfig{Specify: true},
			expected: []Stage{StageSpecify},
		},
		{
			name:     "plan and implement",
			config:   StageConfig{Plan: true, Implement: true},
			expected: []Stage{StagePlan, StageImplement},
		},
		{
			name:     "all core stages",
			config:   StageConfig{Specify: true, Plan: true, Tasks: true, Implement: true},
			expected: []Stage{StageSpecify, StagePlan, StageTasks, StageImplement},
		},
		{
			name:     "tasks and implement (skipping earlier stages)",
			config:   StageConfig{Tasks: true, Implement: true},
			expected: []Stage{StageTasks, StageImplement},
		},
		// Optional stage tests
		{
			name:     "only constitution",
			config:   StageConfig{Constitution: true},
			expected: []Stage{StageConstitution},
		},
		{
			name:     "constitution and specify in canonical order",
			config:   StageConfig{Specify: true, Constitution: true},
			expected: []Stage{StageConstitution, StageSpecify},
		},
		{
			name:     "specify with clarify",
			config:   StageConfig{Specify: true, Clarify: true},
			expected: []Stage{StageSpecify, StageClarify},
		},
		{
			name:     "tasks with checklist and analyze",
			config:   StageConfig{Tasks: true, Checklist: true, Analyze: true},
			expected: []Stage{StageTasks, StageChecklist, StageAnalyze},
		},
		{
			name:     "full workflow with optional stages",
			config:   StageConfig{Constitution: true, Specify: true, Clarify: true, Plan: true, Tasks: true, Checklist: true, Analyze: true, Implement: true},
			expected: []Stage{StageConstitution, StageSpecify, StageClarify, StagePlan, StageTasks, StageChecklist, StageAnalyze, StageImplement},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.GetSelectedStages()
			if len(got) != len(tt.expected) {
				t.Errorf("GetSelectedStages() returned %d stages, want %d", len(got), len(tt.expected))
				return
			}
			for i, stage := range got {
				if stage != tt.expected[i] {
					t.Errorf("GetSelectedStages()[%d] = %v, want %v", i, stage, tt.expected[i])
				}
			}
		})
	}
}

func TestGetCanonicalOrder(t *testing.T) {
	// The canonical order must always be:
	// constitution -> specify -> clarify -> plan -> tasks -> checklist -> analyze -> implement
	// regardless of how the struct fields are set or in what order

	tests := []struct {
		name     string
		config   StageConfig
		expected []Stage
	}{
		{
			name:     "core stages set in reverse order",
			config:   StageConfig{Implement: true, Tasks: true, Plan: true, Specify: true},
			expected: []Stage{StageSpecify, StagePlan, StageTasks, StageImplement},
		},
		{
			name:     "only middle stages",
			config:   StageConfig{Plan: true, Tasks: true},
			expected: []Stage{StagePlan, StageTasks},
		},
		{
			name:     "only first and last",
			config:   StageConfig{Specify: true, Implement: true},
			expected: []Stage{StageSpecify, StageImplement},
		},
		// Optional stages canonical order tests - from spec US-003
		{
			name:     "-ns executes as: constitution, specify",
			config:   StageConfig{Constitution: true, Specify: true},
			expected: []Stage{StageConstitution, StageSpecify},
		},
		{
			name:     "-srp executes as: specify, clarify, plan",
			config:   StageConfig{Specify: true, Clarify: true, Plan: true},
			expected: []Stage{StageSpecify, StageClarify, StagePlan},
		},
		{
			name:     "-tczi executes as: tasks, checklist, analyze, implement",
			config:   StageConfig{Tasks: true, Checklist: true, Analyze: true, Implement: true},
			expected: []Stage{StageTasks, StageChecklist, StageAnalyze, StageImplement},
		},
		{
			name:     "-icts executes as: specify, tasks, checklist, implement (regardless of flag order)",
			config:   StageConfig{Implement: true, Checklist: true, Tasks: true, Specify: true},
			expected: []Stage{StageSpecify, StageTasks, StageChecklist, StageImplement},
		},
		{
			name:     "all stages in canonical order",
			config:   StageConfig{Constitution: true, Specify: true, Clarify: true, Plan: true, Tasks: true, Checklist: true, Analyze: true, Implement: true},
			expected: []Stage{StageConstitution, StageSpecify, StageClarify, StagePlan, StageTasks, StageChecklist, StageAnalyze, StageImplement},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.GetCanonicalOrder()
			if len(got) != len(tt.expected) {
				t.Errorf("GetCanonicalOrder() returned %d stages, want %d", len(got), len(tt.expected))
				return
			}
			for i, stage := range got {
				if stage != tt.expected[i] {
					t.Errorf("GetCanonicalOrder()[%d] = %v, want %v", i, stage, tt.expected[i])
				}
			}
		})
	}
}

func TestSetAll(t *testing.T) {
	sc := &StageConfig{}
	sc.SetAll()
	if !sc.Specify || !sc.Plan || !sc.Tasks || !sc.Implement {
		t.Error("SetAll should enable all core stages")
	}
}

func TestCount(t *testing.T) {
	tests := []struct {
		name     string
		config   StageConfig
		expected int
	}{
		{
			name:     "no stages",
			config:   StageConfig{},
			expected: 0,
		},
		{
			name:     "one core stage",
			config:   StageConfig{Plan: true},
			expected: 1,
		},
		{
			name:     "two core stages",
			config:   StageConfig{Plan: true, Implement: true},
			expected: 2,
		},
		{
			name:     "all core stages",
			config:   StageConfig{Specify: true, Plan: true, Tasks: true, Implement: true},
			expected: 4,
		},
		// Optional stage count tests
		{
			name:     "one optional stage",
			config:   StageConfig{Constitution: true},
			expected: 1,
		},
		{
			name:     "all optional stages",
			config:   StageConfig{Constitution: true, Clarify: true, Checklist: true, Analyze: true},
			expected: 4,
		},
		{
			name:     "mixed core and optional stages",
			config:   StageConfig{Specify: true, Clarify: true, Plan: true, Checklist: true},
			expected: 4,
		},
		{
			name:     "all 8 stages",
			config:   StageConfig{Constitution: true, Specify: true, Clarify: true, Plan: true, Tasks: true, Checklist: true, Analyze: true, Implement: true},
			expected: 8,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.config.Count(); got != tt.expected {
				t.Errorf("Count() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGetArtifactDependencies(t *testing.T) {
	deps := GetArtifactDependencies()

	// 4 core stages + 4 optional stages = 8 total
	if len(deps) != 8 {
		t.Errorf("GetArtifactDependencies() returned %d entries, want 8", len(deps))
	}

	// Verify each stage has a dependency entry
	stages := []Stage{
		// Core stages
		StageSpecify, StagePlan, StageTasks, StageImplement,
		// Optional stages
		StageConstitution, StageClarify, StageChecklist, StageAnalyze,
	}
	for _, stage := range stages {
		if _, ok := deps[stage]; !ok {
			t.Errorf("GetArtifactDependencies() missing entry for %s", stage)
		}
	}
}

func TestGetArtifactDependency(t *testing.T) {
	tests := []struct {
		stage            Stage
		expectedRequires []string
		expectedProduces []string
	}{
		// Core stages
		{
			stage:            StageSpecify,
			expectedRequires: []string{},
			expectedProduces: []string{"spec.yaml"},
		},
		{
			stage:            StagePlan,
			expectedRequires: []string{"spec.yaml"},
			expectedProduces: []string{"plan.yaml"},
		},
		{
			stage:            StageTasks,
			expectedRequires: []string{"plan.yaml"},
			expectedProduces: []string{"tasks.yaml"},
		},
		{
			stage:            StageImplement,
			expectedRequires: []string{"tasks.yaml"},
			expectedProduces: []string{},
		},
		// Optional stages
		{
			stage:            StageConstitution,
			expectedRequires: []string{},
			expectedProduces: []string{},
		},
		{
			stage:            StageClarify,
			expectedRequires: []string{"spec.yaml"},
			expectedProduces: []string{},
		},
		{
			stage:            StageChecklist,
			expectedRequires: []string{"spec.yaml"},
			expectedProduces: []string{},
		},
		{
			stage:            StageAnalyze,
			expectedRequires: []string{"spec.yaml", "plan.yaml", "tasks.yaml"},
			expectedProduces: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(string(tt.stage), func(t *testing.T) {
			dep := GetArtifactDependency(tt.stage)
			if dep.Stage != tt.stage {
				t.Errorf("GetArtifactDependency(%s).Stage = %s, want %s", tt.stage, dep.Stage, tt.stage)
			}
			if len(dep.Requires) != len(tt.expectedRequires) {
				t.Errorf("GetArtifactDependency(%s).Requires = %v, want %v", tt.stage, dep.Requires, tt.expectedRequires)
			}
			if len(dep.Produces) != len(tt.expectedProduces) {
				t.Errorf("GetArtifactDependency(%s).Produces = %v, want %v", tt.stage, dep.Produces, tt.expectedProduces)
			}
		})
	}
}

func TestGetRequiredArtifacts(t *testing.T) {
	tests := []struct {
		stage    Stage
		expected []string
	}{
		// Core stages
		{StageSpecify, []string{}},
		{StagePlan, []string{"spec.yaml"}},
		{StageTasks, []string{"plan.yaml"}},
		{StageImplement, []string{"tasks.yaml"}},
		// Optional stages
		{StageConstitution, []string{}},
		{StageClarify, []string{"spec.yaml"}},
		{StageChecklist, []string{"spec.yaml"}},
		{StageAnalyze, []string{"spec.yaml", "plan.yaml", "tasks.yaml"}},
	}

	for _, tt := range tests {
		t.Run(string(tt.stage), func(t *testing.T) {
			got := GetRequiredArtifacts(tt.stage)
			if len(got) != len(tt.expected) {
				t.Errorf("GetRequiredArtifacts(%s) = %v, want %v", tt.stage, got, tt.expected)
			}
		})
	}
}

func TestGetProducedArtifacts(t *testing.T) {
	tests := []struct {
		stage    Stage
		expected []string
	}{
		// Core stages
		{StageSpecify, []string{"spec.yaml"}},
		{StagePlan, []string{"plan.yaml"}},
		{StageTasks, []string{"tasks.yaml"}},
		{StageImplement, []string{}},
		// Optional stages (none produce tracked artifacts)
		{StageConstitution, []string{}},
		{StageClarify, []string{}},
		{StageChecklist, []string{}},
		{StageAnalyze, []string{}},
	}

	for _, tt := range tests {
		t.Run(string(tt.stage), func(t *testing.T) {
			got := GetProducedArtifacts(tt.stage)
			if len(got) != len(tt.expected) {
				t.Errorf("GetProducedArtifacts(%s) = %v, want %v", tt.stage, got, tt.expected)
			}
		})
	}
}

func TestGetAllRequiredArtifacts(t *testing.T) {
	tests := []struct {
		name     string
		config   StageConfig
		expected []string
	}{
		// Core stages
		{
			name:     "all core stages - no external requirements",
			config:   StageConfig{Specify: true, Plan: true, Tasks: true, Implement: true},
			expected: []string{}, // specify produces what plan needs, etc.
		},
		{
			name:     "only plan - requires spec.yaml",
			config:   StageConfig{Plan: true},
			expected: []string{"spec.yaml"},
		},
		{
			name:     "only tasks - requires plan.yaml",
			config:   StageConfig{Tasks: true},
			expected: []string{"plan.yaml"},
		},
		{
			name:     "only implement - requires tasks.yaml",
			config:   StageConfig{Implement: true},
			expected: []string{"tasks.yaml"},
		},
		{
			name:     "plan and implement - requires spec.yaml (tasks.yaml covered by plan)",
			config:   StageConfig{Plan: true, Implement: true},
			expected: []string{"spec.yaml", "tasks.yaml"},
		},
		{
			name:     "specify only - no requirements",
			config:   StageConfig{Specify: true},
			expected: []string{},
		},
		// Optional stages
		{
			name:     "constitution only - no requirements",
			config:   StageConfig{Constitution: true},
			expected: []string{},
		},
		{
			name:     "clarify only - requires spec.yaml",
			config:   StageConfig{Clarify: true},
			expected: []string{"spec.yaml"},
		},
		{
			name:     "checklist only - requires spec.yaml",
			config:   StageConfig{Checklist: true},
			expected: []string{"spec.yaml"},
		},
		{
			name:     "analyze only - requires spec.yaml, plan.yaml, tasks.yaml",
			config:   StageConfig{Analyze: true},
			expected: []string{"spec.yaml", "plan.yaml", "tasks.yaml"},
		},
		// Mixed core and optional stages
		{
			name:     "specify with clarify - no external requirements (specify produces spec.yaml)",
			config:   StageConfig{Specify: true, Clarify: true},
			expected: []string{},
		},
		{
			name:     "tasks with checklist and analyze - requires plan.yaml (spec.yaml for checklist/analyze)",
			config:   StageConfig{Tasks: true, Checklist: true, Analyze: true},
			expected: []string{"plan.yaml", "spec.yaml"},
		},
		{
			name:     "full workflow with all 8 stages - no external requirements",
			config:   StageConfig{Constitution: true, Specify: true, Clarify: true, Plan: true, Tasks: true, Checklist: true, Analyze: true, Implement: true},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.GetAllRequiredArtifacts()

			// Convert to maps for comparison (order doesn't matter)
			gotMap := make(map[string]bool)
			for _, a := range got {
				gotMap[a] = true
			}
			expectedMap := make(map[string]bool)
			for _, a := range tt.expected {
				expectedMap[a] = true
			}

			if len(gotMap) != len(expectedMap) {
				t.Errorf("GetAllRequiredArtifacts() = %v, want %v", got, tt.expected)
				return
			}
			for artifact := range expectedMap {
				if !gotMap[artifact] {
					t.Errorf("GetAllRequiredArtifacts() missing %s, got %v", artifact, got)
				}
			}
		})
	}
}
