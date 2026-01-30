package validate

import (
	"testing"

	"digital-contracting-service/internal/semantic/loader"
	"digital-contracting-service/internal/semantic/parser"
)

// TestValidateConditionValues tests that JSON-LD data is validated
// against shapes.ttl using Apache Jena SHACL.
func TestValidateConditionValues(t *testing.T) {
	if loader.GetJenaBinPath() == "" {
		t.Skip("Jena not found. Set JENA_HOME environment variable")
	}

	contextResult := loader.LoadJSON("internal/semantic/validate/testdata/condition-values/context.jsonld")
	if contextResult.Error != nil {
		t.Fatalf("failed to load context: %v", contextResult.Error)
	}
	contextJSONLD := contextResult.Content

	shapesResult := loader.LoadTTL("internal/semantic/validate/testdata/condition-values/shapes.ttl")
	if shapesResult.Error != nil {
		t.Fatalf("failed to load shapes: %v", shapesResult.Error)
	}
	shapesTTL := shapesResult.Content

	tests := []struct {
		name               string
		conditionsJSONPath string
		wantValid          bool
		wantErrors         bool
	}{
		{
			name:               "valid ValidityPeriod with startDate and endDate",
			conditionsJSONPath: "internal/semantic/validate/testdata/condition-values/valid_condition.json",
			wantValid:          true,
			wantErrors:         false,
		},
		{
			name:               "valid ValidityPeriod missing optional endDate",
			conditionsJSONPath: "internal/semantic/validate/testdata/condition-values/valid_optional_end_date.json",
			wantValid:          true,
			wantErrors:         false,
		},
		{
			name:               "invalid ValidityPeriod missing required startDate",
			conditionsJSONPath: "internal/semantic/validate/testdata/condition-values/invalid_missing_start_date.json",
			wantValid:          false,
			wantErrors:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conditionsResult := loader.LoadJSON(tt.conditionsJSONPath)
			if conditionsResult.Error != nil {
				t.Fatalf("failed to load conditions JSON: %v", conditionsResult.Error)
			}

			jsonldBytes, err := parser.ConvertToJSONLD(conditionsResult.Content, contextJSONLD, "dcs")
			if err != nil {
				t.Fatalf("failed to convert to JSON-LD: %v", err)
			}

			result := ValidateConditionValues(jsonldBytes, shapesTTL)

			if result.Valid != tt.wantValid {
				t.Errorf("validation result mismatch: got Valid=%v, want %v", result.Valid, tt.wantValid)
			}

			if tt.wantValid {
				if len(result.Errors) > 0 {
					t.Errorf("expected no errors for valid data, but got: %v", result.Errors)
				}
			} else {
				if !tt.wantErrors && len(result.Errors) == 0 {
					t.Error("expected errors for invalid data, but got none")
				}
				if len(result.Errors) > 0 {
					t.Logf("Validation errors (expected): %v", result.Errors)
				}
			}
		})
	}
}
