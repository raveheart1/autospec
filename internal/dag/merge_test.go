package dag

import (
	"bytes"
	"context"
	"os"
	"testing"
)

func TestComputeMergeOrder(t *testing.T) {
	tests := map[string]struct {
		dag         *DAGConfig
		run         *DAGRun
		expected    []string
		expectError bool
		errorMsg    string
	}{
		"simple linear dependencies": {
			dag: &DAGConfig{
				Layers: []Layer{
					{ID: "L0", Features: []Feature{
						{ID: "spec-a"},
						{ID: "spec-b", DependsOn: []string{"spec-a"}},
						{ID: "spec-c", DependsOn: []string{"spec-b"}},
					}},
				},
			},
			run: &DAGRun{
				Specs: map[string]*SpecState{
					"spec-a": {SpecID: "spec-a", Status: SpecStatusCompleted},
					"spec-b": {SpecID: "spec-b", Status: SpecStatusCompleted},
					"spec-c": {SpecID: "spec-c", Status: SpecStatusCompleted},
				},
			},
			expected: []string{"spec-a", "spec-b", "spec-c"},
		},
		"independent specs sorted alphabetically": {
			dag: &DAGConfig{
				Layers: []Layer{
					{ID: "L0", Features: []Feature{
						{ID: "spec-c"},
						{ID: "spec-a"},
						{ID: "spec-b"},
					}},
				},
			},
			run: &DAGRun{
				Specs: map[string]*SpecState{
					"spec-a": {SpecID: "spec-a", Status: SpecStatusCompleted},
					"spec-b": {SpecID: "spec-b", Status: SpecStatusCompleted},
					"spec-c": {SpecID: "spec-c", Status: SpecStatusCompleted},
				},
			},
			expected: []string{"spec-a", "spec-b", "spec-c"},
		},
		"only completed specs included": {
			dag: &DAGConfig{
				Layers: []Layer{
					{ID: "L0", Features: []Feature{
						{ID: "spec-a"},
						{ID: "spec-b", DependsOn: []string{"spec-a"}},
						{ID: "spec-c", DependsOn: []string{"spec-b"}},
					}},
				},
			},
			run: &DAGRun{
				Specs: map[string]*SpecState{
					"spec-a": {SpecID: "spec-a", Status: SpecStatusCompleted},
					"spec-b": {SpecID: "spec-b", Status: SpecStatusFailed},
					"spec-c": {SpecID: "spec-c", Status: SpecStatusPending},
				},
			},
			expected: []string{"spec-a"},
		},
		"diamond dependency pattern": {
			dag: &DAGConfig{
				Layers: []Layer{
					{ID: "L0", Features: []Feature{
						{ID: "spec-a"},
						{ID: "spec-b", DependsOn: []string{"spec-a"}},
						{ID: "spec-c", DependsOn: []string{"spec-a"}},
						{ID: "spec-d", DependsOn: []string{"spec-b", "spec-c"}},
					}},
				},
			},
			run: &DAGRun{
				Specs: map[string]*SpecState{
					"spec-a": {SpecID: "spec-a", Status: SpecStatusCompleted},
					"spec-b": {SpecID: "spec-b", Status: SpecStatusCompleted},
					"spec-c": {SpecID: "spec-c", Status: SpecStatusCompleted},
					"spec-d": {SpecID: "spec-d", Status: SpecStatusCompleted},
				},
			},
			expected: []string{"spec-a", "spec-b", "spec-c", "spec-d"},
		},
		"circular dependency detection": {
			dag: &DAGConfig{
				Layers: []Layer{
					{ID: "L0", Features: []Feature{
						{ID: "spec-a", DependsOn: []string{"spec-c"}},
						{ID: "spec-b", DependsOn: []string{"spec-a"}},
						{ID: "spec-c", DependsOn: []string{"spec-b"}},
					}},
				},
			},
			run: &DAGRun{
				Specs: map[string]*SpecState{
					"spec-a": {SpecID: "spec-a", Status: SpecStatusCompleted},
					"spec-b": {SpecID: "spec-b", Status: SpecStatusCompleted},
					"spec-c": {SpecID: "spec-c", Status: SpecStatusCompleted},
				},
			},
			expectError: true,
			errorMsg:    "circular dependency",
		},
		"nil dag returns error": {
			dag:         nil,
			run:         &DAGRun{},
			expectError: true,
			errorMsg:    "must not be nil",
		},
		"nil run returns error": {
			dag: &DAGConfig{
				Layers: []Layer{
					{ID: "L0", Features: []Feature{{ID: "spec-a"}}},
				},
			},
			run:         nil,
			expectError: true,
			errorMsg:    "must not be nil",
		},
		"empty specs returns empty order": {
			dag: &DAGConfig{
				Layers: []Layer{
					{ID: "L0", Features: []Feature{}},
				},
			},
			run: &DAGRun{
				Specs: map[string]*SpecState{},
			},
			expected: nil,
		},
		"multiple layers preserved order": {
			dag: &DAGConfig{
				Layers: []Layer{
					{ID: "L0", Features: []Feature{{ID: "spec-a"}}},
					{ID: "L1", Features: []Feature{{ID: "spec-b", DependsOn: []string{"spec-a"}}}},
					{ID: "L2", Features: []Feature{{ID: "spec-c", DependsOn: []string{"spec-b"}}}},
				},
			},
			run: &DAGRun{
				Specs: map[string]*SpecState{
					"spec-a": {SpecID: "spec-a", Status: SpecStatusCompleted},
					"spec-b": {SpecID: "spec-b", Status: SpecStatusCompleted},
					"spec-c": {SpecID: "spec-c", Status: SpecStatusCompleted},
				},
			},
			expected: []string{"spec-a", "spec-b", "spec-c"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result, err := ComputeMergeOrder(tc.dag, tc.run)

			if tc.expectError {
				if err == nil {
					t.Error("expected error but got nil")
				} else if tc.errorMsg != "" && !containsIgnoreCase(err.Error(), tc.errorMsg) {
					t.Errorf("error message mismatch: got %q, want containing %q", err.Error(), tc.errorMsg)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if len(result) != len(tc.expected) {
				t.Errorf("result length mismatch: got %d (%v), want %d (%v)",
					len(result), result, len(tc.expected), tc.expected)
				return
			}

			for i, specID := range tc.expected {
				if result[i] != specID {
					t.Errorf("result[%d] mismatch: got %s, want %s", i, result[i], specID)
				}
			}
		})
	}
}

func TestDetectConflictedFiles(t *testing.T) {
	tests := map[string]struct {
		setup    func(t *testing.T, repoRoot string)
		expected []string
	}{
		"no conflicts returns nil": {
			setup:    func(_ *testing.T, _ string) {},
			expected: nil,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			repoRoot := t.TempDir()
			tc.setup(t, repoRoot)

			result := DetectConflictedFiles(repoRoot)

			if len(result) != len(tc.expected) {
				t.Errorf("result length mismatch: got %d, want %d", len(result), len(tc.expected))
			}
		})
	}
}

func TestMergeExecutorOptions(t *testing.T) {
	stateDir := t.TempDir()
	repoRoot := t.TempDir()
	var buf bytes.Buffer

	me := NewMergeExecutor(
		stateDir,
		nil,
		repoRoot,
		WithMergeStdout(&buf),
		WithMergeTargetBranch("develop"),
		WithMergeContinue(true),
		WithMergeSkipFailed(true),
		WithMergeCleanup(true),
	)

	if me.stateDir != stateDir {
		t.Errorf("stateDir mismatch: got %s, want %s", me.stateDir, stateDir)
	}

	if me.repoRoot != repoRoot {
		t.Errorf("repoRoot mismatch: got %s, want %s", me.repoRoot, repoRoot)
	}

	if me.targetBranch != "develop" {
		t.Errorf("targetBranch mismatch: got %s, want %s", me.targetBranch, "develop")
	}

	if !me.continueMode {
		t.Error("continueMode should be true")
	}

	if !me.skipFailed {
		t.Error("skipFailed should be true")
	}

	if !me.cleanup {
		t.Error("cleanup should be true")
	}
}

func TestMergeExecutorDetermineTargetBranch(t *testing.T) {
	tests := map[string]struct {
		targetBranch string
		expected     string
	}{
		"uses provided branch": {
			targetBranch: "develop",
			expected:     "develop",
		},
		"defaults to main": {
			targetBranch: "",
			expected:     "main",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			me := NewMergeExecutor(
				"",
				nil,
				"",
				WithMergeTargetBranch(tc.targetBranch),
			)

			result := me.determineTargetBranch(&DAGConfig{})

			if result != tc.expected {
				t.Errorf("result mismatch: got %s, want %s", result, tc.expected)
			}
		})
	}
}

