package tui

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"loadcalc/internal/config"
	"loadcalc/internal/engine"
	"loadcalc/internal/profile"
	"loadcalc/internal/tui/views"
)

func testResults() engine.CalculationResults {
	plan := &config.TestPlan{
		GlobalDefaults: config.GlobalDefaults{
			Tool:               config.ToolJMeter,
			LoadModel:          config.LoadModelClosed,
			PacingMultiplier:   3.0,
			DeviationTolerance: 2.5,
			GeneratorsCount:    1,
		},
	}
	return engine.CalculationResults{
		Plan: plan,
		ScenarioResults: []engine.ScenarioResult{
			{
				Scenario: config.Scenario{Name: "Login"},
				OptimizeResult: engine.OptimizeResult{
					BestPacingMS:       3300.0,
					MaxDeviationPct:    0.5,
					AllWithinTolerance: true,
					StepResults: []engine.StepResult{
						{Step: profile.Step{StepNumber: 1, PercentOfTarget: 100, RampupSec: 60, ImpactSec: 120, StabilitySec: 300}, TargetRPS: 10.0, Threads: 33, ActualRPS: 10.0, DeviationPct: 0.0},
					},
				},
			},
			{
				Scenario: config.Scenario{Name: "Search"},
				OptimizeResult: engine.OptimizeResult{
					BestPacingMS:       5000.0,
					MaxDeviationPct:    1.8,
					AllWithinTolerance: true,
					StepResults: []engine.StepResult{
						{Step: profile.Step{StepNumber: 1, PercentOfTarget: 100}, TargetRPS: 5.0, Threads: 25, ActualRPS: 5.0, DeviationPct: 1.8},
					},
				},
			},
		},
		Timeline: []profile.TimelinePhase{
			{PhaseName: "Rampup to 100%", StartTimeSec: 0, DurationSec: 60, EndTimeSec: 60, PercentOfTarget: 100},
			{PhaseName: "Stability 100%", StartTimeSec: 60, DurationSec: 300, EndTimeSec: 360, PercentOfTarget: 100},
		},
	}
}

func TestNewModel(t *testing.T) {
	m := NewModel(testResults(), "out.xlsx")
	if m.CurrentView != ViewSummary {
		t.Errorf("initial view should be ViewSummary, got %v", m.CurrentView)
	}
	if m.SelectedRow != 0 {
		t.Errorf("initial selected row should be 0, got %d", m.SelectedRow)
	}
	if m.ColumnGroup != views.ColumnGroupBase {
		t.Errorf("initial column group should be Base")
	}
}

func TestNavigateDown(t *testing.T) {
	m := NewModel(testResults(), "")
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	model := updated.(Model)
	if model.SelectedRow != 1 {
		t.Errorf("expected row 1 after down, got %d", model.SelectedRow)
	}
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	model = updated.(Model)
	if model.SelectedRow != 1 {
		t.Errorf("expected row 1 (clamped), got %d", model.SelectedRow)
	}
}

func TestNavigateUp(t *testing.T) {
	m := NewModel(testResults(), "")
	m.SelectedRow = 1
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	model := updated.(Model)
	if model.SelectedRow != 0 {
		t.Errorf("expected row 0 after up, got %d", model.SelectedRow)
	}
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	model = updated.(Model)
	if model.SelectedRow != 0 {
		t.Errorf("expected row 0 (clamped), got %d", model.SelectedRow)
	}
}

func TestTabSwitchesColumnGroup(t *testing.T) {
	m := NewModel(testResults(), "")
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	model := updated.(Model)
	if model.ColumnGroup != views.ColumnGroupTool {
		t.Errorf("expected Tool column group after Tab")
	}
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyTab})
	model = updated.(Model)
	if model.ColumnGroup != views.ColumnGroupBase {
		t.Errorf("expected Base column group after second Tab")
	}
}

func TestEnterDrillsIntoScenario(t *testing.T) {
	m := NewModel(testResults(), "")
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model := updated.(Model)
	if model.CurrentView != ViewScenario {
		t.Errorf("expected ViewScenario after Enter, got %v", model.CurrentView)
	}
}

func TestEscGoesBack(t *testing.T) {
	m := NewModel(testResults(), "")
	m.CurrentView = ViewScenario
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model := updated.(Model)
	if model.CurrentView != ViewSummary {
		t.Errorf("expected ViewSummary after Esc, got %v", model.CurrentView)
	}
}

