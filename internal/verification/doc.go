// Package verification provides verification configuration for autospec.
//
// The verification package implements a tiered verification configuration system
// that allows users to control the depth of automated validation. It supports
// three verification levels (basic, enhanced, full) with individual feature
// toggles that can override level defaults.
//
// # Features
//
//   - Three verification levels: basic, enhanced, full
//   - Individual feature toggles: adversarial_review, contracts, property_tests, metamorphic_tests
//   - Configurable thresholds: mutation_threshold, coverage_threshold, complexity_max
//   - Resolution order: explicit toggle > level preset > default
//
// # Level Presets
//
//   - Basic: No additional verification features (current autospec behavior)
//   - Enhanced: Contracts verification enabled
//   - Full: All verification features enabled
//
// # Usage
//
//	config := verification.VerificationConfig{
//		Level: verification.LevelEnhanced,
//		MutationThreshold: 0.85,
//	}
//	// Check if a feature is enabled (resolves toggle > level > default)
//	if config.IsEnabled("contracts") {
//		// run contracts verification
//	}
package verification
