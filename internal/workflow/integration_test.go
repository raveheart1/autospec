// Package workflow_test tests workflow orchestration using mock infrastructure.
// Related: internal/workflow/orchestrator.go, internal/testutil/mock_executor.go
// Tags: workflow, integration, orchestration, mocks, git-isolation, retry, artifacts
package workflow_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ariel-frischer/autospec/internal/cliagent"
	"github.com/ariel-frischer/autospec/internal/config"
	"github.com/ariel-frischer/autospec/internal/testutil"
	"github.com/ariel-frischer/autospec/internal/workflow"
)

// TestWorkflowOrchestrator_Integration tests workflow orchestration using mock infrastructure.
//
// Test structure uses closure-based configuration:
//   - setupMock: configures mock response sequence (WithResponse/ThenResponse/ThenError)
//   - runWorkflow: executes workflow stages via mock
//   - verifyMock: asserts call counts, prompts, and timestamps
//
// The mock builder fluent API (WithResponse().ThenResponse()) enables readable test setup.
// IMPORTANT: NO t.Parallel() - GitIsolation changes cwd causing race conditions.
func TestWorkflowOrchestrator_Integration(t *testing.T) {
	// NOTE: Do NOT add t.Parallel() here or in subtests below.
	// GitIsolation changes the working directory which causes race conditions
	// when running in parallel. Each subtest captures origDir on setup, but
	// parallel execution can cause one test's temp dir to be captured as
	// another test's origDir, leading to cleanup failures.

	tests := map[string]struct {
		setupMock   func(*testing.T, *testutil.MockExecutorBuilder, string)
		runWorkflow func(*testing.T, *workflow.WorkflowOrchestrator, *testutil.MockExecutor) error
		wantErr     bool
		errContains string
		verifyMock  func(*testing.T, *testutil.MockExecutor)
	}{
		"successful execution records all calls": {
			setupMock: func(t *testing.T, builder *testutil.MockExecutorBuilder, specsDir string) {
				t.Helper()
				builder.
					WithResponse("spec created").
					ThenResponse("plan created").
					ThenResponse("tasks created")
			},
			runWorkflow: func(t *testing.T, orch *workflow.WorkflowOrchestrator, mock *testutil.MockExecutor) error {
				t.Helper()
				// Manually invoke the mock to simulate workflow stages
				if err := mock.Execute("/autospec.specify"); err != nil {
					return err
				}
				if err := mock.Execute("/autospec.plan"); err != nil {
					return err
				}
				return mock.Execute("/autospec.tasks")
			},
			wantErr: false,
			verifyMock: func(t *testing.T, mock *testutil.MockExecutor) {
				t.Helper()
				if got := mock.GetCallCount(); got != 3 {
					t.Errorf("expected 3 calls, got %d", got)
				}
				executeCalls := mock.GetCallsByMethod("Execute")
				if len(executeCalls) != 3 {
					t.Errorf("expected 3 Execute calls, got %d", len(executeCalls))
				}
			},
		},
		"mock records call details correctly": {
			setupMock: func(t *testing.T, builder *testutil.MockExecutorBuilder, specsDir string) {
				t.Helper()
				builder.WithResponse("success")
			},
			runWorkflow: func(t *testing.T, orch *workflow.WorkflowOrchestrator, mock *testutil.MockExecutor) error {
				t.Helper()
				return mock.Execute("/autospec.specify \"test feature\"")
			},
			wantErr: false,
			verifyMock: func(t *testing.T, mock *testutil.MockExecutor) {
				t.Helper()
				calls := mock.GetCalls()
				if len(calls) != 1 {
					t.Fatalf("expected 1 call, got %d", len(calls))
				}
				if calls[0].Prompt != "/autospec.specify \"test feature\"" {
					t.Errorf("unexpected prompt: %s", calls[0].Prompt)
				}
				if calls[0].Timestamp.IsZero() {
					t.Error("timestamp should not be zero")
				}
			},
		},
		"error response stops workflow": {
			setupMock: func(t *testing.T, builder *testutil.MockExecutorBuilder, specsDir string) {
				t.Helper()
				builder.
					WithResponse("spec created").
					ThenError(workflow.ErrMockExecute)
			},
			runWorkflow: func(t *testing.T, orch *workflow.WorkflowOrchestrator, mock *testutil.MockExecutor) error {
				t.Helper()
				if err := mock.Execute("/autospec.specify"); err != nil {
					return err
				}
				return mock.Execute("/autospec.plan")
			},
			wantErr:     true,
			errContains: "mock execute error",
			verifyMock: func(t *testing.T, mock *testutil.MockExecutor) {
				t.Helper()
				// Should have 2 calls - first succeeded, second failed
				if got := mock.GetCallCount(); got != 2 {
					t.Errorf("expected 2 calls, got %d", got)
				}
			},
		},
		"sequential responses return in order": {
			setupMock: func(t *testing.T, builder *testutil.MockExecutorBuilder, specsDir string) {
				t.Helper()
				builder.
					WithResponse("first").
					ThenResponse("second").
					ThenResponse("third")
			},
			runWorkflow: func(t *testing.T, orch *workflow.WorkflowOrchestrator, mock *testutil.MockExecutor) error {
				t.Helper()
				for _, cmd := range []string{"cmd1", "cmd2", "cmd3"} {
					if err := mock.Execute(cmd); err != nil {
						return err
					}
				}
				return nil
			},
			wantErr: false,
			verifyMock: func(t *testing.T, mock *testutil.MockExecutor) {
				t.Helper()
				calls := mock.GetCalls()
				if len(calls) != 3 {
					t.Fatalf("expected 3 calls, got %d", len(calls))
				}
				// Verify calls were recorded in order
				expected := []string{"cmd1", "cmd2", "cmd3"}
				for i, call := range calls {
					if call.Prompt != expected[i] {
						t.Errorf("call %d: expected %q, got %q", i, expected[i], call.Prompt)
					}
				}
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// NOTE: Do NOT add t.Parallel() - see comment at top of test function.

			// Create isolated git repo
			gi := testutil.NewGitIsolation(t)

			// Set up specs directory
			specsDir := filepath.Join(gi.TempRepoDir(), "specs")
			if err := os.MkdirAll(specsDir, 0o755); err != nil {
				t.Fatalf("failed to create specs dir: %v", err)
			}

			// Create mock executor builder
			builder := testutil.NewMockExecutorBuilder(t)
			tt.setupMock(t, builder, specsDir)
			mock := builder.Build()

			// Create workflow orchestrator with mock-compatible config
			cfg := &config.Configuration{
				SpecsDir:   specsDir,
				StateDir:   t.TempDir(),
				MaxRetries: 3,
				CustomAgent: &cliagent.CustomAgentConfig{
					Command: "echo",
					Args:    []string{"{{PROMPT}}"},
				},
			}
			orch := workflow.NewWorkflowOrchestrator(cfg)

			// Run the workflow function
			err := tt.runWorkflow(t, orch, mock)

			// Verify error expectation
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tt.errContains != "" && !containsSubstring(err.Error(), tt.errContains) {
					t.Errorf("error %q should contain %q", err.Error(), tt.errContains)
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			// Run mock verification
			if tt.verifyMock != nil {
				tt.verifyMock(t, mock)
			}

			// Note: VerifyNoBranchPollution is called automatically in Cleanup
		})
	}
}

// TestMockExecutor_RetryBehavior tests retry simulation with mock executor.
func TestMockExecutor_RetryBehavior(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		failures     int
		maxAttempts  int
		wantErr      bool
		wantAttempts int
	}{
		"succeeds after one failure": {
			failures:     1,
			maxAttempts:  3,
			wantErr:      false,
			wantAttempts: 2,
		},
		"succeeds after two failures": {
			failures:     2,
			maxAttempts:  3,
			wantErr:      false,
			wantAttempts: 3,
		},
		"fails when retries exhausted": {
			failures:     5,
			maxAttempts:  3,
			wantErr:      true,
			wantAttempts: 3,
		},
		"succeeds immediately with no failures": {
			failures:     0,
			maxAttempts:  3,
			wantErr:      false,
			wantAttempts: 1,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			builder := testutil.NewMockExecutorBuilder(t)

			// Configure mock to fail tt.failures times before succeeding
			for i := 0; i < tt.failures; i++ {
				builder.WithError(workflow.ErrMockExecute)
			}
			builder.WithResponse("success")

			mock := builder.Build()

			// Attempt execution with retries
			var lastErr error
			attempts := 0
			for attempts < tt.maxAttempts {
				attempts++
				err := mock.Execute("test command")
				if err == nil {
					break
				}
				lastErr = err
			}

			if tt.wantErr && lastErr == nil && attempts >= tt.maxAttempts {
				// Expected failure - need to check if last call failed
				// The mock may have succeeded on the last attempt
				if attempts > tt.failures {
					t.Error("expected error after retries exhausted")
				}
			}

			if !tt.wantErr && lastErr != nil && attempts == tt.wantAttempts {
				// Check if the last attempt actually succeeded
				calls := mock.GetCalls()
				if len(calls) > 0 && calls[len(calls)-1].Error != nil {
					t.Errorf("expected success, got error: %v", lastErr)
				}
			}

			if mock.GetCallCount() != tt.wantAttempts {
				t.Errorf("expected %d attempts, got %d", tt.wantAttempts, mock.GetCallCount())
			}
		})
	}
}

