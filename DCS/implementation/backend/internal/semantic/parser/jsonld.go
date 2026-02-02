package parser

import (
	"encoding/json"
	"fmt"
	"os"

	"digital-contracting-service/internal/semantic/loader"
)

// BuildConditionJSONLDContext builds a JSON-LD @context for semantic conditions
// from vocabulary and SHACL shapes TTL files.
//
// It uses vocab.ttl to discover condition types and properties, and shapes.ttl
// to derive property datatypes. The result is a Go map representing the @context section,
// which callers can embed into a JSON-LD document or serialize independently.
//
// Parameters:
//   - vocabTTL:  the vocabulary TTL content for prefix resolution
//   - shapesTTL: the SHACL shapes TTL content for data type resolution
//   - prefixName: the prefix name (e.g., "dcs")
//
// Returns a map representing the @context, or an error if construction fails.
func BuildConditionJSONLDContext(vocabTTL, shapesTTL []byte, prefixName string) (map[string]interface{}, error) {
	prefixIRI := extractPrefixIRI(vocabTTL, prefixName)
	if prefixIRI == "" {
		return nil, fmt.Errorf("prefix %q not found in vocabulary TTL", prefixName)
	}

	context := make(map[string]interface{})

	// Add prefix definitions
	context["dcs"] = prefixIRI
	context["xsd"] = "http://www.w3.org/2001/XMLSchema#"

	// Map conditionId and conditionType
	context["conditionId"] = "@id"
	context["conditionType"] = "@type"

	// Query condition types from vocab.ttl
	conditionTypes, err := QueryVocabByClass(vocabTTL, prefixName, "SemanticCondition")
	if err != nil {
		return nil, fmt.Errorf("failed to query condition types: %w", err)
	}

	// Add condition type mappings (case-sensitive, e.g., "ValidityPeriod": "dcs:ValidityPeriod")
	for condType := range conditionTypes {
		context[condType] = fmt.Sprintf("%s:%s", prefixName, condType)
	}

	// parameters is a @nest container for grouping property keys in JSON
	context["parameters"] = "@nest"

	// Query properties from vocab.ttl
	properties, err := QueryVocabByClass(vocabTTL, prefixName, "ConditionProperty")
	if err != nil {
		return nil, fmt.Errorf("failed to query properties: %w", err)
	}

	// Query data types from shapes.ttl
	propertyTypes, err := queryPropertyDataTypes(shapesTTL, prefixName)
	if err != nil {
		return nil, fmt.Errorf("failed to query property data types: %w", err)
	}

	// Add property mappings with @nest and data types
	for prop := range properties {
		propContext := map[string]interface{}{
			"@id":   fmt.Sprintf("%s:%s", prefixName, prop),
			"@nest": "parameters",
		}

		// Add data type if found in shapes
		if dataType, ok := propertyTypes[prop]; ok {
			propContext["@type"] = dataType
		}

		context[prop] = propContext
	}

	return context, nil
}

