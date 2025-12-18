// Package main_test tests mock Claude CLI script behavior with configurable responses and delays.
// Related: /home/ari/repos/autospec/mocks/scripts/mock-claude.sh
// Tags: mocks, testing, claude, shell-script

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestMockClaudeReturnsConfiguredResponse verifies mock returns MOCK_RESPONSE_FILE content
func TestMockClaudeReturnsConfiguredResponse(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		responseContent string
		args            []string
		wantOutput      string
	}{
		"simple response": {
			responseContent: "Hello from mock",
			args:            []string{"--print", "test"},
			wantOutput:      "Hello from mock",
		},
		"multiline response": {
			responseContent: "line1\nline2\nline3",
			args:            []string{"-p", "generate"},
			wantOutput:      "line1\nline2\nline3",
		},
		"yaml response": {
			responseContent: "key: value\nitems:\n  - one\n  - two",
			args:            []string{"--print", "yaml"},
			wantOutput:      "key: value\nitems:\n  - one\n  - two",
		},
		"empty response file": {
			responseContent: "",
			args:            []string{"test"},
			wantOutput:      "",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			tmpDir := t.TempDir()
			responseFile := filepath.Join(tmpDir, "response.txt")
			if err := os.WriteFile(responseFile, []byte(tt.responseContent), 0644); err != nil {
				t.Fatalf("failed to write response file: %v", err)
			}

			cmd := exec.Command(getMockClaudePath(t), tt.args...)
			cmd.Env = append(os.Environ(), "MOCK_RESPONSE_FILE="+responseFile)

			output, err := cmd.Output()
			if err != nil {
				t.Fatalf("mock-claude failed: %v", err)
			}

			if string(output) != tt.wantOutput {
				t.Errorf("got output %q, want %q", string(output), tt.wantOutput)
			}
		})
	}
}

// TestMockClaudeLogsCallsToFile verifies mock logs calls to MOCK_CALL_LOG
func TestMockClaudeLogsCallsToFile(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		args      []string
		wantInLog []string
		exitCode  string
	}{
		"logs simple args": {
			args:      []string{"--print", "hello"},
			wantInLog: []string{"--print", "hello"},
			exitCode:  "0",
		},
		"logs multiple args": {
			args:      []string{"-p", "generate", "--format", "yaml"},
			wantInLog: []string{"-p", "generate", "--format", "yaml"},
			exitCode:  "0",
		},
		"logs exit code": {
			args:      []string{"test"},
			wantInLog: []string{"exit_code: 0"},
			exitCode:  "0",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			tmpDir := t.TempDir()
			callLog := filepath.Join(tmpDir, "calls.log")

			cmd := exec.Command(getMockClaudePath(t), tt.args...)
			cmd.Env = append(os.Environ(),
				"MOCK_CALL_LOG="+callLog,
				"MOCK_EXIT_CODE="+tt.exitCode,
			)

			if err := cmd.Run(); err != nil {
				// Check if it's an expected non-zero exit
				if exitErr, ok := err.(*exec.ExitError); ok {
					if tt.exitCode != "0" {
						// Expected non-zero exit
					} else {
						t.Fatalf("mock-claude failed unexpectedly: %v", exitErr)
					}
				} else {
					t.Fatalf("mock-claude failed: %v", err)
				}
			}

			logContent, err := os.ReadFile(callLog)
			if err != nil {
				t.Fatalf("failed to read call log: %v", err)
			}

			for _, want := range tt.wantInLog {
				if !strings.Contains(string(logContent), want) {
					t.Errorf("call log missing %q, got:\n%s", want, logContent)
				}
			}
		})
	}
}

