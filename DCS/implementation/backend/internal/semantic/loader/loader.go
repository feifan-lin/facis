package loader

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

var projectRoot string

func init() {
	projectRoot = findProjectRoot()
}

// findProjectRoot finds the Go project root.
// Returns the absolute path to the project root, or empty string if not found.
func findProjectRoot() string {
	_, file, _, ok := runtime.Caller(1)
	if !ok {
		return ""
	}

	dir := filepath.Dir(file)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			absPath, err := filepath.Abs(dir)
			if err == nil {
				return absPath
			}
			return dir
		}

		// Move up one directory
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root
			break
		}
		dir = parent
	}

	return ""
}


// loadFile loads a file from the given path relative to project root.
func loadFile(relPath string) ([]byte, error) {
	if projectRoot == "" {
		return nil, fmt.Errorf("project root not found (go.mod not found)")
	}

	fullPath := filepath.Join(projectRoot, relPath)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("file not found: %s (from project root: %s)", relPath, projectRoot)
	}
	return content, nil
}

// LoadResult represents the result of loading a resource.
type LoadResult struct {
	Content []byte
	Error   error
}

// LoadTTL loads a TTL file from the given path relative to project root.
func LoadTTL(relPath string) LoadResult {
	content, err := loadFile(relPath)
	if err != nil {
		return LoadResult{
			Error: fmt.Errorf("failed to load TTL file %s: %w", relPath, err),
		}
	}
	return LoadResult{
		Content: content,
	}
}

// LoadJSON loads a JSON file from the given path relative to project root.
func LoadJSON(relPath string) LoadResult {
	content, err := loadFile(relPath)
	if err != nil {
		return LoadResult{
			Error: fmt.Errorf("failed to load JSON file %s: %w", relPath, err),
		}
	}
	return LoadResult{
		Content: content,
	}
}

// LoadJSONInto loads a JSON file and unmarshals it into the provided value.
func LoadJSONInto(relPath string, v interface{}) error {
	result := LoadJSON(relPath)
	if result.Error != nil {
		return result.Error
	}
	if err := json.Unmarshal(result.Content, v); err != nil {
		return fmt.Errorf("failed to parse JSON file %s: %w", relPath, err)
	}
	return nil
}

// GetJenaBinPath returns the path to Jena's bin directory.
// Returns empty string if Jena is not found.
func GetJenaBinPath() string {
	// JENA_HOME (Jena installation directory)
	if jenaHome := os.Getenv("JENA_HOME"); jenaHome != "" {
		binPath := filepath.Join(jenaHome, "bin")
		if verifyJenaBin(binPath) {
			return binPath
		}
	}

	return ""
}

// verifyJenaBin verifies that the bin directory contains the shacl executable.
func verifyJenaBin(binPath string) bool {
	if _, err := os.Stat(filepath.Join(binPath, "shacl")); err == nil {
		return true
	}
	return false
}
