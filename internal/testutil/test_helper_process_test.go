package testutil

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

// TestHelperProcessFunc is the actual TestHelperProcess entry point for this test file.
// It must be a function named TestHelperProcess* to be discoverable by -test.run.
func TestHelperProcessFunc(t *testing.T) {
	TestHelperProcess(t)
}

func TestConfigureTestCommand(t *testing.T) {
	tests := map[string]struct {
		config   HelperProcessConfig
		args     []string
		wantEnvs map[string]bool // env vars that should be present
	}{
		"default config": {
			config: HelperProcessConfig{},
			args:   []string{"--help"},
			wantEnvs: map[string]bool{
				EnvWantHelperProcess: true,
				EnvHelperProcessArgs: true,
			},
		},
		"with exit code": {
			config: HelperProcessConfig{ExitCode: 1},
			args:   []string{"-p", "test prompt"},
			wantEnvs: map[string]bool{
				EnvWantHelperProcess: true,
			},
		},
		"with stdout": {
			config: HelperProcessConfig{Stdout: "mock output"},
			args:   []string{"arg1", "arg2"},
			wantEnvs: map[string]bool{
				EnvWantHelperProcess: true,
			},
		},
		"empty args": {
			config: HelperProcessConfig{},
			args:   []string{},
			wantEnvs: map[string]bool{
				EnvWantHelperProcess: true,
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			cmd := ConfigureTestCommand(t, "TestHelperProcessFunc", tt.config, tt.args...)

			// Verify it's an exec.Cmd
			if cmd == nil {
				t.Fatal("expected non-nil command")
			}

			// Check required env vars are present
			envMap := envSliceToMap(cmd.Env)
			for envName, required := range tt.wantEnvs {
				_, exists := envMap[envName]
				if required && !exists {
					t.Errorf("expected env var %s to be set", envName)
				}
			}

			// Verify GO_WANT_HELPER_PROCESS=1
			if val := envMap[EnvWantHelperProcess]; val != "1" {
				t.Errorf("expected %s=1, got %q", EnvWantHelperProcess, val)
			}
		})
	}
}

func TestRunHelperCommand_ExitCodes(t *testing.T) {
	tests := map[string]struct {
		exitCode int
		wantCode int
	}{
		"exit 0":   {exitCode: 0, wantCode: 0},
		"exit 1":   {exitCode: 1, wantCode: 1},
		"exit 2":   {exitCode: 2, wantCode: 2},
		"exit 127": {exitCode: 127, wantCode: 127},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			cmd := ConfigureTestCommand(t, "TestHelperProcessFunc", HelperProcessConfig{
				ExitCode: tt.exitCode,
			})

			result := RunHelperCommand(t, cmd)

			if result.ExitCode != tt.wantCode {
				t.Errorf("got exit code %d, want %d", result.ExitCode, tt.wantCode)
			}
		})
	}
}

func TestRunHelperCommand_Output(t *testing.T) {
	tests := map[string]struct {
		config     HelperProcessConfig
		wantStdout string
		wantStderr string
	}{
		"stdout only": {
			config:     HelperProcessConfig{Stdout: "hello stdout"},
			wantStdout: "hello stdout",
			wantStderr: "",
		},
		"stderr only": {
			config:     HelperProcessConfig{Stderr: "hello stderr"},
			wantStdout: "",
			wantStderr: "hello stderr",
		},
		"both streams": {
			config:     HelperProcessConfig{Stdout: "out", Stderr: "err"},
			wantStdout: "out",
			wantStderr: "err",
		},
		"multiline output": {
			config:     HelperProcessConfig{Stdout: "line1\nline2\nline3"},
			wantStdout: "line1\nline2\nline3",
			wantStderr: "",
		},
		"empty output": {
			config:     HelperProcessConfig{},
			wantStdout: "",
			wantStderr: "",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			cmd := ConfigureTestCommand(t, "TestHelperProcessFunc", tt.config)

			result := RunHelperCommand(t, cmd)

			if result.Stdout != tt.wantStdout {
				t.Errorf("stdout = %q, want %q", result.Stdout, tt.wantStdout)
			}
			if result.Stderr != tt.wantStderr {
				t.Errorf("stderr = %q, want %q", result.Stderr, tt.wantStderr)
			}
		})
	}
}

