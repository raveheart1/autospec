package testutil

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWriteAndReadCallLog(t *testing.T) {
	tests := map[string]struct {
		records      []CallRecord
		wantEntries  int
		wantMethod   string
		wantPrompt   string
		wantArgs     []string
		wantEnv      map[string]string
		wantError    string
		wantExitCode int
	}{
		"single record with all fields": {
			records: []CallRecord{
				{
					Method:    "Execute",
					Prompt:    "test prompt",
					Args:      []string{"-p", "test", "--verbose"},
					Env:       map[string]string{"ANTHROPIC_API_KEY": "test-key"},
					Timestamp: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
					Response:  "mock response",
					Error:     nil,
					ExitCode:  0,
				},
			},
			wantEntries:  1,
			wantMethod:   "Execute",
			wantPrompt:   "test prompt",
			wantArgs:     []string{"-p", "test", "--verbose"},
			wantEnv:      map[string]string{"ANTHROPIC_API_KEY": "test-key"},
			wantError:    "",
			wantExitCode: 0,
		},
		"record with error": {
			records: []CallRecord{
				{
					Method:    "ExecuteInteractive",
					Prompt:    "failing prompt",
					Args:      []string{"--fail"},
					Env:       map[string]string{},
					Timestamp: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
					Response:  "",
					Error:     errors.New("execution failed"),
					ExitCode:  1,
				},
			},
			wantEntries:  1,
			wantMethod:   "ExecuteInteractive",
			wantPrompt:   "failing prompt",
			wantArgs:     []string{"--fail"},
			wantEnv:      map[string]string{},
			wantError:    "execution failed",
			wantExitCode: 1,
		},
		"empty records": {
			records:     []CallRecord{},
			wantEntries: 0,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			tmpDir := t.TempDir()
			logPath := filepath.Join(tmpDir, "call_log.yaml")

			// Write the call log
			if err := WriteCallLog(logPath, tt.records); err != nil {
				t.Fatalf("WriteCallLog failed: %v", err)
			}

			// Read it back
			log, err := ReadCallLog(logPath)
			if err != nil {
				t.Fatalf("ReadCallLog failed: %v", err)
			}

			if len(log.Entries) != tt.wantEntries {
				t.Errorf("got %d entries, want %d", len(log.Entries), tt.wantEntries)
			}

			if tt.wantEntries == 0 {
				return
			}

			entry := log.Entries[0]
			if entry.Method != tt.wantMethod {
				t.Errorf("Method = %q, want %q", entry.Method, tt.wantMethod)
			}
			if entry.Prompt != tt.wantPrompt {
				t.Errorf("Prompt = %q, want %q", entry.Prompt, tt.wantPrompt)
			}
			if !equalStringSlice(entry.Args, tt.wantArgs) {
				t.Errorf("Args = %v, want %v", entry.Args, tt.wantArgs)
			}
			if !equalStringMap(entry.Env, tt.wantEnv) {
				t.Errorf("Env = %v, want %v", entry.Env, tt.wantEnv)
			}
			if entry.Error != tt.wantError {
				t.Errorf("Error = %q, want %q", entry.Error, tt.wantError)
			}
			if entry.ExitCode != tt.wantExitCode {
				t.Errorf("ExitCode = %d, want %d", entry.ExitCode, tt.wantExitCode)
			}
		})
	}
}

func TestCallLogSpecialCharacters(t *testing.T) {
	tests := map[string]struct {
		prompt   string
		args     []string
		response string
	}{
		"multiline prompt": {
			prompt:   "line one\nline two\nline three",
			args:     []string{"-p", "multi\nline"},
			response: "response with\nnewlines",
		},
		"unicode characters": {
			prompt:   "test with unicode: \u2603 \u2764 \u2728",
			args:     []string{"--emoji", "\u2603"},
			response: "emoji response: \u2764",
		},
		"quotes and escapes": {
			prompt:   `prompt with "quotes" and 'apostrophes' and \backslash`,
			args:     []string{`--arg="value"`, `--other='test'`},
			response: `response with "special" chars`,
		},
		"tabs and special whitespace": {
			prompt:   "prompt\twith\ttabs",
			args:     []string{"--tab=\t"},
			response: "response\twith\ttabs",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			tmpDir := t.TempDir()
			logPath := filepath.Join(tmpDir, "call_log.yaml")

			records := []CallRecord{
				{
					Method:    "Execute",
					Prompt:    tt.prompt,
					Args:      tt.args,
					Env:       map[string]string{},
					Timestamp: time.Now(),
					Response:  tt.response,
				},
			}

			if err := WriteCallLog(logPath, records); err != nil {
				t.Fatalf("WriteCallLog failed: %v", err)
			}

			log, err := ReadCallLog(logPath)
			if err != nil {
				t.Fatalf("ReadCallLog failed: %v", err)
			}

			entry := log.Entries[0]
			if entry.Prompt != tt.prompt {
				t.Errorf("Prompt roundtrip failed:\ngot:  %q\nwant: %q", entry.Prompt, tt.prompt)
			}
			if !equalStringSlice(entry.Args, tt.args) {
				t.Errorf("Args roundtrip failed:\ngot:  %v\nwant: %v", entry.Args, tt.args)
			}
			if entry.Response != tt.response {
				t.Errorf("Response roundtrip failed:\ngot:  %q\nwant: %q", entry.Response, tt.response)
			}
		})
	}
}

