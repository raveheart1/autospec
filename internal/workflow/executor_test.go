package workflow

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/ariel-frischer/autospec/internal/retry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockClaudeExecutor implements a mock for testing
type mockClaudeExecutor struct {
	executeErr   error
	executeCalls []string
}

func (m *mockClaudeExecutor) Execute(prompt string) error {
	m.executeCalls = append(m.executeCalls, prompt)
	return m.executeErr
}

func (m *mockClaudeExecutor) FormatCommand(prompt string) string {
	return "claude " + prompt
}

func TestGetStageNumber(t *testing.T) {
	tests := map[string]struct {
		stage Stage
		want  int
	}{
		"constitution stage": {stage: StageConstitution, want: 1},
		"specify stage":      {stage: StageSpecify, want: 2},
		"clarify stage":      {stage: StageClarify, want: 3},
		"plan stage":         {stage: StagePlan, want: 4},
		"tasks stage":        {stage: StageTasks, want: 5},
		"checklist stage":    {stage: StageChecklist, want: 6},
		"analyze stage":      {stage: StageAnalyze, want: 7},
		"implement stage":    {stage: StageImplement, want: 8},
		"unknown stage":      {stage: Stage("unknown"), want: 0},
		"empty stage":        {stage: Stage(""), want: 0},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			executor := &Executor{}
			got := executor.getStageNumber(tc.stage)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestBuildStageInfo(t *testing.T) {
	tests := map[string]struct {
		stage       Stage
		retryCount  int
		maxRetries  int
		totalStages int
		wantName    string
		wantNumber  int
	}{
		"specify stage no retries": {
			stage:       StageSpecify,
			retryCount:  0,
			maxRetries:  3,
			totalStages: 4,
			wantName:    "specify",
			wantNumber:  2,
		},
		"plan stage with retries": {
			stage:       StagePlan,
			retryCount:  2,
			maxRetries:  3,
			totalStages: 4,
			wantName:    "plan",
			wantNumber:  4,
		},
		"implement stage max retries": {
			stage:       StageImplement,
			retryCount:  3,
			maxRetries:  3,
			totalStages: 8,
			wantName:    "implement",
			wantNumber:  8,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			executor := &Executor{
				MaxRetries:  tc.maxRetries,
				TotalStages: tc.totalStages,
			}

			info := executor.buildStageInfo(tc.stage, tc.retryCount)

			assert.Equal(t, tc.wantName, info.Name)
			assert.Equal(t, tc.wantNumber, info.Number)
			assert.Equal(t, tc.totalStages, info.TotalStages)
			assert.Equal(t, tc.retryCount, info.RetryCount)
			assert.Equal(t, tc.maxRetries, info.MaxRetries)
		})
	}
}

func TestExecuteStage_Success(t *testing.T) {
	stateDir := t.TempDir()
	specsDir := t.TempDir()

	// Create spec directory with required file
	specDir := filepath.Join(specsDir, "001-test")
	require.NoError(t, os.MkdirAll(specDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(specDir, "spec.md"), []byte("# Test Spec"), 0644))

	executor := &Executor{
		Claude: &ClaudeExecutor{
			ClaudeCmd:  "echo",
			ClaudeArgs: []string{},
		},
		StateDir:   stateDir,
		SpecsDir:   specsDir,
		MaxRetries: 3,
	}

	// Validation function that always succeeds
	validateFunc := func(dir string) error {
		return nil
	}

	result, err := executor.ExecuteStage("001-test", StageSpecify, "/test.command", validateFunc)

	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, StageSpecify, result.Stage)
	assert.Equal(t, 0, result.RetryCount)
	assert.False(t, result.Exhausted)
}

func TestExecuteStage_ValidationFailure(t *testing.T) {
	stateDir := t.TempDir()
	specsDir := t.TempDir()

	executor := &Executor{
		Claude: &ClaudeExecutor{
			ClaudeCmd:  "echo",
			ClaudeArgs: []string{},
		},
		StateDir:   stateDir,
		SpecsDir:   specsDir,
		MaxRetries: 3,
	}

	// Validation function that always fails
	validateFunc := func(dir string) error {
		return errors.New("validation failed: missing spec.md")
	}

	result, err := executor.ExecuteStage("001-test", StageSpecify, "/test.command", validateFunc)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "validation failed")
	assert.Contains(t, err.Error(), "retry exhausted")
	assert.False(t, result.Success)
	assert.Equal(t, 3, result.RetryCount) // After exhausting all retries (MaxRetries=3)
	assert.True(t, result.Exhausted)      // Retries exhausted
}

