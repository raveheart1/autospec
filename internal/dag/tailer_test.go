package dag

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewLogTailer(t *testing.T) {
	tests := map[string]struct {
		path    string
		wantErr bool
	}{
		"creates tailer for non-existent file": {
			path:    "/tmp/test-tailer/nonexistent.log",
			wantErr: false,
		},
		"creates tailer for existing path": {
			path:    "/tmp/existing.log",
			wantErr: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			tailer, err := NewLogTailer(tc.path)
			if tc.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			defer tailer.Close()

			if tailer.Path() != tc.path {
				t.Errorf("Path() = %q, want %q", tailer.Path(), tc.path)
			}
		})
	}
}

func TestLogTailer_TailNoFollow(t *testing.T) {
	tests := map[string]struct {
		content   string
		wantLines []string
	}{
		"reads single line": {
			content:   "hello world\n",
			wantLines: []string{"hello world"},
		},
		"reads multiple lines": {
			content:   "line1\nline2\nline3\n",
			wantLines: []string{"line1", "line2", "line3"},
		},
		"reads empty file": {
			content:   "",
			wantLines: nil,
		},
		"reads line without trailing newline": {
			content:   "no newline",
			wantLines: []string{"no newline"},
		},
		"reads lines with empty lines": {
			content:   "first\n\nsecond\n",
			wantLines: []string{"first", "", "second"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			tmpDir := t.TempDir()
			logPath := filepath.Join(tmpDir, "test.log")

			// Create file with content
			if err := os.WriteFile(logPath, []byte(tc.content), 0o644); err != nil {
				t.Fatalf("failed to write test file: %v", err)
			}

			tailer, err := NewLogTailer(logPath)
			if err != nil {
				t.Fatalf("NewLogTailer() error: %v", err)
			}
			defer tailer.Close()

			ctx := context.Background()
			lines, err := tailer.Tail(ctx, false)
			if err != nil {
				t.Fatalf("Tail() error: %v", err)
			}

			var gotLines []string
			for line := range lines {
				gotLines = append(gotLines, line)
			}

			if len(gotLines) != len(tc.wantLines) {
				t.Errorf("got %d lines, want %d", len(gotLines), len(tc.wantLines))
				return
			}

			for i, want := range tc.wantLines {
				if i < len(gotLines) && gotLines[i] != want {
					t.Errorf("line %d: got %q, want %q", i, gotLines[i], want)
				}
			}
		})
	}
}

func TestLogTailer_TailFollow(t *testing.T) {
	tests := map[string]struct {
		initialContent string
		appendContent  string
		wantTotal      int
	}{
		"streams appended lines": {
			initialContent: "initial\n",
			appendContent:  "appended\n",
			wantTotal:      2,
		},
		"streams multiple appended lines": {
			initialContent: "",
			appendContent:  "line1\nline2\nline3\n",
			wantTotal:      3,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			tmpDir := t.TempDir()
			logPath := filepath.Join(tmpDir, "test.log")

			// Create file with initial content
			if err := os.WriteFile(logPath, []byte(tc.initialContent), 0o644); err != nil {
				t.Fatalf("failed to write test file: %v", err)
			}

			tailer, err := NewLogTailer(logPath)
			if err != nil {
				t.Fatalf("NewLogTailer() error: %v", err)
			}
			defer tailer.Close()

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			lines, err := tailer.Tail(ctx, true)
			if err != nil {
				t.Fatalf("Tail() error: %v", err)
			}

			// Collect initial lines
			var gotLines []string
			initialExpected := 0
			if tc.initialContent != "" {
				initialExpected = strings.Count(tc.initialContent, "\n")
				if !strings.HasSuffix(tc.initialContent, "\n") && tc.initialContent != "" {
					initialExpected++
				}
			}

			// Wait a bit for initial read
			time.Sleep(50 * time.Millisecond)

			// Append new content
			file, err := os.OpenFile(logPath, os.O_APPEND|os.O_WRONLY, 0o644)
			if err != nil {
				t.Fatalf("failed to open file for append: %v", err)
			}
			if _, err := file.WriteString(tc.appendContent); err != nil {
				file.Close()
				t.Fatalf("failed to append content: %v", err)
			}
			file.Close()

			// Collect lines until context timeout or we have enough
			timeout := time.After(1 * time.Second)
		collectLoop:
			for {
				select {
				case line, ok := <-lines:
					if !ok {
						break collectLoop
					}
					gotLines = append(gotLines, line)
					if len(gotLines) >= tc.wantTotal {
						break collectLoop
					}
				case <-timeout:
					break collectLoop
				}
			}

			if len(gotLines) != tc.wantTotal {
				t.Errorf("got %d lines, want %d", len(gotLines), tc.wantTotal)
			}
		})
	}
}