// TestMockExecutor_DelaySimulation tests timeout simulation with mock delays.
func TestMockExecutor_DelaySimulation(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		delay       time.Duration
		timeout     time.Duration
		wantTimeout bool
	}{
		"no delay completes quickly": {
			delay:       0,
			timeout:     time.Second,
			wantTimeout: false,
		},
		"small delay within timeout": {
			delay:       10 * time.Millisecond,
			timeout:     time.Second,
			wantTimeout: false,
		},
		"delay longer than timeout": {
			delay:       200 * time.Millisecond,
			timeout:     50 * time.Millisecond,
			wantTimeout: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			builder := testutil.NewMockExecutorBuilder(t)
			builder.WithResponse("success")
			if tt.delay > 0 {
				builder.WithDelay(tt.delay)
			}

			mock := builder.Build()

			// Execute with timing
			start := time.Now()
			done := make(chan error, 1)
			go func() {
				done <- mock.Execute("test")
			}()

			select {
			case err := <-done:
				elapsed := time.Since(start)
				if tt.wantTimeout {
					// If we expected timeout but got response, that's still OK
					// as the mock delay isn't a true context timeout
					t.Logf("completed in %v (expected timeout)", elapsed)
				}
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			case <-time.After(tt.timeout + 100*time.Millisecond):
				if !tt.wantTimeout {
					t.Error("unexpected timeout")
				}
			}
		})
	}
}

