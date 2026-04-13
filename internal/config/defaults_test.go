package config

import (
	"testing"
)

func TestResolveDefaults_GlobalFillsFromSpec(t *testing.T) {
	plan := &TestPlan{}
	ResolveDefaults(plan)

	if plan.GlobalDefaults.PacingMultiplier != 3.0 {
		t.Errorf("PacingMultiplier = %v, want 3.0", plan.GlobalDefaults.PacingMultiplier)
	}
	if plan.GlobalDefaults.DeviationTolerance != 2.5 {
		t.Errorf("DeviationTolerance = %v, want 2.5", plan.GlobalDefaults.DeviationTolerance)
	}
	if plan.GlobalDefaults.GeneratorsCount != 1 {
		t.Errorf("GeneratorsCount = %v, want 1", plan.GlobalDefaults.GeneratorsCount)
	}
	if plan.GlobalDefaults.LoadModel != LoadModelClosed {
		t.Errorf("LoadModel = %v, want closed", plan.GlobalDefaults.LoadModel)
	}
}

func TestResolveDefaults_ScenarioNilOverridesGetGlobal(t *testing.T) {
	plan := &TestPlan{
		GlobalDefaults: GlobalDefaults{
			Tool:               ToolJMeter,
			LoadModel:          LoadModelClosed,
			PacingMultiplier:   3.0,
			DeviationTolerance: 2.5,
			SpikeParticipate:   true,
			GeneratorsCount:    2,
		},
		Scenarios: []Scenario{
			{Name: "Test", TargetIntensity: 100, MaxScriptTimeMs: 500},
		},
	}
	ResolveDefaults(plan)

	s := plan.Scenarios[0]
	if s.LoadModel == nil || *s.LoadModel != LoadModelClosed {
		t.Error("LoadModel should be closed from global")
	}
	if s.PacingMultiplier == nil || *s.PacingMultiplier != 3.0 {
		t.Error("PacingMultiplier should be 3.0 from global")
	}
	if s.DeviationTolerance == nil || *s.DeviationTolerance != 2.5 {
		t.Error("DeviationTolerance should be 2.5 from global")
	}
	if s.SpikeParticipate == nil || *s.SpikeParticipate != true {
		t.Error("SpikeParticipate should be true from global")
	}
}

func TestResolveDefaults_ScenarioOverridePreserved(t *testing.T) {
	pm := 5.0
	lm := LoadModelOpen
	plan := &TestPlan{
		GlobalDefaults: GlobalDefaults{
			Tool:             ToolJMeter,
			LoadModel:        LoadModelClosed,
			PacingMultiplier: 3.0,
		},
		Scenarios: []Scenario{
			{
				Name:             "Test",
				PacingMultiplier: &pm,
				LoadModel:        &lm,
			},
		},
	}
	ResolveDefaults(plan)

	s := plan.Scenarios[0]
	if *s.PacingMultiplier != 5.0 {
		t.Errorf("PacingMultiplier should remain 5.0, got %v", *s.PacingMultiplier)
	}
	if *s.LoadModel != LoadModelOpen {
		t.Errorf("LoadModel should remain open, got %v", *s.LoadModel)
	}
}

func TestResolveDefaults_ExplicitGlobalNotOverwritten(t *testing.T) {
	plan := &TestPlan{
		GlobalDefaults: GlobalDefaults{
			Tool:               ToolJMeter,
			PacingMultiplier:   5.0,
			DeviationTolerance: 1.0,
			GeneratorsCount:    4,
			LoadModel:          LoadModelOpen,
		},
	}
	ResolveDefaults(plan)

	if plan.GlobalDefaults.PacingMultiplier != 5.0 {
		t.Errorf("PacingMultiplier should stay 5.0, got %v", plan.GlobalDefaults.PacingMultiplier)
	}
	if plan.GlobalDefaults.DeviationTolerance != 1.0 {
		t.Errorf("DeviationTolerance should stay 1.0")
	}
	if plan.GlobalDefaults.GeneratorsCount != 4 {
		t.Errorf("GeneratorsCount should stay 4")
	}
	if plan.GlobalDefaults.LoadModel != LoadModelOpen {
		t.Errorf("LoadModel should stay open")
	}
}
