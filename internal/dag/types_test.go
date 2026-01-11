package dag

import (
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

func TestMergeStatusConstants(t *testing.T) {
	tests := map[string]struct {
		status   MergeStatus
		expected string
	}{
		"pending":      {status: MergeStatusPending, expected: "pending"},
		"merged":       {status: MergeStatusMerged, expected: "merged"},
		"merge_failed": {status: MergeStatusMergeFailed, expected: "merge_failed"},
		"skipped":      {status: MergeStatusSkipped, expected: "skipped"},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if string(tt.status) != tt.expected {
				t.Errorf("MergeStatus: got %q, want %q", tt.status, tt.expected)
			}
		})
	}
}

func TestMergeStateYAMLSerialization(t *testing.T) {
	now := time.Now().Truncate(time.Second)

	tests := map[string]struct {
		state    MergeState
		contains []string
	}{
		"pending state": {
			state: MergeState{
				Status: MergeStatusPending,
			},
			contains: []string{"status: pending"},
		},
		"merged state with timestamp": {
			state: MergeState{
				Status:           MergeStatusMerged,
				MergedAt:         &now,
				ResolutionMethod: "none",
			},
			contains: []string{"status: merged", "merged_at:", "resolution_method: none"},
		},
		"failed state with conflicts": {
			state: MergeState{
				Status:           MergeStatusMergeFailed,
				Conflicts:        []string{"file1.go", "file2.go"},
				ResolutionMethod: "agent",
				Error:            "merge conflict",
			},
			contains: []string{"status: merge_failed", "conflicts:", "file1.go", "file2.go", "error: merge conflict"},
		},
		"skipped state": {
			state: MergeState{
				Status:           MergeStatusSkipped,
				ResolutionMethod: "skipped",
			},
			contains: []string{"status: skipped", "resolution_method: skipped"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			data, err := yaml.Marshal(tt.state)
			if err != nil {
				t.Fatalf("Failed to marshal MergeState: %v", err)
			}

			yamlStr := string(data)
			for _, s := range tt.contains {
				if !containsString(yamlStr, s) {
					t.Errorf("YAML output missing %q in:\n%s", s, yamlStr)
				}
			}
		})
	}
}

func TestMergeStateYAMLRoundTrip(t *testing.T) {
	now := time.Now().Truncate(time.Second)

	tests := map[string]struct {
		state MergeState
	}{
		"minimal state": {
			state: MergeState{
				Status: MergeStatusPending,
			},
		},
		"full state": {
			state: MergeState{
				Status:           MergeStatusMergeFailed,
				MergedAt:         &now,
				Conflicts:        []string{"file1.go", "file2.go"},
				ResolutionMethod: "manual",
				Error:            "unresolved conflicts",
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Marshal
			data, err := yaml.Marshal(tt.state)
			if err != nil {
				t.Fatalf("Failed to marshal: %v", err)
			}

			// Unmarshal
			var decoded MergeState
			if err := yaml.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			// Verify fields
			if decoded.Status != tt.state.Status {
				t.Errorf("Status: got %v, want %v", decoded.Status, tt.state.Status)
			}
			if decoded.ResolutionMethod != tt.state.ResolutionMethod {
				t.Errorf("ResolutionMethod: got %v, want %v", decoded.ResolutionMethod, tt.state.ResolutionMethod)
			}
			if decoded.Error != tt.state.Error {
				t.Errorf("Error: got %v, want %v", decoded.Error, tt.state.Error)
			}
			if len(decoded.Conflicts) != len(tt.state.Conflicts) {
				t.Errorf("Conflicts length: got %d, want %d", len(decoded.Conflicts), len(tt.state.Conflicts))
			}
		})
	}
}

// containsString checks if s contains substring.
func containsString(s, substring string) bool {
	return len(s) >= len(substring) && (s == substring || len(s) > 0 && containsSubstring(s, substring))
}

func containsSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
