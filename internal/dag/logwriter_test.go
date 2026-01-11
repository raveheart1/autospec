package dag

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestTimestampedWriter_Write(t *testing.T) {
	tests := map[string]struct {
		input      string
		wantPrefix string
		wantSuffix string
	}{
		"single line with newline": {
			input:      "hello world\n",
			wantPrefix: "[",
			wantSuffix: "] hello world\n",
		},
		"multiple lines": {
			input:      "line1\nline2\n",
			wantPrefix: "[",
			wantSuffix: "] line2\n",
		},
		"empty line": {
			input:      "\n",
			wantPrefix: "[",
			wantSuffix: "] \n",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var buf bytes.Buffer
			tw := NewTimestampedWriter(&buf)

			n, err := tw.Write([]byte(tc.input))
			if err != nil {
				t.Fatalf("Write() error: %v", err)
			}

			if n != len(tc.input) {
				t.Errorf("Write() = %d, want %d", n, len(tc.input))
			}

			output := buf.String()
			if !strings.HasPrefix(output, tc.wantPrefix) {
				t.Errorf("output %q should start with %q", output, tc.wantPrefix)
			}
			if !strings.HasSuffix(output, tc.wantSuffix) {
				t.Errorf("output %q should end with %q", output, tc.wantSuffix)
			}
		})
	}
}

func TestTimestampedWriter_PartialLines(t *testing.T) {
	tests := map[string]struct {
		writes     []string
		wantLines  int
		wantFlush  bool
		flushLines int
	}{
		"partial then complete": {
			writes:    []string{"hello ", "world\n"},
			wantLines: 1,
			wantFlush: false,
		},
		"partial needs flush": {
			writes:     []string{"partial"},
			wantLines:  0,
			wantFlush:  true,
			flushLines: 1,
		},
		"multiple partials then complete": {
			writes:    []string{"a", "b", "c\n"},
			wantLines: 1,
			wantFlush: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var buf bytes.Buffer
			tw := NewTimestampedWriter(&buf)

			for _, w := range tc.writes {
				if _, err := tw.Write([]byte(w)); err != nil {
					t.Fatalf("Write() error: %v", err)
				}
			}

			lines := strings.Count(buf.String(), "\n")
			if lines != tc.wantLines {
				t.Errorf("got %d lines before flush, want %d", lines, tc.wantLines)
			}

			if tc.wantFlush {
				if err := tw.Flush(); err != nil {
					t.Fatalf("Flush() error: %v", err)
				}
				lines = strings.Count(buf.String(), "\n")
				if lines != tc.flushLines {
					t.Errorf("got %d lines after flush, want %d", lines, tc.flushLines)
				}
			}
		})
	}
}

func TestTimestampedWriter_TimestampFormat(t *testing.T) {
	tests := map[string]struct {
		fixedTime time.Time
		input     string
		wantTime  string
	}{
		"morning time": {
			fixedTime: time.Date(2026, 1, 11, 9, 5, 3, 0, time.UTC),
			input:     "test\n",
			wantTime:  "[09:05:03]",
		},
		"afternoon time": {
			fixedTime: time.Date(2026, 1, 11, 14, 30, 45, 0, time.UTC),
			input:     "test\n",
			wantTime:  "[14:30:45]",
		},
		"midnight": {
			fixedTime: time.Date(2026, 1, 11, 0, 0, 0, 0, time.UTC),
			input:     "test\n",
			wantTime:  "[00:00:00]",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var buf bytes.Buffer
			tw := NewTimestampedWriter(&buf)
			tw.timeFunc = func() time.Time { return tc.fixedTime }

			if _, err := tw.Write([]byte(tc.input)); err != nil {
				t.Fatalf("Write() error: %v", err)
			}

			output := buf.String()
			if !strings.Contains(output, tc.wantTime) {
				t.Errorf("output %q should contain %q", output, tc.wantTime)
			}
		})
	}
}

func TestTimestampedWriter_MultiLine(t *testing.T) {
	tests := map[string]struct {
		input         string
		wantLineCount int
	}{
		"three lines": {
			input:         "line1\nline2\nline3\n",
			wantLineCount: 3,
		},
		"five lines": {
			input:         "a\nb\nc\nd\ne\n",
			wantLineCount: 5,
		},
		"single line": {
			input:         "only one\n",
			wantLineCount: 1,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var buf bytes.Buffer
			tw := NewTimestampedWriter(&buf)

			if _, err := tw.Write([]byte(tc.input)); err != nil {
				t.Fatalf("Write() error: %v", err)
			}

			output := buf.String()
			lines := strings.Count(output, "\n")
			if lines != tc.wantLineCount {
				t.Errorf("got %d lines, want %d", lines, tc.wantLineCount)
			}

			// Each line should have a timestamp prefix
			for _, line := range strings.Split(output, "\n") {
				if line != "" && !strings.HasPrefix(line, "[") {
					t.Errorf("line %q missing timestamp prefix", line)
				}
			}
		})
	}
}

