// Package testutil provides test utilities and helpers for autospec tests.
package testutil

import (
	"errors"
	"testing"
)

func TestCallRecord_EnvCapture(t *testing.T) {
	tests := map[string]struct {
		envKey   string
		envValue string
		method   string
		prompt   string
	}{
		"captures ANTHROPIC_API_KEY": {
			envKey:   "ANTHROPIC_API_KEY",
			envValue: "test-key-12345",
			method:   "Execute",
			prompt:   "test prompt",
		},
		"captures OPENCODE_API_KEY": {
			envKey:   "OPENCODE_API_KEY",
			envValue: "opencode-key-abc",
			method:   "Execute",
			prompt:   "another prompt",
		},
		"captures PATH variable": {
			envKey:   "PATH",
			envValue: "/usr/bin:/bin",
			method:   "ExecuteInteractive",
			prompt:   "interactive prompt",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Set the environment variable for this test
			t.Setenv(tt.envKey, tt.envValue)

			builder := NewMockExecutorBuilder(t)
			mock := builder.WithResponse("success").Build()

			// Execute based on method
			var err error
			switch tt.method {
			case "Execute":
				err = mock.Execute(tt.prompt)
			case "ExecuteInteractive":
				err = mock.ExecuteInteractive(tt.prompt)
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			calls := mock.GetCalls()
			if len(calls) != 1 {
				t.Fatalf("expected 1 call, got %d", len(calls))
			}

			call := calls[0]
			if call.Env == nil {
				t.Fatal("expected Env to be captured, got nil")
			}

			gotValue, ok := call.Env[tt.envKey]
			if !ok {
				t.Errorf("expected env key %q to be present", tt.envKey)
			}
			if gotValue != tt.envValue {
				t.Errorf("expected env[%q] = %q, got %q", tt.envKey, tt.envValue, gotValue)
			}
		})
	}
}

func TestGetCallsByEnv(t *testing.T) {
	tests := map[string]struct {
		setupEnv      map[string]string
		callCount     int
		filterKey     string
		filterValue   string
		expectedCount int
	}{
		"filter by key and value": {
			setupEnv:      map[string]string{"TEST_VAR": "test_value"},
			callCount:     3,
			filterKey:     "TEST_VAR",
			filterValue:   "test_value",
			expectedCount: 3,
		},
		"filter by key only (empty value)": {
			setupEnv:      map[string]string{"FILTER_KEY": "any_value"},
			callCount:     2,
			filterKey:     "FILTER_KEY",
			filterValue:   "",
			expectedCount: 2,
		},
		"no match when key missing": {
			setupEnv:      map[string]string{"OTHER_KEY": "value"},
			callCount:     2,
			filterKey:     "NONEXISTENT",
			filterValue:   "",
			expectedCount: 0,
		},
		"no match when value differs": {
			setupEnv:      map[string]string{"EXACT_KEY": "wrong_value"},
			callCount:     2,
			filterKey:     "EXACT_KEY",
			filterValue:   "expected_value",
			expectedCount: 0,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Set up environment
			for k, v := range tt.setupEnv {
				t.Setenv(k, v)
			}

			builder := NewMockExecutorBuilder(t)
			// Add enough responses for all calls
			for i := 0; i < tt.callCount; i++ {
				builder = builder.WithResponse("response")
			}
			mock := builder.Build()

			// Make calls
			for i := 0; i < tt.callCount; i++ {
				_ = mock.Execute("prompt")
			}

			// Filter by env
			filtered := mock.GetCallsByEnv(tt.filterKey, tt.filterValue)
			if len(filtered) != tt.expectedCount {
				t.Errorf("expected %d calls, got %d", tt.expectedCount, len(filtered))
			}
		})
	}
}

func TestMockExecutor_AllMethodsCaptureEnv(t *testing.T) {
	tests := map[string]struct {
		method string
		call   func(m *MockExecutor) error
	}{
		"Execute captures env": {
			method: "Execute",
			call:   func(m *MockExecutor) error { return m.Execute("prompt") },
		},
		"ExecuteInteractive captures env": {
			method: "ExecuteInteractive",
			call:   func(m *MockExecutor) error { return m.ExecuteInteractive("prompt") },
		},
		"ExecuteSpecKitCommand captures env": {
			method: "ExecuteSpecKitCommand",
			call:   func(m *MockExecutor) error { return m.ExecuteSpecKitCommand("command") },
		},
		"FormatCommand captures env": {
			method: "FormatCommand",
			call: func(m *MockExecutor) error {
				_ = m.FormatCommand("prompt")
				return nil
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			testEnvKey := "MOCK_TEST_ENV_" + name
			testEnvValue := "captured_value"
			t.Setenv(testEnvKey, testEnvValue)

			builder := NewMockExecutorBuilder(t)
			mock := builder.WithResponse("success").Build()

			err := tt.call(mock)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			calls := mock.GetCallsByMethod(tt.method)
			if len(calls) != 1 {
				t.Fatalf("expected 1 call to %s, got %d", tt.method, len(calls))
			}

			call := calls[0]
			if call.Env == nil {
				t.Errorf("expected Env to be captured for %s, got nil", tt.method)
				return
			}

			if val, ok := call.Env[testEnvKey]; !ok || val != testEnvValue {
				t.Errorf("expected env[%q] = %q, got %q (present: %v)",
					testEnvKey, testEnvValue, val, ok)
			}
		})
	}
}

