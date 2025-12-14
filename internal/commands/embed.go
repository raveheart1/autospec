package commands

import (
	"embed"
	"path/filepath"
	"strings"
)

// TemplateFS embeds all command templates from the commands/ directory.
// The templates are stored at the repository root in the commands/ directory.
//
//go:embed all:*.md
var TemplateFS embed.FS

// GetTemplateNames returns a list of all embedded template names (without extension).
func GetTemplateNames() ([]string, error) {
	entries, err := TemplateFS.ReadDir(".")
	if err != nil {
		return nil, err
	}

	var names []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".md") {
			// Remove .md extension
			names = append(names, strings.TrimSuffix(name, ".md"))
		}
	}
	return names, nil
}

// GetTemplate retrieves a template by name (without extension).
func GetTemplate(name string) ([]byte, error) {
	filename := name + ".md"
	return TemplateFS.ReadFile(filename)
}

// GetTemplateByFilename retrieves a template by its full filename.
func GetTemplateByFilename(filename string) ([]byte, error) {
	return TemplateFS.ReadFile(filename)
}

// IsAutospecTemplate returns true if the filename is an autospec template.
func IsAutospecTemplate(filename string) bool {
	base := filepath.Base(filename)
	return strings.HasPrefix(base, "autospec.")
}
