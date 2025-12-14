package cli

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

var (
	// Version information - set via ldflags during build
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Display version information",
	Long:  "Display version, commit, build date, and Go version information for autospec",
	Example: `  # Show version info
  autospec version`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("autospec version %s\n", Version)
		fmt.Printf("Built from commit: %s\n", Commit)
		fmt.Printf("Build date: %s\n", BuildDate)
		fmt.Printf("Go version: %s\n", runtime.Version())
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