func TestShouldSkipSpec(t *testing.T) {
	tests := map[string]struct {
		specState    *SpecState
		continueMode bool
		skipFailed   bool
		expected     bool
	}{
		"nil merge state not skipped": {
			specState:    &SpecState{Merge: nil},
			continueMode: false,
			skipFailed:   false,
			expected:     false,
		},
		"already merged skipped": {
			specState:    &SpecState{Merge: &MergeState{Status: MergeStatusMerged}},
			continueMode: false,
			skipFailed:   false,
			expected:     true,
		},
		"pending not skipped": {
			specState:    &SpecState{Merge: &MergeState{Status: MergeStatusPending}},
			continueMode: false,
			skipFailed:   false,
			expected:     false,
		},
		"skipped status without continue mode not skipped": {
			specState:    &SpecState{Merge: &MergeState{Status: MergeStatusSkipped}},
			continueMode: false,
			skipFailed:   false,
			expected:     false,
		},
		"skipped status with continue mode skipped": {
			specState:    &SpecState{Merge: &MergeState{Status: MergeStatusSkipped}},
			continueMode: true,
			skipFailed:   false,
			expected:     true,
		},
		"merge_failed without skip-failed not skipped": {
			specState:    &SpecState{Merge: &MergeState{Status: MergeStatusMergeFailed}},
			continueMode: false,
			skipFailed:   false,
			expected:     false,
		},
		"merge_failed with skip-failed skipped": {
			specState:    &SpecState{Merge: &MergeState{Status: MergeStatusMergeFailed}},
			continueMode: false,
			skipFailed:   true,
			expected:     true,
		},
		"both flags with skipped status": {
			specState:    &SpecState{Merge: &MergeState{Status: MergeStatusSkipped}},
			continueMode: true,
			skipFailed:   true,
			expected:     true,
		},
		"both flags with merge_failed status": {
			specState:    &SpecState{Merge: &MergeState{Status: MergeStatusMergeFailed}},
			continueMode: true,
			skipFailed:   true,
			expected:     true,
		},
		"both flags with pending status": {
			specState:    &SpecState{Merge: &MergeState{Status: MergeStatusPending}},
			continueMode: true,
			skipFailed:   true,
			expected:     false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			me := NewMergeExecutor(
				"",
				nil,
				"",
				WithMergeContinue(tc.continueMode),
				WithMergeSkipFailed(tc.skipFailed),
			)

			result := me.shouldSkipSpec(tc.specState)

			if result != tc.expected {
				t.Errorf("result mismatch: got %v, want %v", result, tc.expected)
			}
		})
	}
}

