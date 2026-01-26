package validate

import (
	"fmt"
	"os"
	"strings"

	"digital-contracting-service/internal/semantic/loader"
)

// ConditionValueValidationResult represents the result of SHACL validation.
type ConditionValueValidationResult struct {
	Valid  bool     `json:"valid"`
	Errors []string `json:"errors,omitempty"`
}

// ValidateConditionValues validates RDF data against SHACL shapes.
//
// Parameters:
//   - dataTTL: RDF data in Turtle format
//   - shapesTTL: SHACL shapes in Turtle format
//
// Returns a validation result indicating whether the data conforms to the shapes
// and any validation errors if found.
func ValidateConditionValues(dataTTL, shapesTTL []byte) ConditionValueValidationResult {
	// Create temporary files for data and shapes
	dataFile, err := os.CreateTemp("", "data-*.ttl")
	if err != nil {
		return ConditionValueValidationResult{
			Valid:  false,
			Errors: []string{fmt.Sprintf("failed to create temporary data file: %v", err)},
		}
	}
	dataPath := dataFile.Name()
	defer os.Remove(dataPath)
	defer dataFile.Close()

	if _, err := dataFile.Write(dataTTL); err != nil {
		return ConditionValueValidationResult{
			Valid:  false,
			Errors: []string{fmt.Sprintf("failed to write data file: %v", err)},
		}
	}

	shapesFile, err := os.CreateTemp("", "shapes-*.ttl")
	if err != nil {
		return ConditionValueValidationResult{
			Valid:  false,
			Errors: []string{fmt.Sprintf("failed to create temporary shapes file: %v", err)},
		}
	}
	shapesPath := shapesFile.Name()
	defer os.Remove(shapesPath)
	defer shapesFile.Close()

	if _, err := shapesFile.Write(shapesTTL); err != nil {
		return ConditionValueValidationResult{
			Valid:  false,
			Errors: []string{fmt.Sprintf("failed to write shapes file: %v", err)},
		}
	}

	// Call Jena SHACL validator
	cmdResult, err := loader.RunJenaCommand("shacl",
		"validate",
		"--shapes", shapesPath,
		"--data", dataPath)
	if err != nil {
		return ConditionValueValidationResult{
			Valid:  false,
			Errors: []string{fmt.Sprintf("SHACL validation failed: %v", err)},
		}
	}

	// Parse SHACL validation report (Turtle format)
	return parseSHACLReport(cmdResult.Stdout)
}

// parseSHACLReport parses the SHACL validation report in Turtle format.
//
// The report contains sh:conforms and sh:result triples. It extracts validation
// results including whether the data conforms to the shapes and any error messages.
//
// Example report format (valid):
//
//	PREFIX sh: <http://www.w3.org/ns/shacl#>
//	[ rdf:type sh:ValidationReport;
//	  sh:conforms true
//	] .
//
// Example report format (invalid):
//
//	PREFIX sh: <http://www.w3.org/ns/shacl#>
//	[ rdf:type sh:ValidationReport;
//	  sh:conforms false;
//	  sh:result [ sh:resultMessage "minCount[1]: Invalid cardinality: expected min 1: Got count = 0";
//	              sh:resultPath dcs:endDate;
//	            ]
//	] .
func parseSHACLReport(report string) ConditionValueValidationResult {
	result := ConditionValueValidationResult{
		Valid:  false,
		Errors: []string{},
	}

	lines := strings.Split(report, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Check for sh:conforms true
		if strings.Contains(line, "sh:conforms") && strings.Contains(line, "true") {
			result.Valid = true
		}
		// Extract error messages from sh:resultMessage
		if strings.Contains(line, "sh:resultMessage") {
			// Extract the message value
			// Format: sh:resultMessage "message text" ;
			if idx := strings.Index(line, `"`); idx >= 0 {
				start := idx + 1
				if endIdx := strings.Index(line[start:], `"`); endIdx >= 0 {
					msg := line[start : start+endIdx]
					if msg != "" {
						result.Errors = append(result.Errors, msg)
					}
				}
			}
		}
		// Extract property path from sh:resultPath
		if strings.Contains(line, "sh:resultPath") && len(result.Errors) > 0 {
			// Extract the path (e.g., dcs:endDate)
			parts := strings.Fields(line)
			for i, part := range parts {
				if part == "sh:resultPath" && i+1 < len(parts) {
					path := strings.Trim(parts[i+1], ";")
					// Add path context to the last error message
					if len(result.Errors) > 0 {
						lastIdx := len(result.Errors) - 1
						result.Errors[lastIdx] = fmt.Sprintf("%s (path: %s)", result.Errors[lastIdx], path)
					}
					break
				}
			}
		}
	}

	// If no errors found but conforms is false, add a generic error
	if !result.Valid && len(result.Errors) == 0 {
		result.Errors = []string{"SHACL validation failed (no detailed error messages found)"}
	}

	return result
}
