package validate

import (
	"testing"

	"digital-contracting-service/internal/semantic/loader"
)

// TestValidateRDFDataAgainstShapes tests that RDF data (in TTL format) is validated
// against shapes.ttl using Apache Jena SHACL validation.
func TestValidateRDFDataAgainstShapes(t *testing.T) {
	// Check if Jena is available via JENA_HOME environment variable
	if loader.GetJenaBinPath() == "" {
		t.Skip("Jena not found. Set JENA_HOME environment variable")
	}

	tests := []struct {
		name       string
		dataFile   string
		shapesFile string
		wantValid  bool
		wantErrors bool // whether we expect errors
	}{
		{
			name:       "valid RDF data with startDate and endDate",
			dataFile:   "internal/semantic/validate/testdata/condition-values/data.ttl",
			shapesFile: "internal/semantic/validate/testdata/condition-values/shapes.ttl",
			wantValid:  true,
			wantErrors: false,
		},
		{
			name:       "invalid RDF data missing endDate",
			dataFile:   "internal/semantic/validate/testdata/condition-values/data_invalid.ttl",
			shapesFile: "internal/semantic/validate/testdata/condition-values/shapes.ttl",
			wantValid:  false,
			wantErrors: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Load data and shapes using loader
			dataResult := loader.LoadTTL(tt.dataFile)
			if dataResult.Error != nil {
				t.Fatalf("failed to load data file: %v", dataResult.Error)
			}
			dataTTL := dataResult.Content

			shapesResult := loader.LoadTTL(tt.shapesFile)
			if shapesResult.Error != nil {
				t.Fatalf("failed to load shapes file: %v", shapesResult.Error)
			}
			shapesTTL := shapesResult.Content

			// Perform SHACL validation using Apache Jena
			result := ValidateConditionValues(dataTTL, shapesTTL)

			// Verify validation result
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
