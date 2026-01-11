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
		"valid with run-id and spec-id": {
			args:       []string{"20260110_143022_abc12345", "051-retry-backoff"},
			latestFlag: false,
			wantErr:    false,
		},
		"valid with --latest and spec-id": {
			args:       []string{"051-retry-backoff"},
			latestFlag: true,
			wantErr:    false,
		},
		"missing spec-id without --latest": {
			args:       []string{"20260110_143022_abc12345"},
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
			args:       []string{"run-id", "spec-id", "extra"},
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

func TestResolveLogsArgs_ExplicitRunID(t *testing.T) {
	tests := map[string]struct {
		runID       string
		specID      string
		createRun   bool
		wantErr     bool
		errMatch    string
		expectedRun string
	}{
		"valid run and spec": {
			runID:       "20260110_143022_abc12345",
			specID:      "051-feature",
			createRun:   true,
			wantErr:     false,
			expectedRun: "20260110_143022_abc12345",
		},
		"invalid run-id": {
			runID:     "nonexistent_run",
			specID:    "051-feature",
			createRun: false,
			wantErr:   true,
			errMatch:  "run not found",
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
					RunID:     tt.runID,
					DAGFile:   "test.yaml",
					Status:    dag.RunStatusRunning,
					StartedAt: time.Now(),
					Specs: map[string]*dag.SpecState{
						tt.specID: {SpecID: tt.specID, Status: dag.SpecStatusRunning},
					},
				}
				if err := dag.SaveState(stateDir, run); err != nil {
					t.Fatalf("failed to save state: %v", err)
				}
			}

			runID, specID, err := resolveLogsArgs(
				[]string{tt.runID, tt.specID},
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
				if runID != tt.expectedRun {
					t.Errorf("expected run ID %q, got %q", tt.expectedRun, runID)
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
					RunID:     "20260110_100000_older123",
					Status:    dag.RunStatusCompleted,
					StartedAt: time.Now().Add(-2 * time.Hour),
				},
				{
					RunID:     "20260110_120000_newer456",
					Status:    dag.RunStatusRunning,
					StartedAt: time.Now().Add(-1 * time.Hour),
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
				if err := dag.SaveState(stateDir, run); err != nil {
					t.Fatalf("failed to save state: %v", err)
				}
			}

			runID, specID, err := resolveLogsArgs(
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
				if runID != tt.expectedRunID {
					t.Errorf("expected run ID %q, got %q", tt.expectedRunID, runID)
				}
				if specID != tt.specID {
					t.Errorf("expected spec ID %q, got %q", tt.specID, specID)
				}
			}
		})
	}
}

func TestFindLatestRunID(t *testing.T) {
	tests := map[string]struct {
		runs          []*dag.DAGRun
		wantErr       bool
		errMatch      string
		expectedRunID string
	}{
		"finds latest regardless of status": {
			runs: []*dag.DAGRun{
				{
					RunID:     "20260110_100000_old",
					Status:    dag.RunStatusCompleted,
					StartedAt: time.Now().Add(-3 * time.Hour),
				},
				{
					RunID:     "20260110_120000_latest",
					Status:    dag.RunStatusCompleted,
					StartedAt: time.Now().Add(-1 * time.Hour),
				},
			},
			wantErr:       false,
			expectedRunID: "20260110_120000_latest",
		},
		"running run is latest": {
			runs: []*dag.DAGRun{
				{
					RunID:     "20260110_100000_done",
					Status:    dag.RunStatusCompleted,
					StartedAt: time.Now().Add(-2 * time.Hour),
				},
				{
					RunID:     "20260110_120000_running",
					Status:    dag.RunStatusRunning,
					StartedAt: time.Now().Add(-1 * time.Hour),
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
				if err := dag.SaveState(stateDir, run); err != nil {
					t.Fatalf("failed to save state: %v", err)
				}
			}

			result, err := findLatestRunID(stateDir)
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
				if result != tt.expectedRunID {
					t.Errorf("expected %q, got %q", tt.expectedRunID, result)
				}
			}
		})
	}
}

func TestGetLogPath(t *testing.T) {
	tests := map[string]struct {
		runID      string
		specID     string
		specs      map[string]*dag.SpecState
		wantErr    bool
		errMatch   string
		wantSuffix string
	}{
		"valid spec returns correct path": {
			runID:  "20260110_143022_abc12345",
			specID: "051-retry-backoff",
			specs: map[string]*dag.SpecState{
				"051-retry-backoff": {SpecID: "051-retry-backoff", Status: dag.SpecStatusRunning},
			},
			wantErr:    false,
			wantSuffix: "logs/051-retry-backoff.log",
		},
		"multiple specs selects correct one": {
			runID:  "20260110_143022_abc12345",
			specID: "052-watch-mode",
			specs: map[string]*dag.SpecState{
				"051-retry-backoff": {SpecID: "051-retry-backoff", Status: dag.SpecStatusCompleted},
				"052-watch-mode":    {SpecID: "052-watch-mode", Status: dag.SpecStatusRunning},
				"053-cleanup":       {SpecID: "053-cleanup", Status: dag.SpecStatusPending},
			},
			wantErr:    false,
			wantSuffix: "logs/052-watch-mode.log",
		},
		"invalid spec-id returns error": {
			runID:  "20260110_143022_abc12345",
			specID: "nonexistent-spec",
			specs: map[string]*dag.SpecState{
				"051-retry-backoff": {SpecID: "051-retry-backoff", Status: dag.SpecStatusRunning},
			},
			wantErr:  true,
			errMatch: "spec not found",
		},
		"empty specs returns error": {
			runID:    "20260110_143022_abc12345",
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
				RunID:     tt.runID,
				DAGFile:   "test.yaml",
				Status:    dag.RunStatusRunning,
				StartedAt: time.Now(),
				Specs:     tt.specs,
			}
			if err := dag.SaveState(stateDir, run); err != nil {
				t.Fatalf("failed to save state: %v", err)
			}

			result, err := getLogPath(stateDir, tt.runID, tt.specID)
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

func TestGetLogPath_RunNotFound(t *testing.T) {
	tests := map[string]struct {
		runID    string
		specID   string
		wantErr  bool
		errMatch string
	}{
		"nonexistent run returns error": {
			runID:    "nonexistent_run_id",
			specID:   "any-spec",
			wantErr:  true,
			errMatch: "run not found",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			tmpDir := t.TempDir()
			stateDir := filepath.Join(tmpDir, "state", "dag-runs")
			if err := os.MkdirAll(stateDir, 0o755); err != nil {
				t.Fatalf("failed to create state dir: %v", err)
			}

			_, err := getLogPath(stateDir, tt.runID, tt.specID)
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
