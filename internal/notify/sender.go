package notify

import (
	"os/exec"
	"runtime"
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
