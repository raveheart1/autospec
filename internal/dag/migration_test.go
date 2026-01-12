package dag

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMigrateLogs(t *testing.T) {
	tests := map[string]struct {
		setupLegacyLogs bool
		setupCacheLogs  bool
		specIDs         []string
		wantMigrated    int
		wantSkipped     int
		wantError       bool
	}{
		"no legacy logs": {
			setupLegacyLogs: false,
			specIDs:         []string{"spec-001"},
			wantMigrated:    0,
			wantSkipped:     0,
		},
		"single log file migrated": {
			setupLegacyLogs: true,
			specIDs:         []string{"spec-001"},
			wantMigrated:    1,
			wantSkipped:     0,
		},
		"multiple log files migrated": {
			setupLegacyLogs: true,
			specIDs:         []string{"spec-001", "spec-002", "spec-003"},
			wantMigrated:    3,
			wantSkipped:     0,
		},
		"already migrated logs skipped": {
			setupLegacyLogs: true,
			setupCacheLogs:  true,
			specIDs:         []string{"spec-001"},
			wantMigrated:    0,
			wantSkipped:     1,
		},
		"nil run returns error": {
			wantError: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Create temp directories
			stateDir := t.TempDir()
			cacheDir := t.TempDir()

			// Override cache base for testing
			origXDG := os.Getenv("XDG_CACHE_HOME")
			os.Setenv("XDG_CACHE_HOME", cacheDir)
			defer os.Setenv("XDG_CACHE_HOME", origXDG)

			var run *DAGRun
			if !tt.wantError {
				run = &DAGRun{
					RunID:        "test-run",
					WorkflowPath: "workflow.yaml",
					DAGId:        "test-dag",
					ProjectID:    "test-project",
					Specs:        make(map[string]*SpecState),
				}
				for _, specID := range tt.specIDs {
					run.Specs[specID] = &SpecState{SpecID: specID}
				}

				if tt.setupLegacyLogs {
					setupLegacyLogDir(t, stateDir, run.RunID, tt.specIDs)
				}
				if tt.setupCacheLogs {
					setupCacheLogDir(t, cacheDir, run.ProjectID, run.DAGId, tt.specIDs)
				}
			}

			result, err := MigrateLogs(stateDir, run)

			if tt.wantError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.Migrated != tt.wantMigrated {
				t.Errorf("migrated = %d, want %d", result.Migrated, tt.wantMigrated)
			}
			if result.Skipped != tt.wantSkipped {
				t.Errorf("skipped = %d, want %d", result.Skipped, tt.wantSkipped)
			}

			// Verify log files are in cache location
			if tt.setupLegacyLogs && !tt.setupCacheLogs {
				for _, specID := range tt.specIDs {
					cachePath := filepath.Join(cacheDir, "autospec", "dag-logs", run.ProjectID, run.DAGId, specID+".log")
					if !fileExists(cachePath) {
						t.Errorf("expected cache log file at %s", cachePath)
					}
					// Verify legacy file removed
					legacyPath := filepath.Join(stateDir, run.RunID, "logs", specID+".log")
					if fileExists(legacyPath) {
						t.Errorf("expected legacy log file to be removed at %s", legacyPath)
					}
				}
			}

			// Verify spec states updated (only when logs were migrated or skipped)
			if tt.setupLegacyLogs {
				for _, specID := range tt.specIDs {
					if run.Specs[specID].LogFile != specID+".log" {
						t.Errorf("spec %s LogFile = %q, want %q", specID, run.Specs[specID].LogFile, specID+".log")
					}
				}
			}
		})
	}
}

