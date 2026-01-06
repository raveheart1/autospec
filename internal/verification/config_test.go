package verification

import (
	"testing"
)

func TestParseVerificationLevel(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		input     string
		want      VerificationLevel
		wantErr   bool
		errSubstr string
	}{
		"basic level": {
			input: "basic",
			want:  LevelBasic,
		},
		"enhanced level": {
			input: "enhanced",
			want:  LevelEnhanced,
		},
		"full level": {
			input: "full",
			want:  LevelFull,
		},
		"invalid level": {
			input:     "invalid",
			wantErr:   true,
			errSubstr: "invalid verification level",
		},
		"empty string": {
			input:     "",
			wantErr:   true,
			errSubstr: "invalid verification level",
		},
		"uppercase is invalid": {
			input:     "BASIC",
			wantErr:   true,
			errSubstr: "invalid verification level",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got, err := ParseVerificationLevel(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseVerificationLevel(%q) expected error, got nil", tt.input)
					return
				}
				if tt.errSubstr != "" && !contains(err.Error(), tt.errSubstr) {
					t.Errorf("ParseVerificationLevel(%q) error = %q, want substring %q", tt.input, err, tt.errSubstr)
				}
				return
			}

			if err != nil {
				t.Errorf("ParseVerificationLevel(%q) unexpected error: %v", tt.input, err)
				return
			}
			if got != tt.want {
				t.Errorf("ParseVerificationLevel(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestVerificationLevel_IsValid(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		level VerificationLevel
		want  bool
	}{
		"basic is valid":     {level: LevelBasic, want: true},
		"enhanced is valid":  {level: LevelEnhanced, want: true},
		"full is valid":      {level: LevelFull, want: true},
		"invalid is invalid": {level: VerificationLevel("invalid"), want: false},
		"empty is invalid":   {level: VerificationLevel(""), want: false},
		"uppercase invalid":  {level: VerificationLevel("BASIC"), want: false},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if got := tt.level.IsValid(); got != tt.want {
				t.Errorf("VerificationLevel(%q).IsValid() = %v, want %v", tt.level, got, tt.want)
			}
		})
	}
}

func TestGetLevelDefaults(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		level         VerificationLevel
		wantAdverse   bool
		wantContracts bool
		wantProperty  bool
		wantMeta      bool
	}{
		"basic level - all false": {
			level:         LevelBasic,
			wantAdverse:   false,
			wantContracts: false,
			wantProperty:  false,
			wantMeta:      false,
		},
		"enhanced level - contracts only": {
			level:         LevelEnhanced,
			wantAdverse:   false,
			wantContracts: true,
			wantProperty:  false,
			wantMeta:      false,
		},
		"full level - all true": {
			level:         LevelFull,
			wantAdverse:   true,
			wantContracts: true,
			wantProperty:  true,
			wantMeta:      true,
		},
		"unknown level falls back to basic": {
			level:         VerificationLevel("unknown"),
			wantAdverse:   false,
			wantContracts: false,
			wantProperty:  false,
			wantMeta:      false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			preset := GetLevelDefaults(tt.level)

			if preset.AdversarialReview != tt.wantAdverse {
				t.Errorf("GetLevelDefaults(%q).AdversarialReview = %v, want %v", tt.level, preset.AdversarialReview, tt.wantAdverse)
			}
			if preset.Contracts != tt.wantContracts {
				t.Errorf("GetLevelDefaults(%q).Contracts = %v, want %v", tt.level, preset.Contracts, tt.wantContracts)
			}
			if preset.PropertyTests != tt.wantProperty {
				t.Errorf("GetLevelDefaults(%q).PropertyTests = %v, want %v", tt.level, preset.PropertyTests, tt.wantProperty)
			}
			if preset.MetamorphicTests != tt.wantMeta {
				t.Errorf("GetLevelDefaults(%q).MetamorphicTests = %v, want %v", tt.level, preset.MetamorphicTests, tt.wantMeta)
			}
		})
	}
}

func TestVerificationConfig_IsEnabled(t *testing.T) {
	t.Parallel()

	boolPtr := func(b bool) *bool { return &b }

	tests := map[string]struct {
		config  VerificationConfig
		feature string
		want    bool
	}{
		"basic level - adversarial disabled by default": {
			config:  VerificationConfig{Level: LevelBasic},
			feature: FeatureAdversarialReview,
			want:    false,
		},
		"enhanced level - contracts enabled by default": {
			config:  VerificationConfig{Level: LevelEnhanced},
			feature: FeatureContracts,
			want:    true,
		},
		"full level - all features enabled": {
			config:  VerificationConfig{Level: LevelFull},
			feature: FeaturePropertyTests,
			want:    true,
		},
		"explicit true overrides basic level": {
			config:  VerificationConfig{Level: LevelBasic, PropertyTests: boolPtr(true)},
			feature: FeaturePropertyTests,
			want:    true,
		},
		"explicit false overrides enhanced level": {
			config:  VerificationConfig{Level: LevelEnhanced, Contracts: boolPtr(false)},
			feature: FeatureContracts,
			want:    false,
		},
		"explicit false overrides full level": {
			config:  VerificationConfig{Level: LevelFull, AdversarialReview: boolPtr(false)},
			feature: FeatureAdversarialReview,
			want:    false,
		},
		"nil toggle uses level default": {
			config:  VerificationConfig{Level: LevelFull, Contracts: nil},
			feature: FeatureContracts,
			want:    true,
		},
		"unknown feature returns false": {
			config:  VerificationConfig{Level: LevelFull},
			feature: "unknown_feature",
			want:    false,
		},
		"empty level defaults to basic behavior": {
			config:  VerificationConfig{Level: ""},
			feature: FeatureContracts,
			want:    false,
		},
		"metamorphic tests follow level preset": {
			config:  VerificationConfig{Level: LevelFull},
			feature: FeatureMetamorphicTests,
			want:    true,
		},
		"metamorphic tests can be explicitly disabled": {
			config:  VerificationConfig{Level: LevelFull, MetamorphicTests: boolPtr(false)},
			feature: FeatureMetamorphicTests,
			want:    false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if got := tt.config.IsEnabled(tt.feature); got != tt.want {
				t.Errorf("VerificationConfig.IsEnabled(%q) = %v, want %v", tt.feature, got, tt.want)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
