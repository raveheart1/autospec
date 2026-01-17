package dag

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ariel-frischer/autospec/internal/dag"
)

func TestListDAGsInDir(t *testing.T) {
	tests := map[string]struct {
		setupFunc func(dir string) error
		wantCount int
		wantErr   bool
	}{
		"empty directory": {
			setupFunc: func(_ string) error { return nil },
			wantCount: 0,
			wantErr:   false,
		},
		"single DAG directory": {
			setupFunc: func(dir string) error {
				return os.MkdirAll(filepath.Join(dir, "dag-1"), 0o755)
			},
			wantCount: 1,
			wantErr:   false,
		},
		"multiple DAG directories": {
			setupFunc: func(dir string) error {
				for _, name := range []string{"dag-1", "dag-2", "dag-3"} {
					if err := os.MkdirAll(filepath.Join(dir, name), 0o755); err != nil {
						return err
					}
				}
				return nil
			},
			wantCount: 3,
			wantErr:   false,
		},
		"ignores files": {
			setupFunc: func(dir string) error {
				if err := os.MkdirAll(filepath.Join(dir, "dag-1"), 0o755); err != nil {
					return err
				}
				return os.WriteFile(filepath.Join(dir, "some-file.txt"), []byte("test"), 0o644)
			},
			wantCount: 1,
			wantErr:   false,
		},
		"nonexistent directory": {
			setupFunc: nil,
			wantCount: 0,
			wantErr:   true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			var testDir string
			if tt.setupFunc != nil {
				testDir = t.TempDir()
				if err := tt.setupFunc(testDir); err != nil {
					t.Fatalf("setup failed: %v", err)
				}
			} else {
				testDir = filepath.Join(t.TempDir(), "nonexistent")
			}

			entries, err := listDAGsInDir(testDir)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if len(entries) != tt.wantCount {
					t.Errorf("expected %d entries, got %d", tt.wantCount, len(entries))
				}
			}
		})
	}
}

func TestCalculateProjectLogSizes(t *testing.T) {
	tests := map[string]struct {
		setupFunc     func(dir string) error
		expectedTotal int64
		expectedDAGs  int
	}{
		"empty directories": {
			setupFunc: func(dir string) error {
				return os.MkdirAll(filepath.Join(dir, "dag-1"), 0o755)
			},
			expectedTotal: 0,
			expectedDAGs:  1,
		},
		"single DAG with log files": {
			setupFunc: func(dir string) error {
				dagDir := filepath.Join(dir, "dag-1")
				if err := os.MkdirAll(dagDir, 0o755); err != nil {
					return err
				}
				return os.WriteFile(filepath.Join(dagDir, "spec-1.log"), []byte("1234567890"), 0o644)
			},
			expectedTotal: 10,
			expectedDAGs:  1,
		},
		"multiple DAGs with log files": {
			setupFunc: func(dir string) error {
				for i := 1; i <= 3; i++ {
					dagDir := filepath.Join(dir, "dag-"+string(rune('0'+i)))
					if err := os.MkdirAll(dagDir, 0o755); err != nil {
						return err
					}
					data := strings.Repeat("x", 100*i) // 100, 200, 300 bytes
					if err := os.WriteFile(filepath.Join(dagDir, "spec.log"), []byte(data), 0o644); err != nil {
						return err
					}
				}
				return nil
			},
			expectedTotal: 600, // 100 + 200 + 300
			expectedDAGs:  3,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			testDir := t.TempDir()
			if err := tt.setupFunc(testDir); err != nil {
				t.Fatalf("setup failed: %v", err)
			}

			entries, err := listDAGsInDir(testDir)
			if err != nil {
				t.Fatalf("listDAGsInDir failed: %v", err)
			}

			total, dagSizes := calculateProjectLogSizes(testDir, entries)

			if total != tt.expectedTotal {
				t.Errorf("expected total %d, got %d", tt.expectedTotal, total)
			}
			if len(dagSizes) != tt.expectedDAGs {
				t.Errorf("expected %d DAGs, got %d", tt.expectedDAGs, len(dagSizes))
			}
		})
	}
}

