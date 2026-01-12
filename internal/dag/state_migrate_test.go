package dag

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

func TestMigrateLegacyState(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		setupDAG       func(dagPath string) error
		setupLegacy    func(stateDir, dagPath string) error
		wantMigrated   bool
		wantRunStatus  InlineRunStatus
		wantSpecCount  int
		wantLegacyGone bool
	}{
		"no legacy state exists": {
			setupDAG: func(dagPath string) error {
				return writeMinimalDAG(dagPath)
			},
			setupLegacy:    nil,
			wantMigrated:   false,
			wantRunStatus:  "",
			wantSpecCount:  0,
			wantLegacyGone: true,
		},
		"migrate workflow-based state file": {
			setupDAG: func(dagPath string) error {
				return writeMinimalDAG(dagPath)
			},
			setupLegacy: func(stateDir, dagPath string) error {
				return writeLegacyWorkflowState(stateDir, dagPath)
			},
			wantMigrated:   true,
			wantRunStatus:  InlineRunStatusRunning,
			wantSpecCount:  1,
			wantLegacyGone: true,
		},
		"dag already has inline state - skip migration": {
			setupDAG: func(dagPath string) error {
				return writeDAGWithState(dagPath)
			},
			setupLegacy: func(stateDir, dagPath string) error {
				return writeLegacyWorkflowState(stateDir, dagPath)
			},
			wantMigrated:   false,
			wantRunStatus:  InlineRunStatusCompleted, // existing inline state preserved
			wantSpecCount:  1,
			wantLegacyGone: false, // legacy file not removed when inline state exists
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			tmpDir := t.TempDir()
			dagPath := filepath.Join(tmpDir, "dag.yaml")
			stateDir := filepath.Join(tmpDir, ".autospec", "state", "dag-runs")

			// Setup DAG file
			if tt.setupDAG != nil {
				if err := tt.setupDAG(dagPath); err != nil {
					t.Fatalf("setupDAG failed: %v", err)
				}
			}

			// Setup legacy state
			if tt.setupLegacy != nil {
				if err := os.MkdirAll(stateDir, 0o755); err != nil {
					t.Fatalf("Failed to create state dir: %v", err)
				}
				if err := tt.setupLegacy(stateDir, dagPath); err != nil {
					t.Fatalf("setupLegacy failed: %v", err)
				}
			}

			// Run migration (using the WithDir variant for testing)
			err := MigrateLegacyStateWithDir(dagPath, stateDir)
			if err != nil {
				t.Fatalf("MigrateLegacyStateWithDir() error = %v", err)
			}

			// Verify DAG state
			config, err := LoadDAGConfigFull(dagPath)
			if err != nil {
				t.Fatalf("LoadDAGConfigFull() error = %v", err)
			}

			if tt.wantRunStatus == "" {
				if config.Run != nil {
					t.Error("Run should be nil")
				}
			} else {
				if config.Run == nil {
					t.Fatal("Run should not be nil")
				}
				if config.Run.Status != tt.wantRunStatus {
					t.Errorf("Run.Status: got %v, want %v", config.Run.Status, tt.wantRunStatus)
				}
			}

			if len(config.Specs) != tt.wantSpecCount {
				t.Errorf("Specs count: got %d, want %d", len(config.Specs), tt.wantSpecCount)
			}

			// Verify legacy file removal
			legacyPath := GetStatePathForWorkflow(stateDir, dagPath)
			legacyExists := fileExistsAt(legacyPath)
			if tt.wantLegacyGone && legacyExists {
				t.Error("Legacy state file should have been removed")
			}
		})
	}
}

