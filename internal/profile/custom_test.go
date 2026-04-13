package profile

import (
	"testing"

	"loadcalc/internal/config"
)

func TestCustom_ArbitraryList(t *testing.T) {
	b := &CustomProfileBuilder{}
	steps, err := b.BuildSteps(config.TestProfile{
		DefaultRampupSec:    60,
		DefaultImpactSec:    300,
		DefaultStabilitySec: 120,
		DefaultRampdownSec:  0,
		Steps: []config.ProfileStep{
			{PercentOfTarget: 100},
			{PercentOfTarget: 200},
			{PercentOfTarget: 100},
			{PercentOfTarget: 300},
			{PercentOfTarget: 200},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	expected := []float64{100, 200, 100, 300, 200}
	if len(steps) != len(expected) {
		t.Fatalf("got %d steps, want %d", len(steps), len(expected))
	}
	for i, want := range expected {
		if steps[i].PercentOfTarget != want {
			t.Errorf("step %d: percent=%v, want %v", i, steps[i].PercentOfTarget, want)
		}
		if steps[i].StepNumber != i+1 {
			t.Errorf("step %d: number=%d", i, steps[i].StepNumber)
		}
	}
}

func TestCustom_PerStepTimingOverrides(t *testing.T) {
	rampup := 120
	impact := 600
	b := &CustomProfileBuilder{}
	steps, _ := b.BuildSteps(config.TestProfile{
		DefaultRampupSec:    60,
		DefaultImpactSec:    300,
		DefaultStabilitySec: 120,
		Steps: []config.ProfileStep{
			{PercentOfTarget: 100, RampupSec: &rampup, ImpactSec: &impact},
			{PercentOfTarget: 200},
		},
	})
	if steps[0].RampupSec != 120 || steps[0].ImpactSec != 600 {
		t.Errorf("step 0 overrides not applied: %+v", steps[0])
	}
	if steps[1].RampupSec != 60 || steps[1].ImpactSec != 300 {
		t.Errorf("step 1 should use defaults: %+v", steps[1])
	}
}

func TestCustom_Repeats(t *testing.T) {
	b := &CustomProfileBuilder{}
	steps, _ := b.BuildSteps(config.TestProfile{
		Steps: []config.ProfileStep{
			{PercentOfTarget: 50},
			{PercentOfTarget: 50},
			{PercentOfTarget: 50},
		},
	})
	if len(steps) != 3 {
		t.Fatalf("got %d steps, want 3", len(steps))
	}
	for i, s := range steps {
		if s.PercentOfTarget != 50 {
			t.Errorf("step %d: percent=%v", i, s.PercentOfTarget)
		}
	}
}
