package profile

import (
	"testing"

	"loadcalc/internal/config"
)

func TestStability_100Percent(t *testing.T) {
	b := &StabilityProfileBuilder{}
	steps, err := b.BuildSteps(config.TestProfile{
		Percent:             100,
		DefaultRampupSec:    60,
		DefaultImpactSec:    300,
		DefaultStabilitySec: 120,
		DefaultRampdownSec:  30,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(steps))
	}
	s := steps[0]
	if s.PercentOfTarget != 100 {
		t.Errorf("percent = %v, want 100", s.PercentOfTarget)
	}
	if s.RampupSec != 60 || s.ImpactSec != 300 || s.StabilitySec != 120 || s.RampdownSec != 30 {
		t.Errorf("timing mismatch: %+v", s)
	}
}

func TestStability_25Percent(t *testing.T) {
	b := &StabilityProfileBuilder{}
	steps, _ := b.BuildSteps(config.TestProfile{
		Percent:             25,
		DefaultRampupSec:    30,
		DefaultImpactSec:    600,
		DefaultStabilitySec: 60,
	})
	if steps[0].PercentOfTarget != 25 {
		t.Errorf("percent = %v, want 25", steps[0].PercentOfTarget)
	}
}

func TestStability_DefaultPercent(t *testing.T) {
	b := &StabilityProfileBuilder{}
	steps, _ := b.BuildSteps(config.TestProfile{})
	if steps[0].PercentOfTarget != 100 {
		t.Errorf("default percent = %v, want 100", steps[0].PercentOfTarget)
	}
}
