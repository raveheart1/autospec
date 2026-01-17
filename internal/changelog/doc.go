// Package changelog provides YAML-first changelog management for autospec.
//
// This package implements:
//   - CHANGELOG.yaml parsing and validation
//   - Markdown generation following Keep a Changelog format
//   - Version and entry querying for CLI display
//   - Embedded changelog support via go:embed
//
// The CHANGELOG.yaml file at internal/changelog/changelog.yaml serves as the
// single source of truth for all changelog content. CHANGELOG.md is generated
// from this file using the render functionality.
package changelog
