// Package workflow tests phase execution mode logic and options.
// Related: internal/workflow/phase_execution.go
// Tags: workflow, phase, execution, modes, implementation
package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPhaseExecutionOptions_Mode(t *testing.T) {
	tests := map[string]struct {
		opts     PhaseExecutionOptions
		wantMode PhaseExecutionMode
	}{
		"default when no flags set": {
			opts:     PhaseExecutionOptions{},
			wantMode: ModeDefault,
		},
		"all phases mode": {
			opts: PhaseExecutionOptions{
				RunAllPhases: true,
			},
			wantMode: ModeAllPhases,
		},
		"single phase mode": {
			opts: PhaseExecutionOptions{
				SinglePhase: 3,
			},
			wantMode: ModeSinglePhase,
		},
		"from phase mode": {
			opts: PhaseExecutionOptions{
				FromPhase: 2,
			},
			wantMode: ModeFromPhase,
		},
		"all phases takes precedence": {
			opts: PhaseExecutionOptions{
				RunAllPhases: true,
				SinglePhase:  3,
				FromPhase:    2,
			},
			wantMode: ModeAllPhases,
		},
		"single phase over from phase": {
			opts: PhaseExecutionOptions{
				SinglePhase: 3,
				FromPhase:   2,
			},
			wantMode: ModeSinglePhase,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := tc.opts.Mode()
			assert.Equal(t, tc.wantMode, got)
		})
	}
}

func TestPhaseExecutionOptions_IsDefault(t *testing.T) {
	tests := map[string]struct {
		opts PhaseExecutionOptions
		want bool
	}{
		"default when no flags set": {
			opts: PhaseExecutionOptions{},
			want: true,
		},
		"not default with run all phases": {
			opts: PhaseExecutionOptions{
				RunAllPhases: true,
			},
			want: false,
		},
		"not default with single phase": {
			opts: PhaseExecutionOptions{
				SinglePhase: 1,
			},
			want: false,
		},
		"not default with from phase": {
			opts: PhaseExecutionOptions{
				FromPhase: 2,
			},
			want: false,
		},
		"default when phase is 0": {
			opts: PhaseExecutionOptions{
				SinglePhase: 0,
				FromPhase:   0,
			},
			want: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := tc.opts.IsDefault()
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestPhaseExecutionMode_Constants(t *testing.T) {
	// Verify the constants have distinct values
	assert.NotEqual(t, ModeDefault, ModeAllPhases)
	assert.NotEqual(t, ModeDefault, ModeSinglePhase)
	assert.NotEqual(t, ModeDefault, ModeFromPhase)
	assert.NotEqual(t, ModeAllPhases, ModeSinglePhase)
	assert.NotEqual(t, ModeAllPhases, ModeFromPhase)
	assert.NotEqual(t, ModeSinglePhase, ModeFromPhase)
}