// TestMockClaudeReturnsConfiguredExitCode verifies mock returns MOCK_EXIT_CODE
func TestMockClaudeReturnsConfiguredExitCode(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		exitCode     string
		wantExitCode int
	}{
		"exit 0":  {exitCode: "0", wantExitCode: 0},
		"exit 1":  {exitCode: "1", wantExitCode: 1},
		"exit 2":  {exitCode: "2", wantExitCode: 2},
		"exit 42": {exitCode: "42", wantExitCode: 42},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			cmd := exec.Command(getMockClaudePath(t), "test")
			cmd.Env = append(os.Environ(), "MOCK_EXIT_CODE="+tt.exitCode)

			err := cmd.Run()

			if tt.wantExitCode == 0 {
				if err != nil {
					t.Errorf("expected exit 0, got error: %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("expected exit %d, got exit 0", tt.wantExitCode)
				} else if exitErr, ok := err.(*exec.ExitError); ok {
					if exitErr.ExitCode() != tt.wantExitCode {
						t.Errorf("expected exit %d, got %d", tt.wantExitCode, exitErr.ExitCode())
					}
				} else {
					t.Errorf("unexpected error type: %v", err)
				}
			}
		})
	}
}

// TestMockClaudeDelaysWithMockDelay verifies mock delays response by MOCK_DELAY seconds
func TestMockClaudeDelaysWithMockDelay(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		delay       string
		minDuration time.Duration
		maxDuration time.Duration
	}{
		"no delay": {
			delay:       "0",
			minDuration: 0,
			maxDuration: 500 * time.Millisecond,
		},
		"1 second delay": {
			delay:       "1",
			minDuration: 900 * time.Millisecond,
			maxDuration: 2 * time.Second,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			cmd := exec.Command(getMockClaudePath(t), "test")
			cmd.Env = append(os.Environ(), "MOCK_DELAY="+tt.delay)

			start := time.Now()
			err := cmd.Run()
			duration := time.Since(start)

			if err != nil {
				t.Fatalf("mock-claude failed: %v", err)
			}

			if duration < tt.minDuration {
				t.Errorf("command completed too fast: %v < %v", duration, tt.minDuration)
			}
			if duration > tt.maxDuration {
				t.Errorf("command took too long: %v > %v", duration, tt.maxDuration)
			}
		})
	}
}

// TestMockClaudeNeverMakesNetworkCalls verifies mock makes no network calls
// This is verified by ensuring the script doesn't contain curl, wget, or other network commands
func TestMockClaudeNeverMakesNetworkCalls(t *testing.T) {
	t.Parallel()

	scriptPath := getMockClaudePath(t)
	content, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("failed to read mock-claude.sh: %v", err)
	}

	// Check for network command patterns that would indicate network calls
	networkPatterns := []string{
		"curl ",
		"wget ",
		"nc ",
		"netcat ",
		"http://",
		"https://",
		"api.anthropic.com",
		"claude.ai",
	}

	for _, pattern := range networkPatterns {
		if strings.Contains(string(content), pattern) {
			t.Errorf("mock-claude.sh contains network pattern %q - this violates the no-network-calls requirement", pattern)
		}
	}
}

// TestMockClaudeNoResponseFile verifies mock returns empty output when no response file configured
func TestMockClaudeNoResponseFile(t *testing.T) {
	t.Parallel()

	cmd := exec.Command(getMockClaudePath(t), "--print", "test")
	// Explicitly unset MOCK_RESPONSE_FILE
	env := os.Environ()
	filteredEnv := make([]string, 0, len(env))
	for _, e := range env {
		if !strings.HasPrefix(e, "MOCK_RESPONSE_FILE=") {
			filteredEnv = append(filteredEnv, e)
		}
	}
	cmd.Env = filteredEnv

	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("mock-claude failed: %v", err)
	}

	if len(output) != 0 {
		t.Errorf("expected empty output when no response file, got %q", string(output))
	}
}

// getMockClaudePath returns the path to the mock-claude.sh script
func getMockClaudePath(t *testing.T) string {
	t.Helper()

	// Try relative path from test location
	testPath := "mock-claude.sh"
	if _, err := os.Stat(testPath); err == nil {
		absPath, _ := filepath.Abs(testPath)
		return absPath
	}

	// Try from project root
	projectRoot := findProjectRoot(t)
	mockPath := filepath.Join(projectRoot, "mocks", "scripts", "mock-claude.sh")
	if _, err := os.Stat(mockPath); err == nil {
		return mockPath
	}

	t.Fatalf("could not find mock-claude.sh")
	return ""
}

// findProjectRoot finds the project root by looking for go.mod
func findProjectRoot(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root without finding go.mod
			// Fall back to relative path from current dir
			return "."
		}
		dir = parent
	}
}
