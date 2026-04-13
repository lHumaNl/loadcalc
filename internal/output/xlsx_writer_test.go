package output

import (
	"os"
	"path/filepath"
	"testing"

	"loadcalc/internal/config"
	"loadcalc/internal/engine"
	"loadcalc/internal/profile"
	"loadcalc/pkg/units"

	"github.com/xuri/excelize/v2"
)

func testResults(tool config.Tool) engine.CalculationResults {
	tolerance := 2.5
	spikeP := true
	return engine.CalculationResults{
		Plan: &config.TestPlan{
			Version: "1.0",
			GlobalDefaults: config.GlobalDefaults{
				Tool:               tool,
				LoadModel:          config.LoadModelClosed,
				PacingMultiplier:   3.0,
				DeviationTolerance: 2.5,
				SpikeParticipate:   true,
				GeneratorsCount:    2,
			},
			Scenarios: []config.Scenario{
				{
					ScriptID:           1,
					Name:               "Login",
					TargetIntensity:    3600,
					IntensityUnit:      units.OpsPerHour,
					MaxScriptTimeMs:    5000,
					Background:         false,
					DeviationTolerance: &tolerance,
					SpikeParticipate:   &spikeP,
				},
				{
					ScriptID:          2,
					Name:              "Browse",
					TargetIntensity:   60,
					IntensityUnit:     units.OpsPerMinute,
					MaxScriptTimeMs:   3000,
					Background:        true,
					BackgroundPercent: 50,
				},
			},
			Profile: config.TestProfile{
				Type:                config.ProfileCapacity,
				DefaultRampupSec:    60,
				DefaultImpactSec:    300,
				DefaultStabilitySec: 120,
				DefaultRampdownSec:  0,
			},
			OutputFormat: "xlsx",
		},
		ScenarioResults: []engine.ScenarioResult{
			{
				Scenario: config.Scenario{
					ScriptID:           1,
					Name:               "Login",
					TargetIntensity:    3600,
					IntensityUnit:      units.OpsPerHour,
					MaxScriptTimeMs:    5000,
					DeviationTolerance: &tolerance,
					SpikeParticipate:   &spikeP,
				},
				OptimizeResult: engine.OptimizeResult{
					BestPacingMS:           15000,
					BestOpsPerMinPerThread: 4.0,
					StepResults: []engine.StepResult{
						{
							Step:         profile.Step{StepNumber: 1, PercentOfTarget: 50, RampupSec: 60, ImpactSec: 300, StabilitySec: 120, RampdownSec: 0},
							TargetRPS:    0.5,
							Threads:      8,
							ActualRPS:    0.533,
							DeviationPct: 6.67,
						},
						{
							Step:         profile.Step{StepNumber: 2, PercentOfTarget: 100, RampupSec: 60, ImpactSec: 300, StabilitySec: 120, RampdownSec: 30},
							TargetRPS:    1.0,
							Threads:      15,
							ActualRPS:    1.0,
							DeviationPct: 0.0,
						},
					},
					MaxDeviationPct:    6.67,
					AllWithinTolerance: false,
				},
				IsBackground: false,
				IsOpenModel:  false,
			},
			{
				Scenario: config.Scenario{
					ScriptID:          2,
					Name:              "Browse",
					TargetIntensity:   60,
					IntensityUnit:     units.OpsPerMinute,
					MaxScriptTimeMs:   3000,
					Background:        true,
					BackgroundPercent: 50,
				},
				OptimizeResult: engine.OptimizeResult{
					BestPacingMS:           9000,
					BestOpsPerMinPerThread: 6.67,
					StepResults: []engine.StepResult{
						{
							Step:         profile.Step{StepNumber: 1, PercentOfTarget: 50, RampupSec: 60, ImpactSec: 300, StabilitySec: 120, RampdownSec: 0},
							TargetRPS:    0.5,
							Threads:      5,
							ActualRPS:    0.556,
							DeviationPct: 1.11,
						},
					},
					MaxDeviationPct:    1.11,
					AllWithinTolerance: true,
				},
				IsBackground: true,
				IsOpenModel:  false,
			},
		},
		Steps: []profile.Step{
			{StepNumber: 1, PercentOfTarget: 50, RampupSec: 60, ImpactSec: 300, StabilitySec: 120, RampdownSec: 0},
			{StepNumber: 2, PercentOfTarget: 100, RampupSec: 60, ImpactSec: 300, StabilitySec: 120, RampdownSec: 30},
		},
		Timeline: []profile.TimelinePhase{
			{PhaseName: "Step 1 Rampup", StartTimeSec: 0, DurationSec: 60, EndTimeSec: 60, PercentOfTarget: 50},
			{PhaseName: "Step 1 Impact", StartTimeSec: 60, DurationSec: 300, EndTimeSec: 360, PercentOfTarget: 50},
			{PhaseName: "Step 1 Stability", StartTimeSec: 360, DurationSec: 120, EndTimeSec: 480, PercentOfTarget: 50},
			{PhaseName: "Step 2 Rampup", StartTimeSec: 480, DurationSec: 60, EndTimeSec: 540, PercentOfTarget: 100},
			{PhaseName: "Step 2 Impact", StartTimeSec: 540, DurationSec: 300, EndTimeSec: 840, PercentOfTarget: 100},
			{PhaseName: "Step 2 Stability", StartTimeSec: 840, DurationSec: 120, EndTimeSec: 960, PercentOfTarget: 100},
			{PhaseName: "Step 2 Rampdown", StartTimeSec: 960, DurationSec: 30, EndTimeSec: 990, PercentOfTarget: 100},
		},
	}
}

