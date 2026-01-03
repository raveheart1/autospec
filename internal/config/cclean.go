package config

// CcleanConfig holds configuration options for cclean (claude-clean) output formatting.
// These options control how the stream-json output from Claude is formatted for display.
//
// Configuration priority (highest to lowest):
//  1. Environment variables (AUTOSPEC_CCLEAN_*)
//  2. Project config (.autospec/config.yml)
//  3. User config (~/.config/autospec/config.yml)
//  4. Built-in defaults
//
// Example YAML configuration:
//
//	cclean:
//	  verbose: true       # Enable verbose output with usage stats and tool IDs
//	  line_numbers: true  # Show line numbers in formatted output
//	  style: compact      # Output style: default, compact, minimal, plain
type CcleanConfig struct {
	// Verbose enables verbose output with usage stats and tool IDs.
	// Equivalent to cclean -V flag.
	// Default: false
	// Environment variable: AUTOSPEC_CCLEAN_VERBOSE
	Verbose bool `koanf:"verbose"`

	// LineNumbers enables line number display in formatted output.
	// Equivalent to cclean -n flag.
	// Default: false
	// Environment variable: AUTOSPEC_CCLEAN_LINE_NUMBERS
	LineNumbers bool `koanf:"line_numbers"`

	// Style controls the output formatting style.
	// Valid values: default, compact, minimal, plain
	// Equivalent to cclean -s flag.
	// Default: "default" (box-drawing characters with colors)
	// Environment variable: AUTOSPEC_CCLEAN_STYLE
	Style string `koanf:"style"`
}