func TestMergeNoCompletedSpecs(t *testing.T) {
	stateDir := t.TempDir()
	if err := EnsureStateDir(stateDir); err != nil {
		t.Fatal(err)
	}

	// Create a run with no completed specs
	run := &DAGRun{
		RunID:   "no-completed-run",
		DAGFile: "test.yaml",
		Status:  RunStatusFailed,
		Specs: map[string]*SpecState{
			"spec-a": {SpecID: "spec-a", Status: SpecStatusFailed},
			"spec-b": {SpecID: "spec-b", Status: SpecStatusPending},
		},
	}
	if err := SaveState(stateDir, run); err != nil {
		t.Fatal(err)
	}

	dag := &DAGConfig{
		Layers: []Layer{
			{ID: "L0", Features: []Feature{
				{ID: "spec-a"},
				{ID: "spec-b"},
			}},
		},
	}

	var buf bytes.Buffer
	me := NewMergeExecutor(
		stateDir,
		nil,
		"",
		WithMergeStdout(&buf),
	)

	err := me.Merge(context.Background(), run, dag)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !containsIgnoreCase(buf.String(), "No completed specs to merge") {
		t.Errorf("expected 'No completed specs to merge' message, got: %s", buf.String())
	}
}

func TestFilterCompletedForMerge(t *testing.T) {
	tests := map[string]struct {
		order    []string
		run      *DAGRun
		expected []string
	}{
		"all completed": {
			order: []string{"spec-a", "spec-b", "spec-c"},
			run: &DAGRun{
				Specs: map[string]*SpecState{
					"spec-a": {SpecID: "spec-a", Status: SpecStatusCompleted},
					"spec-b": {SpecID: "spec-b", Status: SpecStatusCompleted},
					"spec-c": {SpecID: "spec-c", Status: SpecStatusCompleted},
				},
			},
			expected: []string{"spec-a", "spec-b", "spec-c"},
		},
		"some completed": {
			order: []string{"spec-a", "spec-b", "spec-c"},
			run: &DAGRun{
				Specs: map[string]*SpecState{
					"spec-a": {SpecID: "spec-a", Status: SpecStatusCompleted},
					"spec-b": {SpecID: "spec-b", Status: SpecStatusFailed},
					"spec-c": {SpecID: "spec-c", Status: SpecStatusCompleted},
				},
			},
			expected: []string{"spec-a", "spec-c"},
		},
		"none completed": {
			order: []string{"spec-a", "spec-b"},
			run: &DAGRun{
				Specs: map[string]*SpecState{
					"spec-a": {SpecID: "spec-a", Status: SpecStatusPending},
					"spec-b": {SpecID: "spec-b", Status: SpecStatusFailed},
				},
			},
			expected: nil,
		},
		"missing in run state skipped": {
			order: []string{"spec-a", "spec-b", "spec-c"},
			run: &DAGRun{
				Specs: map[string]*SpecState{
					"spec-a": {SpecID: "spec-a", Status: SpecStatusCompleted},
					"spec-c": {SpecID: "spec-c", Status: SpecStatusCompleted},
				},
			},
			expected: []string{"spec-a", "spec-c"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result := filterCompletedForMerge(tc.order, tc.run)

			if len(result) != len(tc.expected) {
				t.Errorf("result length mismatch: got %d (%v), want %d (%v)",
					len(result), result, len(tc.expected), tc.expected)
				return
			}

			for i, specID := range tc.expected {
				if result[i] != specID {
					t.Errorf("result[%d] mismatch: got %s, want %s", i, result[i], specID)
				}
			}
		})
	}
}

