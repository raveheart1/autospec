package errors

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/fatih/color"
)

var (
	// Color functions with auto-detection for terminal support.
	// These fall back gracefully when colors are unavailable.
	errorLabel  = color.New(color.FgRed, color.Bold).SprintFunc()
	errorMsg    = color.New(color.FgRed).SprintFunc()
	fixLabel    = color.New(color.FgGreen, color.Bold).SprintFunc()
	usageLabel  = color.New(color.FgCyan, color.Bold).SprintFunc()
	usageText   = color.New(color.FgCyan).SprintFunc()
	bullet      = color.New(color.FgGreen).SprintFunc()
	categoryFmt = color.New(color.FgYellow).SprintFunc()
)

// FormatError formats a CLIError for display in the terminal.
// It uses colors when available and falls back to plain text otherwise.
func FormatError(err *CLIError) string {
	if err == nil {
		return ""
	}
	return formatError(err, true)
}

// FormatErrorPlain formats a CLIError without colors.
func FormatErrorPlain(err *CLIError) string {
	if err == nil {
		return ""
	}
	return formatError(err, false)
}

func formatError(err *CLIError, useColors bool) string {
	var sb strings.Builder

	// Error category and message
	if useColors {
		sb.WriteString(errorLabel("Error"))
		sb.WriteString(" [")
		sb.WriteString(categoryFmt(err.Category.String()))
		sb.WriteString("]: ")
		sb.WriteString(errorMsg(err.Message))
	} else {
		sb.WriteString("Error [")
		sb.WriteString(err.Category.String())
		sb.WriteString("]: ")
		sb.WriteString(err.Message)
	}
	sb.WriteString("\n")

	// Correct usage (for argument errors)
	if err.Usage != "" {
		sb.WriteString("\n")
		if useColors {
			sb.WriteString(usageLabel("Usage: "))
			sb.WriteString(usageText(err.Usage))
		} else {
			sb.WriteString("Usage: ")
			sb.WriteString(err.Usage)
		}
		sb.WriteString("\n")
	}

	// Remediation steps
	if len(err.Remediation) > 0 {
		sb.WriteString("\n")
		if useColors {
			sb.WriteString(fixLabel("To fix this:"))
		} else {
			sb.WriteString("To fix this:")
		}
		sb.WriteString("\n")
		for _, step := range err.Remediation {
			if useColors {
				sb.WriteString("  ")
				sb.WriteString(bullet("•"))
				sb.WriteString(" ")
			} else {
				sb.WriteString("  • ")
			}
			sb.WriteString(step)
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

// PrintError prints a formatted CLIError to stderr.
func PrintError(err *CLIError) {
	FprintError(os.Stderr, err)
}

// FprintError prints a formatted CLIError to the given writer.
func FprintError(w io.Writer, err *CLIError) {
	if err == nil {
		return
	}
	fmt.Fprint(w, FormatError(err))
}

// FormatSimpleError formats a regular error with a category.
// Use this when you have a plain error and want structured output.
func FormatSimpleError(err error, category ErrorCategory) string {
	if err == nil {
		return ""
	}
	cliErr := &CLIError{
		Category: category,
		Message:  err.Error(),
	}
	return FormatError(cliErr)
}

// PrintSimpleError prints a formatted regular error to stderr.
func PrintSimpleError(err error, category ErrorCategory) {
	fmt.Fprint(os.Stderr, FormatSimpleError(err, category))
}
