package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/anthropics/auto-claude-speckit/internal/config"
	"github.com/anthropics/auto-claude-speckit/internal/spec"
	"github.com/anthropics/auto-claude-speckit/internal/workflow"
	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run [feature-description]",
	Short: "Run selected workflow phases with flexible phase selection",
	Long: `Run selected workflow phases with flexible phase selection.

Use individual phase flags to select which phases to execute:
  -s, --specify    Include specify phase (requires feature description)
  -p, --plan       Include plan phase
  -t, --tasks      Include tasks phase
  -i, --implement  Include implement phase
  -a, --all        Run all phases (equivalent to -spti)

Phases are always executed in canonical order: specify -> plan -> tasks -> implement

Examples:
  # Run only plan and implement phases on current branch's spec
  autospec run -pi

  # Run all phases for a new feature
  autospec run -a "Add user authentication"

  # Run tasks and implement on a specific spec
  autospec run -ti --spec 007-yaml-output

  # Run plan phase with custom prompt
  autospec run -p "Focus on security best practices"

  # Skip confirmation prompts for automation
  autospec run -ti -y`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get phase flags
		specify, _ := cmd.Flags().GetBool("specify")
		plan, _ := cmd.Flags().GetBool("plan")
		tasks, _ := cmd.Flags().GetBool("tasks")
		implement, _ := cmd.Flags().GetBool("implement")
		all, _ := cmd.Flags().GetBool("all")

		// Get other flags
		specName, _ := cmd.Flags().GetString("spec")
		skipConfirm, _ := cmd.Flags().GetBool("yes")
		configPath, _ := cmd.Flags().GetString("config")
		skipPreflight, _ := cmd.Flags().GetBool("skip-preflight")
		maxRetries, _ := cmd.Flags().GetInt("max-retries")
		resume, _ := cmd.Flags().GetBool("resume")
		debug, _ := cmd.Flags().GetBool("debug")
		progress, _ := cmd.Flags().GetBool("progress")

		// Build PhaseConfig from flags
		phaseConfig := workflow.NewPhaseConfig()
		if all {
			phaseConfig.SetAll()
		} else {
			phaseConfig.Specify = specify
			phaseConfig.Plan = plan
			phaseConfig.Tasks = tasks
			phaseConfig.Implement = implement
		}

		// Validate at least one phase is selected
		if !phaseConfig.HasAnyPhase() {
			return fmt.Errorf("no phases selected. Use -s/-p/-t/-i flags or -a for all phases\n\nRun 'autospec run --help' for usage")
		}

		// Get feature description from args if specify phase is selected
		var featureDescription string
		if phaseConfig.Specify {
			if len(args) < 1 {
				return fmt.Errorf("feature description required when using specify phase (-s)\n\nUsage: autospec run -s \"feature description\"")
			}
			featureDescription = args[0]
		} else if len(args) > 0 {
			// If not specifying but args provided, treat as prompt
			featureDescription = args[0]
		}

		// Load configuration
		cfg, err := config.Load(configPath)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Override settings from flags
		if cmd.Flags().Changed("skip-preflight") {
			cfg.SkipPreflight = skipPreflight
		}
		if cmd.Flags().Changed("max-retries") {
			cfg.MaxRetries = maxRetries
		}
		if cmd.Flags().Changed("progress") {
			cfg.ShowProgress = progress
		}

		// Resolve skip confirmations (flag > env > config)
		if skipConfirm || os.Getenv("AUTOSPEC_YES") != "" || cfg.SkipConfirmations {
			cfg.SkipConfirmations = true
		}

		// Detect or validate spec name
		var specMetadata *spec.Metadata
		if !phaseConfig.Specify {
			// Need to detect or validate spec if not starting with specify
			if specName != "" {
				// Validate explicit spec exists
				specDir := filepath.Join(cfg.SpecsDir, specName)
				if _, err := os.Stat(specDir); os.IsNotExist(err) {
					return fmt.Errorf("spec not found: %s\n\nRun 'autospec specify' to create a new spec or check the spec name", specName)
				}
				specMetadata = &spec.Metadata{
					Name:      specName,
					Directory: specDir,
				}
			} else {
				// Auto-detect from git branch
				specMetadata, err = spec.DetectCurrentSpec(cfg.SpecsDir)
				if err != nil {
					return fmt.Errorf("failed to detect spec: %w\n\nUse --spec flag to specify explicitly or checkout a spec branch", err)
				}
				fmt.Printf("Detected spec: %s-%s\n", specMetadata.Number, specMetadata.Name)
			}
		}

		// Check artifact dependencies before execution
		if !phaseConfig.Specify {
			missingArtifacts := checkMissingArtifacts(phaseConfig, specMetadata.Directory)
			if len(missingArtifacts) > 0 {
				warningMsg := generateArtifactWarning(phaseConfig, missingArtifacts)
				fmt.Fprint(os.Stderr, warningMsg)

				if !cfg.SkipConfirmations {
					// Prompt for confirmation
					shouldContinue, promptErr := workflow.PromptUserToContinue("")
					if promptErr != nil {
						return promptErr
					}
					if !shouldContinue {
						return fmt.Errorf("operation cancelled by user")
					}
				} else {
					fmt.Fprintln(os.Stderr, "Proceeding (skip_confirmations enabled)...")
				}
			}
		}

		// Create workflow orchestrator
		orchestrator := workflow.NewWorkflowOrchestrator(cfg)
		orchestrator.Debug = debug
		orchestrator.Executor.Debug = debug

		if debug {
			fmt.Println("[DEBUG] Debug mode enabled")
			fmt.Printf("[DEBUG] Config: %+v\n", cfg)
			fmt.Printf("[DEBUG] PhaseConfig: %+v\n", phaseConfig)
		}

		// Execute phases in canonical order
		return executePhases(orchestrator, phaseConfig, featureDescription, specMetadata, resume, debug)
	},
}

