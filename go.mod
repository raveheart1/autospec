// Auto Claude SpecKit - Automated SpecKit workflow validation for Claude Code
//
// Dependency Size Summary (production binary):
//   Runtime dependencies:  ~900 KB (validator, koanf, cobra)
//   Indirect dependencies: ~11 MB (mostly golang.org/x/sys at 9.0M)
//   Total binary size:     ~9.7 MB
//
// Note: testify (400K) is test-only and NOT included in the production binary.
// Go automatically excludes dependencies only used in *_test.go files.
//
// The large binary size is primarily due to golang.org/x/sys (9.0M) which provides
// cross-platform OS primitives needed for file operations, process management, etc.

module github.com/anthropics/auto-claude-speckit

go 1.25.1

require (
	// ============================================================================
	// PRODUCTION DEPENDENCIES (included in binary)
	// ============================================================================

	// Struct field validation with tag-based rules (396K)
	// Used for validating configuration struct fields
	github.com/go-playground/validator/v10 v10.28.0

	// Koanf configuration management library (224K total for all koanf packages)
	// Provides flexible config loading from multiple sources with priority ordering
	github.com/knadh/koanf/parsers/json v1.0.0 // JSON parser for config files
	github.com/knadh/koanf/providers/env v1.1.0 // Environment variable provider
	github.com/knadh/koanf/providers/file v1.2.0 // File-based config provider
	github.com/knadh/koanf/v2 v2.3.0 // Core koanf library

	// CLI framework for building command-line applications (292K)
	// Powers autospec's command structure (init, config, workflow, etc.)
	github.com/spf13/cobra v1.10.1

	// ============================================================================
	// TEST-ONLY DEPENDENCIES (NOT included in binary - only in *_test.go files)
	// ============================================================================

	// Testing toolkit with assertions and mocking (400K source, 0 KB in binary)
	// Used in unit and integration tests - Go excludes this automatically
	github.com/stretchr/testify v1.11.1
)

require (
	// ============================================================================
	// TEST-ONLY INDIRECT DEPENDENCIES (NOT in binary - only used by testify)
	// ============================================================================
	github.com/davecgh/go-spew v1.1.1       // indirect - Deep pretty printer (100K source, 0 KB in binary)
	github.com/pmezard/go-difflib v1.0.0    // indirect - Diff library (36K source, 0 KB in binary)
	github.com/rogpeppe/go-internal v1.14.1 // indirect - Go utilities (252K source, 0 KB in binary)

	// YAML parser - pulled in by testify/assert/yaml even though we don't use YAML assertions.
	// We only use assert.Equal, assert.NoError, etc. but testify bundles all assertion types.
	gopkg.in/yaml.v3 v3.0.1 // indirect - YAML parser (364K source, 0 KB in binary)

	// ============================================================================
	// PRODUCTION INDIRECT DEPENDENCIES (included in binary)
	// ============================================================================

	// Configuration and file system utilities
	github.com/fsnotify/fsnotify v1.9.0        // indirect - Cross-platform file system notifications (232K)
	github.com/go-viper/mapstructure/v2 v2.4.0 // indirect - Decode generic maps into structs (152K)
	github.com/knadh/koanf/maps v0.1.2         // indirect - Map manipulation utilities (included in koanf)

	// Validation support libraries
	github.com/gabriel-vasile/mimetype v1.4.10            // indirect - MIME type detection (236K)
	github.com/go-playground/locales v0.14.1              // indirect - Locale/translation support (84K)
	github.com/go-playground/universal-translator v0.18.1 // indirect - i18n translator (84K)
	github.com/leodido/go-urn v1.4.0                      // indirect - URN parser

	// CLI framework dependencies
	github.com/inconshreveable/mousetrap v1.1.0 // indirect - Windows console handling (28K)
	github.com/spf13/pflag v1.0.9               // indirect - POSIX/GNU-style flags (312K)

	// Reflection and struct utilities
	github.com/mitchellh/copystructure v1.2.0 // indirect - Deep copying of Go structures (32K)
	github.com/mitchellh/reflectwalk v1.0.2   // indirect - Reflection-based struct walking (36K)

	// Go standard library extensions
	golang.org/x/crypto v0.42.0 // indirect - Cryptographic functions (156K)
	golang.org/x/sys v0.36.0    // indirect - Low-level OS primitives (9.0M) ⚠️ LARGEST DEPENDENCY
	golang.org/x/text v0.29.0   // indirect - Text processing/encoding (428K)
	golang.org/x/tools v0.36.0  // indirect - Go tools and packages (32K vendored subset)
)
