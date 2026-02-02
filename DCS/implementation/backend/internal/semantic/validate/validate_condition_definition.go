package validate

import (
	"fmt"

	"digital-contracting-service/internal/semantic/parser"
)

// ConditionDefinitionResult represents the validation result.
type ConditionDefinitionResult struct {
	Valid  bool     `json:"valid"`
	Errors []string `json:"errors,omitempty"`
}

// SemanticCondition represents a single semantic condition with its type and parameters.
// This matches the structure in contract template's semantic_conditions array.
type SemanticCondition struct {
	ConditionId   *string                `json:"conditionId"`
	ConditionType string                 `json:"conditionType"`
	Parameters    map[string]interface{} `json:"parameters"`
}

// ValidateConditionDefinition validates condition definitions against the vocabulary and SHACL shapes TTLs.
//
// It checks:
//   - Each conditionType exists as an instance of dcs:SemanticCondition in the vocabulary TTL
//   - Each parameter key exists as an instance of dcs:ConditionProperty in the vocabulary TTL
//   - For each conditionType, parameter keys are allowed/required according to SHACL shapes
//
// The vocabulary TTL file (e.g., dcs.ttl) is provided via vocabTTL parameter.
// The SHACL shapes TTL file (e.g., shapes.ttl) is provided via shapesTTL parameter.
func ValidateConditionDefinition(conditions []SemanticCondition, vocabTTL, shapesTTL []byte) ConditionDefinitionResult {
	// Query vocabulary for condition types
	conditionTypes, err := parser.QueryVocabByClass(vocabTTL, "dcs", "SemanticCondition")
	if err != nil {
		return ConditionDefinitionResult{
			Valid:  false,
			Errors: []string{fmt.Sprintf("failed to query vocabulary condition types: %v", err)},
		}
	}

	// Query vocabulary for properties
	properties, err := parser.QueryVocabByClass(vocabTTL, "dcs", "ConditionProperty")
	if err != nil {
		return ConditionDefinitionResult{
			Valid:  false,
			Errors: []string{fmt.Sprintf("failed to query vocabulary properties: %v", err)},
		}
	}

	// Build condition shapes (allowed and required keys) from SHACL shapes.
	conditionShapes, err := parser.BuildConditionShapesFromSHACL(shapesTTL, "dcs")
	if err != nil {
		return ConditionDefinitionResult{
			Valid:  false,
			Errors: []string{fmt.Sprintf("failed to build condition shapes from SHACL: %v", err)},
		}
	}

	var errors []string

	// Validate each condition independently to keep logic testable and readable.
	for i, condition := range conditions {
		errors = append(errors, validateSingleCondition(i, condition, conditionTypes, properties, conditionShapes)...)
	}

	return ConditionDefinitionResult{
		Valid:  len(errors) == 0,
		Errors: errors,
	}
}

// validateSingleCondition validates one semantic condition against:
//   - vocabulary-derived condition types and properties
//   - SHACL-derived condition shapes (allowed and required keys)
//
// It returns a slice of error messages for this single condition index.
func validateSingleCondition(
	index int,
	condition SemanticCondition,
	conditionTypes map[string]struct{},
	properties map[string]struct{},
	conditionShapes map[string]parser.ConditionShape,
) []string {
	var errors []string

	// Validate conditionType
	conditionTypeExists := false
	if _, ok := conditionTypes[condition.ConditionType]; !ok {
		errors = append(errors, fmt.Sprintf("condition[%d].conditionType %q is not defined in vocabulary", index, condition.ConditionType))
	} else {
		conditionTypeExists = true
	}

	// Validate parameter keys (exist as ConditionProperty)
	var paramKeys []string
	for paramKey := range condition.Parameters {
		if _, ok := properties[paramKey]; !ok {
			errors = append(errors, fmt.Sprintf("condition[%d].parameters.%s is not defined in vocabulary", index, paramKey))
			continue
		}
		paramKeys = append(paramKeys, paramKey)
	}

	// If we have SHACL shapes for this conditionType, enforce:
	//   - All parameter keys must be allowed for this conditionType.
	//   - All required keys (sh:minCount >= 1) must be present.
	if conditionTypeExists && len(paramKeys) > 0 {
		shape, ok := conditionShapes[condition.ConditionType]
		if ok {
			// Build a set of parameter keys for quick lookup.
			paramKeySet := make(map[string]struct{}, len(paramKeys))
			for _, k := range paramKeys {
				paramKeySet[k] = struct{}{}
			}

			// Check that all provided keys are allowed.
			allAllowed := true
			for _, k := range paramKeys {
				if _, ok := shape.AllowedKeys[k]; !ok {
					allAllowed = false
					break
				}
			}
			if !allAllowed {
				errors = append(errors, fmt.Sprintf("condition[%d].parameters contain keys that are not allowed for conditionType %q", index, condition.ConditionType))
			}

			// Check that all required keys are present.
			for requiredKey := range shape.RequiredKeys {
				if _, ok := paramKeySet[requiredKey]; !ok {
					errors = append(errors, fmt.Sprintf("condition[%d].parameters missing required key %q for conditionType %q", index, requiredKey, condition.ConditionType))
				}
			}
		}
	}

	return errors
}
