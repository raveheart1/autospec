package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/ariel-frischer/autospec/internal/yaml"
	"github.com/spf13/cobra"
)

var yamlCheckCmd = &cobra.Command{
	Use:   "check <file>",
	Short: "Validate YAML syntax",
	Long: `Validate the syntax of a YAML file.

Returns exit code 0 if the file is valid YAML.
Returns non-zero exit code with error details if invalid.

Example:
  autospec yaml check specs/007-yaml-structured-output/spec.yaml`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runYamlCheckWithOutput(args[0], cmd.OutOrStdout())
	},
}

func init() {
	yamlCmd.AddCommand(yamlCheckCmd)
}

// runYamlCheck validates a YAML file and returns an error if invalid.
func runYamlCheck(path string) error {
	return yaml.ValidateFile(path)
}

// runYamlCheckWithOutput validates a YAML file and writes the result to the given writer.
func runYamlCheckWithOutput(path string, out io.Writer) error {
	err := yaml.ValidateFile(path)
	if err != nil {
		fmt.Fprintf(out, "✗ %s has errors:\n  %v\n", path, err)
		return fmt.Errorf("validating YAML file %s: %w", path, err)
	}
	fmt.Fprintf(out, "✓ %s is valid YAML\n", path)
	return nil
}

// yamlCheckExitCode returns the appropriate exit code for yaml check results.
func yamlCheckExitCode(err error) int {
	if err != nil {
		// Use exit code 1 for validation failure (retryable)
		if os.IsNotExist(err) {
			return ExitInvalidArguments // File not found
		}
		return ExitValidationFailed
	}
	return ExitSuccess
}
