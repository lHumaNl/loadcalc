package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestQuickModelDefaults(t *testing.T) {
	m := NewQuickModel()
	if m.intensity != "720000" {
		t.Errorf("expected default intensity 720000, got %s", m.intensity)
	}
	if m.scriptTime != "1100" {
		t.Errorf("expected default scriptTime 1100, got %s", m.scriptTime)
	}
	if m.tool != "jmeter" {
		t.Errorf("expected default tool jmeter, got %s", m.tool)
	}
	if m.model != "closed" {
		t.Errorf("expected default model closed, got %s", m.model)
	}
	if m.unit != "ops_h" {
		t.Errorf("expected default unit ops_h, got %s", m.unit)
	}
	if m.multiplier != "3.0" {
		t.Errorf("expected default multiplier 3.0, got %s", m.multiplier)
	}
	if m.generators != "1" {
		t.Errorf("expected default generators 1, got %s", m.generators)
	}
	if m.steps != "50,75,100,125,150" {
		t.Errorf("expected default steps, got %s", m.steps)
	}
	if m.rangeDown != "0.2" {
		t.Errorf("expected default rangeDown 0.2, got %s", m.rangeDown)
	}
	if m.rangeUp != "0.5" {
		t.Errorf("expected default rangeUp 0.5, got %s", m.rangeUp)
	}
	if m.rampup != "60" {
		t.Errorf("expected default rampup 60, got %s", m.rampup)
	}
	if m.activeField != 0 {
		t.Errorf("expected activeField 0, got %d", m.activeField)
	}
}

func TestQuickModelFieldNavigation(t *testing.T) {
	m := NewQuickModel()

	// Tab moves to next field
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	qm := updated.(QuickModel)
	if qm.activeField != 1 {
		t.Errorf("expected activeField 1 after Tab, got %d", qm.activeField)
	}

	// Tab again
	updated, _ = qm.Update(tea.KeyMsg{Type: tea.KeyTab})
	qm = updated.(QuickModel)
	if qm.activeField != 2 {
		t.Errorf("expected activeField 2 after second Tab, got %d", qm.activeField)
	}

	// Shift+Tab goes back
	updated, _ = qm.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	qm = updated.(QuickModel)
	if qm.activeField != 1 {
		t.Errorf("expected activeField 1 after Shift+Tab, got %d", qm.activeField)
	}

	// Tab wraps around
	m2 := NewQuickModel()
	m2.activeField = len(m2.fields) - 1
	updated, _ = m2.Update(tea.KeyMsg{Type: tea.KeyTab})
	qm = updated.(QuickModel)
	if qm.activeField != 0 {
		t.Errorf("expected activeField 0 after wrap, got %d", qm.activeField)
	}
}

func TestQuickModelCycleField(t *testing.T) {
	m := NewQuickModel()
	// Find the tool field index
	toolIdx := -1
	for i, f := range m.fields {
		if f.label == "Tool" {
			toolIdx = i
			break
		}
	}
	if toolIdx == -1 {
		t.Fatal("tool field not found")
	}
	m.activeField = toolIdx

	if m.tool != "jmeter" {
		t.Fatalf("expected initial tool jmeter, got %s", m.tool)
	}

	// Space cycles tool
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeySpace})
	qm := updated.(QuickModel)
	if qm.tool != "lre_pc" {
		t.Errorf("expected tool lre_pc after Space, got %s", qm.tool)
	}

	// Space again cycles back
	updated, _ = qm.Update(tea.KeyMsg{Type: tea.KeySpace})
	qm = updated.(QuickModel)
	if qm.tool != "jmeter" {
		t.Errorf("expected tool jmeter after second Space, got %s", qm.tool)
	}
}

func TestQuickModelTextInput(t *testing.T) {
	m := NewQuickModel()
	// activeField 0 is intensity (text field)
	if m.fields[0].label != "Intensity" {
		t.Fatalf("expected first field to be Intensity, got %s", m.fields[0].label)
	}

	// Clear the field first by selecting all and deleting
	m.intensity = ""

	// Type "500"
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'5'}})
	qm := updated.(QuickModel)
	updated, _ = qm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'0'}})
	qm = updated.(QuickModel)
	updated, _ = qm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'0'}})
	qm = updated.(QuickModel)

	if qm.intensity != "500" {
		t.Errorf("expected intensity 500, got %s", qm.intensity)
	}

	// Backspace deletes
	updated, _ = qm.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	qm = updated.(QuickModel)
	if qm.intensity != "50" {
		t.Errorf("expected intensity 50 after backspace, got %s", qm.intensity)
	}
}

func TestQuickModelValidInputProducesResults(t *testing.T) {
	m := NewQuickModel()
	// Default values should produce valid results
	m.recalculate()
	if m.resultText == "" {
		t.Error("expected non-empty resultText with valid defaults")
	}
	if m.err != "" {
		t.Errorf("expected no error with valid defaults, got: %s", m.err)
	}
}

func TestQuickModelInvalidInputShowsError(t *testing.T) {
	m := NewQuickModel()
	m.intensity = "abc"
	m.recalculate()
	if m.err == "" {
		t.Error("expected error with invalid intensity")
	}
	if m.resultText != "" {
		t.Error("expected empty resultText with invalid input")
	}
}

func TestQuickModelViewContainsTitle(t *testing.T) {
	m := NewQuickModel()
	m.recalculate()
	v := m.View()
	if !strings.Contains(v, "loadcalc quick calculator") {
		t.Error("expected View to contain title")
	}
}

func TestQuickModelQuit(t *testing.T) {
	m := NewQuickModel()
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Error("expected quit command on Ctrl+C")
	}
}
