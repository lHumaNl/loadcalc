package profile

import (
	"testing"

	"loadcalc/internal/config"
)

func TestCapacity_Basic(t *testing.T) {
	b := &CapacityProfileBuilder{}
	steps, err := b.BuildSteps(config.TestProfile{
		StartPercent:        50,
		StepIncrement:       25,
		NumSteps:            5,
		DefaultRampupSec:    60,
		DefaultImpactSec:    300,
		DefaultStabilitySec: 120,
		DefaultRampdownSec:  0,
	})
	if err != nil {
		t.Fatal(err)
	}
	expected := []float64{50, 75, 100, 125, 150}
	if len(steps) != len(expected) {
		t.Fatalf("got %d steps, want %d", len(steps), len(expected))
	}
	for i, want := range expected {
		if steps[i].PercentOfTarget != want {
			t.Errorf("step %d: percent=%v, want %v", i, steps[i].PercentOfTarget, want)
		}
		if steps[i].StepNumber != i+1 {
			t.Errorf("step %d: number=%d, want %d", i, steps[i].StepNumber, i+1)
		}
	}
}

func TestCapacity_WithFineTune(t *testing.T) {
	b := &CapacityProfileBuilder{}
	steps, err := b.BuildSteps(config.TestProfile{
		StartPercent:  100,
		StepIncrement: 100,
		NumSteps:      3,
		FineTune: &config.FineTune{
			AfterPercent:  300,
			StepIncrement: 10,
			NumSteps:      3,
		},
		DefaultRampupSec:    60,
		DefaultImpactSec:    300,
		DefaultStabilitySec: 120,
	})
	if err != nil {
		t.Fatal(err)
	}
	expected := []float64{100, 200, 300, 310, 320, 330}
	if len(steps) != len(expected) {
		t.Fatalf("got %d steps, want %d", len(steps), len(expected))
	}
	for i, want := range expected {
		if steps[i].PercentOfTarget != want {
			t.Errorf("step %d: percent=%v, want %v", i, steps[i].PercentOfTarget, want)
		}
	}
}

func TestCapacity_PerStepOverrides(t *testing.T) {
	rampup := 120
	b := &CapacityProfileBuilder{}
	steps, _ := b.BuildSteps(config.TestProfile{
		StartPercent:        50,
		StepIncrement:       50,
		NumSteps:            3,
		DefaultRampupSec:    60,
		DefaultImpactSec:    300,
		DefaultStabilitySec: 120,
		Steps: []config.ProfileStep{
			{PercentOfTarget: 100, RampupSec: &rampup},
			{PercentOfTarget: 999}, // no match — ignored
		},
	})
	// Step at 100% should have overridden rampup
	if steps[1].RampupSec != 120 {
		t.Errorf("step at 100%%: rampup=%d, want 120", steps[1].RampupSec)
	}
	// Other steps use default
	if steps[0].RampupSec != 60 {
		t.Errorf("step at 50%%: rampup=%d, want 60", steps[0].RampupSec)
	}
}
