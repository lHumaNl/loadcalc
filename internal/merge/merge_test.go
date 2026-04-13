package merge

import (
	"os"
	"path/filepath"
	"testing"

	"loadcalc/internal/config"
)

func TestMergePlans_SingleConfig(t *testing.T) {
	plan := &config.TestPlan{
		Version: "1.0",
		GlobalDefaults: config.GlobalDefaults{
			Tool:             config.ToolJMeter,
			LoadModel:        config.LoadModelClosed,
			PacingMultiplier: 3.0,
		},
		Scenarios: []config.Scenario{
			{Name: "S1", ScriptID: 1},
		},
		Profile: config.TestProfile{Type: config.ProfileStable},
	}

	result := Plans([]*config.TestPlan{plan})

	if len(result.Warnings) != 0 {
		t.Errorf("expected no warnings, got %d", len(result.Warnings))
	}
	if len(result.Plan.Scenarios) != 1 {
		t.Errorf("expected 1 scenario, got %d", len(result.Plan.Scenarios))
	}
	if result.Plan.GlobalDefaults.PacingMultiplier != 3.0 {
		t.Errorf("expected pacing_multiplier 3.0, got %f", result.Plan.GlobalDefaults.PacingMultiplier)
	}
}

func TestMergePlans_TwoConfigs_NoConflicts(t *testing.T) {
	plan1 := &config.TestPlan{
		Version: "1.0",
		GlobalDefaults: config.GlobalDefaults{
			Tool:             config.ToolJMeter,
			LoadModel:        config.LoadModelClosed,
			PacingMultiplier: 3.0,
		},
		Scenarios: []config.Scenario{
			{Name: "S1"},
			{Name: "S2"},
		},
		Profile: config.TestProfile{Type: config.ProfileStable},
	}
	plan2 := &config.TestPlan{
		Version: "1.0",
		GlobalDefaults: config.GlobalDefaults{
			Tool:             config.ToolJMeter,
			LoadModel:        config.LoadModelClosed,
			PacingMultiplier: 3.0,
		},
		Scenarios: []config.Scenario{
			{Name: "S3"},
		},
		Profile: config.TestProfile{Type: config.ProfileStable},
	}

	result := Plans([]*config.TestPlan{plan1, plan2})

	if len(result.Warnings) != 0 {
		t.Errorf("expected no warnings, got %d: %v", len(result.Warnings), result.Warnings)
	}
	if len(result.Plan.Scenarios) != 3 {
		t.Fatalf("expected 3 scenarios, got %d", len(result.Plan.Scenarios))
	}
	if result.Plan.Scenarios[0].Name != "S1" || result.Plan.Scenarios[2].Name != "S3" {
		t.Error("scenarios not in expected order")
	}
}

func TestMergePlans_ConflictingGlobals(t *testing.T) {
	plan1 := &config.TestPlan{
		GlobalDefaults: config.GlobalDefaults{
			Tool:             config.ToolJMeter,
			PacingMultiplier: 3.0,
			GeneratorsCount:  2,
		},
		Scenarios: []config.Scenario{{Name: "S1"}},
		Profile:   config.TestProfile{Type: config.ProfileStable},
	}
	plan2 := &config.TestPlan{
		GlobalDefaults: config.GlobalDefaults{
			Tool:             config.ToolJMeter,
			PacingMultiplier: 4.0,
			GeneratorsCount:  5,
		},
		Scenarios: []config.Scenario{{Name: "S2"}},
		Profile:   config.TestProfile{Type: config.ProfileStable},
	}

	result := Plans([]*config.TestPlan{plan1, plan2})

	if result.Plan.GlobalDefaults.PacingMultiplier != 3.0 {
		t.Errorf("expected first plan's pacing_multiplier 3.0, got %f", result.Plan.GlobalDefaults.PacingMultiplier)
	}
	if result.Plan.GlobalDefaults.GeneratorsCount != 2 {
		t.Errorf("expected first plan's generators_count 2, got %d", result.Plan.GlobalDefaults.GeneratorsCount)
	}

	foundPacing := false
	foundGenerators := false
	for _, w := range result.Warnings {
		if w.Field == "global.pacing_multiplier" {
			foundPacing = true
		}
		if w.Field == "global.generators_count" {
			foundGenerators = true
		}
	}
	if !foundPacing {
		t.Error("expected warning for pacing_multiplier conflict")
	}
	if !foundGenerators {
		t.Error("expected warning for generators_count conflict")
	}
}

func TestMergePlans_ConflictingProfileType(t *testing.T) {
	plan1 := &config.TestPlan{
		Scenarios: []config.Scenario{{Name: "S1"}},
		Profile:   config.TestProfile{Type: config.ProfileStable},
	}
	plan2 := &config.TestPlan{
		Scenarios: []config.Scenario{{Name: "S2"}},
		Profile:   config.TestProfile{Type: config.ProfileMaxSearch},
	}

	result := Plans([]*config.TestPlan{plan1, plan2})

	if result.Plan.Profile.Type != config.ProfileStable {
		t.Errorf("expected first plan's profile type, got %s", result.Plan.Profile.Type)
	}

	found := false
	for _, w := range result.Warnings {
		if w.Field == "profile.type" {
			found = true
		}
	}
	if !found {
		t.Error("expected warning for profile type conflict")
	}
}