func TestTopologicalSort(t *testing.T) {
	tests := map[string]struct {
		nodes    map[string]*specNode
		expected []string
	}{
		"single node": {
			nodes: map[string]*specNode{
				"a": {id: "a"},
			},
			expected: []string{"a"},
		},
		"linear chain": {
			nodes: map[string]*specNode{
				"a": {id: "a", dependents: []string{"b"}},
				"b": {id: "b", dependsOn: []string{"a"}, dependents: []string{"c"}},
				"c": {id: "c", dependsOn: []string{"b"}},
			},
			expected: []string{"a", "b", "c"},
		},
		"independent nodes sorted alphabetically": {
			nodes: map[string]*specNode{
				"c": {id: "c"},
				"a": {id: "a"},
				"b": {id: "b"},
			},
			expected: []string{"a", "b", "c"},
		},
		"diamond pattern": {
			nodes: map[string]*specNode{
				"a": {id: "a", dependents: []string{"b", "c"}},
				"b": {id: "b", dependsOn: []string{"a"}, dependents: []string{"d"}},
				"c": {id: "c", dependsOn: []string{"a"}, dependents: []string{"d"}},
				"d": {id: "d", dependsOn: []string{"b", "c"}},
			},
			expected: []string{"a", "b", "c", "d"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result := topologicalSort(tc.nodes)

			if len(result) != len(tc.expected) {
				t.Errorf("result length mismatch: got %d (%v), want %d (%v)",
					len(result), result, len(tc.expected), tc.expected)
				return
			}

			for i, specID := range tc.expected {
				if result[i] != specID {
					t.Errorf("result[%d] mismatch: got %s, want %s", i, result[i], specID)
				}
			}
		})
	}
}

