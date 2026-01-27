package parser

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"

	"digital-contracting-service/internal/semantic/loader"
)

// LocalName extracts the local name from an RDF term.
//
// It handles various RDF term formats:
//   - Prefixed names: "dcs:startDate" -> "startDate"
//   - IRIs with fragment: "<https://example.org/vocab#startDate>" -> "startDate"
//   - IRIs with path: "<https://example.org/vocab/startDate>" -> "startDate"
//
// Returns empty string if the term format is not recognized.
func LocalName(term string) string {
	term = strings.TrimSpace(term)
	if term == "" {
		return ""
	}
	// If it's an IRI wrapped in angle brackets: <...#startDate> or <.../startDate>
	if strings.HasPrefix(term, "<") && strings.HasSuffix(term, ">") {
		term = term[1 : len(term)-1] // Remove < and >
	}
	// Extract local name from IRI (with # or / separator)
	if j := strings.LastIndex(term, "#"); j >= 0 && j < len(term)-1 {
		return term[j+1:]
	}
	if j := strings.LastIndex(term, "/"); j >= 0 && j < len(term)-1 {
		return term[j+1:]
	}
	// If it's a prefixed name: dcs:startDate
	if i := strings.Index(term, ":"); i > 0 && i < len(term)-1 {
		return term[i+1:]
	}
	return ""
}

// extractPrefixIRI extracts the IRI for a given prefix name from TTL content.
//
// It searches for @prefix declarations in the TTL content and returns the IRI
// associated with the given prefix name. Returns empty string if not found.
func extractPrefixIRI(ttl []byte, prefixName string) string {
	text := string(ttl)
	lines := strings.Split(text, "\n")

	// Pattern: @prefix dcs: <https://...> .
	pattern := regexp.MustCompile(`@prefix\s+` + regexp.QuoteMeta(prefixName) + `:\s*<([^>]+)>`)
	for _, line := range lines {
		matches := pattern.FindStringSubmatch(line)
		if len(matches) == 2 {
			return matches[1]
		}
	}
	return ""
}

// buildSPARQLQueryByClass builds a SPARQL query to find all terms that are instances
// of the specified class within a given prefix namespace.
//
// Parameters:
//   - prefixName: the prefix name (e.g., "dcs")
//   - prefixIRI: the IRI for the prefix (e.g., "https://projects.eclipse.org/xfsc/facis/dcs#")
//   - className: the class name (e.g., "SemanticCondition")
//
// Returns a SPARQL query string that selects terms matching the criteria.
func buildSPARQLQueryByClass(prefixName, prefixIRI, className string) string {
	return fmt.Sprintf(`
PREFIX rdf: <http://www.w3.org/1999/02/22-rdf-syntax-ns#>
PREFIX %s: <%s>

SELECT ?term WHERE {
  ?term a %s:%s .
  FILTER(STRSTARTS(STR(?term), STR(%s:)))
}
`, prefixName, prefixIRI, prefixName, className, prefixName)
}

