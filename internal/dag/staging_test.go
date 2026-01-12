package dag

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStageBranchName(t *testing.T) {
	tests := map[string]struct {
		dagID    string
		layerID  string
		expected string
	}{
		"standard layer": {
			dagID:    "my-dag",
			layerID:  "L0",
			expected: "dag/my-dag/stage-L0",
		},
		"layer with number": {
			dagID:    "feature-dag",
			layerID:  "L1",
			expected: "dag/feature-dag/stage-L1",
		},
		"layer with higher number": {
			dagID:    "complex-dag",
			layerID:  "L10",
			expected: "dag/complex-dag/stage-L10",
		},
		"empty dagID": {
			dagID:    "",
			layerID:  "L0",
			expected: "dag//stage-L0",
		},
		"empty layerID": {
			dagID:    "my-dag",
			layerID:  "",
			expected: "dag/my-dag/stage-",
		},
		"both empty": {
			dagID:    "",
			layerID:  "",
			expected: "dag//stage-",
		},
		"dagID with special characters": {
			dagID:    "dag-with-dashes",
			layerID:  "L2",
			expected: "dag/dag-with-dashes/stage-L2",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := stageBranchName(tt.dagID, tt.layerID)
			if result != tt.expected {
				t.Errorf("stageBranchName(%q, %q) = %q, want %q",
					tt.dagID, tt.layerID, result, tt.expected)
			}
		})
	}
}

func TestMergeConflictError_Error(t *testing.T) {
	tests := map[string]struct {
		err      *MergeConflictError
		contains string
	}{
		"single conflict": {
			err: &MergeConflictError{
				StageBranch: "dag/my-dag/stage-L0",
				SpecBranch:  "dag/my-dag/spec-1",
				SpecID:      "spec-1",
				Conflicts:   []string{"file1.go"},
			},
			contains: "1 file(s)",
		},
		"multiple conflicts": {
			err: &MergeConflictError{
				StageBranch: "dag/my-dag/stage-L0",
				SpecBranch:  "dag/my-dag/spec-2",
				SpecID:      "spec-2",
				Conflicts:   []string{"file1.go", "file2.go", "file3.go"},
			},
			contains: "3 file(s)",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			errMsg := tt.err.Error()
			if errMsg == "" {
				t.Error("Error() returned empty string")
			}
			if !containsSubstring(errMsg, tt.contains) {
				t.Errorf("Error() = %q, want to contain %q", errMsg, tt.contains)
			}
			if !containsSubstring(errMsg, tt.err.SpecID) {
				t.Errorf("Error() = %q, want to contain specID %q", errMsg, tt.err.SpecID)
			}
			if !containsSubstring(errMsg, tt.err.StageBranch) {
				t.Errorf("Error() = %q, want to contain stageBranch %q", errMsg, tt.err.StageBranch)
			}
		})
	}
}

func TestIsMergeInProgress(t *testing.T) {
	tests := map[string]struct {
		setupFunc func(t *testing.T, repoRoot string)
		expected  bool
	}{
		"no merge in progress": {
			setupFunc: func(t *testing.T, repoRoot string) {
				// Create .git directory but no MERGE_HEAD
				gitDir := filepath.Join(repoRoot, ".git")
				if err := os.MkdirAll(gitDir, 0755); err != nil {
					t.Fatalf("failed to create .git dir: %v", err)
				}
			},
			expected: false,
		},
		"merge in progress": {
			setupFunc: func(t *testing.T, repoRoot string) {
				// Create .git directory with MERGE_HEAD
				gitDir := filepath.Join(repoRoot, ".git")
				if err := os.MkdirAll(gitDir, 0755); err != nil {
					t.Fatalf("failed to create .git dir: %v", err)
				}
				mergeHead := filepath.Join(gitDir, "MERGE_HEAD")
				if err := os.WriteFile(mergeHead, []byte("abc123"), 0644); err != nil {
					t.Fatalf("failed to create MERGE_HEAD: %v", err)
				}
			},
			expected: true,
		},
		"no git directory": {
			setupFunc: func(_ *testing.T, _ string) {
				// Don't create .git directory
			},
			expected: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			repoRoot := t.TempDir()
			tt.setupFunc(t, repoRoot)

			result := isMergeInProgress(repoRoot)
			if result != tt.expected {
				t.Errorf("isMergeInProgress() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestValidateMergeResolution(t *testing.T) {
	tests := map[string]struct {
		setupFunc func(t *testing.T, repoRoot string)
		wantErr   bool
		errMsg    string
	}{
		"no merge state - clean repo": {
			setupFunc: func(t *testing.T, repoRoot string) {
				// Create minimal git structure
				gitDir := filepath.Join(repoRoot, ".git")
				if err := os.MkdirAll(gitDir, 0755); err != nil {
					t.Fatalf("failed to create .git dir: %v", err)
				}
			},
			wantErr: false,
		},
		"merge in progress not committed": {
			setupFunc: func(t *testing.T, repoRoot string) {
				// Create .git with MERGE_HEAD (merge started but not committed)
				gitDir := filepath.Join(repoRoot, ".git")
				if err := os.MkdirAll(gitDir, 0755); err != nil {
					t.Fatalf("failed to create .git dir: %v", err)
				}
				mergeHead := filepath.Join(gitDir, "MERGE_HEAD")
				if err := os.WriteFile(mergeHead, []byte("abc123"), 0644); err != nil {
					t.Fatalf("failed to create MERGE_HEAD: %v", err)
				}
			},
			wantErr: true,
			errMsg:  "merge in progress but not committed",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			repoRoot := t.TempDir()
			tt.setupFunc(t, repoRoot)

			err := validateMergeResolution(repoRoot)
			if tt.wantErr {
				if err == nil {
					t.Error("validateMergeResolution() expected error, got nil")
				} else if tt.errMsg != "" && !containsSubstring(err.Error(), tt.errMsg) {
					t.Errorf("error = %q, want to contain %q", err.Error(), tt.errMsg)
				}
			} else if err != nil {
				t.Errorf("validateMergeResolution() unexpected error: %v", err)
			}
		})
	}
}