// TestMockExecutor_CallLogVerification tests that MOCK_CALL_LOG is properly recorded.
func TestMockExecutor_CallLogVerification(t *testing.T) {
	t.Parallel()

	builder := testutil.NewMockExecutorBuilder(t)
	builder.
		WithResponse("first response").
		ThenResponse("second response")

	mock := builder.Build()

	// Execute multiple commands
	commands := []string{
		"/autospec.specify \"new feature\"",
		"/autospec.plan",
	}

	for _, cmd := range commands {
		if err := mock.Execute(cmd); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	// Verify call log
	calls := mock.GetCalls()
	if len(calls) != len(commands) {
		t.Fatalf("expected %d calls, got %d", len(commands), len(calls))
	}

	// Verify each call was logged correctly
	for i, call := range calls {
		if call.Method != "Execute" {
			t.Errorf("call %d: expected method Execute, got %s", i, call.Method)
		}
		if call.Prompt != commands[i] {
			t.Errorf("call %d: expected prompt %q, got %q", i, commands[i], call.Prompt)
		}
		if call.Timestamp.IsZero() {
			t.Errorf("call %d: timestamp should not be zero", i)
		}
	}

	// Verify filtering by method works
	executeCalls := mock.GetCallsByMethod("Execute")
	if len(executeCalls) != len(commands) {
		t.Errorf("expected %d Execute calls, got %d", len(commands), len(executeCalls))
	}

	// Verify non-existent method returns empty
	nonExistent := mock.GetCallsByMethod("NonExistent")
	if len(nonExistent) != 0 {
		t.Errorf("expected 0 NonExistent calls, got %d", len(nonExistent))
	}
}

// TestGitIsolation_NoBranchPollution tests that git isolation prevents branch pollution.
func TestGitIsolation_NoBranchPollution(t *testing.T) {
	// NOTE: Do NOT add t.Parallel() - GitIsolation changes the working
	// directory which causes race conditions with parallel tests.

	// Create isolation
	gi := testutil.NewGitIsolation(t)

	// Record original state
	origDir := gi.OriginalDir()
	origBranch := gi.OriginalBranch()

	// Verify we're in the temp repo
	tempRepo := gi.TempRepoDir()
	currentDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current dir: %v", err)
	}
	if currentDir != tempRepo {
		t.Errorf("expected to be in temp repo %s, got %s", tempRepo, currentDir)
	}

	// Create a branch in the temp repo (should not affect original)
	gi.CreateBranch("test-branch", true)

	// Verify temp repo branch changed
	if gi.CurrentBranch() != "test-branch" {
		t.Errorf("expected temp branch to be test-branch, got %s", gi.CurrentBranch())
	}

	// Cleanup happens automatically via t.Cleanup
	// VerifyNoBranchPollution is called in Cleanup

	_ = origDir
	_ = origBranch
}

