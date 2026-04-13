package whatif

import (
	"testing"

	"loadcalc/internal/config"
)

func TestApplyOverrides_GlobalPacingMultiplier(t *testing.T) {
	plan := &config.TestPlan{
		GlobalDefaults: config.GlobalDefaults{PacingMultiplier: 3.0},
	}
	err := ApplyOverrides(plan, map[string]string{"global.pacing_multiplier": "4.0"})
	if err != nil {
		t.Fatal(err)
	}
	if plan.GlobalDefaults.PacingMultiplier != 4.0 {
		t.Errorf("expected 4.0, got %f", plan.GlobalDefaults.PacingMultiplier)
	}
}

func TestApplyOverrides_GlobalGeneratorsCount(t *testing.T) {
	plan := &config.TestPlan{}
	err := ApplyOverrides(plan, map[string]string{"global.generators_count": "5"})
	if err != nil {
		t.Fatal(err)
	}
	if plan.GlobalDefaults.GeneratorsCount != 5 {
		t.Errorf("expected 5, got %d", plan.GlobalDefaults.GeneratorsCount)
	}
}

func TestApplyOverrides_GlobalDeviationTolerance(t *testing.T) {
	plan := &config.TestPlan{}
	err := ApplyOverrides(plan, map[string]string{"global.deviation_tolerance": "3.0"})
	if err != nil {
		t.Fatal(err)
	}
	if plan.GlobalDefaults.DeviationTolerance != 3.0 {
		t.Errorf("expected 3.0, got %f", plan.GlobalDefaults.DeviationTolerance)
	}
}

func TestApplyOverrides_GlobalTool(t *testing.T) {
	plan := &config.TestPlan{}
	err := ApplyOverrides(plan, map[string]string{"global.tool": "lre_pc"})
	if err != nil {
		t.Fatal(err)
	}
	if plan.GlobalDefaults.Tool != config.ToolLREPC {
		t.Errorf("expected lre_pc, got %s", plan.GlobalDefaults.Tool)
	}
}

func TestApplyOverrides_GlobalLoadModel(t *testing.T) {
	plan := &config.TestPlan{}
	err := ApplyOverrides(plan, map[string]string{"global.load_model": "open"})
	if err != nil {
		t.Fatal(err)
	}
	if plan.GlobalDefaults.LoadModel != config.LoadModelOpen {
		t.Errorf("expected open, got %s", plan.GlobalDefaults.LoadModel)
	}
}

func TestApplyOverrides_GlobalSpikeParticipate(t *testing.T) {
	plan := &config.TestPlan{GlobalDefaults: config.GlobalDefaults{SpikeParticipate: true}}
	err := ApplyOverrides(plan, map[string]string{"global.spike_participate": "false"})
	if err != nil {
		t.Fatal(err)
	}
	if plan.GlobalDefaults.SpikeParticipate != false {
		t.Error("expected false")
	}
}

func TestApplyOverrides_ScenarioByIndex(t *testing.T) {
	plan := &config.TestPlan{
		Scenarios: []config.Scenario{
			{Name: "A", TargetIntensity: 100},
			{Name: "B", TargetIntensity: 200},
		},
	}
	err := ApplyOverrides(plan, map[string]string{"scenarios[1].target_intensity": "500000"})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Scenarios[1].TargetIntensity != 500000 {
		t.Errorf("expected 500000, got %f", plan.Scenarios[1].TargetIntensity)
	}
}

func TestApplyOverrides_ScenarioByName(t *testing.T) {
	plan := &config.TestPlan{
		Scenarios: []config.Scenario{
			{Name: "Main page", MaxScriptTimeMs: 1000},
			{Name: "Other", MaxScriptTimeMs: 500},
		},
	}
	err := ApplyOverrides(plan, map[string]string{"scenarios[Main page].max_script_time_ms": "2000"})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Scenarios[0].MaxScriptTimeMs != 2000 {
		t.Errorf("expected 2000, got %d", plan.Scenarios[0].MaxScriptTimeMs)
	}
}

func TestApplyOverrides_ScenarioPacingMultiplier(t *testing.T) {
	plan := &config.TestPlan{
		Scenarios: []config.Scenario{{Name: "A"}},
	}
	err := ApplyOverrides(plan, map[string]string{"scenarios[0].pacing_multiplier": "4.0"})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Scenarios[0].PacingMultiplier == nil || *plan.Scenarios[0].PacingMultiplier != 4.0 {
		t.Error("expected pacing_multiplier to be 4.0")
	}
}

func TestApplyOverrides_ProfileNumSteps(t *testing.T) {
	plan := &config.TestPlan{Profile: config.TestProfile{NumSteps: 5}}
	err := ApplyOverrides(plan, map[string]string{"profile.num_steps": "10"})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Profile.NumSteps != 10 {
		t.Errorf("expected 10, got %d", plan.Profile.NumSteps)
	}
}

func TestApplyOverrides_ProfileStartPercent(t *testing.T) {
	plan := &config.TestPlan{}
	err := ApplyOverrides(plan, map[string]string{"profile.start_percent": "100"})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Profile.StartPercent != 100 {
		t.Errorf("expected 100, got %f", plan.Profile.StartPercent)
	}
}

func TestApplyOverrides_ProfileStepIncrement(t *testing.T) {
	plan := &config.TestPlan{}
	err := ApplyOverrides(plan, map[string]string{"profile.step_increment": "50"})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Profile.StepIncrement != 50 {
		t.Errorf("expected 50, got %f", plan.Profile.StepIncrement)
	}
}

func TestApplyOverrides_ProfilePercent(t *testing.T) {
	plan := &config.TestPlan{}
	err := ApplyOverrides(plan, map[string]string{"profile.percent": "50"})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Profile.Percent != 50 {
		t.Errorf("expected 50, got %f", plan.Profile.Percent)
	}
}

func TestApplyOverrides_InvalidPath(t *testing.T) {
	plan := &config.TestPlan{}
	err := ApplyOverrides(plan, map[string]string{"invalid.path": "123"})
	if err == nil {
		t.Error("expected error for invalid path")
	}
}

func TestApplyOverrides_InvalidValueType(t *testing.T) {
	plan := &config.TestPlan{}
	err := ApplyOverrides(plan, map[string]string{"global.pacing_multiplier": "not_a_number"})
	if err == nil {
		t.Error("expected error for invalid value type")
	}
}

func TestApplyOverrides_ScenarioIndexOutOfRange(t *testing.T) {
	plan := &config.TestPlan{Scenarios: []config.Scenario{{Name: "A"}}}
	err := ApplyOverrides(plan, map[string]string{"scenarios[5].target_intensity": "100"})
	if err == nil {
		t.Error("expected error for out-of-range index")
	}
}

func TestApplyOverrides_ScenarioNameNotFound(t *testing.T) {
	plan := &config.TestPlan{Scenarios: []config.Scenario{{Name: "A"}}}
	err := ApplyOverrides(plan, map[string]string{"scenarios[NonExistent].target_intensity": "100"})
	if err == nil {
		t.Error("expected error for non-existent scenario name")
	}
}