// QueryVocabByClass queries the vocabulary TTL file using SPARQL to find all terms
// that are instances of the specified class.
//
// It uses Apache Jena's arq command to execute a SPARQL query that finds all
// terms in the vocabulary TTL that are instances of the specified class.
//
// Parameters:
//   - vocabTTL: the vocabulary TTL content
//   - prefixName: the prefix name (e.g., "dcs")
//   - className: the class name (e.g., "SemanticCondition")
//
// Returns a map of term localNames, or an error if the query fails.
func QueryVocabByClass(vocabTTL []byte, prefixName, className string) (map[string]struct{}, error) {
	binPath := loader.GetJenaBinPath()
	if binPath == "" {
		return nil, fmt.Errorf("Jena not found, set JENA_HOME environment variable")
	}

	// Extract prefix IRI from TTL
	prefixIRI := extractPrefixIRI(vocabTTL, prefixName)
	if prefixIRI == "" {
		return nil, fmt.Errorf("prefix %q not found in vocabulary TTL", prefixName)
	}

	// Create temporary TTL file for vocabulary data
	dataFile, err := os.CreateTemp("", "vocab-*.ttl")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(dataFile.Name())
	defer dataFile.Close()

	if _, err := dataFile.Write(vocabTTL); err != nil {
		return nil, fmt.Errorf("failed to write temp file: %w", err)
	}
	dataFile.Close()

	// Build SPARQL query dynamically
	sparqlQuery := buildSPARQLQueryByClass(prefixName, prefixIRI, className)

	// Create temporary SPARQL query file
	queryFile, err := os.CreateTemp("", "query-*.rq")
	if err != nil {
		return nil, fmt.Errorf("failed to create query file: %w", err)
	}
	defer os.Remove(queryFile.Name())
	defer queryFile.Close()

	if _, err := queryFile.WriteString(sparqlQuery); err != nil {
		return nil, fmt.Errorf("failed to write query file: %w", err)
	}
	queryFile.Close()

	// Execute SPARQL query using Jena's arq command
	cmdResult, err := loader.RunJenaCommand("arq",
		"--data", dataFile.Name(),
		"--query", queryFile.Name(),
		"--results", "JSON")
	if err != nil {
		return nil, fmt.Errorf("SPARQL query failed: %w", err)
	}

	// Parse SPARQL JSON results and extract term localNames
	terms, err := parseSPARQLTermVariableResults(cmdResult.Stdout)
	if err != nil {
		return nil, fmt.Errorf("failed to parse SPARQL results: %w", err)
	}

	return terms, nil
}

// parseSPARQLTermVariableResults parses SPARQL query results in JSON format and extracts term localNames.
//
// It parses the JSON output from Apache Jena's arq command and extracts the "term" values
// from the bindings, then converts them to localNames.
//
// Example JSON format:
//
//	{
//	  "head": {
//	    "vars": [ "term" ]
//	  },
//	  "results": {
//	    "bindings": [
//	      {
//	        "term": {
//	          "type": "uri",
//	          "value": "https://projects.eclipse.org/xfsc/facis/dcs#validityPeriod"
//	        }
//	      },
//	      {
//	        "term": {
//	          "type": "uri",
//	          "value": "https://projects.eclipse.org/xfsc/facis/dcs#paymentTerms"
//	        }
//	      }
//	    ]
//	  }
//	}
//
// Returns a map of term localNames (e.g., "validityPeriod", "paymentTerms").
func parseSPARQLTermVariableResults(jsonOutput string) (map[string]struct{}, error) {
	var jsonResult struct {
		Results struct {
			Bindings []struct {
				Term struct {
					Value string `json:"value"`
				} `json:"term"`
			} `json:"bindings"`
		} `json:"results"`
	}

	if err := json.Unmarshal([]byte(jsonOutput), &jsonResult); err != nil {
		return nil, fmt.Errorf("failed to unmarshal SPARQL JSON results: %w", err)
	}

	terms := make(map[string]struct{})
	for _, binding := range jsonResult.Results.Bindings {
		localName := LocalName(binding.Term.Value)
		if localName != "" {
			terms[localName] = struct{}{}
		}
	}

	return terms, nil
}

// buildSPARQLPredicateCheckQuery builds a SPARQL query that retrieves all objects
// related to a specific subject (of a given class) via a given predicate.
//
// The query includes the following parameters:
//   - prefixName:          the prefix name (e.g., "dcs")
//   - prefixIRI:           the IRI for the prefix (e.g., "https://projects.eclipse.org/xfsc/facis/dcs#")
//   - className:           the class name for the subject (e.g., "SemanticCondition")
//   - predicateLocalName:  the local name of the predicate (e.g., "allowedKey")
//   - subjectLocalName:    the local name of the subject term (e.g., "validityPeriod")
//
// Returns a SPARQL query that selects all objects related to the subject.
func buildSPARQLPredicateCheckQuery(
	prefixName,
	prefixIRI,
	className,
	predicateLocalName,
	subjectLocalName string,
) string {
	fullSubjectIRI := prefixIRI + subjectLocalName

	return fmt.Sprintf(`
PREFIX rdf: <http://www.w3.org/1999/02/22-rdf-syntax-ns#>
PREFIX %s: <%s>

SELECT ?object WHERE {
  ?subject a %s:%s ;
           %s:%s ?object .
  FILTER(STR(?subject) = "%s")
}
`, prefixName, prefixIRI, prefixName, className, prefixName, predicateLocalName, fullSubjectIRI)
}

