package parser

import (
	"testing"

	"digital-contracting-service/internal/semantic/loader"
)

// TestBuildConditionJSONLDContext verifies that @context can be generated solely from
// vocab.ttl and shapes.ttl and that key mappings are as expected. Value
// conversion (@graph) is tested separately.
func TestBuildConditionJSONLDContext(t *testing.T) {
	// Load vocabulary TTL
	vocabResult := loader.LoadTTL("internal/semantic/validate/testdata/condition-definition/vocab.ttl")
	if vocabResult.Error != nil {
		t.Fatalf("failed to load vocabulary TTL: %v", vocabResult.Error)
	}

	// Load SHACL shapes TTL
	shapesResult := loader.LoadTTL("internal/semantic/validate/testdata/condition-values/shapes.ttl")
	if shapesResult.Error != nil {
		t.Fatalf("failed to load shapes TTL: %v", shapesResult.Error)
	}

	// Build @context only (no values) from vocab & shapes.
	context, err := BuildConditionJSONLDContext(vocabResult.Content, shapesResult.Content, "dcs")

	if err != nil {
		t.Fatalf("BuildConditionJSONLDContext failed: %v", err)
	}

	// Verify prefix definitions
	if dcsIRI, ok := context["dcs"].(string); !ok || dcsIRI != "https://projects.eclipse.org/xfsc/facis/dcs#" {
		t.Error("@context missing or incorrect dcs prefix")
	}

	if xsdIRI, ok := context["xsd"].(string); !ok || xsdIRI != "http://www.w3.org/2001/XMLSchema#" {
		t.Error("@context missing or incorrect xsd prefix")
	}

	// Verify id and conditionType mappings
	if conditionIdMapping, ok := context["conditionId"].(string); !ok || conditionIdMapping != "@id" {
		t.Error("@context missing or incorrect conditionId mapping to @id")
	}

	if condTypeMapping, ok := context["conditionType"].(string); !ok || condTypeMapping != "@type" {
		t.Error("@context missing or incorrect conditionType mapping")
	}

	// Verify condition type mappings exist (case-sensitive, must match vocab.ttl)
	if validityPeriodMapping, ok := context["ValidityPeriod"].(string); !ok {
		t.Error("@context missing ValidityPeriod mapping")
	} else if validityPeriodMapping != "dcs:ValidityPeriod" {
		t.Errorf("@context ValidityPeriod mapping incorrect: got %s, want dcs:ValidityPeriod", validityPeriodMapping)
	}

	// Verify property mappings with @nest
	startDateMapping, ok := context["startDate"].(map[string]interface{})
	if !ok {
		t.Error("@context missing startDate mapping")
	} else {
		if nest, ok := startDateMapping["@nest"].(string); !ok || nest != "parameters" {
			t.Error("startDate mapping missing @nest: parameters")
		}
		if dataType, ok := startDateMapping["@type"].(string); !ok || dataType != "http://www.w3.org/2001/XMLSchema#date" {
			t.Errorf("startDate mapping missing or incorrect @type, got: %v", startDateMapping["@type"])
		}
	}
}
