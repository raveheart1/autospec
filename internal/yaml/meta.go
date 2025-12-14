package yaml

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// metaWrapper is used to extract only the _meta section from a YAML document.
type metaWrapper struct {
	Meta Meta `yaml:"_meta"`
}

// ExtractMeta extracts the _meta section from a YAML document.
// Returns an empty Meta struct if _meta is not present.
func ExtractMeta(r io.Reader) (Meta, error) {
	var wrapper metaWrapper
	dec := yaml.NewDecoder(r)
	if err := dec.Decode(&wrapper); err != nil {
		if err == io.EOF {
			return Meta{}, nil
		}
		return Meta{}, err
	}
	return wrapper.Meta, nil
}

// ExtractMetaFromBytes extracts the _meta section from YAML bytes.
func ExtractMetaFromBytes(data []byte) (Meta, error) {
	var wrapper metaWrapper
	if err := yaml.Unmarshal(data, &wrapper); err != nil {
		return Meta{}, err
	}
	return wrapper.Meta, nil
}

// Version represents a semantic version with major, minor, patch components.
type Version struct {
	Major int
	Minor int
	Patch int
}

// ParseVersion parses a semver string (e.g., "1.2.3") into a Version.
// Returns an error if the format is invalid.
func ParseVersion(s string) (Version, error) {
	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		return Version{}, fmt.Errorf("invalid version format: %s (expected X.Y.Z)", s)
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return Version{}, fmt.Errorf("invalid major version: %s", parts[0])
	}

	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return Version{}, fmt.Errorf("invalid minor version: %s", parts[1])
	}

	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return Version{}, fmt.Errorf("invalid patch version: %s", parts[2])
	}

	return Version{Major: major, Minor: minor, Patch: patch}, nil
}

// Compare compares two versions.
// Returns -1 if v < other, 0 if v == other, 1 if v > other.
func (v Version) Compare(other Version) int {
	if v.Major != other.Major {
		if v.Major < other.Major {
			return -1
		}
		return 1
	}
	if v.Minor != other.Minor {
		if v.Minor < other.Minor {
			return -1
		}
		return 1
	}
	if v.Patch != other.Patch {
		if v.Patch < other.Patch {
			return -1
		}
		return 1
	}
	return 0
}

// String returns the version as a string in X.Y.Z format.
func (v Version) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

// IsMajorVersionMismatch returns true if two version strings have different major versions.
// Returns false if either version cannot be parsed.
func IsMajorVersionMismatch(v1, v2 string) bool {
	ver1, err1 := ParseVersion(v1)
	ver2, err2 := ParseVersion(v2)
	if err1 != nil || err2 != nil {
		return false
	}
	return ver1.Major != ver2.Major
}

// GetArtifactType extracts the artifact_type from a Meta struct.
// Returns empty string if not set.
func GetArtifactType(meta Meta) string {
	return meta.ArtifactType
}

// ValidArtifactTypes lists all valid artifact types.
var ValidArtifactTypes = []string{
	"spec",
	"plan",
	"tasks",
	"checklist",
	"analysis",
	"constitution",
}

// IsValidArtifactType returns true if the artifact type is valid.
func IsValidArtifactType(artifactType string) bool {
	for _, valid := range ValidArtifactTypes {
		if artifactType == valid {
			return true
		}
	}
	return false
}