// queryPropertyDataTypes queries SHACL shapes to extract datatypes for
// condition properties.
//
// It executes a SPARQL query over the given shapes TTL to find, for each
// property within the specified prefix, the corresponding xsd datatype. The
// result is a map keyed by property local name (e.g., "startDate") whose
// values are datatype IRIs (e.g., "http://www.w3.org/2001/XMLSchema#date").
func queryPropertyDataTypes(shapesTTL []byte, prefixName string) (map[string]string, error) {
	binPath := loader.GetJenaBinPath()
	if binPath == "" {
		return nil, fmt.Errorf("Jena not found, set JENA_HOME environment variable")
	}

	// Extract prefix IRI from TTL
	prefixIRI := extractPrefixIRI(shapesTTL, prefixName)
	if prefixIRI == "" {
		return nil, fmt.Errorf("prefix %q not found in shapes TTL", prefixName)
	}

	// Create temporary TTL file
	dataFile, err := os.CreateTemp("", "shapes-*.ttl")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(dataFile.Name())
	defer dataFile.Close()

	if _, err := dataFile.Write(shapesTTL); err != nil {
		return nil, fmt.Errorf("failed to write temp file: %w", err)
	}
	dataFile.Close()

	// Build SPARQL query to extract property datatypes from SHACL shapes.
	sparqlQuery := buildSPARQLPropertyDataTypesQuery(prefixName, prefixIRI)

	// Create temporary SPARQL query file
	queryFile, err := os.CreateTemp("", "shapes-query-*.rq")
	if err != nil {
		return nil, fmt.Errorf("failed to create query file: %w", err)
	}
	defer os.Remove(queryFile.Name())
	defer queryFile.Close()

	if _, err := queryFile.WriteString(sparqlQuery); err != nil {
		return nil, fmt.Errorf("failed to write query file: %w", err)
	}
	queryFile.Close()

	// Execute SPARQL query
	cmdResult, err := loader.RunJenaCommand("arq",
		"--data", dataFile.Name(),
		"--query", queryFile.Name(),
		"--results", "JSON")
	if err != nil {
		return nil, fmt.Errorf("SPARQL query failed: %w", err)
	}

	// Parse SPARQL JSON results into a mapping of property local name to
	// datatype IRI.
	return parseSHACLPropertyDataTypesResults(cmdResult.Stdout)
}

// buildSPARQLPropertyDataTypesQuery builds a SPARQL query that retrieves
// property paths and their datatypes from SHACL shapes.
//
// Parameters:
//   - prefixName: the prefix name used in the shapes TTL (e.g., "dcs")
//   - prefixIRI:  the IRI for the prefix (e.g., "https://projects.eclipse.org/xfsc/facis/dcs#")
//
// Returns a SPARQL query string that selects property paths and their datatype IRIs from SHACL NodeShapes.
func buildSPARQLPropertyDataTypesQuery(prefixName, prefixIRI string) string {
	return fmt.Sprintf(`
PREFIX sh: <http://www.w3.org/ns/shacl#>
PREFIX %s: <%s>

SELECT ?path ?datatype WHERE {
  ?shape a sh:NodeShape ;
         sh:property [ sh:path ?path ; sh:datatype ?datatype ] .
  FILTER(STRSTARTS(STR(?path), STR(%s:)))
}
`, prefixName, prefixIRI, prefixName)
}

// parseSHACLPropertyDataTypesResults parses SPARQL query results in JSON
// format and builds a map from property local name to datatype IRI.
//
// It expects the JSON output produced by Apache Jena's arq command when
// executing the query generated by buildSPARQLPropertyDataTypesQuery, where
// the variables "path" and "datatype" are bound.
//
// Example JSON format:
//
//	{
//	  "head": {
//	    "vars": [ "path", "datatype" ]
//	  },
//	  "results": {
//	    "bindings": [
//	      {
//	        "path": {
//	          "type": "uri",
//	          "value": "https://projects.eclipse.org/xfsc/facis/dcs#startDate"
//	        },
//	        "datatype": {
//	          "type": "uri",
//	          "value": "http://www.w3.org/2001/XMLSchema#date"
//	        }
//	      }
//	    ]
//	  }
//	}
//
// For each binding, the function:
//   - derives the local name from "path" using LocalName,
//   - and associates it with the datatype IRI from "datatype".
//
// Returns a map keyed by property local name (e.g., "startDate") whose values
// are datatype IRIs (e.g., "http://www.w3.org/2001/XMLSchema#date"), or an
// error if JSON parsing fails.
func parseSHACLPropertyDataTypesResults(jsonOutput string) (map[string]string, error) {
	// Parse results
	var jsonResult struct {
		Results struct {
			Bindings []struct {
				Path struct {
					Value string `json:"value"`
				} `json:"path"`
				Datatype struct {
					Value string `json:"value"`
				} `json:"datatype"`
			} `json:"bindings"`
		} `json:"results"`
	}

	if err := json.Unmarshal([]byte(jsonOutput), &jsonResult); err != nil {
		return nil, fmt.Errorf("failed to unmarshal SPARQL results: %w", err)
	}

	// Build property -> datatype mapping
	propertyTypes := make(map[string]string)
	for _, binding := range jsonResult.Results.Bindings {
		propLocalName := LocalName(binding.Path.Value)
		if propLocalName != "" {
			propertyTypes[propLocalName] = binding.Datatype.Value
		}
	}

	return propertyTypes, nil
}