func TestExecuteStage_RetryExhausted(t *testing.T) {
	stateDir := t.TempDir()
	specsDir := t.TempDir()

	// Pre-set retry count to max so next failure returns exhausted error
	state := &retry.RetryState{
		SpecName:   "001-test",
		Phase:      "specify",
		Count:      3,
		MaxRetries: 3,
	}
	require.NoError(t, retry.SaveRetryState(stateDir, state))

	executor := &Executor{
		Claude: &ClaudeExecutor{
			ClaudeCmd:  "echo",
			ClaudeArgs: []string{},
		},
		StateDir:   stateDir,
		SpecsDir:   specsDir,
		MaxRetries: 3,
	}

	// Validation function that always fails
	validateFunc := func(dir string) error {
		return errors.New("validation failed")
	}

	result, err := executor.ExecuteStage("001-test", StageSpecify, "/test.command", validateFunc)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "exhausted")
	assert.False(t, result.Success)
	assert.True(t, result.Exhausted)
}

func TestExecuteStage_ResetsRetryOnSuccess(t *testing.T) {
	stateDir := t.TempDir()
	specsDir := t.TempDir()

	// Pre-set retry count
	state := &retry.RetryState{
		SpecName:   "001-test",
		Phase:      "specify",
		Count:      2,
		MaxRetries: 3,
	}
	require.NoError(t, retry.SaveRetryState(stateDir, state))

	executor := &Executor{
		Claude: &ClaudeExecutor{
			ClaudeCmd:  "echo",
			ClaudeArgs: []string{},
		},
		StateDir:   stateDir,
		SpecsDir:   specsDir,
		MaxRetries: 3,
	}

	// Validation function that succeeds
	validateFunc := func(dir string) error {
		return nil
	}

	result, err := executor.ExecuteStage("001-test", StageSpecify, "/test.command", validateFunc)

	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, 0, result.RetryCount)

	// Verify retry state was reset
	loaded, err := retry.LoadRetryState(stateDir, "001-test", "specify", 3)
	require.NoError(t, err)
	assert.Equal(t, 0, loaded.Count)
}

func TestExecuteWithRetry_Success(t *testing.T) {
	executor := &Executor{
		Claude: &ClaudeExecutor{
			ClaudeCmd:  "echo",
			ClaudeArgs: []string{"success"},
		},
	}

	err := executor.ExecuteWithRetry("/test.command", 3)
	assert.NoError(t, err)
}

func TestExecuteWithRetry_AllAttemptsFail(t *testing.T) {
	executor := &Executor{
		Claude: &ClaudeExecutor{
			ClaudeCmd:  "false", // Command that always fails
			ClaudeArgs: []string{},
		},
	}

	err := executor.ExecuteWithRetry("/test.command", 2)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "all 2 attempts failed")
}

func TestGetRetryState(t *testing.T) {
	stateDir := t.TempDir()

	// Save initial state
	state := &retry.RetryState{
		SpecName:   "001-test",
		Phase:      "plan",
		Count:      1,
		MaxRetries: 3,
	}
	require.NoError(t, retry.SaveRetryState(stateDir, state))

	executor := &Executor{
		StateDir:   stateDir,
		MaxRetries: 3,
	}

	loaded, err := executor.GetRetryState("001-test", StagePlan)

	require.NoError(t, err)
	assert.Equal(t, 1, loaded.Count)
	assert.Equal(t, "001-test", loaded.SpecName)
	assert.Equal(t, "plan", loaded.Phase)
}

func TestResetStage(t *testing.T) {
	stateDir := t.TempDir()

	// Save initial state with non-zero count
	state := &retry.RetryState{
		SpecName:   "001-test",
		Phase:      "tasks",
		Count:      2,
		MaxRetries: 3,
	}
	require.NoError(t, retry.SaveRetryState(stateDir, state))

	executor := &Executor{
		StateDir:   stateDir,
		MaxRetries: 3,
	}

	err := executor.ResetStage("001-test", StageTasks)
	require.NoError(t, err)

	// Verify reset
	loaded, err := retry.LoadRetryState(stateDir, "001-test", "tasks", 3)
	require.NoError(t, err)
	assert.Equal(t, 0, loaded.Count)
}

