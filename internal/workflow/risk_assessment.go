// Package workflow provides risk assessment instruction generation for agent prompt injection.
package workflow

// riskAssessmentInstructions contains the guidance for agents to document risks in plan.yaml.
// This content is conditionally injected when EnableRiskAssessment config is true.
const riskAssessmentInstructions = `## Risk Assessment

Include a risks section in plan.yaml to document potential implementation risks:

` + "```yaml" + `
risks:
  - id: "RISK-001"           # Optional, format: RISK-NNN
    risk: "Description of the potential risk"
    likelihood: "medium"     # low | medium | high
    impact: "high"           # low | medium | high
    mitigation: "Strategy to address or reduce the risk"
` + "```" + `

Consider risks in these categories:
- **Technical**: Dependencies, performance, scalability, security
- **Integration**: Third-party APIs, data migration, system compatibility
- **Operational**: Deployment, monitoring, maintenance complexity
- **Schedule**: Complexity underestimation, external blockers

For trivial features with no significant risks, use an empty array: ` + "`risks: []`" + `
`

// BuildRiskAssessmentInstructions returns an InjectableInstruction for risk assessment.
// The instruction provides guidance for documenting implementation risks in plan.yaml.
// This is used with InjectInstructions for proper marker wrapping.
func BuildRiskAssessmentInstructions() InjectableInstruction {
	return InjectableInstruction{
		Name:        "RiskAssessment",
		DisplayHint: "document implementation risks in plan.yaml",
		Content:     riskAssessmentInstructions,
	}
}

// InjectRiskAssessment conditionally appends risk assessment instructions to a command.
// If enabled is false, returns the original command unchanged.
// If enabled is true, appends the risk assessment instructions with proper markers.
func InjectRiskAssessment(command string, enabled bool) string {
	if !enabled {
		return command
	}
	instruction := BuildRiskAssessmentInstructions()
	return InjectInstructions(command, []InjectableInstruction{instruction})
}
