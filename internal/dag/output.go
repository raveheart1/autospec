package dag

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
)

// PrefixedWriter wraps an io.Writer and prefixes each line with [spec-id].
type PrefixedWriter struct {
	writer      io.Writer
	prefix      string
	mu          sync.Mutex
	buffer      bytes.Buffer
	atLineStart bool
}

// NewPrefixedWriter creates a new PrefixedWriter.
func NewPrefixedWriter(w io.Writer, specID string) *PrefixedWriter {
	return &PrefixedWriter{
		writer:      w,
		prefix:      fmt.Sprintf("[%s] ", specID),
		atLineStart: true,
	}
}

// Write implements io.Writer, prefixing each line with the spec ID.
func (pw *PrefixedWriter) Write(p []byte) (n int, err error) {
	pw.mu.Lock()
	defer pw.mu.Unlock()

	originalLen := len(p)
	for len(p) > 0 {
		if pw.atLineStart {
			if _, err := pw.writer.Write([]byte(pw.prefix)); err != nil {
				return originalLen - len(p), err
			}
			pw.atLineStart = false
		}

		idx := bytes.IndexByte(p, '\n')
		if idx == -1 {
			// No newline, write remaining data
			if _, err := pw.writer.Write(p); err != nil {
				return originalLen - len(p), err
			}
			break
		}

		// Write up to and including the newline
		if _, err := pw.writer.Write(p[:idx+1]); err != nil {
			return originalLen - len(p), err
		}
		p = p[idx+1:]
		pw.atLineStart = true
	}

	return originalLen, nil
}

// Flush flushes any remaining buffered data.
// If the last line didn't end with a newline, this adds one.
func (pw *PrefixedWriter) Flush() error {
	pw.mu.Lock()
	defer pw.mu.Unlock()

	if !pw.atLineStart {
		if _, err := pw.writer.Write([]byte("\n")); err != nil {
			return err
		}
		pw.atLineStart = true
	}
	return nil
}

// CreateLogFile creates a log file for a spec in the run's log directory.
// Returns the file handle that must be closed by the caller.
func CreateLogFile(stateDir, runID, specID string) (*os.File, error) {
	logDir := GetLogDir(stateDir, runID)
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating log directory: %w", err)
	}

	logPath := filepath.Join(logDir, fmt.Sprintf("%s.log", specID))
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, fmt.Errorf("creating log file: %w", err)
	}

	return file, nil
}

// MultiWriter creates an io.Writer that writes to both terminal and log file.
// The terminal output is prefixed with [spec-id], while the log file gets raw output.
func MultiWriter(terminal io.Writer, logFile io.Writer, specID string) io.Writer {
	prefixedTerminal := NewPrefixedWriter(terminal, specID)
	return io.MultiWriter(prefixedTerminal, logFile)
}

// CreateSpecOutput creates a combined output writer for a spec execution.
// Returns the writer and a cleanup function that must be called when done.
func CreateSpecOutput(stateDir, runID, specID string, terminal io.Writer) (io.Writer, func() error, error) {
	logFile, err := CreateLogFile(stateDir, runID, specID)
	if err != nil {
		return nil, nil, err
	}

	prefixedTerminal := NewPrefixedWriter(terminal, specID)
	multiWriter := io.MultiWriter(prefixedTerminal, logFile)

	cleanup := func() error {
		// Flush prefixed writer to ensure final newline
		if err := prefixedTerminal.Flush(); err != nil {
			logFile.Close()
			return err
		}
		return logFile.Close()
	}

	return multiWriter, cleanup, nil
}

// GetLogPath returns the path to a spec's log file.
func GetLogPath(stateDir, runID, specID string) string {
	return filepath.Join(GetLogDir(stateDir, runID), fmt.Sprintf("%s.log", specID))
}