func TestCollectProjectInfo(t *testing.T) {
	tests := map[string]struct {
		setupFunc        func(dir string) error
		expectedProjects int
	}{
		"no projects": {
			setupFunc: func(_ string) error {
				return nil
			},
			expectedProjects: 0,
		},
		"single project with DAGs": {
			setupFunc: func(dir string) error {
				projectDir := filepath.Join(dir, "project-1", "dag-1")
				if err := os.MkdirAll(projectDir, 0o755); err != nil {
					return err
				}
				return os.WriteFile(filepath.Join(projectDir, "spec.log"), []byte("test"), 0o644)
			},
			expectedProjects: 1,
		},
		"multiple projects with DAGs": {
			setupFunc: func(dir string) error {
				for i := 1; i <= 3; i++ {
					projectDir := filepath.Join(dir, "project-"+string(rune('0'+i)), "dag-1")
					if err := os.MkdirAll(projectDir, 0o755); err != nil {
						return err
					}
					if err := os.WriteFile(filepath.Join(projectDir, "spec.log"), []byte("test"), 0o644); err != nil {
						return err
					}
				}
				return nil
			},
			expectedProjects: 3,
		},
		"empty project directories are skipped": {
			setupFunc: func(dir string) error {
				// Project with empty DAG directory
				if err := os.MkdirAll(filepath.Join(dir, "project-1", "dag-1"), 0o755); err != nil {
					return err
				}
				// Project with actual logs
				projectDir := filepath.Join(dir, "project-2", "dag-1")
				if err := os.MkdirAll(projectDir, 0o755); err != nil {
					return err
				}
				return os.WriteFile(filepath.Join(projectDir, "spec.log"), []byte("test"), 0o644)
			},
			expectedProjects: 1,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			testDir := t.TempDir()
			if err := tt.setupFunc(testDir); err != nil {
				t.Fatalf("setup failed: %v", err)
			}

			projects, _ := os.ReadDir(testDir)
			info := collectProjectInfo(testDir, projects)

			if len(info) != tt.expectedProjects {
				t.Errorf("expected %d projects, got %d", tt.expectedProjects, len(info))
			}
		})
	}
}

func TestDeleteProjectLogs(t *testing.T) {
	tests := map[string]struct {
		setupFunc       func(dir string) error
		expectedDeleted int
		checkFunc       func(t *testing.T, dir string)
	}{
		"delete single DAG": {
			setupFunc: func(dir string) error {
				dagDir := filepath.Join(dir, "dag-1")
				if err := os.MkdirAll(dagDir, 0o755); err != nil {
					return err
				}
				return os.WriteFile(filepath.Join(dagDir, "spec.log"), []byte("test"), 0o644)
			},
			expectedDeleted: 1,
			checkFunc: func(t *testing.T, dir string) {
				if _, err := os.Stat(filepath.Join(dir, "dag-1")); !os.IsNotExist(err) {
					t.Error("dag-1 directory should be deleted")
				}
			},
		},
		"delete multiple DAGs": {
			setupFunc: func(dir string) error {
				for _, name := range []string{"dag-1", "dag-2"} {
					dagDir := filepath.Join(dir, name)
					if err := os.MkdirAll(dagDir, 0o755); err != nil {
						return err
					}
					if err := os.WriteFile(filepath.Join(dagDir, "spec.log"), []byte("test"), 0o644); err != nil {
						return err
					}
				}
				return nil
			},
			expectedDeleted: 2,
			checkFunc: func(t *testing.T, dir string) {
				entries, _ := os.ReadDir(dir)
				if len(entries) != 0 {
					t.Errorf("expected empty directory, got %d entries", len(entries))
				}
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			testDir := t.TempDir()
			if err := tt.setupFunc(testDir); err != nil {
				t.Fatalf("setup failed: %v", err)
			}

			entries, err := listDAGsInDir(testDir)
			if err != nil {
				t.Fatalf("listDAGsInDir failed: %v", err)
			}

			err = deleteProjectLogs(testDir, entries)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if tt.checkFunc != nil {
				tt.checkFunc(t, testDir)
			}
		})
	}
}

func TestDeleteAllProjectLogs(t *testing.T) {
	tests := map[string]struct {
		setupFunc func(dir string) error
		checkFunc func(t *testing.T, dir string)
	}{
		"delete single project": {
			setupFunc: func(dir string) error {
				projectDir := filepath.Join(dir, "project-1", "dag-1")
				if err := os.MkdirAll(projectDir, 0o755); err != nil {
					return err
				}
				return os.WriteFile(filepath.Join(projectDir, "spec.log"), []byte("test"), 0o644)
			},
			checkFunc: func(t *testing.T, dir string) {
				if _, err := os.Stat(filepath.Join(dir, "project-1")); !os.IsNotExist(err) {
					t.Error("project-1 directory should be deleted")
				}
			},
		},
		"delete multiple projects": {
			setupFunc: func(dir string) error {
				for _, name := range []string{"project-1", "project-2"} {
					projectDir := filepath.Join(dir, name, "dag-1")
					if err := os.MkdirAll(projectDir, 0o755); err != nil {
						return err
					}
					if err := os.WriteFile(filepath.Join(projectDir, "spec.log"), []byte("test"), 0o644); err != nil {
						return err
					}
				}
				return nil
			},
			checkFunc: func(t *testing.T, dir string) {
				entries, _ := os.ReadDir(dir)
				if len(entries) != 0 {
					t.Errorf("expected empty directory, got %d entries", len(entries))
				}
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			testDir := t.TempDir()
			if err := tt.setupFunc(testDir); err != nil {
				t.Fatalf("setup failed: %v", err)
			}

			projects, _ := os.ReadDir(testDir)
			info := collectProjectInfo(testDir, projects)

			err := deleteAllProjectLogs(testDir, info)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if tt.checkFunc != nil {
				tt.checkFunc(t, testDir)
			}
		})
	}
}

