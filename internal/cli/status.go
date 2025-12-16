package cli

import (
	"fmt"

	"github.com/ariel-frischer/autospec/internal/config"
	clierrors "github.com/ariel-frischer/autospec/internal/errors"
	"github.com/ariel-frischer/autospec/internal/spec"
	"github.com/ariel-frischer/autospec/internal/validation"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:     "status [spec-name]",
	Aliases: []string{"st"},
	Short:   "Show implementation progress for current feature",
	Args:    cobra.MaximumNArgs(1),
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		configPath, _ := cmd.Flags().GetString("config")
		verbose, _ := cmd.Flags().GetBool("verbose")

		// Load configuration
		cfg, err := config.Load(configPath)
		if err != nil {
			cliErr := clierrors.ConfigParseError(configPath, err)
			clierrors.PrintError(cliErr)
			return cliErr
		}

		// Detect or get spec
		var metadata *spec.Metadata
		if len(args) > 0 {
			metadata, err = spec.GetSpecMetadata(cfg.SpecsDir, args[0])
		} else {
			metadata, err = spec.DetectCurrentSpec(cfg.SpecsDir)
		}
		if err != nil {
			return fmt.Errorf("failed to detect spec: %w", err)
		}

		fmt.Printf("Feature: %s-%s\n", metadata.Number, metadata.Name)
		fmt.Printf("Status: In Progress\n\n")

		// Parse tasks
		tasksPath := fmt.Sprintf("%s/tasks.md", metadata.Directory)
		phases, err := validation.ParseTasksByPhase(tasksPath)
		if err != nil {
			return fmt.Errorf("failed to parse tasks: %w", err)
		}

		// Display phase progress
		fmt.Println("Phase Progress:")
		totalTasks := 0
		totalChecked := 0
		for _, phase := range phases {
			totalTasks += phase.TotalTasks
			totalChecked += phase.CheckedTasks

			progress := 0
			if phase.TotalTasks > 0 {
				progress = (phase.CheckedTasks * 100) / phase.TotalTasks
			}

			status := "[ ]"
			if phase.CheckedTasks == phase.TotalTasks {
				status = "[✓]"
			} else if phase.CheckedTasks > 0 {
				status = "[~]"
			}

			fmt.Printf("  %s %s: %d/%d tasks (%d%%)\n",
				status, phase.Name, phase.CheckedTasks, phase.TotalTasks, progress)
		}

		fmt.Printf("\nOverall: %d/%d tasks completed (%d%%)\n\n",
			totalChecked, totalTasks, (totalChecked*100)/totalTasks)

		// Show next unchecked tasks
		if !verbose {
			fmt.Println("Next unchecked tasks:")
			count := 0
			for _, phase := range phases {
				for _, task := range phase.Tasks {
					if !task.Checked && count < 5 {
						fmt.Printf("  - %s: %s\n", phase.Name, task.Description)
						count++
					}
					if count >= 5 {
						break
					}
				}
				if count >= 5 {
					break
				}
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
	statusCmd.Flags().BoolP("verbose", "v", false, "Show all tasks, not just unchecked")
}