func TestDetectLegacyStateFile(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		setupLegacy func(stateDir, dagPath string) error
		wantFound   bool
	}{
		"no legacy state": {
			setupLegacy: nil,
			wantFound:   false,
		},
		"workflow-based state file exists": {
			setupLegacy: func(stateDir, dagPath string) error {
				return writeLegacyWorkflowState(stateDir, dagPath)
			},
			wantFound: true,
		},
		"run-id based state file exists": {
			setupLegacy: func(stateDir, dagPath string) error {
				return writeLegacyRunIDState(stateDir, dagPath)
			},
			wantFound: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			tmpDir := t.TempDir()
			dagPath := filepath.Join(tmpDir, "dag.yaml")
			stateDir := filepath.Join(tmpDir, ".autospec", "state", "dag-runs")

			// Create state dir and legacy file
			if tt.setupLegacy != nil {
				if err := os.MkdirAll(stateDir, 0o755); err != nil {
					t.Fatalf("Failed to create state dir: %v", err)
				}
				if err := tt.setupLegacy(stateDir, dagPath); err != nil {
					t.Fatalf("setupLegacy failed: %v", err)
				}
			}

			// Use the WithDir variant for testing
			path, found := DetectLegacyStateFileWithDir(dagPath, stateDir)
			if found != tt.wantFound {
				t.Errorf("DetectLegacyStateFile() found = %v, want %v", found, tt.wantFound)
			}
			if tt.wantFound && path == "" {
				t.Error("Expected non-empty path when found")
			}
		})
	}
}

