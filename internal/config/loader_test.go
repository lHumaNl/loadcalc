package config

import (
	"os"
	"path/filepath"
	"testing"

	"loadcalc/pkg/units"

	"github.com/xuri/excelize/v2"
)

func TestLoadFromYAML_ValidConfig(t *testing.T) {
	plan, err := LoadFromYAML(filepath.Join("testdata", "valid_config.yaml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if plan.Version != "1.0" {
		t.Errorf("Version = %q, want %q", plan.Version, "1.0")
	}

	// Global defaults
	if plan.GlobalDefaults.Tool != ToolJMeter {
		t.Errorf("Tool = %v, want jmeter", plan.GlobalDefaults.Tool)
	}
	if plan.GlobalDefaults.LoadModel != LoadModelClosed {
		t.Errorf("LoadModel = %v, want closed", plan.GlobalDefaults.LoadModel)
	}
	if plan.GlobalDefaults.PacingMultiplier != 3.0 {
		t.Errorf("PacingMultiplier = %v, want 3.0", plan.GlobalDefaults.PacingMultiplier)
	}
	if plan.GlobalDefaults.DeviationTolerance != 2.5 {
		t.Errorf("DeviationTolerance = %v, want 2.5", plan.GlobalDefaults.DeviationTolerance)
	}
	if plan.GlobalDefaults.GeneratorsCount != 3 {
		t.Errorf("GeneratorsCount = %v, want 3", plan.GlobalDefaults.GeneratorsCount)
	}

	// Scenarios
	if len(plan.Scenarios) != 4 {
		t.Fatalf("expected 4 scenarios, got %d", len(plan.Scenarios))
	}

	uc01 := plan.Scenarios[0]
	if uc01.Name != "Main page" {
		t.Errorf("Scenario 0 Name = %q, want Main page", uc01.Name)
	}
	if uc01.TargetIntensity != 720000 {
		t.Errorf("UC01 TargetIntensity = %v, want 720000", uc01.TargetIntensity)
	}
	if uc01.IntensityUnit != units.OpsPerHour {
		t.Errorf("UC01 IntensityUnit = %v, want ops_h", uc01.IntensityUnit)
	}
	if uc01.MaxScriptTimeMs != 1100 {
		t.Errorf("UC01 MaxScriptTimeMs = %v, want 1100", uc01.MaxScriptTimeMs)
	}
	// No overrides set
	if uc01.PacingMultiplier != nil {
		t.Error("UC01 PacingMultiplier should be nil (no override)")
	}

	uc02 := plan.Scenarios[1]
	if uc02.PacingMultiplier == nil || *uc02.PacingMultiplier != 4.0 {
		t.Errorf("UC02 PacingMultiplier should be 4.0")
	}

	uc03 := plan.Scenarios[2]
	if !uc03.Background {
		t.Error("UC03 should be background")
	}
	if uc03.BackgroundPercent != 100 {
		t.Errorf("UC03 BackgroundPercent = %v, want 100", uc03.BackgroundPercent)
	}

	uc04 := plan.Scenarios[3]
	if uc04.LoadModel == nil || *uc04.LoadModel != LoadModelOpen {
		t.Error("UC04 LoadModel should be open")
	}
	if uc04.SpikeParticipate == nil || *uc04.SpikeParticipate != false {
		t.Error("UC04 SpikeParticipate should be false")
	}

	// Profile
	if plan.Profile.Type != ProfileMaxSearch {
		t.Errorf("Profile.Type = %v, want max_search", plan.Profile.Type)
	}
	if plan.Profile.StartPercent != 50 {
		t.Errorf("StartPercent = %v, want 50", plan.Profile.StartPercent)
	}
	if plan.Profile.StepIncrement != 25 {
		t.Errorf("StepIncrement = %v, want 25", plan.Profile.StepIncrement)
	}
	if plan.Profile.NumSteps != 5 {
		t.Errorf("NumSteps = %v, want 5", plan.Profile.NumSteps)
	}
	if len(plan.Profile.Steps) != 2 {
		t.Fatalf("expected 2 step overrides, got %d", len(plan.Profile.Steps))
	}
	if plan.Profile.Steps[0].PercentOfTarget != 50 {
		t.Errorf("Step 0 percent = %v, want 50", plan.Profile.Steps[0].PercentOfTarget)
	}
	if plan.Profile.Steps[0].StabilitySec == nil || *plan.Profile.Steps[0].StabilitySec != 600 {
		t.Error("Step 0 stability_sec should be 600")
	}
}

func TestLoadFromYAML_FileNotFound(t *testing.T) {
	_, err := LoadFromYAML("nonexistent.yaml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoadFromYAML_InvalidYAML(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "bad.yaml")
	os.WriteFile(tmp, []byte("{{invalid yaml"), 0o644)
	_, err := LoadFromYAML(tmp)
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestLoadFromYAML_DefaultVersion(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "no_version.yaml")
	os.WriteFile(tmp, []byte(`
global:
  tool: jmeter
scenarios:
  - name: "test"
    target_intensity: 100
    intensity_unit: ops_h
    max_script_time_ms: 500
profile:
  type: stable
  percent: 100
`), 0o644)
	plan, err := LoadFromYAML(tmp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if plan.Version != "1.0" {
		t.Errorf("Version = %q, want default 1.0", plan.Version)
	}
}

func TestLoadFromYAML_MissingGlobalFields(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "minimal.yaml")
	os.WriteFile(tmp, []byte(`
scenarios:
  - name: "test"
    target_intensity: 100
    intensity_unit: ops_h
    max_script_time_ms: 500
profile:
  type: stable
  percent: 100
`), 0o644)
	plan, err := LoadFromYAML(tmp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Global defaults are zero-valued before ResolveDefaults
	if plan.GlobalDefaults.Tool != "" {
		t.Error("Tool should be empty before ResolveDefaults")
	}
}

func TestLoadScenariosFromCSV(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "scenarios.csv")
	os.WriteFile(tmp, []byte("name;script_id;target_intensity;intensity_unit;max_script_time_ms;background;background_percent\nLogin;1;3600;ops_h;5000;false;0\nBrowse;2;600;ops_m;1200;true;50\n"), 0o644)

	scenarios, err := LoadScenariosFromCSV(tmp, ';')
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(scenarios) != 2 {
		t.Fatalf("expected 2 scenarios, got %d", len(scenarios))
	}
	if scenarios[0].Name != "Login" {
		t.Errorf("scenario 0 name = %q, want Login", scenarios[0].Name)
	}
	if scenarios[0].ScriptID != 1 {
		t.Errorf("scenario 0 script_id = %d, want 1", scenarios[0].ScriptID)
	}
	if scenarios[0].TargetIntensity != 3600 {
		t.Errorf("scenario 0 target_intensity = %v, want 3600", scenarios[0].TargetIntensity)
	}
	if scenarios[0].IntensityUnit != units.OpsPerHour {
		t.Errorf("scenario 0 intensity_unit = %v, want ops_h", scenarios[0].IntensityUnit)
	}
	if scenarios[1].Background != true {
		t.Error("scenario 1 should be background")
	}
	if scenarios[1].BackgroundPercent != 50 {
		t.Errorf("scenario 1 background_percent = %v, want 50", scenarios[1].BackgroundPercent)
	}
}

func TestLoadScenariosFromCSV_WithOverrides(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "scenarios.csv")
	os.WriteFile(tmp, []byte("name;script_id;target_intensity;intensity_unit;max_script_time_ms;pacing_multiplier;load_model;spike_participate\nTest;1;1000;ops_h;500;4.0;open;false\n"), 0o644)

	scenarios, err := LoadScenariosFromCSV(tmp, ';')
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(scenarios) != 1 {
		t.Fatalf("expected 1 scenario, got %d", len(scenarios))
	}
	s := scenarios[0]
	if s.PacingMultiplier == nil || *s.PacingMultiplier != 4.0 {
		t.Error("pacing_multiplier should be 4.0")
	}
	if s.LoadModel == nil || *s.LoadModel != LoadModelOpen {
		t.Error("load_model should be open")
	}
	if s.SpikeParticipate == nil || *s.SpikeParticipate != false {
		t.Error("spike_participate should be false")
	}
}

func TestLoadScenariosFromCSV_EmptyFile(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "empty.csv")
	os.WriteFile(tmp, []byte(""), 0o644)

	scenarios, err := LoadScenariosFromCSV(tmp, ';')
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if scenarios != nil {
		t.Errorf("expected nil scenarios for empty CSV, got %d", len(scenarios))
	}
}

