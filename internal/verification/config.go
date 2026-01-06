// Package verification provides verification configuration for autospec.
package verification

import "fmt"

// VerificationLevel defines the preset verification tier.
// Each level enables a predefined set of verification features.
type VerificationLevel string

// Verification level constants define the available tiers.
const (
	// LevelBasic represents no additional verification (current autospec behavior).
	LevelBasic VerificationLevel = "basic"
	// LevelEnhanced enables contracts verification.
	LevelEnhanced VerificationLevel = "enhanced"
	// LevelFull enables all verification features.
	LevelFull VerificationLevel = "full"
)

// ValidLevels lists all valid verification level values.
var ValidLevels = []VerificationLevel{LevelBasic, LevelEnhanced, LevelFull}

// ParseVerificationLevel parses a string into a VerificationLevel.
// Returns an error if the value is not a valid level.
func ParseVerificationLevel(s string) (VerificationLevel, error) {
	level := VerificationLevel(s)
	for _, valid := range ValidLevels {
		if level == valid {
			return level, nil
		}
	}
	return "", fmt.Errorf("invalid verification level %q: valid options are basic, enhanced, full", s)
}

// IsValid returns true if the level is a known verification level.
func (l VerificationLevel) IsValid() bool {
	for _, valid := range ValidLevels {
		if l == valid {
			return true
		}
	}
	return false
}

// String returns the string representation of the level.
func (l VerificationLevel) String() string {
	return string(l)
}

// VerificationConfig holds verification settings including level, feature toggles, and thresholds.
// Feature toggles use *bool to distinguish between "not set" (nil) and "explicitly false".
type VerificationConfig struct {
	// Level is the verification tier (basic, enhanced, full).
	Level VerificationLevel `koanf:"level" yaml:"level"`

	// Feature toggles - nil means use level default, explicit value overrides level.
	AdversarialReview *bool `koanf:"adversarial_review" yaml:"adversarial_review,omitempty"`
	Contracts         *bool `koanf:"contracts" yaml:"contracts,omitempty"`
	PropertyTests     *bool `koanf:"property_tests" yaml:"property_tests,omitempty"`
	MetamorphicTests  *bool `koanf:"metamorphic_tests" yaml:"metamorphic_tests,omitempty"`
	EarsRequirements  *bool `koanf:"ears_requirements" yaml:"ears_requirements,omitempty"`

	// Thresholds for quality gates.
	MutationThreshold float64 `koanf:"mutation_threshold" yaml:"mutation_threshold"`
	CoverageThreshold float64 `koanf:"coverage_threshold" yaml:"coverage_threshold"`
	ComplexityMax     int     `koanf:"complexity_max" yaml:"complexity_max"`
}

// Feature toggle names used for IsEnabled lookups.
const (
	FeatureAdversarialReview = "adversarial_review"
	FeatureContracts         = "contracts"
	FeaturePropertyTests     = "property_tests"
	FeatureMetamorphicTests  = "metamorphic_tests"
	FeatureEarsRequirements  = "ears_requirements"
)

// levelPreset defines which features are enabled by default for a verification level.
type levelPreset struct {
	AdversarialReview bool
	Contracts         bool
	PropertyTests     bool
	MetamorphicTests  bool
	EarsRequirements  bool
}

// levelPresets maps each verification level to its default feature configuration.
// Basic: all false, Enhanced: contracts + EARS, Full: all features enabled.
var levelPresets = map[VerificationLevel]levelPreset{
	LevelBasic: {
		AdversarialReview: false,
		Contracts:         false,
		PropertyTests:     false,
		MetamorphicTests:  false,
		EarsRequirements:  false,
	},
	LevelEnhanced: {
		AdversarialReview: false,
		Contracts:         true,
		PropertyTests:     false,
		MetamorphicTests:  false,
		EarsRequirements:  true,
	},
	LevelFull: {
		AdversarialReview: true,
		Contracts:         true,
		PropertyTests:     true,
		MetamorphicTests:  true,
		EarsRequirements:  true,
	},
}

// GetLevelDefaults returns the default feature preset for a verification level.
// Returns the basic preset if the level is not recognized.
func GetLevelDefaults(level VerificationLevel) levelPreset {
	preset, ok := levelPresets[level]
	if !ok {
		return levelPresets[LevelBasic]
	}
	return preset
}

// IsEnabled checks if a feature is enabled based on the resolution order:
// explicit toggle > level preset > default (false).
func (c *VerificationConfig) IsEnabled(feature string) bool {
	preset := GetLevelDefaults(c.Level)

	switch feature {
	case FeatureAdversarialReview:
		if c.AdversarialReview != nil {
			return *c.AdversarialReview
		}
		return preset.AdversarialReview
	case FeatureContracts:
		if c.Contracts != nil {
			return *c.Contracts
		}
		return preset.Contracts
	case FeaturePropertyTests:
		if c.PropertyTests != nil {
			return *c.PropertyTests
		}
		return preset.PropertyTests
	case FeatureMetamorphicTests:
		if c.MetamorphicTests != nil {
			return *c.MetamorphicTests
		}
		return preset.MetamorphicTests
	case FeatureEarsRequirements:
		if c.EarsRequirements != nil {
			return *c.EarsRequirements
		}
		return preset.EarsRequirements
	default:
		return false
	}
}

// GetEffectiveToggles returns resolved values for all feature toggles.
// Values are computed using IsEnabled logic: explicit toggle > level preset > default.
func (c *VerificationConfig) GetEffectiveToggles() map[string]bool {
	return map[string]bool{
		FeatureAdversarialReview: c.IsEnabled(FeatureAdversarialReview),
		FeatureContracts:         c.IsEnabled(FeatureContracts),
		FeaturePropertyTests:     c.IsEnabled(FeaturePropertyTests),
		FeatureMetamorphicTests:  c.IsEnabled(FeatureMetamorphicTests),
		FeatureEarsRequirements:  c.IsEnabled(FeatureEarsRequirements),
	}
}
