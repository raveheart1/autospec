package validation

import (
	"fmt"
	"regexp"
	"strings"

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

	// Validate risks if present
	risksNode := findNode(rootMapping, "risks")
	if risksNode != nil {
		v.validateRisks(risksNode, result)
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
// Iterates through array elements, validating each phase via validatePhase.
// Path building: "implementation_phases[0]", "implementation_phases[1]", etc.
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
// Required fields: phase (number), name. Optional: deliverables (array).
// Errors include path prefix for precise error location (e.g., "implementation_phases[0].name").
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

// riskIDPattern matches RISK-NNN format (e.g., RISK-001, RISK-999).
var riskIDPattern = regexp.MustCompile(`^RISK-\d{3}$`)

// validateRisks validates the risks section.
func (v *PlanValidator) validateRisks(node *yaml.Node, result *ValidationResult) {
	if !validateFieldType(node, "risks", yaml.SequenceNode, "array", result) {
		return
	}

	for i, riskNode := range node.Content {
		path := fmt.Sprintf("risks[%d]", i)
		v.validateRisk(riskNode, path, result)
	}
}

// validateRisk validates a single risk entry.
func (v *PlanValidator) validateRisk(node *yaml.Node, path string, result *ValidationResult) {
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

	// Validate id field (optional for backward compatibility, but format validated if present)
	idNode := findNode(node, "id")
	if idNode != nil && idNode.Kind == yaml.ScalarNode {
		if !riskIDPattern.MatchString(idNode.Value) {
			result.AddError(&ValidationError{
				Path:     path + ".id",
				Line:     getNodeLine(idNode),
				Message:  fmt.Sprintf("invalid risk ID format: '%s'", idNode.Value),
				Expected: "RISK-NNN (e.g., RISK-001)",
				Actual:   idNode.Value,
				Hint:     "Use format RISK-NNN where NNN is a three-digit number",
			})
		}
	}

	// Validate required field: risk (description)
	riskDescNode := findNode(node, "risk")
	if riskDescNode == nil {
		result.AddError(&ValidationError{
			Path:    path + ".risk",
			Line:    getNodeLine(node),
			Message: "missing required field: risk",
			Hint:    "Add a 'risk' field describing the risk",
		})
	}

	// Validate required field: likelihood (enum)
	likelihoodNode := findNode(node, "likelihood")
	if likelihoodNode == nil {
		result.AddError(&ValidationError{
			Path:    path + ".likelihood",
			Line:    getNodeLine(node),
			Message: "missing required field: likelihood",
			Hint:    "Add a 'likelihood' field with value: low, medium, or high",
		})
	} else {
		validateEnumValue(likelihoodNode, path+".likelihood", []string{"low", "medium", "high"}, result)
	}

	// Validate required field: impact (enum)
	impactNode := findNode(node, "impact")
	if impactNode == nil {
		result.AddError(&ValidationError{
			Path:    path + ".impact",
			Line:    getNodeLine(node),
			Message: "missing required field: impact",
			Hint:    "Add an 'impact' field with value: low, medium, or high",
		})
	} else {
		validateEnumValue(impactNode, path+".impact", []string{"low", "medium", "high"}, result)
	}

	// Check for warning: high-impact risk without mitigation
	v.checkHighImpactMitigation(node, path, idNode, impactNode, result)
}

// checkHighImpactMitigation emits a warning for high-impact risks without mitigation.
func (v *PlanValidator) checkHighImpactMitigation(node *yaml.Node, path string, idNode, impactNode *yaml.Node, result *ValidationResult) {
	if impactNode == nil || impactNode.Kind != yaml.ScalarNode {
		return
	}

	if impactNode.Value != "high" {
		return
	}

	mitigationNode := findNode(node, "mitigation")
	hasMitigation := mitigationNode != nil &&
		mitigationNode.Kind == yaml.ScalarNode &&
		len(strings.TrimSpace(mitigationNode.Value)) > 0

	if !hasMitigation {
		riskID := path
		if idNode != nil && idNode.Kind == yaml.ScalarNode {
			riskID = idNode.Value
		}
		result.AddWarning(&ValidationWarning{
			Path:    path + ".mitigation",
			Line:    getNodeLine(node),
			Message: fmt.Sprintf("high-impact risk %s has no mitigation strategy", riskID),
			Hint:    "Consider adding a mitigation strategy for high-impact risks",
		})
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