func TestLoadScenariosFromCSV_CommaDelimiter(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "scenarios.csv")
	os.WriteFile(tmp, []byte("name,target_intensity,intensity_unit,max_script_time_ms\nTest,1000,ops_h,500\n"), 0o644)

	scenarios, err := LoadScenariosFromCSV(tmp, ',')
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(scenarios) != 1 {
		t.Fatalf("expected 1 scenario, got %d", len(scenarios))
	}
	if scenarios[0].Name != "Test" {
		t.Errorf("name = %q, want Test", scenarios[0].Name)
	}
}

func TestLoadScenariosFromDir(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "01_first.csv"), []byte("name;target_intensity;intensity_unit;max_script_time_ms\nFirst;1000;ops_h;500\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "02_second.csv"), []byte("name;target_intensity;intensity_unit;max_script_time_ms\nSecond;2000;ops_h;600\n"), 0o644)

	scenarios, err := LoadScenariosFromDir(dir, ';')
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(scenarios) != 2 {
		t.Fatalf("expected 2 scenarios, got %d", len(scenarios))
	}
	if scenarios[0].Name != "First" {
		t.Errorf("scenario 0 name = %q, want First", scenarios[0].Name)
	}
	if scenarios[1].Name != "Second" {
		t.Errorf("scenario 1 name = %q, want Second", scenarios[1].Name)
	}
}