func TestValidateSpec(t *testing.T) {
	t.Run("spec exists", func(t *testing.T) {
		t.Parallel()
		specDir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(specDir, "spec.md"), []byte("# Spec"), 0644))

		executor := &Executor{}
		err := executor.ValidateSpec(specDir)
		assert.NoError(t, err)
	})

	t.Run("spec missing", func(t *testing.T) {
		t.Parallel()
		specDir := t.TempDir()

		executor := &Executor{}
		err := executor.ValidateSpec(specDir)
		assert.Error(t, err)
	})
}

func TestValidatePlan(t *testing.T) {
	t.Run("plan exists", func(t *testing.T) {
		t.Parallel()
		specDir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(specDir, "plan.md"), []byte("# Plan"), 0644))

		executor := &Executor{}
		err := executor.ValidatePlan(specDir)
		assert.NoError(t, err)
	})

	t.Run("plan missing", func(t *testing.T) {
		t.Parallel()
		specDir := t.TempDir()

		executor := &Executor{}
		err := executor.ValidatePlan(specDir)
		assert.Error(t, err)
	})
}

func TestValidateTasks(t *testing.T) {
	t.Run("tasks exists", func(t *testing.T) {
		t.Parallel()
		specDir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(specDir, "tasks.md"), []byte("# Tasks"), 0644))

		executor := &Executor{}
		err := executor.ValidateTasks(specDir)
		assert.NoError(t, err)
	})

	t.Run("tasks missing", func(t *testing.T) {
		t.Parallel()
		specDir := t.TempDir()

		executor := &Executor{}
		err := executor.ValidateTasks(specDir)
		assert.Error(t, err)
	})
}

