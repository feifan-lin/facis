package loader

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// JenaCommandResult represents the result of executing a Jena command.
type JenaCommandResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// RunJenaCommand executes a Jena command-line tool.
//
// It sets up the command with JENA_HOME environment variable and captures
// both stdout and stderr output.
//
// Parameters:
//   - tool: the Jena tool name (e.g., "arq", "shacl")
//   - args: command-line arguments for the tool
//
// Returns the command output and error information, or an error if the command fails.
func RunJenaCommand(tool string, args ...string) (*JenaCommandResult, error) {
	binPath := GetJenaBinPath()
	if binPath == "" {
		return nil, fmt.Errorf("Jena not found, set JENA_HOME environment variable")
	}

	toolPath := filepath.Join(binPath, tool)
	cmd := exec.Command(toolPath, args...)

	// Set JENA_HOME to the parent directory of bin
	cmd.Env = append(os.Environ(), "JENA_HOME="+filepath.Dir(binPath))

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	
	result := &JenaCommandResult{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}
	
	if cmd.ProcessState != nil {
		result.ExitCode = cmd.ProcessState.ExitCode()
	}

	if err != nil {
		return result, fmt.Errorf("%s command failed: %w, stderr: %s", tool, err, stderr.String())
	}

	return result, nil
}