func TestSKeyGoesToSteps(t *testing.T) {
	m := NewModel(testResults(), "")
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	model := updated.(Model)
	if model.CurrentView != ViewSteps {
		t.Errorf("expected ViewSteps, got %v", model.CurrentView)
	}
}

func TestTKeyGoesToTimeline(t *testing.T) {
	m := NewModel(testResults(), "")
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("t")})
	model := updated.(Model)
	if model.CurrentView != ViewTimeline {
		t.Errorf("expected ViewTimeline, got %v", model.CurrentView)
	}
}

func TestViewRendering(t *testing.T) {
	m := NewModel(testResults(), "out.xlsx")

	output := m.View()
	assertContains(t, output, "Login")
	assertContains(t, output, "Search")
	assertContains(t, output, "Load Test Summary")

	m.CurrentView = ViewSteps
	output = m.View()
	assertContains(t, output, "Steps: Login")
	assertContains(t, output, "TargetRPS")

	m.CurrentView = ViewScenario
	output = m.View()
	assertContains(t, output, "Scenario: Login")
	assertContains(t, output, "Load Model:")

	m.CurrentView = ViewTimeline
	output = m.View()
	assertContains(t, output, "Test Timeline")
	assertContains(t, output, "Rampup to 100%")
}

func TestExportNoPath(t *testing.T) {
	m := NewModel(testResults(), "")
	cmd := m.exportXLSX()
	msg := cmd()
	exportMsg, ok := msg.(ExportMsg)
	if !ok {
		t.Fatal("expected ExportMsg")
	}
	if exportMsg.Err == nil {
		t.Error("expected error when no output path")
	}
}

func TestArrowKeysInStepsView(t *testing.T) {
	m := NewModel(testResults(), "")
	m.CurrentView = ViewSteps
	m.SelectedRow = 0

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRight})
	model := updated.(Model)
	if model.SelectedRow != 1 {
		t.Errorf("expected row 1 after right arrow, got %d", model.SelectedRow)
	}

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyLeft})
	model = updated.(Model)
	if model.SelectedRow != 0 {
		t.Errorf("expected row 0 after left arrow, got %d", model.SelectedRow)
	}
}

func TestExportMsgSuccess(t *testing.T) {
	m := NewModel(testResults(), "/tmp/test.xlsx")
	updated, _ := m.Update(ExportMsg{Err: nil})
	model := updated.(Model)
	if !strings.Contains(model.StatusMsg, "Exported to") {
		t.Errorf("expected success status, got %q", model.StatusMsg)
	}
}

func TestExportMsgError(t *testing.T) {
	m := NewModel(testResults(), "/tmp/test.xlsx")
	updated, _ := m.Update(ExportMsg{Err: fmt.Errorf("disk full")})
	model := updated.(Model)
	if !strings.Contains(model.StatusMsg, "Export failed") {
		t.Errorf("expected failure status, got %q", model.StatusMsg)
	}
}

func TestStepsViewOutOfBounds(t *testing.T) {
	m := NewModel(testResults(), "")
	m.CurrentView = ViewSteps
	m.SelectedRow = -1
	output := m.View()
	assertContains(t, output, "No scenario selected")
}

func TestScenarioViewOutOfBounds(t *testing.T) {
	m := NewModel(testResults(), "")
	m.CurrentView = ViewScenario
	m.SelectedRow = 99
	output := m.View()
	assertContains(t, output, "No scenario selected")
}

func TestEmptyTimeline(t *testing.T) {
	results := testResults()
	results.Timeline = nil
	m := NewModel(results, "")
	m.CurrentView = ViewTimeline
	output := m.View()
	assertContains(t, output, "No timeline phases available")
}

func assertContains(t *testing.T, s, substr string) {
	t.Helper()
	clean := stripANSI(s)
	if !strings.Contains(clean, substr) {
		t.Errorf("output does not contain %q\noutput (cleaned): %s", substr, clean)
	}
}

func stripANSI(s string) string {
	var result []byte
	i := 0
	for i < len(s) {
		if s[i] == '\x1b' && i+1 < len(s) && s[i+1] == '[' {
			j := i + 2
			for j < len(s) && s[j] != 'm' {
				j++
			}
			if j < len(s) {
				j++
			}
			i = j
		} else {
			result = append(result, s[i])
			i++
		}
	}
	return string(result)
}