func TestLoadScenariosFromDir_Empty(t *testing.T) {
	dir := t.TempDir()
	scenarios, err := LoadScenariosFromDir(dir, ';')
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if scenarios != nil {
		t.Errorf("expected nil scenarios for empty dir, got %d", len(scenarios))
	}
}

func TestScenarioConcatenation_YAML_Plus_CSV(t *testing.T) {
	// Create YAML config with one scenario
	yamlPath := filepath.Join(t.TempDir(), "config.yaml")
	os.WriteFile(yamlPath, []byte(`
version: "1.0"
global:
  tool: jmeter
  load_model: closed
scenarios:
  - name: "YAML Scenario"
    target_intensity: 1000
    intensity_unit: ops_h
    max_script_time_ms: 500
profile:
  type: stable
  percent: 100
`), 0o644)

	// Create CSV with one scenario
	csvPath := filepath.Join(t.TempDir(), "extra.csv")
	os.WriteFile(csvPath, []byte("name;target_intensity;intensity_unit;max_script_time_ms\nCSV Scenario;2000;ops_h;600\n"), 0o644)

	// Load YAML
	plan, err := LoadFromYAML(yamlPath)
	if err != nil {
		t.Fatalf("load YAML: %v", err)
	}

	// Load CSV and concatenate
	csvScenarios, err := LoadScenariosFromCSV(csvPath, ';')
	if err != nil {
		t.Fatalf("load CSV: %v", err)
	}
	plan.Scenarios = append(plan.Scenarios, csvScenarios...)

	if len(plan.Scenarios) != 2 {
		t.Fatalf("expected 2 scenarios, got %d", len(plan.Scenarios))
	}
	if plan.Scenarios[0].Name != "YAML Scenario" {
		t.Errorf("scenario 0 = %q, want YAML Scenario", plan.Scenarios[0].Name)
	}
	if plan.Scenarios[1].Name != "CSV Scenario" {
		t.Errorf("scenario 1 = %q, want CSV Scenario", plan.Scenarios[1].Name)
	}
}

func TestLoadScenariosFromXLSX(t *testing.T) {
	xlsxPath := filepath.Join(t.TempDir(), "scenarios.xlsx")

	// Create test XLSX with excelize
	ef := excelize.NewFile()
	defer ef.Close()
	ef.SetCellValue("Sheet1", "A1", "name")
	ef.SetCellValue("Sheet1", "B1", "script_id")
	ef.SetCellValue("Sheet1", "C1", "target_intensity")
	ef.SetCellValue("Sheet1", "D1", "intensity_unit")
	ef.SetCellValue("Sheet1", "E1", "max_script_time_ms")
	ef.SetCellValue("Sheet1", "A2", "XLSX Test")
	ef.SetCellValue("Sheet1", "B2", 5)
	ef.SetCellValue("Sheet1", "C2", 5000)
	ef.SetCellValue("Sheet1", "D2", "ops_h")
	ef.SetCellValue("Sheet1", "E2", 800)
	if err := ef.SaveAs(xlsxPath); err != nil {
		t.Fatalf("saving test XLSX: %v", err)
	}

	scenarios, err := LoadScenariosFromXLSX(xlsxPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(scenarios) != 1 {
		t.Fatalf("expected 1 scenario, got %d", len(scenarios))
	}
	if scenarios[0].Name != "XLSX Test" {
		t.Errorf("name = %q, want XLSX Test", scenarios[0].Name)
	}
	if scenarios[0].ScriptID != 5 {
		t.Errorf("script_id = %d, want 5", scenarios[0].ScriptID)
	}
	if scenarios[0].TargetIntensity != 5000 {
		t.Errorf("target_intensity = %v, want 5000", scenarios[0].TargetIntensity)
	}
}
