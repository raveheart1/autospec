// Package version holds the autospec version information.
// This is a separate package to avoid import cycles - it has no dependencies
// and can be safely imported from any package.
package version

var (
	// Version information - set via ldflags during build
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)

// IsDevBuild returns true if running a development build (not a release).
// Used to gate experimental features that aren't ready for production.
func IsDevBuild() bool {
	return Version == "dev"
}
