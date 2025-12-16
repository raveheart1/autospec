// Package health provides dependency health checks for autospec. It validates that
// required external tools (Claude CLI, Git) are available and properly configured,
// returning structured reports used by the 'autospec doctor' command.
package health

import (
	"fmt"
	"os/exec"
)

// CheckResult represents the result of a single health check
type CheckResult struct {
	Name    string
	Passed  bool
	Message string
}

// HealthReport contains all health check results
type HealthReport struct {
	Checks []CheckResult
	Passed bool
}

// RunHealthChecks runs all health checks and returns a report
func RunHealthChecks() *HealthReport {
	report := &HealthReport{
		Checks: make([]CheckResult, 0),
		Passed: true,
	}

	// Check Claude CLI
	claudeCheck := CheckClaudeCLI()
	report.Checks = append(report.Checks, claudeCheck)
	if !claudeCheck.Passed {
		report.Passed = false
	}

	// Check Git
	gitCheck := CheckGit()
	report.Checks = append(report.Checks, gitCheck)
	if !gitCheck.Passed {
		report.Passed = false
	}

	return report
}

// CheckClaudeCLI checks if the Claude CLI is available
func CheckClaudeCLI() CheckResult {
	_, err := exec.LookPath("claude")
	if err != nil {
		return CheckResult{
			Name:    "Claude CLI",
			Passed:  false,
			Message: "Claude CLI not found in PATH",
		}
	}

	return CheckResult{
		Name:    "Claude CLI",
		Passed:  true,
		Message: "Claude CLI found",
	}
}

// CheckGit checks if Git is available
func CheckGit() CheckResult {
	_, err := exec.LookPath("git")
	if err != nil {
		return CheckResult{
			Name:    "Git",
			Passed:  false,
			Message: "Git not found in PATH",
		}
	}

	return CheckResult{
		Name:    "Git",
		Passed:  true,
		Message: "Git found",
	}
}

// FormatReport formats the health report for console output
func FormatReport(report *HealthReport) string {
	var output string

	for _, check := range report.Checks {
		if check.Passed {
			output += fmt.Sprintf("✓ %s found\n", check.Name)
		} else {
			output += fmt.Sprintf("✗ Error: %s\n", check.Message)
		}
	}

	return output
}
