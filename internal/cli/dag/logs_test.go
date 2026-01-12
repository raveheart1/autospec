package dag

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ariel-frischer/autospec/internal/dag"
)

func TestValidateLogsArgs(t *testing.T) {
	tests := map[string]struct {
		args       []string
		latestFlag bool
		wantErr    bool
		errMatch   string
	}{
		"valid with workflow-file and spec-id": {
			args:       []string{"workflow.yaml", "051-retry-backoff"},
			latestFlag: false,
			wantErr:    false,
		},
		"valid with --latest and spec-id": {
			args:       []string{"051-retry-backoff"},
			latestFlag: true,
			wantErr:    false,
		},
		"missing spec-id without --latest": {
			args:       []string{"workflow.yaml"},
			latestFlag: false,
			wantErr:    true,
			errMatch:   "requires 2 arguments",
		},
		"missing spec-id with --latest": {
			args:       []string{},
			latestFlag: true,
			wantErr:    true,
			errMatch:   "requires exactly 1 argument",
		},
		"too many args without --latest": {
			args:       []string{"workflow.yaml", "spec-id", "extra"},
			latestFlag: false,
			wantErr:    true,
			errMatch:   "requires 2 arguments",
		},
		"too many args with --latest": {
			args:       []string{"spec-id", "extra"},
			latestFlag: true,
			wantErr:    true,
			errMatch:   "requires exactly 1 argument",
		},
		"no args without --latest": {
			args:       []string{},
			latestFlag: false,
			wantErr:    true,
			errMatch:   "requires 2 arguments",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			cmd := logsCmd
			if tt.latestFlag {
				if err := cmd.Flags().Set("latest", "true"); err != nil {
					t.Fatalf("failed to set latest flag: %v", err)
				}
				defer func() {
					_ = cmd.Flags().Set("latest", "false")
				}()
			}

			err := validateLogsArgs(cmd, tt.args)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tt.errMatch != "" && !bytes.Contains([]byte(err.Error()), []byte(tt.errMatch)) {
					t.Errorf("expected error containing %q, got %q", tt.errMatch, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestResolveLogsArgs_WorkflowPath(t *testing.T) {
	tests := map[string]struct {
		workflowPath string
		specID       string
		createRun    bool
		wantErr      bool
		errMatch     string
	}{
		"valid workflow and spec": {
			workflowPath: "test-workflow.yaml",
			specID:       "051-feature",
			createRun:    true,
			wantErr:      false,
		},
		"nonexistent workflow": {
			workflowPath: "nonexistent.yaml",
			specID:       "051-feature",
			createRun:    false,
			wantErr:      true,
			errMatch:     "no run found for workflow",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			tmpDir := t.TempDir()
			stateDir := filepath.Join(tmpDir, "state", "dag-runs")
			if err := os.MkdirAll(stateDir, 0o755); err != nil {
				t.Fatalf("failed to create state dir: %v", err)
			}

			if tt.createRun {
				run := &dag.DAGRun{
					RunID:        "20260110_143022_abc12345",
					WorkflowPath: tt.workflowPath,
					DAGFile:      tt.workflowPath,
					Status:       dag.RunStatusRunning,
					StartedAt:    time.Now(),
					Specs: map[string]*dag.SpecState{
						tt.specID: {SpecID: tt.specID, Status: dag.SpecStatusRunning},
					},
				}
				if err := dag.SaveStateByWorkflow(stateDir, run); err != nil {
					t.Fatalf("failed to save state: %v", err)
				}
			}

			run, specID, err := resolveLogsArgs(
				[]string{tt.workflowPath, tt.specID},
				false,
				stateDir,
			)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tt.errMatch != "" && !bytes.Contains([]byte(err.Error()), []byte(tt.errMatch)) {
					t.Errorf("expected error containing %q, got %q", tt.errMatch, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if run == nil {
					t.Error("expected run, got nil")
				} else if run.WorkflowPath != tt.workflowPath {
					t.Errorf("expected workflow path %q, got %q", tt.workflowPath, run.WorkflowPath)
				}
				if specID != tt.specID {
					t.Errorf("expected spec ID %q, got %q", tt.specID, specID)
				}
			}
		})
	}
}

func TestResolveLogsArgs_LatestFlag(t *testing.T) {
	tests := map[string]struct {
		runs          []*dag.DAGRun
		specID        string
		wantErr       bool
		errMatch      string
		expectedRunID string
	}{
		"selects most recent run": {
			runs: []*dag.DAGRun{
				{
					RunID:        "20260110_100000_older123",
					WorkflowPath: "workflow1.yaml",
					Status:       dag.RunStatusCompleted,
					StartedAt:    time.Now().Add(-2 * time.Hour),
				},
				{
					RunID:        "20260110_120000_newer456",
					WorkflowPath: "workflow2.yaml",
					Status:       dag.RunStatusRunning,
					StartedAt:    time.Now().Add(-1 * time.Hour),
				},
			},
			specID:        "051-feature",
			wantErr:       false,
			expectedRunID: "20260110_120000_newer456",
		},
		"no runs exist": {
			runs:     nil,
			specID:   "051-feature",
			wantErr:  true,
			errMatch: "no DAG runs exist",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			tmpDir := t.TempDir()
			stateDir := filepath.Join(tmpDir, "state", "dag-runs")
			if err := os.MkdirAll(stateDir, 0o755); err != nil {
				t.Fatalf("failed to create state dir: %v", err)
			}

			for _, run := range tt.runs {
				if err := dag.SaveStateByWorkflow(stateDir, run); err != nil {
					t.Fatalf("failed to save state: %v", err)
				}
			}

			run, specID, err := resolveLogsArgs(
				[]string{tt.specID},
				true,
				stateDir,
			)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tt.errMatch != "" && !bytes.Contains([]byte(err.Error()), []byte(tt.errMatch)) {
					t.Errorf("expected error containing %q, got %q", tt.errMatch, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if run == nil {
					t.Error("expected run, got nil")
				} else if run.RunID != tt.expectedRunID {
					t.Errorf("expected run ID %q, got %q", tt.expectedRunID, run.RunID)
				}
				if specID != tt.specID {
					t.Errorf("expected spec ID %q, got %q", tt.specID, specID)
				}
			}
		})
	}
}

func TestFindLatestRun(t *testing.T) {
	tests := map[string]struct {
		runs          []*dag.DAGRun
		wantErr       bool
		errMatch      string
		expectedRunID string
	}{
		"finds latest regardless of status": {
			runs: []*dag.DAGRun{
				{
					RunID:        "20260110_100000_old",
					WorkflowPath: "workflow1.yaml",
					Status:       dag.RunStatusCompleted,
					StartedAt:    time.Now().Add(-3 * time.Hour),
				},
				{
					RunID:        "20260110_120000_latest",
					WorkflowPath: "workflow2.yaml",
					Status:       dag.RunStatusCompleted,
					StartedAt:    time.Now().Add(-1 * time.Hour),
				},
			},
			wantErr:       false,
			expectedRunID: "20260110_120000_latest",
		},
		"running run is latest": {
			runs: []*dag.DAGRun{
				{
					RunID:        "20260110_100000_done",
					WorkflowPath: "workflow1.yaml",
					Status:       dag.RunStatusCompleted,
					StartedAt:    time.Now().Add(-2 * time.Hour),
				},
				{
					RunID:        "20260110_120000_running",
					WorkflowPath: "workflow2.yaml",
					Status:       dag.RunStatusRunning,
					StartedAt:    time.Now().Add(-1 * time.Hour),
				},
			},
			wantErr:       false,
			expectedRunID: "20260110_120000_running",
		},
		"no runs returns error": {
			runs:     nil,
			wantErr:  true,
			errMatch: "no DAG runs exist",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			tmpDir := t.TempDir()
			stateDir := filepath.Join(tmpDir, "state", "dag-runs")
			if err := os.MkdirAll(stateDir, 0o755); err != nil {
				t.Fatalf("failed to create state dir: %v", err)
			}

			for _, run := range tt.runs {
				if err := dag.SaveStateByWorkflow(stateDir, run); err != nil {
					t.Fatalf("failed to save state: %v", err)
				}
			}

			result, err := findLatestRun(stateDir)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tt.errMatch != "" && !bytes.Contains([]byte(err.Error()), []byte(tt.errMatch)) {
					t.Errorf("expected error containing %q, got %q", tt.errMatch, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if result == nil {
					t.Error("expected run, got nil")
				} else if result.RunID != tt.expectedRunID {
					t.Errorf("expected %q, got %q", tt.expectedRunID, result.RunID)
				}
			}
		})
	}
}

func TestGetLogPath(t *testing.T) {
	tests := map[string]struct {
		runID      string
		logBase    string
		specID     string
		specs      map[string]*dag.SpecState
		wantErr    bool
		errMatch   string
		wantSuffix string
	}{
		"cache-based path when LogBase and LogFile set": {
			runID:   "20260110_143022_abc12345",
			logBase: "/home/user/.cache/autospec/dag-logs/my-project/my-dag",
			specID:  "051-retry-backoff",
			specs: map[string]*dag.SpecState{
				"051-retry-backoff": {
					SpecID:  "051-retry-backoff",
					Status:  dag.SpecStatusRunning,
					LogFile: "051-retry-backoff.log",
				},
			},
			wantErr:    false,
			wantSuffix: "my-project/my-dag/051-retry-backoff.log",
		},
		"legacy path when LogBase empty": {
			runID:   "20260110_143022_abc12345",
			logBase: "",
			specID:  "051-retry-backoff",
			specs: map[string]*dag.SpecState{
				"051-retry-backoff": {
					SpecID: "051-retry-backoff",
					Status: dag.SpecStatusRunning,
					// LogFile is empty - legacy run
				},
			},
			wantErr:    false,
			wantSuffix: "logs/051-retry-backoff.log",
		},
		"legacy path when LogFile empty": {
			runID:   "20260110_143022_abc12345",
			logBase: "/home/user/.cache/autospec/dag-logs/my-project/my-dag",
			specID:  "051-retry-backoff",
			specs: map[string]*dag.SpecState{
				"051-retry-backoff": {
					SpecID: "051-retry-backoff",
					Status: dag.SpecStatusRunning,
					// LogFile is empty despite LogBase being set
				},
			},
			wantErr:    false,
			wantSuffix: "logs/051-retry-backoff.log",
		},
		"multiple specs selects correct cache path": {
			runID:   "20260110_143022_abc12345",
			logBase: "/tmp/cache/dag-logs/project/dag",
			specID:  "052-watch-mode",
			specs: map[string]*dag.SpecState{
				"051-retry-backoff": {
					SpecID:  "051-retry-backoff",
					Status:  dag.SpecStatusCompleted,
					LogFile: "051-retry-backoff.log",
				},
				"052-watch-mode": {
					SpecID:  "052-watch-mode",
					Status:  dag.SpecStatusRunning,
					LogFile: "052-watch-mode.log",
				},
				"053-cleanup": {
					SpecID:  "053-cleanup",
					Status:  dag.SpecStatusPending,
					LogFile: "053-cleanup.log",
				},
			},
			wantErr:    false,
			wantSuffix: "project/dag/052-watch-mode.log",
		},
		"invalid spec-id returns error": {
			runID:   "20260110_143022_abc12345",
			logBase: "/tmp/cache/dag-logs/project/dag",
			specID:  "nonexistent-spec",
			specs: map[string]*dag.SpecState{
				"051-retry-backoff": {
					SpecID:  "051-retry-backoff",
					Status:  dag.SpecStatusRunning,
					LogFile: "051-retry-backoff.log",
				},
			},
			wantErr:  true,
			errMatch: "spec not found",
		},
		"empty specs returns error": {
			runID:    "20260110_143022_abc12345",
			logBase:  "/tmp/cache/dag-logs/project/dag",
			specID:   "any-spec",
			specs:    map[string]*dag.SpecState{},
			wantErr:  true,
			errMatch: "spec not found",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			tmpDir := t.TempDir()
			stateDir := filepath.Join(tmpDir, "state", "dag-runs")
			if err := os.MkdirAll(stateDir, 0o755); err != nil {
				t.Fatalf("failed to create state dir: %v", err)
			}

			run := &dag.DAGRun{
				RunID:        tt.runID,
				WorkflowPath: "test.yaml",
				DAGFile:      "test.yaml",
				LogBase:      tt.logBase,
				Status:       dag.RunStatusRunning,
				StartedAt:    time.Now(),
				Specs:        tt.specs,
			}

			result, err := getLogPath(stateDir, run, tt.specID)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tt.errMatch != "" && !bytes.Contains([]byte(err.Error()), []byte(tt.errMatch)) {
					t.Errorf("expected error containing %q, got %q", tt.errMatch, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if !bytes.HasSuffix([]byte(result), []byte(tt.wantSuffix)) {
					t.Errorf("expected path ending with %q, got %q", tt.wantSuffix, result)
				}
			}
		})
	}
}

func TestLoadRunByWorkflow(t *testing.T) {
	tests := map[string]struct {
		workflowPath string
		createRun    bool
		wantErr      bool
		errMatch     string
	}{
		"existing workflow returns run": {
			workflowPath: "existing.yaml",
			createRun:    true,
			wantErr:      false,
		},
		"nonexistent workflow returns error": {
			workflowPath: "nonexistent.yaml",
			createRun:    false,
			wantErr:      true,
			errMatch:     "no run found for workflow",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			tmpDir := t.TempDir()
			stateDir := filepath.Join(tmpDir, "state", "dag-runs")
			if err := os.MkdirAll(stateDir, 0o755); err != nil {
				t.Fatalf("failed to create state dir: %v", err)
			}

			if tt.createRun {
				run := &dag.DAGRun{
					RunID:        "20260110_143022_abc12345",
					WorkflowPath: tt.workflowPath,
					DAGFile:      tt.workflowPath,
					Status:       dag.RunStatusRunning,
					StartedAt:    time.Now(),
					Specs:        map[string]*dag.SpecState{},
				}
				if err := dag.SaveStateByWorkflow(stateDir, run); err != nil {
					t.Fatalf("failed to save state: %v", err)
				}
			}

			result, err := loadRunByWorkflow(stateDir, tt.workflowPath)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tt.errMatch != "" && !bytes.Contains([]byte(err.Error()), []byte(tt.errMatch)) {
					t.Errorf("expected error containing %q, got %q", tt.errMatch, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if result == nil {
					t.Error("expected run, got nil")
				} else if result.WorkflowPath != tt.workflowPath {
					t.Errorf("expected workflow path %q, got %q", tt.workflowPath, result.WorkflowPath)
				}
			}
		})
	}
}

func TestStreamLogs_NoFollowMode(t *testing.T) {
	tests := map[string]struct {
		logContent    string
		follow        bool
		expectTimeout bool
	}{
		"no-follow dumps and exits": {
			logContent:    "[10:30:45] Starting spec execution\n[10:30:46] Running task 1\n",
			follow:        false,
			expectTimeout: false,
		},
		"empty file no-follow exits immediately": {
			logContent:    "",
			follow:        false,
			expectTimeout: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			tmpDir := t.TempDir()
			logPath := filepath.Join(tmpDir, "test.log")

			if err := os.WriteFile(logPath, []byte(tt.logContent), 0o644); err != nil {
				t.Fatalf("failed to write log file: %v", err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
			defer cancel()

			err := streamLogs(ctx, logPath, tt.follow)
			if tt.expectTimeout {
				if err == nil || err != context.DeadlineExceeded {
					t.Errorf("expected timeout, got: %v", err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestPrintLogLines(t *testing.T) {
	tests := map[string]struct {
		lines    []string
		wantErr  bool
		numLines int
	}{
		"prints multiple lines": {
			lines:    []string{"line 1", "line 2", "line 3"},
			wantErr:  false,
			numLines: 3,
		},
		"handles empty channel": {
			lines:    []string{},
			wantErr:  false,
			numLines: 0,
		},
		"single line": {
			lines:    []string{"only line"},
			wantErr:  false,
			numLines: 1,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			ch := make(chan string, len(tt.lines))
			for _, line := range tt.lines {
				ch <- line
			}
			close(ch)

			err := printLogLines(ch)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}
