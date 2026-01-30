package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"digital-contracting-service/internal/semantic/loader"
	"digital-contracting-service/internal/semantic/parser"
)

// gen-jsonld-context is a small utility that generates a JSON-LD @context
// document from vocab.ttl and shapes.ttl.
//
// Usage example:
//
//	go run ./cmd/gen-jsonld-context \
//	  -vocab internal/semantic/validate/testdata/condition-definition/vocab.ttl \
//	  -shapes internal/semantic/validate/testdata/condition-values/shapes.ttl \
//	  -prefix dcs \
//	  -out internal/semantic/validate/testdata/condition-values/context.jsonld
func main() {
	vocabPath := flag.String("vocab", "", "path to vocabulary TTL file (e.g., dcs.ttl)")
	shapesPath := flag.String("shapes", "", "path to SHACL shapes TTL file")
	prefixName := flag.String("prefix", "dcs", "RDF prefix name used in the TTL files (default: dcs)")
	outPath := flag.String("out", "", "output file path for generated context JSON-LD (default: stdout)")
	flag.Parse()

	validateInputParams(*vocabPath, *shapesPath, *outPath)

	// Load vocab TTL
	vocabResult := loader.LoadTTL(*vocabPath)
	if vocabResult.Error != nil {
		log.Fatalf("failed to load vocab TTL %q: %v", *vocabPath, vocabResult.Error)
	}

	// Load shapes TTL
	shapesResult := loader.LoadTTL(*shapesPath)
	if shapesResult.Error != nil {
		log.Fatalf("failed to load shapes TTL %q: %v", *shapesPath, shapesResult.Error)
	}

	// Build context map
	ctx, err := parser.BuildJSONLDContext(vocabResult.Content, shapesResult.Content, *prefixName)
	if err != nil {
		log.Fatalf("failed to build JSON-LD context: %v", err)
	}

	// Wrap into a top-level object with @context for convenience.
	outObj := map[string]interface{}{
		"@context": ctx,
	}

	data, err := json.MarshalIndent(outObj, "", "  ")
	if err != nil {
		log.Fatalf("failed to marshal JSON-LD context: %v", err)
	}

	if *outPath == "" {
		// Write to stdout
		if _, err := os.Stdout.Write(data); err != nil {
			log.Fatalf("failed to write context to stdout: %v", err)
		}
		fmt.Println()
		return
	}

	// Write to file
	if err := os.WriteFile(*outPath, data, 0o644); err != nil {
		log.Fatalf("failed to write context to file %q: %v", *outPath, err)
	}
}

// validateInputParams checks required args and file extensions (.ttl / .jsonld).
func validateInputParams(vocabPath, shapesPath, outPath string) {
	if vocabPath == "" || shapesPath == "" {
		log.Fatalf("both -vocab and -shapes must be provided")
	}
	if !hasExt(vocabPath, ".ttl") {
		log.Fatalf("-vocab must be a .ttl file, got %q", vocabPath)
	}
	if !hasExt(shapesPath, ".ttl") {
		log.Fatalf("-shapes must be a .ttl file, got %q", shapesPath)
	}
	if outPath != "" && !hasExt(outPath, ".jsonld") {
		log.Fatalf("-out must be a .jsonld file to avoid overwriting other files, got %q", outPath)
	}
}

func hasExt(path, ext string) bool {
	return strings.EqualFold(strings.TrimPrefix(filepath.Ext(path), "."), strings.TrimPrefix(ext, "."))
}
