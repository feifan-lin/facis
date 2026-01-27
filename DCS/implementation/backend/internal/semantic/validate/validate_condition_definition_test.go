package validate

import (
	"encoding/json"
	"testing"

	"digital-contracting-service/internal/semantic/loader"
)

func TestValidateConditionDefinition(t *testing.T) {
	// Load vocabulary TTL
	vocabResult := loader.LoadTTL("internal/semantic/validate/testdata/condition-definition/vocab.ttl")
	if vocabResult.Error != nil {
		t.Fatalf("failed to load vocabulary TTL: %v", vocabResult.Error)
	}
	vocabTTL := vocabResult.Content

	tests := []struct {
		name           string
		inputFile      string
		wantValid      bool
		wantErrorCount int
	}{
		{
			name:           "valid conditions with all types and parameters defined",
			inputFile:      "internal/semantic/validate/testdata/condition-definition/semantic_conditions_valid.json",
			wantValid:      true,
			wantErrorCount: 0,
		},
		{
			name:           "invalid conditionType not in vocabulary",
			inputFile:      "internal/semantic/validate/testdata/condition-definition/semantic_conditions_invalid.json",
			wantValid:      false,
			wantErrorCount: 2, // invalidConditionType and invalidParameter
		},
		{
			name:           "parameter key not allowed for conditionType via dcs:allowedKey",
			inputFile:      "internal/semantic/validate/testdata/condition-definition/semantic_conditions_invalid_allowed_keys.json",
			wantValid:      false,
			wantErrorCount: 1, // currency is a valid property but not allowed for validityPeriod
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Load input JSON
			inputResult := loader.LoadJSON(tt.inputFile)
			if inputResult.Error != nil {
				t.Fatalf("failed to load input file: %v", inputResult.Error)
			}

			// Parse JSON to semantic conditions
			var conditions []SemanticCondition
			if err := json.Unmarshal(inputResult.Content, &conditions); err != nil {
				t.Fatalf("failed to parse input JSON: %v", err)
			}

			// Validate
			result := ValidateConditionDefinition(conditions, vocabTTL)

			// Check validation result
			if result.Valid != tt.wantValid {
				t.Errorf("validation result mismatch: got Valid=%v, want %v", result.Valid, tt.wantValid)
			}

			if len(result.Errors) != tt.wantErrorCount {
				t.Errorf("error count mismatch: got %d errors, want %d. Errors: %v", len(result.Errors), tt.wantErrorCount, result.Errors)
			}

			// Log errors for debugging
			if len(result.Errors) > 0 {
				t.Logf("Validation errors: %v", result.Errors)
			}
		})
	}
}
