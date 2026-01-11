package dag

import (
	"fmt"
	"strings"
)

// ValidationError represents a validation error with source location information.
type ValidationError struct {
	Line    int
	Column  int
	Message string
}

// Error implements the error interface.
func (e *ValidationError) Error() string {
	if e.Line > 0 {
		return fmt.Sprintf("line %d, column %d: %s", e.Line, e.Column, e.Message)
	}
	return e.Message
}

// CycleError represents a cycle detected in feature dependencies.
type CycleError struct {
	// Path is the list of feature IDs forming the cycle.
	Path []string
}

// Error implements the error interface.
func (e *CycleError) Error() string {
	if len(e.Path) == 0 {
		return "cycle detected in feature dependencies"
	}
	return fmt.Sprintf("cycle detected in feature dependencies: %s", strings.Join(e.Path, " -> "))
}

// MissingSpecError represents a reference to a spec folder that doesn't exist.
type MissingSpecError struct {
	// FeatureID is the ID of the feature referencing the missing spec.
	FeatureID string
	// ExpectedPath is the expected path to the spec folder.
	ExpectedPath string
	// Line is the source line number where the feature is defined.
	Line int
	// Column is the source column number where the feature is defined.
	Column int
}

// Error implements the error interface.
func (e *MissingSpecError) Error() string {
	return fmt.Sprintf("line %d: missing spec folder for feature %q: expected %s",
		e.Line, e.FeatureID, e.ExpectedPath)
}

// InvalidLayerRefError represents a reference to a non-existent layer.
type InvalidLayerRefError struct {
	// LayerID is the ID of the layer containing the invalid reference.
	LayerID string
	// InvalidRef is the referenced layer ID that doesn't exist.
	InvalidRef string
	// ValidLayerIDs is the list of valid layer IDs for guidance.
	ValidLayerIDs []string
	// Line is the source line number where the reference appears.
	Line int
	// Column is the source column number where the reference appears.
	Column int
}

// Error implements the error interface.
func (e *InvalidLayerRefError) Error() string {
	validList := strings.Join(e.ValidLayerIDs, ", ")
	return fmt.Sprintf("line %d: layer %q depends on non-existent layer %q; valid layers: [%s]",
		e.Line, e.LayerID, e.InvalidRef, validList)
}

// DuplicateFeatureError represents a duplicate feature ID across layers.
type DuplicateFeatureError struct {
	// FeatureID is the duplicated feature ID.
	FeatureID string
	// FirstLocation describes where the feature was first defined.
	FirstLocation string
	// SecondLocation describes where the duplicate was found.
	SecondLocation string
	// Line is the source line number of the duplicate.
	Line int
	// Column is the source column number of the duplicate.
	Column int
}

// Error implements the error interface.
func (e *DuplicateFeatureError) Error() string {
	return fmt.Sprintf("line %d: duplicate feature ID %q (first defined in %s, duplicate in %s)",
		e.Line, e.FeatureID, e.FirstLocation, e.SecondLocation)
}

// InvalidFeatureRefError represents a reference to a non-existent feature.
type InvalidFeatureRefError struct {
	// FeatureID is the ID of the feature containing the invalid reference.
	FeatureID string
	// InvalidRef is the referenced feature ID that doesn't exist.
	InvalidRef string
	// Line is the source line number where the reference appears.
	Line int
	// Column is the source column number where the reference appears.
	Column int
}

// Error implements the error interface.
func (e *InvalidFeatureRefError) Error() string {
	return fmt.Sprintf("line %d: feature %q depends on non-existent feature %q",
		e.Line, e.FeatureID, e.InvalidRef)
}

// MissingFieldError represents a required field that is missing.
type MissingFieldError struct {
	// Field is the name of the missing field.
	Field string
	// Context describes where the field is expected.
	Context string
	// Line is the source line number where the error applies.
	Line int
	// Column is the source column number where the error applies.
	Column int
}

// Error implements the error interface.
func (e *MissingFieldError) Error() string {
	if e.Line > 0 {
		return fmt.Sprintf("line %d: missing required field %q in %s", e.Line, e.Field, e.Context)
	}
	return fmt.Sprintf("missing required field %q in %s", e.Field, e.Context)
}

// DuplicateDAGIDError represents duplicate resolved IDs across DAG files.
type DuplicateDAGIDError struct {
	// ResolvedID is the duplicate resolved identifier.
	ResolvedID string
	// FirstFile is the path to the first DAG file with this ID.
	FirstFile string
	// SecondFile is the path to the second DAG file with this ID.
	SecondFile string
}

// Error implements the error interface.
func (e *DuplicateDAGIDError) Error() string {
	return fmt.Sprintf("duplicate resolved DAG ID %q: first in %s, also in %s",
		e.ResolvedID, e.FirstFile, e.SecondFile)
}

// DuplicateDAGNameWarning represents duplicate dag.name values (warning, not error).
type DuplicateDAGNameWarning struct {
	// Name is the duplicate DAG name.
	Name string
	// FirstFile is the path to the first DAG file with this name.
	FirstFile string
	// SecondFile is the path to the second DAG file with this name.
	SecondFile string
}

// Warning returns the warning message.
func (w *DuplicateDAGNameWarning) Warning() string {
	return fmt.Sprintf("duplicate DAG name %q: first in %s, also in %s (IDs may differ)",
		w.Name, w.FirstFile, w.SecondFile)
}
