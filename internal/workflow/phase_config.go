package workflow

// PhaseConfig represents the user's selected phases for execution.
// It determines which workflow phases (specify, plan, tasks, implement)
// and optional phases (constitution, clarify, checklist, analyze)
// will be executed during a run.
type PhaseConfig struct {
	// Core workflow phases
	Specify   bool
	Plan      bool
	Tasks     bool
	Implement bool

	// Optional phases
	Constitution bool
	Clarify      bool
	Checklist    bool
	Analyze      bool
}

// NewPhaseConfig creates a new PhaseConfig with all phases disabled.
func NewPhaseConfig() *PhaseConfig {
	return &PhaseConfig{}
}

// NewPhaseConfigAll creates a new PhaseConfig with all phases enabled.
func NewPhaseConfigAll() *PhaseConfig {
	return &PhaseConfig{
		Specify:   true,
		Plan:      true,
		Tasks:     true,
		Implement: true,
	}
}

// HasAnyPhase returns true if any phase (core or optional) is selected.
func (pc *PhaseConfig) HasAnyPhase() bool {
	return pc.Specify || pc.Plan || pc.Tasks || pc.Implement ||
		pc.Constitution || pc.Clarify || pc.Checklist || pc.Analyze
}

// GetSelectedPhases returns a slice of selected phases in canonical order.
// The canonical order is always: constitution -> specify -> clarify -> plan -> tasks -> checklist -> analyze -> implement.
func (pc *PhaseConfig) GetSelectedPhases() []Phase {
	phases := make([]Phase, 0, 8)
	if pc.Constitution {
		phases = append(phases, PhaseConstitution)
	}
	if pc.Specify {
		phases = append(phases, PhaseSpecify)
	}
	if pc.Clarify {
		phases = append(phases, PhaseClarify)
	}
	if pc.Plan {
		phases = append(phases, PhasePlan)
	}
	if pc.Tasks {
		phases = append(phases, PhaseTasks)
	}
	if pc.Checklist {
		phases = append(phases, PhaseChecklist)
	}
	if pc.Analyze {
		phases = append(phases, PhaseAnalyze)
	}
	if pc.Implement {
		phases = append(phases, PhaseImplement)
	}
	return phases
}

// GetCanonicalOrder is an alias for GetSelectedPhases that returns phases
// in the canonical execution order:
// constitution -> specify -> clarify -> plan -> tasks -> checklist -> analyze -> implement
// This ensures phases always execute in the correct order regardless of
// the order in which flags were specified.
func (pc *PhaseConfig) GetCanonicalOrder() []Phase {
	return pc.GetSelectedPhases()
}

// SetAll enables all phases.
func (pc *PhaseConfig) SetAll() {
	pc.Specify = true
	pc.Plan = true
	pc.Tasks = true
	pc.Implement = true
}

// Count returns the number of selected phases (core and optional).
func (pc *PhaseConfig) Count() int {
	count := 0
	// Core phases
	if pc.Specify {
		count++
	}
	if pc.Plan {
		count++
	}
	if pc.Tasks {
		count++
	}
	if pc.Implement {
		count++
	}
	// Optional phases
	if pc.Constitution {
		count++
	}
	if pc.Clarify {
		count++
	}
	if pc.Checklist {
		count++
	}
	if pc.Analyze {
		count++
	}
	return count
}

// ArtifactDependency defines the relationship between a phase and its
// required/produced artifacts.
type ArtifactDependency struct {
	Phase    Phase
	Requires []string // Artifacts required before this phase can run
	Produces []string // Artifacts created by this phase
}

// artifactDependencies is the complete dependency map for all phases.
// It defines what each phase requires as input and produces as output.
var artifactDependencies = map[Phase]ArtifactDependency{
	// Core workflow phases
	PhaseSpecify: {
		Phase:    PhaseSpecify,
		Requires: []string{}, // Specify has no prerequisites
		Produces: []string{"spec.yaml"},
	},
	PhasePlan: {
		Phase:    PhasePlan,
		Requires: []string{"spec.yaml"},
		Produces: []string{"plan.yaml"},
	},
	PhaseTasks: {
		Phase:    PhaseTasks,
		Requires: []string{"plan.yaml"},
		Produces: []string{"tasks.yaml"},
	},
	PhaseImplement: {
		Phase:    PhaseImplement,
		Requires: []string{"tasks.yaml"},
		Produces: []string{}, // Implement modifies existing files, doesn't create new artifacts
	},

	// Optional phases
	PhaseConstitution: {
		Phase:    PhaseConstitution,
		Requires: []string{}, // Constitution has no prerequisites
		Produces: []string{}, // Constitution modifies .specify/memory/constitution.md
	},
	PhaseClarify: {
		Phase:    PhaseClarify,
		Requires: []string{"spec.yaml"}, // Clarify refines an existing spec
		Produces: []string{},            // Clarify updates spec.yaml in place
	},
	PhaseChecklist: {
		Phase:    PhaseChecklist,
		Requires: []string{"spec.yaml"}, // Checklist validates spec requirements
		Produces: []string{},            // Checklist creates checklist files in checklists/ dir
	},
	PhaseAnalyze: {
		Phase:    PhaseAnalyze,
		Requires: []string{"spec.yaml", "plan.yaml", "tasks.yaml"}, // Analyze validates all artifacts
		Produces: []string{},                                       // Analyze outputs analysis report
	},
}

// GetArtifactDependencies returns the complete dependency map for all phases.
func GetArtifactDependencies() map[Phase]ArtifactDependency {
	// Return a copy to prevent modification of the internal map
	result := make(map[Phase]ArtifactDependency, len(artifactDependencies))
	for k, v := range artifactDependencies {
		result[k] = v
	}
	return result
}

// GetArtifactDependency returns the artifact dependency for a specific phase.
func GetArtifactDependency(phase Phase) ArtifactDependency {
	return artifactDependencies[phase]
}

// GetRequiredArtifacts returns the required artifacts for a phase.
func GetRequiredArtifacts(phase Phase) []string {
	if dep, ok := artifactDependencies[phase]; ok {
		return dep.Requires
	}
	return []string{}
}

// GetProducedArtifacts returns the artifacts produced by a phase.
func GetProducedArtifacts(phase Phase) []string {
	if dep, ok := artifactDependencies[phase]; ok {
		return dep.Produces
	}
	return []string{}
}

// GetAllRequiredArtifacts returns all artifacts required by the selected phases,
// excluding artifacts that will be produced by earlier selected phases.
func (pc *PhaseConfig) GetAllRequiredArtifacts() []string {
	required := make(map[string]bool)
	produced := make(map[string]bool)

	// Iterate through phases in canonical order
	for _, phase := range pc.GetCanonicalOrder() {
		dep := artifactDependencies[phase]

		// Add requirements that won't be produced by earlier phases
		for _, req := range dep.Requires {
			if !produced[req] {
				required[req] = true
			}
		}

		// Mark what this phase produces
		for _, prod := range dep.Produces {
			produced[prod] = true
		}
	}

	// Convert map to slice
	result := make([]string, 0, len(required))
	for artifact := range required {
		result = append(result, artifact)
	}
	return result
}
