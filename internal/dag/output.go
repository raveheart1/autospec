package dag

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
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
	return CreateSpecOutputWithConfig(stateDir, runID, specID, terminal, DefaultDAGConfig())
}

// CreateSpecOutputWithConfig creates output writer with custom config.
func CreateSpecOutputWithConfig(
	stateDir, runID, specID string,
	terminal io.Writer,
	cfg *DAGExecutionConfig,
) (io.Writer, func() error, error) {
	logFile, err := CreateLogFile(stateDir, runID, specID)
	if err != nil {
		return nil, nil, err
	}

	logPath := GetLogPath(stateDir, runID, specID)
	maxSize := cfg.MaxLogSizeBytes()

	// Create timestamped writer wrapping the log file
	timestampedWriter := NewTimestampedWriter(logFile)

	// Create truncating writer that monitors file size
	truncatingWriter := NewTruncatingWriter(timestampedWriter, logFile, logPath, maxSize)

	prefixedTerminal := NewPrefixedWriter(terminal, specID)
	multiWriter := io.MultiWriter(prefixedTerminal, truncatingWriter)

	cleanup := func() error {
		// Flush timestamped writer first
		if err := timestampedWriter.Flush(); err != nil {
			logFile.Close()
			return err
		}
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

// TruncatingWriter wraps a writer and periodically checks if truncation is needed.
// Truncation is checked every checkInterval bytes written to avoid excessive I/O.
type TruncatingWriter struct {
	inner        io.Writer
	file         *os.File
	logPath      string
	maxSize      int64
	bytesWritten int64
}

const truncateCheckInterval = 1024 * 1024 // Check every 1MB of writes

// NewTruncatingWriter creates a writer that monitors and truncates when needed.
func NewTruncatingWriter(inner io.Writer, file *os.File, logPath string, maxSize int64) *TruncatingWriter {
	return &TruncatingWriter{
		inner:   inner,
		file:    file,
		logPath: logPath,
		maxSize: maxSize,
	}
}

// Write writes data and periodically checks if truncation is needed.
func (tw *TruncatingWriter) Write(p []byte) (int, error) {
	n, err := tw.inner.Write(p)
	if err != nil {
		return n, err
	}

	newTotal := atomic.AddInt64(&tw.bytesWritten, int64(n))
	if newTotal >= truncateCheckInterval {
		tw.checkAndTruncate()
		atomic.StoreInt64(&tw.bytesWritten, 0)
	}

	return n, nil
}

// checkAndTruncate checks file size and truncates if needed.
func (tw *TruncatingWriter) checkAndTruncate() {
	// Sync file to ensure accurate size reading
	tw.file.Sync()

	shouldTrunc, err := ShouldTruncate(tw.logPath, tw.maxSize)
	if err != nil || !shouldTrunc {
		return
	}

	// Close and reopen for truncation
	tw.file.Close()
	if _, err := TruncateLog(tw.logPath, tw.maxSize); err != nil {
		// Reopen in append mode even on error
		tw.reopenLogFile()
		return
	}

	tw.reopenLogFile()
}

// reopenLogFile reopens the log file for appending after truncation.
func (tw *TruncatingWriter) reopenLogFile() {
	file, err := os.OpenFile(tw.logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err == nil {
		tw.file = file
		// Update the inner writer if it's a TimestampedWriter
		if tsw, ok := tw.inner.(*TimestampedWriter); ok {
			tsw.w = file
		}
	}
}