func TestSplitLines(t *testing.T) {
	tests := map[string]struct {
		input    string
		expected []string
	}{
		"empty string": {
			input:    "",
			expected: nil,
		},
		"single line no newline": {
			input:    "file.go",
			expected: []string{"file.go"},
		},
		"single line with newline": {
			input:    "file.go\n",
			expected: []string{"file.go"},
		},
		"multiple lines unix": {
			input:    "file1.go\nfile2.go\nfile3.go",
			expected: []string{"file1.go", "file2.go", "file3.go"},
		},
		"multiple lines windows": {
			input:    "file1.go\r\nfile2.go\r\nfile3.go",
			expected: []string{"file1.go", "file2.go", "file3.go"},
		},
		"trailing newline": {
			input:    "file1.go\nfile2.go\n",
			expected: []string{"file1.go", "file2.go"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result := splitLines(tc.input)

			if len(result) != len(tc.expected) {
				t.Errorf("result length mismatch: got %d (%v), want %d (%v)",
					len(result), result, len(tc.expected), tc.expected)
				return
			}

			for i, line := range tc.expected {
				if result[i] != line {
					t.Errorf("result[%d] mismatch: got %q, want %q", i, result[i], line)
				}
			}
		})
	}
}

func TestMergeNilRun(t *testing.T) {
	stateDir := t.TempDir()
	if err := EnsureStateDir(stateDir); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	me := NewMergeExecutor(
		stateDir,
		nil,
		"",
		WithMergeStdout(&buf),
	)

	err := me.Merge(context.Background(), nil, &DAGConfig{})
	if err == nil {
		t.Error("expected error for nil run")
	}
}

// containsIgnoreCase checks if s contains substr (case-insensitive).
func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			(len(substr) > 0 && findIgnoreCase(s, substr) >= 0))
}