func TestXLSXWriter_Write_SummarySheet(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "test.xlsx")
	w := &XLSXWriter{}
	results := testResults(config.ToolLREPC)

	if err := w.Write(results, dest); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	f, err := excelize.OpenFile(dest)
	if err != nil {
		t.Fatalf("open xlsx: %v", err)
	}
	defer f.Close()

	// Check Summary sheet exists
	rows, err := f.GetRows("Summary")
	if err != nil {
		t.Fatalf("get Summary rows: %v", err)
	}
	// Header + 2 scenario rows
	if len(rows) < 3 {
		t.Fatalf("Summary: got %d rows, want at least 3", len(rows))
	}
	// Check header
	if rows[0][0] != "Scenario Name" {
		t.Errorf("Summary header[0] = %q, want Scenario Name", rows[0][0])
	}
	// Check first scenario name
	if rows[1][0] != "Login" {
		t.Errorf("Summary row1[0] = %q, want Login", rows[1][0])
	}
	if rows[2][0] != "Browse" {
		t.Errorf("Summary row2[0] = %q, want Browse", rows[2][0])
	}
}

func TestXLSXWriter_Write_StepsDetailSheet(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "test.xlsx")
	w := &XLSXWriter{}
	results := testResults(config.ToolLREPC)

	if err := w.Write(results, dest); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	f, err := excelize.OpenFile(dest)
	if err != nil {
		t.Fatalf("open xlsx: %v", err)
	}
	defer f.Close()

	rows, err := f.GetRows("Steps Detail")
	if err != nil {
		t.Fatalf("get Steps Detail rows: %v", err)
	}
	// Header + 3 step rows (2 for sc1 + 1 for sc2)
	if len(rows) < 4 {
		t.Fatalf("Steps Detail: got %d rows, want at least 4", len(rows))
	}
	if rows[0][0] != "Step #" {
		t.Errorf("Steps Detail header[0] = %q, want 'Step #'", rows[0][0])
	}
}

func TestXLSXWriter_Write_TimelineSheet(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "test.xlsx")
	w := &XLSXWriter{}
	results := testResults(config.ToolLREPC)

	if err := w.Write(results, dest); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	f, err := excelize.OpenFile(dest)
	if err != nil {
		t.Fatalf("open xlsx: %v", err)
	}
	defer f.Close()

	rows, err := f.GetRows("Timeline")
	if err != nil {
		t.Fatalf("get Timeline rows: %v", err)
	}
	// Header + 7 timeline phases
	if len(rows) < 8 {
		t.Fatalf("Timeline: got %d rows, want at least 8", len(rows))
	}
	if rows[0][0] != "Phase" {
		t.Errorf("Timeline header[0] = %q, want Phase", rows[0][0])
	}
	// Chart verification: we can't easily read charts back, but we verify
	// the file is valid and the timeline data is present.
}

func TestXLSXWriter_Write_InputParametersSheet(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "test.xlsx")
	w := &XLSXWriter{}
	results := testResults(config.ToolLREPC)

	if err := w.Write(results, dest); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	f, err := excelize.OpenFile(dest)
	if err != nil {
		t.Fatalf("open xlsx: %v", err)
	}
	defer f.Close()

	rows, err := f.GetRows("Input Parameters")
	if err != nil {
		t.Fatalf("get Input Parameters rows: %v", err)
	}
	if len(rows) < 2 {
		t.Fatalf("Input Parameters: got %d rows, want at least 2", len(rows))
	}
}

func TestXLSXWriter_JMeterConfigSheet_Present(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "test.xlsx")
	w := &XLSXWriter{}
	results := testResults(config.ToolJMeter)

	if err := w.Write(results, dest); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	f, err := excelize.OpenFile(dest)
	if err != nil {
		t.Fatalf("open xlsx: %v", err)
	}
	defer f.Close()

	rows, err := f.GetRows("JMeter Config")
	if err != nil {
		t.Fatalf("JMeter Config sheet should exist for jmeter tool: %v", err)
	}
	// Header + 2 scenarios
	if len(rows) < 3 {
		t.Fatalf("JMeter Config: got %d rows, want at least 3", len(rows))
	}
}

