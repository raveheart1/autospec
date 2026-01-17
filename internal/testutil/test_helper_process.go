// Package testutil provides test utilities and helpers for autospec tests.
package testutil

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// HelperProcessConfig configures the behavior of TestHelperProcess.
type HelperProcessConfig struct {
	// ExitCode is the exit code to return (default 0).
	ExitCode int `json:"exit_code"`
	// Stdout is the content to write to stdout.
	Stdout string `json:"stdout"`
	// Stderr is the content to write to stderr.
	Stderr string `json:"stderr"`
}

// HelperProcessEnvVars contains the environment variable names used by TestHelperProcess.
const (
	// EnvWantHelperProcess signals that the test binary should run as a helper process.
	EnvWantHelperProcess = "GO_WANT_HELPER_PROCESS"
	// EnvHelperProcessConfig contains JSON-encoded HelperProcessConfig.
	EnvHelperProcessConfig = "GO_HELPER_PROCESS_CONFIG"
	// EnvHelperProcessArgs contains the original command-line arguments (JSON array).
	EnvHelperProcessArgs = "GO_HELPER_PROCESS_ARGS"
)

// TestHelperProcess is a function to be called from TestMain or a test function
// to implement the helper process pattern. When invoked with GO_WANT_HELPER_PROCESS=1,
// it behaves as a mock subprocess and exits without returning.
//
// Usage in test file:
//
//	func TestHelperProcess(t *testing.T) {
//	    testutil.TestHelperProcess(t)
//	}
//
// This function checks for GO_WANT_HELPER_PROCESS environment variable.
// If set, it parses configuration from GO_HELPER_PROCESS_CONFIG and exits.
// If not set, it returns immediately, allowing normal test execution.
func TestHelperProcess(t *testing.T) {
	if os.Getenv(EnvWantHelperProcess) != "1" {
		return
	}

	config := parseHelperConfig()
	runHelperProcess(config)
	// runHelperProcess calls os.Exit, so this line is never reached
}

// parseHelperConfig parses HelperProcessConfig from environment variable.
func parseHelperConfig() HelperProcessConfig {
	config := HelperProcessConfig{}
	configJSON := os.Getenv(EnvHelperProcessConfig)
	if configJSON != "" {
		// Ignore parse errors; use defaults on failure
		_ = json.Unmarshal([]byte(configJSON), &config)
	}
	return config
}

// runHelperProcess executes the helper process behavior and always exits.
func runHelperProcess(config HelperProcessConfig) {
	// Write configured output
	if config.Stdout != "" {
		fmt.Fprint(os.Stdout, config.Stdout)
	}
	if config.Stderr != "" {
		fmt.Fprint(os.Stderr, config.Stderr)
	}

	// Always exit with configured code (defaults to 0)
	os.Exit(config.ExitCode)
}

// ConfigureTestCommand creates an exec.Cmd that invokes the test binary
// as a helper process instead of the real command.
//
// Parameters:
//   - t: The test context
//   - testName: Name of the test function containing TestHelperProcess call
//   - config: Configuration for the helper process behavior
//   - args: Original arguments that would be passed to the real command
//
// Returns an exec.Cmd configured to run the test binary with helper process env vars.
func ConfigureTestCommand(t *testing.T, testName string, config HelperProcessConfig, args ...string) *exec.Cmd {
	t.Helper()

	// Get the test binary path
	testBinary, err := os.Executable()
	if err != nil {
		t.Fatalf("failed to get test binary path: %v", err)
	}

	// Build the command to run the test binary with -test.run pointing to the helper
	cmd := exec.Command(testBinary, "-test.run="+testName)

	// Set environment variables for helper process
	cmd.Env = buildHelperEnv(t, config, args)

	return cmd
}

// buildHelperEnv constructs the environment variables for helper process.
func buildHelperEnv(t *testing.T, config HelperProcessConfig, args []string) []string {
	t.Helper()

	// Start with current environment
	env := os.Environ()

	// Add helper process marker
	env = append(env, EnvWantHelperProcess+"=1")

	// Add config as JSON
	if configJSON, err := json.Marshal(config); err == nil {
		env = append(env, EnvHelperProcessConfig+"="+string(configJSON))
	}

	// Add original args as JSON
	if argsJSON, err := json.Marshal(args); err == nil {
		env = append(env, EnvHelperProcessArgs+"="+string(argsJSON))
	}

	return env
}

// GetHelperProcessArgs retrieves the original arguments passed to the helper process.
// This is useful inside TestHelperProcess for argument validation.
func GetHelperProcessArgs() ([]string, error) {
	argsJSON := os.Getenv(EnvHelperProcessArgs)
	if argsJSON == "" {
		return nil, nil
	}

	var args []string
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return nil, fmt.Errorf("parsing helper process args: %w", err)
	}
	return args, nil
}

// HelperProcessResult captures the result of running a helper process command.
type HelperProcessResult struct {
	// Stdout contains the captured stdout output.
	Stdout string
	// Stderr contains the captured stderr output.
	Stderr string
	// ExitCode is the exit code from the process.
	ExitCode int
	// Args contains the arguments that were passed (from env var).
	Args []string
	// Err is any error from cmd.Run() (e.g., exit status error).
	Err error
}

// RunHelperCommand executes a helper process command and captures results.
func RunHelperCommand(t *testing.T, cmd *exec.Cmd) *HelperProcessResult {
	t.Helper()

	result := &HelperProcessResult{}

	// Capture stdout and stderr
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Run the command
	result.Err = cmd.Run()

	result.Stdout = stdout.String()
	result.Stderr = stderr.String()
	result.ExitCode = extractExitCode(cmd)
	result.Args = extractArgsFromEnv(cmd.Env)

	return result
}

// extractExitCode gets the exit code from a completed command.
func extractExitCode(cmd *exec.Cmd) int {
	if cmd.ProcessState == nil {
		return 0
	}
	return cmd.ProcessState.ExitCode()
}

// extractArgsFromEnv parses the original args from the command's env vars.
func extractArgsFromEnv(env []string) []string {
	for _, e := range env {
		if strings.HasPrefix(e, EnvHelperProcessArgs+"=") {
			argsJSON := strings.TrimPrefix(e, EnvHelperProcessArgs+"=")
			var args []string
			if err := json.Unmarshal([]byte(argsJSON), &args); err == nil {
				return args
			}
		}
	}
	return nil
}

// MustConfigureTestCommand is like ConfigureTestCommand but with simplified
// configuration for common cases. It panics on configuration errors.
func MustConfigureTestCommand(t *testing.T, testName string, exitCode int, stdout string, args ...string) *exec.Cmd {
	t.Helper()
	return ConfigureTestCommand(t, testName, HelperProcessConfig{
		ExitCode: exitCode,
		Stdout:   stdout,
	}, args...)
}
