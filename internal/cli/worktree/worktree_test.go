package worktree

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWorktreeCmd_Structure(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "worktree", WorktreeCmd.Use)
	assert.NotEmpty(t, WorktreeCmd.Short)
	assert.NotEmpty(t, WorktreeCmd.Long)
}

func TestWorktreeCmd_Subcommands(t *testing.T) {
	// Not parallel: WorktreeCmd.Commands() has lazy init that races with other tests
	subcommands := WorktreeCmd.Commands()
	names := make([]string, len(subcommands))
	for i, cmd := range subcommands {
		names[i] = cmd.Name()
	}

	assert.Contains(t, names, "create")
	assert.Contains(t, names, "list")
	assert.Contains(t, names, "remove")
	assert.Contains(t, names, "setup")
	assert.Contains(t, names, "prune")
}

func TestCreateCmd_Flags(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		name       string
		shorthand  string
		defValue   string
		shouldSkip bool
	}{
		"branch flag exists with shorthand": {
			name:      "branch",
			shorthand: "b",
			defValue:  "",
		},
		"path flag exists with shorthand": {
			name:      "path",
			shorthand: "p",
			defValue:  "",
		},
		"skip-copy flag exists": {
			name:      "skip-copy",
			shorthand: "",
			defValue:  "false",
		},
		"skip-setup flag exists": {
			name:      "skip-setup",
			shorthand: "",
			defValue:  "false",
		},
		"no-rollback flag exists": {
			name:      "no-rollback",
			shorthand: "",
			defValue:  "false",
		},
	}

	for testName, tt := range tests {
		t.Run(testName, func(t *testing.T) {
			t.Parallel()

			flag := createCmd.Flags().Lookup(tt.name)
			assert.NotNil(t, flag, "%s flag should exist", tt.name)
			if tt.shorthand != "" {
				assert.Equal(t, tt.shorthand, flag.Shorthand)
			}
			assert.Equal(t, tt.defValue, flag.DefValue)
		})
	}
}

func TestCreateCmd_FlagDescriptions(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		flagName        string
		usageContains   string
		descriptionNote string
	}{
		"skip-copy describes copy_dirs bypass": {
			flagName:        "skip-copy",
			usageContains:   "copy_dirs",
			descriptionNote: "flag usage should mention copy_dirs config",
		},
		"skip-setup describes setup script bypass": {
			flagName:        "skip-setup",
			usageContains:   "setup script",
			descriptionNote: "flag usage should mention setup script",
		},
		"no-rollback describes debugging and no cleanup": {
			flagName:        "no-rollback",
			usageContains:   "debugging",
			descriptionNote: "flag usage should mention debugging use case",
		},
	}

	for testName, tt := range tests {
		t.Run(testName, func(t *testing.T) {
			t.Parallel()

			flag := createCmd.Flags().Lookup(tt.flagName)
			assert.NotNil(t, flag, "%s flag should exist", tt.flagName)
			assert.Contains(t, flag.Usage, tt.usageContains, tt.descriptionNote)
		})
	}
}

func TestRemoveCmd_Flags(t *testing.T) {
	t.Parallel()

	forceFlag := removeCmd.Flags().Lookup("force")
	assert.NotNil(t, forceFlag, "force flag should exist")
	assert.Equal(t, "f", forceFlag.Shorthand)
}

func TestRemoveCmd_Aliases(t *testing.T) {
	t.Parallel()

	assert.Contains(t, removeCmd.Aliases, "rm")
}

func TestSetupCmd_Flags(t *testing.T) {
	t.Parallel()

	trackFlag := setupCmd.Flags().Lookup("track")
	assert.NotNil(t, trackFlag, "track flag should exist")
}

func TestRelativeTime(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		input string
		want  string
	}{
		// These tests would require time manipulation
		// For now, just test the basic structure exists
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_ = tt // Placeholder
		})
	}
}

func TestPluralize(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		singular string
		count    int
		want     string
	}{
		"entry singular":   {singular: "entry", count: 1, want: "entry"},
		"entry plural":     {singular: "entry", count: 2, want: "entries"},
		"entry zero":       {singular: "entry", count: 0, want: "entries"},
		"default singular": {singular: "item", count: 1, want: "item"},
		"default plural":   {singular: "item", count: 3, want: "items"},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := pluralize(tt.singular, tt.count)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRepeatString(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		input string
		count int
		want  string
	}{
		"dash 3":   {input: "-", count: 3, want: "---"},
		"dash 0":   {input: "-", count: 0, want: ""},
		"ab 2":     {input: "ab", count: 2, want: "abab"},
		"empty 5":  {input: "", count: 5, want: ""},
		"single 1": {input: "x", count: 1, want: "x"},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := repeatString(tt.input, tt.count)
			assert.Equal(t, tt.want, got)
		})
	}
}
