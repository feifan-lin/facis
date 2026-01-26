package loader

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadTTL(t *testing.T) {
	// Test loading a TTL file from testdata (relative to project root)
	result := LoadTTL("internal/semantic/validate/testdata/condition-definition/vocab.ttl")
	if result.Error != nil {
		t.Fatalf("failed to load TTL: %v", result.Error)
	}
	if len(result.Content) == 0 {
		t.Error("TTL content is empty")
	}
}

func TestLoadJSON(t *testing.T) {
	// Test loading a JSON file from testdata (relative to project root)
	result := LoadJSON("internal/semantic/validate/testdata/condition-definition/semantic_conditions_valid.json")
	if result.Error != nil {
		t.Fatalf("failed to load JSON: %v", result.Error)
	}
	if len(result.Content) == 0 {
		t.Error("JSON content is empty")
	}
}

func TestLoadJSONInto(t *testing.T) {
	var conditions []struct {
		ConditionType string                 `json:"conditionType"`
		Parameters    map[string]interface{} `json:"parameters"`
	}
	err := LoadJSONInto("internal/semantic/validate/testdata/condition-definition/semantic_conditions_valid.json", &conditions)
	if err != nil {
		t.Fatalf("failed to load JSON: %v", err)
	}
	if len(conditions) == 0 {
		t.Error("expected conditions to be loaded")
	}
}

func TestGetJenaBinPath(t *testing.T) {
	binPath := GetJenaBinPath()
	if binPath == "" {
		t.Skip("Jena not found, skipping test")
	}
	shaclPath := filepath.Join(binPath, "shacl")
	if _, err := os.Stat(shaclPath); err != nil {
		t.Errorf("SHACL tool not found at %s: %v", shaclPath, err)
	}
}
