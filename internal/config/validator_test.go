package config

import (
	"testing"

	"loadcalc/pkg/units"
)

func validPlan() *TestPlan {
	closed := LoadModelClosed
	pm := 3.0
	dt := 2.5
	sp := true
	return &TestPlan{
		Version: "1.0",
		GlobalDefaults: GlobalDefaults{
			Tool:               ToolJMeter,
			LoadModel:          LoadModelClosed,
			PacingMultiplier:   3.0,
			DeviationTolerance: 2.5,
			SpikeParticipate:   true,
			GeneratorsCount:    1,
		},
		Scenarios: []Scenario{
			{
				Name:               "Test",
				TargetIntensity:    1000,
				IntensityUnit:      units.OpsPerHour,
				MaxScriptTimeMs:    500,
				LoadModel:          &closed,
				PacingMultiplier:   &pm,
				DeviationTolerance: &dt,
				SpikeParticipate:   &sp,
			},
		},
		Profile: TestProfile{
			Type:    ProfileStability,
			Percent: 100,
		},
	}
}

func findError(errs []ValidationError, field string) *ValidationError {
	for _, e := range errs {
		if e.Field == field {
			return &e
		}
	}
	return nil
}

func findBySeverity(errs []ValidationError, sev Severity) []ValidationError {
	var out []ValidationError
	for _, e := range errs {
		if e.Severity == sev {
			out = append(out, e)
		}
	}
	return out
}

func TestValidate_ValidPlan_NoErrors(t *testing.T) {
	errs := Validate(validPlan())
	if len(errs) != 0 {
		t.Errorf("expected no errors, got %v", errs)
	}
}

func TestValidate_ToolRequired(t *testing.T) {
	p := validPlan()
	p.GlobalDefaults.Tool = ""
	errs := Validate(p)
	if e := findError(errs, "global.tool"); e == nil {
		t.Error("expected error for missing tool")
	}
}

func TestValidate_MaxScriptTimePositive(t *testing.T) {
	p := validPlan()
	p.Scenarios[0].MaxScriptTimeMs = 0
	errs := Validate(p)
	if e := findError(errs, "scenarios[0].max_script_time_ms"); e == nil {
		t.Error("expected error for max_script_time_ms <= 0")
	}
}

func TestValidate_TargetIntensityPositive(t *testing.T) {
	p := validPlan()
	p.Scenarios[0].TargetIntensity = -1
	errs := Validate(p)
	if e := findError(errs, "scenarios[0].target_intensity"); e == nil {
		t.Error("expected error for target_intensity <= 0")
	}
}

func TestValidate_PacingMultiplierMin(t *testing.T) {
	p := validPlan()
	v := 0.5
	p.Scenarios[0].PacingMultiplier = &v
	errs := Validate(p)
	if e := findError(errs, "scenarios[0].pacing_multiplier"); e == nil {
		t.Error("expected error for pacing_multiplier < 1.0")
	}
}

func TestValidate_DeviationToleranceNonNeg(t *testing.T) {
	p := validPlan()
	v := -1.0
	p.Scenarios[0].DeviationTolerance = &v
	errs := Validate(p)
	if e := findError(errs, "scenarios[0].deviation_tolerance"); e == nil {
		t.Error("expected error for deviation_tolerance < 0")
	}
}

func TestValidate_GeneratorsCountMin(t *testing.T) {
	p := validPlan()
	p.GlobalDefaults.GeneratorsCount = 0
	errs := Validate(p)
	if e := findError(errs, "global.generators_count"); e == nil {
		t.Error("expected error for generators_count < 1")
	}
}

func TestValidate_BackgroundPercentRange(t *testing.T) {
	p := validPlan()
	p.Scenarios[0].Background = true
	p.Scenarios[0].BackgroundPercent = 150
	errs := Validate(p)
	if e := findError(errs, "scenarios[0].background_percent"); e == nil {
		t.Error("expected error for background_percent > 100")
	}
}

func TestValidate_EmptyScenarioName(t *testing.T) {
	p := validPlan()
	p.Scenarios[0].Name = ""
	errs := Validate(p)
	if e := findError(errs, "scenarios[0].name"); e == nil {
		t.Error("expected error for empty name")
	}
}

func TestValidate_ScriptIDRequiredForLREPC(t *testing.T) {
	p := validPlan()
	p.GlobalDefaults.Tool = ToolLREPC
	p.GlobalDefaults.LoadModel = LoadModelClosed
	p.Scenarios[0].ScriptID = 0
	errs := Validate(p)
	if e := findError(errs, "scenarios[0].script_id"); e == nil {
		t.Error("expected error for missing script_id with lre_pc")
	}
}

func TestValidate_ScriptIDNotRequiredForJMeter(t *testing.T) {
	p := validPlan()
	p.GlobalDefaults.Tool = ToolJMeter
	p.Scenarios[0].ScriptID = 0
	errs := Validate(p)
	if e := findError(errs, "scenarios[0].script_id"); e != nil {
		t.Error("script_id should not be required for jmeter")
	}
}

