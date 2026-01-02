package validation

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGetRiskStats(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		filename     string
		expectNil    bool
		expectTotal  int
		expectHigh   int
		expectMedium int
		expectLow    int
	}{
		"plan with risks": {
			filename:     "valid.yaml",
			expectNil:    false,
			expectTotal:  2,
			expectHigh:   1,
			expectMedium: 1,
			expectLow:    0,
		},
		"plan with various impact levels": {
			filename:     "risks_valid.yaml",
			expectNil:    false,
			expectTotal:  3,
			expectHigh:   1,
			expectMedium: 1,
			expectLow:    1,
		},
		"plan with empty risks array": {
			filename:  "risks_empty_array.yaml",
			expectNil: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			planPath := filepath.Join("testdata", "plan", tt.filename)
			stats, err := GetRiskStats(planPath)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.expectNil {
				if stats != nil {
					t.Errorf("expected nil stats, got %+v", stats)
				}
				return
			}

			if stats == nil {
				t.Fatal("expected non-nil stats")
			}

			if stats.Total != tt.expectTotal {
				t.Errorf("Total = %d, want %d", stats.Total, tt.expectTotal)
			}
			if stats.High != tt.expectHigh {
				t.Errorf("High = %d, want %d", stats.High, tt.expectHigh)
			}
			if stats.Medium != tt.expectMedium {
				t.Errorf("Medium = %d, want %d", stats.Medium, tt.expectMedium)
			}
			if stats.Low != tt.expectLow {
				t.Errorf("Low = %d, want %d", stats.Low, tt.expectLow)
			}
		})
	}
}

func TestGetRiskStats_NonexistentFile(t *testing.T) {
	t.Parallel()

	stats, err := GetRiskStats("nonexistent/plan.yaml")
	if err != nil {
		t.Errorf("expected no error for nonexistent file, got: %v", err)
	}
	if stats != nil {
		t.Errorf("expected nil stats for nonexistent file, got: %+v", stats)
	}
}

func TestFormatRiskSummary(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		stats    *RiskStats
		contains []string
		isEmpty  bool
	}{
		"nil stats": {
			stats:   nil,
			isEmpty: true,
		},
		"zero total": {
			stats:   &RiskStats{Total: 0},
			isEmpty: true,
		},
		"all high": {
			stats:    &RiskStats{Total: 3, High: 3},
			contains: []string{"3 total", "3 high"},
		},
		"mixed impacts": {
			stats:    &RiskStats{Total: 5, High: 2, Medium: 2, Low: 1},
			contains: []string{"5 total", "2 high", "2 medium", "1 low"},
		},
		"only medium and low": {
			stats:    &RiskStats{Total: 4, Medium: 2, Low: 2},
			contains: []string{"4 total", "2 medium", "2 low"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result := FormatRiskSummary(tt.stats)

			if tt.isEmpty {
				if result != "" {
					t.Errorf("expected empty string, got %q", result)
				}
				return
			}

			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("expected output to contain %q, got %q", expected, result)
				}
			}
		})
	}
}

func TestGetPlanFilePath(t *testing.T) {
	t.Parallel()

	path := GetPlanFilePath("/some/spec/dir")
	expected := filepath.Join("/some/spec/dir", "plan.yaml")

	if path != expected {
		t.Errorf("GetPlanFilePath() = %q, want %q", path, expected)
	}
}

func TestGetRiskStats_MalformedYAML(t *testing.T) {
	t.Parallel()

	// Create a temporary file with malformed YAML
	tmpDir := t.TempDir()
	planPath := filepath.Join(tmpDir, "plan.yaml")
	err := os.WriteFile(planPath, []byte("invalid: yaml: :::"), 0o644)
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	_, err = GetRiskStats(planPath)
	if err == nil {
		t.Error("expected error for malformed YAML")
	}
}