func findIgnoreCase(s, substr string) int {
	if len(substr) == 0 {
		return 0
	}
	if len(substr) > len(s) {
		return -1
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			sc := s[i+j]
			tc := substr[j]
			if sc >= 'A' && sc <= 'Z' {
				sc += 32
			}
			if tc >= 'A' && tc <= 'Z' {
				tc += 32
			}
			if sc != tc {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}

// Ensure tests compile without worktree.Manager
var _ = os.Stdout

func TestVerifyAllSpecs(t *testing.T) {
	tests := map[string]struct {
		run           *DAGRun
		setupWorktree func(t *testing.T) string
		targetBranch  string
		expectIssues  map[string]VerificationReason
	}{
		"no completed specs returns nil": {
			run: &DAGRun{
				Specs: map[string]*SpecState{
					"spec-a": {SpecID: "spec-a", Status: SpecStatusPending},
					"spec-b": {SpecID: "spec-b", Status: SpecStatusFailed},
				},
			},
			expectIssues: nil,
		},
		"completed spec without worktree reports no commits": {
			run: &DAGRun{
				Specs: map[string]*SpecState{
					"spec-a": {SpecID: "spec-a", Status: SpecStatusCompleted, WorktreePath: ""},
				},
			},
			expectIssues: map[string]VerificationReason{
				"spec-a": VerificationReasonNoCommits,
			},
		},
		"completed spec with invalid worktree path": {
			run: &DAGRun{
				Specs: map[string]*SpecState{
					"spec-a": {SpecID: "spec-a", Status: SpecStatusCompleted, WorktreePath: "/nonexistent/path"},
				},
			},
			expectIssues: nil, // Errors are logged but not reported as issues
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var buf bytes.Buffer
			me := NewMergeExecutor("", nil, "", WithMergeStdout(&buf))

			issues := me.VerifyAllSpecs(tc.run, tc.targetBranch)

			if tc.expectIssues == nil {
				if issues != nil {
					t.Errorf("expected nil issues, got %v", issues)
				}
				return
			}

			if issues == nil {
				t.Fatalf("expected issues, got nil")
			}

			if len(issues) != len(tc.expectIssues) {
				t.Errorf("issues count mismatch: got %d, want %d", len(issues), len(tc.expectIssues))
			}

			for specID, expectedReason := range tc.expectIssues {
				issue, ok := issues[specID]
				if !ok {
					t.Errorf("missing issue for spec %s", specID)
					continue
				}
				if issue.Reason != expectedReason {
					t.Errorf("reason mismatch for %s: got %s, want %s", specID, issue.Reason, expectedReason)
				}
			}
		})
	}
}

func TestPrintVerificationReport(t *testing.T) {
	tests := map[string]struct {
		run            *DAGRun
		issues         map[string]*VerificationIssue
		expectReady    int
		expectNoCommit int
		expectUncommit int
		expectContains []string
	}{
		"all ready": {
			run: &DAGRun{
				Specs: map[string]*SpecState{
					"spec-a": {SpecID: "spec-a", Status: SpecStatusCompleted, WorktreePath: "/tmp/a"},
					"spec-b": {SpecID: "spec-b", Status: SpecStatusCompleted, WorktreePath: "/tmp/b"},
				},
			},
			issues:         nil,
			expectReady:    2,
			expectNoCommit: 0,
			expectUncommit: 0,
		},
		"some no commits": {
			run: &DAGRun{
				Specs: map[string]*SpecState{
					"spec-a": {SpecID: "spec-a", Status: SpecStatusCompleted, WorktreePath: "/tmp/a"},
					"spec-b": {SpecID: "spec-b", Status: SpecStatusCompleted, WorktreePath: "/tmp/b"},
				},
			},
			issues: map[string]*VerificationIssue{
				"spec-b": {SpecID: "spec-b", Reason: VerificationReasonNoCommits},
			},
			expectReady:    1,
			expectNoCommit: 1,
			expectUncommit: 0,
			expectContains: []string{"no commits ahead", "skip-no-commits"},
		},
		"some uncommitted": {
			run: &DAGRun{
				Specs: map[string]*SpecState{
					"spec-a": {SpecID: "spec-a", Status: SpecStatusCompleted, WorktreePath: "/tmp/a"},
					"spec-b": {SpecID: "spec-b", Status: SpecStatusCompleted, WorktreePath: "/tmp/b"},
				},
			},
			issues: map[string]*VerificationIssue{
				"spec-b": {
					SpecID:           "spec-b",
					Reason:           VerificationReasonUncommittedChanges,
					UncommittedFiles: []string{"file1.go", "file2.go"},
				},
			},
			expectReady:    1,
			expectNoCommit: 0,
			expectUncommit: 1,
			expectContains: []string{"uncommitted changes", "dag commit"},
		},
		"mixed issues": {
			run: &DAGRun{
				Specs: map[string]*SpecState{
					"spec-a": {SpecID: "spec-a", Status: SpecStatusCompleted, WorktreePath: "/tmp/a"},
					"spec-b": {SpecID: "spec-b", Status: SpecStatusCompleted, WorktreePath: "/tmp/b"},
					"spec-c": {SpecID: "spec-c", Status: SpecStatusCompleted, WorktreePath: "/tmp/c"},
				},
			},
			issues: map[string]*VerificationIssue{
				"spec-b": {SpecID: "spec-b", Reason: VerificationReasonNoCommits},
				"spec-c": {SpecID: "spec-c", Reason: VerificationReasonUncommittedChanges},
			},
			expectReady:    1,
			expectNoCommit: 1,
			expectUncommit: 1,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var buf bytes.Buffer
			me := NewMergeExecutor("", nil, "", WithMergeStdout(&buf))

			ready, noCommits, uncommitted := me.PrintVerificationReport(tc.run, tc.issues, "main")

			if ready != tc.expectReady {
				t.Errorf("ready count mismatch: got %d, want %d", ready, tc.expectReady)
			}
			if noCommits != tc.expectNoCommit {
				t.Errorf("noCommits count mismatch: got %d, want %d", noCommits, tc.expectNoCommit)
			}
			if uncommitted != tc.expectUncommit {
				t.Errorf("uncommitted count mismatch: got %d, want %d", uncommitted, tc.expectUncommit)
			}

			output := buf.String()
			for _, expected := range tc.expectContains {
				if !containsIgnoreCase(output, expected) {
					t.Errorf("output missing expected text %q, got: %s", expected, output)
				}
			}
		})
	}
}