// checkMissingArtifacts checks if required artifacts are missing for the selected phases
func checkMissingArtifacts(phaseConfig *workflow.PhaseConfig, specDir string) []string {
	requiredArtifacts := phaseConfig.GetAllRequiredArtifacts()
	var missing []string

	for _, artifact := range requiredArtifacts {
		artifactPath := filepath.Join(specDir, artifact)
		if _, err := os.Stat(artifactPath); os.IsNotExist(err) {
			missing = append(missing, artifact)
		}
	}

	return missing
}

// generateArtifactWarning generates a warning message for missing artifacts
func generateArtifactWarning(phaseConfig *workflow.PhaseConfig, missingArtifacts []string) string {
	msg := "\nWARNING: Missing prerequisite artifacts:\n"
	for _, artifact := range missingArtifacts {
		msg += fmt.Sprintf("  - %s\n", artifact)
	}
	msg += "\nThe following phases require these artifacts:\n"

	for _, phase := range phaseConfig.GetSelectedPhases() {
		requires := workflow.GetRequiredArtifacts(phase)
		for _, req := range requires {
			for _, missing := range missingArtifacts {
				if req == missing {
					msg += fmt.Sprintf("  - %s requires %s\n", phase, req)
				}
			}
		}
	}

	msg += "\nSuggested action: Run earlier phases first to generate the required artifacts.\n"
	msg += "\nDo you want to continue anyway? [y/N]: "
	return msg
}

// executePhases executes the selected phases in order
func executePhases(orchestrator *workflow.WorkflowOrchestrator, phaseConfig *workflow.PhaseConfig, featureDescription string, specMetadata *spec.Metadata, resume, debug bool) error {
	phases := phaseConfig.GetCanonicalOrder()
	totalPhases := len(phases)
	orchestrator.Executor.TotalPhases = totalPhases

	var specName string
	if specMetadata != nil {
		specName = fmt.Sprintf("%s-%s", specMetadata.Number, specMetadata.Name)
	}

	for i, phase := range phases {
		fmt.Printf("[Phase %d/%d] %s...\n", i+1, totalPhases, phase)

		switch phase {
		case workflow.PhaseSpecify:
			name, err := orchestrator.ExecuteSpecify(featureDescription)
			if err != nil {
				return fmt.Errorf("specify phase failed: %w", err)
			}
			specName = name
			// Update specMetadata for subsequent phases
			specMetadata = &spec.Metadata{
				Name:      name,
				Directory: filepath.Join(orchestrator.SpecsDir, name),
			}

		case workflow.PhasePlan:
			if err := orchestrator.ExecutePlan(specName, featureDescription); err != nil {
				return fmt.Errorf("plan phase failed: %w", err)
			}

		case workflow.PhaseTasks:
			if err := orchestrator.ExecuteTasks(specName, featureDescription); err != nil {
				return fmt.Errorf("tasks phase failed: %w", err)
			}

		case workflow.PhaseImplement:
			if err := orchestrator.ExecuteImplement(specName, featureDescription, resume); err != nil {
				return fmt.Errorf("implement phase failed: %w", err)
			}
		}
	}

	fmt.Printf("\nCompleted %d phase(s) successfully!\n", totalPhases)
	if specName != "" {
		fmt.Printf("Spec: specs/%s/\n", specName)
	}

	return nil
}

func init() {
	rootCmd.AddCommand(runCmd)

	// Phase selection flags
	runCmd.Flags().BoolP("specify", "s", false, "Include specify phase")
	runCmd.Flags().BoolP("plan", "p", false, "Include plan phase")
	runCmd.Flags().BoolP("tasks", "t", false, "Include tasks phase")
	runCmd.Flags().BoolP("implement", "i", false, "Include implement phase")
	runCmd.Flags().BoolP("all", "a", false, "Run all phases (equivalent to -spti)")

	// Spec selection
	runCmd.Flags().String("spec", "", "Specify which spec to work with (overrides branch detection)")

	// Skip confirmation
	runCmd.Flags().BoolP("yes", "y", false, "Skip confirmation prompts")

	// Other flags
	runCmd.Flags().IntP("max-retries", "r", 0, "Override max retry attempts (0 = use config)")
	runCmd.Flags().Bool("resume", false, "Resume implementation from where it left off")
	runCmd.Flags().Bool("progress", false, "Show progress indicators (spinners) during execution")
}