func TestCleanLogsCurrentProject_NoLogs(t *testing.T) {
	// Override cache base to temp directory
	tempDir := t.TempDir()
	originalXDG := os.Getenv("XDG_CACHE_HOME")
	os.Setenv("XDG_CACHE_HOME", tempDir)
	defer os.Setenv("XDG_CACHE_HOME", originalXDG)

	// Create the expected directory structure but leave it empty
	projectID := dag.GetProjectID()
	projectDir := filepath.Join(tempDir, "autospec", "dag-logs", projectID)
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	// Should not error when no logs exist
	err := cleanLogsCurrentProject(true)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCleanLogsAllProjects_NoLogs(t *testing.T) {
	// Override cache base to temp directory
	tempDir := t.TempDir()
	originalXDG := os.Getenv("XDG_CACHE_HOME")
	os.Setenv("XDG_CACHE_HOME", tempDir)
	defer os.Setenv("XDG_CACHE_HOME", originalXDG)

	// Create the expected directory structure but leave it empty
	logBase := filepath.Join(tempDir, "autospec", "dag-logs")
	if err := os.MkdirAll(logBase, 0o755); err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	// Should not error when no logs exist
	err := cleanLogsAllProjects(true)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCleanLogsCurrentProject_WithLogs(t *testing.T) {
	// Override cache base to temp directory
	tempDir := t.TempDir()
	originalXDG := os.Getenv("XDG_CACHE_HOME")
	os.Setenv("XDG_CACHE_HOME", tempDir)
	defer os.Setenv("XDG_CACHE_HOME", originalXDG)

	// Create logs for current project
	projectID := dag.GetProjectID()
	dagDir := filepath.Join(tempDir, "autospec", "dag-logs", projectID, "test-dag")
	if err := os.MkdirAll(dagDir, 0o755); err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	logFile := filepath.Join(dagDir, "spec-1.log")
	if err := os.WriteFile(logFile, []byte("test log content"), 0o644); err != nil {
		t.Fatalf("failed to create log file: %v", err)
	}

	// Force deletion (skip prompt)
	err := cleanLogsCurrentProject(true)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Verify logs were deleted
	if _, err := os.Stat(dagDir); !os.IsNotExist(err) {
		t.Error("dag directory should be deleted")
	}
}

func TestCleanLogsAllProjects_WithLogs(t *testing.T) {
	// Override cache base to temp directory
	tempDir := t.TempDir()
	originalXDG := os.Getenv("XDG_CACHE_HOME")
	os.Setenv("XDG_CACHE_HOME", tempDir)
	defer os.Setenv("XDG_CACHE_HOME", originalXDG)

	// Create logs for multiple projects
	logBase := filepath.Join(tempDir, "autospec", "dag-logs")
	for _, project := range []string{"project-1", "project-2"} {
		dagDir := filepath.Join(logBase, project, "test-dag")
		if err := os.MkdirAll(dagDir, 0o755); err != nil {
			t.Fatalf("setup failed: %v", err)
		}
		logFile := filepath.Join(dagDir, "spec.log")
		if err := os.WriteFile(logFile, []byte("test log content"), 0o644); err != nil {
			t.Fatalf("failed to create log file: %v", err)
		}
	}

	// Force deletion (skip prompt)
	err := cleanLogsAllProjects(true)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Verify logs were deleted
	entries, _ := os.ReadDir(logBase)
	if len(entries) != 0 {
		t.Errorf("expected empty log base, got %d entries", len(entries))
	}
}

func TestDagLogInfoStruct(t *testing.T) {
	tests := map[string]struct {
		info dagLogInfo
	}{
		"basic info": {
			info: dagLogInfo{
				dagID:         "test-dag",
				sizeBytes:     1024,
				sizeFormatted: "1KB",
			},
		},
		"zero size": {
			info: dagLogInfo{
				dagID:         "empty-dag",
				sizeBytes:     0,
				sizeFormatted: "0B",
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if tt.info.dagID == "" {
				t.Error("dagID should not be empty")
			}
		})
	}
}

func TestProjectLogInfoStruct(t *testing.T) {
	tests := map[string]struct {
		info projectLogInfo
	}{
		"single DAG": {
			info: projectLogInfo{
				projectID: "test-project",
				dags: []dagLogInfo{
					{dagID: "dag-1", sizeBytes: 100, sizeFormatted: "100B"},
				},
				totalBytes:     100,
				totalFormatted: "100B",
			},
		},
		"multiple DAGs": {
			info: projectLogInfo{
				projectID: "test-project",
				dags: []dagLogInfo{
					{dagID: "dag-1", sizeBytes: 100, sizeFormatted: "100B"},
					{dagID: "dag-2", sizeBytes: 200, sizeFormatted: "200B"},
				},
				totalBytes:     300,
				totalFormatted: "300B",
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if tt.info.projectID == "" {
				t.Error("projectID should not be empty")
			}
			if len(tt.info.dags) == 0 {
				t.Error("dags should not be empty")
			}
		})
	}
}