// ConvertToJSONLD converts semantic conditions JSON to JSON-LD format using a
// pre-built JSON-LD context document.
//
// It expects contextJSONLD to contain either:
//   - a top-level object with an "@context" field, or
//   - a plain @context object itself.
//
// Parameters:
//   - conditionsJSON: the semantic conditions JSON (already validated)
//   - contextJSONLD:  a JSON document providing the @context definition
//   - prefixName:     the prefix name used for condition IRIs (e.g., "dcs")
//
// Returns JSON-LD formatted data as bytes, or an error if conversion fails.
func ConvertToJSONLD(conditionsJSON, contextJSONLD []byte, prefixName string) ([]byte, error) {
	// Parse context JSON
	var ctxWrapper map[string]interface{}
	if err := json.Unmarshal(contextJSONLD, &ctxWrapper); err != nil {
		return nil, fmt.Errorf("failed to parse context JSON-LD: %w", err)
	}

	var context map[string]interface{}
	if rawCtx, ok := ctxWrapper["@context"]; ok {
		if m, ok := rawCtx.(map[string]interface{}); ok {
			context = m
		} else {
			return nil, fmt.Errorf("@context is not an object")
		}
	} else {
		// Treat the whole document as the @context map.
		context = ctxWrapper
	}

	// Resolve prefix IRI from context so we can build full IRIs for condition types.
	rawPrefixIRI, ok := context[prefixName].(string)
	if !ok || rawPrefixIRI == "" {
		return nil, fmt.Errorf("prefix %q not found in JSON-LD context", prefixName)
	}
	prefixIRI := rawPrefixIRI

	// Parse input JSON
	var conditions []map[string]interface{}
	if err := json.Unmarshal(conditionsJSON, &conditions); err != nil {
		return nil, fmt.Errorf("failed to parse conditions JSON: %w", err)
	}

	// Convert conditions to JSON-LD format
	jsonldConditions := make([]map[string]interface{}, 0, len(conditions))
	for _, condition := range conditions {
		jsonldCondition, err := convertConditionToJSONLD(condition, prefixName, prefixIRI)
		if err != nil {
			return nil, fmt.Errorf("failed to convert condition: %w", err)
		}
		jsonldConditions = append(jsonldConditions, jsonldCondition)
	}

	// Build final JSON-LD structure
	jsonld := map[string]interface{}{
		"@context": context,
		"@graph":   jsonldConditions,
	}

	// Marshal to JSON-LD
	result, err := json.MarshalIndent(jsonld, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JSON-LD: %w", err)
	}

	return result, nil
}

// convertConditionToJSONLD converts a single condition to JSON-LD format.
func convertConditionToJSONLD(condition map[string]interface{}, prefixName, prefixIRI string) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	// Handle identifier
	if s, ok := condition["conditionId"].(string); ok && s != "" {
		result["conditionId"] = fmt.Sprintf("%s:%s", prefixName, s)
	}

	// Convert conditionType to @type (case-sensitive, must match vocab.ttl)
	conditionType, ok := condition["conditionType"].(string)
	if !ok {
		return nil, fmt.Errorf("condition missing conditionType")
	}
	result["conditionType"] = fmt.Sprintf("%s:%s", prefixName, conditionType)

	// Output parameters as nested object; @context has "parameters": "@nest" and
	// each param has "@nest": "parameters", so JSON-LD 1.1 expansion flattens to RDF.
	if params, ok := condition["parameters"].(map[string]interface{}); ok && len(params) > 0 {
		nested := make(map[string]interface{})
		for key, value := range params {
			if value != nil {
				nested[key] = value
			}
		}
		if len(nested) > 0 {
			result["parameters"] = nested
		}
	}

	return result, nil
}
