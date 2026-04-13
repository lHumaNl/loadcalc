package main

import (
	"strings"
	"testing"
)

func TestQuickCmd_SingleStep_JMeterClosed(t *testing.T) {
	code, stdout, stderr := executeCommand("quick", "720000", "1100", "jmeter")
	if code != 0 {
		t.Fatalf("expected exit 0, got %d\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}
	if !strings.Contains(stdout, "Threads:") {
		t.Errorf("expected 'Threads:' in output, got: %s", stdout)
	}
	if !strings.Contains(stdout, "CTT:") {
		t.Errorf("expected 'CTT:' in output, got: %s", stdout)
	}
	if !strings.Contains(stdout, "Pacing:") {
		t.Errorf("expected 'Pacing:' in output, got: %s", stdout)
	}
	if !strings.Contains(stdout, "JMeter") {
		t.Errorf("expected 'JMeter' in output, got: %s", stdout)
	}
}

func TestQuickCmd_SingleStep_LREPC(t *testing.T) {
	code, stdout, stderr := executeCommand("quick", "720000", "1100", "lre_pc")
	if code != 0 {
		t.Fatalf("expected exit 0, got %d\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}
	if !strings.Contains(stdout, "Pacing:") {
		t.Errorf("expected 'Pacing:' in output, got: %s", stdout)
	}
	if strings.Contains(stdout, "CTT:") {
		t.Errorf("expected no 'CTT:' for LRE PC, got: %s", stdout)
	}
	if !strings.Contains(stdout, "LRE PC") {
		t.Errorf("expected 'LRE PC' in output, got: %s", stdout)
	}
}

func TestQuickCmd_SingleStep_JMeterOpen(t *testing.T) {
	code, stdout, stderr := executeCommand("quick", "720000", "1100", "jmeter", "--model", "open")
	if code != 0 {
		t.Fatalf("expected exit 0, got %d\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}
	if !strings.Contains(stdout, "Rate:") {
		t.Errorf("expected 'Rate:' in output, got: %s", stdout)
	}
	if strings.Contains(stdout, "Threads:") {
		t.Errorf("expected no 'Threads:' for open model, got: %s", stdout)
	}
}

func TestQuickCmd_MultiStep_JMeterClosed(t *testing.T) {
	code, stdout, stderr := executeCommand("quick", "720000", "1100", "jmeter", "--steps", "50,75,100,125,150", "-g", "3")
	if code != 0 {
		t.Fatalf("expected exit 0, got %d\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}
	// Should have 5 step rows
	if !strings.Contains(stdout, "50%") {
		t.Errorf("expected '50%%' step in output, got: %s", stdout)
	}
	if !strings.Contains(stdout, "150%") {
		t.Errorf("expected '150%%' step in output, got: %s", stdout)
	}
	if !strings.Contains(stdout, "Threads/Gen") {
		t.Errorf("expected 'Threads/Gen' column header, got: %s", stdout)
	}
}

func TestQuickCmd_MultiStep_LREPC(t *testing.T) {
	code, stdout, stderr := executeCommand("quick", "720000", "1100", "lre_pc", "--steps", "50,75,100", "--rampup", "60")
	if code != 0 {
		t.Fatalf("expected exit 0, got %d\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}
	if !strings.Contains(stdout, "Vusers") {
		t.Errorf("expected 'Vusers' column header, got: %s", stdout)
	}
	if !strings.Contains(stdout, "Delta") {
		t.Errorf("expected 'Delta' column header, got: %s", stdout)
	}
	if !strings.Contains(stdout, "Batch") {
		t.Errorf("expected 'Batch' column header, got: %s", stdout)
	}
}

func TestQuickCmd_WithGenerators(t *testing.T) {
	code, stdout, stderr := executeCommand("quick", "720000", "1100", "jmeter", "-g", "3")
	if code != 0 {
		t.Fatalf("expected exit 0, got %d\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}
	if !strings.Contains(stdout, "3 generator") {
		t.Errorf("expected '3 generator' in output, got: %s", stdout)
	}
}

func TestQuickCmd_WithMultiplier(t *testing.T) {
	code, stdout, stderr := executeCommand("quick", "720000", "1100", "jmeter", "-m", "4.0")
	if code != 0 {
		t.Fatalf("expected exit 0, got %d\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}
	if !strings.Contains(stdout, "pacing ×4.0") {
		t.Errorf("expected 'pacing ×4.0' in output, got: %s", stdout)
	}
	if !strings.Contains(stdout, "Pacing:") {
		t.Errorf("expected 'Pacing:' in output, got: %s", stdout)
	}
}

func TestQuickCmd_InvalidArgs_MissingArgs(t *testing.T) {
	code, _, _ := executeCommand("quick")
	if code != 1 {
		t.Errorf("expected exit 1 for missing args, got %d", code)
	}
}

func TestQuickCmd_InvalidArgs_BadTool(t *testing.T) {
	code, _, _ := executeCommand("quick", "720000", "1100", "gatling")
	if code != 1 {
		t.Errorf("expected exit 1 for bad tool, got %d", code)
	}
}

func TestQuickCmd_InvalidArgs_BadIntensity(t *testing.T) {
	code, _, _ := executeCommand("quick", "notanumber", "1100", "jmeter")
	if code != 1 {
		t.Errorf("expected exit 1 for bad intensity, got %d", code)
	}
}

func TestQuickCmd_InvalidArgs_BadScriptTime(t *testing.T) {
	code, _, _ := executeCommand("quick", "720000", "notanumber", "jmeter")
	if code != 1 {
		t.Errorf("expected exit 1 for bad script time, got %d", code)
	}
}