func TestCallRecord_ExitCode(t *testing.T) {
	// Test that CallRecord has ExitCode field (structural test)
	record := CallRecord{
		Method:   "Execute",
		Prompt:   "test",
		ExitCode: 42,
	}

	if record.ExitCode != 42 {
		t.Errorf("expected ExitCode 42, got %d", record.ExitCode)
	}
}

func TestCallRecord_Args(t *testing.T) {
	// Test that CallRecord has Args field (structural test)
	args := []string{"arg1", "arg2", "--flag", "value"}
	record := CallRecord{
		Method: "Execute",
		Prompt: "test",
		Args:   args,
	}

	if len(record.Args) != 4 {
		t.Fatalf("expected 4 args, got %d", len(record.Args))
	}

	for i, expected := range args {
		if record.Args[i] != expected {
			t.Errorf("Args[%d]: expected %q, got %q", i, expected, record.Args[i])
		}
	}
}

func TestMockExecutor_BackwardCompatibility(t *testing.T) {
	// Verify existing API still works with new fields
	tests := map[string]struct {
		response    string
		expectError bool
	}{
		"success response": {
			response:    "success",
			expectError: false,
		},
		"error response": {
			response:    "",
			expectError: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			builder := NewMockExecutorBuilder(t)
			if tt.expectError {
				builder = builder.WithError(errors.New("test error"))
			} else {
				builder = builder.WithResponse(tt.response)
			}
			mock := builder.Build()

			err := mock.Execute("test prompt")

			if tt.expectError && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			// Verify call was recorded
			calls := mock.GetCalls()
			if len(calls) != 1 {
				t.Errorf("expected 1 call, got %d", len(calls))
			}

			// Verify existing fields still work
			call := calls[0]
			if call.Method != "Execute" {
				t.Errorf("expected Method 'Execute', got %q", call.Method)
			}
			if call.Prompt != "test prompt" {
				t.Errorf("expected Prompt 'test prompt', got %q", call.Prompt)
			}
			if call.Timestamp.IsZero() {
				t.Error("expected non-zero Timestamp")
			}
		})
	}
}

func TestCaptureEnvironment(t *testing.T) {
	tests := map[string]struct {
		envVars map[string]string
	}{
		"single variable": {
			envVars: map[string]string{
				"SINGLE_VAR": "value",
			},
		},
		"multiple variables": {
			envVars: map[string]string{
				"VAR_ONE": "value1",
				"VAR_TWO": "value2",
			},
		},
		"variable with equals in value": {
			envVars: map[string]string{
				"COMPLEX_VAR": "key=value",
			},
		},
		"empty value": {
			envVars: map[string]string{
				"EMPTY_VAR": "",
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Set up test environment variables
			for k, v := range tt.envVars {
				t.Setenv(k, v)
			}

			env := captureEnvironment()

			// Verify our test variables are captured
			for k, expected := range tt.envVars {
				if got, ok := env[k]; !ok {
					t.Errorf("expected key %q to be present", k)
				} else if got != expected {
					t.Errorf("env[%q]: expected %q, got %q", k, expected, got)
				}
			}
		})
	}
}

func TestFindFirstEquals(t *testing.T) {
	tests := map[string]struct {
		input    string
		expected int
	}{
		"simple key=value":  {input: "KEY=value", expected: 3},
		"no equals":         {input: "NOEQUALS", expected: -1},
		"empty string":      {input: "", expected: -1},
		"equals at start":   {input: "=value", expected: 0},
		"multiple equals":   {input: "KEY=val=ue", expected: 3},
		"equals only":       {input: "=", expected: 0},
		"value with equals": {input: "API_KEY=abc=123", expected: 7},
		"empty value":       {input: "EMPTY=", expected: 5},
		"long key":          {input: "VERY_LONG_ENVIRONMENT_VARIABLE_NAME=x", expected: 35},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := findFirstEquals(tt.input)
			if got != tt.expected {
				t.Errorf("findFirstEquals(%q) = %d, want %d", tt.input, got, tt.expected)
			}
		})
	}
}