func TestCallLogToCallRecords(t *testing.T) {
	tests := map[string]struct {
		entries         []CallLogEntry
		wantRecordCount int
		wantErr         bool
	}{
		"valid entries": {
			entries: []CallLogEntry{
				{
					Method:    "Execute",
					Prompt:    "test",
					Timestamp: "2024-01-15T10:30:00Z",
					ExitCode:  0,
				},
				{
					Method:    "StreamCommand",
					Prompt:    "stream test",
					Timestamp: "2024-01-15T10:31:00Z",
					ExitCode:  0,
				},
			},
			wantRecordCount: 2,
			wantErr:         false,
		},
		"invalid timestamp": {
			entries: []CallLogEntry{
				{
					Method:    "Execute",
					Timestamp: "not-a-timestamp",
				},
			},
			wantRecordCount: 0,
			wantErr:         true,
		},
		"empty entries": {
			entries:         []CallLogEntry{},
			wantRecordCount: 0,
			wantErr:         false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			log := &CallLog{Entries: tt.entries}
			records, err := log.ToCallRecords()

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(records) != tt.wantRecordCount {
				t.Errorf("got %d records, want %d", len(records), tt.wantRecordCount)
			}
		})
	}
}

func TestCallLogEntryHasError(t *testing.T) {
	tests := map[string]struct {
		entry    CallLogEntry
		wantBool bool
	}{
		"with error": {
			entry:    CallLogEntry{Error: "something went wrong"},
			wantBool: true,
		},
		"without error": {
			entry:    CallLogEntry{Error: ""},
			wantBool: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if got := tt.entry.HasError(); got != tt.wantBool {
				t.Errorf("HasError() = %v, want %v", got, tt.wantBool)
			}
		})
	}
}

func TestReadCallLogFileNotFound(t *testing.T) {
	_, err := ReadCallLog("/nonexistent/path/call_log.yaml")
	if err == nil {
		t.Error("expected error for non-existent file, got nil")
	}
}

func TestWriteCallLogInvalidPath(t *testing.T) {
	err := WriteCallLog("/nonexistent/directory/call_log.yaml", []CallRecord{})
	if err == nil {
		t.Error("expected error for invalid path, got nil")
	}
}

func TestCallLogYAMLFormat(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "call_log.yaml")

	records := []CallRecord{
		{
			Method:    "Execute",
			Prompt:    "test prompt",
			Args:      []string{"-p", "test"},
			Env:       map[string]string{"KEY": "value"},
			Timestamp: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
			Response:  "response",
			ExitCode:  0,
		},
	}

	if err := WriteCallLog(logPath, records); err != nil {
		t.Fatalf("WriteCallLog failed: %v", err)
	}

	// Read raw YAML and verify structure
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("reading file failed: %v", err)
	}

	content := string(data)
	// Verify YAML has expected top-level structure
	if !containsSubstr(content, "entries:") {
		t.Error("YAML should contain 'entries:' key")
	}
	if !containsSubstr(content, "method: Execute") {
		t.Error("YAML should contain method field")
	}
	if !containsSubstr(content, "prompt: test prompt") {
		t.Error("YAML should contain prompt field")
	}
}

// Helper functions

func equalStringSlice(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func equalStringMap(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if bv, ok := b[k]; !ok || bv != v {
			return false
		}
	}
	return true
}

func containsSubstr(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstrHelper(s, substr))
}

func containsSubstrHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
