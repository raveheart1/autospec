package dag

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// TimestampedWriter wraps an io.Writer and prefixes each line with a timestamp.
// It is thread-safe and handles partial line writes correctly.
type TimestampedWriter struct {
	w          io.Writer
	mu         sync.Mutex
	lineBuffer bytes.Buffer
	timeFunc   func() time.Time
}

// NewTimestampedWriter creates a new TimestampedWriter wrapping the given writer.
func NewTimestampedWriter(w io.Writer) *TimestampedWriter {
	return &TimestampedWriter{
		w:        w,
		timeFunc: time.Now,
	}
}

// Write writes data to the underlying writer, prefixing each complete line
// with a timestamp in [HH:MM:SS] format. Partial lines are buffered until
// a newline is received.
func (tw *TimestampedWriter) Write(p []byte) (int, error) {
	tw.mu.Lock()
	defer tw.mu.Unlock()

	totalWritten := 0
	for len(p) > 0 {
		idx := bytes.IndexByte(p, '\n')
		if idx == -1 {
			// No newline, buffer the partial line
			tw.lineBuffer.Write(p)
			totalWritten += len(p)
			break
		}

		// Complete line found
		n, err := tw.writeCompleteLine(p[:idx])
		if err != nil {
			return totalWritten + n, err
		}
		totalWritten += idx + 1 // Include the newline in count
		p = p[idx+1:]
	}

	return totalWritten, nil
}

// writeCompleteLine writes a complete line with timestamp prefix.
func (tw *TimestampedWriter) writeCompleteLine(lineData []byte) (int, error) {
	timestamp := tw.timeFunc().Format("[15:04:05] ")

	// Combine buffered content with current line
	var fullLine []byte
	if tw.lineBuffer.Len() > 0 {
		fullLine = append(tw.lineBuffer.Bytes(), lineData...)
		tw.lineBuffer.Reset()
	} else {
		fullLine = lineData
	}

	// Write timestamp + line + newline
	_, err := fmt.Fprintf(tw.w, "%s%s\n", timestamp, fullLine)
	if err != nil {
		return 0, fmt.Errorf("writing timestamped line: %w", err)
	}

	return len(lineData), nil
}

// Flush writes any buffered partial line with a timestamp.
// Call this when you're done writing to ensure all content is flushed.
func (tw *TimestampedWriter) Flush() error {
	tw.mu.Lock()
	defer tw.mu.Unlock()

	if tw.lineBuffer.Len() == 0 {
		return nil
	}

	timestamp := tw.timeFunc().Format("[15:04:05] ")
	_, err := fmt.Fprintf(tw.w, "%s%s\n", timestamp, tw.lineBuffer.Bytes())
	if err != nil {
		return fmt.Errorf("flushing partial line: %w", err)
	}
	tw.lineBuffer.Reset()

	return nil
}

// TruncationMarker is the marker added at the beginning of truncated logs.
const TruncationMarker = "[TRUNCATED at %s]\n"

// TruncateLog truncates a log file when it exceeds maxSize bytes.
// It removes the oldest 20% of the file and adds a truncation marker.
// Returns the new file size or an error.
func TruncateLog(path string, maxSize int64) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, fmt.Errorf("checking log file size: %w", err)
	}

	if info.Size() <= maxSize {
		return info.Size(), nil
	}

	return performTruncation(path, info.Size())
}

// performTruncation handles the actual truncation logic.
func performTruncation(path string, currentSize int64) (int64, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return 0, fmt.Errorf("reading log file for truncation: %w", err)
	}

	// Calculate 20% removal point
	removeBytes := int64(float64(currentSize) * 0.2)
	keepFrom := findNextLineStart(content, removeBytes)

	// Create truncation marker
	marker := fmt.Sprintf(TruncationMarker, time.Now().Format("15:04:05"))

	// Write truncated content
	if err := writeTruncatedContent(path, marker, content[keepFrom:]); err != nil {
		return 0, err
	}

	return getFileSize(path)
}

// findNextLineStart finds the start of the next line after offset.
func findNextLineStart(content []byte, offset int64) int64 {
	if offset >= int64(len(content)) {
		return int64(len(content))
	}

	for i := offset; i < int64(len(content)); i++ {
		if content[i] == '\n' {
			return i + 1
		}
	}

	return offset
}

// writeTruncatedContent writes the marker and remaining content to file.
func writeTruncatedContent(path, marker string, remaining []byte) error {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating truncated log file: %w", err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)

	if _, err := writer.WriteString(marker); err != nil {
		return fmt.Errorf("writing truncation marker: %w", err)
	}

	if _, err := writer.Write(remaining); err != nil {
		return fmt.Errorf("writing remaining content: %w", err)
	}

	if err := writer.Flush(); err != nil {
		return fmt.Errorf("flushing truncated content: %w", err)
	}

	return nil
}

// getFileSize returns the current size of the file.
func getFileSize(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, fmt.Errorf("getting truncated file size: %w", err)
	}
	return info.Size(), nil
}

// ShouldTruncate checks if a log file needs truncation.
func ShouldTruncate(path string, maxSize int64) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("checking file size: %w", err)
	}
	return info.Size() > maxSize, nil
}