func TestConvertRunStatus(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		input RunStatus
		want  InlineRunStatus
	}{
		"running":     {input: RunStatusRunning, want: InlineRunStatusRunning},
		"completed":   {input: RunStatusCompleted, want: InlineRunStatusCompleted},
		"failed":      {input: RunStatusFailed, want: InlineRunStatusFailed},
		"interrupted": {input: RunStatusInterrupted, want: InlineRunStatusInterrupted},
		"unknown":     {input: RunStatus("unknown"), want: InlineRunStatusPending},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := convertRunStatus(tt.input)
			if got != tt.want {
				t.Errorf("convertRunStatus(%v) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestConvertSpecStatus(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		input SpecStatus
		want  InlineSpecStatus
	}{
		"pending":   {input: SpecStatusPending, want: InlineSpecStatusPending},
		"running":   {input: SpecStatusRunning, want: InlineSpecStatusRunning},
		"completed": {input: SpecStatusCompleted, want: InlineSpecStatusCompleted},
		"failed":    {input: SpecStatusFailed, want: InlineSpecStatusFailed},
		"blocked":   {input: SpecStatusBlocked, want: InlineSpecStatusBlocked},
		"unknown":   {input: SpecStatus("unknown"), want: InlineSpecStatusPending},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := convertSpecStatus(tt.input)
			if got != tt.want {
				t.Errorf("convertSpecStatus(%v) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestConvertSpecState(t *testing.T) {
	t.Parallel()
	now := time.Now()
	exitCode := 0

	input := &SpecState{
		SpecID:        "spec-a",
		LayerID:       "L0",
		Status:        SpecStatusCompleted,
		WorktreePath:  "/tmp/worktree",
		StartedAt:     &now,
		CompletedAt:   &now,
		CurrentStage:  "implement",
		CommitSHA:     "abc123",
		CommitStatus:  CommitStatusCommitted,
		FailureReason: "",
		ExitCode:      &exitCode,
		Merge:         &MergeState{Status: MergeStatusMerged},
	}

	result := convertSpecState(input)

	if result.Status != InlineSpecStatusCompleted {
		t.Errorf("Status: got %v, want %v", result.Status, InlineSpecStatusCompleted)
	}
	if result.Worktree != "/tmp/worktree" {
		t.Errorf("Worktree: got %v, want %v", result.Worktree, "/tmp/worktree")
	}
	if result.CommitSHA != "abc123" {
		t.Errorf("CommitSHA: got %v, want %v", result.CommitSHA, "abc123")
	}
	if result.CommitStatus != CommitStatusCommitted {
		t.Errorf("CommitStatus: got %v, want %v", result.CommitStatus, CommitStatusCommitted)
	}
	if result.Merge == nil || result.Merge.Status != MergeStatusMerged {
		t.Error("Merge state not converted correctly")
	}
}

func TestConvertStagingBranches(t *testing.T) {
	t.Parallel()
	now := time.Now()

	input := map[string]*StagingBranchInfo{
		"L0": {
			Branch:      "dag/test/stage-L0",
			CreatedAt:   now,
			SpecsMerged: []string{"spec-a", "spec-b"},
		},
		"L1": {
			Branch:      "dag/test/stage-L1",
			CreatedAt:   now,
			SpecsMerged: []string{"spec-c"},
		},
	}

	result := convertStagingBranches(input)

	if len(result) != 2 {
		t.Fatalf("Result length: got %d, want 2", len(result))
	}

	l0 := result["L0"]
	if l0 == nil {
		t.Fatal("L0 should exist in result")
	}
	if l0.Branch != "dag/test/stage-L0" {
		t.Errorf("L0.Branch: got %v, want %v", l0.Branch, "dag/test/stage-L0")
	}
	if len(l0.SpecsMerged) != 2 {
		t.Errorf("L0.SpecsMerged length: got %d, want 2", len(l0.SpecsMerged))
	}
}

func TestConvertStagingBranchesNil(t *testing.T) {
	t.Parallel()
	result := convertStagingBranches(nil)
	if result != nil {
		t.Error("Expected nil for nil input")
	}

	result = convertStagingBranches(map[string]*StagingBranchInfo{})
	if result != nil {
		t.Error("Expected nil for empty input")
	}
}

func TestConvertSpecStatesNil(t *testing.T) {
	t.Parallel()
	result := convertSpecStates(nil)
	if result != nil {
		t.Error("Expected nil for nil input")
	}

	result = convertSpecStates(map[string]*SpecState{})
	if result != nil {
		t.Error("Expected nil for empty input")
	}
}

// Helper functions for test setup

func writeMinimalDAG(path string) error {
	config := &DAGConfig{
		SchemaVersion: "1.0",
		DAG:           DAGMetadata{Name: "Test DAG"},
		Layers: []Layer{{
			ID: "L0",
			Features: []Feature{{
				ID:          "spec-a",
				Description: "Test spec",
			}},
		}},
	}
	return SaveDAGWithState(path, config)
}

func writeDAGWithState(path string) error {
	now := time.Now()
	config := &DAGConfig{
		SchemaVersion: "1.0",
		DAG:           DAGMetadata{Name: "Test DAG"},
		Layers: []Layer{{
			ID: "L0",
			Features: []Feature{{
				ID:          "spec-a",
				Description: "Test spec",
			}},
		}},
		Run: &InlineRunState{
			Status:    InlineRunStatusCompleted,
			StartedAt: &now,
		},
		Specs: map[string]*InlineSpecState{
			"spec-a": {Status: InlineSpecStatusCompleted},
		},
	}
	return SaveDAGWithState(path, config)
}

func writeLegacyWorkflowState(stateDir, dagPath string) error {
	now := time.Now()
	run := &DAGRun{
		WorkflowPath: dagPath,
		DAGFile:      dagPath,
		Status:       RunStatusRunning,
		StartedAt:    now,
		Specs: map[string]*SpecState{
			"spec-a": {
				SpecID:  "spec-a",
				LayerID: "L0",
				Status:  SpecStatusCompleted,
			},
		},
	}

	data, err := yaml.Marshal(run)
	if err != nil {
		return err
	}

	statePath := GetStatePathForWorkflow(stateDir, dagPath)
	return os.WriteFile(statePath, data, 0o644)
}

func writeLegacyRunIDState(stateDir, dagPath string) error {
	now := time.Now()
	run := &DAGRun{
		RunID:        "20240101_120000_abc123",
		WorkflowPath: dagPath,
		DAGFile:      dagPath,
		Status:       RunStatusRunning,
		StartedAt:    now,
		Specs: map[string]*SpecState{
			"spec-a": {
				SpecID:  "spec-a",
				LayerID: "L0",
				Status:  SpecStatusCompleted,
			},
		},
	}

	data, err := yaml.Marshal(run)
	if err != nil {
		return err
	}

	statePath := filepath.Join(stateDir, run.RunID+".yaml")
	return os.WriteFile(statePath, data, 0o644)
}
