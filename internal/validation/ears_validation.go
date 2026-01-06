// Package validation provides EARS (Easy Approach to Requirements Syntax) validation.
// Related: internal/validation/schema.go, internal/validation/artifact_spec.go
// Tags: validation, ears, requirements, pattern-validation
package validation

import (
	"fmt"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// EarsPattern defines the valid EARS pattern types.
type EarsPattern string

const (
	// PatternUbiquitous: "The [system] shall [action]"
	PatternUbiquitous EarsPattern = "ubiquitous"
	// PatternEventDriven: "When [trigger], the [system] shall [action]"
	PatternEventDriven EarsPattern = "event-driven"
	// PatternStateDriven: "While [state], the [system] shall [action]"
	PatternStateDriven EarsPattern = "state-driven"
	// PatternUnwanted: "If [condition], then the [system] shall [action]"
	PatternUnwanted EarsPattern = "unwanted"
	// PatternOptional: "Where [feature], the [system] shall [action]"
	PatternOptional EarsPattern = "optional"
)

// ValidPatterns lists all valid EARS pattern values.
var ValidPatterns = []EarsPattern{
	PatternUbiquitous,
	PatternEventDriven,
	PatternStateDriven,
	PatternUnwanted,
	PatternOptional,
}

// EarsTestType defines the valid test types for EARS requirements.
type EarsTestType string

const (
	TestTypeInvariant    EarsTestType = "invariant"
	TestTypeProperty     EarsTestType = "property"
	TestTypeStateMachine EarsTestType = "state-machine"
	TestTypeException    EarsTestType = "exception"
	TestTypeFeatureFlag  EarsTestType = "feature-flag"
)

// ValidTestTypes lists all valid EARS test type values.
var ValidTestTypes = []EarsTestType{
	TestTypeInvariant,
	TestTypeProperty,
	TestTypeStateMachine,
	TestTypeException,
	TestTypeFeatureFlag,
}

// PatternTemplates maps EARS patterns to their sentence templates.
var PatternTemplates = map[EarsPattern]string{
	PatternUbiquitous:  "The [system] shall [action]",
	PatternEventDriven: "When [trigger], the [system] shall [action]",
	PatternStateDriven: "While [state], the [system] shall [action]",
	PatternUnwanted:    "If [condition], then the [system] shall [action]",
	PatternOptional:    "Where [feature], the [system] shall [action]",
}

// PatternRequiredFields maps EARS patterns to their required fields.
var PatternRequiredFields = map[EarsPattern][]string{
	PatternUbiquitous:  {}, // No additional required fields
	PatternEventDriven: {"trigger", "expected"},
	PatternStateDriven: {"state"},
	PatternUnwanted:    {"condition"},
	PatternOptional:    {"feature"},
}

// earsIDPattern validates EARS requirement IDs (EARS-NNN format).
var earsIDPattern = regexp.MustCompile(`^EARS-\d+$`)

// EarsRequirement represents a single EARS requirement.
type EarsRequirement struct {
	ID        string       `yaml:"id"`
	Pattern   EarsPattern  `yaml:"pattern"`
	Text      string       `yaml:"text"`
	TestType  EarsTestType `yaml:"test_type"`
	Trigger   string       `yaml:"trigger,omitempty"`
	Expected  string       `yaml:"expected,omitempty"`
	State     string       `yaml:"state,omitempty"`
	Condition string       `yaml:"condition,omitempty"`
	Feature   string       `yaml:"feature,omitempty"`
}

// ValidateEarsRequirements validates EARS requirements in a spec.yaml file.
// Returns validation errors if EARS requirements are malformed.
// This function is called during spec validation when ears_requirements is present.
func ValidateEarsRequirements(earsNode *yaml.Node, result *ValidationResult) {
	if earsNode == nil || earsNode.Kind != yaml.SequenceNode {
		return
	}

	// Empty array is valid
	if len(earsNode.Content) == 0 {
		return
	}

	seenIDs := make(map[string]bool)

	for i, reqNode := range earsNode.Content {
		path := fmt.Sprintf("ears_requirements[%d]", i)
		validateSingleEarsRequirement(reqNode, path, seenIDs, result)
	}
}

// validateSingleEarsRequirement validates a single EARS requirement.
func validateSingleEarsRequirement(node *yaml.Node, path string, seenIDs map[string]bool, result *ValidationResult) {
	if node.Kind != yaml.MappingNode {
		result.AddError(&ValidationError{
			Path:     path,
			Line:     getNodeLine(node),
			Message:  fmt.Sprintf("wrong type for '%s'", path),
			Expected: "object",
			Actual:   nodeKindToString(node.Kind),
		})
		return
	}

	// Validate required fields: id, pattern, text, test_type
	idNode := validateRequiredField(node, "id", result)
	patternNode := validateRequiredField(node, "pattern", result)
	validateRequiredField(node, "text", result)
	testTypeNode := validateRequiredField(node, "test_type", result)

	// Validate ID format and uniqueness
	if idNode != nil && idNode.Value != "" {
		validateEarsID(idNode, path, seenIDs, result)
	}

	// Validate pattern enum and pattern-specific fields
	if patternNode != nil && patternNode.Value != "" {
		validateEarsPattern(node, patternNode.Value, path, result)
	}

	// Validate test_type enum
	if testTypeNode != nil && testTypeNode.Value != "" {
		validateEarsTestType(testTypeNode, path, result)
	}
}

// validateEarsID validates the EARS requirement ID format and uniqueness.
func validateEarsID(idNode *yaml.Node, path string, seenIDs map[string]bool, result *ValidationResult) {
	id := idNode.Value
	if !earsIDPattern.MatchString(id) {
		result.AddError(&ValidationError{
			Path:    path + ".id",
			Line:    getNodeLine(idNode),
			Message: fmt.Sprintf("invalid EARS ID format: %q", id),
			Hint:    "EARS IDs must match pattern EARS-NNN (e.g., EARS-001, EARS-042)",
		})
	}

	if seenIDs[id] {
		result.AddError(&ValidationError{
			Path:    path + ".id",
			Line:    getNodeLine(idNode),
			Message: fmt.Sprintf("duplicate EARS ID: %q", id),
			Hint:    "Each EARS requirement must have a unique ID",
		})
	}
	seenIDs[id] = true
}

// validateEarsPattern validates the EARS pattern enum and pattern-specific required fields.
func validateEarsPattern(node *yaml.Node, patternStr string, path string, result *ValidationResult) {
	pattern := EarsPattern(patternStr)

	// Validate pattern is a known value
	validPattern := false
	for _, p := range ValidPatterns {
		if p == pattern {
			validPattern = true
			break
		}
	}

	if !validPattern {
		result.AddError(&ValidationError{
			Path:    path + ".pattern",
			Message: fmt.Sprintf("invalid EARS pattern: %q", patternStr),
			Hint:    fmt.Sprintf("Valid patterns: %s", strings.Join(patternStrings(), ", ")),
		})
		return
	}

	// Validate pattern-specific required fields
	requiredFields := PatternRequiredFields[pattern]
	for _, field := range requiredFields {
		fieldNode := findNode(node, field)
		if fieldNode == nil || fieldNode.Value == "" {
			result.AddError(&ValidationError{
				Path:    path + "." + field,
				Message: fmt.Sprintf("missing required field '%s' for %s pattern", field, pattern),
				Hint:    fmt.Sprintf("Template: %s", PatternTemplates[pattern]),
			})
		}
	}
}

// validateEarsTestType validates the test_type enum value.
func validateEarsTestType(testTypeNode *yaml.Node, path string, result *ValidationResult) {
	testType := EarsTestType(testTypeNode.Value)

	valid := false
	for _, t := range ValidTestTypes {
		if t == testType {
			valid = true
			break
		}
	}

	if !valid {
		result.AddError(&ValidationError{
			Path:    path + ".test_type",
			Line:    getNodeLine(testTypeNode),
			Message: fmt.Sprintf("invalid test_type: %q", testTypeNode.Value),
			Hint:    fmt.Sprintf("Valid test types: %s", strings.Join(testTypeStrings(), ", ")),
		})
	}
}

// ValidateCrossSectionIDs validates that EARS IDs don't conflict with functional requirement IDs.
// This ensures global uniqueness of requirement identifiers across both sections.
func ValidateCrossSectionIDs(earsNode *yaml.Node, requirementsNode *yaml.Node, result *ValidationResult) {
	if earsNode == nil || requirementsNode == nil {
		return
	}

	// Collect functional requirement IDs
	functionalIDs := collectFunctionalRequirementIDs(requirementsNode)

	// Check EARS IDs against functional IDs
	for i, reqNode := range earsNode.Content {
		if reqNode.Kind != yaml.MappingNode {
			continue
		}
		idNode := findNode(reqNode, "id")
		if idNode == nil {
			continue
		}
		id := idNode.Value
		if functionalIDs[id] {
			result.AddError(&ValidationError{
				Path:    fmt.Sprintf("ears_requirements[%d].id", i),
				Line:    getNodeLine(idNode),
				Message: fmt.Sprintf("EARS ID %q conflicts with functional requirement ID", id),
				Hint:    "Requirement IDs must be unique across both functional requirements and EARS requirements",
			})
		}
	}
}

// collectFunctionalRequirementIDs extracts IDs from functional requirements.
func collectFunctionalRequirementIDs(requirementsNode *yaml.Node) map[string]bool {
	ids := make(map[string]bool)
	if requirementsNode.Kind != yaml.MappingNode {
		return ids
	}

	functionalNode := findNode(requirementsNode, "functional")
	if functionalNode == nil || functionalNode.Kind != yaml.SequenceNode {
		return ids
	}

	for _, reqNode := range functionalNode.Content {
		if reqNode.Kind != yaml.MappingNode {
			continue
		}
		idNode := findNode(reqNode, "id")
		if idNode != nil && idNode.Value != "" {
			ids[idNode.Value] = true
		}
	}

	return ids
}

// patternStrings returns pattern values as strings for error messages.
func patternStrings() []string {
	result := make([]string, len(ValidPatterns))
	for i, p := range ValidPatterns {
		result[i] = string(p)
	}
	return result
}

// testTypeStrings returns test type values as strings for error messages.
func testTypeStrings() []string {
	result := make([]string, len(ValidTestTypes))
	for i, t := range ValidTestTypes {
		result[i] = string(t)
	}
	return result
}
