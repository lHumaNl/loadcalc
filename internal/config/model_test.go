package config

import (
	"testing"

	"loadcalc/pkg/units"
)

func TestGlobalDefaultsValues(t *testing.T) {
	d := DefaultGlobalDefaults()
	if d.PacingMultiplier != 3.0 {
		t.Errorf("PacingMultiplier = %v, want 3.0", d.PacingMultiplier)
	}
	if d.DeviationTolerance != 2.5 {
		t.Errorf("DeviationTolerance = %v, want 2.5", d.DeviationTolerance)
	}
	if d.SpikeParticipate != true {
		t.Error("SpikeParticipate should default to true")
	}
	if d.GeneratorsCount != 1 {
		t.Errorf("GeneratorsCount = %v, want 1", d.GeneratorsCount)
	}
}

func TestScenarioCreation(t *testing.T) {
	s := Scenario{
		Name:              "Login",
		ScriptID:          1,
		TargetIntensity:   3600,
		IntensityUnit:     units.OpsPerHour,
		MaxScriptTimeMs:   5000,
		Background:        false,
		BackgroundPercent: 0,
	}
	if s.Name != "Login" {
		t.Errorf("Name = %v, want Login", s.Name)
	}
	if s.IntensityUnit != units.OpsPerHour {
		t.Errorf("IntensityUnit = %v, want %v", s.IntensityUnit, units.OpsPerHour)
	}
}

func TestScenarioOverridesNilByDefault(t *testing.T) {
	s := Scenario{}
	if s.LoadModel != nil {
		t.Error("LoadModel override should be nil by default")
	}
	if s.PacingMultiplier != nil {
		t.Error("PacingMultiplier override should be nil by default")
	}
	if s.DeviationTolerance != nil {
		t.Error("DeviationTolerance override should be nil by default")
	}
	if s.SpikeParticipate != nil {
		t.Error("SpikeParticipate override should be nil by default")
	}
}

func TestProfileTypeConstants(t *testing.T) {
	if ProfileStable != "stable" {
		t.Error("ProfileStable constant wrong")
	}
	if ProfileMaxSearch != "max_search" {
		t.Error("ProfileMaxSearch constant wrong")
	}
	if ProfileCustom != "custom" {
		t.Error("ProfileCustom constant wrong")
	}
	if ProfileSpike != "spike" {
		t.Error("ProfileSpike constant wrong")
	}
}

func TestToolConstants(t *testing.T) {
	if ToolLREPC != "lre_pc" {
		t.Error("ToolLREPC constant wrong")
	}
	if ToolJMeter != "jmeter" {
		t.Error("ToolJMeter constant wrong")
	}
}

func TestLoadModelConstants(t *testing.T) {
	if LoadModelClosed != "closed" {
		t.Error("LoadModelClosed constant wrong")
	}
	if LoadModelOpen != "open" {
		t.Error("LoadModelOpen constant wrong")
	}
}

func TestTestPlanStructure(t *testing.T) {
	plan := TestPlan{
		GlobalDefaults: DefaultGlobalDefaults(),
		Scenarios: []Scenario{
			{Name: "Test"},
		},
		Profile: TestProfile{
			Type: ProfileStable,
		},
	}
	if len(plan.Scenarios) != 1 {
		t.Errorf("expected 1 scenario, got %d", len(plan.Scenarios))
	}
	if plan.Profile.Type != ProfileStable {
		t.Errorf("profile type = %v, want stable", plan.Profile.Type)
	}
}