// IsAllowedKeyForCondition checks whether all given parameter keys are allowed
// for a given condition type in the vocabulary TTL.
//
// Returns true when all keys are allowed, false otherwise.
func IsAllowedKeyForCondition(
	vocabTTL []byte,
	prefixName string,
	className string,
	predicateLocalName string,
	conditionLocalName string,
	keyLocalNames []string,
) (bool, error) {
	binPath := loader.GetJenaBinPath()
	if binPath == "" {
		return false, fmt.Errorf("Jena not found, set JENA_HOME environment variable")
	}

	// Extract prefix IRI from TTL
	prefixIRI := extractPrefixIRI(vocabTTL, prefixName)
	if prefixIRI == "" {
		return false, fmt.Errorf("prefix %q not found in vocabulary TTL", prefixName)
	}

	// Create temporary TTL file for vocabulary data
	dataFile, err := os.CreateTemp("", "vocab-*.ttl")
	if err != nil {
		return false, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(dataFile.Name())
	defer dataFile.Close()

	if _, err := dataFile.Write(vocabTTL); err != nil {
		return false, fmt.Errorf("failed to write temp file: %w", err)
	}
	dataFile.Close()

	// Build SPARQL query to fetch all objects (keys) for the given
	// (conditionType, predicate) pair.
	sparqlQuery := buildSPARQLPredicateCheckQuery(
		prefixName,
		prefixIRI,
		className,
		predicateLocalName,
		conditionLocalName,
	)

	// Create temporary SPARQL query file
	queryFile, err := os.CreateTemp("", "allowed-key-check-*.rq")
	if err != nil {
		return false, fmt.Errorf("failed to create query file: %w", err)
	}
	defer os.Remove(queryFile.Name())
	defer queryFile.Close()

	if _, err := queryFile.WriteString(sparqlQuery); err != nil {
		return false, fmt.Errorf("failed to write query file: %w", err)
	}
	queryFile.Close()

	// Execute SPARQL query using Jena's arq command
	cmdResult, err := loader.RunJenaCommand("arq",
		"--data", dataFile.Name(),
		"--query", queryFile.Name(),
		"--results", "JSON")
	if err != nil {
		return false, fmt.Errorf("SPARQL query for allowed key failed: %w", err)
	}

	// Parse SPARQL JSON results and extract object localNames
	allowedSet, err := parseSPARQLObjectVariableResults(cmdResult.Stdout)
	if err != nil {
		return false, fmt.Errorf("failed to parse SPARQL allowed-key results: %w", err)
	}

	// Ensure all requested keys are present in the allowed set.
	for _, key := range keyLocalNames {
		if _, ok := allowedSet[key]; !ok {
			return false, nil
		}
	}

	return true, nil
}

// parseSPARQLObjectVariableResults parses SPARQL query results in JSON format and
// extracts localNames from the "object" variable.
//
// Example JSON format:
//
//	{
//	  "head": {
//	    "vars": [ "object" ]
//	  },
//	  "results": {
//	    "bindings": [
//	      {
//	        "object": {
//	          "type": "uri",
//	          "value": "https://projects.eclipse.org/xfsc/facis/dcs#startDate"
//	        }
//	      }
//	    ]
//	  }
//	}
//
// Returns a map of object localNames.
func parseSPARQLObjectVariableResults(jsonOutput string) (map[string]struct{}, error) {
	var jsonResult struct {
		Results struct {
			Bindings []struct {
				Object struct {
					Value string `json:"value"`
				} `json:"object"`
			} `json:"bindings"`
		} `json:"results"`
	}

	if err := json.Unmarshal([]byte(jsonOutput), &jsonResult); err != nil {
		return nil, fmt.Errorf("failed to unmarshal SPARQL JSON results: %w", err)
	}

	objects := make(map[string]struct{})
	for _, binding := range jsonResult.Results.Bindings {
		localName := LocalName(binding.Object.Value)
		if localName != "" {
			objects[localName] = struct{}{}
		}
	}

	return objects, nil
}
