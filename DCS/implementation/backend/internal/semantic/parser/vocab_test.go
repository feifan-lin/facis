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

func TestIsAllowedKeyForCondition(t *testing.T) {
	vocabResult := loader.LoadTTL("internal/semantic/validate/testdata/condition-definition/vocab.ttl")
	if vocabResult.Error != nil {
		t.Fatalf("failed to load vocabulary TTL: %v", vocabResult.Error)
	}

	// validityPeriod should allow startDate and endDate
	allowed, err := IsAllowedKeyForCondition(
		vocabResult.Content,
		"dcs",
		"SemanticCondition",
		"allowedKey",
		"validityPeriod",
		[]string{"startDate", "endDate"},
	)
	if err != nil {
		t.Fatalf("IsAllowedKeyForCondition returned error: %v", err)
	}
	if !allowed {
		t.Errorf("expected startDate and endDate to be allowed for validityPeriod")
	}

	// validityPeriod should NOT allow currency
	notAllowed, err := IsAllowedKeyForCondition(
		vocabResult.Content,
		"dcs",
		"SemanticCondition",
		"allowedKey",
		"validityPeriod",
		[]string{"startDate", "currency"},
	)
	if err != nil {
		t.Fatalf("IsAllowedKeyForCondition (currency) returned error: %v", err)
	}
	if notAllowed {
		t.Errorf("did not expect currency to be allowed for validityPeriod")
	}
}
