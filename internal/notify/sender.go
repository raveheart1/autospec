package notify

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Sender defines the interface for platform-specific notification senders
type Sender interface {
	// SendVisual sends a visual notification to the OS notification system
	SendVisual(n Notification) error

	// SendSound plays an audio notification
	SendSound(soundFile string) error

	// VisualAvailable returns true if visual notifications are supported
	VisualAvailable() bool

	// SoundAvailable returns true if sound notifications are supported
	SoundAvailable() bool
}

// NewSender creates a platform-specific notification sender based on the current OS.
// It returns a sender appropriate for darwin (macOS), linux, or windows.
// For unsupported platforms, it returns a no-op sender.
func NewSender() Sender {
	switch runtime.GOOS {
	case "darwin":
		return newDarwinSender()
	case "linux":
		return newLinuxSender()
	case "windows":
		return newWindowsSender()
	default:
		return &noopSender{}
	}
}

// Platform returns the current operating system name
func Platform() string {
	return runtime.GOOS
}

// toolAvailable checks if a command-line tool is available in PATH
func toolAvailable(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// noopSender is a sender that does nothing (for unsupported platforms)
type noopSender struct{}

func (s *noopSender) SendVisual(_ Notification) error { return nil }
func (s *noopSender) SendSound(_ string) error        { return nil }
func (s *noopSender) VisualAvailable() bool           { return false }
func (s *noopSender) SoundAvailable() bool            { return false }

// supportedAudioExtensions contains file extensions supported for custom sounds
var supportedAudioExtensions = map[string]bool{
	".wav":  true,
	".mp3":  true,
	".aiff": true,
	".aif":  true,
	".ogg":  true,
	".flac": true,
	".m4a":  true,
}

// ValidateSoundFile checks if the sound file exists and has a supported format.
// Returns the validated path to use (original if valid, or empty for fallback to default).
// If the file is invalid, logs a warning and returns empty string for fallback.
func ValidateSoundFile(soundFile string) string {
	if soundFile == "" {
		return "" // No custom file, use default
	}

	// Check if file exists
	info, err := os.Stat(soundFile)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("[notify] warning: custom sound file not found: %s, falling back to default", soundFile)
		} else {
			log.Printf("[notify] warning: cannot access custom sound file %s: %v, falling back to default", soundFile, err)
		}
		return ""
	}

	// Check if it's a regular file (not a directory)
	if info.IsDir() {
		log.Printf("[notify] warning: sound path is a directory, not a file: %s, falling back to default", soundFile)
		return ""
	}

	// Validate file extension
	ext := strings.ToLower(filepath.Ext(soundFile))
	if !supportedAudioExtensions[ext] {
		log.Printf("[notify] warning: unsupported audio format '%s' for file: %s, falling back to default", ext, soundFile)
		return ""
	}

	return soundFile
}
