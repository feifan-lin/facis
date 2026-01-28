package parser

import (
	"testing"

	"digital-contracting-service/internal/semantic/loader"
)

func TestLocalName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "prefixed name",
			input:    "dcs:startDate",
			expected: "startDate",
		},
		{
			name:     "IRI with fragment",
			input:    "<https://example.org/vocab#startDate>",
			expected: "startDate",
		},
		{
			name:     "IRI with path",
			input:    "<https://example.org/vocab/startDate>",
			expected: "startDate",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "whitespace",
			input:    "  dcs:startDate  ",
			expected: "startDate",
		},
		{
			name:     "invalid format",
			input:    "invalid",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := LocalName(tt.input)
			if result != tt.expected {
				t.Errorf("LocalName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestExtractPrefixIRI(t *testing.T) {
	ttl := `@prefix dcs: <https://projects.eclipse.org/xfsc/facis/dcs#> .
dcs:test a rdfs:Class .`

	iri := extractPrefixIRI([]byte(ttl), "dcs")
	if iri != "https://projects.eclipse.org/xfsc/facis/dcs#" {
		t.Errorf("expected IRI %q, got %q", "https://projects.eclipse.org/xfsc/facis/dcs#", iri)
	}

	iri = extractPrefixIRI([]byte(ttl), "nonexistent")
	if iri != "" {
		t.Errorf("expected empty IRI for nonexistent prefix, got %q", iri)
	}
}

func TestQueryVocabByClass(t *testing.T) {
	vocabResult := loader.LoadTTL("internal/semantic/validate/testdata/condition-definition/vocab.ttl")
	if vocabResult.Error != nil {
		t.Fatalf("failed to load vocab.ttl: %v", vocabResult.Error)
	}

	properties, err := QueryVocabByClass(vocabResult.Content, "dcs", "ConditionProperty")
	if err != nil {
		t.Fatalf("QueryVocabByClass failed: %v", err)
	}

	// Check expected properties exist
	expectedProperties := []string{"startDate", "endDate", "rentAmount", "currency", "dueDayOfMonth", "region"}
	for _, expected := range expectedProperties {
		if _, ok := properties[expected]; !ok {
			t.Errorf("expected property %q not found in results. Got: %v", expected, properties)
		}
	}
}

func TestBuildConditionShapesFromSHACL(t *testing.T) {
	// Load SHACL shapes TTL used for condition values.
	shapesResult := loader.LoadTTL("internal/semantic/validate/testdata/condition-values/shapes.ttl")
	if shapesResult.Error != nil {
		t.Fatalf("failed to load shapes.ttl: %v", shapesResult.Error)
	}

	shapes, err := BuildConditionShapesFromSHACL(shapesResult.Content, "dcs")
	if err != nil {
		t.Fatalf("BuildConditionShapesFromSHACL failed: %v", err)
	}

	// Helper to compare expected keys with actual keys in a map[string]struct{}.
	checkKeys := func(kind, cond string, actual map[string]struct{}, expected []string) {
		if len(actual) != len(expected) {
			t.Fatalf("%s keys for %s length mismatch: got %d, want %d (got=%v)",
				kind, cond, len(actual), len(expected), actual)
		}
		for _, k := range expected {
			if _, ok := actual[k]; !ok {
				t.Errorf("%s keys for %s missing expected key %q (got=%v)", kind, cond, k, actual)
			}
		}
	}

	getShape := func(cond string) ConditionShape {
		shape, ok := shapes[cond]
		if !ok {
			t.Fatalf("expected condition shape for %s", cond)
		}
		return shape
	}

	// ValidityPeriod
	vShape := getShape("ValidityPeriod")
	checkKeys("allowed", "ValidityPeriod", vShape.AllowedKeys, []string{"startDate", "endDate"})
	checkKeys("required", "ValidityPeriod", vShape.RequiredKeys, []string{"startDate"})

	// PaymentTerms
	pShape := getShape("PaymentTerms")
	checkKeys("allowed", "PaymentTerms", pShape.AllowedKeys, []string{"rentAmount", "currency", "dueDayOfMonth"})
	checkKeys("required", "PaymentTerms", pShape.RequiredKeys, []string{"rentAmount", "currency", "dueDayOfMonth"})

	// DataAccessScope
	dShape := getShape("DataAccessScope")
	checkKeys("allowed", "DataAccessScope", dShape.AllowedKeys, []string{"region"})
	checkKeys("required", "DataAccessScope", dShape.RequiredKeys, []string{"region"})
}
