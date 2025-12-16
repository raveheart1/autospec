package validation

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// PlanValidator validates plan.yaml artifacts.
type PlanValidator struct {
	baseValidator
}

// Type returns the artifact type.
func (v *PlanValidator) Type() ArtifactType {
	return ArtifactTypePlan
}

// Validate validates a plan.yaml file at the given path.
func (v *PlanValidator) Validate(path string) *ValidationResult {
	result := &ValidationResult{Valid: true}

	// Parse the YAML file
	root, err := parseYAMLFile(path)
	if err != nil {
		result.AddError(&ValidationError{
			Path:    path,
			Message: fmt.Sprintf("failed to parse YAML: %v", err),
			Hint:    "Check the YAML syntax for errors",
		})
		return result
	}

	rootMapping := getRootMapping(root)
	if rootMapping == nil {
		result.AddError(&ValidationError{
			Path:    path,
			Message: "expected a YAML mapping at document root",
			Hint:    "The plan.yaml file should start with key-value pairs, not a list or scalar",
		})
		return result
	}

	// Validate required fields
	planNode := validateRequiredField(rootMapping, "plan", result)
	summaryNode := validateRequiredField(rootMapping, "summary", result)
	techContextNode := validateRequiredField(rootMapping, "technical_context", result)

	// Validate plan section
	if planNode != nil {
		v.validatePlanSection(planNode, result)
	}

	// Validate summary is a string
	if summaryNode != nil {
		if summaryNode.Kind != yaml.ScalarNode {
			result.AddError(&ValidationError{
				Path:     "summary",
				Line:     getNodeLine(summaryNode),
				Message:  "wrong type for field 'summary'",
				Expected: "string",
				Actual:   nodeKindToString(summaryNode.Kind),
			})
		}
	}

	// Validate technical_context section
	if techContextNode != nil {
		v.validateTechnicalContext(techContextNode, result)
	}

	// Validate implementation_phases if present
	phasesNode := findNode(rootMapping, "implementation_phases")
	if phasesNode != nil {
		v.validateImplementationPhases(phasesNode, result)
	}

	// Build summary if valid
	if result.Valid {
		result.Summary = v.buildSummary(rootMapping)
	}

	return result
}

// validatePlanSection validates the plan section.
func (v *PlanValidator) validatePlanSection(node *yaml.Node, result *ValidationResult) {
	if !validateFieldType(node, "plan", yaml.MappingNode, "object", result) {
		return
	}

	// Required fields in plan
	validateRequiredField(node, "branch", result)
	validateRequiredField(node, "spec_path", result)
}

// validateTechnicalContext validates the technical_context section.
func (v *PlanValidator) validateTechnicalContext(node *yaml.Node, result *ValidationResult) {
	if !validateFieldType(node, "technical_context", yaml.MappingNode, "object", result) {
		return
	}

	// Technical context fields are mostly optional, just validate types if present
	// primary_dependencies should be an array if present
	depsNode := findNode(node, "primary_dependencies")
	if depsNode != nil {
		validateFieldType(depsNode, "technical_context.primary_dependencies", yaml.SequenceNode, "array", result)
	}

	// constraints should be an array if present
	constraintsNode := findNode(node, "constraints")
	if constraintsNode != nil {
		validateFieldType(constraintsNode, "technical_context.constraints", yaml.SequenceNode, "array", result)
	}
}

// validateImplementationPhases validates the implementation_phases section.
func (v *PlanValidator) validateImplementationPhases(node *yaml.Node, result *ValidationResult) {
	if !validateFieldType(node, "implementation_phases", yaml.SequenceNode, "array", result) {
		return
	}

	for i, phaseNode := range node.Content {
		path := fmt.Sprintf("implementation_phases[%d]", i)
		v.validatePhase(phaseNode, path, result)
	}
}

// validatePhase validates a single implementation phase.
func (v *PlanValidator) validatePhase(node *yaml.Node, path string, result *ValidationResult) {
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

	// Required fields: phase (number), name
	phaseNumNode := findNode(node, "phase")
	if phaseNumNode == nil {
		result.AddError(&ValidationError{
			Path:    path + ".phase",
			Line:    getNodeLine(node),
			Message: "missing required field: phase",
			Hint:    "Add a 'phase' field with the phase number",
		})
	}

	nameNode := findNode(node, "name")
	if nameNode == nil {
		result.AddError(&ValidationError{
			Path:    path + ".name",
			Line:    getNodeLine(node),
			Message: "missing required field: name",
			Hint:    "Add a 'name' field with the phase name",
		})
	}

	// deliverables should be an array if present
	deliverablesNode := findNode(node, "deliverables")
	if deliverablesNode != nil {
		validateFieldType(deliverablesNode, path+".deliverables", yaml.SequenceNode, "array", result)
	}
}

// buildSummary builds the summary for a valid plan artifact.
func (v *PlanValidator) buildSummary(root *yaml.Node) *ArtifactSummary {
	summary := &ArtifactSummary{
		Type:   ArtifactTypePlan,
		Counts: make(map[string]int),
	}

	// Count implementation phases
	phasesNode := findNode(root, "implementation_phases")
	if phasesNode != nil && phasesNode.Kind == yaml.SequenceNode {
		summary.Counts["implementation_phases"] = len(phasesNode.Content)
	}

	// Count risks
	risksNode := findNode(root, "risks")
	if risksNode != nil && risksNode.Kind == yaml.SequenceNode {
		summary.Counts["risks"] = len(risksNode.Content)
	}

	// Count open questions
	questionsNode := findNode(root, "open_questions")
	if questionsNode != nil && questionsNode.Kind == yaml.SequenceNode {
		summary.Counts["open_questions"] = len(questionsNode.Content)
	}

	// Count data model entities
	dataModelNode := findNode(root, "data_model")
	if dataModelNode != nil {
		entitiesNode := findNode(dataModelNode, "entities")
		if entitiesNode != nil && entitiesNode.Kind == yaml.SequenceNode {
			summary.Counts["data_model_entities"] = len(entitiesNode.Content)
		}
	}

	return summary
}