func TestValidateTasksComplete(t *testing.T) {
	tests := map[string]struct {
		content string
		wantErr bool
	}{
		"all tasks complete": {
			content: `# Tasks
- [x] Task 1
- [x] Task 2
- [x] Task 3
`,
			wantErr: false,
		},
		"some tasks incomplete": {
			content: `# Tasks
- [x] Task 1
- [ ] Task 2
- [x] Task 3
`,
			wantErr: true,
		},
		"all tasks incomplete": {
			content: `# Tasks
- [ ] Task 1
- [ ] Task 2
`,
			wantErr: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			tasksPath := filepath.Join(dir, "tasks.md")
			require.NoError(t, os.WriteFile(tasksPath, []byte(tc.content), 0644))

			executor := &Executor{}
			err := executor.ValidateTasksComplete(tasksPath)

			if tc.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "tasks remain")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDebugLog(t *testing.T) {
	t.Run("debug disabled does not print", func(t *testing.T) {
		t.Parallel()
		executor := &Executor{Debug: false}
		// Should not panic and should not print
		executor.debugLog("test message %s", "arg")
	})

	t.Run("debug enabled prints", func(t *testing.T) {
		t.Parallel()
		executor := &Executor{Debug: true}
		// Should not panic - we can't easily capture stdout in this test
		// but we verify it doesn't crash
		executor.debugLog("test message %s", "arg")
	})
}

func TestFormatRetryContext(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		attemptNum       int
		maxRetries       int
		validationErrors []string
		want             string
	}{
		"no errors": {
			attemptNum:       2,
			maxRetries:       3,
			validationErrors: nil,
			want:             "RETRY 2/3",
		},
		"empty errors slice": {
			attemptNum:       1,
			maxRetries:       3,
			validationErrors: []string{},
			want:             "RETRY 1/3",
		},
		"single error": {
			attemptNum:       2,
			maxRetries:       3,
			validationErrors: []string{"missing required field: feature.branch"},
			want:             "RETRY 2/3\nSchema validation failed:\n- missing required field: feature.branch",
		},
		"multiple errors": {
			attemptNum:       1,
			maxRetries:       5,
			validationErrors: []string{"error one", "error two", "error three"},
			want:             "RETRY 1/5\nSchema validation failed:\n- error one\n- error two\n- error three",
		},
		"exactly 10 errors": {
			attemptNum: 2,
			maxRetries: 3,
			validationErrors: []string{
				"error 1", "error 2", "error 3", "error 4", "error 5",
				"error 6", "error 7", "error 8", "error 9", "error 10",
			},
			want: "RETRY 2/3\nSchema validation failed:\n- error 1\n- error 2\n- error 3\n- error 4\n- error 5\n- error 6\n- error 7\n- error 8\n- error 9\n- error 10",
		},
		"more than 10 errors truncated": {
			attemptNum: 3,
			maxRetries: 3,
			validationErrors: []string{
				"error 1", "error 2", "error 3", "error 4", "error 5",
				"error 6", "error 7", "error 8", "error 9", "error 10",
				"error 11", "error 12",
			},
			want: "RETRY 3/3\nSchema validation failed:\n- error 1\n- error 2\n- error 3\n- error 4\n- error 5\n- error 6\n- error 7\n- error 8\n- error 9\n- error 10\n...and 2 more errors",
		},
		"15 errors shows truncation": {
			attemptNum: 1,
			maxRetries: 5,
			validationErrors: []string{
				"e1", "e2", "e3", "e4", "e5", "e6", "e7", "e8", "e9", "e10",
				"e11", "e12", "e13", "e14", "e15",
			},
			want: "RETRY 1/5\nSchema validation failed:\n- e1\n- e2\n- e3\n- e4\n- e5\n- e6\n- e7\n- e8\n- e9\n- e10\n...and 5 more errors",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := FormatRetryContext(tc.attemptNum, tc.maxRetries, tc.validationErrors)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestBuildRetryCommand(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		command      string
		retryContext string
		originalArgs string
		want         string
	}{
		"no context no args": {
			command:      "/autospec.plan",
			retryContext: "",
			originalArgs: "",
			want:         "/autospec.plan",
		},
		"no context with args": {
			command:      "/autospec.specify",
			retryContext: "",
			originalArgs: "add user auth",
			want:         "/autospec.specify add user auth",
		},
		"context only no original args": {
			command:      "/autospec.plan",
			retryContext: "RETRY 2/3\nSchema validation failed:\n- missing field",
			originalArgs: "",
			want:         "/autospec.plan RETRY 2/3\nSchema validation failed:\n- missing field",
		},
		"context with original args": {
			command:      "/autospec.specify",
			retryContext: "RETRY 1/3\nSchema validation failed:\n- error",
			originalArgs: "feature description",
			want:         "/autospec.specify RETRY 1/3\nSchema validation failed:\n- error\n\nfeature description",
		},
		"multiline original args": {
			command:      "/autospec.implement",
			retryContext: "RETRY 2/3",
			originalArgs: "--phase 1\n--verbose",
			want:         "/autospec.implement RETRY 2/3\n\n--phase 1\n--verbose",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := BuildRetryCommand(tc.command, tc.retryContext, tc.originalArgs)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestExtractValidationErrors(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		err  error
		want []string
	}{
		"nil error": {
			err:  nil,
			want: nil,
		},
		"single line error no bullets": {
			err:  errors.New("file not found"),
			want: []string{"file not found"},
		},
		"standard validation error format": {
			err:  errors.New("schema validation failed for spec.yaml:\n- missing required field: feature.branch\n- invalid enum value"),
			want: []string{"missing required field: feature.branch", "invalid enum value"},
		},
		"multiple bullet errors": {
			err:  errors.New("validation failed:\n- error one\n- error two\n- error three"),
			want: []string{"error one", "error two", "error three"},
		},
		"mixed content with bullets": {
			err:  errors.New("header line\n- bullet one\nregular line\n- bullet two"),
			want: []string{"bullet one", "bullet two"},
		},
		"whitespace handling": {
			err:  errors.New("  - trimmed error  \n  - another error  "),
			want: []string{"trimmed error", "another error"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := ExtractValidationErrors(tc.err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestHandleValidationFailure_StoresValidationErrors(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()

	executor := &Executor{
		Claude: &ClaudeExecutor{
			ClaudeCmd:  "echo",
			ClaudeArgs: []string{},
		},
		StateDir:   stateDir,
		MaxRetries: 3,
	}

	result := &StageResult{
		Stage:   StageSpecify,
		Success: false,
	}

	retryState := &retry.RetryState{
		SpecName:   "test-spec",
		Phase:      "specify",
		Count:      0,
		MaxRetries: 3,
	}

	stageInfo := executor.buildStageInfo(StageSpecify, 0)

	// Create a validation error in the expected format
	validationErr := errors.New("schema validation failed for spec.yaml:\n- missing required field: feature.branch\n- invalid enum value: status")

	// Call handleValidationFailure
	_ = executor.handleValidationFailure(result, retryState, stageInfo, validationErr)

	// Verify validation errors were extracted and stored
	assert.NotNil(t, result.ValidationErrors)
	assert.Len(t, result.ValidationErrors, 2)
	assert.Contains(t, result.ValidationErrors, "missing required field: feature.branch")
	assert.Contains(t, result.ValidationErrors, "invalid enum value: status")
}

// TestExecuteStage_RetryLoopActuallyRetries verifies that the retry loop
// actually executes multiple times when validation fails.
// This test prevents regression of the bug where the loop was missing.
func TestExecuteStage_RetryLoopActuallyRetries(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()
	specsDir := t.TempDir()

	// Track how many times validation is called
	validationCallCount := 0

	executor := &Executor{
		Claude: &ClaudeExecutor{
			ClaudeCmd:  "echo",
			ClaudeArgs: []string{"success"},
		},
		StateDir:   stateDir,
		SpecsDir:   specsDir,
		MaxRetries: 2, // Allow 2 retries (3 total attempts)
	}

	// Validation function that always fails
	validateFunc := func(dir string) error {
		validationCallCount++
		return errors.New("schema validation failed for spec.yaml:\n- missing required field: requirements")
	}

	result, err := executor.ExecuteStage("001-test", StageSpecify, "/test.command", validateFunc)

	// Should have an error (exhausted)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exhausted")
	assert.True(t, result.Exhausted)

	// Verify the loop ran 3 times (1 initial + 2 retries)
	assert.Equal(t, 3, validationCallCount, "validation should be called 3 times (1 initial + 2 retries)")
}

// TestExecuteStage_ValidationSuccessOnRetry verifies that if validation
// succeeds on a retry attempt, the function returns success.
func TestExecuteStage_ValidationSuccessOnRetry(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()
	specsDir := t.TempDir()

	callCount := 0
	executor := &Executor{
		Claude: &ClaudeExecutor{
			ClaudeCmd:  "echo",
			ClaudeArgs: []string{"success"},
		},
		StateDir:   stateDir,
		SpecsDir:   specsDir,
		MaxRetries: 2,
	}

	// Validation function that fails first time, succeeds second time
	validateFunc := func(dir string) error {
		callCount++
		if callCount == 1 {
			return errors.New("schema validation failed for spec.yaml:\n- missing field")
		}
		return nil // Success on second attempt
	}

	result, err := executor.ExecuteStage("001-test", StageSpecify, "/test.command", validateFunc)

	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.False(t, result.Exhausted)
	assert.Equal(t, 2, callCount, "validation should be called twice (1 initial + 1 retry that succeeds)")
}

// TestExecuteStage_MaxRetriesZeroNoRetries verifies that with max_retries=0,
// no retries happen and the function returns error on first failure.
func TestExecuteStage_MaxRetriesZeroNoRetries(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()
	specsDir := t.TempDir()

	validationCallCount := 0
	executor := &Executor{
		Claude: &ClaudeExecutor{
			ClaudeCmd:  "echo",
			ClaudeArgs: []string{"success"},
		},
		StateDir:   stateDir,
		SpecsDir:   specsDir,
		MaxRetries: 0, // No retries allowed
	}

	// Validation function that always fails
	validateFunc := func(dir string) error {
		validationCallCount++
		return errors.New("validation failed")
	}

	result, err := executor.ExecuteStage("001-test", StageSpecify, "/test.command", validateFunc)

	require.Error(t, err)
	assert.True(t, result.Exhausted)
	assert.Equal(t, 1, validationCallCount, "validation should only be called once with max_retries=0")
}

// TestExecuteStage_RetryCountMatchesConfig verifies that the number of
// retries matches the max_retries configuration.
func TestExecuteStage_RetryCountMatchesConfig(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		maxRetries         int
		expectedValidCalls int
	}{
		"max_retries=0 means 1 attempt":  {maxRetries: 0, expectedValidCalls: 1},
		"max_retries=1 means 2 attempts": {maxRetries: 1, expectedValidCalls: 2},
		"max_retries=2 means 3 attempts": {maxRetries: 2, expectedValidCalls: 3},
		"max_retries=3 means 4 attempts": {maxRetries: 3, expectedValidCalls: 4},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			stateDir := t.TempDir()
			specsDir := t.TempDir()
			validationCallCount := 0

			executor := &Executor{
				Claude: &ClaudeExecutor{
					ClaudeCmd:  "echo",
					ClaudeArgs: []string{"success"},
				},
				StateDir:   stateDir,
				SpecsDir:   specsDir,
				MaxRetries: tc.maxRetries,
			}

			validateFunc := func(dir string) error {
				validationCallCount++
				return errors.New("always fails")
			}

			result, err := executor.ExecuteStage("001-test", StageSpecify, "/test.command", validateFunc)

			require.Error(t, err)
			assert.True(t, result.Exhausted)
			assert.Equal(t, tc.expectedValidCalls, validationCallCount,
				"with max_retries=%d, validation should be called %d times",
				tc.maxRetries, tc.expectedValidCalls)
		})
	}
}