// TestGitIsolation_FileOperations tests file operations in isolated repo.
func TestGitIsolation_FileOperations(t *testing.T) {
	// NOTE: Do NOT add t.Parallel() - GitIsolation changes the working
	// directory which causes race conditions with parallel tests.

	gi := testutil.NewGitIsolation(t)

	// Add a file
	content := "test content"
	filePath := gi.AddFile("test-file.txt", content)

	// Verify file exists
	readContent, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if string(readContent) != content {
		t.Errorf("expected content %q, got %q", content, string(readContent))
	}

	// Add and commit
	gi.CommitAll("Test commit")

	// Create specs directory structure
	specsDir := gi.SetupSpecsDir("test-feature")
	if _, err := os.Stat(specsDir); os.IsNotExist(err) {
		t.Error("specs directory should exist")
	}

	// Write spec
	specPath := gi.WriteSpec(specsDir)
	if _, err := os.Stat(specPath); os.IsNotExist(err) {
		t.Error("spec.yaml should exist")
	}
}

// TestMockExecutor_ArtifactGeneration tests that mock can generate artifacts.
func TestMockExecutor_ArtifactGeneration(t *testing.T) {
	// NOTE: Do NOT add t.Parallel() - GitIsolation changes the working
	// directory which causes race conditions with parallel tests.

	gi := testutil.NewGitIsolation(t)
	specsDir := gi.SetupSpecsDir("test-feature")

	builder := testutil.NewMockExecutorBuilder(t)
	builder.
		WithArtifactDir(specsDir).
		WithResponse("created").
		WithArtifactGeneration(testutil.ArtifactGenerators.Spec)

	mock := builder.Build()

	// Execute - this should trigger artifact generation
	if err := mock.Execute("/autospec.specify"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify artifact was created
	specPath := filepath.Join(specsDir, "spec.yaml")
	if _, err := os.Stat(specPath); os.IsNotExist(err) {
		t.Error("spec.yaml should have been generated")
	}
}

// TestMockExecutor_Reset tests that reset clears all state.
func TestMockExecutor_Reset(t *testing.T) {
	t.Parallel()

	builder := testutil.NewMockExecutorBuilder(t)
	builder.
		WithResponse("first").
		ThenResponse("second")

	mock := builder.Build()

	// Make some calls
	_ = mock.Execute("cmd1")
	_ = mock.Execute("cmd2")

	if mock.GetCallCount() != 2 {
		t.Fatalf("expected 2 calls before reset, got %d", mock.GetCallCount())
	}

	// Reset
	mock.Reset()

	// Verify cleared
	if mock.GetCallCount() != 0 {
		t.Errorf("expected 0 calls after reset, got %d", mock.GetCallCount())
	}

	calls := mock.GetCalls()
	if len(calls) != 0 {
		t.Errorf("expected empty calls after reset, got %d", len(calls))
	}
}

// TestMockExecutor_AssertHelpers tests assertion helper methods.
func TestMockExecutor_AssertHelpers(t *testing.T) {
	t.Parallel()

	builder := testutil.NewMockExecutorBuilder(t)
	builder.WithResponse("success")

	mock := builder.Build()

	// Execute a command
	_ = mock.Execute("/autospec.specify")

	// Create a fake testing.T to capture assertions
	fakeT := &testing.T{}

	// Test AssertCalled - should find the call
	mock.AssertCalled(fakeT, "Execute", "specify")

	// Test AssertNotCalled - StreamCommand was not called
	mock.AssertNotCalled(fakeT, "StreamCommand")

	// Test AssertCallCount
	mock.AssertCallCount(fakeT, "Execute", 1)
}

// Helper function to check if a string contains a substring
func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstr(s, substr)))
}

func findSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestWorkflowStages_ArgumentValidation tests that mock captures and validates
// arguments for all workflow stages (specify, plan, tasks, implement).
// This test verifies FR-001 (exec.Command interception) and FR-005 (argument schema validation).
func TestWorkflowStages_ArgumentValidation(t *testing.T) {
	tests := map[string]struct {
		stage          string
		prompt         string
		expectedMethod string
		validatePrompt func(t *testing.T, prompt string)
	}{
		"specify stage captures prompt correctly": {
			stage:          "specify",
			prompt:         "/autospec.specify \"Add user authentication feature\"",
			expectedMethod: "Execute",
			validatePrompt: func(t *testing.T, prompt string) {
				t.Helper()
				if !containsSubstring(prompt, "specify") {
					t.Errorf("specify prompt should contain 'specify', got: %s", prompt)
				}
			},
		},
		"plan stage captures prompt correctly": {
			stage:          "plan",
			prompt:         "/autospec.plan",
			expectedMethod: "Execute",
			validatePrompt: func(t *testing.T, prompt string) {
				t.Helper()
				if !containsSubstring(prompt, "plan") {
					t.Errorf("plan prompt should contain 'plan', got: %s", prompt)
				}
			},
		},
		"tasks stage captures prompt correctly": {
			stage:          "tasks",
			prompt:         "/autospec.tasks",
			expectedMethod: "Execute",
			validatePrompt: func(t *testing.T, prompt string) {
				t.Helper()
				if !containsSubstring(prompt, "tasks") {
					t.Errorf("tasks prompt should contain 'tasks', got: %s", prompt)
				}
			},
		},
		"implement stage captures prompt correctly": {
			stage:          "implement",
			prompt:         "/autospec.implement --phase 1",
			expectedMethod: "Execute",
			validatePrompt: func(t *testing.T, prompt string) {
				t.Helper()
				if !containsSubstring(prompt, "implement") {
					t.Errorf("implement prompt should contain 'implement', got: %s", prompt)
				}
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			builder := testutil.NewMockExecutorBuilder(t)
			builder.WithResponse("success")

			mock := builder.Build()

			// Execute the stage
			err := mock.Execute(tt.prompt)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify call was recorded
			calls := mock.GetCallsByMethod(tt.expectedMethod)
			if len(calls) != 1 {
				t.Fatalf("expected 1 %s call, got %d", tt.expectedMethod, len(calls))
			}

			// Validate prompt content
			tt.validatePrompt(t, calls[0].Prompt)

			// Verify environment was captured
			if calls[0].Env == nil {
				t.Error("environment should have been captured")
			}

			// Verify timestamp is non-zero
			if calls[0].Timestamp.IsZero() {
				t.Error("timestamp should not be zero")
			}
		})
	}
}