func TestXLSXWriter_JMeterConfigSheet_Absent_ForLRE(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "test.xlsx")
	w := &XLSXWriter{}
	results := testResults(config.ToolLREPC)

	if err := w.Write(results, dest); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	f, err := excelize.OpenFile(dest)
	if err != nil {
		t.Fatalf("open xlsx: %v", err)
	}
	defer f.Close()

	_, err = f.GetRows("JMeter Config")
	if err == nil {
		t.Error("JMeter Config sheet should NOT exist for lre_pc tool")
	}
}

func TestXLSXWriter_LREPCConfigSheet_Present(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "test.xlsx")
	w := &XLSXWriter{}
	results := testResults(config.ToolLREPC)

	if err := w.Write(results, dest); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	f, err := excelize.OpenFile(dest)
	if err != nil {
		t.Fatalf("open xlsx: %v", err)
	}
	defer f.Close()

	rows, err := f.GetRows("LRE PC Config")
	if err != nil {
		t.Fatalf("LRE PC Config sheet should exist for lre_pc tool: %v", err)
	}
	// Header + 2 steps for Login + 1 row for Browse (background) = 4 rows
	if len(rows) < 4 {
		t.Fatalf("LRE PC Config: got %d rows, want at least 4", len(rows))
	}
	// Check headers
	if rows[0][0] != "Scenario" {
		t.Errorf("header[0] = %q, want Scenario", rows[0][0])
	}
	if rows[0][5] != "Batch Size" {
		t.Errorf("header[5] = %q, want Batch Size", rows[0][5])
	}
	// First data row should be Login
	if rows[1][0] != "Login" {
		t.Errorf("row1[0] = %q, want Login", rows[1][0])
	}
}

func TestXLSXWriter_LREPCConfigSheet_Absent_ForJMeter(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "test.xlsx")
	w := &XLSXWriter{}
	results := testResults(config.ToolJMeter)

	if err := w.Write(results, dest); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	f, err := excelize.OpenFile(dest)
	if err != nil {
		t.Fatalf("open xlsx: %v", err)
	}
	defer f.Close()

	_, err = f.GetRows("LRE PC Config")
	if err == nil {
		t.Error("LRE PC Config sheet should NOT exist for jmeter tool")
	}
}

func TestXLSXWriter_JMeterConfigSheet_PerStepData(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "test.xlsx")
	w := &XLSXWriter{}
	results := testResults(config.ToolJMeter)

	if err := w.Write(results, dest); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	f, err := excelize.OpenFile(dest)
	if err != nil {
		t.Fatalf("open xlsx: %v", err)
	}
	defer f.Close()

	rows, err := f.GetRows("JMeter Config")
	if err != nil {
		t.Fatalf("JMeter Config sheet should exist: %v", err)
	}
	// Header row should have per-step columns
	if rows[0][0] != "Scenario" {
		t.Errorf("header[0] = %q, want Scenario", rows[0][0])
	}
	if rows[0][1] != "Step" {
		t.Errorf("header[1] = %q, want Step", rows[0][1])
	}
	if rows[0][4] != "Threads Delta" {
		t.Errorf("header[4] = %q, want 'Threads Delta'", rows[0][4])
	}
	// Should have per-step rows: Login 2 steps + Browse 1 BG row = 3 data rows
	if len(rows) < 4 {
		t.Fatalf("JMeter Config: got %d rows, want at least 4", len(rows))
	}
}

func TestXLSXWriter_ConditionalFormatting(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "test.xlsx")
	w := &XLSXWriter{}
	results := testResults(config.ToolLREPC)

	if err := w.Write(results, dest); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Verify file is valid and can be opened
	f, err := excelize.OpenFile(dest)
	if err != nil {
		t.Fatalf("open xlsx: %v", err)
	}
	defer f.Close()

	// Check that the file exists and has content
	info, err := os.Stat(dest)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Size() == 0 {
		t.Error("output file is empty")
	}
}

func TestFormatIntensity(t *testing.T) {
	tests := []struct {
		value float64
		unit  units.IntensityUnit
		want  string
	}{
		{3600, units.OpsPerHour, "3600.00 ops/h"},
		{60, units.OpsPerMinute, "60.00 ops/m"},
		{1.5, units.OpsPerSecond, "1.50 ops/s"},
	}
	for _, tt := range tests {
		got := FormatIntensity(tt.value, tt.unit)
		if got != tt.want {
			t.Errorf("FormatIntensity(%v, %v) = %q, want %q", tt.value, tt.unit, got, tt.want)
		}
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		sec  int
		want string
	}{
		{30, "30s"},
		{60, "1m"},
		{90, "1m30s"},
	}
	for _, tt := range tests {
		got := FormatDuration(tt.sec)
		if got != tt.want {
			t.Errorf("FormatDuration(%d) = %q, want %q", tt.sec, got, tt.want)
		}
	}
}

func TestFormatPercent(t *testing.T) {
	got := FormatPercent(6.67)
	if got != "6.67%" {
		t.Errorf("FormatPercent(6.67) = %q, want 6.67%%", got)
	}
}
