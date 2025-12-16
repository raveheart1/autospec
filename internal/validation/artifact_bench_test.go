package validation

import (
	"testing"
)

// BenchmarkValidateSpec benchmarks spec.yaml validation.
// Performance contract: <10ms per validation.
func BenchmarkValidateSpec(b *testing.B) {
	validator, err := NewArtifactValidator(ArtifactTypeSpec)
	if err != nil {
		b.Fatalf("failed to create validator: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = validator.Validate("testdata/spec/valid.yaml")
	}
}

// BenchmarkValidatePlan benchmarks plan.yaml validation.
// Performance contract: <10ms per validation.
func BenchmarkValidatePlan(b *testing.B) {
	validator, err := NewArtifactValidator(ArtifactTypePlan)
	if err != nil {
		b.Fatalf("failed to create validator: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = validator.Validate("testdata/plan/valid.yaml")
	}
}

// BenchmarkValidateTasks benchmarks tasks.yaml validation.
// Performance contract: <10ms per validation.
func BenchmarkValidateTasks(b *testing.B) {
	validator, err := NewArtifactValidator(ArtifactTypeTasks)
	if err != nil {
		b.Fatalf("failed to create validator: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = validator.Validate("testdata/tasks/valid.yaml")
	}
}

// BenchmarkValidateSpecWithErrors benchmarks spec validation with errors.
// This should still be fast even when errors are found.
func BenchmarkValidateSpecWithErrors(b *testing.B) {
	validator, err := NewArtifactValidator(ArtifactTypeSpec)
	if err != nil {
		b.Fatalf("failed to create validator: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = validator.Validate("testdata/spec/missing_feature.yaml")
	}
}

// BenchmarkValidateTasksWithCircularDeps benchmarks circular dependency detection.
// This tests the graph traversal performance.
func BenchmarkValidateTasksWithCircularDeps(b *testing.B) {
	validator, err := NewArtifactValidator(ArtifactTypeTasks)
	if err != nil {
		b.Fatalf("failed to create validator: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = validator.Validate("testdata/tasks/invalid_dep_circular.yaml")
	}
}

// BenchmarkGetSchema benchmarks schema retrieval.
// This should be extremely fast as schemas are pre-built.
func BenchmarkGetSchema(b *testing.B) {
	types := []ArtifactType{ArtifactTypeSpec, ArtifactTypePlan, ArtifactTypeTasks}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, t := range types {
			_, _ = GetSchema(t)
		}
	}
}

// BenchmarkParseArtifactType benchmarks artifact type parsing.
func BenchmarkParseArtifactType(b *testing.B) {
	types := []string{"spec", "plan", "tasks"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, t := range types {
			_, _ = ParseArtifactType(t)
		}
	}
}