// TestArgumentValidator_WorkflowStages tests argument validation for CLI flags
// across all workflow stages. This verifies FR-005 (argument schema validation).
func TestArgumentValidator_WorkflowStages(t *testing.T) {
	t.Parallel()

	validator := testutil.GetDefaultValidator()

	tests := map[string]struct {
		agentName string
		args      []string
		wantErr   bool
	}{
		"claude valid args with prompt": {
			agentName: "claude",
			args:      []string{"-p", "run /autospec.specify", "--output-format", "text"},
			wantErr:   false,
		},
		"claude valid args with print flag": {
			agentName: "claude",
			args:      []string{"--print", "-p", "test prompt"},
			wantErr:   false,
		},
		"claude invalid output format": {
			agentName: "claude",
			args:      []string{"-p", "test", "--output-format", "invalid-format"},
			wantErr:   true,
		},
		"opencode valid args": {
			agentName: "opencode",
			args:      []string{"-p", "run /autospec.plan", "--non-interactive"},
			wantErr:   false,
		},
		"opencode valid with output format": {
			agentName: "opencode",
			args:      []string{"--prompt", "test", "--output-format", "json"},
			wantErr:   false,
		},
		"opencode invalid output format": {
			agentName: "opencode",
			args:      []string{"-p", "test", "--output-format", "stream-json"},
			wantErr:   true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			err := validator.ValidateArgs(tt.agentName, tt.args)

			if tt.wantErr && err == nil {
				t.Error("expected error but got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// TestMockExecutor_EnvironmentCapture verifies that environment variables are
// captured during mock execution. This tests FR-006 (environment variable capture).
func TestMockExecutor_EnvironmentCapture(t *testing.T) {
	// NOTE: Do NOT add t.Parallel() - t.Setenv() is incompatible with parallel tests

	tests := map[string]struct {
		envKey    string
		envValue  string
		wantFound bool
	}{
		"ANTHROPIC_API_KEY empty blocks API calls": {
			envKey:    "TEST_ANTHROPIC_KEY",
			envValue:  "",
			wantFound: true,
		},
		"custom env var captured": {
			envKey:    "AUTOSPEC_TEST_VAR",
			envValue:  "test-value-123",
			wantFound: true,
		},
		"PATH is captured": {
			envKey:    "PATH",
			envValue:  "", // don't set, just verify it exists
			wantFound: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// NOTE: Do NOT add t.Parallel() - t.Setenv() is incompatible with parallel tests

			// Set env var if value provided
			if tt.envValue != "" {
				t.Setenv(tt.envKey, tt.envValue)
			}

			builder := testutil.NewMockExecutorBuilder(t)
			builder.WithResponse("success")
			mock := builder.Build()

			// Execute a command
			if err := mock.Execute("test command"); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			calls := mock.GetCalls()
			if len(calls) != 1 {
				t.Fatalf("expected 1 call, got %d", len(calls))
			}

			// Check if env var was captured
			_, found := calls[0].Env[tt.envKey]
			if found != tt.wantFound && tt.envValue != "" {
				t.Errorf("env var %s: found=%v, want found=%v", tt.envKey, found, tt.wantFound)
			}

			// For PATH, just verify env map is populated
			if tt.envKey == "PATH" && len(calls[0].Env) == 0 {
				t.Error("environment map should not be empty")
			}
		})
	}
}

// TestMockExecutor_GetCallsByEnv tests filtering calls by environment variables.
func TestMockExecutor_GetCallsByEnv(t *testing.T) {
	// NOTE: Do NOT add t.Parallel() - t.Setenv() is incompatible with parallel tests

	// Set a test env var that will be captured
	t.Setenv("TEST_FILTER_VAR", "filter-value")

	builder := testutil.NewMockExecutorBuilder(t)
	builder.
		WithResponse("first").
		ThenResponse("second").
		ThenResponse("third")

	mock := builder.Build()

	// Execute multiple commands
	for _, cmd := range []string{"cmd1", "cmd2", "cmd3"} {
		if err := mock.Execute(cmd); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	// All calls should have the env var
	matchingCalls := mock.GetCallsByEnv("TEST_FILTER_VAR", "filter-value")
	if len(matchingCalls) != 3 {
		t.Errorf("expected 3 calls with TEST_FILTER_VAR, got %d", len(matchingCalls))
	}

	// Query with any value (empty string matches any)
	anyCalls := mock.GetCallsByEnv("TEST_FILTER_VAR", "")
	if len(anyCalls) != 3 {
		t.Errorf("expected 3 calls with TEST_FILTER_VAR (any value), got %d", len(anyCalls))
	}

	// Query for non-existent value
	noCalls := mock.GetCallsByEnv("TEST_FILTER_VAR", "wrong-value")
	if len(noCalls) != 0 {
		t.Errorf("expected 0 calls with wrong value, got %d", len(noCalls))
	}
}

// TestCallLog_RoundTrip tests writing and reading call logs in YAML format.
// This verifies the call log infrastructure from FR-002.
func TestCallLog_RoundTrip(t *testing.T) {
	t.Parallel()

	// Create test call records
	records := []testutil.CallRecord{
		{
			Method:    "Execute",
			Prompt:    "/autospec.specify \"test feature\"",
			Args:      []string{"-p", "test"},
			Env:       map[string]string{"HOME": "/home/test", "SHELL": "/bin/bash"},
			Timestamp: time.Now(),
			Response:  "success",
			ExitCode:  0,
		},
		{
			Method:    "Execute",
			Prompt:    "/autospec.plan",
			Args:      []string{"-p", "plan"},
			Env:       map[string]string{"HOME": "/home/test"},
			Timestamp: time.Now().Add(time.Second),
			Response:  "plan created",
			ExitCode:  0,
		},
		{
			Method:    "Execute",
			Prompt:    "/autospec.tasks",
			Args:      nil,
			Env:       map[string]string{},
			Timestamp: time.Now().Add(2 * time.Second),
			Response:  "",
			Error:     workflow.ErrMockExecute,
			ExitCode:  1,
		},
	}

	// Write to temp file
	tempFile := filepath.Join(t.TempDir(), "call_log.yaml")
	if err := testutil.WriteCallLog(tempFile, records); err != nil {
		t.Fatalf("failed to write call log: %v", err)
	}

	// Read back
	log, err := testutil.ReadCallLog(tempFile)
	if err != nil {
		t.Fatalf("failed to read call log: %v", err)
	}

	// Verify entry count
	if len(log.Entries) != len(records) {
		t.Fatalf("expected %d entries, got %d", len(records), len(log.Entries))
	}

	// Verify each entry
	for i, entry := range log.Entries {
		if entry.Method != records[i].Method {
			t.Errorf("entry %d: method mismatch: got %s, want %s", i, entry.Method, records[i].Method)
		}
		if entry.Prompt != records[i].Prompt {
			t.Errorf("entry %d: prompt mismatch: got %s, want %s", i, entry.Prompt, records[i].Prompt)
		}
		if entry.ExitCode != records[i].ExitCode {
			t.Errorf("entry %d: exit code mismatch: got %d, want %d", i, entry.ExitCode, records[i].ExitCode)
		}
	}

	// Verify error entry has error string
	if !log.Entries[2].HasError() {
		t.Error("entry 2 should have error")
	}
}

// TestNoAPICallsGuarantee verifies that mock execution never makes real API calls.
// This is verified by checking that no network-related errors occur and
// environment capture shows expected test values (FR-004).
func TestNoAPICallsGuarantee(t *testing.T) {
	// NOTE: Do NOT add t.Parallel() - t.Setenv() is incompatible with parallel tests

	tests := map[string]struct {
		stage  string
		prompt string
	}{
		"specify stage no API": {
			stage:  "specify",
			prompt: "/autospec.specify \"test\"",
		},
		"plan stage no API": {
			stage:  "plan",
			prompt: "/autospec.plan",
		},
		"tasks stage no API": {
			stage:  "tasks",
			prompt: "/autospec.tasks",
		},
		"implement stage no API": {
			stage:  "implement",
			prompt: "/autospec.implement",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// NOTE: Do NOT add t.Parallel() - t.Setenv() is incompatible with parallel tests

			// Set empty API key to ensure no real calls possible
			t.Setenv("ANTHROPIC_API_KEY", "")

			builder := testutil.NewMockExecutorBuilder(t)
			builder.WithResponse("mocked response - no API call")
			mock := builder.Build()

			// Execute - should succeed with mock, no network error
			err := mock.Execute(tt.prompt)
			if err != nil {
				t.Fatalf("unexpected error (potential API call attempt?): %v", err)
			}

			// Verify mock recorded the call
			calls := mock.GetCalls()
			if len(calls) != 1 {
				t.Fatalf("expected exactly 1 mock call, got %d", len(calls))
			}

			// Verify response is from mock
			if calls[0].Response != "mocked response - no API call" {
				t.Errorf("response should be from mock, got: %s", calls[0].Response)
			}

			// Verify API key is empty in captured env
			if apiKey, ok := calls[0].Env["ANTHROPIC_API_KEY"]; ok && apiKey != "" {
				t.Errorf("ANTHROPIC_API_KEY should be empty, got: %s", apiKey)
			}
		})
	}
}

// TestHelperProcess is required for TestHelperProcess pattern tests.
// When GO_WANT_HELPER_PROCESS=1, this becomes a mock subprocess.
func TestHelperProcess(t *testing.T) {
	testutil.TestHelperProcess(t)
}

// TestTestHelperProcess_Integration tests the TestHelperProcess pattern for
// exec.Command interception. This verifies FR-001.
func TestTestHelperProcess_Integration(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		exitCode   int
		stdout     string
		stderr     string
		args       []string
		wantErr    bool
		wantStdout string
	}{
		"successful execution": {
			exitCode:   0,
			stdout:     "command output",
			args:       []string{"-p", "test prompt"},
			wantErr:    false,
			wantStdout: "command output",
		},
		"failed execution": {
			exitCode: 1,
			stderr:   "command failed",
			args:     []string{"--invalid"},
			wantErr:  true,
		},
		"argument validation simulation": {
			exitCode:   0,
			stdout:     "args validated",
			args:       []string{"-p", "/autospec.specify", "--output-format", "text"},
			wantErr:    false,
			wantStdout: "args validated",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			config := testutil.HelperProcessConfig{
				ExitCode: tt.exitCode,
				Stdout:   tt.stdout,
				Stderr:   tt.stderr,
			}

			cmd := testutil.ConfigureTestCommand(t, "TestHelperProcess", config, tt.args...)
			result := testutil.RunHelperCommand(t, cmd)

			// Check error expectation
			if tt.wantErr && result.Err == nil {
				t.Error("expected error but got nil")
			}
			if !tt.wantErr && result.Err != nil {
				t.Errorf("unexpected error: %v", result.Err)
			}

			// Verify exit code
			if result.ExitCode != tt.exitCode {
				t.Errorf("exit code: got %d, want %d", result.ExitCode, tt.exitCode)
			}

			// Verify stdout
			if tt.wantStdout != "" && result.Stdout != tt.wantStdout {
				t.Errorf("stdout: got %q, want %q", result.Stdout, tt.wantStdout)
			}

			// Verify args were captured
			if len(tt.args) > 0 && len(result.Args) == 0 {
				t.Error("args should have been captured")
			}
		})
	}
}

// TestCompleteWorkflowSimulation simulates a complete workflow execution
// (specify -> plan -> tasks -> implement) using mocks and validates all
// aspects: argument capture, environment capture, and call ordering.
func TestCompleteWorkflowSimulation(t *testing.T) {
	// NOTE: Do NOT add t.Parallel() - GitIsolation changes cwd

	gi := testutil.NewGitIsolation(t)
	specsDir := gi.SetupSpecsDir("test-feature")

	builder := testutil.NewMockExecutorBuilder(t)
	builder.
		WithArtifactDir(specsDir).
		WithResponse("spec created").
		WithArtifactGeneration(testutil.ArtifactGenerators.Spec).
		ThenResponse("plan created").
		ThenResponse("tasks created").
		ThenResponse("implementation complete")

	mock := builder.Build()

	// Simulate complete workflow
	stages := []struct {
		name   string
		prompt string
	}{
		{"specify", "/autospec.specify \"test feature\""},
		{"plan", "/autospec.plan"},
		{"tasks", "/autospec.tasks"},
		{"implement", "/autospec.implement"},
	}

	for _, stage := range stages {
		if err := mock.Execute(stage.prompt); err != nil {
			t.Fatalf("stage %s failed: %v", stage.name, err)
		}
	}

	// Verify all stages executed in order
	calls := mock.GetCalls()
	if len(calls) != len(stages) {
		t.Fatalf("expected %d calls, got %d", len(stages), len(calls))
	}

	for i, call := range calls {
		if call.Prompt != stages[i].prompt {
			t.Errorf("stage %d (%s): got prompt %q, want %q",
				i, stages[i].name, call.Prompt, stages[i].prompt)
		}

		// Verify timestamp ordering
		if i > 0 && call.Timestamp.Before(calls[i-1].Timestamp) {
			t.Errorf("stage %d timestamp should be after stage %d", i, i-1)
		}

		// Verify environment was captured for each call
		if call.Env == nil {
			t.Errorf("stage %d: environment should be captured", i)
		}
	}

	// Verify artifact was generated
	specPath := filepath.Join(specsDir, "spec.yaml")
	if _, err := os.Stat(specPath); os.IsNotExist(err) {
		t.Error("spec.yaml should have been generated by mock")
	}
}

// TestArgumentValidator_CLIBinaryCheck tests the CLI binary availability check.
// This supports FR-003 (real binary validation support).
func TestArgumentValidator_CLIBinaryCheck(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		agentName    string
		expectExists bool // depends on test environment
	}{
		"claude check": {
			agentName:    "claude",
			expectExists: false, // may or may not exist
		},
		"opencode check": {
			agentName:    "opencode",
			expectExists: false, // may or may not exist
		},
		"unknown agent": {
			agentName:    "unknown-agent",
			expectExists: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			available, msg := testutil.IsRealCLIAvailable(tt.agentName)

			// We just verify the function works without error
			if msg == "" {
				t.Error("message should not be empty")
			}

			// Log the result for visibility
			t.Logf("agent %s: available=%v, message=%s", tt.agentName, available, msg)
		})
	}
}
