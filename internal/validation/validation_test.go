// Package validation_test tests artifact file validation and result processing.
// Related: internal/validation/validation.go
// Tags: validation, artifact, spec, plan, tasks, result, exit-code
package validation

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateSpecFile(t *testing.T) {
	tests := map[string]struct {
		setup   func(t *testing.T) string
		wantErr bool
	}{
		"spec.md exists": {
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				specPath := filepath.Join(dir, "spec.md")
				if err := os.WriteFile(specPath, []byte("# Spec"), 0644); err != nil {
					t.Fatal(err)
				}
				return dir
			},
			wantErr: false,
		},
		"spec.md missing": {
			setup: func(t *testing.T) string {
				return t.TempDir()
			},
			wantErr: true,
		},
		"directory doesn't exist": {
			setup: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "nonexistent")
			},
			wantErr: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			specDir := tc.setup(t)
			err := ValidateSpecFile(specDir)
			if (err != nil) != tc.wantErr {
				t.Errorf("ValidateSpecFile() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestValidatePlanFile(t *testing.T) {
	tests := map[string]struct {
		setup   func(t *testing.T) string
		wantErr bool
	}{
		"plan.md exists": {
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				planPath := filepath.Join(dir, "plan.md")
				if err := os.WriteFile(planPath, []byte("# Plan"), 0644); err != nil {
					t.Fatal(err)
				}
				return dir
			},
			wantErr: false,
		},
		"plan.md missing": {
			setup: func(t *testing.T) string {
				return t.TempDir()
			},
			wantErr: true,
		},
		"directory doesn't exist": {
			setup: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "nonexistent")
			},
			wantErr: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			specDir := tc.setup(t)
			err := ValidatePlanFile(specDir)
			if (err != nil) != tc.wantErr {
				t.Errorf("ValidatePlanFile() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestValidateTasksFile(t *testing.T) {
	tests := map[string]struct {
		setup   func(t *testing.T) string
		wantErr bool
	}{
		"tasks.md exists": {
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				tasksPath := filepath.Join(dir, "tasks.md")
				if err := os.WriteFile(tasksPath, []byte("# Tasks"), 0644); err != nil {
					t.Fatal(err)
				}
				return dir
			},
			wantErr: false,
		},
		"tasks.md missing": {
			setup: func(t *testing.T) string {
				return t.TempDir()
			},
			wantErr: true,
		},
		"directory doesn't exist": {
			setup: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "nonexistent")
			},
			wantErr: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			specDir := tc.setup(t)
			err := ValidateTasksFile(specDir)
			if (err != nil) != tc.wantErr {
				t.Errorf("ValidateTasksFile() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestResult_ShouldRetry(t *testing.T) {
	tests := map[string]struct {
		result   *Result
		canRetry bool
		want     bool
	}{
		"successful result - no retry": {
			result:   &Result{Success: true},
			canRetry: true,
			want:     false,
		},
		"failed result - can retry": {
			result:   &Result{Success: false},
			canRetry: true,
			want:     true,
		},
		"failed result - cannot retry": {
			result:   &Result{Success: false},
			canRetry: false,
			want:     false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := tc.result.ShouldRetry(tc.canRetry)
			if got != tc.want {
				t.Errorf("ShouldRetry() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestResult_ExitCode(t *testing.T) {
	tests := map[string]struct {
		result *Result
		want   int
	}{
		"success": {
			result: &Result{Success: true},
			want:   0,
		},
		"missing dependencies": {
			result: &Result{Success: false, Error: "missing dependencies"},
			want:   4,
		},
		"invalid arguments": {
			result: &Result{Success: false, Error: "invalid arguments"},
			want:   3,
		},
		"generic failure": {
			result: &Result{Success: false, Error: "some error"},
			want:   1,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := tc.result.ExitCode()
			if got != tc.want {
				t.Errorf("ExitCode() = %v, want %v", got, tc.want)
			}
		})
	}
}
