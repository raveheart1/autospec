package progress

import (
	"fmt"
	"os"
	"time"

	"github.com/briandowns/spinner"
)

// ProgressDisplay orchestrates the display of progress indicators
type ProgressDisplay struct {
	capabilities TerminalCapabilities
	currentStage *StageInfo
	spinner      *spinner.Spinner
	symbols      ProgressSymbols
}

// NewProgressDisplay creates a new progress display with the given terminal capabilities
func NewProgressDisplay(caps TerminalCapabilities) *ProgressDisplay {
	return &ProgressDisplay{
		capabilities: caps,
		symbols:      SelectSymbols(caps),
	}
}

// StartStage begins displaying progress for a stage.
// TTY mode: starts animated spinner on stderr (avoids interfering with Claude's stdout).
// Non-TTY mode: prints static message (CI-friendly, no escape codes).
func (p *ProgressDisplay) StartStage(stage StageInfo) error {
	// Validate stage info
	if err := stage.Validate(); err != nil {
		return fmt.Errorf("validating stage info: %w", err)
	}

	p.currentStage = &stage

	// Build the stage message
	msg := buildStageMessage(stage, "Running")

	if p.capabilities.IsTTY {
		// TTY mode: Start spinner animation
		p.spinner = spinner.New(
			spinner.CharSets[p.symbols.SpinnerSet],
			100*time.Millisecond,
		)
		p.spinner.Writer = os.Stderr // Write to stderr to avoid interfering with Claude's stdout
		p.spinner.Suffix = " " + msg
		p.spinner.Start()
	} else {
		// Non-interactive mode: Just print the message
		fmt.Println(msg)
	}

	return nil
}

// UpdateRetry updates the display with retry count information
func (p *ProgressDisplay) UpdateRetry(stage StageInfo) error {
	return p.StartStage(stage)
}

// CompleteStage stops the spinner and displays completion status
func (p *ProgressDisplay) CompleteStage(stage StageInfo) error {
	// Stop spinner if running
	if p.spinner != nil {
		p.spinner.Stop()
		p.spinner = nil
	}

	// Display completion message
	mark := checkmark(p.symbols, p.capabilities.SupportsColor)
	counter := formatStageCounter(stage.Number, stage.TotalStages)
	fmt.Printf("%s %s %s stage complete\n", mark, counter, capitalize(stage.Name))

	p.currentStage = nil
	return nil
}

// FailStage stops the spinner and displays failure status
func (p *ProgressDisplay) FailStage(stage StageInfo, err error) error {
	// Stop spinner if running
	if p.spinner != nil {
		p.spinner.Stop()
		p.spinner = nil
	}

	// Display failure message
	mark := failureMark(p.symbols, p.capabilities.SupportsColor)
	counter := formatStageCounter(stage.Number, stage.TotalStages)
	fmt.Printf("%s %s %s stage failed: %v\n", mark, counter, capitalize(stage.Name), err)

	p.currentStage = nil
	return nil
}

// StopSpinner stops the spinner without showing completion/failure
// This is useful when you want to pause progress display during interactive output
func (p *ProgressDisplay) StopSpinner() {
	if p.spinner != nil {
		p.spinner.Stop()
		p.spinner = nil
	}
}
