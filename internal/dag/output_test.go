package dag

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPrefixedWriter(t *testing.T) {
	tests := map[string]struct {
		specID string
		writes []string
		want   string
	}{
		"single line with newline": {
			specID: "spec-a",
			writes: []string{"hello world\n"},
			want:   "[spec-a] hello world\n",
		},
		"single line without newline": {
			specID: "spec-a",
			writes: []string{"hello world"},
			want:   "[spec-a] hello world",
		},
		"multiple lines in one write": {
			specID: "spec-b",
			writes: []string{"line1\nline2\nline3\n"},
			want:   "[spec-b] line1\n[spec-b] line2\n[spec-b] line3\n",
		},
		"multiple writes continuing line": {
			specID: "spec-c",
			writes: []string{"start", " middle", " end\n"},
			want:   "[spec-c] start middle end\n",
		},
		"multiple writes with newlines": {
			specID: "spec-d",
			writes: []string{"line1\n", "line2\n"},
			want:   "[spec-d] line1\n[spec-d] line2\n",
		},
		"empty write": {
			specID: "spec-e",
			writes: []string{""},
			want:   "",
		},
		"only newline": {
			specID: "spec-f",
			writes: []string{"\n"},
			want:   "[spec-f] \n",
		},
		"multiple empty lines": {
			specID: "spec-g",
			writes: []string{"\n\n\n"},
			want:   "[spec-g] \n[spec-g] \n[spec-g] \n",
		},
		"partial lines then complete": {
			specID: "test",
			writes: []string{"hello ", "world", "!\n", "next line\n"},
			want:   "[test] hello world!\n[test] next line\n",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var buf bytes.Buffer
			pw := NewPrefixedWriter(&buf, tc.specID)

			for _, w := range tc.writes {
				n, err := pw.Write([]byte(w))
				if err != nil {
					t.Errorf("Write() error: %v", err)
				}
				if n != len(w) {
					t.Errorf("Write() returned %d, want %d", n, len(w))
				}
			}

			got := buf.String()
			if got != tc.want {
				t.Errorf("output mismatch:\ngot:  %q\nwant: %q", got, tc.want)
			}
		})
	}
}

func TestPrefixedWriterFlush(t *testing.T) {
	tests := map[string]struct {
		specID     string
		writes     []string
		wantBefore string
		wantAfter  string
	}{
		"flush adds newline if needed": {
			specID:     "spec-a",
			writes:     []string{"incomplete line"},
			wantBefore: "[spec-a] incomplete line",
			wantAfter:  "[spec-a] incomplete line\n",
		},
		"flush does nothing if line complete": {
			specID:     "spec-a",
			writes:     []string{"complete line\n"},
			wantBefore: "[spec-a] complete line\n",
			wantAfter:  "[spec-a] complete line\n",
		},
		"flush on empty is no-op": {
			specID:     "spec-a",
			writes:     []string{},
			wantBefore: "",
			wantAfter:  "",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var buf bytes.Buffer
			pw := NewPrefixedWriter(&buf, tc.specID)

			for _, w := range tc.writes {
				if _, err := pw.Write([]byte(w)); err != nil {
					t.Errorf("Write() error: %v", err)
				}
			}

			gotBefore := buf.String()
			if gotBefore != tc.wantBefore {
				t.Errorf("before flush: got %q, want %q", gotBefore, tc.wantBefore)
			}

			if err := pw.Flush(); err != nil {
				t.Errorf("Flush() error: %v", err)
			}

			gotAfter := buf.String()
			if gotAfter != tc.wantAfter {
				t.Errorf("after flush: got %q, want %q", gotAfter, tc.wantAfter)
			}
		})
	}
}

