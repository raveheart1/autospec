package workflow

import (
	"testing"
)

func TestNewPhaseConfig(t *testing.T) {
	pc := NewPhaseConfig()
	if pc == nil {
		t.Fatal("NewPhaseConfig returned nil")
	}
	if pc.Specify || pc.Plan || pc.Tasks || pc.Implement {
		t.Error("NewPhaseConfig should have all phases disabled")
	}
}

func TestNewPhaseConfigAll(t *testing.T) {
	pc := NewPhaseConfigAll()
	if pc == nil {
		t.Fatal("NewPhaseConfigAll returned nil")
	}
	if !pc.Specify || !pc.Plan || !pc.Tasks || !pc.Implement {
		t.Error("NewPhaseConfigAll should have all phases enabled")
	}
}

func TestHasAnyPhase(t *testing.T) {
	tests := []struct {
		name     string
		config   PhaseConfig
		expected bool
	}{
		{
			name:     "no phases selected",
			config:   PhaseConfig{},
			expected: false,
		},
		{
			name:     "only specify selected",
			config:   PhaseConfig{Specify: true},
			expected: true,
		},
		{
			name:     "only plan selected",
			config:   PhaseConfig{Plan: true},
			expected: true,
		},
		{
			name:     "only tasks selected",
			config:   PhaseConfig{Tasks: true},
			expected: true,
		},
		{
			name:     "only implement selected",
			config:   PhaseConfig{Implement: true},
			expected: true,
		},
		{
			name:     "all phases selected",
			config:   PhaseConfig{Specify: true, Plan: true, Tasks: true, Implement: true},
			expected: true,
		},
		{
			name:     "plan and implement selected",
			config:   PhaseConfig{Plan: true, Implement: true},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.config.HasAnyPhase(); got != tt.expected {
				t.Errorf("HasAnyPhase() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGetSelectedPhases(t *testing.T) {
	tests := []struct {
		name     string
		config   PhaseConfig
		expected []Phase
	}{
		{
			name:     "no phases selected",
			config:   PhaseConfig{},
			expected: []Phase{},
		},
		{
			name:     "only specify",
			config:   PhaseConfig{Specify: true},
			expected: []Phase{PhaseSpecify},
		},
		{
			name:     "plan and implement",
			config:   PhaseConfig{Plan: true, Implement: true},
			expected: []Phase{PhasePlan, PhaseImplement},
		},
		{
			name:     "all phases",
			config:   PhaseConfig{Specify: true, Plan: true, Tasks: true, Implement: true},
			expected: []Phase{PhaseSpecify, PhasePlan, PhaseTasks, PhaseImplement},
		},
		{
			name:     "tasks and implement (skipping earlier phases)",
			config:   PhaseConfig{Tasks: true, Implement: true},
			expected: []Phase{PhaseTasks, PhaseImplement},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.GetSelectedPhases()
			if len(got) != len(tt.expected) {
				t.Errorf("GetSelectedPhases() returned %d phases, want %d", len(got), len(tt.expected))
				return
			}
			for i, phase := range got {
				if phase != tt.expected[i] {
					t.Errorf("GetSelectedPhases()[%d] = %v, want %v", i, phase, tt.expected[i])
				}
			}
		})
	}
}

func TestGetCanonicalOrder(t *testing.T) {
	// The canonical order must always be specify -> plan -> tasks -> implement
	// regardless of how the struct fields are set or in what order

	tests := []struct {
		name     string
		config   PhaseConfig
		expected []Phase
	}{
		{
			name:     "phases set in reverse order",
			config:   PhaseConfig{Implement: true, Tasks: true, Plan: true, Specify: true},
			expected: []Phase{PhaseSpecify, PhasePlan, PhaseTasks, PhaseImplement},
		},
		{
			name:     "only middle phases",
			config:   PhaseConfig{Plan: true, Tasks: true},
			expected: []Phase{PhasePlan, PhaseTasks},
		},
		{
			name:     "only first and last",
			config:   PhaseConfig{Specify: true, Implement: true},
			expected: []Phase{PhaseSpecify, PhaseImplement},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.GetCanonicalOrder()
			if len(got) != len(tt.expected) {
				t.Errorf("GetCanonicalOrder() returned %d phases, want %d", len(got), len(tt.expected))
				return
			}
			for i, phase := range got {
				if phase != tt.expected[i] {
					t.Errorf("GetCanonicalOrder()[%d] = %v, want %v", i, phase, tt.expected[i])
				}
			}
		})
	}
}

func TestSetAll(t *testing.T) {
	pc := &PhaseConfig{}
	pc.SetAll()
	if !pc.Specify || !pc.Plan || !pc.Tasks || !pc.Implement {
		t.Error("SetAll should enable all phases")
	}
}

func TestCount(t *testing.T) {
	tests := []struct {
		name     string
		config   PhaseConfig
		expected int
	}{
		{
			name:     "no phases",
			config:   PhaseConfig{},
			expected: 0,
		},
		{
			name:     "one phase",
			config:   PhaseConfig{Plan: true},
			expected: 1,
		},
		{
			name:     "two phases",
			config:   PhaseConfig{Plan: true, Implement: true},
			expected: 2,
		},
		{
			name:     "all phases",
			config:   PhaseConfig{Specify: true, Plan: true, Tasks: true, Implement: true},
			expected: 4,
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

	if len(deps) != 4 {
		t.Errorf("GetArtifactDependencies() returned %d entries, want 4", len(deps))
	}

	// Verify each phase has a dependency entry
	phases := []Phase{PhaseSpecify, PhasePlan, PhaseTasks, PhaseImplement}
	for _, phase := range phases {
		if _, ok := deps[phase]; !ok {
			t.Errorf("GetArtifactDependencies() missing entry for %s", phase)
		}
	}
}

func TestGetArtifactDependency(t *testing.T) {
	tests := []struct {
		phase            Phase
		expectedRequires []string
		expectedProduces []string
	}{
		{
			phase:            PhaseSpecify,
			expectedRequires: []string{},
			expectedProduces: []string{"spec.yaml"},
		},
		{
			phase:            PhasePlan,
			expectedRequires: []string{"spec.yaml"},
			expectedProduces: []string{"plan.yaml"},
		},
		{
			phase:            PhaseTasks,
			expectedRequires: []string{"plan.yaml"},
			expectedProduces: []string{"tasks.yaml"},
		},
		{
			phase:            PhaseImplement,
			expectedRequires: []string{"tasks.yaml"},
			expectedProduces: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(string(tt.phase), func(t *testing.T) {
			dep := GetArtifactDependency(tt.phase)
			if dep.Phase != tt.phase {
				t.Errorf("GetArtifactDependency(%s).Phase = %s, want %s", tt.phase, dep.Phase, tt.phase)
			}
			if len(dep.Requires) != len(tt.expectedRequires) {
				t.Errorf("GetArtifactDependency(%s).Requires = %v, want %v", tt.phase, dep.Requires, tt.expectedRequires)
			}
			if len(dep.Produces) != len(tt.expectedProduces) {
				t.Errorf("GetArtifactDependency(%s).Produces = %v, want %v", tt.phase, dep.Produces, tt.expectedProduces)
			}
		})
	}
}

func TestGetRequiredArtifacts(t *testing.T) {
	tests := []struct {
		phase    Phase
		expected []string
	}{
		{PhaseSpecify, []string{}},
		{PhasePlan, []string{"spec.yaml"}},
		{PhaseTasks, []string{"plan.yaml"}},
		{PhaseImplement, []string{"tasks.yaml"}},
	}

	for _, tt := range tests {
		t.Run(string(tt.phase), func(t *testing.T) {
			got := GetRequiredArtifacts(tt.phase)
			if len(got) != len(tt.expected) {
				t.Errorf("GetRequiredArtifacts(%s) = %v, want %v", tt.phase, got, tt.expected)
			}
		})
	}
}

func TestGetProducedArtifacts(t *testing.T) {
	tests := []struct {
		phase    Phase
		expected []string
	}{
		{PhaseSpecify, []string{"spec.yaml"}},
		{PhasePlan, []string{"plan.yaml"}},
		{PhaseTasks, []string{"tasks.yaml"}},
		{PhaseImplement, []string{}},
	}

	for _, tt := range tests {
		t.Run(string(tt.phase), func(t *testing.T) {
			got := GetProducedArtifacts(tt.phase)
			if len(got) != len(tt.expected) {
				t.Errorf("GetProducedArtifacts(%s) = %v, want %v", tt.phase, got, tt.expected)
			}
		})
	}
}

func TestGetAllRequiredArtifacts(t *testing.T) {
	tests := []struct {
		name     string
		config   PhaseConfig
		expected []string
	}{
		{
			name:     "all phases - no external requirements",
			config:   PhaseConfig{Specify: true, Plan: true, Tasks: true, Implement: true},
			expected: []string{}, // specify produces what plan needs, etc.
		},
		{
			name:     "only plan - requires spec.yaml",
			config:   PhaseConfig{Plan: true},
			expected: []string{"spec.yaml"},
		},
		{
			name:     "only tasks - requires plan.yaml",
			config:   PhaseConfig{Tasks: true},
			expected: []string{"plan.yaml"},
		},
		{
			name:     "only implement - requires tasks.yaml",
			config:   PhaseConfig{Implement: true},
			expected: []string{"tasks.yaml"},
		},
		{
			name:     "plan and implement - requires spec.yaml (tasks.yaml covered by plan)",
			config:   PhaseConfig{Plan: true, Implement: true},
			expected: []string{"spec.yaml", "tasks.yaml"},
		},
		{
			name:     "specify only - no requirements",
			config:   PhaseConfig{Specify: true},
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