func TestTimestampedWriter_ThreadSafety(t *testing.T) {
	tests := map[string]struct {
		goroutines int
		writes     int
	}{
		"10 goroutines 100 writes each": {
			goroutines: 10,
			writes:     100,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var buf bytes.Buffer
			tw := NewTimestampedWriter(&buf)

			done := make(chan bool, tc.goroutines)
			for i := 0; i < tc.goroutines; i++ {
				go func(id int) {
					for j := 0; j < tc.writes; j++ {
						_, _ = tw.Write([]byte("test line\n"))
					}
					done <- true
				}(i)
			}

			for i := 0; i < tc.goroutines; i++ {
				<-done
			}

			lines := strings.Count(buf.String(), "\n")
			expected := tc.goroutines * tc.writes
			if lines != expected {
				t.Errorf("got %d lines, want %d", lines, expected)
			}
		})
	}
}

func TestTruncateLog(t *testing.T) {
	tests := map[string]struct {
		content      string
		maxSize      int64
		wantTruncate bool
		wantMarker   bool
	}{
		"file under limit": {
			content:      "small content\n",
			maxSize:      1000,
			wantTruncate: false,
			wantMarker:   false,
		},
		"file at limit": {
			content:      strings.Repeat("x", 100) + "\n",
			maxSize:      101,
			wantTruncate: false,
			wantMarker:   false,
		},
		"file over limit": {
			content:      strings.Repeat("line\n", 100),
			maxSize:      100,
			wantTruncate: true,
			wantMarker:   true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			tmpDir := t.TempDir()
			logPath := filepath.Join(tmpDir, "test.log")

			if err := os.WriteFile(logPath, []byte(tc.content), 0o644); err != nil {
				t.Fatalf("failed to write test file: %v", err)
			}

			originalSize := int64(len(tc.content))

			newSize, err := TruncateLog(logPath, tc.maxSize)
			if err != nil {
				t.Fatalf("TruncateLog() error: %v", err)
			}

			if tc.wantTruncate {
				if newSize >= originalSize {
					t.Errorf("expected smaller size after truncation: got %d, original %d",
						newSize, originalSize)
				}
			} else {
				if newSize != originalSize {
					t.Errorf("expected unchanged size: got %d, want %d", newSize, originalSize)
				}
			}

			if tc.wantMarker {
				content, err := os.ReadFile(logPath)
				if err != nil {
					t.Fatalf("failed to read truncated file: %v", err)
				}
				if !strings.Contains(string(content), "[TRUNCATED at") {
					t.Error("truncation marker not found in file")
				}
			}
		})
	}
}

func TestTruncateLog_RemovesOldest20Percent(t *testing.T) {
	tests := map[string]struct {
		lineCount   int
		maxSize     int64
		wantRemoved float64
		tolerance   float64
	}{
		"100 lines truncated": {
			lineCount:   100,
			maxSize:     100,
			wantRemoved: 0.20,
			tolerance:   0.05,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			tmpDir := t.TempDir()
			logPath := filepath.Join(tmpDir, "test.log")

			// Create file with numbered lines
			var content strings.Builder
			for i := 0; i < tc.lineCount; i++ {
				content.WriteString("line content here\n")
			}

			if err := os.WriteFile(logPath, []byte(content.String()), 0o644); err != nil {
				t.Fatalf("failed to write test file: %v", err)
			}

			originalSize := int64(content.Len())

			_, err := TruncateLog(logPath, tc.maxSize)
			if err != nil {
				t.Fatalf("TruncateLog() error: %v", err)
			}

			truncated, err := os.ReadFile(logPath)
			if err != nil {
				t.Fatalf("failed to read file: %v", err)
			}

			// Account for truncation marker in size comparison
			markerSize := len("[TRUNCATED at 00:00:00]\n")
			effectiveRemoved := float64(originalSize-int64(len(truncated))+int64(markerSize)) / float64(originalSize)

			if effectiveRemoved < tc.wantRemoved-tc.tolerance ||
				effectiveRemoved > tc.wantRemoved+tc.tolerance {
				t.Errorf("removed %.2f%%, want ~%.2f%% (tolerance %.2f%%)",
					effectiveRemoved*100, tc.wantRemoved*100, tc.tolerance*100)
			}
		})
	}
}

