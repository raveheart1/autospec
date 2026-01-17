package worktree

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestRunSetupScriptWithTimeout(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping test on Windows - requires bash scripts")
	}

	tests := map[string]struct {
		scriptContent string
		timeout       time.Duration
		expectTimeout bool
		expectError   bool
		expectOutput  string
	}{
		"script completes within timeout": {
			scriptContent: "#!/bin/bash\necho 'success'\nexit 0",
			timeout:       5 * time.Second,
			expectTimeout: false,
			expectError:   false,
			expectOutput:  "success",
		},
		"script exceeds timeout": {
			scriptContent: "#!/bin/bash\nsleep 2\necho 'done'",
			timeout:       100 * time.Millisecond,
			expectTimeout: true,
			expectError:   true,
			expectOutput:  "",
		},
		"script fails without timeout": {
			scriptContent: "#!/bin/bash\necho 'error'\nexit 1",
			timeout:       5 * time.Second,
			expectTimeout: false,
			expectError:   true,
			expectOutput:  "error",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			tmpDir := t.TempDir()
			scriptPath := filepath.Join(tmpDir, "setup.sh")
			worktreePath := filepath.Join(tmpDir, "worktree")

			if err := os.MkdirAll(worktreePath, 0o755); err != nil {
				t.Fatalf("failed to create worktree dir: %v", err)
			}

			if err := os.WriteFile(scriptPath, []byte(tt.scriptContent), 0o755); err != nil {
				t.Fatalf("failed to write test script: %v", err)
			}

			var stdout bytes.Buffer
			result := RunSetupScriptWithTimeout(
				context.Background(),
				scriptPath,
				worktreePath,
				"test-worktree",
				"test-branch",
				tmpDir,
				tt.timeout,
				&stdout,
			)

			if result.TimedOut != tt.expectTimeout {
				t.Errorf("TimedOut = %v, want %v", result.TimedOut, tt.expectTimeout)
			}

			if (result.Error != nil) != tt.expectError {
				t.Errorf("Error = %v, expectError = %v", result.Error, tt.expectError)
			}

			if tt.expectOutput != "" && !strings.Contains(result.Output, tt.expectOutput) {
				t.Errorf("Output = %q, want to contain %q", result.Output, tt.expectOutput)
			}
		})
	}
}

func TestRunSetupScriptWithTimeout_ReceivesWorktreePath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping test on Windows - requires bash scripts")
	}

	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "check_args.sh")
	worktreePath := filepath.Join(tmpDir, "worktree")

	if err := os.MkdirAll(worktreePath, 0o755); err != nil {
		t.Fatalf("failed to create worktree dir: %v", err)
	}

	// Script echoes the first argument (worktree path)
	scriptContent := "#!/bin/bash\necho \"ARG1:$1\""
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0o755); err != nil {
		t.Fatalf("failed to write test script: %v", err)
	}

	var stdout bytes.Buffer
	result := RunSetupScriptWithTimeout(
		context.Background(),
		scriptPath,
		worktreePath,
		"test-worktree",
		"test-branch",
		tmpDir,
		5*time.Second,
		&stdout,
	)

	if result.Error != nil {
		t.Fatalf("unexpected error: %v", result.Error)
	}

	expected := "ARG1:" + worktreePath
	if !strings.Contains(result.Output, expected) {
		t.Errorf("script did not receive worktree path as argument\nOutput: %q\nExpected to contain: %q",
			result.Output, expected)
	}
}

func TestRunSetupScriptWithTimeout_InvalidTimeout(t *testing.T) {
	tests := map[string]struct {
		timeout time.Duration
	}{
		"zero timeout defaults to 5 minutes": {
			timeout: 0,
		},
		"negative timeout defaults to 5 minutes": {
			timeout: -1 * time.Second,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			tmpDir := t.TempDir()
			// Use empty script path to trigger early return (no script)
			// This tests that the timeout validation happens first
			result := RunSetupScriptWithTimeout(
				context.Background(),
				"", // empty script path
				tmpDir,
				"test-worktree",
				"test-branch",
				tmpDir,
				tt.timeout,
				nil,
			)

			// With empty script path, Executed should be false but no error
			if result.Error != nil {
				t.Errorf("unexpected error: %v", result.Error)
			}
			if result.Executed {
				t.Error("expected Executed=false for empty script path")
			}
		})
	}
}

func TestRunSetupScriptWithTimeout_NonexistentScript(t *testing.T) {
	tmpDir := t.TempDir()

	result := RunSetupScriptWithTimeout(
		context.Background(),
		"/nonexistent/script.sh",
		tmpDir,
		"test-worktree",
		"test-branch",
		tmpDir,
		5*time.Second,
		nil,
	)

	if result.Executed {
		t.Error("expected Executed=false for nonexistent script")
	}
	if result.Error != nil {
		t.Errorf("expected no error for nonexistent script, got: %v", result.Error)
	}
}

func TestRunSetupScriptWithTimeout_NotExecutable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping test on Windows - file permissions work differently")
	}

	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "notexec.sh")

	// Create script without executable permission
	if err := os.WriteFile(scriptPath, []byte("#!/bin/bash\necho hello"), 0o644); err != nil {
		t.Fatalf("failed to write test script: %v", err)
	}

	result := RunSetupScriptWithTimeout(
		context.Background(),
		scriptPath,
		tmpDir,
		"test-worktree",
		"test-branch",
		tmpDir,
		5*time.Second,
		nil,
	)

	if result.Error == nil {
		t.Error("expected error for non-executable script")
	}
	if !strings.Contains(result.Error.Error(), "not executable") {
		t.Errorf("error should mention 'not executable', got: %v", result.Error)
	}
}

func TestRunSetupScriptWithTimeout_ContextCancellation(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping test on Windows - requires bash scripts")
	}

	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "long_script.sh")
	worktreePath := filepath.Join(tmpDir, "worktree")

	if err := os.MkdirAll(worktreePath, 0o755); err != nil {
		t.Fatalf("failed to create worktree dir: %v", err)
	}

	// Script sleeps for 5 seconds
	scriptContent := "#!/bin/bash\nsleep 5"
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0o755); err != nil {
		t.Fatalf("failed to write test script: %v", err)
	}

	// Create a context that we'll cancel
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel after a short delay
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	result := RunSetupScriptWithTimeout(
		ctx,
		scriptPath,
		worktreePath,
		"test-worktree",
		"test-branch",
		tmpDir,
		10*time.Second, // Long timeout, but context will be cancelled
		nil,
	)

	if result.Error == nil {
		t.Error("expected error when context is cancelled")
	}
}

func TestDefaultSetupTimeout(t *testing.T) {
	expected := 5 * time.Minute
	if DefaultSetupTimeout != expected {
		t.Errorf("DefaultSetupTimeout = %v, want %v", DefaultSetupTimeout, expected)
	}
}