func TestValidate_OpenModelLREPC_Global(t *testing.T) {
	p := validPlan()
	p.GlobalDefaults.Tool = ToolLREPC
	p.GlobalDefaults.LoadModel = LoadModelOpen
	errs := Validate(p)
	if e := findError(errs, "global"); e == nil {
		t.Error("expected error for open+lre_pc")
	}
}

func TestValidate_OpenModelLREPC_Scenario(t *testing.T) {
	p := validPlan()
	p.GlobalDefaults.Tool = ToolLREPC
	p.GlobalDefaults.LoadModel = LoadModelClosed
	open := LoadModelOpen
	p.Scenarios[0].LoadModel = &open
	errs := Validate(p)
	if e := findError(errs, "scenarios[0].load_model"); e == nil {
		t.Error("expected error for scenario open+lre_pc")
	}
}

func TestValidate_StabilityPercentPositive(t *testing.T) {
	p := validPlan()
	p.Profile.Percent = 0
	errs := Validate(p)
	if e := findError(errs, "profile.percent"); e == nil {
		t.Error("expected error for stability percent <= 0")
	}
}

func TestValidate_CapacityStepIncrementPositive(t *testing.T) {
	p := validPlan()
	p.Profile.Type = ProfileCapacity
	p.Profile.StartPercent = 50
	p.Profile.StepIncrement = 0
	p.Profile.NumSteps = 3
	errs := Validate(p)
	if e := findError(errs, "profile.step_increment"); e == nil {
		t.Error("expected error for step_increment <= 0")
	}
}

func TestValidate_CustomStepsEmpty(t *testing.T) {
	p := validPlan()
	p.Profile.Type = ProfileCustom
	p.Profile.Steps = nil
	errs := Validate(p)
	if e := findError(errs, "profile.steps"); e == nil {
		t.Error("expected error for empty custom steps")
	}
}

func TestValidate_CustomStepPercentPositive(t *testing.T) {
	p := validPlan()
	p.Profile.Type = ProfileCustom
	p.Profile.Steps = []ProfileStep{{PercentOfTarget: -10}}
	errs := Validate(p)
	if e := findError(errs, "profile.steps[0].percent"); e == nil {
		t.Error("expected error for custom step percent <= 0")
	}
}

func TestValidate_StepOverrideMismatchWarning(t *testing.T) {
	p := validPlan()
	p.Profile.Type = ProfileCapacity
	p.Profile.StartPercent = 50
	p.Profile.StepIncrement = 25
	p.Profile.NumSteps = 3 // generates 50, 75, 100
	p.Profile.Steps = []ProfileStep{
		{PercentOfTarget: 99}, // doesn't match
	}
	errs := Validate(p)
	warnings := findBySeverity(errs, SeverityWarning)
	found := false
	for _, w := range warnings {
		if w.Field == "profile.steps" {
			found = true
		}
	}
	if !found {
		t.Error("expected warning for step override mismatch")
	}
}

func TestValidate_FineTuneAfterPercentMismatch(t *testing.T) {
	p := validPlan()
	p.Profile.Type = ProfileCapacity
	p.Profile.StartPercent = 100
	p.Profile.StepIncrement = 100
	p.Profile.NumSteps = 3 // generates 100, 200, 300
	p.Profile.FineTune = &FineTune{
		AfterPercent:  250, // should be 300
		StepIncrement: 10,
		NumSteps:      3,
	}
	errs := Validate(p)
	if e := findError(errs, "profile.fine_tune.after_percent"); e == nil {
		t.Error("expected error for fine_tune.after_percent mismatch")
	}
}

func TestValidate_FineTuneAfterPercentCorrect(t *testing.T) {
	p := validPlan()
	p.Profile.Type = ProfileCapacity
	p.Profile.StartPercent = 100
	p.Profile.StepIncrement = 100
	p.Profile.NumSteps = 3
	p.Profile.FineTune = &FineTune{
		AfterPercent:  300,
		StepIncrement: 10,
		NumSteps:      3,
	}
	errs := Validate(p)
	if e := findError(errs, "profile.fine_tune.after_percent"); e != nil {
		t.Errorf("unexpected error: %v", e)
	}
}

func TestValidate_BackgroundSpikeWarning(t *testing.T) {
	p := validPlan()
	sp := true
	p.Scenarios[0].Background = true
	p.Scenarios[0].BackgroundPercent = 50
	p.Scenarios[0].SpikeParticipate = &sp
	errs := Validate(p)
	warnings := findBySeverity(errs, SeverityWarning)
	if len(warnings) == 0 {
		t.Error("expected warning for background + spike_participate")
	}
}

func TestValidate_HasErrors(t *testing.T) {
	errs := []ValidationError{
		{Severity: SeverityWarning},
	}
	if HasErrors(errs) {
		t.Error("warnings-only should not count as errors")
	}
	errs = append(errs, ValidationError{Severity: SeverityError})
	if !HasErrors(errs) {
		t.Error("should detect errors")
	}
}

func TestValidate_SpikeBasePercentPositive(t *testing.T) {
	p := validPlan()
	p.Profile.Type = ProfileSpike
	p.Profile.BasePercent = 0
	errs := Validate(p)
	if e := findError(errs, "profile.base_percent"); e == nil {
		t.Error("expected error for spike base_percent <= 0")
	}
}