func TestHasOldLogs(t *testing.T) {
	tests := map[string]struct {
		setupLegacyLogs bool
		specIDs         []string
		nilRun          bool
		want            bool
	}{
		"nil run": {
			nilRun: true,
			want:   false,
		},
		"no legacy directory": {
			specIDs: []string{"spec-001"},
			want:    false,
		},
		"legacy logs exist": {
			setupLegacyLogs: true,
			specIDs:         []string{"spec-001"},
			want:            true,
		},
		"legacy dir exists but no log files": {
			setupLegacyLogs: false,
			specIDs:         []string{"spec-001"},
			want:            false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			stateDir := t.TempDir()

			var run *DAGRun
			if !tt.nilRun {
				run = &DAGRun{
					RunID: "test-run",
					Specs: make(map[string]*SpecState),
				}
				for _, specID := range tt.specIDs {
					run.Specs[specID] = &SpecState{SpecID: specID}
				}

				if tt.setupLegacyLogs {
					setupLegacyLogDir(t, stateDir, run.RunID, tt.specIDs)
				}
			}

			got := HasOldLogs(stateDir, run)
			if got != tt.want {
				t.Errorf("HasOldLogs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMoveFile(t *testing.T) {
	tests := map[string]struct {
		srcContent  string
		srcMissing  bool
		dstExists   bool
		wantErr     bool
		wantContent string
	}{
		"successful move": {
			srcContent:  "log content here",
			wantContent: "log content here",
		},
		"source does not exist": {
			srcMissing: true,
			wantErr:    true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			tmpDir := t.TempDir()
			srcPath := filepath.Join(tmpDir, "src.log")
			dstPath := filepath.Join(tmpDir, "dst.log")

			if !tt.srcMissing {
				if err := os.WriteFile(srcPath, []byte(tt.srcContent), 0644); err != nil {
					t.Fatalf("failed to create source file: %v", err)
				}
			}

			bytesWritten, err := moveFile(srcPath, dstPath)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if bytesWritten != int64(len(tt.srcContent)) {
				t.Errorf("bytesWritten = %d, want %d", bytesWritten, len(tt.srcContent))
			}

			// Verify content moved
			content, err := os.ReadFile(dstPath)
			if err != nil {
				t.Fatalf("failed to read destination: %v", err)
			}
			if string(content) != tt.wantContent {
				t.Errorf("content = %q, want %q", string(content), tt.wantContent)
			}

			// Verify source removed
			if fileExists(srcPath) {
				t.Error("source file should be removed after move")
			}
		})
	}
}

func TestDirExists(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "file.txt")
	os.WriteFile(tmpFile, []byte("test"), 0644)

	tests := map[string]struct {
		path string
		want bool
	}{
		"existing directory": {
			path: tmpDir,
			want: true,
		},
		"non-existing path": {
			path: filepath.Join(tmpDir, "nonexistent"),
			want: false,
		},
		"file is not a directory": {
			path: tmpFile,
			want: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if got := dirExists(tt.path); got != tt.want {
				t.Errorf("dirExists(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestFileExists(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "file.txt")
	os.WriteFile(tmpFile, []byte("test"), 0644)

	tests := map[string]struct {
		path string
		want bool
	}{
		"existing file": {
			path: tmpFile,
			want: true,
		},
		"non-existing path": {
			path: filepath.Join(tmpDir, "nonexistent.txt"),
			want: false,
		},
		"directory is not a file": {
			path: tmpDir,
			want: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if got := fileExists(tt.path); got != tt.want {
				t.Errorf("fileExists(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func setupLegacyLogDir(t *testing.T, stateDir, runID string, specIDs []string) {
	t.Helper()
	logDir := filepath.Join(stateDir, runID, "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		t.Fatalf("failed to create legacy log dir: %v", err)
	}
	for _, specID := range specIDs {
		logPath := filepath.Join(logDir, specID+".log")
		content := []byte("log content for " + specID)
		if err := os.WriteFile(logPath, content, 0644); err != nil {
			t.Fatalf("failed to create legacy log file: %v", err)
		}
	}
}

func setupCacheLogDir(t *testing.T, cacheDir, projectID, dagID string, specIDs []string) {
	t.Helper()
	logDir := filepath.Join(cacheDir, "autospec", "dag-logs", projectID, dagID)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		t.Fatalf("failed to create cache log dir: %v", err)
	}
	for _, specID := range specIDs {
		logPath := filepath.Join(logDir, specID+".log")
		content := []byte("cached log content for " + specID)
		if err := os.WriteFile(logPath, content, 0644); err != nil {
			t.Fatalf("failed to create cache log file: %v", err)
		}
	}
}