func TestRunHelperCommand_ArgsCapture(t *testing.T) {
	tests := map[string]struct {
		args     []string
		wantArgs []string
	}{
		"single arg": {
			args:     []string{"--help"},
			wantArgs: []string{"--help"},
		},
		"multiple args": {
			args:     []string{"-p", "test prompt", "--verbose"},
			wantArgs: []string{"-p", "test prompt", "--verbose"},
		},
		"args with spaces": {
			args:     []string{"-p", "prompt with spaces"},
			wantArgs: []string{"-p", "prompt with spaces"},
		},
		"args with special chars": {
			args:     []string{"--msg", "hello \"world\""},
			wantArgs: []string{"--msg", "hello \"world\""},
		},
		"empty args": {
			args:     []string{},
			wantArgs: []string{},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			cmd := ConfigureTestCommand(t, "TestHelperProcessFunc", HelperProcessConfig{}, tt.args...)

			result := RunHelperCommand(t, cmd)

			if len(result.Args) != len(tt.wantArgs) {
				t.Fatalf("got %d args, want %d", len(result.Args), len(tt.wantArgs))
			}

			for i, arg := range result.Args {
				if arg != tt.wantArgs[i] {
					t.Errorf("arg[%d] = %q, want %q", i, arg, tt.wantArgs[i])
				}
			}
		})
	}
}

func TestMustConfigureTestCommand(t *testing.T) {
	tests := map[string]struct {
		exitCode int
		stdout   string
		args     []string
	}{
		"success case": {
			exitCode: 0,
			stdout:   "success output",
			args:     []string{"--flag"},
		},
		"failure case": {
			exitCode: 1,
			stdout:   "error message",
			args:     []string{"-p", "prompt"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			cmd := MustConfigureTestCommand(t, "TestHelperProcessFunc", tt.exitCode, tt.stdout, tt.args...)

			if cmd == nil {
				t.Fatal("expected non-nil command")
			}

			result := RunHelperCommand(t, cmd)

			if result.ExitCode != tt.exitCode {
				t.Errorf("exit code = %d, want %d", result.ExitCode, tt.exitCode)
			}
			if result.Stdout != tt.stdout {
				t.Errorf("stdout = %q, want %q", result.Stdout, tt.stdout)
			}
		})
	}
}

func TestGetHelperProcessArgs(t *testing.T) {
	// Test when env var is not set
	t.Run("no env var", func(t *testing.T) {
		// Clear any existing env var
		originalVal := os.Getenv(EnvHelperProcessArgs)
		os.Unsetenv(EnvHelperProcessArgs)
		defer func() {
			if originalVal != "" {
				os.Setenv(EnvHelperProcessArgs, originalVal)
			}
		}()

		args, err := GetHelperProcessArgs()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if args != nil {
			t.Errorf("expected nil args, got %v", args)
		}
	})

	// Test with valid JSON
	t.Run("valid json", func(t *testing.T) {
		originalVal := os.Getenv(EnvHelperProcessArgs)
		os.Setenv(EnvHelperProcessArgs, `["arg1","arg2"]`)
		defer func() {
			if originalVal != "" {
				os.Setenv(EnvHelperProcessArgs, originalVal)
			} else {
				os.Unsetenv(EnvHelperProcessArgs)
			}
		}()

		args, err := GetHelperProcessArgs()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if len(args) != 2 || args[0] != "arg1" || args[1] != "arg2" {
			t.Errorf("unexpected args: %v", args)
		}
	})

	// Test with invalid JSON
	t.Run("invalid json", func(t *testing.T) {
		originalVal := os.Getenv(EnvHelperProcessArgs)
		os.Setenv(EnvHelperProcessArgs, "not valid json")
		defer func() {
			if originalVal != "" {
				os.Setenv(EnvHelperProcessArgs, originalVal)
			} else {
				os.Unsetenv(EnvHelperProcessArgs)
			}
		}()

		_, err := GetHelperProcessArgs()
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})
}