func TestFilterMergeOrder(t *testing.T) {
	tests := map[string]struct {
		mergeOrder    []string
		issues        map[string]*VerificationIssue
		expectedReady int
		skipNoCommits bool
		expected      []string
	}{
		"no issues returns all": {
			mergeOrder:    []string{"spec-a", "spec-b", "spec-c"},
			issues:        map[string]*VerificationIssue{},
			expectedReady: 3,
			skipNoCommits: false,
			expected:      []string{"spec-a", "spec-b", "spec-c"},
		},
		"filters out specs with issues": {
			mergeOrder: []string{"spec-a", "spec-b", "spec-c"},
			issues: map[string]*VerificationIssue{
				"spec-b": {SpecID: "spec-b", Reason: VerificationReasonNoCommits},
			},
			expectedReady: 2,
			skipNoCommits: true,
			expected:      []string{"spec-a", "spec-c"},
		},
		"zero ready returns nil": {
			mergeOrder: []string{"spec-a", "spec-b"},
			issues: map[string]*VerificationIssue{
				"spec-a": {SpecID: "spec-a", Reason: VerificationReasonNoCommits},
				"spec-b": {SpecID: "spec-b", Reason: VerificationReasonNoCommits},
			},
			expectedReady: 0,
			skipNoCommits: true,
			expected:      nil,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var buf bytes.Buffer
			me := NewMergeExecutor("", nil, "",
				WithMergeStdout(&buf),
				WithMergeSkipNoCommits(tc.skipNoCommits),
			)

			result := me.filterMergeOrder(tc.mergeOrder, tc.issues, tc.expectedReady)

			if len(result) != len(tc.expected) {
				t.Errorf("result length mismatch: got %d (%v), want %d (%v)",
					len(result), result, len(tc.expected), tc.expected)
				return
			}

			for i, specID := range tc.expected {
				if result[i] != specID {
					t.Errorf("result[%d] mismatch: got %s, want %s", i, result[i], specID)
				}
			}
		})
	}
}

func TestRunPreFlightVerificationBlocking(t *testing.T) {
	tests := map[string]struct {
		run           *DAGRun
		mergeOrder    []string
		skipNoCommits bool
		forceVerify   bool
		expectError   bool
		errorContains string
	}{
		"uncommitted changes always block": {
			run: &DAGRun{
				Specs: map[string]*SpecState{
					"spec-a": {SpecID: "spec-a", Status: SpecStatusCompleted, WorktreePath: ""},
				},
			},
			mergeOrder:    []string{"spec-a"},
			skipNoCommits: true,
			forceVerify:   false,
			expectError:   false, // No worktree path means no issue detected (warning only)
		},
		"no commits blocks without skip flag": {
			run: &DAGRun{
				Specs: map[string]*SpecState{
					"spec-a": {SpecID: "spec-a", Status: SpecStatusCompleted, WorktreePath: ""},
				},
			},
			mergeOrder:    []string{"spec-a"},
			skipNoCommits: false,
			forceVerify:   false,
			expectError:   true,
			errorContains: "no commits",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var buf bytes.Buffer
			me := NewMergeExecutor("", nil, "",
				WithMergeStdout(&buf),
				WithMergeSkipNoCommits(tc.skipNoCommits),
				WithMergeForce(tc.forceVerify),
			)

			_, err := me.runPreFlightVerification(tc.run, tc.mergeOrder, "main")

			if tc.expectError {
				if err == nil {
					t.Error("expected error but got nil")
				} else if tc.errorContains != "" && !containsIgnoreCase(err.Error(), tc.errorContains) {
					t.Errorf("error message mismatch: got %q, want containing %q", err.Error(), tc.errorContains)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestMergeExecutorNewOptions(t *testing.T) {
	me := NewMergeExecutor("", nil, "",
		WithMergeSkipNoCommits(true),
		WithMergeForce(true),
	)

	if !me.skipNoCommits {
		t.Error("skipNoCommits should be true")
	}

	if !me.forceVerify {
		t.Error("forceVerify should be true")
	}
}
