package cli

import (
	"fmt"
	"os"

	"github.com/ariel-frischer/autospec/internal/commands"
	"github.com/ariel-frischer/autospec/internal/prereqs"
	"github.com/spf13/cobra"
)

var (
	renderOutput string
)

var renderCommandCmd = &cobra.Command{
	Use:   "render-command <command-name>",
	Short: "Render a command template with current feature context",
	Long: `Render an autospec command template with the current feature context.

This command pre-computes prereqs context (feature directory, spec paths, etc.)
and renders the specified command template with those values already filled in.

The rendered output can be used directly or piped to an agent.`,
	Example: `  # Render the plan command for current feature
  autospec render-command autospec.plan

  # Save rendered command to a file
  autospec render-command autospec.tasks --output /tmp/tasks-prompt.md

  # Pipe to clipboard (macOS)
  autospec render-command autospec.implement | pbcopy`,
	Args: cobra.ExactArgs(1),
	RunE: runRenderCommand,
}

func init() {
	renderCommandCmd.GroupID = GroupInternal
	renderCommandCmd.Flags().StringVarP(&renderOutput, "output", "o", "", "Output file path (default: stdout)")
	rootCmd.AddCommand(renderCommandCmd)
}

func runRenderCommand(cmd *cobra.Command, args []string) error {
	commandName := args[0]

	specsDir, err := cmd.Flags().GetString("specs-dir")
	if err != nil || specsDir == "" {
		specsDir = "./specs"
	}

	content, err := commands.GetTemplate(commandName)
	if err != nil {
		return fmt.Errorf("loading template %s: %w", commandName, err)
	}

	opts := getOptionsForCommand(commandName, specsDir)

	ctx, err := prereqs.ComputeContext(opts)
	if err != nil {
		return fmt.Errorf("computing prereqs context: %w", err)
	}

	rendered, err := commands.RenderAndValidate(commandName, content, ctx)
	if err != nil {
		return fmt.Errorf("rendering template: %w", err)
	}

	if renderOutput != "" {
		if err := os.WriteFile(renderOutput, rendered, 0o644); err != nil {
			return fmt.Errorf("writing output file: %w", err)
		}
		fmt.Printf("Rendered template written to %s\n", renderOutput)
		return nil
	}

	fmt.Print(string(rendered))
	return nil
}

// getOptionsForCommand returns the appropriate prereqs options based on command requirements.
func getOptionsForCommand(commandName, specsDir string) prereqs.Options {
	requiredVars := commands.GetRequiredVars(commandName)

	opts := prereqs.Options{
		SpecsDir: specsDir,
	}

	for _, v := range requiredVars {
		switch v {
		case "FeatureSpec":
			opts.RequireSpec = true
		case "ImplPlan":
			opts.RequirePlan = true
		case "TasksFile":
			opts.RequireTasks = true
		}
	}

	if len(requiredVars) == 0 {
		opts.PathsOnly = true
	}

	return opts
}