func TestHelperProcessConfig_JSON(t *testing.T) {
	tests := map[string]struct {
		config HelperProcessConfig
	}{
		"empty config": {
			config: HelperProcessConfig{},
		},
		"with all fields": {
			config: HelperProcessConfig{
				ExitCode: 42,
				Stdout:   "test stdout",
				Stderr:   "test stderr",
			},
		},
		"special characters": {
			config: HelperProcessConfig{
				Stdout: "line1\nline2\ttab",
				Stderr: "quote \"test\"",
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			cmd := ConfigureTestCommand(t, "TestHelperProcessFunc", tt.config)

			// Run and verify the output matches config
			result := RunHelperCommand(t, cmd)

			if result.Stdout != tt.config.Stdout {
				t.Errorf("stdout = %q, want %q", result.Stdout, tt.config.Stdout)
			}
			if result.Stderr != tt.config.Stderr {
				t.Errorf("stderr = %q, want %q", result.Stderr, tt.config.Stderr)
			}
			if result.ExitCode != tt.config.ExitCode {
				t.Errorf("exit code = %d, want %d", result.ExitCode, tt.config.ExitCode)
			}
		})
	}
}

// envSliceToMap converts an environment slice to a map for easier lookup.
func envSliceToMap(env []string) map[string]string {
	result := make(map[string]string)
	for _, e := range env {
		if idx := strings.Index(e, "="); idx != -1 {
			result[e[:idx]] = e[idx+1:]
		}
	}
	return result
}

// TestExecCommandIntegration demonstrates how the helper process pattern
// integrates with exec.Command for intercepting subprocess calls.
func TestExecCommandIntegration(t *testing.T) {
	t.Run("intercept claude-like command", func(t *testing.T) {
		// Simulate what autospec does when calling claude CLI
		args := []string{"-p", "test prompt", "--verbose", "--output-format", "text"}
		expectedOutput := "Mock response from helper process"

		cmd := ConfigureTestCommand(t, "TestHelperProcessFunc", HelperProcessConfig{
			ExitCode: 0,
			Stdout:   expectedOutput,
		}, args...)

		result := RunHelperCommand(t, cmd)

		// Verify we can intercept and get expected behavior
		if result.ExitCode != 0 {
			t.Errorf("expected exit code 0, got %d", result.ExitCode)
		}
		if result.Stdout != expectedOutput {
			t.Errorf("stdout = %q, want %q", result.Stdout, expectedOutput)
		}
		if len(result.Args) != len(args) {
			t.Errorf("captured args length = %d, want %d", len(result.Args), len(args))
		}
	})

	t.Run("simulate validation error", func(t *testing.T) {
		// Simulate CLI validation failure
		args := []string{"--invalid-flag"}
		errorOutput := "Error: unknown flag --invalid-flag"

		cmd := ConfigureTestCommand(t, "TestHelperProcessFunc", HelperProcessConfig{
			ExitCode: 1,
			Stderr:   errorOutput,
		}, args...)

		result := RunHelperCommand(t, cmd)

		if result.ExitCode != 1 {
			t.Errorf("expected exit code 1, got %d", result.ExitCode)
		}
		if result.Stderr != errorOutput {
			t.Errorf("stderr = %q, want %q", result.Stderr, errorOutput)
		}
	})
}

// BenchmarkConfigureTestCommand benchmarks command configuration.
func BenchmarkConfigureTestCommand(b *testing.B) {
	// Create a minimal testing.T for the benchmark
	t := &testing.T{}

	config := HelperProcessConfig{
		ExitCode: 0,
		Stdout:   "benchmark output",
	}
	args := []string{"-p", "benchmark prompt"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ConfigureTestCommand(t, "TestHelperProcessFunc", config, args...)
	}
}

// Example demonstrates basic usage of TestHelperProcess pattern.
func Example() {
	// In your test file, first define a helper process entry point:
	//
	//   func TestHelperProcess(t *testing.T) {
	//       testutil.TestHelperProcess(t)
	//   }
	//
	// Then use ConfigureTestCommand to create intercepted commands:
	//
	//   cmd := testutil.ConfigureTestCommand(t, "TestHelperProcess",
	//       testutil.HelperProcessConfig{
	//           ExitCode: 0,
	//           Stdout:   "mock response",
	//       },
	//       "-p", "my prompt",
	//   )
	//
	//   result := testutil.RunHelperCommand(t, cmd)
	//   // result.Stdout == "mock response"
	//   // result.Args == ["-p", "my prompt"]
}

// verifyCommandSetup is a helper that validates command configuration.
func verifyCommandSetup(t *testing.T, cmd *exec.Cmd) {
	t.Helper()

	if cmd == nil {
		t.Fatal("command is nil")
	}

	envMap := envSliceToMap(cmd.Env)
	if envMap[EnvWantHelperProcess] != "1" {
		t.Errorf("helper process env var not set correctly")
	}
}