func TestCreateLogFile(t *testing.T) {
	tests := map[string]struct {
		runID       string
		specID      string
		wantPath    string
		expectError bool
	}{
		"creates log file": {
			runID:    "run-123",
			specID:   "spec-a",
			wantPath: "run-123/logs/spec-a.log",
		},
		"creates nested log directory": {
			runID:    "20240101_120000_abc12345",
			specID:   "feature-auth",
			wantPath: "20240101_120000_abc12345/logs/feature-auth.log",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			stateDir := t.TempDir()

			file, err := CreateLogFile(stateDir, tc.runID, tc.specID)
			if tc.expectError {
				if err == nil {
					t.Error("expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			defer file.Close()

			// Verify file was created at expected path
			expectedPath := filepath.Join(stateDir, tc.wantPath)
			if file.Name() != expectedPath {
				t.Errorf("file path mismatch: got %s, want %s", file.Name(), expectedPath)
			}

			// Verify we can write to the file
			testData := "test log content\n"
			n, err := file.WriteString(testData)
			if err != nil {
				t.Errorf("error writing to log file: %v", err)
			}
			if n != len(testData) {
				t.Errorf("wrote %d bytes, want %d", n, len(testData))
			}
		})
	}
}

func TestMultiWriter(t *testing.T) {
	tests := map[string]struct {
		specID       string
		input        string
		wantTerminal string
		wantLog      string
	}{
		"writes to both with prefix on terminal": {
			specID:       "spec-a",
			input:        "hello world\n",
			wantTerminal: "[spec-a] hello world\n",
			wantLog:      "hello world\n",
		},
		"multiple lines": {
			specID:       "spec-b",
			input:        "line1\nline2\n",
			wantTerminal: "[spec-b] line1\n[spec-b] line2\n",
			wantLog:      "line1\nline2\n",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var terminal bytes.Buffer
			var logFile bytes.Buffer

			mw := MultiWriter(&terminal, &logFile, tc.specID)

			_, err := mw.Write([]byte(tc.input))
			if err != nil {
				t.Errorf("Write() error: %v", err)
			}

			gotTerminal := terminal.String()
			if gotTerminal != tc.wantTerminal {
				t.Errorf("terminal output mismatch:\ngot:  %q\nwant: %q", gotTerminal, tc.wantTerminal)
			}

			gotLog := logFile.String()
			if gotLog != tc.wantLog {
				t.Errorf("log output mismatch:\ngot:  %q\nwant: %q", gotLog, tc.wantLog)
			}
		})
	}
}

func TestCreateSpecOutput(t *testing.T) {
	tests := map[string]struct {
		runID        string
		specID       string
		input        string
		wantTerminal string
	}{
		"creates combined output": {
			runID:        "run-123",
			specID:       "spec-a",
			input:        "test output\n",
			wantTerminal: "[spec-a] test output\n",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			stateDir := t.TempDir()
			var terminal bytes.Buffer

			writer, cleanup, err := CreateSpecOutput(stateDir, tc.runID, tc.specID, &terminal)
			if err != nil {
				t.Errorf("CreateSpecOutput() error: %v", err)
				return
			}

			// Write test content
			_, err = writer.Write([]byte(tc.input))
			if err != nil {
				t.Errorf("Write() error: %v", err)
			}

			// Call cleanup
			if err := cleanup(); err != nil {
				t.Errorf("cleanup() error: %v", err)
			}

			// Verify terminal output
			gotTerminal := terminal.String()
			if gotTerminal != tc.wantTerminal {
				t.Errorf("terminal mismatch: got %q, want %q", gotTerminal, tc.wantTerminal)
			}

			// Verify log file was created and contains timestamped content
			logPath := GetLogPath(stateDir, tc.runID, tc.specID)
			logContent, err := os.ReadFile(logPath)
			if err != nil {
				t.Errorf("error reading log file: %v", err)
				return
			}

			// Log content should have timestamp prefix [HH:MM:SS]
			logStr := string(logContent)
			if !strings.Contains(logStr, "test output") {
				t.Errorf("log content missing expected text: got %q", logStr)
			}
			// Verify timestamp format [HH:MM:SS]
			if !strings.HasPrefix(logStr, "[") || logStr[9] != ']' {
				t.Errorf("log content missing timestamp prefix: got %q", logStr)
			}
		})
	}
}