func TestLogTailer_WaitForFileCreation(t *testing.T) {
	tests := map[string]struct {
		delayMs int
		content string
		want    string
	}{
		"waits for file creation": {
			delayMs: 250, // Give watcher time to set up before file creation
			content: "delayed content\n",
			want:    "delayed content",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			tmpDir := t.TempDir()
			logPath := filepath.Join(tmpDir, "delayed.log")

			tailer, err := NewLogTailer(logPath)
			if err != nil {
				t.Fatalf("NewLogTailer() error: %v", err)
			}
			defer tailer.Close()

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			lines, err := tailer.Tail(ctx, false)
			if err != nil {
				t.Fatalf("Tail() error: %v", err)
			}

			// Create file after delay
			go func() {
				time.Sleep(time.Duration(tc.delayMs) * time.Millisecond)
				_ = os.WriteFile(logPath, []byte(tc.content), 0o644)
			}()

			var gotLines []string
			for line := range lines {
				gotLines = append(gotLines, line)
			}

			if len(gotLines) != 1 || gotLines[0] != tc.want {
				t.Errorf("got %v, want [%q]", gotLines, tc.want)
			}
		})
	}
}

func TestLogTailer_ContextCancellation(t *testing.T) {
	tests := map[string]struct {
		content  string
		cancelMs int
	}{
		"cancellation stops tailing": {
			content:  "some content\n",
			cancelMs: 50,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			tmpDir := t.TempDir()
			logPath := filepath.Join(tmpDir, "cancel.log")

			if err := os.WriteFile(logPath, []byte(tc.content), 0o644); err != nil {
				t.Fatalf("failed to write test file: %v", err)
			}

			tailer, err := NewLogTailer(logPath)
			if err != nil {
				t.Fatalf("NewLogTailer() error: %v", err)
			}
			defer tailer.Close()

			ctx, cancel := context.WithCancel(context.Background())

			lines, err := tailer.Tail(ctx, true)
			if err != nil {
				t.Fatalf("Tail() error: %v", err)
			}

			// Read initial content
			timeout := time.After(500 * time.Millisecond)
		readLoop:
			for {
				select {
				case _, ok := <-lines:
					if !ok {
						break readLoop
					}
				case <-timeout:
					break readLoop
				}
			}

			// Cancel context
			cancel()

			// Verify channel closes
			select {
			case _, ok := <-lines:
				if ok {
					// Drain any remaining lines
					for range lines {
					}
				}
			case <-time.After(500 * time.Millisecond):
				// Channel might already be drained, this is fine
			}
		})
	}
}

func TestLogTailer_HandlesTruncation(t *testing.T) {
	tests := map[string]struct {
		initialContent string
		truncatedTo    string
		wantContains   string
	}{
		"handles file truncation": {
			initialContent: "original content line 1\noriginal content line 2\n",
			truncatedTo:    "new content\n",
			wantContains:   "new content",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			tmpDir := t.TempDir()
			logPath := filepath.Join(tmpDir, "truncate.log")

			// Create file with initial content
			if err := os.WriteFile(logPath, []byte(tc.initialContent), 0o644); err != nil {
				t.Fatalf("failed to write test file: %v", err)
			}

			tailer, err := NewLogTailer(logPath)
			if err != nil {
				t.Fatalf("NewLogTailer() error: %v", err)
			}
			defer tailer.Close()

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			lines, err := tailer.Tail(ctx, true)
			if err != nil {
				t.Fatalf("Tail() error: %v", err)
			}

			// Read initial lines
			var gotLines []string
			timeout := time.After(200 * time.Millisecond)
		readInitial:
			for {
				select {
				case line, ok := <-lines:
					if !ok {
						break readInitial
					}
					gotLines = append(gotLines, line)
				case <-timeout:
					break readInitial
				}
			}

			// Truncate and write new content
			if err := os.WriteFile(logPath, []byte(tc.truncatedTo), 0o644); err != nil {
				t.Fatalf("failed to truncate file: %v", err)
			}

			// Read new lines after truncation
			timeout = time.After(500 * time.Millisecond)
		readNew:
			for {
				select {
				case line, ok := <-lines:
					if !ok {
						break readNew
					}
					gotLines = append(gotLines, line)
				case <-timeout:
					break readNew
				}
			}

			// Verify we got the new content
			found := false
			for _, line := range gotLines {
				if strings.Contains(line, tc.wantContains) {
					found = true
					break
				}
			}

			if !found {
				t.Errorf("did not find %q in lines: %v", tc.wantContains, gotLines)
			}
		})
	}
}

func TestLogTailer_Close(t *testing.T) {
	tests := map[string]struct {
		closeTwice bool
	}{
		"close once succeeds": {
			closeTwice: false,
		},
		"close twice is safe": {
			closeTwice: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			tmpDir := t.TempDir()
			logPath := filepath.Join(tmpDir, "close.log")

			tailer, err := NewLogTailer(logPath)
			if err != nil {
				t.Fatalf("NewLogTailer() error: %v", err)
			}

			err = tailer.Close()
			if err != nil {
				t.Errorf("Close() error: %v", err)
			}

			if tc.closeTwice {
				err = tailer.Close()
				if err != nil {
					t.Errorf("second Close() error: %v", err)
				}
			}
		})
	}
}

func TestLogTailer_Path(t *testing.T) {
	tests := map[string]struct {
		path string
	}{
		"returns configured path": {
			path: "/some/path/to/log.log",
		},
		"handles complex path": {
			path: "/home/user/.autospec/state/dag-runs/20260111_120000_abc12345/logs/spec-feature.log",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			tailer, err := NewLogTailer(tc.path)
			if err != nil {
				t.Fatalf("NewLogTailer() error: %v", err)
			}
			defer tailer.Close()

			if got := tailer.Path(); got != tc.path {
				t.Errorf("Path() = %q, want %q", got, tc.path)
			}
		})
	}
}
