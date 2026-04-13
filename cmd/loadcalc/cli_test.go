package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestValidateCommand_ValidConfig(t *testing.T) {
	code, stdout, stderr := executeCommand("validate", "-i", "../../testdata/valid_config.yaml")
	if code != 0 {
		t.Errorf("expected exit 0 for valid config, got %d\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}
	if !strings.Contains(stdout, "Validation passed") {
		t.Errorf("expected 'Validation passed' in output, got: %s", stdout)
	}
}

func TestValidateCommand_InvalidConfig(t *testing.T) {
	code, stdout, _ := executeCommand("validate", "-i", "../../testdata/invalid_configs/missing_tool.yaml")
	if code != 1 {
		t.Errorf("expected exit 1 for invalid config, got %d\nstdout: %s", code, stdout)
	}
	if !strings.Contains(stdout, "Error") {
		t.Errorf("expected error message in output, got: %s", stdout)
	}
}

func TestVersionCommand(t *testing.T) {
	code, stdout, _ := executeCommand("version")
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	if !strings.Contains(stdout, "loadcalc") {
		t.Errorf("expected version string, got: %s", stdout)
	}
}

func TestTemplateYAML(t *testing.T) {
	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "template.yaml")

	code, _, stderr := executeCommand("template", "--format", "yaml", "-o", outPath)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d, stderr: %s", code, stderr)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("failed to read template: %v", err)
	}

	// Verify it's valid YAML
	var parsed map[string]interface{}
	if err := yaml.Unmarshal(data, &parsed); err != nil {
		t.Errorf("template is not valid YAML: %v", err)
	}

	// Should have expected top-level keys
	for _, key := range []string{"version", "global", "scenarios", "profile"} {
		if _, ok := parsed[key]; !ok {
			t.Errorf("template missing key: %s", key)
		}
	}
}

func TestTemplateXLSXPlaceholder(t *testing.T) {
	code, stdout, _ := executeCommand("template", "--format", "xlsx", "-o", "/tmp/dummy.xlsx")
	// Should indicate not yet implemented
	if code != 0 {
		t.Logf("exit code: %d", code)
	}
	if !strings.Contains(stdout, "not yet implemented") {
		t.Errorf("expected 'not yet implemented' message, got: %s", stdout)
	}
}

func TestCalculateCommand_JSONOutput(t *testing.T) {
	code, stdout, stderr := executeCommand("calculate", "-i", "../../testdata/valid_config.yaml", "--format", "json")
	if code != 0 {
		t.Fatalf("expected exit 0, got %d\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}

	// Should be valid JSON
	var parsed interface{}
	if err := json.Unmarshal([]byte(stdout), &parsed); err != nil {
		t.Errorf("output is not valid JSON: %v\noutput: %s", err, stdout)
	}
}

func TestCalculateCommand_TableOutput(t *testing.T) {
	code, stdout, stderr := executeCommand("calculate", "-i", "../../testdata/valid_config.yaml", "--format", "table")
	if code != 0 {
		t.Fatalf("expected exit 0, got %d\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}

	if stdout == "" {
		t.Error("expected table output, got empty string")
	}
}

func TestCalculateCommand_XLSXOutput(t *testing.T) {
	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "results.xlsx")

	code, _, stderr := executeCommand("calculate", "-i", "../../testdata/valid_config.yaml", "-o", outPath, "--format", "xlsx")
	if code != 0 {
		t.Fatalf("expected exit 0, got %d\nstderr: %s", code, stderr)
	}

	if _, err := os.Stat(outPath); os.IsNotExist(err) {
		t.Error("expected output XLSX file to exist")
	}
}

func TestCalculateCommand_InvalidConfig(t *testing.T) {
	code, _, _ := executeCommand("calculate", "-i", "../../testdata/invalid_configs/missing_tool.yaml", "--format", "table")
	if code != 1 {
		t.Errorf("expected exit 1 for invalid config, got %d", code)
	}
}

// executeCommand runs the root cobra command with args and captures output + exit code.
func executeCommand(args ...string) (exitCode int, stdout, stderr string) {
	var outBuf, errBuf bytes.Buffer
	cmd := newRootCmd()
	cmd.SetOut(&outBuf)
	cmd.SetErr(&errBuf)
	cmd.SetArgs(args)

	exitCode = 0
	err := cmd.Execute()
	if err != nil {
		exitCode = 1
	}

	return exitCode, outBuf.String(), errBuf.String()
}
