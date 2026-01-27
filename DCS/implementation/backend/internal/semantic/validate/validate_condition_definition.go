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
	ID            *string                `json:"id,omitempty"`
	ConditionType string                 `json:"conditionType"`
	Parameters    map[string]interface{} `json:"parameters"`
}

// ValidateConditionDefinition validates condition definitions against the vocabulary TTL.
//
// It checks:
//   - Each conditionType exists as an instance of dcs:SemanticCondition in the vocabulary TTL
//   - Each parameter key exists as an instance of dcs:ConditionProperty in the vocabulary TTL
//   - Each parameter key is allowed for the given conditionType via dcs:allowedKey
//
// The vocabulary TTL file (e.g., dcs.ttl) is provided via vocabTTL parameter.
func ValidateConditionDefinition(conditions []SemanticCondition, vocabTTL []byte) ConditionDefinitionResult {
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

	var errors []string

	// Validate each condition
	for i, condition := range conditions {
		// Validate conditionType
		conditionTypeExists := false
		if _, ok := conditionTypes[condition.ConditionType]; !ok {
			errors = append(errors, fmt.Sprintf("condition[%d].conditionType %q is not defined in vocabulary", i, condition.ConditionType))
		} else {
			conditionTypeExists = true
		}

		// Validate parameter keys
		var paramKeys []string
		for paramKey := range condition.Parameters {
			if _, ok := properties[paramKey]; !ok {
				errors = append(errors, fmt.Sprintf("condition[%d].parameters.%s is not defined in vocabulary", i, paramKey))
				continue
			}
			paramKeys = append(paramKeys, paramKey)
		}

		// check that all parameter keys are explicitly allowed
		// for this existing conditionType via dcs:allowedKey in the vocabulary.
		if conditionTypeExists && len(paramKeys) > 0 {
			allowed, err := parser.IsAllowedKeyForCondition(
				vocabTTL,
				"dcs",
				"SemanticCondition",
				"allowedKey",
				condition.ConditionType,
				paramKeys,
			)
			if err != nil {
				return ConditionDefinitionResult{
					Valid:  false,
					Errors: []string{fmt.Sprintf("failed to query vocabulary allowed keys: %v", err)},
				}
			}
			if !allowed {
				errors = append(errors, fmt.Sprintf("condition[%d].parameters contain keys that are not allowed for conditionType %q", i, condition.ConditionType))
			}
		}
	}

	return ConditionDefinitionResult{
		Valid:  len(errors) == 0,
		Errors: errors,
	}
}