func TestGetLogPath(t *testing.T) {
	tests := map[string]struct {
		stateDir string
		runID    string
		specID   string
		want     string
	}{
		"basic path": {
			stateDir: "/tmp/state",
			runID:    "run-123",
			specID:   "spec-a",
			want:     "/tmp/state/run-123/logs/spec-a.log",
		},
		"nested state dir": {
			stateDir: "/home/user/.autospec/state/dag-runs",
			runID:    "20240101_120000_abc12345",
			specID:   "feature-auth",
			want:     "/home/user/.autospec/state/dag-runs/20240101_120000_abc12345/logs/feature-auth.log",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := GetLogPath(tc.stateDir, tc.runID, tc.specID)
			if got != tc.want {
				t.Errorf("GetLogPath() = %s, want %s", got, tc.want)
			}
		})
	}
}

func TestPrefixedWriterConcurrentWrites(t *testing.T) {
	var buf bytes.Buffer
	pw := NewPrefixedWriter(&buf, "test")

	// Simulate concurrent writes by writing from goroutines
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(n int) {
			data := strings.Repeat("x", 100) + "\n"
			_, _ = pw.Write([]byte(data))
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify each line has the prefix
	lines := strings.Split(strings.TrimSuffix(buf.String(), "\n"), "\n")
	for i, line := range lines {
		if !strings.HasPrefix(line, "[test] ") {
			t.Errorf("line %d missing prefix: %q", i, line)
		}
	}
}

func TestNewPrefixedWriter(t *testing.T) {
	tests := map[string]struct {
		specID     string
		wantPrefix string
	}{
		"simple spec id": {
			specID:     "spec-a",
			wantPrefix: "[spec-a] ",
		},
		"spec with numbers": {
			specID:     "feature-123",
			wantPrefix: "[feature-123] ",
		},
		"empty spec id": {
			specID:     "",
			wantPrefix: "[] ",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			pw := NewPrefixedWriter(io.Discard, tc.specID)
			if pw.prefix != tc.wantPrefix {
				t.Errorf("prefix mismatch: got %q, want %q", pw.prefix, tc.wantPrefix)
			}
			if !pw.atLineStart {
				t.Error("atLineStart should be true initially")
			}
		})
	}
}

func TestTruncatingWriter(t *testing.T) {
	tests := map[string]struct {
		maxSize       int64
		writeSize     int
		expectTrunc   bool
		writeMultiple int
	}{
		"no truncation under limit": {
			maxSize:       1024,
			writeSize:     100,
			expectTrunc:   false,
			writeMultiple: 1,
		},
		"truncation when over limit": {
			maxSize:       100,
			writeSize:     50,
			expectTrunc:   true,
			writeMultiple: 5,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Create temp file
			tmpDir := t.TempDir()
			logPath := filepath.Join(tmpDir, "test.log")
			file, err := os.Create(logPath)
			if err != nil {
				t.Fatalf("failed to create file: %v", err)
			}

			// Create writer
			tw := NewTruncatingWriter(file, file, logPath, tc.maxSize)

			// Write data
			data := strings.Repeat("x", tc.writeSize) + "\n"
			for i := 0; i < tc.writeMultiple; i++ {
				_, err := tw.Write([]byte(data))
				if err != nil {
					t.Errorf("Write() error: %v", err)
				}
			}

			file.Sync()
			file.Close()

			// Check file size
			info, err := os.Stat(logPath)
			if err != nil {
				t.Fatalf("failed to stat file: %v", err)
			}

			// File should exist with some content
			if info.Size() == 0 {
				t.Error("file should not be empty")
			}
		})
	}
}
