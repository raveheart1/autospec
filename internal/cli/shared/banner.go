// Package shared provides constants and types used across CLI subpackages.
package shared

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/fatih/color"
	"golang.org/x/term"
)

// Logo is the ASCII art logo for autospec - minimal block style.
// Both lines are exactly 33 display characters wide.
var Logo = []string{
	"▄▀█ █ █ ▀█▀ █▀█ █▀ █▀█ █▀▀ █▀▀",
	"█▀█ █▄█  █  █▄█ ▄█ █▀▀ ██▄ █▄▄",
}

// LogoDisplayWidth is the visual width of the logo (for centering).
const LogoDisplayWidth = 33

// Tagline is the project tagline.
const Tagline = "Spec-Driven Development Automation"

// Box drawing characters
const (
	BoxTopLeft     = "╭"
	BoxTopRight    = "╮"
	BoxBottomLeft  = "╰"
	BoxBottomRight = "╯"
	BoxHorizontal  = "─"
	BoxVertical    = "│"
)

// GetTerminalWidth returns the terminal width, defaulting to 80 if unavailable.
func GetTerminalWidth() int {
	if width, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil && width > 0 {
		return width
	}
	return 80
}

// CenterText centers text within a given width.
func CenterText(text string, width int) string {
	textLen := len([]rune(text))
	if textLen >= width {
		return text
	}
	padding := (width - textLen) / 2
	return strings.Repeat(" ", padding) + text
}

// PrintBanner prints the colored ASCII logo and tagline.
// Uses cyan for the logo and dim for the tagline.
// Logo and tagline are left-aligned for consistency with command output.
func PrintBanner(out io.Writer) {
	// Color setup
	cyan := color.New(color.FgCyan, color.Bold).SprintFunc()
	dim := color.New(color.Faint).SprintFunc()

	// Print logo left-aligned
	fmt.Fprintln(out)
	for _, line := range Logo {
		fmt.Fprintln(out, cyan(line))
	}
	fmt.Fprintln(out)

	// Tagline left-aligned
	fmt.Fprintln(out, dim(Tagline))
	fmt.Fprintln(out)
}

// PrintBannerCompact prints just the logo without tagline (for init command).
// Logo is left-aligned for consistency with command output.
func PrintBannerCompact(out io.Writer) {
	// Color setup
	cyan := color.New(color.FgCyan, color.Bold).SprintFunc()

	// Print logo left-aligned
	fmt.Fprintln(out)
	for _, line := range Logo {
		fmt.Fprintln(out, cyan(line))
	}
	fmt.Fprintln(out)
}

// Colors provides reusable color functions for CLI output.
type Colors struct {
	Cyan    func(a ...interface{}) string
	Green   func(a ...interface{}) string
	Yellow  func(a ...interface{}) string
	Red     func(a ...interface{}) string
	Dim     func(a ...interface{}) string
	White   func(a ...interface{}) string
	Magenta func(a ...interface{}) string
}

// NewColors creates a new Colors instance with standard terminal colors.
func NewColors() *Colors {
	return &Colors{
		Cyan:    color.New(color.FgCyan, color.Bold).SprintFunc(),
		Green:   color.New(color.FgGreen).SprintFunc(),
		Yellow:  color.New(color.FgYellow).SprintFunc(),
		Red:     color.New(color.FgRed).SprintFunc(),
		Dim:     color.New(color.Faint).SprintFunc(),
		White:   color.New(color.FgWhite, color.Bold).SprintFunc(),
		Magenta: color.New(color.FgMagenta).SprintFunc(),
	}
}
