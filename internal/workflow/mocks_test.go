package workflow

import (
	"errors"
	"fmt"
	"io"
)

// MockClaudeExecutor is a mock implementation of ClaudeExecutor for testing.
// It records method calls and allows configuring return values and errors.
type MockClaudeExecutor struct {
	// Configuration
	ExecuteError     error
	StreamError      error
	FormatCmdFunc    func(string) string
	ExecuteFunc      func(string) error
	StreamFunc       func(string, io.Writer, io.Writer) error
	ExecuteDelay     int // Number of times to fail before succeeding
	executeCallCount int

	// Call tracking
	ExecuteCalls    []string
	StreamCalls     []StreamCall
	FormatCmdCalls  []string
	SpecKitCmdCalls []string
}

// StreamCall records a call to StreamCommand
type StreamCall struct {
	Prompt string
	Stdout io.Writer
	Stderr io.Writer
}

// NewMockClaudeExecutor creates a new mock executor with default behavior
func NewMockClaudeExecutor() *MockClaudeExecutor {
	return &MockClaudeExecutor{
		ExecuteCalls:    make([]string, 0),
		StreamCalls:     make([]StreamCall, 0),
		FormatCmdCalls:  make([]string, 0),
		SpecKitCmdCalls: make([]string, 0),
	}
}

// WithExecuteError configures the mock to return an error on Execute
func (m *MockClaudeExecutor) WithExecuteError(err error) *MockClaudeExecutor {
	m.ExecuteError = err
	return m
}

// WithStreamError configures the mock to return an error on StreamCommand
func (m *MockClaudeExecutor) WithStreamError(err error) *MockClaudeExecutor {
	m.StreamError = err
	return m
}

// WithExecuteFunc configures a custom execute function
func (m *MockClaudeExecutor) WithExecuteFunc(fn func(string) error) *MockClaudeExecutor {
	m.ExecuteFunc = fn
	return m
}

// WithExecuteDelay configures the mock to fail N times before succeeding
func (m *MockClaudeExecutor) WithExecuteDelay(count int) *MockClaudeExecutor {
	m.ExecuteDelay = count
	return m
}

// Execute records the call and returns configured error
func (m *MockClaudeExecutor) Execute(prompt string) error {
	m.ExecuteCalls = append(m.ExecuteCalls, prompt)
	m.executeCallCount++

	if m.ExecuteFunc != nil {
		return m.ExecuteFunc(prompt)
	}

	if m.ExecuteDelay > 0 && m.executeCallCount <= m.ExecuteDelay {
		return fmt.Errorf("mock execute failure %d/%d", m.executeCallCount, m.ExecuteDelay)
	}

	return m.ExecuteError
}

// FormatCommand records the call and returns formatted command
func (m *MockClaudeExecutor) FormatCommand(prompt string) string {
	m.FormatCmdCalls = append(m.FormatCmdCalls, prompt)

	if m.FormatCmdFunc != nil {
		return m.FormatCmdFunc(prompt)
	}

	return "claude " + prompt
}

// ExecuteSpecKitCommand records the call and delegates to Execute
func (m *MockClaudeExecutor) ExecuteSpecKitCommand(command string) error {
	m.SpecKitCmdCalls = append(m.SpecKitCmdCalls, command)
	return m.Execute(command)
}

// StreamCommand records the call and returns configured error
func (m *MockClaudeExecutor) StreamCommand(prompt string, stdout, stderr io.Writer) error {
	m.StreamCalls = append(m.StreamCalls, StreamCall{
		Prompt: prompt,
		Stdout: stdout,
		Stderr: stderr,
	})

	if m.StreamFunc != nil {
		return m.StreamFunc(prompt, stdout, stderr)
	}

	return m.StreamError
}

// Reset clears all recorded calls
func (m *MockClaudeExecutor) Reset() {
	m.ExecuteCalls = make([]string, 0)
	m.StreamCalls = make([]StreamCall, 0)
	m.FormatCmdCalls = make([]string, 0)
	m.SpecKitCmdCalls = make([]string, 0)
	m.executeCallCount = 0
}

// AssertExecuteCalled checks if Execute was called with the given prompt
func (m *MockClaudeExecutor) AssertExecuteCalled(prompt string) bool {
	for _, call := range m.ExecuteCalls {
		if call == prompt {
			return true
		}
	}
	return false
}

// ExecuteCallCount returns the number of times Execute was called
func (m *MockClaudeExecutor) ExecuteCallCount() int {
	return len(m.ExecuteCalls)
}

// Common test errors
var (
	ErrMockExecute        = errors.New("mock execute error")
	ErrMockValidation     = errors.New("mock validation error")
	ErrMockTimeout        = errors.New("mock timeout error")
	ErrMockRetryExhausted = errors.New("mock retry exhausted")
)