func TestMergePlans_ThreeConfigs_ScenariosOrder(t *testing.T) {
	plans := []*config.TestPlan{
		{Scenarios: []config.Scenario{{Name: "A1"}, {Name: "A2"}}},
		{Scenarios: []config.Scenario{{Name: "B1"}}},
		{Scenarios: []config.Scenario{{Name: "C1"}, {Name: "C2"}, {Name: "C3"}}},
	}

	result := Plans(plans)

	if len(result.Plan.Scenarios) != 6 {
		t.Fatalf("expected 6 scenarios, got %d", len(result.Plan.Scenarios))
	}
	expected := []string{"A1", "A2", "B1", "C1", "C2", "C3"}
	for i, name := range expected {
		if result.Plan.Scenarios[i].Name != name {
			t.Errorf("scenario %d: expected %s, got %s", i, name, result.Plan.Scenarios[i].Name)
		}
	}
}

func TestMergePlans_FromTestdataFiles(t *testing.T) {
	web, err := config.LoadFromYAML("testdata/config_web.yaml")
	if err != nil {
		t.Fatalf("loading config_web.yaml: %v", err)
	}
	api, err := config.LoadFromYAML("testdata/config_api.yaml")
	if err != nil {
		t.Fatalf("loading config_api.yaml: %v", err)
	}

	result := Plans([]*config.TestPlan{web, api})

	if len(result.Plan.Scenarios) != 4 {
		t.Fatalf("expected 4 scenarios, got %d", len(result.Plan.Scenarios))
	}
	if result.Plan.GlobalDefaults.PacingMultiplier != 3.0 {
		t.Errorf("expected pacing_multiplier from first (3.0), got %f", result.Plan.GlobalDefaults.PacingMultiplier)
	}
	// Profile type conflict: stable vs max_search
	foundProfileWarn := false
	foundPacingWarn := false
	for _, w := range result.Warnings {
		if w.Field == "profile.type" {
			foundProfileWarn = true
		}
		if w.Field == "global.pacing_multiplier" {
			foundPacingWarn = true
		}
	}
	if !foundProfileWarn {
		t.Error("expected profile.type warning")
	}
	if !foundPacingWarn {
		t.Error("expected pacing_multiplier warning")
	}
}

func TestWriteMergedYAML(t *testing.T) {
	plan := &config.TestPlan{
		Version: "1.0",
		GlobalDefaults: config.GlobalDefaults{
			Tool:             config.ToolJMeter,
			LoadModel:        config.LoadModelClosed,
			PacingMultiplier: 3.0,
		},
		Scenarios: []config.Scenario{
			{Name: "S1", ScriptID: 1},
		},
		Profile: config.TestProfile{Type: config.ProfileStable, Percent: 100},
	}

	dir := t.TempDir()
	dest := filepath.Join(dir, "merged.yaml")

	err := WriteMergedYAML(plan, dest)
	if err != nil {
		t.Fatalf("WriteMergedYAML failed: %v", err)
	}

	// Verify it can be loaded back
	loaded, err := config.LoadFromYAML(dest)
	if err != nil {
		t.Fatalf("loading merged yaml: %v", err)
	}
	if loaded.GlobalDefaults.Tool != config.ToolJMeter {
		t.Errorf("expected tool jmeter, got %s", loaded.GlobalDefaults.Tool)
	}
	if len(loaded.Scenarios) != 1 {
		t.Errorf("expected 1 scenario, got %d", len(loaded.Scenarios))
	}
}

func TestWriteMergedYAML_Stdout(t *testing.T) {
	plan := &config.TestPlan{
		Version:   "1.0",
		Scenarios: []config.Scenario{{Name: "S1"}},
	}

	// Empty dest means stdout; just ensure no error
	// We can't easily capture stdout here, so test with a file
	dir := t.TempDir()
	dest := filepath.Join(dir, "out.yaml")
	err := WriteMergedYAML(plan, dest)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadAndMergeFromDir(t *testing.T) {
	dir := t.TempDir()

	// Write two yaml configs into temp dir
	cfg1 := `version: "1.0"
global:
  tool: jmeter
  load_model: closed
  pacing_multiplier: 3.0
scenarios:
  - name: "DirS1"
    target_intensity: 100
    intensity_unit: ops_m
    max_script_time_ms: 500
profile:
  type: stable
  percent: 100
`
	cfg2 := `version: "1.0"
global:
  tool: jmeter
  load_model: closed
  pacing_multiplier: 3.0
scenarios:
  - name: "DirS2"
    target_intensity: 200
    intensity_unit: ops_m
    max_script_time_ms: 600
profile:
  type: stable
  percent: 100
`
	os.WriteFile(filepath.Join(dir, "a_config.yaml"), []byte(cfg1), 0o644)
	os.WriteFile(filepath.Join(dir, "b_config.yaml"), []byte(cfg2), 0o644)

	plans, err := LoadPlansFromDir(dir)
	if err != nil {
		t.Fatalf("LoadPlansFromDir: %v", err)
	}
	if len(plans) != 2 {
		t.Fatalf("expected 2 plans, got %d", len(plans))
	}

	result := Plans(plans)
	if len(result.Plan.Scenarios) != 2 {
		t.Fatalf("expected 2 scenarios, got %d", len(result.Plan.Scenarios))
	}
	if result.Plan.Scenarios[0].Name != "DirS1" {
		t.Errorf("expected DirS1 first, got %s", result.Plan.Scenarios[0].Name)
	}
}
