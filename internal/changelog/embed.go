package changelog

import (
	"bytes"
	_ "embed"
	"fmt"
)

//go:embed changelog.yaml
var embeddedChangelog []byte

// Embedded returns the raw embedded changelog.yaml content.
// This content is embedded at build time and represents
// the changelog as of that build.
func Embedded() []byte {
	return embeddedChangelog
}

// LoadEmbedded parses and validates the embedded changelog.yaml.
// Returns the parsed Changelog struct or an error if parsing/validation fails.
// This is useful for displaying changelog entries in the CLI without
// requiring network access or file system access.
func LoadEmbedded() (*Changelog, error) {
	if len(embeddedChangelog) == 0 {
		return nil, fmt.Errorf("embedded changelog is empty (binary may have been built without embedded content)")
	}

	return LoadFromReader(bytes.NewReader(embeddedChangelog))
}
