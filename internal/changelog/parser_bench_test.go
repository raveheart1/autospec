package changelog

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

// generateLargeChangelog creates a CHANGELOG.yaml string with the specified
// number of entries distributed across multiple versions.
func generateLargeChangelog(entryCount int) string {
	var buf bytes.Buffer

	buf.WriteString("project: benchmark-project\n")
	buf.WriteString("versions:\n")

	// Create versions with ~10 entries each
	entriesPerVersion := 10
	versionCount := (entryCount + entriesPerVersion - 1) / entriesPerVersion

	entriesRemaining := entryCount

	for v := versionCount; v >= 1 && entriesRemaining > 0; v-- {
		buf.WriteString(fmt.Sprintf("  - version: \"%d.0.0\"\n", v))
		buf.WriteString(fmt.Sprintf("    date: \"2024-%02d-%02d\"\n", (v%12)+1, (v%28)+1))
		buf.WriteString("    changes:\n")

		entriesInThisVersion := entriesPerVersion
		if entriesRemaining < entriesPerVersion {
			entriesInThisVersion = entriesRemaining
		}

		writeVersionEntries(&buf, entriesInThisVersion)
		entriesRemaining -= entriesInThisVersion
	}

	return buf.String()
}

// writeVersionEntries distributes entries across categories.
func writeVersionEntries(buf *bytes.Buffer, count int) {
	categories := []string{"added", "changed", "fixed", "removed", "deprecated", "security"}

	// Distribute entries across categories
	perCategory := count / len(categories)
	remainder := count % len(categories)

	for i, cat := range categories {
		entriesForCat := perCategory
		if i < remainder {
			entriesForCat++
		}
		if entriesForCat > 0 {
			buf.WriteString(fmt.Sprintf("      %s:\n", cat))
			for j := 0; j < entriesForCat; j++ {
				buf.WriteString(fmt.Sprintf("        - \"Entry %d for %s category with some description text\"\n", j+1, cat))
			}
		}
	}
}

// BenchmarkLoadFromReader_1000Entries benchmarks parsing a changelog with
// 1000 entries to verify the <10ms performance requirement.
func BenchmarkLoadFromReader_1000Entries(b *testing.B) {
	yamlContent := generateLargeChangelog(1000)
	reader := strings.NewReader(yamlContent)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader.Reset(yamlContent)
		_, err := LoadFromReader(reader)
		if err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
	}
}

// BenchmarkLoadFromReader_100Entries benchmarks a typical changelog size.
func BenchmarkLoadFromReader_100Entries(b *testing.B) {
	yamlContent := generateLargeChangelog(100)
	reader := strings.NewReader(yamlContent)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader.Reset(yamlContent)
		_, err := LoadFromReader(reader)
		if err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
	}
}

// BenchmarkLoadFromReader_10Entries benchmarks a small changelog.
func BenchmarkLoadFromReader_10Entries(b *testing.B) {
	yamlContent := generateLargeChangelog(10)
	reader := strings.NewReader(yamlContent)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader.Reset(yamlContent)
		_, err := LoadFromReader(reader)
		if err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
	}
}

// BenchmarkValidate_1000Entries benchmarks validation only for large changelog.
func BenchmarkValidate_1000Entries(b *testing.B) {
	yamlContent := generateLargeChangelog(1000)
	changelog, err := LoadFromReader(strings.NewReader(yamlContent))
	if err != nil {
		b.Fatalf("failed to load changelog: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := Validate(changelog)
		if err != nil {
			b.Fatalf("unexpected validation error: %v", err)
		}
	}
}

// TestParsing1000EntriesUnder10ms verifies the <10ms performance requirement.
// This is a test (not a benchmark) so it runs with regular tests and fails
// if the requirement is not met.
func TestParsing1000EntriesUnder10ms(t *testing.T) {
	yamlContent := generateLargeChangelog(1000)

	// Parse multiple times to get a stable measurement
	const iterations = 10
	var totalNs int64

	for i := 0; i < iterations; i++ {
		reader := strings.NewReader(yamlContent)
		start := testing.Benchmark(func(b *testing.B) {
			for j := 0; j < b.N; j++ {
				reader.Reset(yamlContent)
				_, _ = LoadFromReader(reader)
			}
		})
		totalNs += start.NsPerOp()
	}

	avgNs := totalNs / iterations
	avgMs := float64(avgNs) / 1e6

	t.Logf("Average parsing time for 1000 entries: %.3f ms", avgMs)

	// Performance requirement: parsing must complete in under 10ms
	const maxMs = 10.0
	if avgMs > maxMs {
		t.Errorf("parsing 1000 entries took %.3f ms, exceeds %.1f ms requirement", avgMs, maxMs)
	}
}