func TestTruncateLog_PreservesLineIntegrity(t *testing.T) {
	tests := map[string]struct {
		linePrefix string
	}{
		"numbered lines": {
			linePrefix: "line-",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			tmpDir := t.TempDir()
			logPath := filepath.Join(tmpDir, "test.log")

			// Create 50 numbered lines
			var content strings.Builder
			for i := 0; i < 50; i++ {
				content.WriteString(tc.linePrefix)
				content.WriteString("content\n")
			}

			if err := os.WriteFile(logPath, []byte(content.String()), 0o644); err != nil {
				t.Fatalf("failed to write test file: %v", err)
			}

			_, err := TruncateLog(logPath, 100)
			if err != nil {
				t.Fatalf("TruncateLog() error: %v", err)
			}

			truncated, err := os.ReadFile(logPath)
			if err != nil {
				t.Fatalf("failed to read file: %v", err)
			}

			lines := strings.Split(string(truncated), "\n")
			for i, line := range lines {
				if line == "" {
					continue
				}
				// First line should be truncation marker
				if i == 0 {
					if !strings.HasPrefix(line, "[TRUNCATED at") {
						t.Errorf("first line should be truncation marker: %q", line)
					}
					continue
				}
				// Other lines should start with line prefix
				if !strings.HasPrefix(line, tc.linePrefix) {
					t.Errorf("line %d malformed: %q", i, line)
				}
			}
		})
	}
}

func TestTruncateLog_NonExistentFile(t *testing.T) {
	tests := map[string]struct {
		path    string
		wantErr bool
	}{
		"non-existent file": {
			path:    "/nonexistent/path/file.log",
			wantErr: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := TruncateLog(tc.path, 1000)
			if tc.wantErr && err == nil {
				t.Error("expected error for non-existent file")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestShouldTruncate(t *testing.T) {
	tests := map[string]struct {
		content string
		maxSize int64
		want    bool
		wantErr bool
	}{
		"file under limit": {
			content: "small\n",
			maxSize: 100,
			want:    false,
		},
		"file at limit": {
			content: strings.Repeat("x", 100),
			maxSize: 100,
			want:    false,
		},
		"file over limit": {
			content: strings.Repeat("x", 101),
			maxSize: 100,
			want:    true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			tmpDir := t.TempDir()
			logPath := filepath.Join(tmpDir, "test.log")

			if err := os.WriteFile(logPath, []byte(tc.content), 0o644); err != nil {
				t.Fatalf("failed to write test file: %v", err)
			}

			got, err := ShouldTruncate(logPath, tc.maxSize)
			if tc.wantErr {
				if err == nil {
					t.Error("expected error")
				}
				return
			}

			if err != nil {
				t.Fatalf("ShouldTruncate() error: %v", err)
			}

			if got != tc.want {
				t.Errorf("ShouldTruncate() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestShouldTruncate_NonExistentFile(t *testing.T) {
	tests := map[string]struct {
		want    bool
		wantErr bool
	}{
		"non-existent file returns false": {
			want:    false,
			wantErr: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := ShouldTruncate("/nonexistent/file.log", 1000)

			if tc.wantErr && err == nil {
				t.Error("expected error")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("ShouldTruncate() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestTimestampedWriter_Flush(t *testing.T) {
	tests := map[string]struct {
		writes      []string
		wantContent string
	}{
		"flush empty buffer": {
			writes:      []string{},
			wantContent: "",
		},
		"flush after complete line": {
			writes:      []string{"complete\n"},
			wantContent: "] complete\n",
		},
		"flush partial line": {
			writes:      []string{"partial"},
			wantContent: "] partial\n",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var buf bytes.Buffer
			tw := NewTimestampedWriter(&buf)

			for _, w := range tc.writes {
				if _, err := tw.Write([]byte(w)); err != nil {
					t.Fatalf("Write() error: %v", err)
				}
			}

			if err := tw.Flush(); err != nil {
				t.Fatalf("Flush() error: %v", err)
			}

			output := buf.String()
			if tc.wantContent != "" && !strings.Contains(output, tc.wantContent) {
				t.Errorf("output %q should contain %q", output, tc.wantContent)
			}
			if tc.wantContent == "" && output != "" {
				t.Errorf("expected empty output, got %q", output)
			}
		})
	}
}
